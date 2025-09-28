# NilStore Network: A Protocol for Decentralized, Verifiable, and Economically Efficient Storage

**(White Paper Draft v0.5)**

**Date:** 2025-09-15
**Status:** Working Draft (plaintext‑proofs baseline)
**Authors:** NilStore Core Team

## Abstract

NilStore is a decentralized storage network designed to provide high-throughput, verifiable data storage with significantly reduced operational overhead. It leverages a novel consensus mechanism, Proof-of-Spacetime-Squared (PoS²), which merges storage verification with bandwidth accounting in a single, succinct proof. By utilizing CPU-efficient sealing based on KZG commitments and a topological data placement strategy (Nil-Lattice), NilStore drastically lowers the hardware barrier to entry, enabling participation from edge devices to data centers. This paper details the system architecture, the NilFS data abstraction layer, the Nil-Mesh routing protocol, the dual-token ($STOR/$BW) economic model, and the hybrid L1/L2 settlement architecture designed for EVM compatibility and robust governance.

## 1. Introduction

### 1.1 Motivation

While existing decentralized storage protocols have demonstrated the viability of incentive-driven storage, they often rely on computationally intensive Proof-of-Replication (PoRep) stacks requiring significant GPU investment. This centralizes the network around large-scale operators and increases the total cost per byte.

NilStore retains strong cryptographic guarantees while reducing the "sealing" process to minutes on standard CPUs. This democratization of access increases network resilience through geographic distribution and enables a more efficient storage marketplace.

### 1.2 Key Innovations

*   **Plaintext possession as first‑class:** Storage providers keep the **cleartext** bytes of assigned Data Units (DUs) on disk and prove it regularly with near‑certain full‑coverage over time.
*   **PoUD + PoDE:** **PoUD** (KZG‑based Provable Data Possession over DU cleartext) + **PoDE** (timed window derivations) are the **normative** per‑epoch proofs.
*   **CPU‑only sealing (research scaffold):** A sealed PoS² path exists **only** as a **research‑only supplement** (**`rfcs/PoS2L_scaffold_v1.md`**), intended for phased rollout experiments and incident‑response modeling. It is **non‑normative** and **disabled** in all profiles; plaintext is the only supported mode in Core.
*   **Nil-Mesh Routing:** Heisenberg-lifted K-shortest paths for optimized latency and Sybil resistance.
*   **Dual-Token Economy:** Decoupling long-term capacity commitment ($STOR) from immediate utility ($BW).
*   **Hybrid Settlement:** A specialized L1 for efficient proof verification bridged via ZK-Rollup to an EVM L2 for liquidity and composability.

## 2. System Architecture
Manifest & crypto policy follow Core Appendix A (Root CID, DU CID, HPKE FMK wraps, HKDF‑derived CEKs, AES‑GCM per DU).


NilStore employs a hybrid architecture that decouples Data Availability (DA) consensus from economic settlement, optimizing for both cryptographic efficiency and ecosystem composability.

### 2.1 Architectural Layers

1.  **Data Layer (NilFS):** Handles object ingestion, erasure coding, and placement.
2.  **Network Layer (Nil-Mesh):** Manages peer discovery, routing, and QoS measurement.
3.  **Consensus Layer (DA Chain - L1):** Verifies **PoUD (KZG multi‑open) + PoDE timing attestations**, manages stake, and mints rewards.
4.  **Settlement Layer (L2 Rollup):** Handles economic transactions, liquidity, and governance.

### 2.2 The DA Chain (L1)

The Data Availability Chain is a minimal L1 (built using Cosmos-SDK/Tendermint BFT) optimized for NilStore's cryptographic operations.

*   **Function:** Verifying **KZG openings (multi‑open)** for PoUD and enforcing PoDE timing bounds via watcher digests, managing $STOR staking, and executing slashing logic. It does not run a general‑purpose VM.
*   **Required pre‑compiles (normative):** (a) **BLAKE2s‑256**, (b) **Poseidon** (for Merkle paths), (c) **KZG** (G1/G2 ops; multi‑open), and (d) **VDF Verification**. Chains lacking these MUST expose equivalent syscalls in the DA module.
*   **Rationale:** The intensive cryptographic operations required for daily proof verification are best handled natively.

### 2.3 The Settlement Layer (L2)

Settlement occurs on a ZK-Rollup (using PlonK/Kimchi) bridged to a major EVM ecosystem (e.g., Ethereum L2).

*   **Function:** Manages ERC-20 representations of $STOR and $BW, mints Deal NFTs, hosts the NilDAO, and integrates with DeFi.

### 2.4 The ZK-Bridge

