---

## § 0 Notation, Dial System & Versioning ( Baseline Profile “S‑q1” )

\### 0.1 Symbols, Typography, and Conventions

| Markup                    | Meaning                                               | Example         |
| ------------------------- | ----------------------------------------------------- | --------------- |
| `u8`, `u16`, `u32`, `u64` | Little‑endian unsigned integers of the stated width   | `0x0100 → 256`  |
| `≡`                       | Congruence *mod q* unless another modulus is explicit | `a ≡ b (mod q)` |
| `‖`                       | Concatenation of byte strings                         | `x‖y`           |
| `Σ`, `Π`                  | Field‑sum / product in 𝔽\_q (wrap at *q*)            | `Σ_i x_i mod q` |
| `NTT_k`                   | Length‑*k* forward Number‑Theoretic Transform         | `ntt64()`       |

All integers, vectors, and matrices are interpreted **little‑endian** unless indicated otherwise.

\### 0.2 Dial Parameters

A **dial profile** is an ordered 7‑tuple
`(m, k, r, λ, H, γ, q)`:

| Symbol | Description                                | Baseline “S‑q1”                 |
| ------ | ------------------------------------------ | ------------------------------- |
| `q`    | Prime field modulus                        | **998 244 353 (= 119·2²³ + 1)** |
| `m`    | Vector length (*nilhash*, PoSS²)           | 1 024                           |
| `k`    | NTT block size (radix‑k)                   | 64                              |
| `r`    | Passes of data‑dependent shear permutation | 2                               |
| `λ`    | Gaussian noise σ (compression)             | 2.8                             |
| `H`    | Argon2‑drizzle passes                      | 0                               |
| `γ`    | Interleave fragment size (MiB)             | 0 (sequential)                  |

Dial parameters are **frozen** per profile string (e.g. `"S-q1"`).  Changes introduce a new profile ID (see § 6).

\### 0.3 Version Triple

Every on‑chain 32‑byte digest begins with a **version triple**

```
Version = {major : u8 = 0x02, minor : u8 = 0x00, patch : u8 = 0x00}
digest  = Blake2s‑256( Version ‖ DomainID ‖ payload )
```

* **major** increments on any change to `q` or `m` (affects SIS hardness).
* **minor** increments when tuning `k, r, λ, H, γ`.
* **patch** increments for non‑semantic errata (typos, clarifications).

\### 0.4 Domain Identifiers

`DomainID : u16` partitions digests by purpose.  Reserved values:

| ID (hex)  | Domain                             | Source section |
| --------- | ---------------------------------- | -------------- |
|  `0x0000` | Internal primitives                | § 2–5          |
|  `0x0100` | nilseal row Merkle roots (`h_row`) | § 3            |
|  `0x0200` | poss² window delta proofs          | § 4            |
|  `0x0300` | Nil‑VRF transcripts                | § 5            |

Further IDs are allocated by Nilcoin governance (informative Appendix D).

\### 0.5 Change‑Control and Notice

* Parameter changes follow § 6 governance rules.
* Implementations **must** reject digests whose version triple or DomainID is unknown at compile‑time.

---

## § 1 Field & NTT Module (`nilfield`)

\### 1.1 Constants – Prime *q₁* = 998 244 353

