#!/usr/bin/env bash
# E2E: wrong-data evidence triggers slash + jail + repair.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STACK_SCRIPT="$ROOT_DIR/scripts/run_devnet_alpha_multi_sp.sh"

CHAIN_HOME="${NIL_HOME:-$ROOT_DIR/_artifacts/nilchain_data_devnet_alpha}"
CHAIN_ID="${CHAIN_ID:-31337}"
RPC_STATUS="${RPC_STATUS:-http://127.0.0.1:26657/status}"
LCD_BASE="${LCD_BASE:-http://127.0.0.1:1317}"
GATEWAY_BASE="${GATEWAY_BASE:-http://127.0.0.1:8080}"

NILCHAIND_BIN="${NILCHAIND_BIN:-$ROOT_DIR/nilchain/nilchaind}"

PROVIDER_COUNT="${PROVIDER_COUNT:-12}"

UPLOAD_FILE="${UPLOAD_FILE:-$ROOT_DIR/test_1mb.bin}"
FILE_PATH="${FILE_PATH:-test_1mb.bin}"
RAW_BLOB_PAYLOAD_BYTES="${RAW_BLOB_PAYLOAD_BYTES:-126976}"

TMP_DIR="${TMP_DIR:-$ROOT_DIR/_artifacts/e2e_evidence_tmp}"
mkdir -p "$TMP_DIR"

cleanup() {
  echo "==> Stopping devnet alpha multi-SP stack..."
  "$STACK_SCRIPT" stop || true
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: missing required command: $1" >&2
    exit 1
  fi
}

wait_for_http() {
  local name="$1"
  local url="$2"
  local expect_codes="${3:-200}"
  local max_attempts="${4:-60}"
  local delay_secs="${5:-1}"

  echo "==> Waiting for $name at $url ..."
  for attempt in $(seq 1 "$max_attempts"); do
    local code
    code=$(timeout 10s curl -s -o /dev/null -w '%{http_code}' --max-time 3 "$url" 2>/dev/null || true)
    code="${code:-000}"
    if echo ",$expect_codes," | grep -q ",$code,"; then
      echo "    $name reachable (HTTP $code) after $attempt attempt(s)."
      return 0
    fi
    sleep "$delay_secs"
  done

  echo "ERROR: $name at $url not reachable" >&2
  return 1
}

rpc_height() {
  timeout 10s curl -s --max-time 2 "$RPC_STATUS" | python3 -c '
import json, sys
try:
  data = json.load(sys.stdin)
  print(int(data["result"]["sync_info"]["latest_block_height"]))
except Exception:
  print(0)
'
}

wait_for_tx() {
  local txhash="$1"
  local attempts="${2:-30}"
  local delay="${3:-1}"
  if [ -z "$txhash" ]; then
    return 1
  fi
  for _ in $(seq 1 "$attempts"); do
    if "$NILCHAIND_BIN" query tx "$txhash" --node tcp://127.0.0.1:26657 --output json --home "$CHAIN_HOME" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$delay"
  done
  return 1
}

extract_last_json() {
  python3 -c '
import json, sys
s = sys.stdin.read()
start = s.find("{")
end = s.rfind("}")
if start == -1 or end == -1 or end <= start:
  print("")
  raise SystemExit(0)
snippet = s[start : end + 1]
try:
  obj = json.loads(snippet)
except Exception:
  last = s.rfind("{")
  if last == -1:
    print("")
    raise SystemExit(0)
  snippet = s[last : end + 1]
  obj = json.loads(snippet)
print(json.dumps(obj))
'
}

parse_create_deal_id() {
  python3 -c '
import json, sys
try:
  tx = json.load(sys.stdin)
except Exception:
  print("")
  raise SystemExit(0)
logs = tx.get("logs") or []
events = []
for item in logs:
  events.extend(item.get("events") or [])
if not events:
  events = tx.get("events") or []
for ev in events:
  if (ev.get("type") or "") != "create_deal":
    continue
  for a in ev.get("attributes") or []:
    if (a.get("key") or "") == "deal_id":
      print(a.get("value") or "")
      raise SystemExit(0)
print("")
'
}

