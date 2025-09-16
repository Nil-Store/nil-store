# NilStore Network Specification: Product, Economics, and Network Layers (v0.1)

**Status:** Draft
**Date:** 2025-09-15

## 1. Introduction

### 1.1 Overview
This document specifies the higher-level layers of the NilStore decentralized storage network. It defines the data abstraction mechanisms, the dual-token economic model, the routing and retrieval protocols (Nil-mesh), and the architecture for governance and cross-chain bridging. NilStore aims to provide a highly resilient, low-latency storage solution that is accessible to both large-scale data centers and edge devices.

### 1.2 Relationship to the Cryptographic Core
The mechanisms described herein rely entirely on the primitives defined in the **NilStore Cryptographic Core Specification (spec.md)**. The Core Subspec establishes the foundation for:

*   **PoS² (Proof-of-Spacetime-Squared):** The consensus mechanism merging storage verification and bandwidth accounting.
*   **Commitment Scheme:** KZG polynomial commitments for data integrity.
*   **Sealing:** The CPU-efficient process for generating verifiable replicas.
*   **Nil-Lattice:** The mathematical structure underpinning data placement.

This document assumes the functionality and security guarantees provided by the Core Subspec.

## 2. Data Layer: Automatic Object Handling (NilFS Abstraction)

The Data Layer abstracts the complexity of data segmentation, replication, and placement, ensuring a seamless experience for users and standardized operations for Storage Providers (miners).

### 2.1 Object Ingestion and Chunking
Users submit arbitrary objects (files). The client software automatically processes them:

1.  **Chunking:** Objects are split into variable-sized blocks using **Content-Defined Chunking** (e.g., Rabin fingerprinting). This optimizes for network-wide deduplication.
2.  **Merkle DAG:** Chunks are organized into a Merkle DAG (compatible with IPFS CIDv1). The root CID identifies the complete object.

### 2.2 Segmentation and Data Units
To standardize the data miners handle, chunks are packed into standardized **Data Units (DUs)**.

1.  **Packing:** Chunks are serialized and packed into DUs.
2.  **Standardization:** DUs have standardized sizes, selectable powers-of-two between 1 MiB and 8 GiB, determined by the client based on the total object size and network conditions.
3.  **Abstraction:** Miners only interact with standardized DUs, remaining agnostic to the original file structure or chunk boundaries.

### 2.3 Erasure Coding and Sharding
Each Data Unit is independently erasure-coded into shards.

1.  **Reed-Solomon (RS):** DUs are encoded using systematic RS codes.
2.  **Parameters:** Default configuration is (n=12, k=9), derived from `n = k + ⌈k / 4⌉`, providing 1.33x redundancy. Parameters may vary based on user-selected durability profiles.
3.  **Shards:** The output is 'n' shards, which inherit the standardized size of the parent DU.

### 2.4 Deterministic Placement (Nil-Lattice)
Shards are placed on the directed hex-ring lattice to maximize fault tolerance.

1.  **Lattice Coordinates:** The position (r, θ) is determined by:
    `pos := Hash(CID_DU ∥ ClientSalt ∥ SlotIndex) → (r, θ)`
2.  **Client Salt:** A 32-byte random salt ensures cryptographic uniqueness across different deals, preventing reuse attacks.
3.  **Topological Distribution:** The lattice ensures that redundant shards are placed topologically distant, minimizing the risk of correlated failures.

### 2.5 Automatic Repair and Resilience
The network autonomously maintains data durability.

1.  **Monitoring:** Shard availability is monitored via gossip protocols and retrieval success rates.
2.  **Repair Trigger:** If the availability of a DU drops below a resilience threshold (e.g., k+1), a repair process is initiated.
3.  **Bounty Mechanism:** Any node can reconstruct the missing shards using the available 'k' shards and place them in new slots. Successful repairers claim a resilience bounty (e.g., 5% of the storage fee escrow).

## 3. Economic Layer (Dual-Token Model)

NilStore utilizes a dual-token economy to separate the incentives for long-term security (capacity commitment) from short-term utility (bandwidth provision).

### 3.1 $STOR (Staking and Capacity Token)
$STOR is the primary governance and security token.

*   **Supply:** Fixed (1 Billion $STOR).
*   **Function:**
    *   **Staking/Collateral:** Validators and Storage Providers bond $STOR as collateral against their commitments (consensus participation and data storage).
    *   **Governance:** $STOR holders control the NilDAO (See Section 6).
    *   **Storage Fees:** Users pay for storage capacity in $STOR, which is escrowed for the duration of the deal.
*   **Yield:** Target 5% annual staking yield, plus earned storage fees.
*   **Sink:** Slashing events (failure to submit PoS², consensus faults).

### 3.2 $BW (Bandwidth Scrip)
$BW is the utility token rewarding data retrieval.

