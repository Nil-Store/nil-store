# NilStore Core v 2.0

### Cryptographic Primitives & Proof System Specification

---

## Abstract


It specifies, in a fully reproducible manner:

1. **Canonical byte possession proofs** — **Proof‑of‑Useful‑Data (PoUD)** using **Kate–Zaverucha–Goldberg (KZG) polynomial commitments** and **Proof‑of‑Delayed‑Encode (PoDE)** timed window derivations (normative liveness path), evaluated strictly over a canonical byte representation per deal (`privacy_mode`).
2. **BLS VRF** and BATMAN aggregation for unbiased epoch beacons.
3. **Dial policy** and governance process for safe parameter evolution.
4. **Security rationale** and Known‑Answer Tests for all normative components.

All constants and vectors in this specification are reproducible and accompanied by deterministic Known‑Answer Tests (Annex A–B). Version 2.0 supersedes v 1.0 and v 1.0‑rc series; it MUST be implemented as a single cohesive unit by main‑network clients targeting activation height ▢`H_ACT`.

---
## § 0 Notation, Dial System & Versioning ( Baseline Profile “S‑512” )

### 0.1 Symbols, Typography, and Conventions

| Markup                    | Meaning                                               | Example         |
| ------------------------- | ----------------------------------------------------- | --------------- |
| `u8`, `u16`, `u32`, `u64` | Little‑endian unsigned integers of the stated width   | `0x0100 → 256`  |
| `≡`                       | Congruence *mod q* unless another modulus is explicit | `a ≡ b (mod q)` |
| `‖`                       | Concatenation of byte strings                         | `x‖y`           |
| `Σ`, `Π`                  | Field‑sum / product in 𝔽\_q (wrap at *q*)            | `Σ_i x_i mod q` |
| `NTT_k`                   | Length‑*k* forward Number‑Theoretic Transform         | `ntt64()`       |

All integers, vectors, and matrices are interpreted **little‑endian** unless indicated otherwise.

### 0.2 Dial Parameters

A **dial profile** defines the core cryptographic parameters and the Proof-of-Delayed-Encode (PoDE) settings.

**Economic Param Set (normative pointer).** The DAO MUST publish a hash‑pinned **$STOR‑1559 Parameter Set** (`PSet`) on L2 under `"STOR-1559-PARAMS-V1"` covering `{U* by region‑class, δ bands, β, β_min, β_max, β_emergency_min, σ_sec_max, price_cap_GiB by region/class, premium_max, egress_cap_epoch}`. Clients and watchers MUST reject price or payout calculations not derived from the current `PSet` digest, including caps and premium bounds.

| Symbol | Description                                | Baseline "S‑512"                |
| ------ | ------------------------------------------ | ------------------------------- |
| `Curve`| Elliptic Curve (for KZG and VRF)           | **BLS12-381** (Mandatory)       |
| `r`    | BLS12-381 subgroup order                   | (See §5.1)                      |
| `Nonce`| Profile Nonce (high-entropy)               | 0x1A2B3C4D5E6F7890AABBCCDDEEFF0011 (example) |
| `H_t`  | PoDE Argon2id time cost (iterations)       | 3 (Calibrated for Δ_work=1s)    |
| `H_m`  | PoDE Argon2id memory cost (KiB)            | 1048576 (1 GiB)                 |
| `H_p`  | PoDE Argon2id parallelism                  | 1 (Mandatory Sequential)        |

**Normative (PoDE Recalibration):** The NilDAO MUST establish a process for periodically monitoring baseline hardware performance and recalibrating the `H_t` and `H_m` parameters to ensure the target `Δ_work` (1s) is maintained. Recalibration requires a Minor version increment (§0.3) and MUST be announced with sufficient lead time (minimum 30 days).

Dial parameters are **frozen** per profile string (e.g., `"S-512"`).  Changes introduce a new profile ID (see § 6).

### 0.3 Version Triple

Every on‑chain 32‑byte digest begins with a **version triple**

```
Version = {major : u8 = 0x02, minor : u8 = 0x00, patch : u8 = 0x00}
digest  = Blake2s‑256( Version ‖ DomainID ‖ payload )
```

* **minor** increments when tuning `H_t, H_m`.
* **patch** increments for non‑semantic errata (typos, clarifications).

### 0.4 Domain Identifiers

`DomainID : u16` partitions digests by purpose.  Reserved values:

