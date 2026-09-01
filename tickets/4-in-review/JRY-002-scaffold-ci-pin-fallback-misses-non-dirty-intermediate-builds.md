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

### Feature branch

`feat/JRY-002-pin-fallback-non-dirty-builds` in the jerry repo (root-path child; base
`main`).

### Prerequisite gate (hard)

None. No `depends-on:`; nothing else must be true before starting.

### Confirmed design decisions (do not deviate without asking)

1. **Broaden the fallback condition from `strings.Contains(pin, "-dirty")` to
   `strings.Contains(pin, "-")`** in `replaceTokens`
   (`internal/scaffold/scaffold.go:177`). `git describe --tags --always --dirty` (the
   `justfile`'s version stamp) emits a bare `v<semver>` string only for an exact, clean tag
   build; every other case — dirty-at-tag (`-dirty` suffix) or commits-since-tag
   (`-<count>-g<hash>` suffix, optionally followed by `-dirty`) — always contains a hyphen.
   This one check folds in the existing `-dirty` case rather than sitting alongside it.
2. **Accept the known trade-off**: a semver pre-release tag that itself contains a hyphen
   (e.g. `v1.0.0-rc1`) would also fall back to `@latest` even when built exactly at that
   tag. No such tag exists in this project's history and none is planned; not special-cased
   further here.

### Tasks

#### Task 1 — narrow the fallback check

In `internal/scaffold/scaffold.go`, `replaceTokens` (~line 177), change:

```go
if pin == "" || pin == "dev" || strings.Contains(pin, "-dirty") {
```

to:

```go
if pin == "" || pin == "dev" || strings.Contains(pin, "-") {
```

#### Task 2 — cover the fixed case in the existing test table

In `internal/scaffold/scaffold_test.go`, `TestVersionPinning`'s `cases` table (~line 112),
add a case for the exact pseudo-version from this ticket's Description:

```go
{"NonDirtyIntermediateBuildFallsBackToLatest", "v0.1.1-3-g3f336b9", "@latest"},
```

#### Task 3 — changelog

Add a `### Fixed` bullet under `## [Unreleased]` in `CHANGELOG.md`: `jerry init` run from a
binary built between tags (e.g. `just build`'s own pseudo-version, `v0.1.1-3-g<sha>`) now
falls back to `@latest` in scaffolded CI instead of pinning to an unresolvable
commit-describe string.

### Acceptance test

```sh
go test ./internal/scaffold/... -run TestVersionPinning -v
```

All five cases pass, including the new `NonDirtyIntermediateBuildFallsBackToLatest` one —
confirming it fails against the pre-fix condition first (red), then passes after Task 1
(green). Then:

```sh
just test
just lint
```

both clean.

### Docs update (mandatory when user-facing)

`CHANGELOG.md` — Task 3, above. No other user-facing surface changes (the fix is internal
to `replaceTokens`; no flag, command, or scaffolded-file behaviour changes beyond correcting
this one pin case).

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` all
   clean.
2. `CHANGELOG.md` updated (Task 3).
3. Write a summary: files touched, the confirmed decisions above, anything deferred (the
   pre-release-tag trade-off, decision 2).
4. Suggest a Conventional Commit message, e.g.:

   ```
   fix(scaffold): fall back to @latest for any non-exact-tag pin (JRY-002)

   `replaceTokens` only fell back when the version pin was empty, "dev", or contained
   "-dirty" — a binary built between tags (e.g. `just build`'s own pseudo-version) matched
   none of those and pinned scaffolded CI to an unresolvable commit-describe string.
   Broaden the check to any pin containing a hyphen, which covers every non-exact-tag case
   `git describe` can produce.
   ```

5. This is a root-path child (`path = "."`) — tidy WIP commits into a small number of
   atomic commits before presenting them.
6. Commit locally on the ticket branch. Publish only per the commit policy (no push / no
   MR without user approval). Present the commit message; only after approval finalize
   (keep the tidied history, the root-path default) and push, verifying the remote base is
   not behind first (`git fetch origin main && git diff --name-only
   origin/main...HEAD | grep '^tickets/'` prints nothing). Open the MR — merging is always
   the human's. Hand back to the user.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-01 — created (TO DO). source: review: found during JRY-001's independent review audit
- 2026-09-01 — TO DO → READY: plan complete
- 2026-09-01 — READY → IN DEVELOPMENT: picked up
- 2026-09-01 — IN DEVELOPMENT → IN REVIEW: acceptance green
