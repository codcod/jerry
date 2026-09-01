---
schema_version: 1
id: ADR-NNNN                # must match the NNNN in the filename
title: Short title of the decision
status: Proposed            # Proposed | Accepted | Rejected | Deprecated | Superseded
# superseded_by: [ADR-0012] # required when status is Superseded. Same folder:
#                           # ADR-0012. Other folder: payments/ADR-0012.
# supersedes: [ADR-0004]    # optional, the reverse pointer
team: your-team             # must match the folder name under teams/
# teams: [payments, platform]   # cross-cutting ADRs use this instead of `team`
date: YYYY-MM-DD
deciders: [alice, bob]
# applies_to: [services/ledger]  # optional: paths or service ids this governs
---

# ADR-NNNN: Short title of the decision

## Context

What is the issue we're seeing that motivates this decision? Include relevant
constraints (technical, organizational, timeline). If this supersedes an earlier
ADR, reference it here and say what changed.

## Decision

What is the change we're making? State it plainly, in one or two sentences, then
elaborate if needed.

## Consequences

What becomes easier or harder as a result? Include negative consequences and
trade-offs honestly — this is the most valuable part of the record for future
readers.

## Alternatives Considered

- **Option A** — why it was rejected
- **Option B** — why it was rejected

## Related

- Related ADRs, Solution Designs, or external docs. Reference ADRs in other
  folders with their scope: `payments/ADR-0007`, `cross-cutting/ADR-0003`.