| ID (hex)  | Domain                             | Source section |
| --------- | ---------------------------------- | -------------- |
|  `0x0000` | Internal primitives                | § 2–5          |
|  `0x0200` | PoDE/Derive digest (window‑local)  | § 4            |
|  `0x0300` | Nil‑VRF transcripts                | § 5            |

Further IDs are allocated by NilStore governance.

#### 0.4.1 String Domain Tags (Blake2s separators)

| `"FILE-MANIFEST-V1"` | File manifest digest (Root CID)               | § A |
| `"DU-CID-V1"`        | DU ciphertext digest (per DU)                 | § A |
| `"FMK-WRAP-HPKE-V1"` | HPKE envelope for FMK                         | § A |
| `"GRANT-TOKEN-V1"`   | Retrieval grant token claims                  | § B |
For transparency and auditability, Core defines the following fixed ASCII domain strings used with Blake2s‑256 across modules:

| Tag                      | Purpose                                                     | Section |
| ------------------------ | ----------------------------------------------------------- | ------- |
| `"GRANT-TOKEN-V1"`       | Retrieval grant token claims                                | § B     |
| `"STOR-1559-PARAMS-V1"`  | Digest of $STOR‑1559 economic params (`U*`, `δ`, `β`, bands, `β_emergency_min`, `σ_sec_max`, `price_cap_GiB`, `premium_max`, `egress_cap_epoch`) published on L2 | § 0.2, § 9 |
| `"SECURITY-TREASURY-V1"` | Accounting route identifier for § 6.6.3 escalation financing | § 6.6.3 |
| `"PODE_DERIVE_ARGON_V1"` | PoDE Argon2id derive function tag                           | § 4.0a    |
| `"PODE_INPUT_DIGEST_V1"` | PoDE canon window prehashing                                 | § 4.0a    |
| `"NILSTORE-SALT-V1"`     | Placement anti‑grinding salt derivation                      | Appendix A |

| Tag                  | Purpose                                  | Section    |
| -------------------- | ---------------------------------------- | ---------- |
| `"NIL_VRF_OUT"`     | VRF output compression                    | § 5.2      |
| `"BLS12381G2_XMD:SHA-256_SSWU_RO_NIL_VRF_H2G"` | VRF hash_to_G2 DST                        | § 5.0, 5.2 |
| `"NIL_BEACON"`      | Epoch beacon derivation from VRF output   | § 5.3      |
| `"NilStore-Sample"` | Retrieval‑sampling seed from epoch beacon | § 5.7 (new) |
| `"SAMPLE-EXP"`      | PRF expansion for sampling indices        | § 5.7 (new) |
| `"DERIVE_SALT_EXP"` | Salt expansion XOF for PoDE Derive          | § 4        |
| `"BATMAN-SHARE"`    | Deterministic share‑selection label       | § 5.4.3    |

### 0.5 Change‑Control and Notice

* Parameter changes follow § 6 governance rules.
* Implementations **must** reject digests whose version triple or DomainID is unknown at compile‑time.

### 0.6 Reproducibility & Deterministic Build Charter (normative)

* **Public transcripts:** Any claim in §§ 2–5, 7 that depends on concrete parameters MUST have a reproducible transcript (JSON/CSV) in the release package (`_artifacts/`), accompanied by `SHA256SUMS`.
* **Pinned inputs:** All randomness derives from fixed domain tags (see § 0.4.1) and explicit inputs; scripts MUST use integer‑only operations for consensus‑sensitive calculations.
* **Make target:** Reference repos MUST provide `make publish` that regenerates `_artifacts/*` and `SHA256SUMS` from a clean checkout.
* **Activation gate:** Mainnet/testnet activation MUST ship: (a) production SRS digests appended to `_artifacts/SRS_HASHES.md` (demo hashes MUST remain for audit), and (b) KAT bundles for PoDE Derive and VRF/BATMAN with SHA256 sums. Clients/watchers MUST refuse to start without verifying these artifacts.


---

## § 4 Proof‑of‑Useful‑Data (PoUD) & Proof‑of‑Delayed‑Encode (PoDE)  — Normative

### 4.0a Derive (window‑scoped, normative for PoDE)

*Purpose:* deterministically compress a canonical DU window into a verifier‑recomputable digest, domain‑separated by the epoch beacon.

**Signature (normative):**

```
Derive(canon_window: bytes, beacon_salt: bytes,
       row_id: u32, epoch_id: u64, du_id: u128) -> (leaf64: bytes[64], Δ_W: bytes[32])
```

