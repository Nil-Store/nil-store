#!/usr/bin/env bash
# E2E: Mode2 deputy retrieval triggers repair (make-before-break).
#
# Scenario:
# 1) Start devnet alpha multi-SP stack.
# 2) Create a Mode 2 deal.
# 3) Upload + commit a file (NilFS).
# 4) Plan a retrieval session for the first blob and open it on-chain.
# 5) Kill the assigned slot provider ("ghost").
# 6) Fetch through the router: it should fall back to a deputy provider.
# 7) Deputy submits the on-chain retrieval proof.
# 8) At epoch end, chain marks slot as REPAIRING with PendingProvider and the
#    gateway planner routes around the repairing slot.
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
MANAGE_STACK="${MANAGE_STACK:-1}"
EXPECT_REPAIR="${EXPECT_REPAIR:-1}"
STACK_STARTED=0

UPLOAD_FILE="${UPLOAD_FILE:-$ROOT_DIR/test_1mb.bin}"
FILE_PATH="${FILE_PATH:-test_1mb.bin}"

RAW_BLOB_PAYLOAD_BYTES="${RAW_BLOB_PAYLOAD_BYTES:-126976}"
REPAIR_BLOB_COUNT="${REPAIR_BLOB_COUNT:-4}"

# Ensure credits can satisfy repair readiness in this E2E (caps default to 0 on devnet).
: "${NIL_CREDIT_CAP_BPS_HOT:=10000}"
: "${NIL_CREDIT_CAP_BPS_COLD:=10000}"
export NIL_CREDIT_CAP_BPS_HOT
export NIL_CREDIT_CAP_BPS_COLD

cleanup() {
  if [ "$STACK_STARTED" -eq 1 ]; then
    echo "==> Stopping devnet alpha multi-SP stack..."
    "$STACK_SCRIPT" stop || true
  fi
}
if [ "$MANAGE_STACK" -eq 1 ]; then
  trap cleanup EXIT
fi

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

wait_for_height() {
  local target="$1"
  local attempts="${2:-120}"
  local delay="${3:-1}"
  for _ in $(seq 1 "$attempts"); do
    local h
    h="$(rpc_height)"
    if [ "$h" -ge "$target" ]; then
      return 0
    fi
    sleep "$delay"
  done
  return 1
}

json_get() {
  local key="$1"
  python3 -c '
import json, sys
key = sys.argv[1]
data = json.load(sys.stdin)
cur = data
for part in key.split("."):
  if part == "":
    continue
  if isinstance(cur, dict) and part in cur:
    cur = cur[part]
    continue
  print("")
  sys.exit(0)
if cur is None:
  print("")
elif isinstance(cur, (dict, list)):
  print(json.dumps(cur))
else:
  print(cur)
' "$key"
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
  # Fallback: try to parse the last JSON object in the stream.
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
tx = json.load(sys.stdin)
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
      sys.exit(0)
print("")
'
}

decode_session_id_hex() {
  python3 -c '
import base64, json, sys
data = json.load(sys.stdin)
raw = data.get("session_id") or ""
if not raw:
  print("")
  sys.exit(0)
try:
  bz = base64.b64decode(raw)
except Exception:
  try:
    bz = base64.urlsafe_b64decode(raw + "==")
  except Exception:
    print("")
    sys.exit(0)
print("0x" + bz.hex())
'
}

extract_tcp_port() {
  python3 -c '
import re, sys
ep = sys.stdin.read().strip()
m = re.search(r"/tcp/(\d+)(?:/|$)", ep)
print(m.group(1) if m else "")
'
}

require_cmd curl
require_cmd python3

if [ ! -f "$UPLOAD_FILE" ]; then
  echo "ERROR: UPLOAD_FILE does not exist: $UPLOAD_FILE" >&2
  exit 1
fi

# Speed up the repair loop for E2E.
export PROVIDER_COUNT
export START_WEB="${START_WEB:-0}"
export NIL_EPOCH_LEN_BLOCKS="${NIL_EPOCH_LEN_BLOCKS:-20}"
export NIL_EVICT_AFTER_MISSED_EPOCHS="${NIL_EVICT_AFTER_MISSED_EPOCHS:-1}"
export NIL_EVICT_AFTER_MISSED_EPOCHS_HOT="${NIL_EVICT_AFTER_MISSED_EPOCHS_HOT:-1}"
export NIL_EVICT_AFTER_MISSED_EPOCHS_COLD="${NIL_EVICT_AFTER_MISSED_EPOCHS_COLD:-1}"
export NIL_QUOTA_MIN_BLOBS="${NIL_QUOTA_MIN_BLOBS:-1}"
export NIL_QUOTA_MAX_BLOBS="${NIL_QUOTA_MAX_BLOBS:-1}"

if [ "$MANAGE_STACK" -eq 1 ]; then
  echo "==> Starting devnet alpha multi-SP stack (providers=$PROVIDER_COUNT)..."
  "$STACK_SCRIPT" start
  STACK_STARTED=1
fi

wait_for_http "lcd" "$LCD_BASE/cosmos/base/tendermint/v1beta1/node_info" "200" 60 1
wait_for_http "nilchain lcd" "$LCD_BASE/nilchain/nilchain/v1/params" "200" 60 1
wait_for_http "gateway router" "$GATEWAY_BASE/health" "200" 60 1

