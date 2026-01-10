#!/usr/bin/env bash
# End-to-end econ accounting regression test:
# - lock-in deposit on ingest
# - retrieval open burns base + locks variable fee
# - retrieval confirm burns cut + pays provider
# - retrieval cancel refunds locked fee only
set -euo pipefail
set -x

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STACK_SCRIPT="$ROOT_DIR/scripts/run_local_stack.sh"

CHAIN_HOME="${NIL_HOME:-$ROOT_DIR/_artifacts/nilchain_data}"
CHAIN_ID="${CHAIN_ID:-31337}"
RPC_STATUS="${RPC_STATUS:-http://127.0.0.1:26657/status}"
LCD_BASE="${LCD_BASE:-http://127.0.0.1:1317}"
GATEWAY_BASE="${GATEWAY_BASE:-http://127.0.0.1:8080}"

NILCHAIND_BIN="${NILCHAIND_BIN:-$ROOT_DIR/nilchain/nilchaind}"

UPLOAD_FILE="${UPLOAD_FILE:-$ROOT_DIR/README.md}"
FILE_PATH="${FILE_PATH:-README.md}"
RAW_BLOB_PAYLOAD_BYTES="${RAW_BLOB_PAYLOAD_BYTES:-126976}"

DURATION_BLOCKS="${DURATION_BLOCKS:-200}"
INITIAL_ESCROW="${INITIAL_ESCROW:-1000000}"
MAX_MONTHLY_SPEND="${MAX_MONTHLY_SPEND:-500000}"
BROADCAST_MODE="${BROADCAST_MODE:-sync}"