*   **Supply:** Elastic and inflationary, based on network activity.
*   **Minting Formula:** Inflation per epoch (e.g., 24h) is calculated using a sublinear function to incentivize usage while controlling inflation:
    `I_epoch = α · sqrt(Total_Bytes_Served_NetworkWide)`
    (α is a governance-tunable constant).
*   **Distribution:** Minted $BW is distributed pro-rata to miners who submit verified, signed client retrieval receipts in their PoS² proofs.
*   **Sink (Burning):** Users can optionally "tip" miners for priority retrieval by including a burn amount in the receipt.

### 3.3 Incentive Alignment and Flywheel
*   **Merged Proofs (PoS²):** Miners must prove both continued storage (via KZG openings) AND bandwidth served (via receipt aggregation) in a single SNARK. This ensures miners are incentivized to be highly available.
*   **Edge Participation:** Low hardware requirements (CPU-only sealing) and variable shard sizes (1 MiB minimum) allow edge devices (phones, Raspberry Pis) to participate and earn $BW by serving hot data, even without significant $STOR collateral.

## 4. Network and Routing Layer (Nil-Mesh)

Nil-Mesh is the specialized overlay network optimized for low-latency discovery and retrieval.

### 4.1 Heisenberg-Lifted Routing
Nil-Mesh utilizes the underlying lattice topology for pathfinding.

*   **Mechanism:** Peer IDs (clients and miners) are "lifted" to elements in a 2-step nilpotent Lie group (Heisenberg-like structure).
*   **Pathfinding:** K-shortest paths are computed in this covering space and then projected back to the physical network topology.
*   **Advantage:** Offers significant latency reduction, particularly in sparse regions, compared to standard DHT hop-count routing.

### 4.2 Quality of Service (QoS) and RTT Attestation
*   **RTT Attestation:** Nodes continuously monitor and sign Round-Trip Time (RTT) attestations with their peers. These are gossiped and periodically anchored on-chain.
*   **Path Selection:** Clients use these verifiable QoS metrics to select the fastest 'k' providers for retrieval.
*   **Fraud Prevention:** RTT attestations are used to verify that bandwidth receipts are genuine and not self-dealt (verifying RTT > network floor).

### 4.3 Sybil Resistance
The topological structure enhances Sybil resistance. Disrupting routing requires controlling an entire "Ring Cell" (a topological area on the hex lattice), which is significantly more costly than controlling individual peer hops.

## 5. Retrieval Market

### 5.1 Retrieval Flow
1.  **Discovery:** Client resolves the Object CID to DU CIDs, then queries Nil-Mesh for the locations of the required shards, prioritizing by RTT attestations.
2.  **Parallel Fetch:** Client downloads any 'k' shards simultaneously.
3.  **Verification:** The client requests KZG openings at a client-chosen random evaluation point (x†) and verifies them against the committed root.
4.  **Reconstruction:** Client performs RS decoding.

### 5.2 Bandwidth Accounting
1.  **Receipts:** Upon successful verification, the client signs a `Receipt(CID_DU, Bytes, Nonce, Optional_Tip_Burn)`.
2.  **Aggregation:** Miners aggregate these receipts into a Poseidon Merkle tree and include the root in their Epoch PoS² proof to claim $BW.

## 6. Bridge and Governance Architecture

### 6.1 Hybrid Architecture (L1/L2 Bridge)
NilStore separates Data Availability (DA) consensus from economic settlement.

*   **Core Consensus L1 (DA Chain):** A minimal, specialized L1 (e.g., Cosmos-SDK/Tendermint) optimized for verifying PoS² proofs, managing staking, and handling data blobs efficiently.
*   **Settlement L2 (ZK-Rollup):** A ZK-Rollup (utilizing PlonK/Kimchi) deployed on an EVM-compatible ecosystem (e.g., Ethereum L2).

### 6.2 Bridging and Composability
*   **Proof Aggregation:** The L1 aggregates all epoch PoS² proofs into a single recursive SNARK, which is posted to the L2 bridge contract. This securely updates the state without trusted relayers.
*   **Liquidity:** $STOR and $BW are bridged to the L2 as ERC-20 tokens.
*   **Deal NFTs:** Storage contracts are represented as NFTs (ERC-721) minted on the L2. This enables integration with DeFi, allowing storage contracts to be traded or used as collateral.

### 6.3 Governance (NilDAO)
The network is governed by the NilDAO, using stake-weighted ($STOR) voting.

*   **Parameter Tuning:** The DAO controls economic constants (α), slashing ratios, RS parameters (n, k), and standard DU sizes.
*   **Upgrade Process:** Standard upgrades require a proposal, voting period, and timelock.
*   **Emergency Powers:** A timelocked emergency circuit exists for urgent fixes. It requires a super-majority (e.g., ⅔) of a Security Council or a 2-of-2 sign-off (e.g., Architect/Security Lead). Emergency patches implemented via the 2-of-2 mechanism automatically expire after 14 days unless ratified by a full DAO vote.