The L1 aggregates epoch verification results into a single proof/digest and posts it to the L2 bridge contract. **Normative circuit boundary**:
1) **Public inputs**: `{epoch_id, DA_state_root, poud_root, bw_root, validator_set_hash}`.
2) **Verification key**: `vk_hash = sha256(vk_bytes)` pinned in the L2 bridge at deployment; upgrades require DAO action and timelock. In addition, an Emergency Circuit MAY perform an expedited **VK‑only** upgrade with a shorter timelock under §9.2, restricted to a pre‑published whitelist (hash‑pinned on L1). **No other code paths or parameters may change under Emergency mode.** During “yellow‑flag”, the bridge **MUST**:
  • continue updating `{epoch_id, poud_root, bw_root}`;
  • **disable** all fund‑moving paths: vesting payouts, token transfers/mints/burns, withdrawals/deposits, deal‑escrow releases, slashing executions, new deal creation (`CreateDeal`), and deal uptake (`MinerUptake`);
  • freeze parameter change entrypoints and all governance actions (proposal submission and voting), **EXCEPT** for the DAO vote required to ratify the emergency patch (§9.2);
  • halt the economic processing (reward minting and slashing execution) derived from `poud_root` and `bw_root` updates;
  • require an independent auditor attestation **and** a hash of the patched verifier bytecode;  
and auto‑revert to normal after sunset unless ratified by DAO.
3) **State mapping**: On accept, the bridge **atomically** updates `{poud_root, bw_root, epoch_id}`; monotonic `epoch_id` prevents replay.
4) **Failure domains**: any mismatch in roots or non‑monotonic epoch initiates a **Grace Period** (DAO-tunable, default 24h). If the mismatch persists after the Grace Period, the bridge halts (hard reject). No trusted relayers or multisigs are required because validity is enforced by the proof and pinned `vk_hash`.
5) **Proof Generation (Normative):** The ZK proof MUST be generated by a decentralized prover network or a rotating committee selected from the L1 validator set, with slashing penalties for failure to submit valid proofs within the epoch window.

### 2.5 Cryptographic Core Dependency

All layers rely on the primitives defined in the **NilStore Cryptographic Core Specification (`spec.md`)**, which establishes the security guarantees for data integrity and proof soundness.

## 3. Data Layer (NilFS)

NilFS abstracts data management complexity, automating the preparation, distribution, and maintenance of data, ensuring neither users nor Storage Providers (SPs) manage exact file representations or replication strategies manually.

### 3.1 Object Ingestion and Data Units (DUs)

1.  **Content-Defined Chunking (CDC):** Ingested objects are automatically split using CDC (e.g., Rabin fingerprinting) to maximize deduplication. Chunks are organized into a Merkle DAG (CIDv1 compatible).
2.  **Data Unit Packing:** Chunks are serialized and packed into standardized **Data Units (DUs)**. DU sizes are powers‑of‑two (1 MiB to 8 GiB). SPs interact only with DUs.

#### 3.1.1 Upload Walkthrough (Informative)

This walkthrough illustrates what happens when a client uploads an object **F** to NilStore:

1) **Chunk & DAG (CDC).** The client runs content‑defined chunking (e.g., Rabin) over **F**, producing a Merkle‑DAG (CIDv1‑compatible).
2) **Pack into DUs.** Chunks are serialized into one or more **Data Units (DUs)** (power‑of‑two size between 1 MiB and 8 GiB). Each DU is self‑contained.
3) **Commit & deal intent.** The client computes a DU commitment (`C_root`) and prepares `CreateDeal` parameters (price, term, redundancy, QoS).
4) **Erasure coding.** Each DU is encoded with Reed–Solomon **RS(n,k)** (default **(12,9)**), yielding **n** shards (k data + n−k parity).
5) **Deterministic placement.** For every shard `j`, compute a Nil‑Lattice **ring‑cell** target via
   `pos := Hash(CID_DU ∥ ClientSalt_32B ∥ j) → (r,θ)` and enforce placement constraints (one shard per SP per cell; cross‑cell distance threshold).
6) **Deal creation (L2).** The client calls **`CreateDeal`** on L2, posting `C_root`, locking $STOR escrow, and minting a **Deal NFT**.
7) **Miner uptake (L2+L1).** Selected SPs bond $STOR, fetch their assigned shards, and store the **plaintext DU bytes** locally. The client‑posted **DU KZG commitment (`C_root`)** binds content for all future proofs.
8) **Epoch service.** During each epoch, SPs (a) serve retrievals; clients sign **Ed25519 receipts**; SPs aggregate receipts into a Poseidon Merkle (`BW_root`), and (b) post **PoUD + PoDE** storage proofs against the original `C_root`.
9) **Settlement & rewards.** L1 verifies KZG openings and enforces PoDE timing and posts a compressed digest to L2 (**ZK‑Bridge**). L2 updates `{poud_root, bw_root, epoch_id}` and releases vested fees / $BW rewards per distribution rules.
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

On‑chain action (DoS‑safe): Any party MAY call `MarkInconsistent(C_root, evidence_root)` on L1 with a refundable **bond** ≥ `B_min` (DAO‑tunable). The contract verifies at most `K_max` symbols/openings per call (cost‑capped). If ≥ f+1 proofs from distinct SPs verify, the DU is marked invalid, excluded from PoS²/BW accounting, the writer’s escrowed $STOR is slashed per §7.3, and the bond is refunded; otherwise the bond is burned. Repeat submissions for the same `C_root` within a cool‑off window are rejected.

#### 3.2.z.5 Lattice Coupling
Resolved profiles MUST respect §3.2.y placement rules for 2D cases, and the standard ring‑cell separation for RS.

#### 3.2.z.6 Governance
NilDAO maintains the mapping table (durability target → profile), caps allowed ranges, and sets cost multipliers per profile.

