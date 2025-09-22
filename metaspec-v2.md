# NilStore Network: A Protocol for Decentralized, Verifiable, and Economically Efficient Storage

**(White Paper Draft v0.5)**

**Date:** 2025-09-15
**Status:** Working Draft
**Authors:** NilStore Core Team

## Abstract

NilStore is a decentralized storage network designed to provide high-throughput, verifiable data storage with significantly reduced operational overhead. It leverages a novel consensus mechanism, Proof-of-Spacetime-Squared (PoS²), which merges storage verification with bandwidth accounting in a single, succinct proof. By utilizing CPU-efficient sealing based on KZG commitments and a topological data placement strategy (Nil-Lattice), NilStore drastically lowers the hardware barrier to entry, enabling participation from edge devices to data centers. This paper details the system architecture, the NilFS data abstraction layer, the Nil-Mesh routing protocol, the dual-token ($STOR/$BW) economic model, and the hybrid L1/L2 settlement architecture designed for EVM compatibility and robust governance.

## 1. Introduction

### 1.1 Motivation

While existing decentralized storage protocols have demonstrated the viability of incentive-driven storage, they often rely on computationally intensive Proof-of-Replication (PoRep) stacks requiring significant GPU investment. This centralizes the network around large-scale operators and increases the total cost per byte.

NilStore retains strong cryptographic guarantees while reducing the "sealing" process to minutes on standard CPUs. This democratization of access increases network resilience through geographic distribution and enables a more efficient storage marketplace.

### 1.2 Key Innovations

*   **PoS² (Proof-of-Spacetime-Squared):** A merged SNARK attesting to both continued data storage and bandwidth served within an epoch.
*   **CPU‑Only Sealing (baseline/target):** **Baseline** ≤ 8 min for **32 GiB** on an 8‑core 2025 CPU (see Nilcoin Core v2.0 § 9); **Target** ≤ 5–8 min for **64 GiB** (subject to disk bandwidth and profile). State targets as *target* until reproduced in Annex benchmarks.
*   **Nil-Mesh Routing:** Heisenberg-lifted K-shortest paths for optimized latency and Sybil resistance.
*   **Dual-Token Economy:** Decoupling long-term capacity commitment ($STOR) from immediate utility ($BW).
*   **Hybrid Settlement:** A specialized L1 for efficient proof verification bridged via ZK-Rollup to an EVM L2 for liquidity and composability.

## 2. System Architecture

NilStore employs a hybrid architecture that decouples Data Availability (DA) consensus from economic settlement, optimizing for both cryptographic efficiency and ecosystem composability.

### 2.1 Architectural Layers

1.  **Data Layer (NilFS):** Handles object ingestion, erasure coding, and placement.
2.  **Network Layer (Nil-Mesh):** Manages peer discovery, routing, and QoS measurement.
3.  **Consensus Layer (DA Chain - L1):** Verifies PoS² proofs, manages stake, and mints rewards.
4.  **Settlement Layer (L2 Rollup):** Handles economic transactions, liquidity, and governance.

### 2.2 The DA Chain (L1)

The Data Availability Chain is a minimal L1 (built using Cosmos-SDK/Tendermint BFT) optimized for NilStore's cryptographic operations.

*   **Function:** Verifying KZG openings and PoS² SNARKs efficiently via pre‑compiles (avoiding expensive EVM gas costs), managing $STOR staking, and executing slashing logic. It does not run a general‑purpose VM.
*   **Required pre‑compiles (normative):** (a) BLAKE2s‑256, (b) Poseidon (for Merkle paths), and (c) KZG (G1/G2 ops; multi‑open). Chains lacking these MUST expose equivalent syscalls in the DA module.
*   **Rationale:** The intensive cryptographic operations required for daily proof verification are best handled natively.

### 2.3 The Settlement Layer (L2)

Settlement occurs on a ZK-Rollup (using PlonK/Kimchi) bridged to a major EVM ecosystem (e.g., Ethereum L2).

*   **Function:** Manages ERC-20 representations of $STOR and $BW, mints Deal NFTs, hosts the NilDAO, and integrates with DeFi.

### 2.4 The ZK-Bridge

The L1 aggregates all epoch PoS² proofs into a single recursive SNARK and posts it to the L2 bridge contract. **Normative circuit boundary**:
1) **Public inputs**: `{epoch_id, DA_state_root, poss2_root, bw_root, validator_set_hash}`.
2) **Verification key**: `vk_hash = sha256(vk_bytes)` pinned in the L2 bridge at deployment; upgrades require DAO action and timelock. In addition, an Emergency Circuit MAY perform an expedited VK upgrade with a shorter timelock under §9.2, **restricted to a whitelisted diff** whose hash is pre‑published on L1. Emergency mode MUST operate in “yellow‑flag” state: the bridge updates `epoch_id` and roots but quarantines fund‑moving paths until DAO ratifies or the sunset elapses. An independent auditor attestation (posted on‑chain) is REQUIRED.
3) **State mapping**: On accept, the bridge **atomically** updates `{poss2_root, bw_root, epoch_id}`; monotonic `epoch_id` prevents replay.
4) **Failure domains**: any mismatch in roots or non‑monotonic epoch causes a hard reject. No trusted relayers or multisigs are required because validity is enforced by the proof and pinned `vk_hash`.

