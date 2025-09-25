"""
Nilcoin KZG (ckzg) demo — educational only.

This is a rewrite of t.py using the real ckzg (c-kzg-4844) bindings.
It maps a file deterministically into EIP‑4844 blobs (4096 field elements)
and produces KZG commitments and example proofs using ckzg.

Usage:
  python3 demos/kzg/kzg_ckzg.py --file spec.md --trusted-setup ./trust_setup.txt \
                    --out nilcoin_kzg_ckzg_output.json

Note: file moved to demos/kzg as kzg_ckzg.py.

Notes:
  - This is a demo; not for production.
  - The mapping from bytes → field elements uses SHA‑256 modular reduction
    per 1 KiB block to ensure canonical Fr elements.
  - Requires the ckzg module and a trusted setup file compatible with ckzg.
"""

import argparse
import hashlib
import json
import math
import os
import random
import time
from typing import List, Tuple

try:
    import ckzg  # type: ignore
except Exception as e:  # pragma: no cover
    ckzg = None
    _ckzg_import_err = e
else:
    _ckzg_import_err = None


# BLS12-381 scalar field modulus (Fr), per EIP-4844 (little-endian 32-byte encoding)
FR_MODULUS_HEX = "73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001"
FR_MODULUS = int(FR_MODULUS_HEX, 16)

# EIP-4844 constants
FE_BYTES = 32
FE_PER_BLOB = 4096
BLOB_BYTES = FE_PER_BLOB * FE_BYTES  # 131072 bytes

# Our plaintext mapping constants (kept consistent with t.py where possible)
SRC_CHUNK = 1024  # 1 KiB source block hashed → Fr


def fr_from_hash(block: bytes) -> int:
    h = hashlib.sha256(block).digest()
    return int.from_bytes(h, "big") % FR_MODULUS


def fr_to_le_bytes(x: int) -> bytes:
    if not (0 <= x < FR_MODULUS):
        raise ValueError("field element out of range")
    return x.to_bytes(FE_BYTES, "little")


def file_to_fr_elements(path: str) -> List[int]:
    with open(path, "rb") as f:
        data = f.read()
    # Split into 1 KiB source blocks; pad final to 1 KiB for determinism
    blocks = [data[i : i + SRC_CHUNK] for i in range(0, len(data), SRC_CHUNK)]
    if len(blocks) == 0:
        blocks = [b""]
    if len(blocks[-1]) < SRC_CHUNK:
        blocks[-1] = blocks[-1] + b"\x00" * (SRC_CHUNK - len(blocks[-1]))
    # Hash each 1 KiB block to Fr
    return [fr_from_hash(b) for b in blocks]


def fr_elements_to_blobs(ys: List[int]) -> List[bytes]:
    # Group into blobs of 4096 field elements; pad last blob with zeros
    out: List[bytes] = []
    for i in range(0, len(ys), FE_PER_BLOB):
        group = ys[i : i + FE_PER_BLOB]
        if len(group) < FE_PER_BLOB:
            group = group + [0] * (FE_PER_BLOB - len(group))
        blob = b"".join(fr_to_le_bytes(x) for x in group)
        assert len(blob) == BLOB_BYTES
        out.append(blob)
    if not out:  # empty input → one zero blob
        out.append(b"\x00" * BLOB_BYTES)
    return out


def load_ts(ts_path: str):
    if ckzg is None:  # pragma: no cover
        raise RuntimeError(
            f"ckzg import failed: {_ckzg_import_err}. Install ckzg and retry."
        )
    return ckzg.load_trusted_setup(ts_path, 0)


def blob_commitment(blob: bytes, ts) -> bytes:
    return ckzg.blob_to_kzg_commitment(blob, ts)


def blob_proof(blob: bytes, commitment: bytes, ts) -> Tuple[bytes, bool]:
    proof = ckzg.compute_blob_kzg_proof(blob, commitment, ts)
    valid = ckzg.verify_blob_kzg_proof(blob, commitment, proof, ts)
    return proof, valid


def point_eval_proof(blob: bytes, commitment: bytes, z: int, ts) -> Tuple[bytes, bytes, bool]:
    z_bytes = fr_to_le_bytes(z)
    proof, y = ckzg.compute_kzg_proof(blob, z_bytes, ts)
    ok = ckzg.verify_kzg_proof(commitment, z_bytes, y, proof, ts)
    return proof, y, ok