**Algorithm (Argon2id Sequential, normative):**

```
tag  = "PODE_DERIVE_ARGON_V1"
salt = Blake2s-256(tag ‖ beacon_salt ‖ u32_le(row_id) ‖ u64_le(epoch_id) ‖ u128_le(du_id))
// Hash the input data to ensure high entropy input to Argon2id, mitigating TMTO risks with low-entropy data.
input_digest = Blake2s-256("PODE_INPUT_DIGEST_V1" ‖ canon_window)
// Parameters (H_t, H_m, H_p) are defined by the Dial Profile (§0.2).
// H_p MUST be strictly 1 to enforce sequentiality. Implementations MUST reject profiles where H_p != 1.
acc = Argon2id(password=input_digest, salt=salt, t_cost=H_t, m_cost=H_m, parallelism=1, output_len=64)
leaf64 := acc
Δ_W    := Blake2s-256(canon_window)
return (leaf64, Δ_W)
```

**Notes:** (i) Domain separation prevents cross‑context collisions; (ii) Argon2id enforces timed locality via memory hardness and sequentiality; (iii) `Δ_W` supports watcher‑side caching; (iv) Known‑Answer Tests for `Derive` appear in Annex A.

### 4.0 Objective & Model

Attest, per epoch, that an SP (a) stores the canonical bytes of their assigned DU intervals and (b) can perform **timed, beacon‑salted derivations** over randomly selected windows quickly enough that fetching from elsewhere is infeasible within the proof window.

**Security anchors:** (i) DU **KZG commitment** `C_root` recorded at deal creation; (ii) BLS‑VRF epoch beacon for unbiased challenges; (iii) on‑chain **KZG multi‑open** pre‑compiles; (iv) watcher‑enforced timing digests.

### 4.1 DU Representation & Commitment
A **Data Unit (DU)** is the canonical chunking unit used for commitment and sampling.

Let a DU be encoded with systematic RS(n,k) over GF(2⁸) and segmented logically into **1 KiB symbols**.

**Normative (KZG Embedding):** To commit the data using KZG (which operates over the BLS12-381 scalar field), the DU bytes (as selected by `privacy_mode`) MUST be serialized and chunked into 31-byte elements. Each chunk is interpreted as an integer (little-endian) and embedded as a field element. The KZG commitment `C_root` is computed over the polynomial formed by these field elements.

**Normative (Deal Privacy Mode):** A deal parameter `privacy_mode ∈ {"ciphertext" (default), "plaintext"}` selects the canonical bytes over which `C_root` and PoDE derivations are computed. Providers do not require decryption keys when `privacy_mode = "ciphertext"`.

The client computes `C_root` at deal creation and posts `C_root` on L2; all subsequent storage proofs **must open against this original commitment**.

### 4.1.1 KZG Structured Reference String (SRS) (Normative)

All KZG operations MUST utilize a common, pinned Structured Reference String (SRS).
* **Provenance:** The SRS MUST be generated via a verifiable Multi-Party Computation (MPC) ceremony (e.g., Perpetual Powers of Tau). Public transcripts of the ceremony MUST be available for audit.
* **Parameters:** The SRS parameters (curve, degree) MUST align with BLS12-381 and support the maximum degree required for the largest DU commitment.
* **Verification:** Implementations MUST verify the integrity of the loaded SRS against the hash-pinned canonical SRS defined in the protocol constants. The reference digest for the ckzg demo setup is published in `_artifacts/SRS_HASHES.md` (`d39b9f2d047cc9dca2de58f264b6a09448ccd34db967881a6713eacacf0f26b7` for `demos/kzg/trusted_setup.txt`); production ceremonies MUST append their digests to this file.
* **Production requirement:** Mainnet/testnet deployments MUST NOT accept proofs until the production SRS digest has been published in `_artifacts/SRS_HASHES.md` and verified at startup.

### 4.2 Challenge Derivation

For epoch `t` with beacon `beacon_t`, expand domain‑separated randomness to pick `q` **distinct symbol indices** per DU interval and `R` **PoDE windows** of size `W = 8 MiB`. Selection MUST be modulo‑bias‑free.

### 4.3 Prover Obligations per DU Interval