FAUCET_ADDR="$("$NILCHAIND_BIN" keys show faucet -a --home "$CHAIN_HOME" --keyring-backend test 2>/dev/null || true)"
if [ -z "$FAUCET_ADDR" ]; then
  echo "ERROR: failed to resolve faucet address" >&2
  exit 1
fi
echo "==> Using deal owner: $FAUCET_ADDR"

SERVICE_HINT="General:replicas=${PROVIDER_COUNT}:rs=8+4"

echo "==> Creating Mode 2 deal..."
DEAL_ID=""
for attempt in $(seq 1 5); do
  CREATE_RES_RAW="$("$NILCHAIND_BIN" tx nilchain create-deal 200 1000000 500000 \
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
    CREATE_TX_RAW="$("$NILCHAIND_BIN" query tx "$TXHASH" --node tcp://127.0.0.1:26657 --output json --home "$CHAIN_HOME" 2>/dev/null || true)"
    CREATE_TX="$(echo "$CREATE_TX_RAW" | extract_last_json)"
    DEAL_ID="$(echo "$CREATE_TX" | parse_create_deal_id)"
    if [ -n "$DEAL_ID" ]; then
      break
    fi
  done

  if [ -n "$DEAL_ID" ]; then
    break
  fi
  echo "WARN: create-deal tx not found yet (attempt $attempt)" >&2
done
if [ -z "$DEAL_ID" ]; then
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
FILENAME="$(echo "$UPLOAD_RESP" | python3 -c 'import sys, json; j=json.load(sys.stdin); print(j.get("filename") or j.get("file_path") or "")' 2>/dev/null || true)"

if [ -z "$MANIFEST_ROOT" ] || [ -z "$SIZE_BYTES" ] || [ -z "$TOTAL_MDUS" ] || [ -z "$WITNESS_MDUS" ] || [ -z "$FILENAME" ]; then
  echo "ERROR: upload response missing required fields" >&2
  echo "$UPLOAD_RESP" >&2
  exit 1
fi
MAX_BLOBS="$(( (SIZE_BYTES + RAW_BLOB_PAYLOAD_BYTES - 1) / RAW_BLOB_PAYLOAD_BYTES ))"
if [ "$MAX_BLOBS" -lt 1 ]; then
  MAX_BLOBS=1
fi
if [ "$REPAIR_BLOB_COUNT" -gt "$MAX_BLOBS" ]; then
  REPAIR_BLOB_COUNT="$MAX_BLOBS"
fi
echo "    manifest_root=$MANIFEST_ROOT size_bytes=$SIZE_BYTES total_mdus=$TOTAL_MDUS witness_mdus=$WITNESS_MDUS file=$FILENAME"

echo "==> Committing deal content on-chain..."
"$NILCHAIND_BIN" tx nilchain update-deal-content \
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
  --output json >/dev/null
sleep 2

echo "==> Waiting for deal manifest_root to be visible..."
for _ in $(seq 1 30); do
  DEAL_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID" || echo "{}")"
  CHAIN_ROOT_HEX="$(echo "$DEAL_JSON" | python3 -c '
import base64, json, sys
d = json.load(sys.stdin)
deal = d.get("deal") or {}
root = deal.get("manifest_root") or ""
if isinstance(root, str) and root.startswith("0x"):
  print(root)
  raise SystemExit(0)
if not root:
  print("")
  raise SystemExit(0)
try:
  bz = base64.b64decode(root)
except Exception:
  try:
    bz = base64.urlsafe_b64decode(root + "==")
  except Exception:
    print("")
    raise SystemExit(0)
print("0x" + bz.hex())
' 2>/dev/null || true)"
  if [ "$CHAIN_ROOT_HEX" = "$MANIFEST_ROOT" ]; then
    break
  fi
  sleep 1
done
if [ "${CHAIN_ROOT_HEX:-}" != "$MANIFEST_ROOT" ]; then
  echo "ERROR: deal manifest_root not updated on-chain" >&2
  echo "    expected=$MANIFEST_ROOT got=${CHAIN_ROOT_HEX:-}" >&2
  echo "$DEAL_JSON" >&2
  exit 1
fi

echo "==> Planning retrieval session for first blob..."
PLAN_RESP="$(timeout 10s curl -sS "$GATEWAY_BASE/gateway/plan-retrieval-session/$MANIFEST_ROOT?deal_id=$DEAL_ID&owner=$FAUCET_ADDR&file_path=$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))' "$FILENAME")&range_start=0&range_len=$RAW_BLOB_PAYLOAD_BYTES")"
PLAN_PROVIDER="$(echo "$PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("provider",""))' 2>/dev/null || true)"
PLAN_START_MDU="$(echo "$PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("start_mdu_index",""))' 2>/dev/null || true)"
PLAN_START_BLOB="$(echo "$PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("start_blob_index",""))' 2>/dev/null || true)"
PLAN_BLOB_COUNT="$(echo "$PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("blob_count",""))' 2>/dev/null || true)"
if [ -z "$PLAN_PROVIDER" ] || [ -z "$PLAN_START_MDU" ] || [ -z "$PLAN_START_BLOB" ] || [ -z "$PLAN_BLOB_COUNT" ]; then
  echo "ERROR: plan response missing required fields" >&2
  echo "$PLAN_RESP" >&2
  exit 1
