#!/usr/bin/env python3
"""
Nilcoin PoUD (plaintext, KZG-PDP on 1 KiB symbols) — eval-form ckzg demo.
Educational ONLY. Not for production.

What this demo shows
--------------------
1) Read a file; split into 1 KiB "symbols".
2) Map each 1 KiB symbol to Fr: y_i = SHA-256(symbol) mod r.   # demo-only injection
3) Pack y_i into EIP-4844 eval-form blobs (4096 field elements per blob).
4) Commit each blob with Ethereum's c-kzg-4844 (ckzg), and compute example point proofs.
5) For a few demo seeds: derive a symbol index; prove opening at the exact EIP-4844
   domain point z = ω^j for that symbol's slot j; verify against the blob commitment.
6) Bind the returned y to the provided symbol bytes by checking y == H(symbol) mod r.

Spec alignment (intent)
-----------------------
• PoUD over plaintext with 1 KiB symbols and KZG openings at verifier-chosen indices.
  (Nilcoin: PoUD+PoDE is normative; we only demo the PoUD half here for clarity.)   # metaspec §6.0b
• DU commitment anchor: we aggregate per-blob commitments into a demo root (Blake2s-tag).
  Nilcoin's on-chain witnesses generally use Poseidon Merkle; this demo keeps it simple.  # metaspec §2.2, §3.2.y.2
• Index derivation uses seeds instead of a VRF beacon; that's fine for a demo.           # Core §5

Requires
--------
pip install ckzg         # Python bindings for c-kzg-4844
A trusted setup file (e.g., mainnet/dev); or CKZG_TRUSTED_SETUP env var.

Usage
-----
python3 nilcoin_poud_ckzg_eval_demo.py \
    --file ./spec.md \
    --trusted-setup ./trusted_setup.txt \
    --seeds 5,17,42 \
    --out nilcoin_poud_ckzg_output.json
"""

from __future__ import annotations
import argparse, hashlib, json, math, os, random, sys
from dataclasses import dataclass
from typing import List, Tuple

try:
    import ckzg  # Python bindings for c-kzg-4844
except Exception as e:
    ckzg = None
    _ckzg_import_err = e
else:
    _ckzg_import_err = None

# -------------------------
# BLS12-381 Fr (EIP-4844)
# -------------------------
FR_MODULUS = int("73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001", 16)
FE_BYTES = 32
FE_PER_BLOB = 4096
SRC_SYMBOL = 1024  # 1 KiB symbols (Normative granularity in Nilcoin; demo skips RS)  # metaspec §3.2

# Endianness for Fr encodings in this demo: EIP-4844 uses 32-byte LITTLE-ENDIAN.
FR_ENDIAN = "little"

def fr(x: int) -> int:
    return x % FR_MODULUS

def fr_from_symbol(block: bytes) -> int:
    h = hashlib.sha256(block).digest()
    return int.from_bytes(h, "big") % FR_MODULUS

def fr_to_bytes(x: int) -> bytes:
    if not (0 <= x < FR_MODULUS):
        raise ValueError("Fr element out of range")
    return x.to_bytes(FE_BYTES, FR_ENDIAN)

def bytes_to_fr(b: bytes) -> int:
    if len(b) != FE_BYTES:
        raise ValueError("bad Fr length")
    return int.from_bytes(b, FR_ENDIAN)

# -------------------------
# EIP-4844 domain (ω)
# -------------------------