1) **PoUD — KZG‑PDP (content correctness):** Provide KZG **multi‑open** at the chosen 1 KiB symbol indices proving membership in `C_root`.
2) **PoDE — Timed derivation:** For each challenged window, compute `deriv = Derive(canon_window, beacon_salt, row_id, epoch_id, du_id)` and submit `H(deriv)` plus the **minimal** canonical bytes for verifier recompute, all **within** the per‑epoch `Δ_submit` window. Enforce **Σ verified bytes ≥ B_min = 128 MiB** over all windows and **R ≥ 16** sub‑challenges/window (defaults; DAO‑tunable).
   **Normative (Security Bounds):** Governance MUST NOT set R < 8 or B_min < 64 MiB. Changes below these thresholds require a Major version increment and associated security analysis.
   **Normative (PoDE Linkage):** The prover MUST provide a KZG opening proof `π_kzg` demonstrating that the `canon_window` input bytes correspond exactly to the data committed in `C_root`.

### 4.4 Verifier (On‑chain / Watchers)

* **On‑chain:** Verify **KZG multi‑open** against `C_root`; check counters for `R` and `B_min`.
* **On‑chain (PoDE):** Verify `π_kzg` against `C_root` for the `canon_window`.
* **Watchers:** Verify PoDE recomputations and timing (RTT‑oracle transcripts). Aggregate pass/fail into an on‑chain digest per SP.

### 4.5 Coverage & Parameters (Auditor math)

Let DU contain **M** symbols. With **q** fresh symbols per epoch over **T** epochs, the chance any symbol is never checked is `M · (1 − q/M)^T`. Choose `q·T` to push this below δ (e.g., 2⁻¹²⁸) for the DU class. Governance publishes defaults and bounds.

### 4.6 On‑chain Interfaces (normative)

L1 **MUST** expose: `verify_kzg_multiopen(...)`, `verify_poseidon_merkle(...)`, `blake2s(bytes)`. Proof acceptance window: `T_epoch = 86 400 s`, `Δ_submit = 120 s` (DAO‑tunable; normative floor 60 s). Per‑replica work bound used by timing invariants: `Δ_work = 1 s`.

---

 


## § 5 Nil‑VRF / Epoch Beacon (`nilvrf`)

We use a BLS12‑381‑based **verifiable random function (VRF)** to derive unbiased epoch randomness.
### 5.0 Purpose & Design Choice

NilStore derives per‑epoch randomness from a **BLS12‑381‑based Verifiable Random Function (VRF)** that is:

* **Uniquely provable** – a single, deterministic proof per `(pk,msg)`.
* **Deterministically verifiable** on‑chain with **one pairing**.
* **Aggregate‑friendly** – shares combine linearly (BATMAN threshold, ≥ 2/3 honest).

We instantiate a **BLS‑signature‑based VRF**: VRF proofs are BLS signatures on `hash_to_G2(msg)`, and verification is a single pairing check. We follow **RFC 9380** for `hash_to_G2` (Simple SWU, XMD:SHA‑256) with a NilStore‑specific DST. **Note:** The IETF VRF standard **RFC 9381** does not define a BLS VRF; our construction relies on BLS signature **uniqueness**, which also implies an aggregator cannot grind the beacon by subset selection.  
DST (normative): `"BLS12381G2_XMD:SHA-256_SSWU_RO_NIL_VRF_H2G"`.

---

### 5.1 Notation & Parameters

| Object | Group | Encoding   | Comment                             |
| ------ | ----- | ---------- | ----------------------------------- |
| `pk`   | `G1`  | 48 B comp. | `pk = sk·G₁`                        |
| `π`    | `G2`  | 96 B comp. | Proof (BLS signature)               |
| `H`    | `G2`  | 96 B       | `H = hash_to_G2("BLS12381G2_XMD:SHA-256_SSWU_RO_NIL_VRF_H2G", msg)` |
| `e`    | —     | —          | Optimal Ate pairing `e: G1×G2→G_T`  |
| `Hash` | —     | 32 B       | Blake2s‑256, domain `"NIL_VRF_OUT"` |

Curve: **BLS12‑381**; subgroup order
`r = 0x73EDA753299D7D483339D80809A1D80553BDA402FFFE5BFEFFFFFFFF00000001`.

---

### 5.2 Algorithms (IETF BLS VRF)

#### 5.2.1 Key Generation

```rust
fn vrf_keygen(rng) -> (sk: Scalar, pk: G1) {
    sk ←$ rng();                     // 1 … r−1
    pk = sk · G1_GENERATOR;
    return (sk, pk);
}
```

#### 5.2.2 Evaluation (`vrf_eval`)