fi
echo "    planned slot provider=$PLAN_PROVIDER start_mdu=$PLAN_START_MDU start_blob=$PLAN_START_BLOB blob_count=$PLAN_BLOB_COUNT"

declare -a REPAIR_START_MDUS
declare -a REPAIR_START_BLOBS
declare -a REPAIR_RANGE_STARTS
declare -a REPAIR_BLOB_COUNTS
REPAIR_START_MDUS[0]="$PLAN_START_MDU"
REPAIR_START_BLOBS[0]="$PLAN_START_BLOB"
REPAIR_RANGE_STARTS[0]=0
REPAIR_BLOB_COUNTS[0]="$PLAN_BLOB_COUNT"
repair_idx=1
if [ "$REPAIR_BLOB_COUNT" -gt 1 ]; then
  for i in $(seq 1 $((REPAIR_BLOB_COUNT - 1))); do
    range_start="$((i * RAW_BLOB_PAYLOAD_BYTES))"
    REPAIR_PLAN_RESP="$(timeout 10s curl -sS "$GATEWAY_BASE/gateway/plan-retrieval-session/$MANIFEST_ROOT?deal_id=$DEAL_ID&owner=$FAUCET_ADDR&file_path=$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))' "$FILENAME")&range_start=$range_start&range_len=$RAW_BLOB_PAYLOAD_BYTES")"
    range_provider="$(echo "$REPAIR_PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("provider",""))' 2>/dev/null || true)"
    start_mdu="$(echo "$REPAIR_PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("start_mdu_index",""))' 2>/dev/null || true)"
    start_blob="$(echo "$REPAIR_PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("start_blob_index",""))' 2>/dev/null || true)"
    blob_count="$(echo "$REPAIR_PLAN_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("blob_count",""))' 2>/dev/null || true)"
    if [ -z "$range_provider" ] || [ -z "$start_mdu" ] || [ -z "$start_blob" ] || [ -z "$blob_count" ]; then
      echo "ERROR: repair plan response missing required fields" >&2
      echo "$REPAIR_PLAN_RESP" >&2
      exit 1
    fi
    if [ "$range_provider" != "$PLAN_PROVIDER" ]; then
      echo "    skipping range_start=$range_start (provider=$range_provider != $PLAN_PROVIDER)"
      continue
    fi
    REPAIR_START_MDUS[$repair_idx]="$start_mdu"
    REPAIR_START_BLOBS[$repair_idx]="$start_blob"
    REPAIR_RANGE_STARTS[$repair_idx]="$range_start"
    REPAIR_BLOB_COUNTS[$repair_idx]="$blob_count"
    repair_idx=$((repair_idx + 1))
  done
fi
REPAIR_PLAN_COUNT="${#REPAIR_START_MDUS[@]}"
if [ "$REPAIR_PLAN_COUNT" -lt 1 ]; then
  echo "ERROR: no repair ranges match planned provider $PLAN_PROVIDER" >&2
  exit 1
fi

echo "==> Resolving planned provider endpoint..."
PROVIDER_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/providers/$PLAN_PROVIDER")"
ENDPOINT="$(echo "$PROVIDER_JSON" | python3 -c 'import sys, json; d=json.load(sys.stdin); p=d.get("provider") or {}; eps=p.get("endpoints") or []; print(eps[0] if eps else "")' 2>/dev/null || true)"
PORT="$(echo "$ENDPOINT" | extract_tcp_port)"
if [ -z "$PORT" ]; then
  echo "ERROR: failed to parse provider endpoint port from: $ENDPOINT" >&2
  exit 1
fi
echo "    planned provider endpoint=$ENDPOINT port=$PORT"

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

echo "==> Opening on-chain retrieval session..."
"$NILCHAIND_BIN" tx nilchain open-retrieval-session \
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
  --output json >/dev/null

echo "==> Waiting for retrieval session to appear..."
SESSION_HEX=""
SESSION_JSON=""
SESSION_PREMIUM=""
for _ in $(seq 1 30); do
  SESSIONS_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/retrieval-sessions/by-owner/$FAUCET_ADDR" || echo "{}")"
  SESSION_JSON="$(echo "$SESSIONS_JSON" | DEAL_ID="$DEAL_ID" NONCE="$NONCE" python3 -c '
import json, os, sys
deal_id = str(os.environ.get("DEAL_ID",""))
nonce = int(os.environ.get("NONCE","0") or 0)
data = json.load(sys.stdin)
sessions = data.get("sessions") or []
for s in sessions:
  if str(s.get("deal_id","")) != deal_id:
    continue
  try:
    if int(s.get("nonce",0)) != nonce:
      continue
  except Exception:
    continue
  print(json.dumps(s))
  raise SystemExit(0)
print("")
')"
  if [ -n "$SESSION_JSON" ]; then
    SESSION_HEX="$(echo "$SESSION_JSON" | decode_session_id_hex)"
    SESSION_PREMIUM="$(echo "$SESSION_JSON" | python3 -c 'import json, sys; print(json.load(sys.stdin).get("locked_premium_fee",""))' 2>/dev/null || true)"
    break
  fi
  sleep 1