| Name     |            Value (decimal) | Hex                | Comment                  |
| -------- | -------------------------: | ------------------ | ------------------------ |
| `Q`      |                998 244 353 | 0x3B9ACA01         | NTT-friendly prime (≈2³⁰)|
| `R`      |                932 051 910 | 0x378DFBC6         | 2⁶⁴ mod Q                |
| `R²`     |                299 560 064 | 0x11DAEC80         | *R²* mod Q               |
| `Q_INV`  | 17 450 252 288 407 896 063 | 0xF22BC0003B7FFFFF | −Q⁻¹ mod 2⁶⁴             |
| `g`      |                          3 | —                  | Generator of 𝔽\*\_Q     |
| `ψ_64`   |                922 799 308 | 0x3700CCCC         | Primitive 64‑th root     |
| `ψ_128`  |                781 712 469 | 0x2E97FC55         | Primitive 128‑th root    |
| `ψ_256`  |                476 477 967 | 0x1C667A0F         | Primitive 256‑th root    |
| `ψ_1024` |                258 648 936 | 0x0F6AAB68         | Primitive 1 024‑th root  |
| `ψ_2048` |                584 193 783 | 0x22D216F7         | Primitive 2 048‑th root  |
| `64⁻¹`   |                982 646 785 | 0x3A920001         | For INTT scaling         |
| `128⁻¹`  |                990 445 569 | 0x3B090001         | —                        |
| `256⁻¹`  |                994 344 961 | 0x3B448001         | —                        |
| `1024⁻¹` |                997 269 505 | 0x3B712001         | —                        |
| `2048⁻¹` |                997 756 929 | 0x3B789001         | —                        |

*Origin:* generated verbatim by the normative script in **Annex C**.
All reference implementations embed these literals exactly.

\### 1.2 API Definition (Rust signature, normative)

```rust
pub mod nilfield {
    /* ---------- modulus & Montgomery ---------- */
    pub const Q:      u32 = 998_244_353;
    pub const R:      u32 = 932_051_910;
    pub const R2:     u32 = 299_560_064;
    pub const Q_INV:  u64 = 0xF22BC0003B7FFFFF;

    /* ---------- field ops (constant‑time) ----- */
    pub fn add(a: u32, b: u32) -> u32;   // (a + b) mod Q
    pub fn sub(a: u32, b: u32) -> u32;   // (a − b) mod Q
    pub fn mul(a: u32, b: u32) -> u32;   // Montgomery product
    pub fn inv(a: u32) -> u32;           // a⁻¹ mod Q (Fermat)

    /* ---------- radix‑k NTT ------------------- */
    pub fn ntt64(f: &mut [u32; 64]);     // forward DIF, in‑place
    pub fn intt64(f: &mut [u32; 64]);    // inverse DIT, scaled 1/64
}
```

Implementations **shall** provide equivalent APIs in other languages.

\### 1.3 Constant‑Time Requirement

All `nilfield` functions operating on secret data **must** execute in time independent of their inputs.  Compliance criteria:

* **ctgrind**: zero variable‑time findings.
* **dudect**: Welch’s *t*‑test Δt ≤ 5 ns at 2¹⁹ traces, clock @ 3 GHz.
* **cargo‑geiger** (Rust): no `unsafe` or FFI inside `nilfield`.
  *(Inline assembly permitted in `nightly+stdsimd` with `#![feature(asm_const)]`.)*

\### 1.4 Radix‑*k* NTT Specification

* The forward transform `ntt_k` is a breadth‑first DIF algorithm using `ψ_k` twiddles; input and output are in natural order.
* The inverse transform `intt_k` is DIT with twiddles `ψ_k⁻¹`.
* Post‑inverse scaling multiplies every coefficient by `k⁻¹ mod Q`.
* For `k ∈ {64,128,256,1024,2048}` the corresponding `ψ_k` **must** be used; extending to higher powers of two requires governance approval (§ 6).

**Memory layout:** vectors are contiguous arrays of `u32` little‑endian limbs.  No bit‑reversal copy is permitted outside the NTT kernels.

**Known‑Answer Tests:** Annex A.1 & A.2 contain round‑trip vectors
`[1,0,…] → NTT → INTT → [1,0,…]` for every supported *k*.

\### 1.5 Implementation Guidance (non‑normative)

* Use 64‑bit multiplication followed by Montgomery reduction (`REDC`) with constants `(R, Q_INV)` for predictable timing on both 32‑bit and 64‑bit targets.
* For WASM or micro‑controllers lacking wide multiply, adopt Barrett reduction with pre‑computed μ = ⌊2⁶⁴ / q⌋.
* Inline `k⁻¹` scaling into the last butterfly stage to save one loop.

---

---

## § 2 Nil‑Lattice Hash / “Nilweave” (`nilhash`)

\### 2.0 Scope