cleanup() {
  echo "==> Stopping local stack..."
  "$STACK_SCRIPT" stop || true
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

wait_for_tx() {
  local txhash="$1"
  local attempts="${2:-40}"
  local delay="${3:-1}"
  if [ -z "$txhash" ]; then
    echo "ERROR: missing txhash" >&2
    return 1
  fi
  for _ in $(seq 1 "$attempts"); do
    local resp
    resp="$(timeout 10s curl -sS "$LCD_BASE/cosmos/tx/v1beta1/txs/$txhash" || true)"
    local code
    code="$(echo "$resp" | python3 -c 'import json, sys
try:
  data=json.load(sys.stdin)
  resp=data.get("tx_response") or {}
  code=resp.get("code")
  print("" if code is None else str(code))
except Exception:
  print("")
')"
    if [ -n "$code" ]; then
      if [ "$code" != "0" ]; then
        echo "ERROR: tx $txhash failed with code $code" >&2
        echo "$resp" >&2
        return 1
      fi
      echo "$resp"
      return 0
    fi
    sleep "$delay"
  done
  echo "ERROR: timed out waiting for tx $txhash" >&2
  return 1
}

get_max_deal_id() {
  timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals" | python3 -c '
import json, sys
try:
  data = json.load(sys.stdin)
  deals = data.get("deals") or []
  ids = [int(d.get("id", 0)) for d in deals if isinstance(d, dict)]
  print(max(ids) if ids else 0)
except Exception:
  print(0)
'
}

get_module_account() {
  local module_name="$1"
  timeout 10s curl -sS "$LCD_BASE/cosmos/auth/v1beta1/module_accounts" | python3 -c '
import json, sys
name = sys.argv[1]
try:
  data = json.load(sys.stdin)
  accounts = data.get("accounts") or []
  for entry in accounts:
    base = entry.get("base_account") or entry.get("baseAccount") or {}
    if entry.get("name") == name:
      print(base.get("address", ""))
      raise SystemExit(0)
  print("")
except Exception:
  print("")
' "$module_name"
}

get_balance() {
  local addr="$1"
  local denom="$2"
  timeout 10s curl -sS "$LCD_BASE/cosmos/bank/v1beta1/balances/$addr" | python3 -c '
import json, sys
addr = sys.argv[1]
denom = sys.argv[2]
try:
  data = json.load(sys.stdin)
  balances = data.get("balances") or []
  for entry in balances:
    if entry.get("denom") == denom:
      print(entry.get("amount", "0"))
      raise SystemExit(0)
  print("0")
except Exception:
  print("0")
' "$addr" "$denom"
}

assert_eq() {
  local label="$1"
  local expected="$2"
  local actual="$3"
  LABEL="$label" EXPECTED="$expected" ACTUAL="$actual" python3 - <<'PY'
import os
label = os.environ["LABEL"]
expected = int(os.environ["EXPECTED"])
actual = int(os.environ["ACTUAL"])
if expected != actual:
  raise SystemExit(f"ERROR: {label} expected {expected} got {actual}")
PY
}

require_cmd curl
require_cmd python3

echo "==> Starting local stack..."
"$STACK_SCRIPT" start

wait_for_http "LCD" "$LCD_BASE/cosmos/base/tendermint/v1beta1/node_info" 200 60 2
wait_for_http "Gateway" "$GATEWAY_BASE/health" 200 60 2

FAUCET_ADDR="$($NILCHAIND_BIN keys show faucet -a --home "$CHAIN_HOME" --keyring-backend test 2>/dev/null || true)"
if [ -z "$FAUCET_ADDR" ]; then
  echo "ERROR: failed to resolve faucet address" >&2
  exit 1
fi

echo "==> Using deal owner: $FAUCET_ADDR"

PARAMS_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/params")"
STORAGE_PRICE="$(echo "$PARAMS_JSON" | json_get "params.storage_price")"
BASE_FEE_AMOUNT="$(echo "$PARAMS_JSON" | json_get "params.base_retrieval_fee.amount")"
BASE_FEE_DENOM="$(echo "$PARAMS_JSON" | json_get "params.base_retrieval_fee.denom")"
RETRIEVAL_PRICE_PER_BLOB="$(echo "$PARAMS_JSON" | json_get "params.retrieval_price_per_blob.amount")"
RETRIEVAL_BURN_BPS="$(echo "$PARAMS_JSON" | json_get "params.retrieval_burn_bps")"

if [ -z "$BASE_FEE_DENOM" ]; then
  echo "ERROR: failed to read base_retrieval_fee.denom" >&2
  exit 1
fi

MODULE_ADDR="$(get_module_account nilchain)"
if [ -z "$MODULE_ADDR" ]; then
  echo "ERROR: failed to resolve nilchain module account" >&2
  exit 1
fi

BEFORE_MAX_DEAL_ID="$(get_max_deal_id)"

echo "==> Creating deal..."
CREATE_TX_JSON="$("$NILCHAIND_BIN" tx nilchain create-deal "$DURATION_BLOCKS" "$INITIAL_ESCROW" "$MAX_MONTHLY_SPEND" \
  --service-hint General \
  --from faucet \
  --chain-id "$CHAIN_ID" \
  --node tcp://127.0.0.1:26657 \
  --home "$CHAIN_HOME" \
  --keyring-backend test \
  --yes \
  --gas auto \
  --gas-adjustment 1.6 \
  --gas-prices 0.001aatom \
  --broadcast-mode "$BROADCAST_MODE" \
  --output json)"

CREATE_TX_HASH="$(echo "$CREATE_TX_JSON" | python3 -c '
import json, sys
try:
  data = json.load(sys.stdin)
except Exception:
  print("")
  raise SystemExit(0)
print(data.get("txhash") or data.get("tx_hash") or "")
')"

CREATE_TX_RESP="$(wait_for_tx "$CREATE_TX_HASH")"

DEAL_ID="$(echo "$CREATE_TX_RESP" | python3 -c '
import json, sys
try:
  data = json.load(sys.stdin)
  resp = data.get("tx_response") or {}
  events = resp.get("events") or []
  for ev in events:
    for attr in ev.get("attributes") or []:
      key = attr.get("key") or ""
      if key in ("deal_id", "dealId"):
        print(attr.get("value") or "")
        raise SystemExit(0)
  logs = resp.get("logs") or []
  for log in logs:
    for ev in log.get("events") or []:
      for attr in ev.get("attributes") or []:
        key = attr.get("key") or ""
        if key in ("deal_id", "dealId"):
          print(attr.get("value") or "")
          raise SystemExit(0)
  print("")
except Exception:
  print("")
')"

if [ -z "$DEAL_ID" ]; then
  for _ in $(seq 1 20); do
    latest="$(get_max_deal_id)"
    if [ "$latest" -ge "$BEFORE_MAX_DEAL_ID" ]; then
      DEAL_ID="$latest"
      break
    fi
    sleep 1
  done
fi
if [ -z "$DEAL_ID" ]; then
  echo "ERROR: failed to resolve newly created deal id" >&2
  echo "$CREATE_TX_JSON" >&2
  exit 1
fi

echo "    Deal ID: $DEAL_ID"

DEAL_JSON_BEFORE="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID")"
ESCROW_BEFORE="$(echo "$DEAL_JSON_BEFORE" | json_get "deal.escrow_balance")"
START_BLOCK="$(echo "$DEAL_JSON_BEFORE" | json_get "deal.start_block")"
END_BLOCK="$(echo "$DEAL_JSON_BEFORE" | json_get "deal.end_block")"