done
if [ -z "$SESSION_HEX" ] || [ -z "$SESSION_JSON" ]; then
  echo "ERROR: failed to resolve session id" >&2
  echo "$SESSIONS_JSON" >&2
  exit 1
fi
if [ -z "$SESSION_PREMIUM" ]; then
  echo "ERROR: failed to resolve locked premium fee" >&2
  echo "$SESSION_JSON" >&2
  exit 1
fi
if ! python3 -c "import sys; print(int(sys.argv[1]) > 0)" "$SESSION_PREMIUM" | grep -q True; then
  echo "ERROR: expected locked premium fee to be > 0 (got $SESSION_PREMIUM)" >&2
  exit 1
fi
echo "    session_id=$SESSION_HEX"
echo "    locked_premium_fee=$SESSION_PREMIUM"

DEAL_ESCROW_BEFORE="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID" | json_get "deal.escrow_balance")"
if [ -z "$DEAL_ESCROW_BEFORE" ]; then
  echo "ERROR: failed to resolve deal escrow balance before settlement" >&2
  exit 1
fi
echo "    escrow_before=$DEAL_ESCROW_BEFORE"

echo "==> Simulating ghosting: stopping planned provider..."
# Only kill the listener on that port (avoid killing the router, which may have
# outbound connections to the provider port).
PIDS="$(lsof -nP -iTCP:"$PORT" -sTCP:LISTEN -t 2>/dev/null || true)"
if [ -n "$PIDS" ]; then
  kill $PIDS 2>/dev/null || true
fi
sleep 1

echo "==> Fetching first blob via router (should fall back to deputy)..."
OUT_FILE="$(mktemp)"
HDR_FILE="$(mktemp)"
start_end="$((RAW_BLOB_PAYLOAD_BYTES - 1))"
FETCH_EXIT=0
HTTP_CODE="$(timeout 120s curl -sS -D "$HDR_FILE" -o "$OUT_FILE" \
  -H "X-Nil-Session-Id: $SESSION_HEX" \
  -H "Range: bytes=0-${start_end}" \
  "$GATEWAY_BASE/gateway/fetch/$MANIFEST_ROOT?deal_id=$DEAL_ID&owner=$FAUCET_ADDR&file_path=$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))' "$FILENAME")" \
  -w '%{http_code}')" || FETCH_EXIT=$?

if [ "$FETCH_EXIT" -ne 0 ]; then
  echo "ERROR: fetch request failed (exit=$FETCH_EXIT)" >&2
  if [ -s "$HDR_FILE" ]; then
    echo "---- response headers ----" >&2
    cat "$HDR_FILE" >&2 || true
    echo "--------------------------" >&2
  fi
  exit 1
fi

if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "206" ]; then
  echo "ERROR: fetch returned HTTP $HTTP_CODE" >&2
  echo "---- response headers ----" >&2
  cat "$HDR_FILE" >&2 || true
  echo "--------------------------" >&2
  if [ -s "$OUT_FILE" ]; then
    echo "---- response body (first 4KB) ----" >&2
    head -c 4096 "$OUT_FILE" >&2 || true
    echo "-----------------------------------" >&2
  fi
  exit 1
fi

DEPUTY_PROVIDER="$(grep -i '^X-Nil-Provider:' "$HDR_FILE" | tail -n 1 | awk '{print $2}' | tr -d '\r')"
if [ -z "$DEPUTY_PROVIDER" ]; then
  echo "ERROR: missing X-Nil-Provider header" >&2
  cat "$HDR_FILE" >&2 || true
  exit 1
fi
echo "    fetch HTTP $HTTP_CODE via provider=$DEPUTY_PROVIDER"
if [ "$DEPUTY_PROVIDER" = "$PLAN_PROVIDER" ]; then
  echo "ERROR: expected a deputy provider (got planned provider)" >&2
  exit 1
fi

OUT_BYTES="$(python3 - <<PY
import os
print(os.path.getsize("$OUT_FILE"))
PY
)"
if [ "$OUT_BYTES" -le 0 ]; then
  echo "ERROR: fetched zero bytes" >&2
  exit 1
fi

echo "==> Asking deputy to submit retrieval session proof..."
PROOF_SUBMIT_RESP="$(timeout 120s curl -sS -X POST "$GATEWAY_BASE/gateway/session-proof" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\":\"$SESSION_HEX\",\"provider\":\"$DEPUTY_PROVIDER\"}")"
STATUS="$(echo "$PROOF_SUBMIT_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("status",""))' 2>/dev/null || true)"
if [ "$STATUS" != "success" ]; then
  echo "ERROR: session proof submission failed" >&2
  echo "$PROOF_SUBMIT_RESP" >&2
  exit 1
fi

echo "==> Confirming retrieval session (user validation)..."
"$NILCHAIND_BIN" tx nilchain confirm-retrieval-session "$SESSION_HEX" \
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
  --output json >/dev/null

echo "==> Waiting for retrieval session to complete..."
SESSION_STATUS=""
SESSION_PREMIUM_AFTER=""
for _ in $(seq 1 30); do
  SESSIONS_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/retrieval-sessions/by-owner/$FAUCET_ADDR" || echo "{}")"
  SESSION_STATUS="$(echo "$SESSIONS_JSON" | SESSION_HEX="$SESSION_HEX" python3 -c '