`nilhash` is Nilcoin’s *vector‑commitment* primitive. It maps an arbitrary‑length byte string to an **m‑limb vector** `h ∈ 𝔽_q^m` (baseline *m = 1 024*) and—optionally—into a fixed‑size on‑chain digest. Security reduces to the hardness of the *Short‑Integer‑Solution* (SIS) problem over 𝔽\_q; the reduction appears in § 7.3.

---

\### 2.1 Message → Vector Injection (“SVT order”)

\#### 2.1.1 Padding

```
msg' = |len_u64|_LE  ‖  msg  ‖  0x80  ‖  0x00 …           // pad to even length
```

* `|len_u64|` is the original message length in **bytes**.
* Append `0x80`, then zero‑bytes until `len(msg')` is even (≥ 8 + |msg| + 1).
  *(ISO/IEC 9797‑1 scheme 1 adapted to 16‑bit limbs.)*

\#### 2.1.2 Limb parsing

`x_raw` = `msg'` parsed as little‑endian 16‑bit limbs
`x_raw = [x₀, x₁, …, x_{L−1}]` with `L = len(msg')/2`.

If `L > m` → **reject** (“message too long for profile”).
If `L < m` pad the tail with zeros.

\#### 2.1.3 SVT order (stride‑vector‑transpose)

Let `B = m / k` blocks (baseline `k = 64`, `B = 16`).
Conceptually arrange the limb array as a **k × B** row‑major matrix

```
Row r (0 … k-1) :  x_raw[r·B + c] ,  c = 0 … B-1
```

**SVT order** is the **column‑major read‑out** of this matrix:

```
SVT(x_raw)[ i ] = x_raw[ (i mod k) · B  +  ⌊i / k⌋ ] ,  0 ≤ i < m.
```

Intuition: every NTT block (row) receives one limb from each stride column, maximising inter‑block diffusion.

---
\### 2.2 Algorithms (revised)

> **Public parameters** (fixed per dial profile, derived in Annex C)
>
> * Circulant matrix **A** generated from first row `α` as before.
> * Independent circulant matrix **B** generated from first row `β[i] = g^i mod Q`, where `g = 3` (primitive root).
>     `A` and `B` are linearly independent; `rank(A‖B) = m`.

Let `rand()` sample uniform limbs in 𝔽\_Q.

| Function      | Signature                                               | Definition                                                                      |
| ------------- | ------------------------------------------------------- | ------------------------------------------------------------------------------- |
| **commit**    | `fn commit(DID, msg, rng) → (h: [u32; m], r: [u32; m])` | 1. `x = SVT(pad(msg))` (§ 2.1)  2. `r ←$ rng()` (m limbs)  3. `h = A·x  +  B·r` |
| **open**      | `fn open(msg, r) → (msg, r)`                            | Simply output the original message and the blinding vector `r`.                 |
| **verify**    | `fn verify(h, msg, r) → bool`                           | `x = SVT(pad(msg))`; return `h == A·x + B·r`.                                   |
| **update**    | *unchanged* (requires re‑commit)                        | Any change to `msg` or `r` requires a fresh `commit`.                           |
| **aggregate** | `Σ_field`                                               | Component‑wise addition of commitment vectors.                                  |

*Complexity* – Commit: two length‑m circulant multiplications (2 NTTs + 2 INTTs). Verify: same cost.
*Security* – **Perfect hiding** because `r` is uniform, **binding** under SIS(m,q) since `(A‖B)` has full rank (see § 7.3).

> **Note:** Attribute‑selective openings will appear in v 2.1 using a zero‑knowledge inner‑product argument.  For v 2.0 all openings disclose the entire message.

\#### 2.2.1  KAT impact

Annex A.3 now includes:

* `nilhash_empty_vec`   — vector `h` for `msg = ""`, `r = 0^m` (test mode)
* `nilhash_empty_full` — vector and digest for `msg = ""`, random `r` seeded with RNG = ChaCha20(`0x01`).

Existing nilseal and poss² vectors are unaffected, because they depend only on the commitment vector `h`, not on `r`.