### 2.5 Cryptographic Core Dependency

All layers rely on the primitives defined in the **NilStore Cryptographic Core Specification (`spec.md`)**, which establishes the security guarantees for data integrity and proof soundness.

## 3. Data Layer (NilFS)

NilFS abstracts data management complexity, automating the preparation, distribution, and maintenance of data, ensuring neither users nor Storage Providers (SPs) manage exact file representations or replication strategies manually.

### 3.1 Object Ingestion and Data Units (DUs)

1.  **Content-Defined Chunking (CDC):** Ingested objects are automatically split using CDC (e.g., Rabin fingerprinting) to maximize deduplication. Chunks are organized into a Merkle DAG (CIDv1 compatible).
2.  **Data Unit Packing:** Chunks are serialized and packed into standardized **Data Units (DUs)**. DU sizes are powers-of-two (1 MiB to 8 GiB). SPs interact only with DUs.

#### 3.1.1 Upload Walkthrough (Informative)

This walkthrough illustrates what happens when a client uploads an object **F** to NilStore:

1) **Chunk & DAG (CDC).** The client runs content‑defined chunking (e.g., Rabin) over **F**, producing a Merkle‑DAG (CIDv1‑compatible).
2) **Pack into DUs.** Chunks are serialized into one or more **Data Units (DUs)** (power‑of‑two size between 1 MiB and 8 GiB). Each DU is self‑contained.
3) **Commit & deal intent.** The client computes a DU commitment (`C_root`) and prepares `CreateDeal` parameters (price, term, redundancy, QoS).
4) **Erasure coding.** Each DU is encoded with Reed–Solomon **RS(n,k)** (default **(12,9)**), yielding **n** shards (k data + n−k parity).
5) **Deterministic placement.** For every shard `j`, compute a Nil‑Lattice **ring‑cell** target via
   `pos := Hash(CID_DU ∥ ClientSalt_32B ∥ j) → (r,θ)` and enforce placement constraints (one shard per SP per cell; cross‑cell distance threshold).
6) **Deal creation (L2).** The client calls **`CreateDeal`** on L2, posting `C_root`, locking $STOR escrow, and minting a **Deal NFT**.
7) **Miner uptake (L2+L1).** Selected SPs bond $STOR, fetch their assigned shards, and seal them into sectors on L1 using **`nilseal`**, producing row commitments `h_row` and delta heads `delta_head` needed by **PoS²**.
8) **Epoch service.** During each epoch, SPs (a) serve retrievals; clients sign **Ed25519 receipts**; SPs aggregate receipts into a Poseidon Merkle (`BW_root`), and (b) post an epoch **PoS²** proof binding storage and bandwidth.
9) **Settlement & rewards.** L1 recursively aggregates PoS² and posts a validity proof to L2 (**ZK‑Bridge**). L2 updates `{poss2_root, bw_root, epoch_id}` and releases vested fees / $BW rewards per distribution rules.
10) **Repair (as needed).** If shard availability drops below threshold, the network triggers **Autonomous Repair**—repaired shards must open against the original DU commitment (no drift).

**Message flow (illustrative):**
```
Client  →  SDK:  CDC+DAG → DU pack → RS(n,k) → placement map
Client  →  L2:  CreateDeal(C_root, terms) → Deal NFT
SP      ←  L2:  MinerUptake(collateral)   ← selected SPs
SP      ↔  L1:  Seal (h_row, delta_head)  ; Epoch PoS²(bw_root)
Clients ↔  SP:  Retrieval ↔ Receipts(Ed25519) → SP aggregates
L1      →  L2:  ZK-Bridge(recursive SNARK) → state update & vesting
```

### 3.2 Erasure Coding and Placement

**Normative (Redundancy & Sharding).** Each DU is encoded with **systematic Reed–Solomon over GF(2^8)** using **symbol_size = 1 KiB**. A DU is striped into **k** equal data stripes; **n−k** parity stripes are computed; stripes are concatenated per‑shard to form **n shards** of near‑equal size. The default profile is **RS(n=12, k=9)** (≈ **1.33×** overhead), tolerating **f = n−k = 3** arbitrary shard losses **without** data loss.
**Normative (Placement constraints).** At most **one shard per SP per ring‑cell**; shards of the same DU MUST be placed across distinct ring‑cells with a minimum topological separation (governance‑tunable).
**Normative (Repair triggers).** A **RepairNeeded** event is raised when `healthy_shards ≤ k+1`. Repairs MUST produce openings **against the original DU commitment**; re‑committing a DU is invalid.
**Deal metadata MUST include:**
`{profile_type, RS_n?, RS_k?, rows?, cols?, symbol_size, ClientSalt_32B, lattice_params, placement_constraints, repair_threshold, epoch_written, meta_root?, meta_scheme?}`.