### 3.3 Autonomous Repair Protocol

The network autonomously maintains durability through a bounty system.

1.  **Detection:** If DU availability drops below the resilience threshold (e.g., k+1), a `RepairNeeded` event is triggered.
2.  **Execution:** Any node can reconstruct the missing shards **and MUST produce openings against the original DU KZG commitment** (no new commitment is accepted). The repair submission includes a Merkle proof to the DU’s original `C_root` plus KZG openings for the repaired shards.
3.  **Resilience Bounty:** The first node to submit proof of correct regeneration claims the bounty (default: 5% of the remaining escrowed fee for that DU).

**Normative (Anti‑withholding):** When a repair for shard `j` is accepted, the SP originally assigned `j` incurs an immediate penalty on their bonded $STOR strictly greater than the repair bounty (Default: Penalty = 1.5 × Bounty), in addition to an automatic demerit. Repeated events within a sliding window escalate to further slashing unless the SP supplies signed RTT‑oracle transcripts proving inclusion in a whitelisted incident.
**Normative (Collocation Filter):** An identity is disqualified from claiming bounty on any shard it was previously assigned for `Δ_repair_cooldown` epochs (DAO‑tunable). Furthermore, any identity within the same /24 IPv4 (or /48 IPv6) OR the same ASN **for the same window** is likewise disqualified. Cooldown MUST be ≥ 2× mean repair time.
    **Normative (Dynamic Bounty):** The bounty MUST be dynamically adjusted based on the urgency of the repair, the cost of reconstruction, and network conditions (DAO‑tunable parameters).

## 4. Network Layer (Nil-Mesh)

Nil-Mesh is the network overlay optimized for low-latency, topologically aware routing.

### 4.1 Heisenberg-Lifted Routing

Nil-Mesh utilizes the geometric properties of the Nil-Lattice for efficient pathfinding.

*   **Secure Identity Binding (Normative):** Peer IDs (NodeIDs) are securely bound to lattice coordinates (r, θ) through a costly registration process. To register or move a coordinate, an SP MUST:
    (1) Bond a minimum amount of $STOR (Stake_Min_Cell), specific to the target Ring Cell.
    (2) Compute a Verifiable Delay Function (VDF) proof anchored to their NodeID and the target coordinate: `Proof_Bind = VDF(NodeID, r, θ, difficulty)`.
    This prevents rapid movement across the lattice and ensures that capturing a Ring Cell requires significant capital ($STOR) and time (VDF computation).
*   **Mechanism:** Peer IDs are mapped ("lifted") to elements in a 2-step nilpotent Lie group (Heisenberg-like structure) corresponding to their lattice coordinates.
*   **Pathfinding:** K-shortest paths (K=3) are computed in this covering space and projected back to the physical network. This offers superior latency performance compared to standard DHTs and increases Sybil resistance by requiring attackers to control entire topological regions ("Ring Cells").
**Normative (Capture cost):** DAO MUST publish and periodically update the `Stake_Min_Cell` and VDF `difficulty` parameters. These parameters MUST be raised automatically if empirical concentration increases.

### 4.2 RTT Attestation and QoS Oracle

Verifiable Quality of Service (QoS) is crucial for performance and security.

*   **Attestation:** Nodes continuously monitor and sign Round-Trip Time (RTT) attestations with peers.
*   **On‑Chain Oracle:** A **stake‑weighted attester set** posts RTT digests (Poseidon Merkle roots) to the DA chain. **Normative**:
    1) **Challenge‑response**: clients issue random tokens; SPs must echo tokens within `T_max`; vantage nodes verify end‑to‑end.
    2) **VDF Enforcement (Mandatory Baseline + Conditional Escalation):** Every attestation MUST include a short-delay VDF proof (Baseline VDF). If the anomaly rate exceeds `ε_sys` for ≥ 3 consecutive epochs, the VDF delay is increased (Conditional Escalation) until the anomaly rate drops for 2 consecutive clean epochs. Total VDF cost per probe is capped by the **Verification Load Cap** (§ 6.1). Protocol MUST publish the current VDF parameters (delay, modulus) on‑chain per epoch.
    3) **Diversity & rotation**: The attester set MUST achieve a minimum diversity score (e.g., Shannon index over ASN/Region distribution) defined by governance (default: score equivalent to uniform distribution over ≥ 5 regions and ≥ 8 ASNs). Assignments are epoch‑randomized and **committed on‑chain** (rotation proof) before measurements begin.
    4) **Slashing**: equivocation or forged attestations are slashable with on‑chain fraud proofs (submit raw transcripts).
    5) **Sybil control**: weight attesters using **quadratic weighting** of bonded $STOR (weight ∝ √STOR) to reduce the influence of large stakeholders. Apply decay weights for co‑located /24s and ASNs.
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