---

\### 2.3 On‑Chain Digest Format

```
commit_digest =
    Blake2s‑256( Version ‖ DomainID ‖ h )           // 32 bytes

where  Version  = {0x02,0x00,0x00}
       DomainID = 0x0000  (internal primitive namespace)
```

The entire vector `h` (2 KiB baseline) **must** be supplied in calldata when `Version.major` increases; otherwise the 32‑byte digest is sufficient.

---

\### 2.4 Worked Example (Baseline “S‑q1”)

Input: empty string `""`, `DID = 0x0000`.

| Step              | Result (hex, little‑endian)       |
| ----------------- | --------------------------------- |
| `h` (1 024 limbs) | `f170 75ce 9788 65d7 … c386 7881` |
| `commit_digest`   | `af01 c186 … e3d9 990d` (32 B)    |

*The complete vector and digest appear in Annex A.3 as KAT `nilhash_empty`.*

---

\### 2.5 Parameterisation & Extensibility

* Increasing `m` or changing `q` → **major** version bump (§ 0.3).
* Tuning `k` or replacing `α` with a higher‑order root (e.g., `ψ_128`)
  → **minor** bump; implementers must regenerate the *A* row using Annex C.

---

\### 2.6 Implementation Notes (informative)

* **Vectorised FFT:** two 64‑point NTTs fit in AVX‑2 registers; unroll eight butterflies per stage for maximum ILP.
* **Memory‑hard variants:** set `k = 256` and keep `B = m/k` fixed to quadruple cache footprint.
* **Open/verify kernels:** the circulant property lets one reuse a single 64‑point NTT per dot‑product.

---


---

## § 3 Sealing Codec (`nilseal`)

\### 3.0 Scope & Threat Model

`nilseal` transforms a miner‑supplied **sector**—an opaque byte array of size
`S = 2^n` bytes, *n ≥ 26* (≥ 64 MiB)—into a **replica** that:

1. **Binds storage** Reproducing the replica from the clear sector and secret key takes ≥ `t_recreate_replica` seconds (§ 6).
2. **Hides data** The replica is computationally indistinguishable from uniform given only public parameters and the miner’s address.
3. **Supports proofs** It yields *row commitments* `h_row` and *delta heads* `delta_head` consumed by the Proof‑of‑Spacetime‑Squared protocol (§ 4).

Adversary capabilities: unbounded offline pre‑computation, full control of public parameters, but cannot learn the miner’s VRF secret key `sk`.

\### 3.1 Symbol Glossary (dial profile “S‑q1”)

| Symbol   | Type / default | Definition                                |
| -------- | -------------- | ----------------------------------------- |
| `S`      | 32 GiB         | Sector size (benchmark)                   |
| `row_i`  | `u32`          | `BLAKE2s-32(path‖sector_digest) mod rows` |
| `salt`   | `[u8;32]`      | `vrf(sk, row_i)`                          |
| `chunk`  | `[u32;k]`      | Radix‑*k* NTT buffer (*k = 64*)           |
| `pass`   | `0 … r−1`      | Shear‑permutation round (*r = 2*)         |
| `ζ_pass` | `u32`          | Round offset (data‑dependent)             |
| `λ`      | 2.8            | Gaussian σ (noise compression)            |
| `γ`      | 0              | MiB interleave fragment size              |

\### 3.2 Pre‑Processing – Argon2 “Drizzle”

If `H = 0` → skip.
Else perform `H` in‑place passes of **Argon2id** on the sector:

```
argon2id(
    pwd   = sector_bytes,          // streaming mode
    salt  = salt,                  // 32 B
    mem   = ⌈S / 1 MiB⌉  Kib,
    iters = 1,
    lanes = 4,
    paral = 2
)
```

Each 1 MiB Argon2 block XORs back into its original offset.  This yields a *memory‑hard* whitening keyed by the miner.

\### 3.3 Radix‑k Transform Loop

Let `N_chunks = S / (2·k)` little‑endian 16‑bit chunks.