resolve_key_for_address() {
  local addr="$1"
  "$NILCHAIND_BIN" keys list --home "$CHAIN_HOME" --keyring-backend test --output json | python3 -c '
import json, sys
addr = sys.argv[1].strip()
try:
  data = json.load(sys.stdin)
except Exception:
  print("")
  raise SystemExit(0)
for entry in data:
  if (entry.get("address") or "").strip() == addr:
    print(entry.get("name") or "")
    raise SystemExit(0)
print("")
' "$addr"
}

wait_for_provider_status() {
  local addr="$1"
  local want="$2"
  local attempts="${3:-30}"
  local delay="${4:-1}"
  local status
  for _ in $(seq 1 "$attempts"); do
    status=$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/providers/$addr" | python3 -c 'import json, sys
try:
  d=json.load(sys.stdin)
  p=d.get("provider") or {}
  print((p.get("status") or ""))
except Exception:
  print("")
' 2>/dev/null || true)
    if [ "$status" == "$want" ]; then
      echo "$status"
      return 0
    fi
    sleep "$delay"
  done
  echo "$status"
  return 1
}

wait_for_repair_slot() {
  local deal_id="$1"
  local provider="$2"
  local attempts="${3:-30}"
  local delay="${4:-1}"
  local slot_json
  for _ in $(seq 1 "$attempts"); do
    slot_json=$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$deal_id" | PROVIDER="$provider" python3 -c '
import json, os, sys
provider = os.environ.get("PROVIDER","" ).strip()
try:
  data = json.load(sys.stdin)
except Exception:
  print("")
  raise SystemExit(0)
slots = (data.get("deal") or {}).get("mode2_slots") or []
for s in slots:
  if not s:
    continue
  if (s.get("provider") or "").strip() != provider:
    continue
  print(json.dumps(s))
  raise SystemExit(0)
print("")
' 2>/dev/null || true)
    if [ -n "$slot_json" ]; then
      echo "$slot_json"
      return 0
    fi
    sleep "$delay"
  done
  echo ""
  return 1
}

require_cmd curl
require_cmd python3

if [ ! -f "$UPLOAD_FILE" ]; then
  echo "ERROR: UPLOAD_FILE does not exist: $UPLOAD_FILE" >&2
  exit 1
fi

export PROVIDER_COUNT
export START_WEB="${START_WEB:-0}"

echo "==> Starting devnet alpha multi-SP stack (providers=$PROVIDER_COUNT)..."
"$STACK_SCRIPT" start

wait_for_http "lcd" "$LCD_BASE/cosmos/base/tendermint/v1beta1/node_info" "200" 60 1
wait_for_http "nilchain lcd" "$LCD_BASE/nilchain/nilchain/v1/params" "200" 60 1
wait_for_http "gateway router" "$GATEWAY_BASE/health" "200" 60 1

FAUCET_ADDR="$($NILCHAIND_BIN keys show faucet -a --home "$CHAIN_HOME" --keyring-backend test 2>/dev/null || true)"
if [ -z "$FAUCET_ADDR" ]; then
  echo "ERROR: failed to resolve faucet address" >&2
  exit 1
fi

SERVICE_HINT="General:replicas=${PROVIDER_COUNT}:rs=8+4"

echo "==> Creating Mode 2 deal..."
DEAL_ID=""
for attempt in $(seq 1 5); do
  CREATE_RES_RAW="$($NILCHAIND_BIN tx nilchain create-deal 200 1000000 500000 \
    --service-hint "$SERVICE_HINT" \
    --from faucet \
    --chain-id "$CHAIN_ID" \
    --node tcp://127.0.0.1:26657 \
    --home "$CHAIN_HOME" \
    --keyring-backend test \
    --yes \
    --gas 250000 \
    --gas-prices 0.001aatom \
    --broadcast-mode sync \
    --output json 2>/dev/null || true)"
  CREATE_RES="$(echo "$CREATE_RES_RAW" | extract_last_json)"
  if [ -z "$CREATE_RES" ]; then
    echo "WARN: create-deal returned no JSON (attempt $attempt)" >&2
    sleep 2
    continue
  fi
  TXHASH="$(echo "$CREATE_RES" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("txhash", ""))' 2>/dev/null || true)"
  if [ -z "$TXHASH" ]; then
    echo "WARN: failed to parse create-deal txhash (attempt $attempt)" >&2
    sleep 2
    continue
  fi

  for _ in $(seq 1 10); do
    sleep 1
    CREATE_TX_RAW="$($NILCHAIND_BIN query tx "$TXHASH" --node tcp://127.0.0.1:26657 --output json --home "$CHAIN_HOME" 2>/dev/null || true)"
    CREATE_TX="$(echo "$CREATE_TX_RAW" | extract_last_json)"
    DEAL_ID="$(echo "$CREATE_TX" | parse_create_deal_id)"
    if [ -n "$DEAL_ID" ]; then
      break
    fi
  done

  if [ -n "$DEAL_ID" ] && [ "$DEAL_ID" != "0" ]; then
    break
  fi
  echo "WARN: create-deal tx not found yet (attempt $attempt)" >&2
  sleep 2
