#!/bin/bash
set -euo pipefail

# E2E Regression Test: Multi-SP Retrieval Proofs
# Tests that a Gateway can submit a retrieval proof for a deal owned by a DIFFERENT
# account (e.g. Provider A owns deal, Provider B hosts data).
#
# Requires: run_devnet_alpha_multi_sp.sh stack to be running.

GATEWAY_ROUTER="http://localhost:8080"
NILCHAIND="nilchain/nilchaind"
CHAIN_HOME="_artifacts/nilchain_data_devnet_alpha"
TMP_DIR="_artifacts/e2e_multi_sp_tmp"
mkdir -p "$TMP_DIR"

banner() { printf '\n>>> %s\n' "$*"; }
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

wait_for_tx_json() {
  local hash="$1"
  local attempts="${2:-40}"
  local delay="${3:-1}"
  local i out
  for i in $(seq 1 "$attempts"); do
    out=$($NILCHAIND query tx "$hash" --home "$CHAIN_HOME" --output json 2>/dev/null || true)
    if [ -n "$out" ] && echo "$out" | jq -e '.txhash // .tx_response.txhash // empty' >/dev/null 2>&1; then
      echo "$out"
      return 0
    fi
    sleep "$delay"
  done
  return 1
}

parse_create_deal_id() {
  python3 -c '
import base64, json, sys
def maybe_b64(s: str) -> str:
  if not s:
    return ""
  try:
    pad = "=" * ((4 - (len(s) % 4)) % 4)
    return base64.b64decode((s + pad).encode("utf-8"), validate=False).decode("utf-8", errors="ignore")
  except Exception:
    return ""
try:
  tx = json.load(sys.stdin)
except Exception:
  print("")
  raise SystemExit(0)
events = []
for item in (tx.get("logs") or []):
  events.extend(item.get("events") or [])
events.extend(tx.get("events") or [])
for ev in events:
  ev_type = ev.get("type") or ""
  if ev_type not in ("create_deal", "nilchain.nilchain.v1.EventCreateDeal"):
    continue
  for a in (ev.get("attributes") or []):
    key = a.get("key") or ""
    val = a.get("value") or ""
    dkey = maybe_b64(key)
    dval = maybe_b64(val)
    if key in ("deal_id", "id"):
      print(val)
      raise SystemExit(0)
    if dkey in ("deal_id", "id"):
      print(dval or val)
      raise SystemExit(0)
print("")
'
}

current_epoch() {
  local epoch_len height

  epoch_len=$($NILCHAIND query nilchain params --home "$CHAIN_HOME" --output json | jq -r '.params.epoch_len_blocks // "0"')
  if [ -z "$epoch_len" ] || [ "$epoch_len" = "null" ]; then
    epoch_len="0"
  fi

  # Prefer CometBFT RPC directly (faster than nilchaind status).
  height=$(curl -s "http://127.0.0.1:26657/status" | jq -r '.result.sync_info.latest_block_height // "1"')
  if [ -z "$height" ] || [ "$height" = "null" ]; then
    height="1"
  fi

  if [ "$epoch_len" -le 0 ]; then
    echo "1"
    return 0
  fi

  # epoch_id is 1-indexed: epoch=(height-1)/epoch_len + 1
  echo $(( (height - 1) / epoch_len + 1 ))
}

# 1. Setup
banner "Generating Test Data"
dd if=/dev/urandom of="$TMP_DIR/payload.bin" bs=1024 count=1024 2>/dev/null # 1MB

# 2. Identify Test Accounts (Provider1 = Owner)
banner "Resolving Accounts"
OWNER_ADDR=$($NILCHAIND keys show provider1 -a --home "$CHAIN_HOME" --keyring-backend test)
echo "Owner (Provider1): $OWNER_ADDR"

# 3. Create Deal
banner "Creating Deal"
# Use a 3-slot Mode 2 stripe for the multi-SP devnet (K=2,M=1).
# The gateway /gateway/prove-retrieval endpoint reconstructs the full MDU from per-slot shards on the router
# and submits the proof "as" the assigned provider.
CREATE_OUT=$($NILCHAIND tx nilchain create-deal 1000 1000000 1000000 --service-hint "General:rs=2+1" --chain-id 31337 --from provider1 --yes --keyring-backend test --home "$CHAIN_HOME" --gas-prices 0.001aatom --output json)
TX_HASH=$(echo "$CREATE_OUT" | jq -r '.txhash')
echo "Create Deal Tx: $TX_HASH"

banner "Waiting for Deal on Chain..."
TX_QUERY="$(wait_for_tx_json "$TX_HASH" 40 1 || true)"
if [ -z "$TX_QUERY" ]; then
  echo "Create deal failed: tx not found for hash $TX_HASH"
  exit 1
fi
CREATE_CODE="$(echo "$TX_QUERY" | jq -r '.code // .tx_response.code // 0')"
if [ "$CREATE_CODE" != "0" ]; then
  CREATE_RAW_LOG="$(echo "$TX_QUERY" | jq -r '.raw_log // .tx_response.raw_log // empty')"
  echo "Create deal failed: tx code=$CREATE_CODE raw_log=${CREATE_RAW_LOG:-unknown}"
  exit 1
