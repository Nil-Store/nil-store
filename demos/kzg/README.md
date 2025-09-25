Nilcoin KZG Demos (educational)

This folder contains two small demos showing KZG commitments over file data.

- kzg_toy.py — Toy KZG simulator (no real curves/pairings). Useful for learning only.
- kzg_ckzg.py — Real KZG using ckzg (c-kzg-4844) bindings over BLS12-381.

Quick start
- Toy: python3 demos/kzg/kzg_toy.py
  - Commits the repository's spec.md and writes a JSON artifact.

- Real (ckzg): python3 demos/kzg/kzg_ckzg.py --file spec.md --trusted-setup ./trust_setup.txt
  - trust_setup.txt: provide a c-kzg-4844 compatible trusted setup file in repo root or pass a path.
  - Produces commitments (one per EIP-4844 blob) and example proofs; verifies them and writes JSON.

Notes
- Both scripts are educational. Do not use in production.
- kzg_toy.py pads the file into 1 KiB blocks and simulates KZG with an exponent-tracker model.
- kzg_ckzg.py hashes 1 KiB blocks to Fr, packs 4096 Fr per blob (128 KiB), then uses ckzg.