done
if [ -z "$DEAL_ID" ] || [ "$DEAL_ID" == "0" ]; then
  DEAL_ID="$($NILCHAIND_BIN query nilchain list-deals --node tcp://127.0.0.1:26657 --output json --home "$CHAIN_HOME" 2>/dev/null | python3 -c $'import json, sys\ntry:\n  data = json.load(sys.stdin)\n  deals = data.get(\"deals\") or []\n  if deals:\n    print(deals[-1].get(\"id\") or \"\")\n  else:\n    print(\"\")\nexcept Exception:\n  print(\"\")\n')"
fi
if [ -z "$DEAL_ID" ] || [ "$DEAL_ID" == "0" ]; then
  echo "ERROR: failed to parse deal id from tx" >&2
  echo "$CREATE_RES_RAW" >&2
  exit 1
fi

echo "    Deal ID: $DEAL_ID"

echo "==> Uploading file via router gateway..."
UPLOAD_RESP="$(timeout 120s curl -sS -X POST \
  -F "file=@$UPLOAD_FILE" \
  -F "owner=$FAUCET_ADDR" \
  -F "file_path=$FILE_PATH" \
  "$GATEWAY_BASE/gateway/upload?deal_id=$DEAL_ID")"
MANIFEST_ROOT="$(echo "$UPLOAD_RESP" | python3 -c 'import sys, json; j=json.load(sys.stdin); print(j.get("manifest_root") or j.get("cid") or "")' 2>/dev/null || true)"
SIZE_BYTES="$(echo "$UPLOAD_RESP" | python3 -c 'import sys, json; j=json.load(sys.stdin); print(j.get("size_bytes") or j.get("sizeBytes") or "")' 2>/dev/null || true)"
TOTAL_MDUS="$(echo "$UPLOAD_RESP" | python3 -c 'import sys, json; j=json.load(sys.stdin); print(j.get("total_mdus") or j.get("totalMdus") or "")' 2>/dev/null || true)"
WITNESS_MDUS="$(echo "$UPLOAD_RESP" | python3 -c 'import sys, json; j=json.load(sys.stdin); print(j.get("witness_mdus") or j.get("witnessMdus") or "")' 2>/dev/null || true)"
if [ -z "$MANIFEST_ROOT" ] || [ -z "$SIZE_BYTES" ] || [ -z "$TOTAL_MDUS" ] || [ -z "$WITNESS_MDUS" ]; then
  echo "ERROR: upload response missing required fields" >&2
  echo "$UPLOAD_RESP" >&2
  exit 1
fi

echo "==> Committing deal content on-chain..."
COMMIT_OUT_RAW="$($NILCHAIND_BIN tx nilchain update-deal-content \
  --deal-id "$DEAL_ID" \
  --cid "$MANIFEST_ROOT" \
  --size "$SIZE_BYTES" \
  --total-mdus "$TOTAL_MDUS" \
  --witness-mdus "$WITNESS_MDUS" \
  --from faucet \
  --chain-id "$CHAIN_ID" \
  --node tcp://127.0.0.1:26657 \
  --home "$CHAIN_HOME" \
  --keyring-backend test \
  --yes \
  --gas auto \
  --gas-adjustment 1.6 \
  --gas-prices 0.001aatom \
  --broadcast-mode sync \
  --output json)"