def hex0(b: bytes) -> str:
    return "0x" + b.hex()


def short_hex_int(x: int, bytes_len=32) -> str:
    h = x.to_bytes(bytes_len, "big").hex()
    return f"0x{h[:8]}…{h[-8:]}"


def demo(filename: str, ts_path: str, seeds: List[int], out_path: str) -> str:
    t0 = time.time()
    # Load TS
    ts = load_ts(ts_path)
    # Map file → Fr elements → blobs
    ys = file_to_fr_elements(filename)
    blobs = fr_elements_to_blobs(ys)
    # Commit each blob, and attach a blob-level proof for integrity
    commitments: List[bytes] = []
    blob_proofs: List[dict] = []
    for idx, blob in enumerate(blobs):
        C = blob_commitment(blob, ts)
        commitments.append(C)
        prf, ok = blob_proof(blob, C, ts)
        blob_proofs.append(
            {
                "blob_index": idx,
                "commitment": hex0(C),
                "blob_proof": hex0(prf),
                "valid": ok,
            }
        )
    # Point-evaluation proofs (random z per seed over random blob)
    evals: List[dict] = []
    for sd in seeds:
        rnd = random.Random(sd)
        bidx = rnd.randrange(len(blobs))
        # derive z from seed, blob index, and a label
        z_int = int.from_bytes(
            hashlib.sha256(f"z|{sd}|{bidx}".encode()).digest(), "big"
        ) % FR_MODULUS
        if z_int == 0:
            z_int = 1
        prf, y, ok = point_eval_proof(blobs[bidx], commitments[bidx], z_int, ts)
        evals.append(
            {
                "seed": sd,
                "blob_index": bidx,
                "z_hex": short_hex_int(z_int),
                "y_hex": "0x" + y.hex(),
                "proof": hex0(prf),
                "valid": ok,
            }
        )
    t1 = time.time()

    # File metadata
    size = os.path.getsize(filename)
    with open(filename, "rb") as f:
        file_sha256 = hashlib.sha256(f.read()).hexdigest()

    out = {
        "filename": filename,
        "file_size_bytes": size,
        "file_sha256_hex": "0x" + file_sha256,
        "src_chunk_bytes": SRC_CHUNK,
        "fe_per_blob": FE_PER_BLOB,
        "fe_bytes": FE_BYTES,
        "blob_bytes": BLOB_BYTES,
        "num_src_chunks": len(ys),
        "num_blobs": len(blobs),
        "fr_modulus_hex": "0x" + FR_MODULUS_HEX,
        "commitments": [hex0(c) for c in commitments],
        "blob_proofs": blob_proofs,
        "point_eval_proofs": evals,
        "build_ms": int((t1 - t0) * 1000),
        "note": "KZG via ckzg; bytes→Fr via SHA-256 mod r; demo only.",
    }
    with open(out_path, "w") as fp:
        json.dump(out, fp, indent=2)
    print(f"[Saved ckzg demo artifact] {out_path}")
    return out_path


def main():
    p = argparse.ArgumentParser(description="Nilcoin ckzg demo (educational)")
    p.add_argument("--file", required=False, default="spec.md", help="input file path")
    p.add_argument(
        "--trusted-setup",
        dest="ts_path",
        required=False,
        default=os.environ.get("CKZG_TRUSTED_SETUP", "trust_setup.txt"),
        help="path to ckzg trusted setup (default: ./trust_setup.txt)",
    )
    p.add_argument(
        "--seeds",
        type=str,
        default="5,17,42",
        help="comma-separated integer seeds for point-eval proofs",
    )
    p.add_argument(
        "--out",
        required=False,
        default="nilcoin_kzg_ckzg_output.json",
        help="output JSON path",
    )
    args = p.parse_args()

    if ckzg is None:  # pragma: no cover
        raise SystemExit(
            f"ckzg not available: {_ckzg_import_err}. Install ckzg and retry."
        )

    if not os.path.exists(args.ts_path):  # pragma: no cover
        raise SystemExit(
            f"Trusted setup file not found: {args.ts_path}. Provide --trusted-setup."
        )

    seeds = [int(s.strip()) for s in args.seeds.split(",") if s.strip()]
    demo(args.file, args.ts_path, seeds, args.out)


if __name__ == "__main__":
    main()