Notes:
- `profile_type ∈ {"RS-Standard","RS-2D-Hex","dial"}`.
- `epoch_written` records the epoch the DU was first committed and is REQUIRED for routing (§7.4).
- `meta_root` (Poseidon) and `meta_scheme` are REQUIRED when the resolved profile uses encoded metadata (all RS‑2D‑Hex profiles; optional for RS).
- Fields marked `?` are present if required by the resolved profile and/or metadata scheme (see §3.2.y–§3.2.z).

*   **Deterministic Placement (Nil-Lattice):** Shards are placed on a directed hex-ring lattice to maximize topological distance. The coordinate (r, θ) is determined by:
    `pos := Hash(CID_DU ∥ ClientSalt_32B ∥ SlotIndex) → (r, θ)`
The `ClientSalt` ensures cross-deal uniqueness.

### 3.2.y RS‑2D‑Hex Profile (Optional)

#### 3.2.y.0 Objective
The RS‑2D‑Hex profile couples two‑dimensional erasure coding with NilStore’s hex‑lattice.
It maps row redundancy → radial rings and column redundancy → angular slices, enabling O(|DU|/n) repair bandwidth under churn.

#### 3.2.y.1 Encoding
- Partition the DU into an [r × c] symbol matrix (baseline r = f+1, c = 2f+1).
- Column‑wise RS(n,r) → primary slivers; row‑wise RS(n,c) → secondary slivers.
- Each SP is assigned a (primary_i, secondary_i) sliver pair.

#### 3.2.y.2 Commitments & Metadata
- Each sliver MUST be bound by a KZG commitment; the DU MUST expose a blob commitment `C_root`.
- Deal metadata MUST carry `{rows, cols, symbol_size, commitment_root, placement_constraints}`.

#### 3.2.y.3 Lattice Placement (Normative)
- Row→rings: All slivers in a given row MUST lie on distinct radial rings.
- Col→slices: All slivers in a given column MUST lie on distinct angular slices.
- Cross‑cell: An SP MUST NOT hold >1 sliver in the same (r,θ) cell.

#### 3.2.y.4 Read & Repair
- Read: Collect ≥ 2f+1 secondary slivers across rings, reconstruct DU, re‑encode, verify `C_root`.
- Repair:
  - Secondary sliver repair: query f+1 neighbors on the same ring.
  - Primary sliver repair: query 2f+1 neighbors on the same slice.
- All repairs MUST open against the original DU commitment (§3.3).

#### 3.2.y.5 Integration with PoS²
RS‑2D‑Hex affects only NilFS shard layout; PoS² remains file‑agnostic and unmodified (§6).

#### 3.2.y.6 Governance
- Default RS(12,9) remains mandatory.
- RS‑2D‑Hex MAY be enabled per pool/class; SPs MUST advertise support at deal negotiation.

### 3.2.z Durability Dial Abstraction

#### 3.2.z.0 Objective
Expose a user‑visible durability_target ∈ [0.90, 0.999999999] that deterministically resolves to a governance‑approved redundancy profile (RS‑Standard or RS‑2D‑Hex).

#### 3.2.z.1 Mapping & Metadata
- Client sets `profile_type="dial"` and `durability_target`.
- The resolver MUST produce `resolved_profile := {RS_n, RS_k} | {rows, cols}` and placement constraints.
- Deal metadata MUST record `{profile_type="dial", durability_target, resolved_profile}`.

#### 3.2.z.2 Late‑Joiner Bootstrap (Completeness)
Any SP missing its assigned sliver after dispersal MUST be able to reconstruct it without the writer online, using row/column intersections per the resolved profile. This guarantees eventual completeness.

#### 3.2.z.3 Encoded Metadata (Scalability)
For RS‑2D‑Hex, sliver‑commitment metadata MUST be encoded linearly (e.g., 1D RS over the metadata vector). SPs store only their share; gateways/clients reconstruct on demand.

#### 3.2.z.3.1 Encoded Metadata Object (Normative)

meta_scheme: `RS1D(n_meta, k_meta)` with default `k_meta = f+2` and `n_meta = n`.
Each Storage Provider stores one `MetaShard`:

`MetaShard := { du_id, shard_index, payload, sig_SP }`

- `payload` encodes that SP’s share of the sliver‑commitment vector (and per‑sliver KZG commitments as needed).
- `sig_SP` binds the shard to `du_id` and `shard_index`.
- All `MetaShard.payload` chunks are Poseidon‑Merkleized to form `meta_root` recorded in Deal metadata (§3.2).

Verification (clients/gateways):
1) Fetch ≥ `k_meta` `MetaShard`s with Merkle proofs to `meta_root`.
2) Reconstruct the commitment vector.
3) Verify sliver openings against the DU commitment during reads/repairs.