if [ -z "$ESCROW_BEFORE" ] || [ -z "$START_BLOCK" ] || [ -z "$END_BLOCK" ]; then
  echo "ERROR: failed to read deal fields" >&2
  echo "$DEAL_JSON_BEFORE" >&2
  exit 1
fi

MODULE_BAL_BEFORE_UPDATE="$(get_balance "$MODULE_ADDR" "$BASE_FEE_DENOM")"

if [ ! -f "$UPLOAD_FILE" ]; then
  echo "ERROR: UPLOAD_FILE does not exist: $UPLOAD_FILE" >&2
  exit 1
fi

UPLOAD_BYTES="$(python3 - "$UPLOAD_FILE" <<'PY'
import os
import sys
print(os.path.getsize(sys.argv[1]))
PY
)"

UPLOAD_RESP=$(timeout 600s curl --verbose -X POST -F "file=@$UPLOAD_FILE" \
  -F "owner=$FAUCET_ADDR" \
  "$GATEWAY_BASE/gateway/upload?deal_id=$DEAL_ID")

echo "    Response: $UPLOAD_RESP"

MANIFEST_ROOT="$(echo "$UPLOAD_RESP" | python3 -c "import sys, json; j=json.load(sys.stdin); print(j.get('manifest_root') or j.get('cid') or '')")"
SIZE_BYTES="$(echo "$UPLOAD_RESP" | python3 -c "import sys, json; j=json.load(sys.stdin); print(j.get('size_bytes') or j.get('sizeBytes') or '')")"
TOTAL_MDUS="$(echo "$UPLOAD_RESP" | python3 -c "import sys, json; j=json.load(sys.stdin); print(j.get('total_mdus') or j.get('totalMdus') or j.get('allocated_length') or '')")"
WITNESS_MDUS="$(echo "$UPLOAD_RESP" | python3 -c "import sys, json; j=json.load(sys.stdin); print(j.get('witness_mdus') or j.get('witnessMdus') or '')")"

if [ -z "$MANIFEST_ROOT" ] || [ -z "$SIZE_BYTES" ] || [ -z "$TOTAL_MDUS" ] || [ -z "$WITNESS_MDUS" ]; then
  echo "ERROR: failed to parse gateway upload response" >&2
  exit 1
fi

UPDATE_TX_JSON="$("$NILCHAIND_BIN" tx nilchain update-deal-content \
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
  --broadcast-mode "$BROADCAST_MODE" \
  --output json)"
UPDATE_TX_HASH="$(echo "$UPDATE_TX_JSON" | python3 -c 'import json, sys
try:
  data=json.load(sys.stdin)
  print(data.get("txhash") or data.get("tx_hash") or "")
except Exception:
  print("")
')"
wait_for_tx "$UPDATE_TX_HASH" >/dev/null