```rust
fn vrf_eval(sk: Scalar, pk: G1, msg: &[u8]) -> (y: [u8;32], π: G2) {
    H = hash_to_G2("BLS12381G2_XMD:SHA-256_SSWU_RO_NIL_VRF_H2G", msg); // RFC 9380 Simple SWU with DST
    π = sk · H;                          // BLS signature
    y = Blake2s-256("NIL_VRF_OUT" ‖ compress(pk) ‖ compress(H) ‖ compress(π));
    return (y, π);
}
```

*The output `y` is the VRF value (32 B); `π` is the proof (96 B).*

#### 5.2.3 Verification (`vrf_verify`)

```rust
fn vrf_verify(pk: G1, msg: &[u8], y: [u8;32], π: G2) -> bool {
    H   = hash_to_G2("BLS12381G2_XMD:SHA-256_SSWU_RO_NIL_VRF_H2G", msg);
    ok  = (e(pk, H) == e(G1_GENERATOR, π));      // one pairing + eq
    y′  = Blake2s-256("NIL_VRF_OUT" ‖ compress(pk) ‖ compress(H) ‖ compress(π));
    return ok && (y′ == y);
}
```

Security follows directly from the EUF‑CMA security of BLS signatures under the co‑Gap‑Diffie‑Hellman assumption on BLS12‑381.

---

### 5.3 Epoch Beacon ( solo miner )

For epoch counter `ctr`:

```
(y, π)   = vrf_eval(sk, pk, int_to_bytes_le(ctr, 8));
beacon_t = Blake2s‑256("NIL_BEACON" ‖ y);
```

The 32‑byte `beacon_t` feeds **§ 4.2** challenge derivation and seeds the retrieval‑sampling RNG per **§ 5.7**.

---

### 5.4 BATMAN Threshold Aggregation (t ≥ 2/3)

#### 5.4.1 Setup

* Committee size `N`; threshold `t = ⌈2N/3⌉`.
* Polynomial secret sharing: master key `s (= sk_master)` split into `sk_i = f(i)` degree `d = N−t`.
* Public key shares `pk_i = sk_i·G1`.
* Proof of Possession (PoP): each participant MUST provide a signature `pop_i = Sign(sk_i, pk_i)` during registration to prevent rogue‑key attacks.
* Constant public coefficients for Lagrange interpolation modulo `r`.

#### 5.4.2 Per‑epoch share posting

Each participant `i` publishes `(pk_i, π_i)` where

```
π_i = sk_i · H(epoch_ctr);
```

No `y_i` is required.

#### 5.4.3 Aggregator

Collect any `t` valid shares; compute Lagrange coefficients `λ_i` in ℤ\_r:

```
π_agg = Σ λ_i · π_i          ∈ G2           // 96 bytes
```

(No pairing, no `G_T` exponentiation.)

**Deterministic Share‑Selection (Normative, strengthened):** Participants MUST post `(pk_i, π_i)` **on L1** before `τ_close`. Let `candidate_set` be all valid shares posted before `τ_close`; let `candidate_root = MerkleRoot( canonical_encode(pk_i, π_i) for all shares in candidate_set)` published on L1 with `epoch_ctr`. The aggregator MUST:
  (a) Collect all valid shares posted on L1 before `τ_close` (the candidate set) and publish `candidate_root` (Poseidon or Blake2s) in the epoch transcript.
  (b) Calculate the ID for each share:
```
share_id_i := Blake2s‑256("BATMAN-SHARE" ‖ compress(pk_i) ‖ u64_le(epoch_ctr))
```
  (c) **Grinding Mitigation (Normative):** Let `Seed_select` be the finalized beacon of the previous epoch (`beacon_{t-1}`). This seed is fixed before share posting begins.
  (d) **Canonical Set Definition (Normative):** Compute `Score_i = HMAC-SHA256(Key=Seed_select, Message=share_id_i)` for all shares in the candidate set. The canonical aggregation set is strictly defined as the `t` shares with the lowest `Score_i` values (using `share_id_i` as a deterministic tie-breaker if scores collide). The aggregator MUST provide Merkle inclusion proofs for each selected share against `candidate_root`.
  (e) **Completeness (Normative):** The epoch transcript MUST include the total count of valid shares seen on L1 before `τ_close`. Watchers/rollup circuits MUST verify that `candidate_root` includes every timely share (fraud proofs allowed on omission) and that the selected lowest-score subset is computed from that full set.

