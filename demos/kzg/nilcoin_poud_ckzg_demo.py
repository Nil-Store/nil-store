#!/usr/bin/env python3
"""
Nilcoin PoUD (KZG-PDP on 1 KiB symbols) — demo using Ethereum's c-kzg-4844.

Educational ONLY. Not for production.

What this does
--------------
1) Read a file; split into 1 KiB "symbols".
2) Inject each 1 KiB symbol to Fr: y_i = SHA-256(symbol) mod r.
3) Choose public x_i on a simple rational grid:
      - single-shard (<=4096 symbols):  x_i = i / N  (field division)
      - multi-shard (>4096):            x_i = j / m  within each shard of m symbols
4) Interpolate a polynomial P with P(x_i) = y_i (per shard), derive KZG commitment(s).
5) DU commitment C_root:
      - single-shard: the shard commitment
      - multi-shard:  Blake2s-256("NIL_DEMO_C_ROOT" || concat(commitments))   # demo-only aggregator
6) Challenge (seed) -> global index i; prove and verify:
      - Prover returns (symbol_bytes, y_i, z=x_i, proof π) for the shard containing i.
      - Verifier recomputes y_i from bytes, checks KZG verify(commitment, z, y_i, π),
        and checks the shard commitment is bound into C_root (by recomputing the simple root).

Why interpolation?
------------------
ckzg APIs accept *coefficients*; PoUD binds *values at public points*. We therefore
interpolate coefficients from the (x_i, y_i) pairs so that verify(z=x_i, y_i) proves
membership of the i-th 1 KiB unit.

Dependencies
------------
- pip install ckzg  (Python bindings for c-kzg-4844)
- a trusted setup file (e.g., mainnet or dev), or CKZG_TRUSTED_SETUP env var.

CLI
---
python3 nilcoin_poud_ckzg_demo.py --file ./spec.md --trusted-setup ./trusted_setup.txt \
                                  --seeds 5,17,42 --out demo_artifact.json
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

# Demo knobs
SRC_SYMBOL = 1024           # 1 KiB
MAX_SYMBOLS_PER_SHARD = 1024  # keep interpolation snappy (<=4096 is allowed; 1024 is okay in Python)

def fr(x: int) -> int:
    return x % FR_MODULUS

def fr_inv(x: int) -> int:
    if x == 0:
        raise ZeroDivisionError("inverse of 0 in Fr")
    return pow(x, FR_MODULUS - 2, FR_MODULUS)

def fr_from_symbol(block: bytes) -> int:
    h = hashlib.sha256(block).digest()
    return int.from_bytes(h, "big") % FR_MODULUS

def fe_le_bytes(x: int) -> bytes:
    if not (0 <= x < FR_MODULUS):
        raise ValueError("Fr element out of range")
    return x.to_bytes(FE_BYTES, "little")

# -------------------------
# Interpolation utilities
# -------------------------

def poly_mul_linear(coeffs: List[int], a: int) -> List[int]:
    """Return coeffs * (X - a). coeffs is little-endian: c0 + c1 X + ..."""
    n = len(coeffs)
    out = [0] * (n + 1)
    # out[k] += coeffs[k-1]  (for X^{k})
    for k in range(n):
        out[k + 1] = coeffs[k]
    # out[k] -= a * coeffs[k]
    amod = a % FR_MODULUS
    for k in range(n):
        out[k] = fr(out[k] - amod * coeffs[k])
    return out

def poly_div_linear(v: List[int], root: int) -> List[int]:
    """Given v(X) = prod_j (X - x_j), return v(X) / (X - root). Synthetic division."""
    n = len(v) - 1  # degree n
    q = [0] * n
    q[-1] = v[-1]  # leading coeff
    # Work downwards: v_k + root * q_k = q_{k-1}
    for k in range(n - 1, 0, -1):
        q[k - 1] = fr(v[k] + root * q[k])
    # ignore remainder (must be 0 if root is true root)
    return q

def interpolate_coeffs(xs: List[int], ys: List[int]) -> List[int]:
    """
    Compute coefficients of P with P(xs[i]) = ys[i].
    Algorithm: Build v(X) = Π (X - x_i) once. Then for each i:
       L_i(X) = v(X) / (X - x_i) / v'(x_i)
    P = Σ ys[i] * L_i(X)
    Complexity: O(n^2). Suitable for n <= ~1k in Python for a demo.
    """
    n = len(xs)
    if n != len(ys):
        raise ValueError("xs, ys length mismatch")
    # v(X) = ∏ (X - x_i)
    v = [1]  # constant 1
    for xi in xs:
        v = poly_mul_linear(v, xi)
    # v'(x_i) = Π_{m != i} (x_i - x_m)
    coeffs = [0] * n  # degree <= n-1
    for i in range(n):
        denom = 1
        xi = xs[i]
        for m in range(n):
            if m == i: continue
            denom = fr(denom * (xi - xs[m]))
        denom_inv = fr_inv(denom)
        # basis poly = v(X)/(X - x_i)
        basis = poly_div_linear(v, xi)  # degree n-1
        scale = fr(ys[i] * denom_inv)
        # coeffs += scale * basis
        for k in range(n):
            coeffs[k] = fr(coeffs[k] + scale * basis[k])
    return coeffs  # degree <= n-1

def coeffs_to_blob(coeffs: List[int]) -> bytes:
    """Pad coefficients to FE_PER_BLOB and serialize little-endian 32B each."""
    if len(coeffs) > FE_PER_BLOB:
        raise ValueError(f"Polynomial degree too large for one blob: {len(coeffs)}")
    padded = coeffs + [0] * (FE_PER_BLOB - len(coeffs))
    return b"".join(fe_le_bytes(c) for c in padded)

# -------------------------
# File → shards → commitments
# -------------------------

@dataclass
class Shard:
    start: int           # global symbol start index (inclusive)
    size: int            # number of symbols in this shard
    xs: List[int]        # domain points in Fr for this shard (length=size)
    ys: List[int]        # field values y_i (length=size)
    coeffs: List[int]    # interpolated coefficients (degree < size)
    commitment: bytes    # 48 byte G1 commitment (c-kzg-4844 compressed)

@dataclass
class DUCommitment:
    C_root: bytes               # DU commitment (see make_du_root)
    commitments: List[bytes]    # per-shard commitments, in order
    shard_size: int             # nominal shard size (<= MAX_SYMBOLS_PER_SHARD)
    total_symbols: int          # N

def chunk_symbols(path: str) -> List[bytes]:
    data = open(path, "rb").read()
    blocks = [data[i:i + SRC_SYMBOL] for i in range(0, len(data), SRC_SYMBOL)]
    if not blocks:
        blocks = [b""]
    if len(blocks[-1]) < SRC_SYMBOL:
        blocks[-1] = blocks[-1] + b"\x00" * (SRC_SYMBOL - len(blocks[-1]))
    return blocks

def inject_symbols_to_fr(blocks: List[bytes]) -> List[int]:
    return [fr_from_symbol(b) for b in blocks]

def make_du_root(commitments: List[bytes]) -> bytes:
    """
    Demo-only aggregation: Blake2s-256 over concatenated commitments with a domain tag.
    Production systems would use Poseidon/Merkle per spec; we stay readable here.
    """
    h = hashlib.blake2s()
    h.update(b"NIL_DEMO_C_ROOT")
    for c in commitments:
        h.update(c)
    return h.digest()

def load_ts(ts_path: str):
    if ckzg is None:
        raise RuntimeError(f"ckzg import failed: {_ckzg_import_err}. Install ckzg.")
    if not os.path.exists(ts_path):
        raise RuntimeError(f"Trusted setup not found: {ts_path}")
    return ckzg.load_trusted_setup(ts_path, 0)

def build_shards(ys: List[int], ts, max_per_shard: int = MAX_SYMBOLS_PER_SHARD) -> Tuple[List[Shard], DUCommitment]:
    N = len(ys)
    shards: List[Shard] = []
    i = 0
    while i < N:
        size = min(max_per_shard, N - i)
        # domain: x_j = j / size  (field division)
        inv_size = fr_inv(size)
        xs = [fr(j * inv_size) for j in range(size)]
        ys_slice = ys[i:i + size]
        coeffs = interpolate_coeffs(xs, ys_slice)
        blob = coeffs_to_blob(coeffs)
        C = ckzg.blob_to_kzg_commitment(blob, ts)
        shards.append(Shard(start=i, size=size, xs=xs, ys=ys_slice, coeffs=coeffs, commitment=C))
        i += size
    C_root = make_du_root([s.commitment for s in shards])
    return shards, DUCommitment(C_root=C_root,
                                commitments=[s.commitment for s in shards],
                                shard_size=max_per_shard,
                                total_symbols=N)

# -------------------------
# Prove / Verify (single index)
# -------------------------

@dataclass
class Proof:
    index: int           # global symbol index
    shard_idx: int       # which shard
    local_idx: int       # index within shard
    z_le: bytes          # point (Fr) in LE bytes
    y_le: bytes          # value (Fr) in LE bytes
    proof: bytes         # KZG opening proof
    symbol_hex: str      # first few bytes of the 1 KiB for human inspection

def draw_index(seed: int, N: int) -> int:
    # Deterministic PRF for demo; spec uses VRF beacons and rejection sampling.
    # We salt with N to avoid trivial cycles across different files.
    src = f"NIL_DEMO|{seed}|N={N}".encode()
    i = int.from_bytes(hashlib.sha256(src).digest(), "big") % N
    return i

def prove_index(shards: List[Shard], du: DUCommitment, blocks: List[bytes], ts, index: int) -> Proof:
    # Map global index -> shard + local index
    shard_idx = 0
    while shard_idx < len(shards) and not (shards[shard_idx].start <= index < shards[shard_idx].start + shards[shard_idx].size):
        shard_idx += 1
    if shard_idx == len(shards):
        raise IndexError("index out of range")
    sh = shards[shard_idx]
    local = index - sh.start
    z = sh.xs[local]
    y_expected = sh.ys[local]
    # Commit input is the blob (coeffs); compute KZG proof at z
    blob = coeffs_to_blob(sh.coeffs)
    z_le = fe_le_bytes(z)
    proof_bytes, y_bytes = ckzg.compute_kzg_proof(blob, z_le, ts)
    # Sanity: ckzg returns y = P(z). It must equal our y_expected (field)
    y_int = int.from_bytes(y_bytes, "little")
    if y_int != y_expected:
        raise RuntimeError("Interpolation/compute mismatch: y != P(z)")
    sym = blocks[index]
    return Proof(index=index,
                 shard_idx=shard_idx,
                 local_idx=local,
                 z_le=z_le,
                 y_le=y_bytes,
                 proof=proof_bytes,
                 symbol_hex=sym[:16].hex() + ("…" if len(sym) > 16 else ""))

def verify_proof(du: DUCommitment, proof: Proof, shards: List[Shard], blocks: List[bytes], ts) -> bool:
    sh = shards[proof.shard_idx]
    # 1) Recompute DU root from shard commitments (demo-only aggregation)
    C_root_check = make_du_root(du.commitments)
    if C_root_check != du.C_root:
        return False
    # 2) Bind y to the provided bytes for index
    sym = blocks[proof.index]
    y_from_bytes = fr_from_symbol(sym)
    if y_from_bytes != int.from_bytes(proof.y_le, "little"):
        return False
    # 3) KZG verify at z against the corresponding shard commitment
    C = du.commitments[proof.shard_idx]
    return ckzg.verify_kzg_proof(C, proof.z_le, proof.y_le, proof.proof, ts)

# -------------------------
# Orchestration
# -------------------------

def run_demo(filename: str, ts_path: str, seeds: List[int], out_path: str) -> str:
    ts = load_ts(ts_path)
    blocks = chunk_symbols(filename)
    ys = inject_symbols_to_fr(blocks)
    shards, du = build_shards(ys, ts, max_per_shard=MAX_SYMBOLS_PER_SHARD)
    results = []
    for sd in seeds:
        idx = draw_index(sd, du.total_symbols)
        pr = prove_index(shards, du, blocks, ts, idx)
        ok = verify_proof(du, pr, shards, blocks, ts)
        results.append({
            "seed": sd,
            "index": idx,
            "shard_idx": pr.shard_idx,
            "local_idx": pr.local_idx,
            "z_hex": "0x" + pr.z_le[::-1].hex(),  # show big-endian-ish for humans
            "y_hex": "0x" + pr.y_le[::-1].hex(),
            "commitment": "0x" + du.commitments[pr.shard_idx].hex(),
            "symbol_preview": pr.symbol_hex,
            "verified": ok,
        })
    out = {
        "filename": filename,
        "file_size": os.path.getsize(filename),
        "symbols_1KiB": len(blocks),
        "shard_count": len(shards),
        "shard_size_nominal": MAX_SYMBOLS_PER_SHARD,
        "du_C_root_hex": "0x" + du.C_root.hex(),
        "commitments_hex": ["0x" + c.hex() for c in du.commitments],
        "proofs": results,
        "note": "Demo PoUD (KZG-PDP) over 1KiB symbols. Values bound via SHA-256->Fr. Aggregation root is Blake2s(tag||concat).",
    }
    with open(out_path, "w") as fp:
        json.dump(out, fp, indent=2)
    print(f"[saved] {out_path}")
    return out_path

def parse_args(argv=None):
    p = argparse.ArgumentParser(description="Nilcoin PoUD demo on ckzg (educational)")
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

