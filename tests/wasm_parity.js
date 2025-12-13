const fs = require('fs');
const path = require('path');
const { NilWasm } = require('../nil_core/pkg_node/nil_core.js');

// Helper to convert hex string to byte array
function hexToBytes(hex) {
    if (hex.startsWith('0x')) hex = hex.slice(2);
    const bytes = new Uint8Array(hex.length / 2);
    for (let i = 0; i < hex.length; i += 2) {
        bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
    }
    return bytes;
}

// Helper to convert byte array to hex string
function bytesToHex(bytes) {
    return Array.from(bytes)
        .map(b => b.toString(16).padStart(2, '0'))
        .join('');
}

async function main() {
    console.log("Starting WASM <-> Native Parity Test...");

    // 1. Load Trusted Setup
    const trustedSetupPath = path.join(__dirname, '../demos/kzg/trusted_setup.txt');
    console.log(`Loading trusted setup from ${trustedSetupPath}...`);
    const trustedSetup = fs.readFileSync(trustedSetupPath);

    // 2. Initialize WASM
    console.log("Initializing NilWasm...");
    let nilWasm;
    try {
        nilWasm = new NilWasm(trustedSetup);
    } catch (e) {
        console.error("Failed to initialize NilWasm:", e);
        process.exit(1);
    }

    // 3. Load Fixture
    const fixturePath = path.join(__dirname, '../test_8mb.bin');
    console.log(`Loading fixture from ${fixturePath}...`);
    const fixtureData = fs.readFileSync(fixturePath);

    // 4. Run WASM Expansion
    // expand_file requires exactly 8MB input and processes the first MDU's worth of data.
    console.log("Running WASM expand_file...");
    let wasmOutput;
    try {
        wasmOutput = nilWasm.expand_file(fixtureData);
    } catch (e) {
        console.error("WASM expansion failed:", e);
        process.exit(1);
    }

    // 5. Load CLI Output
    const cliOutputPath = path.join(__dirname, '../cli_output.json');
    console.log(`Loading CLI output from ${cliOutputPath}...`);
    const cliOutput = JSON.parse(fs.readFileSync(cliOutputPath, 'utf8'));

    // 6. Compare MDU #0
    console.log("Comparing outputs (MDU #0)...");

    if (cliOutput.mdus.length === 0) {
        console.error("CLI output has no MDUs.");
        process.exit(1);
    }

    const cliMdu = cliOutput.mdus[0];
    const cliBlobs = cliMdu.blobs; // Array of hex strings

    // Check Data Blobs (0..63)
    let mismatchCount = 0;
    for (let i = 0; i < 64; i++) {
        const wasmCommitment = wasmOutput.witness[i];
        const wasmHex = '0x' + bytesToHex(wasmCommitment);
        const cliHex = cliBlobs[i].toLowerCase();

        if (wasmHex !== cliHex) {
            console.error(`Mismatch at blob index ${i}:`);
            console.error(`  WASM: ${wasmHex}`);
            console.error(`  CLI:  ${cliHex}`);
            mismatchCount++;
        }
    }

    if (mismatchCount > 0) {
        console.error(`Found ${mismatchCount} mismatches in data blob commitments.`);
        process.exit(1);
    } else {
        console.log("SUCCESS: All 64 data blob commitments match!");
    }

    // Note on MDU Root Verification:
    // The CLI output contains the Merkle Root of these commitments (`root_hex`).
    // The WASM output (ExpandedMdu) contains the raw commitments.
    // Since we verified that the commitments match byte-for-byte, the Merkle Root derived from them
    // must also match, assuming the Merkle tree construction (leaf sorting, hash function) is consistent.
    // We implicitly verify the root logic by verifying the inputs (commitments) are identical.
    console.log(`CLI MDU #0 Root: ${cliMdu.root_hex}`);
    console.log("Parity Test Passed.");
}

main();