Implementations MAY cache reconstructed metadata; caches MUST be invalidated on DU invalidation events (§3.2.z.4).

#### 3.2.z.4 Writer Inconsistency Proofs (Fraud)

If an SP detects inconsistency between a received sliver and `C_root`, it MUST produce an `InconsistencyProof`:

**Normative (Authenticated Transfer):** During initial data dispersal, the writer MUST sign each sliver sent to the SPs. The signature MUST cover the sliver content, the `du_id`, and the `sliver_index`. The `InconsistencyProof` MUST include this signature (`Sig_Writer`).

`InconsistencyProof := { du_id, sliver_index, symbols[], openings[], meta_inclusion[], witness_meta_root, witness_C_root }`

- `symbols[]` and `openings[]` provide the minimum symbol‑level data needed to re‑encode and check commitment equality.
- `meta_inclusion[]` are Merkle proofs to `meta_root` for the relevant sliver commitments.
- `witness_meta_root` and `witness_C_root` bind to on‑chain Deal metadata and DU commit.

On‑chain action: Any party MAY call `MarkInconsistent(C_root, evidence_root)` on L1 (DA chain), where `evidence_root` is a Poseidon‑Merkle root of ≥ f+1 `InconsistencyProof`s from distinct SPs. If valid, the DU is marked invalid, excluded from PoS²/BW accounting, and the writer’s escrowed $STOR is slashed per §7.3.

#### 3.2.z.5 Lattice Coupling
Resolved profiles MUST respect §3.2.y placement rules for 2D cases, and the standard ring‑cell separation for RS.

#### 3.2.z.6 Governance
NilDAO maintains the mapping table (durability target → profile), caps allowed ranges, and sets cost multipliers per profile.

### 3.3 Autonomous Repair Protocol

The network autonomously maintains durability through a bounty system.

1.  **Detection:** If DU availability drops below the resilience threshold (e.g., k+1), a `RepairNeeded` event is triggered.
2.  **Execution:** Any node can reconstruct the missing shards **and MUST produce openings against the original DU KZG commitment** (no new commitment is accepted). The repair submission includes a Merkle proof to the DU’s original `C_root` plus KZG openings for the repaired shards.
3.  **Resilience Bounty:** The first node to submit proof of correct regeneration claims the bounty (default: 5% of the remaining escrowed fee for that DU).

**Normative (Anti‑withholding):** When a repair for shard `j` is accepted, the SP originally assigned `j` incurs an automatic demerit; repeated events within a sliding window escalate to slashing unless the SP supplies signed RTT‑oracle transcripts proving inclusion in a whitelisted incident. An identity is disqualified from claiming bounty on any shard it was previously assigned for `Δ_repair_cooldown` epochs (DAO‑tunable).
    **Normative (Dynamic Bounty):** The bounty MUST be dynamically adjusted based on the urgency of the repair, the cost of reconstruction, and network conditions (DAO‑tunable parameters).

## 4. Network Layer (Nil-Mesh)

Nil-Mesh is the network overlay optimized for low-latency, topologically aware routing.

### 4.1 Heisenberg-Lifted Routing

Nil-Mesh utilizes the geometric properties of the Nil-Lattice for efficient pathfinding.

*   **Mechanism:** Peer IDs are mapped ("lifted") to elements in a 2-step nilpotent Lie group (Heisenberg-like structure) corresponding to their lattice coordinates.
*   **Pathfinding:** K-shortest paths (K=3) are computed in this covering space and projected back to the physical network. This offers superior latency performance compared to standard DHTs and increases Sybil resistance by requiring attackers to control entire topological regions ("Ring Cells").

### 4.2 RTT Attestation and QoS Oracle

Verifiable Quality of Service (QoS) is crucial for performance and security.

*   **Attestation:** Nodes continuously monitor and sign Round-Trip Time (RTT) attestations with peers.
*   **On‑Chain Oracle:** A **stake‑weighted attester set** posts RTT digests (Poseidon Merkle roots) to the DA chain. **Normative**:
    1) **Challenge‑response**: clients issue random tokens; SPs must echo tokens within `T_max`; vantage nodes verify end‑to‑end.
    2) **VDF Enforcement (Conditional):** MAY be activated only if the anomaly rate exceeds `ε_sys` for ≥ 3 consecutive epochs and MUST be deactivated after 2 consecutive clean epochs. VDF cost per probe is capped by the **Verification Load Cap** (§ 6.1). Protocol MUST publish the VDF parameters (delay, modulus) on‑chain per epoch when active.
    3) **Diversity**: attesters span ≥ 5 regions and ≥ 8 ASNs; assignments are epoch‑randomized.
    4) **Slashing**: equivocation or forged attestations are slashable with on‑chain fraud proofs (submit raw transcripts).
    5) **Sybil control**: weight attesters by bonded $STOR and decay weights for co‑located /24s.
