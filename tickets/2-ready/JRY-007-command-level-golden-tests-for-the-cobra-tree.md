---
id: JRY-007
title: Command-level golden tests for the cobra tree
project: jerry
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# JRY-007 — Command-level golden tests for the cobra tree

## Outcome

Every command's actual user-visible behaviour is pinned: the cobra tree is executed in-process
against fixture repositories, and stdout, stderr and the exit code are byte-compared against
golden files. A change to what a user sees becomes a test failure rather than a review comment.

## Description

Phase 1 has structural tests (`internal/cli/write_safety_test.go` enforcing the `jerry/kind`
annotation and `--dry-run` on every write leaf) and package-level tests for rules, index,
frontmatter, config, scaffold and hooks. No test asserts what a user actually sees. Two of the
three defects found during the Phase 1 pass were exactly that class: cobra's `Print*` writing
to stderr and breaking the git hook, and the scaffold shipping a repo that failed its own CI.

`internal/cli/cli.go` is already shaped for this — commands are constructors taking `*globals`,
with no package-level root and no `init()` registration, so two trees in one process never share
flag state and a test can build a fresh tree per case. `cmd.OutOrStdout()` is used throughout.

Scope: fixture repositories (one clean, one with a representative finding of each severity),
in-process execution of each leaf command, and golden files for the three observable channels.
Exit codes matter as much as output — `validate` returning `errFailed` with `SilenceErrors` is
behaviour a golden test should hold, since a validator's exit code is what CI reads.

Follow `AGENTS.md` on fixtures: they are rendered from the production path, never hand-edited
(`just test-update`), and each golden test is paired with a `…FixtureContract` test asserting
the properties the fixture exists to demonstrate, so regenerating cannot quietly drop one.

**Why it is scheduled early.** `PLAN.md` v1 parked this in a cross-cutting section with no
scheduling rule, which is where infrastructure tickets go to be permanently outranked by
features. It is buildable today, every later ticket benefits, and nothing about it gets easier
by waiting. JRY-006 (placeholder rule fenced-block skip) has since shipped without waiting on
this — the coupling was only ever soft — but the still-unfiled `validate --diff` hardening
(`PLAN.md`'s `diff-hardening` row) is exactly the kind of observable-output assertion this
ticket exists to make cheap, so it stays scheduled ahead of it.

## Implementation Plan

### 0. Feature branch (mandatory)

```
git checkout main
git checkout -b feat/JRY-007-command-golden-tests
```

Work and commit locally on this branch. Publish only per the project's commit policy: no
push / no merge request without explicit user approval; tidy WIP commits into atomic ones
before presenting them (root-path child, `pickle.toml` `path = "."`).

### Prerequisite gate (hard)

None. `depends-on: []`; the tree already has no package-level root and no `init()`
registration (`internal/cli/cli.go`), so a fresh tree per test case is already free.

### Confirmed design decisions (do not deviate without asking)

1. **Fixture repositories are built through the production path, never hand-authored
   markdown.** `scaffold.Run` (the same call `internal/scaffold/scaffold_test.go` uses via its
   `emit` helper) produces both fixtures; a schema or template change then updates every
   fixture the same way a real repo would, instead of rotting a hand-written one silently.
   This extends AGENTS.md's golden-fixture rule (rendered, never hand-edited) from output
   goldens to these command-test *inputs* as well.
2. **Two fixtures, each pinned by its own `…FixtureContract` test:** `cleanFixture(t)` is a
   freshly scaffolded repo (`scaffold.Options{Forge: scaffold.ForgeGitHub}`) with `git init`
   run in it (so `hooks install`/`uninstall` have a real `.git` to act on); its contract is
   `rules.Check` returning zero findings, matching `TestScaffoldValidatesClean`. `dirtyFixture(t)`
   is the same scaffold with exactly one `rules.SeverityError` and one `rules.SeverityWarning`
   finding introduced by driving the tree's own commands (e.g. `status` to leave an ADR
   `Proposed` past the staleness window under the pinned clock, plus one deliberately
   malformed frontmatter key) — never by hand-editing a doc's markdown body. Its contract test
   asserts exactly one of each severity, so regenerating cannot silently drop the property the
   fixture exists to demonstrate.
3. **Commands run in-process via the existing `newRoot(version)`** (`internal/cli/cli.go`) —
   no new exported seam. Each case builds a fresh tree, sets `g.configPath` to the fixture's
   `jerry.yaml` (never `os.Chdir`), and calls `cmd.SetOut`/`cmd.SetErr` with separate buffers
   so stdout/stderr are compared independently, matching the reasoning already in
   `internal/cli/cli.go`'s `versionCmd` comment about `cmd.OutOrStdout()` vs cobra's `Print*`.