COMMIT_HASH="$(printf '%s' "$COMMIT_OUT_RAW" | python3 -c $'import json, re, sys\ns = sys.stdin.read()\nmatches = re.findall(r\"\\\"txhash\\\"\\\\s*:\\\\s*\\\"([^\\\"]+)\\\"\", s)\nif matches:\n  print(matches[-1])\n  raise SystemExit(0)\nstart = s.find(\"{\")\nend = s.rfind(\"}\")\nif start != -1 and end != -1 and end > start:\n  snippet = s[start:end+1]\n  try:\n    data = json.loads(snippet)\n    print(data.get(\"txhash\") or data.get(\"tx_hash\") or \"\")\n    raise SystemExit(0)\n  except Exception:\n    pass\nprint(\"\")\n')"
if [ -z "$COMMIT_HASH" ]; then
  echo "ERROR: update-deal-content returned no txhash" >&2
  echo "$COMMIT_OUT_RAW" >&2
  exit 1
fi
wait_for_tx "$COMMIT_HASH" >/dev/null || { echo "ERROR: update-deal-content tx not found" >&2; exit 1; }

for _ in $(seq 1 20); do
  DEAL_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID")"
  ONCHAIN_ROOT="$(printf '%s' "$DEAL_JSON" | python3 -c $'import json, sys\ntry:\n  data = json.load(sys.stdin)\n  deal = data.get(\"deal\") or {}\n  print(deal.get(\"manifest_root\") or \"\")\nexcept Exception:\n  print(\"\")\n')"
  if [ -n "$ONCHAIN_ROOT" ]; then
    break
  fi
  sleep 1
done
if [ -z "${ONCHAIN_ROOT:-}" ]; then
  echo "ERROR: deal manifest_root not set after update-deal-content" >&2
  echo "$DEAL_JSON" >&2
  exit 1
fi

PLAN_RESP="$(timeout 10s curl -sS "$GATEWAY_BASE/gateway/plan-retrieval-session/$MANIFEST_ROOT?deal_id=$DEAL_ID&owner=$FAUCET_ADDR&file_path=$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))' "$FILE_PATH")&range_start=0&range_len=$RAW_BLOB_PAYLOAD_BYTES")"
PLAN_PROVIDER="$(echo "$PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("provider",""))' 2>/dev/null || true)"
PLAN_START_MDU="$(echo "$PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("start_mdu_index",""))' 2>/dev/null || true)"
PLAN_START_BLOB="$(echo "$PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("start_blob_index",""))' 2>/dev/null || true)"
PLAN_BLOB_COUNT="$(echo "$PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("blob_count",""))' 2>/dev/null || true)"
if [ -z "$PLAN_PROVIDER" ] || [ -z "$PLAN_START_MDU" ] || [ -z "$PLAN_START_BLOB" ] || [ -z "$PLAN_BLOB_COUNT" ]; then
  echo "ERROR: plan response missing required fields" >&2
  echo "$PLAN_RESP" >&2
  exit 1
fi

echo "==> Opening retrieval session..."
HEIGHT="$(rpc_height)"
if [ "$HEIGHT" -le 0 ]; then
  echo "ERROR: failed to resolve chain height" >&2
  exit 1
fi
EXPIRES_AT="$((HEIGHT + 200))"
NONCE="$(python3 - <<'PY'
import time
print(time.time_ns())
PY
)"

OPEN_TX_RAW="$($NILCHAIND_BIN tx nilchain open-retrieval-session \
  --deal-id "$DEAL_ID" \
  --provider "$PLAN_PROVIDER" \
  --manifest-root "$MANIFEST_ROOT" \
  --start-mdu-index "$PLAN_START_MDU" \
  --start-blob-index "$PLAN_START_BLOB" \
  --blob-count "$PLAN_BLOB_COUNT" \
  --nonce "$NONCE" \
  --expires-at "$EXPIRES_AT" \
  --from faucet \
  --chain-id "$CHAIN_ID" \
  --node tcp://127.0.0.1:26657 \
  --home "$CHAIN_HOME" \
  --keyring-backend test \
  --yes \
  --gas auto \
  --gas-adjustment 1.6 \
  --gas-prices 0.001aatom \
  --broadcast-mode sync \
  --output json)"