`I_epoch = clamp( α · sqrt(Total_Bytes_Served_NetworkWide), 0, I_epoch_max )`  // **Normative bounds:** (a) Per‑epoch cap: `I_epoch_max ≤ 0.10%` of circulating $BW (DAO‑tunable within [0.02%, 0.10%]); (b) Rolling cap: over any 30‑day window, Σ I_epoch ≤ `I_30d_max` (DAO‑tunable corridor, default 6%); (c) Attack‑traffic filter: bytes counted toward inflation MUST be discounted by an abuse‑score derived from §6.3 sampling failures and RTT‑oracle anomalies;  

 (c.1) **Normative (Abuse Score Formula):** The abuse score $S_{abuse} \in [0, 1]$ for an SP is calculated as:
 $S_{abuse} = clamp( w_1 \cdot (F_{sample} - \epsilon) + w_2 \cdot A_{rtt} + w_3 \cdot C_{topology}, 0, 1 )$
 Where $F_{sample}$ is the fraction of failed receipt samples (§6.3), $A_{rtt}$ is the fraction of anomalous RTT attestations (§4.2), $C_{topology}$ is the topological concentration score of the clients served (measuring centralization in the Nil-Mesh graph), $\epsilon$ is the tolerance threshold, and $w_1, w_2, w_3$ are governance-tunable weights (default w1=5, w2=1, w3=2). Discounted bytes = $Bytes_\!Served \cdot (1 - S_{abuse})$.

 (d) **Per‑counterparty caps (Normative):** apply caps per (Client,SP) pair and per DU to bound bilateral wash‑retrieval. Default caps (DAO-tunable): $Cap_{Pair\_Epoch} \leq 1\,TiB$; $Cap_{DU\_Epoch} \leq 10\,GiB$.
 (e) **Honeypot zeroing:** any receipt associated with a Honeypot DU (§ 6.3.1, 6.3.4) that fails auditor checks contributes **zero** to inflation and triggers stake‑weighted escalation for the SP.
where:
- `α ∈ [α_min, α_max]` (DAO‑tunable);
- `I_epoch_max` caps epoch inflation (DAO‑tunable);
- per‑DU and per‑Miner **service caps** apply to the counted bytes to mitigate wash‑trading.

*   α (Alpha) is a governance-tunable constant scaling the inflation rate.

#### 5.2.2 Distribution

Minted $BW is distributed pro-rata to SPs based on the volume of verified retrieval receipts submitted in their PoS² proofs.

#### 5.2.3 Burn Mechanism (Tipping)

**Normative (Mandatory Base Burn):** A fixed fraction $\beta$ (DAO-tunable, default 5%) of the $BW$ reward generated by every verified receipt MUST be burned upon settlement. This introduces a baseline cost for all retrievals, increasing the absolute cost of wash-retrieval.

Users can optionally "tip" for priority retrieval by including a `tip_bw` amount in the receipt, which is burned upon settlement.

*   **Incentive Alignment:** Distribution shares are calculated *before* the burn. By capturing tipped traffic, a miner increases their effective share of the total inflation pie relative to others and improves their QoS reputation, creating competition for prioritized traffic.

### 5.3 Stablecoin UX

While the protocol strictly uses $BW for tips, client software can provide a seamless stablecoin (e.g., USDC) experience by executing a DEX swap (USDC -> $BW) client-side before signing the receipt.

## 6. Consensus and Verification (Storage + Bandwidth)

The economic model is enforced cryptographically through the PoS² consensus mechanism on the L1 DA Chain.

### 6.0a  Proof Mode — **Plaintext only (research phase)**

Core supports **one** normative proof mode: **plaintext** — **PoUD** (KZG multi‑open on DU cleartext) + **PoDE** (timed derivations).  
The sealed **PoS²‑L** path is **not part of Core**; it is archived as a **research‑only supplement** in **`rfcs/PoS2L_scaffold_v1.md`** and is **disabled** in all profiles. Any experimental activation MUST follow the governance and sunset requirements defined in that supplement, and does **not** alter the Core’s plaintext primacy.

### 6.0b  PoUD (Proof of Useful Data) – Plaintext Mode (normative)

For each epoch and each assigned DU sliver interval:

1) Content correctness: The SP MUST provide one or more **KZG multi‑open** proofs at verifier‑chosen 1 KiB symbol indices proving membership in the DU commitment `C_root` recorded at deal creation. When multiple indices are scheduled for the same DU in the epoch, SPs SHOULD batch using multi‑open to minimize calldata.

2) Timed derivation (PoDE): Let `W = 8 MiB` (governance‑tunable). The SP MUST compute `deriv = Derive(clear_bytes[interval], beacon_salt, row_id)` within the proof window; `deriv` is fixed by the micro‑seal profile (Core § 3.3.1) restricted to the bytes of the interval (no cross‑window state). The proof includes `H(deriv)` and the clear bytes needed for recomputation.

3) Concurrency & volume: The prover MUST satisfy at least `R` parallel PoDE sub‑challenges per proof window, each targeting a distinct DU interval (default `R ≥ 16`; DAO‑tunable). The aggregate verified bytes per window MUST be ≥ `B_min` (default `B_min ≥ 128 MiB`, DAO‑tunable). B_min counts only bytes that are both (a) KZG‑opened and (b) successfully derived under PoDE.