import base64, json, os, sys
target = os.environ.get("SESSION_HEX","").lower().replace("0x","")
data = json.load(sys.stdin)
sessions = data.get("sessions") or []
for s in sessions:
  raw = s.get("session_id") or ""
  if not raw:
    continue
  try:
    bz = base64.b64decode(raw)
  except Exception:
    try:
      bz = base64.urlsafe_b64decode(raw + "==")
    except Exception:
      continue
  if bz.hex() != target:
    continue
  status = s.get("status")
  if isinstance(status, (int, float)):
    print(str(int(status)))
  else:
    print(str(status or ""))
  raise SystemExit(0)
print("")
')"
  SESSION_PREMIUM_AFTER="$(echo "$SESSIONS_JSON" | SESSION_HEX="$SESSION_HEX" python3 -c '
import base64, json, os, sys
target = os.environ.get("SESSION_HEX","").lower().replace("0x","")
data = json.load(sys.stdin)
sessions = data.get("sessions") or []
for s in sessions:
  raw = s.get("session_id") or ""
  if not raw:
    continue
  try:
    bz = base64.b64decode(raw)
  except Exception:
    try:
      bz = base64.urlsafe_b64decode(raw + "==")
    except Exception:
      continue
  if bz.hex() != target:
    continue
  print(s.get("locked_premium_fee",""))
  raise SystemExit(0)
print("")
')"
  if [ "$SESSION_STATUS" = "RETRIEVAL_SESSION_STATUS_COMPLETED" ] || [ "$SESSION_STATUS" = "4" ]; then
    break
  fi
  sleep 1
done
if [ "$SESSION_STATUS" != "RETRIEVAL_SESSION_STATUS_COMPLETED" ] && [ "$SESSION_STATUS" != "4" ]; then
  echo "ERROR: retrieval session did not complete (status=$SESSION_STATUS)" >&2
  exit 1
fi
if [ -z "$SESSION_PREMIUM_AFTER" ]; then
  echo "ERROR: failed to resolve locked premium after completion" >&2
  exit 1
fi
if ! python3 -c "import sys; print(int(sys.argv[1]) == 0)" "$SESSION_PREMIUM_AFTER" | grep -q True; then
  echo "ERROR: expected locked premium to be zero after completion (got $SESSION_PREMIUM_AFTER)" >&2
  exit 1
fi

DEAL_ESCROW_AFTER="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID" | json_get "deal.escrow_balance")"
if [ -z "$DEAL_ESCROW_AFTER" ]; then
  echo "ERROR: failed to resolve deal escrow balance after settlement" >&2
  exit 1
fi
python3 - <<PY
import sys
before = int("$DEAL_ESCROW_BEFORE")
after = int("$DEAL_ESCROW_AFTER")
premium = int("$SESSION_PREMIUM")
if after - before >= premium:
  print("ERROR: escrow increased by premium fee (proxy premium likely refunded)", file=sys.stderr)
  sys.exit(1)
PY

if [ "$EXPECT_REPAIR" -eq 0 ]; then
  echo "==> Deputy ghosting E2E passed."
  exit 0
fi

echo "==> Waiting for epoch end to trigger deputy-miss repair..."
EPOCH_LEN="$NIL_EPOCH_LEN_BLOCKS"
CUR_H="$(rpc_height)"
NEXT_EPOCH_END="$(( ( (CUR_H + EPOCH_LEN - 1) / EPOCH_LEN ) * EPOCH_LEN ))"
if [ "$NEXT_EPOCH_END" -le "$CUR_H" ]; then
  NEXT_EPOCH_END="$((CUR_H + EPOCH_LEN))"
fi
wait_for_height "$NEXT_EPOCH_END" 180 1 || { echo "ERROR: timed out waiting for epoch end" >&2; exit 1; }
sleep 2

DEAL_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID")"
REPAIR_SLOT_JSON="$(echo "$DEAL_JSON" | PLANNED_PROVIDER="$PLAN_PROVIDER" python3 -c '
import json, os, sys
planned = (os.environ.get("PLANNED_PROVIDER","") or "").strip()
data = json.load(sys.stdin)
deal = data.get("deal") or {}
slots = deal.get("mode2_slots") or []
for s in slots:
  if not s:
    continue
  if (s.get("provider") or "").strip() != planned:
    continue
  print(json.dumps(s))
  sys.exit(0)
print("")
')"
if [ -z "$REPAIR_SLOT_JSON" ]; then
  echo "ERROR: failed to find slot for planned provider in deal state" >&2
  echo "$DEAL_JSON" >&2
  exit 1
fi
SLOT_STATUS="$(echo "$REPAIR_SLOT_JSON" | python3 -c 'import sys, json; print((json.load(sys.stdin).get("status") or ""))' 2>/dev/null || true)"
PENDING_PROVIDER="$(echo "$REPAIR_SLOT_JSON" | python3 -c 'import sys, json; print((json.load(sys.stdin).get("pending_provider") or ""))' 2>/dev/null || true)"
if [ "$SLOT_STATUS" != "SLOT_STATUS_REPAIRING" ] && [ "$SLOT_STATUS" != "2" ]; then
  echo "ERROR: expected slot to be REPAIRING, got status=$SLOT_STATUS" >&2
  echo "$REPAIR_SLOT_JSON" >&2
  exit 1
