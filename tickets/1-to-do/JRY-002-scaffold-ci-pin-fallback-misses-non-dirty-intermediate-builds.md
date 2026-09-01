---
id: JRY-002
title: Scaffold CI pin fallback misses non-dirty intermediate builds
project: jerry
depends-on: []
spawned-by: [JRY-001]
impact: medium
complexity: low
cost: S
---

# JRY-002 — Scaffold CI pin fallback misses non-dirty intermediate builds

## Outcome

`jerry init` run from any binary build — including a local `just build` between tags, not
only a tagged release or an explicit dev/dirty build — emits scaffolded CI that never pins
to a module version that cannot resolve.

## Description

Found during JRY-001's review: `internal/scaffold/scaffold.go`'s `replaceTokens` (~line 172)
only falls back to `@latest` when `pin == "" || pin == "dev" || strings.Contains(pin,
"-dirty")`. A binary built between tags but not dirty — e.g. `just build`'s own output,
which stamps a pseudo-version like `v0.1.1-3-g3f336b9` — matches none of those branches. If
someone runs `jerry init` from such a binary, the emitted CI pins to
`go install github.com/codcod/jerry/cmd/jerry@v0.1.1-3-g3f336b9`, a commit-describe string
that is not a real module version and will never resolve — the same failure class JRY-001
found and fixed twice for the base install path, just one case its acceptance test (built
from a clean tagged install) didn't happen to hit.

Pre-existing in Phase 1 code, not introduced by JRY-001 — out of that ticket's stated scope
(release verification), filed here instead. Fix is narrowing the fallback condition to
match any non-exact-tag pin (e.g. anything containing a `-` after the version, not just
`-dirty`), or equivalently inverting the check to require the pin look like a clean semver
tag before using it verbatim.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-01 — created (TO DO). source: review: found during JRY-001's independent review audit