### 6.0c  PDP‑PLUS Coverage SLO (normative)
Define CoverageTargetDays (default 365). The governance scheduler MUST choose per‑epoch index sets (challenge rate $q/M$) so that for every active DU:
  • the expected fraction of uncovered bytes $(1-q/M)^T$ after $T$=CoverageTargetDays is ≤ $2^{-18}$ (implying $q/M \approx 3.4\%$ if $T=365$); and
  • the scheduler is commit‑then‑sample: indices for epoch t are pseudorandomly derived from the epoch beacon and a DU‑local salt and are not known to SPs before the BW_commit deadline of epoch t−1.
Chains MUST publish (and auditors MUST reproduce) the per‑epoch index‑set transcript. Failure to meet the SLO MUST trigger an automatic increase of B_min (×1.25 per epoch, capped by the Verification Load Cap) until the SLO is restored.

4) Deadline: The derivations MUST complete before `Δ_submit` (§ 7.3). RTT‑Oracle transcripts (§ 4.2) are included when a remote verifier is used.

On‑chain: L1 verifies KZG openings (precompile § 2.2) and checks `B_min` & `R` counters. Watchers enforce timing via RTT‑Oracle and publish pass/fail digests; repeated failures escalate slashing per § 6.3.3.

### 6.1 Retrieval Receipts

To account for bandwidth, clients sign receipts upon successful retrieval.

*   **Receipt Schema (Normative):**
    `Receipt := { CID_DU, Bytes, ChallengeNonce, ExpiresAt, Tip_BW, Miner_ID, Client_Pubkey, Sig_Ed25519 [, GatewaySig?] }`
    - `ChallengeNonce` is issued per‑session by the SP/gateway and bound to the DU slice; `ExpiresAt` prevents replay.
    - **Verification model:** Ed25519 signatures are verified **off‑chain by watchers and/or on the DA chain**; PoS² only commits to a **Poseidon Merkle root** of receipts and proves byte‑sum consistency. In‑circuit Ed25519 verification is **not required**.  
      **Commit requirement:** For epoch `t`, SPs MUST have posted `BW_commit := Blake2s‑256(BW_root)` by the last block of epoch `t−1` (see § 6.3.1). Receipts not covered by `BW_commit` are ineligible.  
      **Penalty:** Failure to post `BW_commit` for epoch `t` sets counted bytes to zero for `t` and forfeits all $BW rewards for `t`.  
      **Normative anchor:** At least **2% of receipts by byte‑volume per epoch** MUST be verified on the DA chain (randomly sampled via § 6.3) and escalate automatically under anomaly (§ 6.3.4).  
      **Normative (Verification Load Cap):** The total on‑chain verification load MUST be capped (DAO‑tunable) to prevent DoS via forced escalation.
      **Normative (VLC Prioritization and Security Floors):** Governance MUST define Security Floors for critical parameters ($p_{kzg\_floor}$, $R_{floor}$, $B_{min\_floor}$). The system MUST NOT automatically reduce these parameters below their floors.
      **Normative (Economic Circuit Breaker):** If the Verification Load Cap (VLC) is reached during a security escalation (e.g., increase in $p$), and parameters are already at their floors, the system MUST activate an Economic Circuit Breaker instead of suppressing the escalation:
      1. **Throttle $BW$ Minting:** Apply a global discount factor to the counted bytes for $BW$ inflation for the epoch, proportional to the excess load.
      2. **Prioritize High-Risk Receipts:** The sampling mechanism MUST prioritize receipts associated with SPs exhibiting high abuse scores (§5.2.1.c.1).
      This ensures that security auditing proceeds unimpeded during an attack, while imposing an economic cost on the network instead of compromising storage integrity.

### 6.2 Storage Proof Binding (PoUD + PoDE)

For each SP and each assigned DU interval per epoch the DA chain enforces:

1. **PoUD (KZG‑PDP on cleartext):** The SP submits one or more **KZG openings** at verifier‑chosen **1 KiB symbol indices** proving membership in the **original** DU commitment `C_root` recorded at deal creation. Multi‑open is RECOMMENDED; indices are derived from the epoch beacon.
2. **PoDE (timed derivation):** For each challenged **W = 8 MiB** window, compute a salted local transform `Derive(clear_window, beacon_salt, row_id)` **within the proof window** and submit `H(deriv)` with the minimal bytes to recompute. **`R ≥ 16`** sub‑challenges/window and **Σ verified bytes ≥ B_min = 128 MiB** per epoch (defaults; DAO‑tunable).
3. **Deadlines:** Proofs must arrive within `Δ_submit` after epoch end. Timing may be attested by RTT‑oracle transcripts for remote verification.

**On‑chain checks:** L1 verifies KZG openings via pre‑compiles and enforces `R` and `B_min`; watchers produce timing digests for PoDE. The rollup compresses per‑SP results into `poud_root` for the bridge.

### 6.3 Probabilistic Retrieval Sampling (QoS Auditing)

#### 6.3.0 Objective
Strengthen retrieval QoS without suspending reads by sampling and verifying a governance‑tunable fraction of receipts each epoch. This mechanism is additive to PoS² and does not alter the proof object.