fi
DEAL_ID="$(echo "$TX_QUERY" | parse_create_deal_id)"
if [ -z "$DEAL_ID" ]; then
  for _ in {1..10}; do
    DEAL_LIST="$($NILCHAIND query nilchain list-deals --output json)"
    DEAL_ID="$(echo "$DEAL_LIST" | jq -r '.deals // [] | .[-1].id // empty')"
    if [ -n "$DEAL_ID" ] && [ "$DEAL_ID" != "null" ]; then
      break
    fi
    sleep 1
  done
fi
echo "Deal ID: $DEAL_ID"
if [ -z "$DEAL_ID" ] || [ "$DEAL_ID" == "null" ]; then
    echo "Create deal failed: deal_id not found"
    exit 1
fi

# 4. Upload Content (via Router)
banner "Uploading Content"
UPLOAD_RESP=$(curl -s -X POST -F "file=@$TMP_DIR/payload.bin;filename=payload.bin" "$GATEWAY_ROUTER/gateway/upload?deal_id=$DEAL_ID")
CID=$(echo "$UPLOAD_RESP" | jq -r '.cid')
SIZE=$(echo "$UPLOAD_RESP" | jq -r '.size_bytes')
TOTAL_MDUS=$(echo "$UPLOAD_RESP" | jq -r '.total_mdus')
WITNESS_MDUS=$(echo "$UPLOAD_RESP" | jq -r '.witness_mdus')

if [ "$CID" == "null" ]; then
    echo "Upload failed: $UPLOAD_RESP"
    exit 1
fi
echo "CID: $CID"

# 5. Commit Content
banner "Committing Content"
COMMIT_OUT=$($NILCHAIND tx nilchain update-deal-content --deal-id "$DEAL_ID" --cid "$CID" --size "$SIZE" --total-mdus "$TOTAL_MDUS" --witness-mdus "$WITNESS_MDUS" --chain-id 31337 --from provider1 --yes --keyring-backend test --home "$CHAIN_HOME" --gas-prices 0.001aatom --output json)
echo "Commit Tx: $(echo "$COMMIT_OUT" | jq -r '.txhash')"
sleep 6

# 6. Resolve Assigned Provider
banner "Resolving Assigned Provider"
DEAL_INFO=$($NILCHAIND query nilchain get-deal --id "$DEAL_ID" --output json)
ASSIGNED_ADDR=$(echo "$DEAL_INFO" | jq -r --arg owner "$OWNER_ADDR" '.deal.providers[] | select(. != $owner) | . ' | head -n1)
if [ -z "$ASSIGNED_ADDR" ] || [ "$ASSIGNED_ADDR" == "null" ]; then
  ASSIGNED_ADDR=$(echo "$DEAL_INFO" | jq -r '.deal.providers[0]')
fi
echo "Assigned Provider: $ASSIGNED_ADDR"

if [ "$ASSIGNED_ADDR" == "$OWNER_ADDR" ]; then
    echo "WARNING: Assigned provider IS the owner. This test works best when they differ."
    echo "Continuing anyway, as signature mismatch could still occur if code is wrong."
else
    echo "Confirmed: Assigned provider != Owner. Testing cross-account signing."
fi

PROVIDER_INFO=$($NILCHAIND query nilchain get-provider --address "$ASSIGNED_ADDR" --output json)
ENDPOINT=$(echo "$PROVIDER_INFO" | jq -r '.provider.endpoints[0]')
# Extract port from /ip4/127.0.0.1/tcp/PORT/http
PORT=$(echo "$ENDPOINT" | awk -F/ '{print $5}')
echo "Provider Port: $PORT"

# 7. Prove Retrieval (The Regression Test)
banner "Proving Retrieval (via Router, submitting as assigned provider)"
EPOCH_ID="$(current_epoch)"
echo "Current Epoch: $EPOCH_ID"
# This call triggers 'submitRetrievalProofNew' on the router gateway, which reconstructs the Mode 2 MDU and
# submits the proof using the assigned provider key (shared keyring in local devnet).
PROVE_RESP=$(curl -s -X POST -H "Content-Type: application/json" -d '{
    "deal_id": '$DEAL_ID',
    "manifest_root": "'$CID'",
    "file_path": "payload.bin",
    "owner": "'$OWNER_ADDR'",
    "provider": "'$ASSIGNED_ADDR'",
    "epoch_id": '$EPOCH_ID'
}' "$GATEWAY_ROUTER/gateway/prove-retrieval")

echo "Prove Response: $PROVE_RESP"

ERR=$(echo "$PROVE_RESP" | jq -r '.error // empty')
if [ -n "$ERR" ]; then
    echo "❌ TEST FAILED: $ERR"
    exit 1
fi

TX_HASH_PROOF=$(echo "$PROVE_RESP" | jq -r '.tx_hash')
if [ "$TX_HASH_PROOF" == "null" ]; then
    echo "❌ TEST FAILED: No tx_hash in response"
    exit 1
fi

echo "✅ TEST PASSED: Retrieval proof submitted successfully."