*   **Usage:**
    1.  **Path Selection:** Clients use the Oracle to select the fastest 'k' providers.
    2.  **Fraud Prevention:** The Oracle verifies that bandwidth receipts are genuine (verifying RTT > network floor), preventing Sybil self-dealing.

## 5. Economic Model (Dual-Token)

NilStore employs a dual-token economy to decouple long-term security incentives from short-term utility incentives.

### 5.1 $STOR (Staking and Capacity Token)

*   **Supply:** Fixed (1 Billion).
*   **Functions:** Staking collateral for SPs and Validators; medium of exchange for storage capacity; governance voting power.
*   **Sink:** Slashing events.

### 5.2 $BW (Bandwidth Scrip)

$BW is the utility token rewarding data retrieval. It is elastic and minted based on network activity.

#### 5.2.1 Inflation Formula (Minting)

Inflation per epoch (I_epoch) is calculated using a sublinear function to incentivize usage while controlling inflation:

`I_epoch = clamp( α · sqrt(Total_Bytes_Served_NetworkWide), 0, I_epoch_max )`  // **Normative bounds:** (a) Per‑epoch cap: `I_epoch_max ≤ 0.10%` of circulating $BW (DAO‑tunable within [0.02%, 0.10%]); (b) Rolling cap: over any 30‑day window, Σ I_epoch ≤ `I_30d_max` (DAO‑tunable corridor, default 6%); (c) Attack‑traffic filter: bytes counted toward inflation MUST be discounted by an abuse‑score derived from §6.3 sampling failures and RTT‑oracle anomalies.
where:
- `α ∈ [α_min, α_max]` (DAO‑tunable);
- `I_epoch_max` caps epoch inflation (DAO‑tunable);
- per‑DU and per‑Miner **service caps** apply to the counted bytes to mitigate wash‑trading.

*   α (Alpha) is a governance-tunable constant scaling the inflation rate.

#### 5.2.2 Distribution

Minted $BW is distributed pro-rata to SPs based on the volume of verified retrieval receipts submitted in their PoS² proofs.

#### 5.2.3 Burn Mechanism (Tipping)

Users can optionally "tip" for priority retrieval by including a `tip_bw` amount in the receipt, which is burned upon settlement.

*   **Incentive Alignment:** Distribution shares are calculated *before* the burn. By capturing tipped traffic, a miner increases their effective share of the total inflation pie relative to others and improves their QoS reputation, creating competition for prioritized traffic.

### 5.3 Stablecoin UX

While the protocol strictly uses $BW for tips, client software can provide a seamless stablecoin (e.g., USDC) experience by executing a DEX swap (USDC -> $BW) client-side before signing the receipt.

## 6. Consensus and Verification (PoS²)

The economic model is enforced cryptographically through the PoS² consensus mechanism on the L1 DA Chain.

### 6.1 Retrieval Receipts

To account for bandwidth, clients sign receipts upon successful retrieval.

*   **Receipt Schema (Normative):**
    `Receipt := { CID_DU, Bytes, ChallengeNonce, ExpiresAt, Tip_BW, Miner_ID, Client_Pubkey, Sig_Ed25519 [, GatewaySig?] }`
    - `ChallengeNonce` is issued per‑session by the SP/gateway and bound to the DU slice; `ExpiresAt` prevents replay.
    - **Verification model:** Ed25519 signatures are verified **off‑chain by watchers and/or on the DA chain**; PoS² only commits to a **Poseidon Merkle root** of receipts and proves byte‑sum consistency. In‑circuit Ed25519 verification is **not required**.  
      **Commit requirement:** For epoch `t`, SPs MUST have posted `BW_commit := Blake2s‑256(BW_root)` by the last block of epoch `t−1` (see § 6.3.1). Receipts not covered by `BW_commit` are ineligible.  
      **Penalty:** Failure to post `BW_commit` for epoch `t` sets counted bytes to zero for `t` and forfeits all $BW rewards for `t`.  
      **Normative anchor:** At least **2% of receipts by byte‑volume per epoch** MUST be verified on the DA chain (randomly sampled via § 6.3) and escalate automatically under anomaly (§ 6.3.4).  
      **Normative (Verification Load Cap):** The total on‑chain verification load MUST be capped (DAO‑tunable) to prevent DoS via forced escalation.

### 6.2 PoS² Binding (Storage + Bandwidth)

The PoS² SNARK proves two statements simultaneously:

1.  **Storage Verification:** Knowledge of KZG openings for randomly challenged shards at the beacon-derived evaluation point (x★).
2.  **Bandwidth Accounting:**
    *   SPs aggregate epoch receipts into a Poseidon Merkle Tree (`BW_root`), using the domain separator "NilStore-BW-v1".
    *   The SNARK verifies the consistency of `BW_root`.
    *   The SNARK asserts that the total bytes served meets a minimum threshold (`Σ bytes ≥ B_min`).

### 6.3 Probabilistic Retrieval Sampling (QoS Auditing)