#### 6.3.1 Sampling Set Derivation
0) **Commit‑then‑sample (Normative):** Each SP MUST post `BW_commit := Blake2s‑256(BW_root)` no later than the last block of epoch `t−1`.
1) At epoch boundary `t`, derive `seed_t := Blake2s-256("NilStore-Sample" ‖ beacon_t ‖ epoch_id)`, where `beacon_t` is the Nil‑VRF epoch beacon.
2) Expand `seed_t` into a PRF stream and select a fraction `p` of receipts **from the set committed by `BW_commit`** for verification (`0.5% ≤ p ≤ 10%`, default ≥ 5%). Receipts not committed in `t−1` MUST NOT be counted for `t`.
3) The sample MUST be unpredictable to SPs prior to epoch end and sized so that expected coverage ≥ 1 receipt per active SP. Auditor assignment SHOULD be stake‑weighted and region/ASN‑diverse (per §4.2) to avoid correlated blind‑spots and to bound per‑auditor load.
4) **Honeypot DUs:** MUST be **profile‑indistinguishable** from ordinary DUs: sizes drawn from the same power‑of‑two distribution; RS profiles sampled from governance‑approved mixes; Nil‑Lattice slots assigned via the standard hash; and metadata randomized within normal bounds. Any retrieval receipt for a Honeypot DU is automatically selected for 100% verification.
   **Normative (Indistinguishability):** Honeypot DUs MUST be created and funded pseudonymously (e.g., using zero-knowledge proofs of funding) to prevent identification via on-chain analysis. Retrieval patterns MUST mimic organic traffic distributions.

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
NilDAO MAY tune: sampling fraction `p`, tolerance `ε` (default 0.1%), slashing ratio, and escalation behavior.

**Normative (Escalation Guard):** Escalation MUST be triggered both system-wide and per-SP. Per-SP escalation MUST immediately increase the SP's individual sampling rate $p_{sp}$ if their failure rate significantly exceeds $\epsilon$. System-wide escalation MUST increase $p$ stepwise by at most ×2 per epoch. However, if the anomaly rate exceeds $5 \times \epsilon_{sys}$, $p$ MUST immediately escalate to the maximum allowed by the **Verification Load Cap** (§ 6.1). Escalation auto‑reverts after 2 clean epochs. All changes MUST be announced in‑protocol.

Additional dial (content‑audited receipts):
- `p_kzg ∈ [0,1]` — Fraction of sampled receipts that MUST include one or more KZG openings at 1 KiB RS symbol boundaries corresponding to claimed bytes. Default 0.05. In plaintext mode, `p_kzg` MUST be ≥ 0.05 unless disabled by DAO vote under the Verification Load Cap. Honeypot DUs MUST use `p_kzg = 1.0`. On‑chain verification uses KZG precompiles when available; otherwise, auditors verify off‑chain with fraud‑proof slashing. Adjust `p_kzg` under the **Verification Load Cap** (§ 6.1).


#### 6.3.5 Security & Liveness
Sampling renders expected value of receipt fraud negative under rational slashing. Unlike asynchronous challenges that pause reads, NilStore maintains continuous liveness; PoS² remains valid regardless of sampling outcomes.


### 6.4  Bandwidth‑Driven Redundancy (Normative)

NilStore aligns replica count and provider selection with observed demand and measured provider capability:

1) **Heat Index.** For DU `d` at epoch `e`, define `H_e(d)` as an EMA over verifiable retrieval receipts (served bytes; p95 latency), half‑life `τ`. Watchers aggregate via BATMAN.

2) **Target Redundancy & Lanes.** Redundancy `r_e(d) = clamp(r_min, r_max, ceil(H_e / μ_target))`. The number of parallel client lanes `m(req) = clamp(1, m_max, ceil(B_req / μ_conn))`.

3) **Placement (WRP).** The per‑DU provider set is chosen by **weighted rendezvous hashing** on `(du_id, epoch)`, with weight `w_i = f(cap_i, conc_i, rel_i, price_i, geo_fit)` derived from **Provider Capability Vectors (PCV)** and watcher probes. Clients stripe requests across the top‑score providers (m lanes), failing over to the next candidates if SLA is not met.

4) **Hot replicas.** When `H_e(d)` crosses tier `T_hot(k)`, a VRF committee assigns `Δr` short‑TTL replicas to additional providers chosen by the same WRP. Providers post a `bond_bw`; rewards per verified byte follow `R_hot(H)`. Replicas expire when `H_e(d) < T_cool(k)` (hysteresis).

5) **Receipts.** Per‑chunk receipts commit to `{du_id, chunk_id, bytes, t_start, t_end, rtt, p99, client_nonce, provider_id}` with provider signatures. Receipts aggregate into `BW_root` per provider per epoch. Rewards apply only if quality factor `q ≥ q_floor`.


## 7. The Deal Lifecycle


### 7.y  Bandwidth Receipts & BW_root (Normative)

- **Receipt schema:** `{ du_id, chunk_id, bytes, t_start, t_end, rtt_ms, p99_ms, client_nonce, provider_id, payer_id?, sig_provider }` hashed under `"BW-RECEIPT-V1"`.
- **Aggregation:** leaves → Poseidon Merkle → `"BW-ROOT-V1"`; providers submit `(provider_id, epoch, BW_root, served_bytes, med_latency, agg_sig)`.
- **Eligibility:** payer‑funded (uploader or sponsor); fraud screens (mono‑ASN discount); watcher sampling.


