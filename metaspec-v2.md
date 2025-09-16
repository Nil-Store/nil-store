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

*   **Function:** Verifying KZG openings and PoS² SNARKs efficiently via pre-compiles (avoiding expensive EVM gas costs), managing $STOR staking, and executing slashing logic. It does not run a general-purpose VM.
*   **Rationale:** The intensive cryptographic operations required for daily proof verification are best handled natively.

### 2.3 The Settlement Layer (L2)

Settlement occurs on a ZK-Rollup (using PlonK/Kimchi) bridged to a major EVM ecosystem (e.g., Ethereum L2).

*   **Function:** Manages ERC-20 representations of $STOR and $BW, mints Deal NFTs, hosts the NilDAO, and integrates with DeFi.

### 2.4 The ZK-Bridge

The L1 aggregates all epoch PoS² proofs into a single recursive SNARK and posts it to the L2 bridge contract. **Normative circuit boundary**:
1) **Public inputs**: `{epoch_id, DA_state_root, poss2_root, bw_root, validator_set_hash}`.
2) **Verification key**: `vk_hash = sha256(vk_bytes)` pinned in the L2 bridge at deployment; upgrades require DAO action and timelock.
3) **State mapping**: On accept, the bridge **atomically** updates `{poss2_root, bw_root, epoch_id}`; monotonic `epoch_id` prevents replay.
4) **Failure domains**: any mismatch in roots or non‑monotonic epoch causes a hard reject. No trusted relayers or multisigs are required because validity is enforced by the proof and pinned `vk_hash`.

### 2.5 Cryptographic Core Dependency

All layers rely on the primitives defined in the **NilStore Cryptographic Core Specification (`spec.md`)**, which establishes the security guarantees for data integrity and proof soundness.

## 3. Data Layer (NilFS)

NilFS abstracts data management complexity, automating the preparation, distribution, and maintenance of data, ensuring neither users nor Storage Providers (SPs) manage exact file representations or replication strategies manually.

### 3.1 Object Ingestion and Data Units (DUs)

1.  **Content-Defined Chunking (CDC):** Ingested objects are automatically split using CDC (e.g., Rabin fingerprinting) to maximize deduplication. Chunks are organized into a Merkle DAG (CIDv1 compatible).
2.  **Data Unit Packing:** Chunks are serialized and packed into standardized **Data Units (DUs)**. DU sizes are powers-of-two (1 MiB to 8 GiB). SPs interact only with DUs.

### 3.2 Erasure Coding and Placement

DUs are erasure‑coded using **systematic Reed‑Solomon over GF(2^8)** with **symbol size = 1 KiB** (normative). The default profile is (n=12, k=9), derived from `n = k + ⌈k / 4⌉` (1.33× redundancy). Implementations **MUST** re‑encode on churn events and publish the RS parameters in the deal metadata.

*   **Deterministic Placement (Nil-Lattice):** Shards are placed on a directed hex-ring lattice to maximize topological distance. The coordinate (r, θ) is determined by:
    `pos := Hash(CID_DU ∥ ClientSalt_32B ∥ SlotIndex) → (r, θ)`
    The `ClientSalt` ensures cross-deal uniqueness.

### 3.3 Autonomous Repair Protocol

The network autonomously maintains durability through a bounty system.

1.  **Detection:** If DU availability drops below the resilience threshold (e.g., k+1), a `RepairNeeded` event is triggered.
2.  **Execution:** Any node can reconstruct the missing shards **and MUST produce openings against the original DU KZG commitment** (no new commitment is accepted). The repair submission includes a Merkle proof to the DU’s original `C_root` plus KZG openings for the repaired shards.
3.  **Resilience Bounty:** The first node to submit proof of correct regeneration claims the bounty (default: 5% of the remaining escrowed fee for that DU).

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
    2) **Diversity**: attesters span ≥ 5 regions and ≥ 8 ASNs; assignments are epoch‑randomized.
    3) **Slashing**: equivocation or forged attestations are slashable with on‑chain fraud proofs (submit raw transcripts).
    4) **Sybil control**: weight attesters by bonded $STOR and decay weights for co‑located /24s.
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

`I_epoch = clamp( α · sqrt(Total_Bytes_Served_NetworkWide), 0, I_epoch_max )`
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

### 6.2 PoS² Binding (Storage + Bandwidth)

The PoS² SNARK proves two statements simultaneously:

1.  **Storage Verification:** Knowledge of KZG openings for randomly challenged shards at the beacon-derived evaluation point (x★).
2.  **Bandwidth Accounting:**
    *   SPs aggregate epoch receipts into a Poseidon Merkle Tree (`BW_root`), using the domain separator "NilStore-BW-v1".
    *   The SNARK verifies the consistency of `BW_root`.
    *   The SNARK asserts that the total bytes served meets a minimum threshold (`Σ bytes ≥ B_min`).

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
    `Penalty = 0.05 × (Consecutive_Missed_Epochs)²`
    The penalty resets upon submission of a valid proof.

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

The DAO controls economic parameters (α, slashing ratios, bounty percentages), network upgrades, and the treasury.

### 9.2 Upgrade Process

*   **Standard Upgrades:** Require a proposal, a voting period, and a mandatory 72-hour execution timelock.
*   **Emergency Circuit (Hot-Patch):** A predefined 2-of-2 multisig (e.g., Protocol Architect and Security Lead) can enact emergency patches.
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