CHAIN_ROOT_HEX=""
for _ in $(seq 1 20); do
  DEAL_JSON_UPDATED="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID" || echo '{}')"
  CHAIN_ROOT_HEX="$(echo "$DEAL_JSON_UPDATED" | python3 -c '
import base64, json, sys
try:
  deal = json.load(sys.stdin).get("deal") or {}
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
except Exception:
  print("")
')"
  if [ "$CHAIN_ROOT_HEX" = "$MANIFEST_ROOT" ]; then
    break
  fi
  sleep 1
done
if [ "$CHAIN_ROOT_HEX" != "$MANIFEST_ROOT" ]; then
  echo "ERROR: deal manifest_root not updated on-chain" >&2
  echo "    expected=$MANIFEST_ROOT got=${CHAIN_ROOT_HEX:-}" >&2
  exit 1
fi

DEAL_JSON_AFTER="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID")"
ESCROW_AFTER_UPDATE="$(echo "$DEAL_JSON_AFTER" | json_get "deal.escrow_balance")"

STORAGE_COST="$(STORAGE_PRICE="$STORAGE_PRICE" DELTA_BYTES="$SIZE_BYTES" START_BLOCK="$START_BLOCK" END_BLOCK="$END_BLOCK" python3 - <<'PY'
import os
from decimal import Decimal, getcontext, ROUND_CEILING
getcontext().prec = 80
price = Decimal(os.environ.get("STORAGE_PRICE", "0"))
delta = int(os.environ.get("DELTA_BYTES", "0"))
start = int(os.environ.get("START_BLOCK", "0"))
end = int(os.environ.get("END_BLOCK", "0"))
duration = max(0, end - start)
if price <= 0 or delta <= 0 or duration <= 0:
  print(0)
  raise SystemExit(0)
cost = (price * Decimal(delta) * Decimal(duration)).to_integral_value(rounding=ROUND_CEILING)
print(int(cost))
PY
)"

ESCROW_EXPECTED="$(python3 - <<PY
before = int("$ESCROW_BEFORE")
expected = before + int("$STORAGE_COST")
print(expected)
PY
)"
assert_eq "escrow after update" "$ESCROW_EXPECTED" "$ESCROW_AFTER_UPDATE"

MODULE_BAL_AFTER_UPDATE="$(get_balance "$MODULE_ADDR" "$BASE_FEE_DENOM")"
MODULE_EXPECTED_AFTER_UPDATE="$(python3 - <<PY
before = int("$MODULE_BAL_BEFORE_UPDATE")
expected = before + int("$STORAGE_COST")
print(expected)
PY
)"
assert_eq "module balance after update" "$MODULE_EXPECTED_AFTER_UPDATE" "$MODULE_BAL_AFTER_UPDATE"

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

FEES_JSON="$(BASE_FEE="$BASE_FEE_AMOUNT" PRICE_PER_BLOB="$RETRIEVAL_PRICE_PER_BLOB" BLOB_COUNT="$PLAN_BLOB_COUNT" BURN_BPS="$RETRIEVAL_BURN_BPS" python3 - <<'PY'
import json, os
base = int(os.environ.get("BASE_FEE", "0"))
price = int(os.environ.get("PRICE_PER_BLOB", "0"))
count = int(os.environ.get("BLOB_COUNT", "0"))
burn_bps = int(os.environ.get("BURN_BPS", "0"))
variable = price * count
burn = 0
if variable > 0 and burn_bps > 0:
  burn = (variable * burn_bps + 9999) // 10000
  if burn > variable:
    burn = variable
payout = variable - burn
print(json.dumps({
  "variable": variable,
  "total": base + variable,
  "burn": burn,
  "payout": payout,
}))
PY
)"
VARIABLE_FEE="$(echo "$FEES_JSON" | json_get "variable")"
TOTAL_FEE="$(echo "$FEES_JSON" | json_get "total")"
BURN_CUT="$(echo "$FEES_JSON" | json_get "burn")"
PAYOUT="$(echo "$FEES_JSON" | json_get "payout")"