### 7.x  L2 Registries & Calls (New)

- `register_pcv(provider_id, PCV, proof_bundle)` — Provider Capability Vector registry; watcher probes attached and aggregated via BATMAN.
- `submit_bw_root(provider_id, epoch, BW_root, served_bytes, med_latency, agg_sig)` — Aggregation of per‑chunk receipts into a per‑epoch bandwidth root.
- `spawn_hot_replicas(du_id, epoch, Δr, TTL)` — VRF‑mediated hot‑replica assignment; requires capacity bonds and enforces TTL/hysteresis.



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
3.  **`SealAttest`:** Before any PoS² proofs for this deal are counted toward vesting,
    the SP MUST post on L1/L2 an attestation tuple
    `{sector_id, h_row_root, delta_head_root, origin_root, deal_id}`
    where `origin_root` commits to per‑row `{du_id, sliver_index, symbol_range, C_root}` (see Core § 3.7.1).
    Vesting and $BW distribution for this sector are **disabled** unless a matching `SealAttest` exists.

### 7.3 Vesting and Slashing

*   **Vesting:** The escrowed fee is released linearly to the SP each epoch, contingent on a valid **PoUD + PoDE** submission.
*   **Consensus Parameters (Normative):**
    *   **Epoch Length (`T_epoch`)**: 86,400 s (24 h).
    *   **Proof Window (`Δ_submit`)**: 1,800 s (30 min) after epoch end — this is the *network scheduling window* for accepting PoS² proofs.
    *   **Per‑replica Work Bound (`Δ_work`)**: 60 s (baseline profile), the minimum wall‑clock work per replica referenced by the Core Spec’s § 6.2 security invariant. Implementations **MUST** ensure `t_recreate_replica ≥ 5·Δ_work` (see Nilcoin Core v2.0 § 6.2).
    *   **Block Time** (Tendermint BFT): 6 s.
*   **Slashing Rule (Normative):** Missed **PoUD** proofs (plaintext mode) or missed **PoS²** proofs (scaffold mode) trigger a quadratic penalty on the bonded $STOR collateral:
    `Penalty = min(0.50, 0.05 × (Consecutive_Missed_Epochs)²) × Correlation_Factor(F)`
* `F` is computed **per diversity cluster** (ASN×region cell) and globally.
* **Correlation_Factor(F) (Revised):**
      * Apply an SP‑level floor `floor_SP = 0.10`.
      * Allocate the global correlation discount to SPs via a **Shapley‑like share** of their clusters’ contribution to the global miss set, so that dispersion across clusters does not trivially reduce aggregate penalties.
      * For `F_global > F*` (default 15%), cap network‑aggregate burn at 2%/epoch while preserving `floor_SP`.
      * Collocated identities (same /24 or ASN) are merged for F‑computation to prevent sybil dilution.
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
    **Normative (Oracle Input Hardening):** The input `Graph_t` MUST be filtered to exclude manipulative patterns, such as rapid creation/deletion of deals by the same entity (Sybil filtering) and traffic associated with high abuse scores (§5.2.1.c.1).
*   **Application (Dynamic Collateral):** The required collateral for a deal is dynamically adjusted based on volatility:
    `Required_Collateral := Base_Collateral · f(σ, σ_price)`
    where $\sigma_{price}$ is the realized volatility of the $STOR token price.
    Higher volatility (σ) necessitates higher slashable stake. This also informs pricing for storage ETFs and insurance pools.
    **Normative (Oracle Dampening and Management):** The function $f(\sigma, \sigma_{price})$ MUST incorporate a dampening mechanism (e.g., a 30-day Exponential Moving Average).
    **Normative (Circuit Breakers and Rate Limits):** The rate of change in Required_Collateral MUST be capped per epoch (e.g., max 10% increase) to prevent sudden shocks.
    **Normative (Dynamic Grace Period):** A mechanism for collateral top-ups MUST be defined. The grace period before liquidation/slashing (default 72 hours) MUST be dynamically extended (up to 7 days) if $\sigma_{price}$ exceeds a high volatility threshold (DAO-tunable).

## 9. Governance (NilDAO)

### 9.x  Bandwidth Quota, Auto‑Top‑Up & Sponsors (Normative)

The protocol uses a **hybrid** bandwidth model: each file has an **included quota** (budget reserved per epoch from uploader deposits in $STOR, converted to $BW on verified receipts). On exhaustion, the file enters a **grace tier** with reduced placement weight until **auto‑top‑up** or **sponsor** budgets restore full weight. APIs: `set_quota`, `set_auto_top_up`, `sponsor`. Governance sets `w_grace`, roll‑over caps, region multipliers, price bands, sponsor caps, and ASN/geo abuse discounts.



The network is governed by the NilDAO, utilizing stake-weighted ($STOR) voting on the L2 Settlement Layer.

### 9.1 Scope