fi
if [ -z "$PENDING_PROVIDER" ]; then
  echo "ERROR: expected pending_provider to be set" >&2
  echo "$REPAIR_SLOT_JSON" >&2
  exit 1
fi
echo "    repair started: pending_provider=$PENDING_PROVIDER"

echo "==> Resolving pending provider endpoint..."
PENDING_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/providers/$PENDING_PROVIDER")"
PENDING_ENDPOINT="$(echo "$PENDING_JSON" | python3 -c 'import sys, json; d=json.load(sys.stdin); p=d.get("provider") or {}; eps=p.get("endpoints") or []; print(eps[0] if eps else "")' 2>/dev/null || true)"
PENDING_PORT="$(echo "$PENDING_ENDPOINT" | extract_tcp_port)"
if [ -z "$PENDING_PORT" ]; then
  echo "ERROR: failed to parse pending provider endpoint port from: $PENDING_ENDPOINT" >&2
  exit 1
fi
PENDING_BASE="http://127.0.0.1:$PENDING_PORT"
echo "    pending provider endpoint=$PENDING_ENDPOINT port=$PENDING_PORT"

MANIFEST_KEY="${MANIFEST_ROOT#0x}"
LOG_DIR="${LOG_DIR:-$ROOT_DIR/_artifacts/devnet_alpha_multi_sp}"
PENDING_INDEX="$((PENDING_PORT - 8091 + 1))"
if [ "$PENDING_INDEX" -le 0 ]; then
  echo "ERROR: failed to compute pending provider index from port $PENDING_PORT" >&2
  exit 1
fi
PENDING_DEAL_DIR="$LOG_DIR/providers/provider$PENDING_INDEX/deals/$DEAL_ID/$MANIFEST_KEY"
NEED_COPY=0
if [ ! -f "$PENDING_DEAL_DIR/mdu_0.bin" ] || [ ! -f "$PENDING_DEAL_DIR/manifest.bin" ] || [ ! -f "$PENDING_DEAL_DIR/.slab_complete" ]; then
  NEED_COPY=1
fi
if [ "$NEED_COPY" -eq 0 ]; then
  SHARD_COUNT="$(find "$PENDING_DEAL_DIR" -maxdepth 1 -type f -name 'mdu_*_slot_*.bin' | wc -l | tr -d ' ')"
  if [ "$SHARD_COUNT" -eq 0 ]; then
    NEED_COPY=1
  fi
fi

if [ "$NEED_COPY" -eq 1 ]; then
  SRC_DIR="$(find "$LOG_DIR/router_tmp" -type d \( -path "*/deals/$DEAL_ID/$MANIFEST_KEY" -o -path "*/$MANIFEST_KEY" \) | head -n 1 || true)"
  if [ -z "$SRC_DIR" ]; then
    SRC_DIR="$(find "$LOG_DIR/providers" -type d \( -path "*/deals/$DEAL_ID/$MANIFEST_KEY" -o -path "*/$MANIFEST_KEY" \) | head -n 1 || true)"
  fi
  if [ -z "$SRC_DIR" ]; then
    echo "ERROR: failed to locate source slab for catch-up (deal $DEAL_ID root $MANIFEST_KEY)" >&2
    exit 1
  fi
  if [ "$SRC_DIR" != "$PENDING_DEAL_DIR" ]; then
    echo "==> Copying slab for pending provider: $SRC_DIR -> $PENDING_DEAL_DIR"
    mkdir -p "$(dirname "$PENDING_DEAL_DIR")"
    rm -rf "$PENDING_DEAL_DIR"
    cp -R "$SRC_DIR" "$PENDING_DEAL_DIR"
  fi
fi

if [ -d "$PENDING_DEAL_DIR" ]; then
  echo "==> Aggregating slot shards for pending provider..."
  for src in "$LOG_DIR/providers"/provider*/deals/"$DEAL_ID"/"$MANIFEST_KEY"; do
    if [ -d "$src" ]; then
      find "$src" -maxdepth 1 -type f -name 'mdu_*_slot_*.bin' -exec cp -f {} "$PENDING_DEAL_DIR"/ \;
    fi
  done
fi

SLOT_INDEX="$(echo "$REPAIR_SLOT_JSON" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("slot",""))' 2>/dev/null || true)"
if [ -z "$SLOT_INDEX" ]; then
  echo "ERROR: failed to parse repair slot index" >&2
  echo "$REPAIR_SLOT_JSON" >&2
  exit 1
fi

for i in $(seq 0 $((REPAIR_PLAN_COUNT - 1))); do
  echo "==> Opening retrieval session against pending provider (blob $i)..."
  HEIGHT2="$(rpc_height)"
  if [ "$HEIGHT2" -le 0 ]; then
    echo "ERROR: failed to resolve chain height for repair session" >&2
    exit 1
  fi
  EXPIRES_AT2="$((HEIGHT2 + 200))"
  NONCE2="$(python3 - <<'PY'