The aggregator MUST use this canonical set to compute and publish `(π_agg, pk_agg, candidate_root)` where `pk_agg = Σ λ_i · pk_i`. Aggregations derived from incomplete candidate sets are invalid and slashable.

#### 5.4.4 On‑chain Verification & Beacon

```solidity
function verify_beacon(
    bytes48 pkAgg, bytes96 piAgg, bytes8 ctr, bytes32 candidateRoot
) returns (bytes32 beacon)
{
    G2 H = hash_to_G2("BLS12381G2_XMD:SHA-256_SSWU_RO_NIL_VRF_H2G", ctr);
    require( pairing(pkAgg, H) == pairing(G1_GEN, piAgg) );
    // Verify selected shares are the lowest-score subset of candidateRoot (inclusion + score check) off-chain or via light circuit.
    bytes32 y  = blake2s256(
        "NIL_VRF_OUT" ‖ compress(pkAgg) ‖ compress(H) ‖ compress(piAgg)
    );
    return blake2s256("NIL_BEACON" ‖ y);
}
```

*Gas:* **≈ 97 k** (1 pairing + hashes), independent of `N`.

---

### 5.5 Parameter Changes & Versioning

| Parameter                     | Governance tier | Effect                          |
| ----------------------------- | --------------- | ------------------------------- |
| Curve ID, hash‑to‑curve map   | **major**       | Affects security level          |
| Output hash (Blake2 → BLAKE3) | **minor**       | Beacon derivation               |
| Threshold `t` (`N` fixed)     | **minor**       | Liveness vs. security trade‑off |

All changes require updated KATs in Annex A.5. Derive vectors are parameterized by `(row_id, epoch_id, du_id)`.

---

### 5.6 Known‑Answer Tests (Annex A.5)

* Deterministic `vrf_keygen` seeds via ChaCha20(`seed=1`).
* Solo VRF vectors: `(msg, pk, π, y)`.
* BATMAN vectors: `(ctr, pkAgg, piAgg, beacon)` for `N=5`, `t=4`.
* All vectors and seeds MUST be published under `_artifacts/` with SHA256 digests; implementations MUST verify against the published hashes before acceptance.

---

### 5.7 Sampling Seed & Expansion (normative)

NilStore’s retrieval‑sampling RNG derives from the epoch beacon and is independent of storage‑proof circuits.

Definition (per‑epoch):

```
seed_t := Blake2s‑256("NilStore-Sample" ‖ beacon_t ‖ epoch_id)
```

Expansion (deterministic PRF stream for sampling indices):

```
ExpandSample(seed_t, i) := Blake2s‑256("SAMPLE-EXP" ‖ seed_t ‖ u32_le(i))
```

Notes:
- `seed_t` and `ExpandSample` are used off‑chain by watchers/validators to select receipts for auditing.
- Domain strings are fixed ASCII constants (see § 0.4.1).
- This derivation does not alter public inputs or verification keys.

*Sections § 6 through § 9 discuss governance, security proofs, and performance metrics building on this VRF construction.*


---

### 0.6  File Manifests & Encryption (normative pointers)

NilStore uses a content‑addressed **file manifest** (Root CID) that enumerates per‑file **Data Units (DUs)** and a **crypto policy**: a File Master Key (FMK) is HPKE‑wrapped to authorized retrieval keys; per‑DU Content Encryption Keys (CEKs) and Nonces are derived deterministically (Appendix A), and each DU is encrypted with **AES‑256‑GCM**. DU ciphertexts are addressed by `DU‑CID` = `Blake2s‑256("DU-CID-V1" || C)`. Appendix A specifies the canonical manifest, key wrapping, rekey/delete, and KATs. By default, deals set `privacy_mode = "ciphertext"`; a deal MAY opt into `privacy_mode = "plaintext"` for public/open‑data workloads.

## § 6 Product‑Aligned Economics & Operations