DEAL_ESCROW_PRE_OPEN="$ESCROW_AFTER_UPDATE"
MODULE_BAL_PRE_OPEN="$MODULE_BAL_AFTER_UPDATE"
PROVIDER_BAL_PRE_OPEN="$(get_balance "$PLAN_PROVIDER" "$BASE_FEE_DENOM")"

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

echo "==> Opening retrieval session..."
OPEN_TX_JSON="$("$NILCHAIND_BIN" tx nilchain open-retrieval-session \
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
  --broadcast-mode "$BROADCAST_MODE" \
  --output json)"
OPEN_TX_HASH="$(echo "$OPEN_TX_JSON" | python3 -c 'import json, sys
try:
  data=json.load(sys.stdin)
  print(data.get("txhash") or data.get("tx_hash") or "")
except Exception:
  print("")
')"
wait_for_tx "$OPEN_TX_HASH" >/dev/null

SESSION_HEX=""
for _ in $(seq 1 30); do
  SESSIONS_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/retrieval-sessions/by-owner/$FAUCET_ADDR" || echo "{}")"
  SESSION_HEX="$(echo "$SESSIONS_JSON" | DEAL_ID="$DEAL_ID" NONCE="$NONCE" python3 -c '
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
  try:
    bz = base64.b64decode(raw)
  except Exception:
    try:
      bz = base64.urlsafe_b64decode(raw + "==")
    except Exception:
      print("")
      raise SystemExit(0)
  print("0x" + bz.hex())
except Exception:
  print("")
')"
  if [ -n "$SESSION_HEX" ]; then
    break
  fi
  sleep 1
done
if [ -z "$SESSION_HEX" ]; then
  echo "ERROR: failed to resolve session id" >&2
  echo "$SESSIONS_JSON" >&2
  exit 1
fi

DEAL_JSON_OPEN="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID")"
ESCROW_AFTER_OPEN="$(echo "$DEAL_JSON_OPEN" | json_get "deal.escrow_balance")"

EXPECTED_ESCROW_AFTER_OPEN="$(python3 - <<PY
before = int("$DEAL_ESCROW_PRE_OPEN")
expected = before - int("$TOTAL_FEE")
print(expected)
PY
)"
assert_eq "escrow after open" "$EXPECTED_ESCROW_AFTER_OPEN" "$ESCROW_AFTER_OPEN"

MODULE_BAL_AFTER_OPEN="$(get_balance "$MODULE_ADDR" "$BASE_FEE_DENOM")"
EXPECTED_MODULE_AFTER_OPEN="$(python3 - <<PY
before = int("$MODULE_BAL_PRE_OPEN")
expected = before - int("$BASE_FEE_AMOUNT")
print(expected)
PY
)"
assert_eq "module balance after open" "$EXPECTED_MODULE_AFTER_OPEN" "$MODULE_BAL_AFTER_OPEN"

start_end="$((RAW_BLOB_PAYLOAD_BYTES - 1))"
OUT_FILE="$(mktemp)"
HDR_FILE="$(mktemp)"
FETCH_EXIT=0
HTTP_CODE="$(timeout 120s curl -sS -D "$HDR_FILE" -o "$OUT_FILE" \
  -H "X-Nil-Session-Id: $SESSION_HEX" \
  -H "Range: bytes=0-${start_end}" \
  "$GATEWAY_BASE/gateway/fetch/$MANIFEST_ROOT?deal_id=$DEAL_ID&owner=$FAUCET_ADDR&file_path=$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))' "$FILE_PATH")" \
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
  exit 1
fi

rm -f "$OUT_FILE" "$HDR_FILE"

PROOF_SUBMIT_RESP="$(timeout 120s curl -sS -X POST "$GATEWAY_BASE/gateway/session-proof" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\":\"$SESSION_HEX\",\"provider\":\"$PLAN_PROVIDER\"}")"
PROOF_STATUS="$(echo "$PROOF_SUBMIT_RESP" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("status",""))' 2>/dev/null || true)"
if [ "$PROOF_STATUS" != "success" ]; then
  echo "ERROR: session proof submission failed" >&2
  echo "$PROOF_SUBMIT_RESP" >&2
  exit 1