#### 6.3.0 Objective
Strengthen retrieval QoS without suspending reads by sampling and verifying a governance‑tunable fraction of receipts each epoch. This mechanism is additive to PoS² and does not alter the proof object.

#### 6.3.1 Sampling Set Derivation
0) **Commit‑then‑sample (Normative):** Each SP MUST post `BW_commit := Blake2s‑256(BW_root)` no later than the last block of epoch `t−1`.
1) At epoch boundary `t`, derive `seed_t := Blake2s-256("NilStore-Sample" ‖ beacon_t ‖ epoch_id)`, where `beacon_t` is the Nil‑VRF epoch beacon.
2) Expand `seed_t` into a PRF stream and select a fraction `p` of receipts **from the set committed by `BW_commit`** for verification (`0.5% ≤ p ≤ 10%`, default ≥ 5%). Receipts not committed in `t−1` MUST NOT be counted for `t`.
3) The sample MUST be unpredictable to SPs prior to epoch end and sized so that expected coverage ≥ 1 receipt per active SP. Auditor assignment SHOULD be stake‑weighted and region/ASN‑diverse (per §4.2) to avoid correlated blind‑spots and to bound per‑auditor load.
4) **Honeypot DUs:** MUST be **profile‑indistinguishable** from ordinary DUs: sizes drawn from the same power‑of‑two distribution; RS profiles sampled from governance‑approved mixes; Nil‑Lattice slots assigned via the standard hash; and metadata randomized within normal bounds. Any retrieval receipt for a Honeypot DU is automatically selected for 100% verification.

#### 6.3.2 Verification Procedure
Watchers (or DA validators) MUST, for each sampled receipt:
- Verify Ed25519 client signature and expiry.
- Check `ChallengeNonce` uniqueness and binding to the DU slice.
- Verify RTT transcript via the QoS Oracle (§4.2) meets declared bounds.
- Verify inclusion in `BW_root` (Poseidon path).
Aggregate results into `SampleReport_t`.

#### 6.3.3 Enforcement
- Pass: If ≥ (1−ε) of sampled receipts per SP verify (`ε` default 1%), rewards vest as normal.
- Fail (Minor): If failures ≤ ε, deduct failing receipts from counted bytes and forfeit all $BW rewards for the epoch.
- Fail (Major): If failures > ε, deduct failing receipts, forfeit rewards, and apply quadratic slashing to bonded $STOR. Repeat offenders MAY be suspended pending DAO vote.

#### 6.3.4 Governance Dials
NilDAO MAY tune: sampling fraction `p`, tolerance `ε` (default 0.1%), slashing ratio, and a system‑wide escalation that raises `p` up to 100% if anomaly rate exceeds `ε_sys`. Default system‑wide anomaly tolerance is `ε_sys = 0.25%` (DAO‑tunable).

**Normative (Escalation Guard):** Escalation MUST increase `p` stepwise by at most ×2 per epoch and is capped by the **Verification Load Cap** from § 6.1 (on‑chain checks MUST reject steps that would exceed the cap). Escalation above 20% requires a signed anomaly report sustained over a moving 6‑epoch window and auto‑reverts after 2 clean epochs. All changes MUST be announced in‑protocol.

#### 6.3.5 Security & Liveness
Sampling renders expected value of receipt fraud negative under rational slashing. Unlike asynchronous challenges that pause reads, NilStore maintains continuous liveness; PoS² remains valid regardless of sampling outcomes.

## 7. The Deal Lifecycle

### 7.1 Quoting and Negotiation (Off-Chain)

1.  **Discovery:** Client queries Nil-Mesh for SPs near the required lattice slots.
2.  **Quoting:** SPs respond with a `Quote {Price_STOR_per_GiB_Month, Required_Collateral, QoS_Caps}`.
3.  **Selection:** Client selects the optimal bundle based on price and RTT (via QoS Oracle).

### 7.2 Deal Initiation (On-Chain - L2)

1.  **`CreateDeal`:** Client calls the function on the L2 settlement contract.
    *   It posts the Commitment Root (C_root) and the initial seal SNARK.
    *   It locks the total storage fee in $STOR escrow.
    *   A **Deal NFT** (ERC-721) is minted to the client, representing the contract.
2.  **`MinerUptake`:** The selected SP bonds the required $STOR collateral and commences service.

### 7.3 Vesting and Slashing

*   **Vesting:** The escrowed fee is released linearly to the SP each epoch, contingent on a valid PoS² submission.
*   **Consensus Parameters (Normative):**
    *   **Epoch Length (`T_epoch`)**: 86,400 s (24 h).
    *   **Proof Window (`Δ_submit`)**: 1,800 s (30 min) after epoch end — this is the *network scheduling window* for accepting PoS² proofs.
    *   **Per‑replica Work Bound (`Δ_work`)**: 60 s (baseline profile), the minimum wall‑clock work per replica referenced by the Core Spec’s § 6.2 security invariant. Implementations **MUST** ensure `t_recreate_replica ≥ 5·Δ_work` (see Nilcoin Core v2.0 § 6.2).
    *   **Block Time** (Tendermint BFT): 6 s.