### 6.0 Pricing & Ask Book (normative)
- The `$STOR-1559` Parameter Set (`PSet`) published on L2 MUST include a **price curve schema**: `{BaseFee[region/class], β bands, β_floor, β_ceiling, surge_multiplier_max, σ_sec_max, price_cap_GiB by region/class, premium_max, egress_cap_epoch}`. `σ_sec_max` caps per‑epoch BaseFee adjustments (default ≤ 10% change per epoch) to damp volatility; clients and watchers MUST reject price or payout calculations not derived from the current hash‑pinned `PSet`, including caps and premium bounds.
- Providers MUST publish **standing asks** on L2: `{provider_id, capacity_free_GiB, qos_class, min_term_epochs, price_curve_id, latency_profile, region_cells[]}`. The DA chain maintains an **AskBook root** (Poseidon) updated each epoch; deals using providers absent from the AskBook are ineligible for PoUD/PoDE payouts.
- **Canonical rate:** For a DU, the client MUST select providers from the AskBook whose `price_curve_id` is in the current `PSet`; the total quoted rate = `BaseFee × β_band × surge_multiplier` (bounded by `[β_floor, β_ceiling]`). Frontends MAY display “good rate” badges when `β_band ≤ median_band + 1`.

### 6.1 Capacity Health Score, Entry & Exit (normative)
- **Capacity Health Score (`H_score`)** is computed per epoch from `{free_capacity_ratio, proof_success_rate (30d), churn, RTT_outlier_rate}`. It is hash‑pinned in the epoch transcript and drives entry/exit parameters.
- **Entry ramp:** New providers serve under probation for `N_probation = 7` epochs (DAO‑tunable). Rewards ramp linearly from 50% → 100%; slashing multiplier = `λ_entry = 1.25` during probation.
- **Exit policy:** Providers request exit with `ExitIntent{epoch, provider_id}`. Unbonding time `T_unbond` and exit fee `F_exit` are functions of capacity headroom `H_free`:
  - `headroom_raw = free_capacity_ratio / target_headroom` (default `target_headroom = 0.20`, chosen to preserve 20% spare for repairs/unbonding; governance re‑pins this in the `PSet` based on observed utilization). Systems MAY smooth `headroom_raw` via a 7‑day EMA to avoid oscillations; use `headroom = clamp(0,1, headroom_raw_smoothed)`.
  - `F_base = 0.5%` of bonded collateral; `k_fee = 2.0`; bounds `F_min = 0.5%`, `F_max = 10%`.
  - `T_base = 24h`, `k_time = 72h`; bounds `T_min = 12h`, `T_max = 7d`.
  - `F_exit = F_base × (1 + k_fee × (1 − headroom))`, clamped `[F_min, F_max]`.
  - `T_unbond = T_base + k_time × (1 − headroom)`, clamped `[T_min, T_max]`.
  When `headroom_raw ≥ 1` (surplus), `headroom` saturates at `1` so `F_exit → F_min` and `T_unbond → T_min`; when `headroom` is low, exits cost more and take longer.
- **Handoff guard:** An exit is final only after (a) repairs restore each affected DU to its target redundancy profile and (b) the provider serves proofs until `T_unbond` elapses. Early deactivation without handoff is slashable.

### 6.2 Durability Dial & Auto‑Rebalance (normative)
- **Dial mapping:** Governance MUST publish a pinned table `durability_target → {RS_n, RS_k} | {rows, cols} | profile_id`, including at least:
  - `Standard`: RS(12,9), min ring distance ≥ 2.
  - `Archive`: RS(16,12), min ring distance ≥ 3.
  - `Mission-Critical`: RS‑2D‑Hex `{rows=4, cols=7}`, ring distance ≥ 3, slice distance ≥ 2.
  The mapping table is versioned and referenced in deal metadata; clients MUST record `durability_target` and resolved profile.
- **Auto‑rebalance:** If any DU falls below its profile (due to exits or slashing), the scheduler MUST queue repairs within `T_repair_max` (DAO‑tunable) to restore the profile. Repairs MUST open against the original `C_root`; no re‑commit is allowed. Placement MUST obey the same lattice constraints and one‑shard‑per‑cell rule. Health state (`healthy/degraded/repairing`) is committed in epoch transcripts.
  - Defaults: `T_repair_max = 24h` (RS), `T_repair_max = 8h` (RS‑2D‑Hex); governance MAY tighten but MUST NOT loosen without a minor version bump.