For `pass = 0 … r−1` (baseline `r = 2`):

1. **Chunk iteration order** – determined by the **data‑dependent shear permutation** (3.4).

2. **NTT pipeline**

   ```
   NTT_k(chunk)                    // forward DIF
   for j in 0..k-1:
       chunk[j] = chunk[j] + salt[j mod 16]   mod Q
   INTT_k(chunk)                   // inverse DIT, scaled k⁻¹
   ```
**Rationale:** Salt is added in the frequency domain (after the NTT) to ensure its influence is uniformly diffused across all output limbs following the inverse transform, rather than being localized.

3. **Interleaved write**

   *If* `γ = 0` → write back to original offset.
   *Else* compute `stride = γ MiB / (2·k)` and write chunk to
   `offset = (logical_index ⋅ stride) mod N_chunks`.

\### 3.4 Data‑Dependent Shear Permutation (normative)

\#### 3.4.1 Fixed shear map

Index chunks by coordinates `(x,y)` with dimensions
`p = k` (power of two) and `q = N_chunks / p`.

A **shear step** maps `(x,y) → (x + y , y) mod (p,q)`.

\#### 3.4.2 Round‑offset ζ<sub>pass</sub>

After finishing pass `p−1`, compute a digest of the entire pass's data that is sensitive to chunk order.

`ChunkHashes_{p-1} = [SHA256(chunk_0^{p-1}), SHA256(chunk_1^{p-1}), ...]`
`ChunkDigest_{p-1} = MerkleRoot(ChunkHashes_{p-1})`
`ζ_p = little‑endian 32 bits of BLAKE2s-256( salt ‖ p ‖ ChunkDigest_{p-1} )`

**Rationale:** Using a Merkle root instead of a simple sum ensures that `ChunkDigest` depends on the precise ordering of all chunks written in the previous pass, not just their content.

Round `p` traverses chunks in ascending order of

```
(x', y') = ( (x + y + ζ_p) mod p ,  y )        // shear + data offset
```

*Security intuition* – ζ<sub>p</sub> is **unknowable** until all writes of
pass `p−1` complete, enforcing sequential work (§ 7.4.1).

\### 3.5 Gaussian Noise Compression

For every 2 KiB window **W** (post‑transform):

```
σ_Q = Q / √12                         // std‑dev of uniform limb
W' = Quantize( W + N(0, λ²·σ_Q²) )    // λ = 2.8  baseline
```

*Quantize* rounds to the nearest valid limb mod `Q`.  Noise is generated by a 32‑bit Ziggurat sampler (constant‑time).  This step thwarts statistical detection of ciphertext structure.

\### 3.6 Checkpoint Merkle Tree

* Leaf: **Blake2s‑256** of every 2 MiB slice *after* compression.
* Tree: unbalanced binary; left nodes hashed as `H = B2s(L‖R)`, rightmost branch truncated.
* Root of row *i* → `h_row[i]` (DomainID `0x0100`).
* Crash‑resume: sealing restarts from the last fully committed leaf whose authentication path exists on disk.

\### 3.7 Delta‑Row Accumulator

During compression the encoder also computes per‑1 MiB limb sums `δ_j`.
For row *i* (two windows):

```
Δ_row[i] = (δ_{2i} + δ_{2i+1})   mod Q
delta_head[i] = Blake2s-256("P2Δ" ‖ i ‖ Δ_row[i])    // DomainID 0x0200
```

Tuple `(h_row[i], delta_head[i])` is written to the **Row‑Commit file** that will be posted on‑chain after sealing.

\### 3.8 Reference Encoder (pseudocode)

```rust
fn seal_sector(path, sector_bytes, miner_sk, params) {
    let sector_digest = blake2s256(sector_bytes);
    let row_i = blake2s32(path || sector_digest) % rows;
    let salt  = vrf(miner_sk, row_i);                 // 32 B

    argon2_drizzle_if(params.H, sector_bytes, salt);

    for pass in 0..params.r {
        let ζ = compute_offset(pass, salt, sector_bytes);
        for (idx, chunk) in iter_chunks(params.k, ζ, sector_bytes) {
            ntt_k(chunk);
            add_salt(chunk, &salt, params.Q);
            intt_k(chunk);
            interleave_write(chunk, idx, params.γ, sector_bytes);
        }
    }
    gaussian_compress(sector_bytes, params.λ, params.Q);
    build_merkle_and_rowcommit(sector_bytes, salt, path);
}
```

