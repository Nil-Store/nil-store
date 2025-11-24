# NilStore / Nilcoin quick guide for agents

- Purpose: specs and drafts for NilStore (a $STOR-only storage network) built on PoUD (KZG proofs of canonical bytes) + PoDE (timed Argon2id derivations), with DA L1 + ZK-bridged L2 settlement.
- Canonical data: each deal sets `privacy_mode` (`ciphertext` default, `plaintext` optional). All commitments and PoDE work run over those canonical bytes; never assume decrypt access when `ciphertext`.

## Core documents
- `spec.md` — NilStore Core v2.0 cryptography (normative). Holds the version triple (0x02.00.00), domain IDs/strings, dial profile S-512, PoDE Argon2id params (`H_t=3`, `H_m=1 GiB`, `H_p=1` sequential), KZG embedding (BLS12-381 SRS), PoUD obligations, Nil-VRF + BATMAN beacon rules, governance dial process, and Known-Answer-Test requirements.
- `whitepaper.md` — White Paper v1.0 (2025-10-02). Architecture, economic model, Nil-Mesh, L1/L2 split, Nil-VRF. Tracks Core defaults (PoUD+PoDE, RS(12,9), nil-lattice placement, yellow-flag bridge freeze).
- `metaspec.md` — Working draft v0.5; similar coverage as the whitepaper, slightly older wording; references Core Appendix A for manifests/crypto policy.
- `rfcs/PoS2L_*` — Research-only sealed proof scaffold (PoS²-L). Explicitly disabled for mainnet/testnet. Activation needs DAO supermajority + 4-of-7 emergency signers, 14-day auto-sunset, payouts still tied to plaintext correctness fractions. Treat as archive unless a research runbook is requested.
- Demos: `demos/kzg/README.md` with `kzg_toy.py` (sim), `kzg_ckzg.py` (ckzg bindings) — educational only. `trusted_setup.txt` expected for real demo. `build_whitepaper.sh` builds PDFs via pandoc (needs LaTeX/wkhtmltopdf).

## Protocol quick facts
- Commitment: DU bytes chunked into 31-byte little-endian field elements; commit with KZG over the shared BLS12-381 SRS. SRS integrity must be hash-pinned and audited.
- Versioning: every digest prefixes the version triple; unknown triples/domain IDs are rejected. Parameter changes are dialed via governance and hash-pinned `$STOR-1559` PSet on L2.
- PoUD: per-epoch KZG multi-open over random 1 KiB symbols. Windows and symbol counts are driven by the Nil-VRF epoch beacon.
- PoDE Derive (normative): Argon2id sequential only (`H_p=1`), salted by the beacon and identifiers; outputs `(leaf64, Δ_W)`. Prover must link `canon_window` to `C_root` with a KZG opening. Governance floors: `R ≥ 16` windows, `B_min ≥ 128 MiB` verified bytes (floors cannot go below 8/64 MiB without major version bump).
- Beacon: Nil-VRF (BLS12-381) with BATMAN threshold aggregation. Domain tags include `NIL_VRF_OUT`, `NIL_BEACON`, `NilStore-Sample`, `SAMPLE-EXP`, etc. Beacon seeds sampling and PoDE salts.
- Data layer: objects → CDC DAG → packed DUs. Default erasure coding RS(12,9) with 1 KiB symbols; optional RS-2D-Hex profile and durability dial mapping targets to `{RS_n, RS_k}` or `{rows, cols}`. Deterministic placement via nil-lattice hash(`CID_DU ∥ ClientSalt_32B ∥ shard_index`), at most one shard per SP per ring-cell; ClientSalt derived from signed deal params to stop grinding. Repairs must open against the original `C_root`.
- Economics/bridge: DA L1 with KZG/Poseidon/Blake2s/VDF precompiles verifies proofs and watcher timing; posts recursive proof/digest to L2 rollup (PlonK/Kimchi) with pinned `vk_hash`. Yellow-flag freeze halts fund-moving/governance paths except the ratification vote; auto-sunset if not ratified.

## Working guidance for future agents
- Default to `spec.md` for anything cryptographic or normative; use `whitepaper.md`/`metaspec.md` for architecture/economics narratives. Call out whether guidance is normative or informative.
- Keep PoS²-L isolated: do not propose sealed-mode proofs or payouts unless explicitly in research mode and after gating steps.
- Preserve canonical-byte rules and the `privacy_mode` choice in any design; avoid assuming plaintext access when `ciphertext`.
- When referencing proof parameters, quote the dial/profile values (S-512 defaults above) and note that changes require governance + new profile ID.
- If writing code samples, state reliance on the BLS12-381 SRS, domain separation tags, and Argon2id sequentiality; avoid inventing parameters not present in the docs.
- $STOR-only: contracts and economics exclude other assets/oracles; conversions stay off-protocol.