### 6.3 Billing, Escrow & Spend Guards (normative)
- **Single escrow:** Each deal escrows storage + baseline egress in $STOR. Escrow MUST cover at least `K_epoch` of service (DAO‑tunable); if balance < `K_low`, auto top‑up MAY be enabled by the payer; otherwise the deal enters **grace mode** (reduced QoS weight, no new replicas) until topped up.
- Defaults: `K_epoch = 7` epochs funded; `K_low = 3` epochs trigger grace; auto top‑up resumes service weight upon funding.
- **Retrieval charging:** Receipts are billed per epoch at `Burn = β·BaseFee × bytes` (bounded by `β_floor/β_ceiling`) plus optional `PremiumPerByte` within `[0, premium_max]`. Users MAY set `max_monthly_spend`; SDKs MUST refuse retrieval that would exceed it unless the payer consents.
- Defaults: `premium_max = 0.5 × BaseFee`; `price_cap_GiB = 2.0 × median(BaseFee by region/class)`; `β_floor = 0.70`, `β_ceiling = 1.30`.
- **Rate caps:** Governance sets `price_cap_GiB` (per region/class) and `egress_cap_epoch`. Quotes above cap MUST be rejected by clients/watchers.
- **Emergency (yellow‑flag) behavior:** Freeze disables all fund‑moving paths (vesting, transfers/mints/burns, withdrawals/deposits), new deals/uptake, exits, slashing executions, and governance actions except the ratification vote. Storage proofs and retrieval billing continue; auto top‑ups pause. Yellow‑flag auto‑sunsets after 14 days unless ratified by a full DAO vote. These states MUST be emitted as events (see 6.4).

### 6.4 Status & Events (normative)
Chains and SDKs MUST surface the following canonical events with hashes binding to on‑chain state: `DealCreated`, `RedundancyDegraded`, `RepairScheduled`, `RepairComplete`, `ProofMissed`, `ExitRequested`, `ExitFinalized`, `FreezeActivated`, `FreezeCleared`, `SpendGuardHit`. Event hashes MUST be reproducible from the epoch transcript.

### 6.5 Research Isolation (normative)
The PoS²‑L scaffold (`rfcs/PoS2L_*`) is **research‑only** and **disabled** for all production profiles. Deals or payouts MUST NOT reference sealed‑mode proofs unless a DAO‑ratified research activation with auto‑sunset is in effect.

### 6.6 Correlation‑Aware Slashing (normative)
- **Penalty:** Missed PoUD+PoDE epochs incur `Penalty = min(0.50, 0.05 × (Consecutive_Missed_Epochs)²) × Correlation_Factor(F)`.
- **Correlation_Factor(F):** Let `F_cluster` be the fraction of total capacity within a diversity cluster (ASN×region cell, merged with any collocated /24 IPv4, /48 IPv6, or high RTT Profile Similarity) that failed in the current epoch. `Correlation_Factor(F) = 1 + α · (F_cluster)^β`, capped by `cap_corr`, with an SP floor `floor_SP = 1.0`. Defaults: `α = 1.0`, `β = 2.0`, `cap_corr = 5.0`. Bounds (DAO‑tunable, time‑locked): `α ∈ [0.5, 2.0]`, `β ∈ [2, 4]`, `cap_corr ∈ [3, 8]`. Parameters MUST stay within bounds.
- **Global events:** If `F_global > F*` (default 15%), governance MAY apply a temporary cap on network‑aggregate burn (e.g., 2%/epoch) but MUST NOT waive per‑SP penalties.
- **Enforcement:** Rollup circuits/watchers MUST apply the correlation factor to bonded $STOR slashing for missed proofs. Collocated identities MUST be merged before computing `F_cluster` to prevent Sybil dilution.

## Appendix A  File Manifest & Crypto Policy (Normative)

# Addendum A — File Manifest & Crypto Policy (Normative)
(Integrated summary)
- Root CID = Blake2s-256("FILE-MANIFEST-V1" || CanonicalCBOR(manifest)).
- DU CID = Blake2s-256("DU-CID-V1" || ciphertext||tag).
- FMK (32B) HPKE-wrapped to retrieval keys ("FMK-WRAP-HPKE-V1").
- **Normative (Deterministic Key/Nonce Derivation):**
- (CEK_32B, Nonce_12B) = HKDF-SHA256(IKM=FMK, info="DU-KEYS-V1" || du_id, L=44).
- AEAD: AES-256-GCM using the derived CEK and the deterministic 96-bit (12-byte) Nonce.
  **Normative (Security Warning - Nonce Reuse):** The security of AES-GCM is catastrophically broken if a (Key, Nonce) pair is reused. The deterministic derivation above is secure ONLY under the strict assumption that DUs are immutable (Write-Once) and that `du_id` is unique for every distinct plaintext under the same FMK.
- Rekey by adding/removing FMK wraps; delete by crypto-erasure (remove wraps).

<!-- Appendices B–F excised: application-level material lives in metaspec. -->