def eip4844_omega() -> int:
    # 4096-th root of unity for BLS12-381 Fr; primitive generator 5 is standard.
    primitive = 5
    return pow(primitive, (FR_MODULUS - 1) // FE_PER_BLOB, FR_MODULUS)

# -------------------------
# File → 1KiB symbols → Fr → eval-form blobs
# -------------------------

def read_symbols_1k(path: str) -> List[bytes]:
    data = open(path, "rb").read()
    blocks = [data[i:i + SRC_SYMBOL] for i in range(0, len(data), SRC_SYMBOL)]
    if not blocks:
        blocks = [b""]
    if len(blocks[-1]) < SRC_SYMBOL:
        blocks[-1] = blocks[-1] + b"\x00" * (SRC_SYMBOL - len(blocks[-1]))
    return blocks

def symbols_to_fr(blocks: List[bytes]) -> List[int]:
    return [fr_from_symbol(b) for b in blocks]

def frs_to_eval_blobs(ys: List[int]) -> List[bytes]:
    """
    Pack field elements as EVALUATIONS at the EIP-4844 domain: 4096 slots per blob.
    We place y's sequentially; pad tail of last blob with zeros.
    """
    blobs: List[bytes] = []
    for i in range(0, len(ys), FE_PER_BLOB):
        group = ys[i:i + FE_PER_BLOB]
        if len(group) < FE_PER_BLOB:
            group = group + [0] * (FE_PER_BLOB - len(group))
        blob = b"".join(fr_to_bytes(v) for v in group)
        blobs.append(blob)
    if not blobs:
        blobs.append(b"\x00" * (FE_PER_BLOB * FE_BYTES))
    return blobs

# -------------------------
# Trusted setup / ckzg helpers
# -------------------------

def load_ts(ts_path: str):
    if ckzg is None:
        raise RuntimeError(f"ckzg import failed: {_ckzg_import_err}. Install ckzg.")
    if not os.path.exists(ts_path):
        raise RuntimeError(f"Trusted setup not found: {ts_path}")
    return ckzg.load_trusted_setup(ts_path, 0)

def commitment_for_blob(blob: bytes, ts) -> bytes:
    return ckzg.blob_to_kzg_commitment(blob, ts)

def point_proof(blob: bytes, z_fr: int, ts) -> Tuple[bytes, bytes]:
    """Return (proof_bytes, y_bytes) for evaluation at point z."""
    z_be = fr_to_bytes(z_fr)  # LE by default
    proof, y = ckzg.compute_kzg_proof(blob, z_be, ts)
    return proof, y

def verify_point(C: bytes, z_fr: int, y: bytes, proof: bytes, ts) -> bool:
    z_be = fr_to_bytes(z_fr)  # LE
    return ckzg.verify_kzg_proof(C, z_be, y, proof, ts)

def blob_soundness(blob: bytes, C: bytes, ts) -> Tuple[bytes, bool]:
    proof = ckzg.compute_blob_kzg_proof(blob, C, ts)
    ok = ckzg.verify_blob_kzg_proof(blob, C, proof, ts)
    return proof, ok

# -------------------------
# Demo scaffolding
# -------------------------

@dataclass
class DU:
    commitments: List[bytes]   # one per blob
    C_root: bytes              # demo aggregation root over commitments (Blake2s-tag)
    num_symbols: int

def make_C_root(commitments: List[bytes]) -> bytes:
    # Demo-only: Blake2s(tag||concat(commitments)).
    # Nilcoin uses Poseidon Merkle for on-chain proofs/witnesses in related contexts.  # metaspec §2.2
    h = hashlib.blake2s()
    h.update(b"NIL_DEMO_C_ROOT")
    for c in commitments:
        h.update(c)
    return h.digest()

def draw_index(seed: int, N: int) -> int:
    # Demo PRF: indices derive from seed and N. Nilcoin uses a VRF epoch beacon.    # Core §5
    x = hashlib.sha256(f"NIL_DEMO|{seed}|N={N}".encode()).digest()
    return int.from_bytes(x, "big") % N

def build_du(blocks: List[bytes], ts) -> Tuple[DU, List[bytes], List[bytes], List[bytes]]:
    ys = symbols_to_fr(blocks)
    blobs = frs_to_eval_blobs(ys)
    Cs = [commitment_for_blob(b, ts) for b in blobs]
    C_root = make_C_root(Cs)
    return DU(commitments=Cs, C_root=C_root, num_symbols=len(blocks)), blobs, ys, blocks

# -------------------------
# Prove / Verify for a single index
# -------------------------

@dataclass
class Proof:
    index: int
    blob_idx: int
    slot: int
    z: int
    y: bytes
    proof: bytes
    commitment: bytes
    symbol_preview_hex: str

def prove_index(seedsd: int, du: DU, blobs: List[bytes], ys: List[int], blocks: List[bytes], ts) -> Proof:
    N = du.num_symbols
    idx = draw_index(seedsd, N)
    blob_idx = idx // FE_PER_BLOB
    slot = idx % FE_PER_BLOB

    # Point to open: z = ω^slot
    ω = eip4844_omega()
    z = pow(ω, slot, FR_MODULUS)

    blob = blobs[blob_idx]
    C = du.commitments[blob_idx]

    prf, y = point_proof(blob, z, ts)

    # Human-readable preview of the 1 KiB symbol (first 16B)
    sym = blocks[idx]
    preview = sym[:16].hex() + ("…" if len(sym) > 16 else "")

    return Proof(
        index=idx,
        blob_idx=blob_idx,
        slot=slot,
        z=z,
        y=y,
        proof=prf,
        commitment=C,
        symbol_preview_hex=preview,
    )

def verify_proof(du: DU, p: Proof, blobs: List[bytes], ys: List[int], blocks: List[bytes], ts) -> bool:
    # 1) Recompute demo root (DU anchor)
    if make_C_root(du.commitments) != du.C_root:
        return False

    # 2) Bind y to the clear symbol bytes: y == H(symbol) mod r
    sym = blocks[p.index]
    y_expected = fr_from_symbol(sym)
    if y_expected != bytes_to_fr(p.y):
        return False

    # 3) KZG verify at z against the corresponding blob commitment
    ok = verify_point(p.commitment, p.z, p.y, p.proof, ts)
    return ok

# -------------------------
# Self-test: sanity check endianness & domain
# -------------------------

def selftest(ts):
    # Construct a trivial blob with evals[0] = 1, others 0; open at z=ω^0=1 → y=1.
    evals = [1] + [0] * (FE_PER_BLOB - 1)
    blob = b"".join(fr_to_bytes(v) for v in evals)
    C = commitment_for_blob(blob, ts)
    ω = eip4844_omega()
    z = pow(ω, 0, FR_MODULUS)  # = 1
    prf, y = point_proof(blob, z, ts)
    if bytes_to_fr(y) != 1:
        raise RuntimeError("Self-test failed: endianness/domain mismatch (y!=1). "
                           "If you modified FR_ENDIAN, flip to 'little' or verify your ckzg build.")

# -------------------------
# Orchestration
# -------------------------

def run_demo(filename: str, ts_path: str, seeds: List[int], out_path: str) -> str:
    ts = load_ts(ts_path)
    selftest(ts)

    blocks = read_symbols_1k(filename)
    du, blobs, ys, _ = build_du(blocks, ts)

    # Per-blob soundness proofs (optional, integrity check of blob→commitment)
    blob_results = []
    for i, (b, C) in enumerate(zip(blobs, du.commitments)):
        prf, ok = blob_soundness(b, C, ts)
        blob_results.append({
            "blob_index": i,
            "commitment": "0x" + C.hex(),
            "blob_proof": "0x" + prf.hex(),
            "valid": bool(ok),
        })

    # Point-evaluation proofs for each seed
    evals = []
    for sd in seeds:
        pr = prove_index(sd, du, blobs, ys, blocks, ts)
        ok = verify_proof(du, pr, blobs, ys, blocks, ts)
        evals.append({
            "seed": sd,
            "index": pr.index,
            "blob_index": pr.blob_idx,
            "slot": pr.slot,
            "z_hex": "0x" + fr_to_bytes(pr.z).hex(),
            "y_hex": "0x" + pr.y.hex(),
            "commitment": "0x" + pr.commitment.hex(),
            "symbol_preview": pr.symbol_preview_hex,
            "verified": bool(ok),
        })

    out = {
        "filename": filename,
        "file_size": os.path.getsize(filename),
        "symbols_1KiB": len(blocks),
        "blob_count": len(blobs),
        "fe_per_blob": FE_PER_BLOB,
        "fe_bytes": FE_BYTES,
        "du_C_root_hex": "0x" + du.C_root.hex(),
        "commitments_hex": ["0x" + c.hex() for c in du.commitments],
        "blob_proofs": blob_results,
        "point_eval_proofs": evals,
        "note": (
            "PoUD demo (plaintext KZG-PDP) on 1KiB symbols using EIP-4844 eval-form blobs. "
            "Values bound via SHA-256->Fr (demo-only). DU root is Blake2s(tag||concat). "
            "Spec: PoUD + PoDE are normative; this file demos only PoUD."
        ),
    }
    with open(out_path, "w") as fp:
        json.dump(out, fp, indent=2)
    print(f"[saved] {out_path}")
    return out_path

def parse_args(argv=None):
    p = argparse.ArgumentParser(description="Nilcoin PoUD demo on ckzg (eval-form; educational)")
    p.add_argument("--file", default="spec.md", help="input file path")
    p.add_argument("--trusted-setup", dest="ts_path",
                   default=os.environ.get("CKZG_TRUSTED_SETUP", "trusted_setup.txt"),
                   help="ckzg trusted setup path")
    p.add_argument("--seeds", type=str, default="5,17,42",
                   help="comma-separated integer seeds to demo")
    p.add_argument("--out", default="nilcoin_poud_ckzg_output.json",
                   help="output JSON path")
    return p.parse_args(argv)

def main():
    args = parse_args()
    if ckzg is None:
        raise SystemExit(f"ckzg not available: {_ckzg_import_err}. Install ckzg and retry.")
    seeds = [int(s.strip()) for s in args.seeds.split(",") if s.strip()]
    run_demo(args.file, args.ts_path, seeds, args.out)

if __name__ == "__main__":
    main()