import time
print(time.time_ns())
PY
)"

  "$NILCHAIND_BIN" tx nilchain open-retrieval-session \
    --deal-id "$DEAL_ID" \
    --provider "$PENDING_PROVIDER" \
    --manifest-root "$MANIFEST_ROOT" \
    --start-mdu-index "${REPAIR_START_MDUS[$i]}" \
    --start-blob-index "${REPAIR_START_BLOBS[$i]}" \
    --blob-count "${REPAIR_BLOB_COUNTS[$i]}" \
    --nonce "$NONCE2" \
    --expires-at "$EXPIRES_AT2" \
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
    --output json >/dev/null

  echo "==> Waiting for repair session to appear..."
  SESSION2_HEX=""
  for _ in $(seq 1 30); do
    SESSIONS_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/retrieval-sessions/by-owner/$FAUCET_ADDR" || echo "{}")"
    SESSION2_HEX="$(echo "$SESSIONS_JSON" | DEAL_ID="$DEAL_ID" NONCE="$NONCE2" python3 -c '
import base64, json, os, sys
deal_id = str(os.environ.get("DEAL_ID",""))
nonce = int(os.environ.get("NONCE","0") or 0)
data = json.load(sys.stdin)
sessions = data.get("sessions") or []
raw = ""
for s in sessions:
  if str(s.get("deal_id","")) != deal_id:
    continue
  try:
    if int(s.get("nonce",0)) != nonce:
      continue
  except Exception:
    continue
  raw = s.get("session_id") or ""
  break
if not raw:
  print("")
  raise SystemExit(0)
try:
  bz = base64.b64decode(raw)
except Exception:
  try:
    bz = base64.urlsafe_b64decode(raw + "==")
  except Exception:
    print("")
    raise SystemExit(0)
print("0x" + bz.hex())
')"
    if [ -n "$SESSION2_HEX" ]; then
      break
    fi
    sleep 1
  done
  if [ -z "$SESSION2_HEX" ]; then
    echo "ERROR: failed to resolve repair session id" >&2
    echo "$SESSIONS_JSON" >&2
    exit 1
  fi
  echo "    session_id=$SESSION2_HEX"

  range_start="${REPAIR_RANGE_STARTS[$i]}"
  range_end="$((range_start + RAW_BLOB_PAYLOAD_BYTES - 1))"
  OUT_FILE2="$(mktemp)"
  HDR_FILE2="$(mktemp)"
  FETCH_EXIT=0
  HTTP_CODE="$(timeout 120s curl -sS -D "$HDR_FILE2" -o "$OUT_FILE2" \
    -H "X-Nil-Session-Id: $SESSION2_HEX" \
    -H "Range: bytes=${range_start}-${range_end}" \
    "$PENDING_BASE/gateway/fetch/$MANIFEST_ROOT?deal_id=$DEAL_ID&owner=$FAUCET_ADDR&file_path=$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))' "$FILENAME")&deputy=1" \
    -w '%{http_code}')" || FETCH_EXIT=$?

  if [ "$FETCH_EXIT" -ne 0 ]; then
    echo "ERROR: repair fetch request failed (exit=$FETCH_EXIT)" >&2
    if [ -s "$HDR_FILE2" ]; then
      echo "---- response headers ----" >&2
      cat "$HDR_FILE2" >&2 || true
      echo "--------------------------" >&2
    fi
    exit 1
  fi

  if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "206" ]; then
    echo "ERROR: repair fetch returned HTTP $HTTP_CODE" >&2
    echo "---- response headers ----" >&2
    cat "$HDR_FILE2" >&2 || true
    echo "--------------------------" >&2
    if [ -s "$OUT_FILE2" ]; then
      echo "---- response body ----" >&2
      cat "$OUT_FILE2" >&2 || true
      echo "------------------------" >&2
    fi
    exit 1
  fi

  REPAIR_PROVIDER="$(grep -i '^X-Nil-Provider:' "$HDR_FILE2" | tail -n 1 | awk '{print $2}' | tr -d '\r')"
  if [ "$REPAIR_PROVIDER" != "$PENDING_PROVIDER" ]; then
    echo "ERROR: expected repair fetch via pending provider $PENDING_PROVIDER (got $REPAIR_PROVIDER)" >&2
    cat "$HDR_FILE2" >&2 || true
    exit 1
  fi

  echo "==> Submitting retrieval session proof for pending provider..."
  SP_AUTH="$(cat "$LOG_DIR/sp_auth.txt" 2>/dev/null || true)"
  if [ -z "$SP_AUTH" ]; then
    echo "ERROR: missing SP auth token at $LOG_DIR/sp_auth.txt" >&2
    exit 1
  fi
  PROOF_SUBMIT_RESP2="$(timeout 120s curl -sS -X POST "$PENDING_BASE/sp/session-proof" \
    -H "Content-Type: application/json" \
    -H "X-Nil-Gateway-Auth: $SP_AUTH" \
    -d "{\"session_id\":\"$SESSION2_HEX\"}")"
  STATUS2="$(echo "$PROOF_SUBMIT_RESP2" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("status",""))' 2>/dev/null || true)"
  if [ "$STATUS2" != "success" ]; then
    echo "ERROR: repair session proof submission failed" >&2
    echo "$PROOF_SUBMIT_RESP2" >&2
    exit 1
  fi
