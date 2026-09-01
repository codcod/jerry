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

- [x] Reviewer independence settled (step 0): **delegated**. The reviewing agent (this
  session) authored the branch, so steps 2–4a were delegated to a fresh, adversarially-briefed
  independent agent with no memory of writing the code. Every delegated finding below was
  re-verified by hand before recording — one (F1) was reclassified from blocking to
  non-blocking on that re-verification (see F1).
- [x] Implementation audit — acceptance test re-run verbatim on
  `feat/JRY-002-pin-fallback-non-dirty-builds`:
  `go test ./internal/scaffold/... -run TestVersionPinning -v` — all 5 subtests pass,
  including the new `NonDirtyIntermediateBuildFallsBackToLatest`. Red-before-green
  independently confirmed (reverting the condition back to `-dirty`-only fails exactly that
  subtest, others still pass — not a tautology). All 3 tasks landed in the files the plan
  names; both confirmed design decisions honoured verbatim. `just build`/`test`/`lint`/
  `docs-check` all pass.
- [x] Quality audit (step 3) — idiomatic Go, no security/error-handling concerns; behaviour
  change is strictly toward the safer `@latest`. CHANGELOG entry accurate, correctly placed.
  One finding (F2, below).
- [x] Consistency audit (step 4) — no duplicate copy of the fallback logic elsewhere; `jerry
  version` and `cmd/jerry/main.go`'s `Version` pass the raw string through untouched, so the
  broadened check has no interaction with them; the release path (`.goreleaser.yaml`) stamps
  a hyphen-free tag for real releases, so tagged releases still pin correctly. Two findings
  (F1, F3).
- [x] Documentation audit (step 4a) — `just docs-check` passes; no new user-facing surface
  beyond the CHANGELOG entry the plan already scoped (fallback logic is internal to
  `replaceTokens`). Whole-tree sweep surfaced F1 and F3.
- [ ] Docs-readability pass (step 4b, optional) — **conscious skip**: no docs-readability
  reviewer configured in this host session.
- [x] Findings recorded below with severity, class, and disposition; disposition summary and
  cost line present (step 5).
- [x] Ticket moved to `tickets/6-done/`; `## History` appended (step 6).
- [x] Other references updated; governing document `RELEASING.md` reconciled inline (F1);
  board regenerated by the move (step 7).
- [x] Remaining-tickets impact sweep (step 8) — no ticket in `1-to-do/`/`2-ready/` references
  JRY-002 in `depends-on:` or Description; nothing to patch.
- [x] Summary, commit message, and MR attributes presented for approval; overarching
  bookkeeping committed on `main`; next-ticket suggestion given (step 9).

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | stale-xref | fixed inline | `RELEASING.md`'s description of the fallback ("a development build (`dev`, or a `-dirty` describe) falls back to `@latest`") was made stale by this branch's broadened condition | `RELEASING.md:37-38` (pre-fix) | independent reviewer flagged this **blocking**; reclassified non-blocking on re-verification — `resources/review-protocol.md` §7 is explicit that a governing document merely made stale by the branch "breaks no golden path, ships no wrong behaviour, and contradicts no ticket's locked decision," so it is non-blocking by that step's own rule. Fixed inline: reworded to describe any non-exact-tag build, cites JRY-002 |
| F2 | non-blocking | other | fixed inline | The `replaceTokens` comment this branch edited didn't cite the ticket, per `AGENTS.md`'s "comments explain why, not what, and cite tickets and design sections inline" convention | `internal/scaffold/scaffold.go:175-178` (pre-fix); `AGENTS.md:42-43` | fixed inline: added `(JRY-002)` to the comment |
| F3 | non-blocking | test-gap | noted | `TestVersionPinning` only asserts the GitHub `docs.yml` output; the identical `__JERRY_VERSION__` token in `.gitlab-ci.yml` is untested for any pin case — pre-existing gap (all 4 original cases were GitHub-only too), not introduced or widened by this ticket | `internal/scaffold/scaffold_test.go:125` (`Forge: ForgeGitHub` only) | leave as-is; a GitLab-coverage ticket would need to weigh against `TestScaffoldValidatesClean`/`TestNoScriptsDirectoryIsEmitted` already exercising `ForgeGitLab` at the file-emission level — doesn't pass the promotion test on its own |
| F4 | non-blocking | stale-xref | noted | Both emitted CI templates comment "Pinned, never @latest" next to the install line — already inaccurate for `dev`/`-dirty` builds before this ticket; this branch widens which builds hit `@latest` but did not originate the inaccuracy | `internal/scaffold/templates/github/.github/workflows/docs.yml:29`, `internal/scaffold/templates/gitlab/.gitlab-ci.yml:13` | pre-existing per the rules §5's causation test ("did this branch break it?" — no); leave as-is, not worth a ticket on its own |

**Disposition summary:** 2 fixed inline (F1, F2), 2 noted (F3, F4). No blocking findings.

cost: estimated S, actual S — plan executed as written; one extra review round of two small
fix-inline edits (RELEASING.md wording, a comment citation) stayed well within scope.

## History

- 2026-09-01 — created (TO DO). source: review: found during JRY-001's independent review audit
- 2026-09-01 — TO DO → READY: plan complete
- 2026-09-01 — READY → IN DEVELOPMENT: picked up
- 2026-09-01 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-09-01 — IN REVIEW → DONE: review clean; 2 fixed inline, 2 noted