fi

CONFIRM_TX_JSON="$("$NILCHAIND_BIN" tx nilchain confirm-retrieval-session "$SESSION_HEX" \
  --from faucet \
  --chain-id "$CHAIN_ID" \
  --node tcp://127.0.0.1:26657 \
  --home "$CHAIN_HOME" \
  --keyring-backend test \
  --yes \
  --gas auto \
  --gas-adjustment 1.6 \
  --gas-prices 0.001aatom \
  --broadcast-mode "$BROADCAST_MODE" \
  --output json)"
CONFIRM_TX_HASH="$(echo "$CONFIRM_TX_JSON" | python3 -c 'import json, sys
try:
  data=json.load(sys.stdin)
  print(data.get("txhash") or data.get("tx_hash") or "")
except Exception:
  print("")
')"
wait_for_tx "$CONFIRM_TX_HASH" >/dev/null

MODULE_BAL_AFTER_CONFIRM="$(get_balance "$MODULE_ADDR" "$BASE_FEE_DENOM")"
PROVIDER_BAL_AFTER_CONFIRM="$(get_balance "$PLAN_PROVIDER" "$BASE_FEE_DENOM")"

EXPECTED_MODULE_AFTER_CONFIRM="$(python3 - <<PY
before = int("$MODULE_BAL_AFTER_OPEN")
expected = before - int("$VARIABLE_FEE")
print(expected)
PY
)"
assert_eq "module balance after confirm" "$EXPECTED_MODULE_AFTER_CONFIRM" "$MODULE_BAL_AFTER_CONFIRM"

EXPECTED_PROVIDER_BAL_AFTER="$(python3 - <<PY
before = int("$PROVIDER_BAL_PRE_OPEN")
expected = before + int("$PAYOUT")
print(expected)
PY
)"
assert_eq "provider balance after confirm" "$EXPECTED_PROVIDER_BAL_AFTER" "$PROVIDER_BAL_AFTER_CONFIRM"

# --- Cancel path ---
HEIGHT2="$(rpc_height)"
if [ "$HEIGHT2" -le 0 ]; then
  echo "ERROR: failed to resolve chain height for cancel session" >&2
  exit 1
fi
EXPIRES_AT_2="$((HEIGHT2 + 5))"
NONCE_2="$(python3 - <<'PY'
import time
print(time.time_ns())
PY
)"

OPEN2_TX_JSON="$("$NILCHAIND_BIN" tx nilchain open-retrieval-session \
  --deal-id "$DEAL_ID" \
  --provider "$PLAN_PROVIDER" \
  --manifest-root "$MANIFEST_ROOT" \
  --start-mdu-index "$PLAN_START_MDU" \
  --start-blob-index "$PLAN_START_BLOB" \
  --blob-count "$PLAN_BLOB_COUNT" \
  --nonce "$NONCE_2" \
  --expires-at "$EXPIRES_AT_2" \
  --from faucet \
  --chain-id "$CHAIN_ID" \
  --node tcp://127.0.0.1:26657 \
  --home "$CHAIN_HOME" \
  --keyring-backend test \
  --yes \
  --gas auto \
  --gas-adjustment 1.6 \
  --gas-prices 0.001aatom \
  --broadcast-mode "$BROADCAST_MODE" \
  --output json)"
OPEN2_TX_HASH="$(echo "$OPEN2_TX_JSON" | python3 -c 'import json, sys
try:
  data=json.load(sys.stdin)
  print(data.get("txhash") or data.get("tx_hash") or "")
except Exception:
  print("")
')"
wait_for_tx "$OPEN2_TX_HASH" >/dev/null