done

echo "==> Waiting for repair proof to land..."
CUR_H2="$(rpc_height)"
TARGET_H2="$((CUR_H2 + 2))"
wait_for_height "$TARGET_H2" 60 1 || { echo "ERROR: timed out waiting for repair proof inclusion" >&2; exit 1; }

echo "==> Completing slot repair on-chain..."
COMPLETE_RESP="$("$NILCHAIND_BIN" tx nilchain complete-slot-repair \
  --deal-id "$DEAL_ID" \
  --slot "$SLOT_INDEX" \
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
  --output json 2>&1)" || {
  echo "ERROR: complete-slot-repair failed" >&2
  echo "$COMPLETE_RESP" >&2
  exit 1
}

COMPLETE_JSON="$(echo "$COMPLETE_RESP" | extract_last_json)"
if [ -z "$COMPLETE_JSON" ]; then
  echo "ERROR: failed to parse complete-slot-repair response" >&2
  echo "$COMPLETE_RESP" >&2
  exit 1
fi
COMPLETE_TXHASH="$(echo "$COMPLETE_JSON" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("txhash",""))' 2>/dev/null || true)"
if [ -z "$COMPLETE_TXHASH" ]; then
  echo "ERROR: missing txhash in complete-slot-repair response" >&2
  echo "$COMPLETE_JSON" >&2
  exit 1
fi

COMPLETE_TX=""
for _ in $(seq 1 10); do
  sleep 1
  COMPLETE_TX_RAW="$("$NILCHAIND_BIN" query tx "$COMPLETE_TXHASH" --node tcp://127.0.0.1:26657 --output json --home "$CHAIN_HOME" 2>/dev/null || true)"
  COMPLETE_TX="$(echo "$COMPLETE_TX_RAW" | extract_last_json)"
  if [ -n "$COMPLETE_TX" ]; then
    COMPLETE_CODE="$(echo "$COMPLETE_TX" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("code",""))' 2>/dev/null || true)"
    if [ -n "$COMPLETE_CODE" ] && [ "$COMPLETE_CODE" != "0" ]; then
      COMPLETE_LOG="$(echo "$COMPLETE_TX" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("raw_log",""))' 2>/dev/null || true)"
      echo "ERROR: complete-slot-repair tx failed (code=$COMPLETE_CODE)" >&2
      echo "$COMPLETE_LOG" >&2
      exit 1
    fi
    break
  fi
done
if [ -z "$COMPLETE_TX" ]; then
  echo "ERROR: complete-slot-repair tx not found" >&2
  exit 1
fi

CUR_H3="$(rpc_height)"
TARGET_H3="$((CUR_H3 + 2))"
wait_for_height "$TARGET_H3" 60 1 || { echo "ERROR: timed out waiting for complete-slot-repair inclusion" >&2; exit 1; }

DEAL_JSON2="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID")"
UPDATED_SLOT_JSON="$(echo "$DEAL_JSON2" | SLOT_INDEX="$SLOT_INDEX" python3 -c '
import json, os, sys
idx = int(os.environ.get("SLOT_INDEX","-1"))
data = json.load(sys.stdin)
deal = data.get("deal") or {}
slots = deal.get("mode2_slots") or []
if idx < 0 or idx >= len(slots):
  print("")
  raise SystemExit(0)
slot = slots[idx]
print(json.dumps(slot) if slot else "")
')"
if [ -z "$UPDATED_SLOT_JSON" ]; then
  echo "ERROR: failed to load updated slot $SLOT_INDEX" >&2
  echo "$DEAL_JSON2" >&2
  exit 1
fi
UPDATED_STATUS="$(echo "$UPDATED_SLOT_JSON" | python3 -c 'import sys, json; print((json.load(sys.stdin).get("status") or ""))' 2>/dev/null || true)"
UPDATED_PROVIDER="$(echo "$UPDATED_SLOT_JSON" | python3 -c 'import sys, json; print((json.load(sys.stdin).get("provider") or ""))' 2>/dev/null || true)"
UPDATED_PENDING="$(echo "$UPDATED_SLOT_JSON" | python3 -c 'import sys, json; print((json.load(sys.stdin).get("pending_provider") or ""))' 2>/dev/null || true)"
if [ "$UPDATED_STATUS" != "SLOT_STATUS_ACTIVE" ] && [ "$UPDATED_STATUS" != "1" ]; then
  echo "ERROR: expected slot to be ACTIVE after repair, got status=$UPDATED_STATUS" >&2
  echo "$UPDATED_SLOT_JSON" >&2
  exit 1
fi
if [ "$UPDATED_PROVIDER" != "$PENDING_PROVIDER" ]; then
  echo "ERROR: expected slot provider to be $PENDING_PROVIDER after repair (got $UPDATED_PROVIDER)" >&2
  echo "$UPDATED_SLOT_JSON" >&2
  exit 1
fi
if [ -n "$UPDATED_PENDING" ]; then
  echo "ERROR: expected pending_provider to be cleared (got $UPDATED_PENDING)" >&2
  echo "$UPDATED_SLOT_JSON" >&2
  exit 1
fi

echo "==> Deputy ghost repair E2E passed."