4. **The clock is pinned for every case.** `now` (`internal/cli/corpus.go:26`, a package var
   for exactly this reason) is set to a fixed `time.Time` for the duration of each test, since
   `validate`, `new` and `supersede` all read it and an unpinned clock would churn dates in the
   golden files whenever a run crosses a staleness boundary.
5. **The golden fixture format is one JSON file per case**, `{stdout, stderr, failed}`,
   following the existing `-update` flag convention (`internal/rules/rules_test.go`,
   `internal/index/index_test.go`) — no new mechanism, and `just test-update` (already
   `go test ./... -update`) regenerates these with no separate wiring.
6. **"Exit code" means whether `RunE` returned a non-nil error, recorded as `failed: bool`,
   not an OS exit code.** `cmd/jerry/main.go` maps every non-nil error from `cli.Execute` to
   `os.Exit(1)` uniformly — there is no second exit code to distinguish — so a golden field
   holding a literal `1` for every failing case would only restate that constant.

### Tasks

#### Task 1 — Fixture builders (`internal/cli/fixtures_test.go`)
`cleanFixture(t *testing.T) string` and `dirtyFixture(t *testing.T) string`, each returning a
`t.TempDir()` root built per decision 2. Reuse `scaffold.Run` exactly as
`internal/scaffold/scaffold_test.go`'s `emit` does; shell out to `git init` (`exec.Command`)
after scaffolding.

#### Task 2 — `…FixtureContract` tests (`internal/cli/fixtures_test.go`)
`TestCleanFixtureContract`: load the corpus, assert `rules.Check(...)` is empty.
`TestDirtyFixtureContract`: assert the findings contain exactly one `SeverityError` and
exactly one `SeverityWarning`. Both run under the pinned clock (decision 4).

#### Task 3 — In-process command runner (`internal/cli/golden_test.go`)
`runCLI(t *testing.T, root string, args ...string) (stdout, stderr string, failed bool)`:
builds `newRoot("golden")`, sets `g.configPath` to `filepath.Join(root, "jerry.yaml")`,
`SetOut`/`SetErr` to `bytes.Buffer`s, `SetArgs(args)`, calls `.Execute()`, returns the buffers
and `err != nil`. Pins `now` for the call per decision 4 (save/restore around each case, since
`now` is package state shared with production code).

#### Task 4 — Golden case table and test (`internal/cli/golden_test.go`)
A table of `{name string, fixture func(*testing.T) string, args []string}` with one entry per
leaf command in the tree today: `version`, `init` (`--dry-run`, against an empty temp dir, not
a fixture), `new adr`/`new sd` (`--dry-run`), `validate` against *both* fixtures, `fmt`
(`--dry-run`), `index` (`--check`), `supersede`, `status`, `schema`, `hooks install`/
`uninstall` (`--dry-run`) and `hooks status`. For each case, run via `runCLI`, marshal
`{Stdout, Stderr, Failed}` to indented JSON, and byte-compare against
`internal/cli/testdata/golden/<name>.json`, following the `updateGolden = flag.Bool("update", …)`
pattern from `internal/rules/rules_test.go` verbatim (write-then-return under `-update`, else
read-and-compare).

#### Task 5 — Coverage guard (`internal/cli/golden_test.go`)
`TestGoldenCoversEveryLeaf`: reuse `leaves(root)` from `write_safety_test.go` (same package) to
enumerate the real tree, and assert every non-generated leaf's `CommandPath()` appears in the
Task 4 case table — so a new leaf command ships with no golden case as a test failure, not a
silent gap.

### Acceptance test

```
go test ./internal/cli/... -v
go test ./... -update && git status --porcelain internal/cli/testdata   # must print nothing: goldens are reproducible
just test
just lint
```

### Docs update (mandatory when user-facing)

No user-facing surface — this is a test-only change to `internal/cli`. AGENTS.md's existing
"Golden fixtures are rendered from the production path" bullet already covers the convention
this ticket extends to fixture *inputs*; no wording change needed there.

### Finish (mandatory)

1. Acceptance test green; `just build test lint docs-check` clean.
2. No docs to update (see above).
3. Write a summary: files touched, which representative finding each fixture carries, anything
   deferred.
4. Suggest a Conventional Commit message, e.g.:
   ```
   test(cli): add command-level golden tests for the cobra tree (JRY-007)
   ```
5. Tidy WIP commits into a small number of atomic commits before presenting (root-path child).
6. Commit locally on the ticket branch; do not push or open a merge request without user
   approval. Present the commit message; only after approval finalize and push, verifying
   (under `layout = "in-tree"`) that `origin/main...HEAD` carries no `tickets/` path before
   pushing — then open the merge request. Hand back to the user.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new
- 2026-09-03 — TO DO → READY: plan complete