\### 3.9 Dial Guardrails (normative limits)

| Dial | Range         | Complexity effect | Guard‑rail                            |
| ---- | ------------- | ----------------- | ------------------------------------- |
| `k`  | 64 → 256      | CPU ∝ k log k     | `k ≤ 256` fits L3 cache               |
| `r`  | 2 → 5         | Time ∝ r          | Seal time ≤ 2× network median         |
| `λ`  | 2.8 → 5.0     | Disk ↑            | λ > 4 requires compression‑ratio vote |
| `m`  | 1 024 → 2 048 | CPU ∝ m²          | Proof size constant                   |
| `H`  | 0 → 2         | DRAM × H          | H ≤ 2                                 |
| `γ`  | 0 → 4 MiB     | Seeks ↑           | γ > 0 needs HDD‑impact vote           |

Profiles violating a guard‑rail are **invalid** until approved by governance (§ 6).

---

\### 3.10 Performance Targets (baseline hardware, informative)

| Task                   | 4× SATA SSD | 8‑core 2025 CPU |
| ---------------------- | ----------- | --------------- |
| Seal 32 GiB            | ≤ 8 min     | ≤ 20 min        |
| Re‑seal from last leaf | ≤ 1 min     | ≤ 3 min         |

---

\### 3.11 Security References

Detailed proofs for sequential‑work and indistinguishability appear in § 7.4.

---

*Section § 4 describes the Proof‑of‑Spacetime‑Squared protocol that consumes `h_row` and `delta_head` produced here.*


---

## § 4 Proof‑of‑Spacetime‑Squared (`poss²`)

\### 4.0 Objective & Security Model

`poss²` is Nilcoin’s on‑chain **storage‑liveness** protocol. For each epoch `t` it forces a miner to:

1. prove that an *authenticated replica* (sealed in § 3) **still exists on local disk**, and
2. spend ≥ `Δ/5` wall‑clock time (governance parameter) per replica to recompute it, thus preventing “lazy” proofs.

Soundness relies on:

* The sequential‑work bound of `nilseal` (data‑dependent shear permutation, § 7.4.1).
* Collision resistance of Blake2s‑256 and the Merkle tree.
* The additively homomorphic row delta commitment (`delta_head`, § 3.7).

\### 4.1 Replica Layout (“Row/Column Model”)

* `S`    Sector size (bytes)
* `rows`  `= S / 2 MiB`             (Row height fixed to 2 MiB)
* `cols`  `= 2 MiB / 64 B = 32 768` (Each 64‑byte leaf index within a row)
* `window` `= 1 MiB`                (Proof reads 8 adjacent windows, ≤ 6 MiB)

A miner sealing `S = 32 GiB` obtains:

```
rows = 16 384      (indexed 0 … 16 383)
cols = 32 768      (indexed 0 … 32 767)
```

Row `i` has two 1 MiB windows **W₂i** and **W₂i+1**; their Merkle root is `h_row[i]`.  Their limb sums form `Δ_row[i]`, committed as `delta_head[i]` (§ 3.7).

\### 4.2 Challenge Derivation (Beacon Mix)

For epoch counter `ctr` and chain beacon block‑hash `B_t`:

```
ρ = Blake2s‑256( "POSS2-MIX" ‖ B_t ‖ miner_addr ‖ ctr )      // 32 B
row = u32_le(ρ[0..4]) % rows
col = u32_le(ρ[4..8]) % cols
offset = (row * 2 MiB) + (col * 64 B)                        // byte index
```

The prover **must** read eight 1 MiB windows covering
`offset - 3 MiB … offset + 4 MiB` (wrap modulo `S`).  This is ≤ 6 MiB I/O even when crossing sector boundary.

