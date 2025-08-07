#!/usr/bin/env python3
"""
Nilcoin Appendix C – Constant‑Generation Script
----------------------------------------------
Given a prime modulus q and a generator g of the multiplicative group
𝔽*_q, derive all Montgomery and NTT constants required by the Nilcoin
Core spec:

  • R  = 2^64 mod q                (for one‑limb Montgomery)
  • R2 = R^2 mod q
  • Q_INV = −q⁻¹ mod 2^64
  • ψ_k  (primitive k‑th roots)    for k in {64,128,256,1024,2048}
  • k⁻¹ mod q                      (needed by INTT scaling)

Usage (example for q₁ = 998 244 353):

    $ python3 appendix_c_constants.py 998244353 3

The script exits non‑zero if the inputs do not satisfy the required
group properties.
"""
from __future__ import annotations
import sys
from math import gcd
from typing import Tuple

K_LIST = [64, 128, 256, 1024, 2048]
LIMB   = 1 << 64  # 2^64

# ----------------------------------------------------------------------
# Helpers
# ----------------------------------------------------------------------
def modinv(a: int, m: int) -> int:
    "Multiplicative inverse via extended Euclidean algorithm."
    if gcd(a, m) != 1:
        raise ValueError(f"{a=} has no inverse mod {m}")
    t, new_t, r, new_r = 0, 1, m, a
    while new_r:
        q = r // new_r
        t, new_t = new_t, t - q * new_t
        r, new_r = new_r, r - q * new_r
    return t % m

def is_primitive_root(g: int, q: int) -> bool:
    "Check if g is a multiplicative generator of F_q."
    phi = q - 1
    # Factor phi (small since q < 2^63); naive but fine once.
    factors = set()
    n = phi
    p = 2
    while p * p <= n:
        if n % p == 0:
            factors.add(p)
            while n % p == 0:
                n //= p
        p += 1 + (p & 1)  # 2,3,5,7,...
    if n > 1:
        factors.add(n)
    return all(pow(g, phi // p, q) != 1 for p in factors)

# ----------------------------------------------------------------------
# Main
# ----------------------------------------------------------------------
def derive_constants(q: int, g: int) -> Tuple[str, str]:
    if q >= LIMB:
        raise ValueError("q must be < 2^64 for 64‑bit Montgomery parameters")
    if not is_primitive_root(g, q):
        raise ValueError(f"{g} is not a primitive root modulo {q}")

    R  = LIMB % q
    R2 = (R * R) % q
    Q_INV = (-modinv(q, LIMB)) % LIMB

    table_lines = []

    table_lines.append("# -- Montgomery parameters")
    table_lines.append(f"Q      = {q}         # 0x{q:08X}")
    table_lines.append(f"R      = {R}         # 0x{R:08X}")
    table_lines.append(f"R2     = {R2}         # 0x{R2:08X}")
    table_lines.append(f"Q_INV  = {Q_INV}   # 0x{Q_INV:016X}")

    # roots and inverses
    table_lines.append("\n# -- principal roots of unity and inverses")
    for k in K_LIST:
        if (q - 1) % k != 0:
            raise ValueError(f"{k}-th root does not exist for this q")
        psi = pow(g, (q - 1) // k, q)
        # verify primitiveness
        if pow(psi, k // 2, q) == 1:
            raise ValueError(f"ψ_{k} is not primitive")
        inv_k = modinv(k, q)
        table_lines.append(
            f"k = {k:<5} ψ_k = {psi:<10} # 0x{psi:08X}   k⁻¹ = {inv_k} # 0x{inv_k:08X}"
        )

    rust_lines = [
        "\n/* Rust constants */",
        f"pub const Q:     u32 = {q};",
        f"pub const R:     u32 = {R};",
        f"pub const R2:    u32 = {R2};",
        f"pub const Q_INV: u64 = 0x{Q_INV:016X};",
        f"pub const G:     u32 = {g};",
    ]
    for k in K_LIST:
        psi = pow(g, (q - 1) // k, q)
        inv_k = modinv(k, q)
        rust_lines.append(f"pub const PSI_{k}:  u32 = {psi};")
    for k in K_LIST:
        inv_k = modinv(k, q)
        rust_lines.append(f"pub const INV_{k}:  u32 = {inv_k};")

    return "\n".join(table_lines), "\n".join(rust_lines)


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python3 appendix_c_constants.py <prime_q> <generator_g>", file=sys.stderr)
        sys.exit(1)

    q_in  = int(sys.argv[1])
    g_in  = int(sys.argv[2])

    try:
        human, rust = derive_constants(q_in, g_in)
    except ValueError as e:
        print(f"error: {e}", file=sys.stderr)
        sys.exit(1)

    print(human)
    print(rust)