OPEN_TX_HASH="$(printf '%s' "$OPEN_TX_RAW" | python3 -c $'import json, re, sys\ns = sys.stdin.read()\nmatches = re.findall(r\"\\\"txhash\\\"\\\\s*:\\\\s*\\\"([^\\\"]+)\\\"\", s)\nif matches:\n  print(matches[-1])\n  raise SystemExit(0)\nstart = s.find(\"{\")\nend = s.rfind(\"}\")\nif start != -1 and end != -1 and end > start:\n  snippet = s[start:end+1]\n  try:\n    data = json.loads(snippet)\n    print(data.get(\"txhash\") or data.get(\"tx_hash\") or \"\")\n    raise SystemExit(0)\n  except Exception:\n    pass\nprint(\"\")\n')"
if [ -z "$OPEN_TX_HASH" ]; then
  echo "ERROR: open-retrieval-session returned no txhash" >&2
  echo "$OPEN_TX_RAW" >&2
  exit 1
fi
wait_for_tx "$OPEN_TX_HASH" >/dev/null || { echo "ERROR: retrieval session tx not found" >&2; exit 1; }

SESSION_B64=""
for _ in $(seq 1 30); do
  SESSIONS_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/retrieval-sessions/by-owner/$FAUCET_ADDR" || echo "{}")"
  SESSION_B64="$(echo "$SESSIONS_JSON" | DEAL_ID="$DEAL_ID" NONCE="$NONCE" python3 -c '
import base64, json, os, sys
deal_id = str(os.environ.get("DEAL_ID", ""))
nonce = int(os.environ.get("NONCE", "0") or 0)
try:
  data = json.load(sys.stdin)
  sessions = data.get("sessions") or []
  raw = ""
  for s in sessions:
    if str(s.get("deal_id", "")) != deal_id:
      continue
    try:
      if int(s.get("nonce", 0)) != nonce:
        continue
    except Exception:
      continue
    raw = s.get("session_id") or ""
    break
  if not raw:
    print("")
    raise SystemExit(0)
  print(raw)
except Exception:
  print("")
')"
  if [ -n "$SESSION_B64" ]; then
    break
  fi
  sleep 1
done
if [ -z "$SESSION_B64" ]; then
  echo "ERROR: failed to resolve session id" >&2
  echo "$SESSIONS_JSON" >&2
  exit 1
fi

echo "==> Resolving provider key for proof submitter..."
PROVIDER_KEY="$(resolve_key_for_address "$PLAN_PROVIDER")"
if [ -z "$PROVIDER_KEY" ]; then
  echo "ERROR: failed to resolve key name for provider $PLAN_PROVIDER" >&2
  exit 1
fi

BOND_BEFORE="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/providers/$PLAN_PROVIDER/bond" | python3 -c 'import json, sys
try:
  data=json.load(sys.stdin)
  bond=(data.get("bond") or {}).get("bonded_amount") or {}
  print(bond.get("amount") or "")
except Exception:
  print("")
' 2>/dev/null || true)"

INVALID_PROOF_JSON="$TMP_DIR/invalid_session_proof.json"
PLAN_START_MDU="$PLAN_START_MDU" PLAN_START_BLOB="$PLAN_START_BLOB" PLAN_BLOB_COUNT="$PLAN_BLOB_COUNT" SESSION_B64="$SESSION_B64" \
  python3 - <<'PY' >"$INVALID_PROOF_JSON"
import json
import os
blob_count = int(os.environ.get("PLAN_BLOB_COUNT", "1"))
start_mdu = int(os.environ.get("PLAN_START_MDU", "0"))
start_blob = int(os.environ.get("PLAN_START_BLOB", "0"))
session_b64 = os.environ.get("SESSION_B64", "")
wrong_mdu = start_mdu + 1
proofs = []
for _ in range(blob_count):
  proofs.append({
    "mdu_index": wrong_mdu,
    "mdu_root_fr": "",
    "manifest_opening": "",
    "blob_commitment": "",
    "merkle_path": [],
    "blob_index": start_blob,
    "z_value": "",
    "y_value": "",
    "kzg_opening_proof": "",
  })
print(json.dumps({"session_id": session_b64, "proofs": proofs}, indent=2))
PY