The DAO controls economic parameters (α, slashing ratios, bounty percentages), QoS sampling dials (`p`, `ε`, `ε_sys`), multi‑stage reconfiguration thresholds (`Δ_ready_timeout`, quorum, `Δ_ready_backoff`), Durability Dial mapping (target → profile), metadata‑encoding parameters (`n_meta`, `k_meta`, `meta_scheme`), network upgrades, and the treasury.
It also controls content‑binding dials across Core and Metaspec: PoS² linking fraction `p_link` (Core § 4.2.1), PoDE fraction `p_derive` (Core § 4.2.2), the micro‑seal profile (`micro_seal`, Core § 3.4.3), and receipt‑level content‑check fraction `p_kzg` (this § 6.3.4).
Additional PoDE/PoUD pressure dials (plaintext/scaffold modes):
- `R` — Minimum parallel PoDE sub‑challenges per proof window (default `R ≥ 16` for plaintext mode; `R ≥ 8` for scaffold mode).
- `B_min` — Minimum verified bytes per proof window (default `≥ 128 MiB` for plaintext mode; `≥ 64 MiB` for scaffold mode).
- Escalation: If sampled fail rate `ε_sys` exceeds the threshold for 2 epochs, increase `R` and/or `B_min` stepwise (×1.5 max per epoch) subject to the Verification Load Cap (§ 6.1).

### 9.2 Upgrade Process

*   **Standard Upgrades:** Require a proposal, a voting period, and a mandatory 72-hour execution timelock.
*   **Emergency Circuit (Hot-Patch):** A predefined **5‑of‑9** threshold **with role diversity** can enact **VK‑only** emergency patches (see § 2.4). Keys MUST be HSM/air‑gapped.
    *   **Key Allocation and Independence (Normative):** The 9 keys MUST be strictly allocated as: Core Team (3), Independent Security Auditor (3), Community/Validator Rep (3). The Auditor role MUST be filled by entities with no financial or control relationship with the Core Team, ratified by DAO vote annually. The 5-of-9 threshold MUST include at least one valid signature from each of these three groups.
    *   **Sunset Clause (Normative):** Emergency patches automatically expire 14 days after activation unless ratified by a full DAO vote.
    *   **Sunset Integrity (Normative):** The emergency patch mechanism MUST NOT be capable of modifying the Sunset Clause duration or the ratification requirement. If an emergency patch is ratified by a full DAO vote during the 14-day window, the automatic expiration MUST be disabled. The ratified patch remains active until it is superseded by the standard upgrade cycle.

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
| **Bridge/rollup trust** | VK swap or replay of old epoch | L2 bridge pins `vk_hash`; public inputs `{epoch_id, DA_state_root, poud_root, bw_root}`; monotone `epoch_id`; timelocked VK upgrades | §2.4 (ZK‑Bridge) |
| **Lattice capture (ring‑cell cartel)** | SPs concentrate shards topologically | One‑shard‑per‑SP‑per‑cell; minimum cell distance; DAO can raise separation if concentration increases | §3.2 (Placement constraints), §9 (Governance) |
| **Shard withholding (availability)** | SP stores but doesn’t serve | Vesting tied to valid PoUD + PoDE; $BW distribution requires receipts; slashing for missed epochs | §7.3 (Vesting/Slashing), §6 |

---
## Annex B (Optional): Sealed PoS²‑L Scaffold

> Disabled by default. This annex preserves an optional sealed path for phased rollout or emergency response. When activated by governance, SPs maintain a sealed scaffold (fractional row coverage) and answer PoS²‑L window proofs bound back to the original DU commitment via KZG content openings. PoDE derivations MAY be required on a fraction of sealed rows to ensure the scaffold is still linked to plaintext. Switching out of scaffold mode MUST be monotonic and resets vesting gates to PoUD + PoDE.

Governance dials (bounded): `φ_seal` (sealed‑row fraction), `p_link` (fraction of PoS²‑L challenges with KZG content binding), `p_derive` (fraction requiring row‑local PoDE), all subject to a Verification Load Cap.
| **Beacon grinding** | Bias challenges | BLS VRF uniqueness; BATMAN threshold; on‑chain pairing check; domain separation | spec §5 (VRF), metaspec §6.2 (Challenge) |
| **Merkle truncation misuse** | Excessive path truncation weakens PoS² | Prefer higher‑arity Merkle or longer per‑sibling bytes; security bound documented | spec §4.3.1 (witness); §4.1 (arity option) |
| **Economic instability** | Excess $BW inflation via spam | Epoch cap `I_epoch_max`; α bounds; per‑DU caps; tips burned; RTT oracle weights | §5.2 (BW), §4.2 (RTT Oracle) |
| **Deal fraud / mispricing** | Non‑delivery after payment | Escrowed $STOR; vesting gated by PoS²; repair bounties | §7.2–7.3, §3.3 |

| **Receipt inflation via colluding gateways** | Faked signatures/RTT | Probabilistic sampling (§6.3); RTT oracle transcripts; slashing (`ε` threshold) | §6.3, §4.2 |
| **Writer inconsistency (2D profiles)** | Malicious sliver encodings | Encoded metadata; f+1 inconsistency proofs → mark invalid; writer slashing | §3.2.z.3 |
| **Epoch handover stalls** | New committee not ready | Multi‑stage reconfiguration; readiness signaling; emergency bounties | §7.4 |

**Reviewer note:** Items labeled *Normative* above correspond to concrete MUST/SHALL language in the main text; others are informative guidance tied to governance levers.
