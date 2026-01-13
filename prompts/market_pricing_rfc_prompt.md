# External Agent Prompt: Market-Equilibrium Pricing RFC

You are an external agent tasked with drafting a **new RFC** that defines a market‑equilibrium pricing mechanism for NilStore storage and retrieval fees. You only have the assets provided in `market_pricing_rfc_assets.md`.

## Objective
Produce a complete, implementation‑oriented RFC that introduces **automatic price discovery** for storage (GiB‑month) and retrieval (GiB) pricing. The RFC must integrate with existing escrow/lock‑in accounting and the current param model without breaking invariants.

## Required Output
- A single RFC document in the repo’s RFC style and tone.
- Include:
  - Problem statement and motivation.
  - Design goals and non‑goals.
  - Mechanism overview (algorithm / dynamics).
  - State additions (on‑chain storage), params, and default values.
  - Update cadence and bounds (e.g., per epoch, max delta).
  - Interaction with escrow, spend windows, and retrieval settlement.
  - Governance control surface (what can be overridden, emergency kill switch).
  - Security / manipulation analysis (Sybil, wash retrievals, griefing, oscillations).
  - Backward compatibility / migration path.
  - Testing plan (unit + e2e + simulation gate).
  - Open questions + calibration signals (metrics and alert thresholds).

## Hard Constraints
- Must not alter the **accounting contract** already defined in `rfcs/rfc-pricing-and-escrow-accounting.md`.
- Must be compatible with the current params model and `nilchain` keeper patterns.
- Must include a minimal viable mechanism that can be implemented in stages.
- Avoid requiring off‑chain oracles or external price feeds.

## Guidance
Use only the context in `market_pricing_rfc_assets.md`. If a dependency is missing, list it under **Open Questions** instead of inventing details. Prefer deterministic and auditable mechanisms (e.g., utilization‑based control, supply/demand clearing within constraints, capped EMA updates).

## Deliverable Format
- Place the RFC in `rfcs/` with a filename like `rfc-market-equilibrium-pricing.md`.
- Use headings consistent with existing RFCs in the repo.

## Clarifications to Resolve in RFC
- What signals drive price updates? (utilization, backlog, repair load, audit budget spend, etc.)
- How to avoid feedback loops that destabilize spend windows or escrow?
- How are cold vs hot deals handled? Same pricing or separate markets?
- How to prevent manipulation via fake retrievals or trivial writes?
- How does the mechanism behave at bootstrap when there is little demand?

## Success Criteria
- RFC is detailed enough for an engineer to implement without further product decisions.
- Uses consistent terminology and existing chain parameters where possible.
- Includes explicit state/param definitions and a concrete testing plan.
