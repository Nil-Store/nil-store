#!/bin/bash
set -e

# Build nil_core WASM (nodejs target)
echo "Building nil_core WASM..."
cd nil_core
wasm-pack build --target nodejs --out-dir pkg_node
cd ..

# Build nil_cli (native)
echo "Building nil_cli..."
cd nil_cli
cargo build --bin nil_cli
cd ..

CLI_BIN="./nil_cli/target/debug/nil_cli"
FIXTURE="test_8mb.bin"
CLI_OUTPUT="cli_output.json"

# Generate 8MB deterministic fixture
echo "Generating 8MB deterministic fixture..."
# Using python to generate a deterministic pattern (0, 1, 2... 255 repeating)
python3 -c "import sys; sys.stdout.buffer.write(bytearray([i % 256 for i in range(8 * 1024 * 1024)]))" > $FIXTURE

# Run nil_cli shard
echo "Running nil_cli shard..."
$CLI_BIN shard $FIXTURE --out $CLI_OUTPUT --seeds "1,2,3"

# Run Parity Test Script
echo "Running Parity Test Script..."
node tests/wasm_parity.js

# Cleanup
rm $FIXTURE $CLI_OUTPUT

echo "Done."