*   **Slashing Rule (Normative):** Missed PoS² proofs trigger a quadratic penalty on the bonded $STOR collateral:
    `Penalty = min(0.50, 0.05 × (Consecutive_Missed_Epochs)²) × Correlation_Factor(F)`
    * `F` is the fraction of total network stake that missed the proof in the current epoch.
    * **Correlation_Factor(F) (Revised):**
      * Apply an SP‑level floor `floor_SP = 0.10` to the multiplicative penalty (each SP bears at least 10% of its computed penalty even under correlated events).
      * Above a correlation threshold `F*` (default 15%), decrease the network‑aggregate penalty linearly toward a cap (e.g., 2% of total stake per epoch), but do not reduce below `floor_SP` per SP.
      * Compute F per diversity cluster (ASN/region cell); large cartels concentrated in one cluster use that cluster’s F to prevent gaming via global correlation.
    The penalty resets upon submission of a valid proof.

### 7.4 Multi‑Stage Epoch Reconfiguration

#### 7.4.0 Objective
Ensure uninterrupted availability during committee churn by directing writes to epoch e+1 immediately, while reads remain served by epoch e until the new committee reaches readiness quorum.

#### 7.4.1 Metadata
Each DU MUST carry `epoch_written`. During handover, gateways/clients route reads by `epoch_written`; if `epoch_written < current_epoch`, they MAY continue reading from the old committee until readiness is signaled.

#### 7.4.2 Committee Readiness Signaling
New‑epoch SPs MUST signal readiness once all assigned slivers are bootstrapped. A signed message: `{epoch_id, SP_ID, slivers_bootstrapped, timestamp, sig_SP}` is posted on L1. When ≥ 2f+1 SPs signal, the DA chain emits `CommitteeReady(epoch_id)`.

Readiness Audit (Normative). Before counting an SP toward quorum, watchers MUST successfully retrieve and verify a random audit sample of that SP’s assigned slivers (sample size ≥ 1% or ≥ 1 sliver, whichever is larger). Failures cause the SP’s readiness flag to be cleared and a backoff timer `Δ_ready_backoff` (default 30 min) to apply before re‑signal.

#### 7.4.3 Routing Rules
- Writes: MUST target the current (newest) epoch.
- Reads:
  - If `epoch_written = current_epoch`, read from current.
  - If `epoch_written < current_epoch`, prefer old committee until `CommitteeReady`, then switch to new.
Gateways MUST NOT request slivers from SPs that have not signaled readiness.

#### 7.4.4 Failure Modes
- SPs failing to signal by the epoch deadline are slashed per policy.
- If quorum is not reached by `Δ_ready_timeout`, the DAO MAY trigger emergency repair bounties.
- False readiness is slashable and MAY cause temporary suspension from deal uptake.

#### 7.4.5 Governance Dials
DAO‑tunable: `Δ_ready_timeout` (default 24h), quorum (default 2f+1), slashing ratios, and the emergency bounty path.

## 8. Advanced Features: Spectral Risk Oracle (σ)

To manage systemic risk and enable sophisticated financial instruments, NilStore incorporates an on-chain volatility oracle (σ).

*   **Mechanism:** σ is calculated daily from the Laplacian eigen-drift of the storage demand graph (tracking object-to-region flows).
    `σ_t := ||Δλ₁..k(Graph_t)||₂` (tracking the k lowest eigenvalues).
*   **Application (Dynamic Collateral):** The required collateral for a deal is dynamically adjusted based on volatility:
    `Required_Collateral := Base_Collateral · f(σ)`
    Higher volatility (σ) necessitates higher slashable stake. This also informs pricing for storage ETFs and insurance pools.

## 9. Governance (NilDAO)

The network is governed by the NilDAO, utilizing stake-weighted ($STOR) voting on the L2 Settlement Layer.

### 9.1 Scope

The DAO controls economic parameters (α, slashing ratios, bounty percentages), QoS sampling dials (`p`, `ε`, `ε_sys`), multi‑stage reconfiguration thresholds (`Δ_ready_timeout`, quorum, `Δ_ready_backoff`), Durability Dial mapping (target → profile), metadata‑encoding parameters (`n_meta`, `k_meta`, `meta_scheme`), network upgrades, and the treasury.

### 9.2 Upgrade Process

*   **Standard Upgrades:** Require a proposal, a voting period, and a mandatory 72-hour execution timelock.
*   **Emergency Circuit (Hot-Patch):** A predefined **3‑of‑5** multisig (e.g., Protocol Architect, Security Lead, DAO Steward, External Auditor, Community Rep) can enact narrowly scoped emergency patches.
    *   **Sunset Clause (Normative):** Emergency patches automatically expire 14 days after activation unless ratified by a full DAO vote.

### 9.3 Freeze Points