SESSION_HEX_2=""
for _ in $(seq 1 30); do
  SESSIONS_JSON="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/retrieval-sessions/by-owner/$FAUCET_ADDR" || echo "{}")"
  SESSION_HEX_2="$(echo "$SESSIONS_JSON" | DEAL_ID="$DEAL_ID" NONCE="$NONCE_2" python3 -c '
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
  try:
    bz = base64.b64decode(raw)
  except Exception:
    try:
      bz = base64.urlsafe_b64decode(raw + "==")
    except Exception:
      print("")
      raise SystemExit(0)
  print("0x" + bz.hex())
except Exception:
  print("")
')"
  if [ -n "$SESSION_HEX_2" ]; then
    break
  fi
  sleep 1
done
if [ -z "$SESSION_HEX_2" ]; then
  echo "ERROR: failed to resolve second session id" >&2
  echo "$SESSIONS_JSON" >&2
  exit 1
fi

DEAL_JSON_OPEN_2="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID")"
ESCROW_AFTER_OPEN_2="$(echo "$DEAL_JSON_OPEN_2" | json_get "deal.escrow_balance")"
MODULE_BAL_AFTER_OPEN_2="$(get_balance "$MODULE_ADDR" "$BASE_FEE_DENOM")"

EXPECTED_ESCROW_AFTER_OPEN_2="$(python3 - <<PY
before = int("$ESCROW_AFTER_OPEN")
expected = before - int("$TOTAL_FEE")
print(expected)
PY
)"
assert_eq "escrow after second open" "$EXPECTED_ESCROW_AFTER_OPEN_2" "$ESCROW_AFTER_OPEN_2"

EXPECTED_MODULE_AFTER_OPEN_2="$(python3 - <<PY
before = int("$MODULE_BAL_AFTER_CONFIRM")
expected = before - int("$BASE_FEE_AMOUNT")
print(expected)
PY
)"
assert_eq "module balance after second open" "$EXPECTED_MODULE_AFTER_OPEN_2" "$MODULE_BAL_AFTER_OPEN_2"

wait_for_height "$((EXPIRES_AT_2 + 1))" 120 1 || { echo "ERROR: timed out waiting for session expiry" >&2; exit 1; }

CANCEL_TX_JSON="$("$NILCHAIND_BIN" tx nilchain cancel-retrieval-session "$SESSION_HEX_2" \
  --from faucet \
  --chain-id "$CHAIN_ID" \
  --node tcp://127.0.0.1:26657 \
  --home "$CHAIN_HOME" \
  --keyring-backend test \
  --yes \
  --gas auto \
  --gas-adjustment 1.6 \
  --gas-prices 0.001aatom \
  --broadcast-mode "$BROADCAST_MODE" \
  --output json)"
CANCEL_TX_HASH="$(echo "$CANCEL_TX_JSON" | python3 -c 'import json, sys
try:
  data=json.load(sys.stdin)
  print(data.get("txhash") or data.get("tx_hash") or "")
except Exception:
  print("")
')"
wait_for_tx "$CANCEL_TX_HASH" >/dev/null

DEAL_JSON_CANCEL="$(timeout 10s curl -sS "$LCD_BASE/nilchain/nilchain/v1/deals/$DEAL_ID")"
ESCROW_AFTER_CANCEL="$(echo "$DEAL_JSON_CANCEL" | json_get "deal.escrow_balance")"
MODULE_BAL_AFTER_CANCEL="$(get_balance "$MODULE_ADDR" "$BASE_FEE_DENOM")"

EXPECTED_ESCROW_AFTER_CANCEL="$(python3 - <<PY
before = int("$ESCROW_AFTER_OPEN_2")
expected = before + int("$VARIABLE_FEE")
print(expected)
PY
)"
assert_eq "escrow after cancel" "$EXPECTED_ESCROW_AFTER_CANCEL" "$ESCROW_AFTER_CANCEL"

assert_eq "module balance after cancel" "$MODULE_BAL_AFTER_OPEN_2" "$MODULE_BAL_AFTER_CANCEL"

echo "==> Econ parity E2E test passed."