`ctr` increments monotonically; replaying an old proof with the same counter is rejected on‑chain.

\### 4.3 Proof Object `Proof64`

```
struct Proof64 {
    u16  idx_row;     // little‑endian
    u16  idx_col;
    u32  reserved = 0;
    u8   witness[56];
}
```

\#### 4.3.1 Witness layout (baseline “S‑q1”)

| Purpose               | Bytes                  | Encoding                                                        |
| --------------------- | ---------------------- | --------------------------------------------------------------- |
| Merkle path (≤ 7)     | 7 × 7 = 49 bytes       | Each sibling hash truncated to 7 bytes (Blake2s-xof)            |
| Homomorphic delta `Δ` | 4 bytes                | `u32` little-endian                                             |
| Reserved              | 3 bytes                | Padding                                                         |
| **Total** | **56 bytes** |                                                                 |
Compression is lossless for security ≥ 110 bits (§ 7.5).

\### 4.4 Prover Algorithm `pos2_prove`

```
fn pos2_prove(path, row_i, col_j, ρ) -> Proof64 {
    // 1. Locate 64‑byte leaf at (row_i, col_j)
    let leaf_offset = row_i*2MiB + col_j*64B;
    let leaf = read(path, leaf_offset, 64);

    // 2. Build compressed Merkle path (56 B)
    let witness = truncated_path(row_i, col_j, path);

    // 3. Compute Δ over the eight 1 MiB windows
    let Δ = 0;
    for wnd in sample_windows(ρ, path) {
        Δ += limb_sum(wnd);               // mod Q
    }

    // 4. Assemble proof
    let final_witness = witness_path ‖ Δ.to_le_bytes(4) ‖ [0;3];
    return Proof64 {
        idx_row = row_i,
        idx_col = col_j,
        witness = final_witness,
    }
}
```

\### 4.5 Verifier Logic

On‑chain function `poss2_verify(h_row_root, delta_head_root, proof) → bool`.

```solidity
function poss2_verify(
    bytes32 hRow, bytes32 deltaHead, Proof64 calldata p
) external pure returns (bool ok) {
    // --- Merkle inclusion check -----------------
    bytes32 leaf = blake2s_256(readLeaf(p.idx_row, p.idx_col));
    bytes32 root = reconstruct(leaf, p.witness);      // ≤ 7 hashes
    if (root != hRow) return false;

    // --- Homomorphic delta check ----------------
    // Extract Δ from the witness field
    uint32 Δ = bytes_to_u32_le(p.witness[49..53]);
    bytes32 chk = blake2s_256(abi.encode("P2Δ", p.idx_row, Δ));
    if (chk != deltaHead) return false;

    return true;
}
```

Gas upper bound (Berlin): **9 700 ± 50** with pre‑compiled Blake2s.

\### 4.6 Performance Targets

| Step              | Disk I/O | CPU (ms) | Gas      |
| ----------------- | -------- | -------- | -------- |
| Prove (miner)     | ≤ 6 MiB  | ≤ 50     | —        |
| Verify (on‑chain) | —        | —        | ≤ 10 000 |

\### 4.7 Security Assertions (reference § 7.5)

* **Soundness:** Any prover who forges `(row, col)` without the replica must either (i) invert Blake2s (Merkle path) or (ii) solve SIS by finding a new `Δ` that collides with committed `delta_head`.
* **Sequentiality:** Challenge uses fresh beacon hash `B_t`; proofs prepared in advance fail with overwhelming probability.
* **Window overlap:** 8 windows (12.5 % amplification) achieves 110‑bit failure probability over 24 h for β = 0.2 fault rate.

\### 4.8 Versioning

`poss²` is bound to the dial profile.  Changing `(rows, window, hash)` requires a **minor** version bump (§ 0.3) and regenerated Annex B vectors.

---

*Section § 5 defines the Nil‑VRF used to derive the `salt` input of `nilseal` and the proof‑epoch beacon above.*