The cryptographic specification (`spec.md@<git-sha>`) and the tokenomics parameters (`tokenomics@<git-sha>`) are hash-pinned and frozen prior to external audits and the formal DAO launch.

## 10. Roadmap and KPIs

### 10.1 Phased Rollout

1.  **MVP SDK (Rust/TS):** (2025-09)
2.  **DAO Launch & Tokenomics Freeze:** (2025-11)
3.  **Public Testnet-0 (L1 DA Chain):** (2026-01) - PoS², basic economics.
4.  **Edge-Swarm Beta (Retrieval Economy):** (2026-04) - Mobile client, $BW activated.
5.  **Rollup Bridge Mainnet (L2 Settlement):** (2026-06) - EVM L2 integration, Deal NFTs.
6.  **Mainnet-1:** (2026-09).

### 10.2 Key Performance Indicators (Targets)

| Metric                  | Target                               |
| ----------------------- | ------------------------------------ |
| Seal Time (64 GiB)      | ≤ 5 min (8-core CPU, AVX2)           |
| Epoch Proof Size (PoS²) | ≤ 1.2 kB (post-recursion)            |
| Retrieval RTT (p95)     | ≤ 400 ms (across 5 geo regions)      |
| On-chain Verify Gas (L2)| ≤ 120k Gas                           |
| Durability              | ≥ 11 nines (modeled)                 |
| Sampling FP/FN rate       | ≤ 0.5% / ≤ 0.1% (monthly audit)    |
| Handover ready time (p50) | ≤ 2 h (RS‑2D‑Hex), ≤ 6 h (RS)       |

## Annex A: Threat & Abuse Scenarios and Mitigations (Informative)

| Scenario | Attack surface | Detect / Prevent (Design) | Normative anchor(s) |
| --- | --- | --- | --- |
| **Wash‑retrieval / Self‑dealing** | SP scripts fake clients to farm $BW receipts | Challenge‑nonce + expiry in receipts; watchers or L1 verify Ed25519 off‑chain/on‑chain; PoS² only commits to **Poseidon receipt root** and byte‑sum; per‑DU/epoch service caps; /16 down‑weighting | §6.1 (Receipt schema & verification model), §6.2 (BW_root), §5.2.1 (caps) |
| **RTT Oracle collusion** | Gateways/attesters collude to post low RTT | Stake‑weighted attesters; challenge‑response tokens; ASN/region diversity; randomized assignments; slashable fraud proofs with raw transcripts | §4.2 (RTT Oracle) |
| **Commitment drift in repair** | Repaired shards bound to a *new* commitment | Repaired shards MUST open against the **original DU KZG**; reject new commitments | §3.3 (Autonomous Repair) |
| **Bridge/rollup trust** | VK swap or replay of old epoch | L2 bridge pins `vk_hash`; public inputs `{epoch_id, DA_state_root, poss2_root, bw_root}`; monotone `epoch_id`; timelocked VK upgrades | §2.4 (ZK‑Bridge) |
| **Lattice capture (ring‑cell cartel)** | SPs concentrate shards topologically | One‑shard‑per‑SP‑per‑cell; minimum cell distance; DAO can raise separation if concentration increases | §3.2 (Placement constraints), §9 (Governance) |
| **Shard withholding (availability)** | SP stores but doesn’t serve | Vesting tied to valid PoS²; $BW distribution requires receipts; slashing for missed epochs | §7.3 (Vesting/Slashing), §6 (PoS²) |
| **Beacon grinding** | Bias challenges | BLS VRF uniqueness; BATMAN threshold; on‑chain pairing check; domain separation | spec §5 (VRF), metaspec §6.2 (Challenge) |
| **Merkle truncation misuse** | Excessive path truncation weakens PoS² | Prefer higher‑arity Merkle or longer per‑sibling bytes; security bound documented | spec §4.3.1 (witness); §4.1 (arity option) |
| **Economic instability** | Excess $BW inflation via spam | Epoch cap `I_epoch_max`; α bounds; per‑DU caps; tips burned; RTT oracle weights | §5.2 (BW), §4.2 (RTT Oracle) |
| **Deal fraud / mispricing** | Non‑delivery after payment | Escrowed $STOR; vesting gated by PoS²; repair bounties | §7.2–7.3, §3.3 |

| **Receipt inflation via colluding gateways** | Faked signatures/RTT | Probabilistic sampling (§6.3); RTT oracle transcripts; slashing (`ε` threshold) | §6.3, §4.2 |
| **Writer inconsistency (2D profiles)** | Malicious sliver encodings | Encoded metadata; f+1 inconsistency proofs → mark invalid; writer slashing | §3.2.z.3 |
| **Epoch handover stalls** | New committee not ready | Multi‑stage reconfiguration; readiness signaling; emergency bounties | §7.4 |

**Reviewer note:** Items labeled *Normative* above correspond to concrete MUST/SHALL language in the main text; others are informative guidance tied to governance levers.