echo "==> Submitting invalid retrieval session proof (expect wrong-data evidence)..."
SUBMIT_OUT_RAW="$($NILCHAIND_BIN tx nilchain submit-retrieval-proof "$INVALID_PROOF_JSON" \
  --from "$PROVIDER_KEY" \
  --chain-id "$CHAIN_ID" \
  --node tcp://127.0.0.1:26657 \
  --home "$CHAIN_HOME" \
  --keyring-backend test \
  --yes \
  --gas auto \
  --gas-adjustment 1.6 \
  --gas-prices 0.001aatom \
  --broadcast-mode sync \
  --output json)"
SUBMIT_HASH="$(printf '%s' "$SUBMIT_OUT_RAW" | python3 -c $'import json, re, sys\ns = sys.stdin.read()\nmatches = re.findall(r\"\\\"txhash\\\"\\\\s*:\\\\s*\\\"([^\\\"]+)\\\"\", s)\nif matches:\n  print(matches[-1])\n  raise SystemExit(0)\nstart = s.find(\"{\")\nend = s.rfind(\"}\")\nif start != -1 and end != -1 and end > start:\n  snippet = s[start:end+1]\n  try:\n    data = json.loads(snippet)\n    print(data.get(\"txhash\") or data.get(\"tx_hash\") or \"\")\n    raise SystemExit(0)\n  except Exception:\n    pass\nprint(\"\")\n')"
if [ -z "$SUBMIT_HASH" ]; then
  echo "ERROR: submit-retrieval-proof returned no txhash" >&2
  echo "$SUBMIT_OUT_RAW" >&2
  exit 1
fi
wait_for_tx "$SUBMIT_HASH" >/dev/null || { echo "ERROR: submit proof tx not found" >&2; exit 1; }

STATUS_AFTER="$(wait_for_provider_status "$PLAN_PROVIDER" "Jailed" 30 1 || true)"
if [ "$STATUS_AFTER" != "Jailed" ]; then
  echo "ERROR: expected provider to be Jailed, got status=$STATUS_AFTER" >&2
  exit 1
fi

echo "==> Checking provider bond slash..."
BOND_AFTER="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/providers/$PLAN_PROVIDER/bond" | python3 -c 'import json, sys
try:
  data=json.load(sys.stdin)
  bond=(data.get("bond") or {}).get("bonded_amount") or {}
  print(bond.get("amount") or "")
except Exception:
  print("")
' 2>/dev/null || true)"
if [ -z "$BOND_BEFORE" ] || [ -z "$BOND_AFTER" ]; then
  echo "ERROR: failed to resolve provider bond amounts" >&2
  exit 1
fi
if [ "$BOND_BEFORE" -le 0 ]; then
  echo "ERROR: expected provider bond to be > 0 before slashing" >&2
  exit 1
fi
if [ "$BOND_AFTER" -ge "$BOND_BEFORE" ]; then
  echo "ERROR: expected provider bond to decrease after slashing (before=$BOND_BEFORE after=$BOND_AFTER)" >&2
  exit 1
fi

SLOT_JSON="$(wait_for_repair_slot "$DEAL_ID" "$PLAN_PROVIDER" 30 1)"
if [ -z "$SLOT_JSON" ]; then
  echo "ERROR: failed to resolve mode2 slot for provider" >&2
  exit 1
fi
SLOT_STATUS="$(echo "$SLOT_JSON" | python3 -c 'import sys, json
try:
  print((json.load(sys.stdin).get("status") or ""))
except Exception:
  print("")
')"
PENDING_PROVIDER="$(echo "$SLOT_JSON" | python3 -c 'import sys, json
try:
  print((json.load(sys.stdin).get("pending_provider") or ""))
except Exception:
  print("")
')"
if [ "$SLOT_STATUS" != "SLOT_STATUS_REPAIRING" ] && [ "$SLOT_STATUS" != "2" ]; then
  echo "ERROR: expected slot to be REPAIRING, got status=$SLOT_STATUS" >&2
  echo "$SLOT_JSON" >&2
  exit 1
fi
if [ -z "$PENDING_PROVIDER" ]; then
  echo "ERROR: expected pending_provider to be set" >&2
  echo "$SLOT_JSON" >&2
  exit 1
fi

echo "✅ Evidence E2E passed: provider jailed, bond slashed, slot repairing (pending=$PENDING_PROVIDER)."
