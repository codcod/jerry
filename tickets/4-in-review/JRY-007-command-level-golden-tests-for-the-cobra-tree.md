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
   finding, introduced by a targeted two-line splice into the example ADR's frontmatter block
   (`applies_to: ["../escape"]` for the error, one unrecognised key for the warning) rather than
   by hand-authoring a document body — necessary because `doc.Front` has no field for an
   unrecognised key by definition, so there is no production-command path that adds one; the
   `status`-driven staleness route this decision originally named does not apply, since the
   scaffolded example ADR ships `status: Accepted` (never subject to that rule). Its contract
   test asserts exactly one of each severity, so regenerating cannot silently drop the property
   the fixture exists to demonstrate.
3. **Commands run in-process via the existing `newRoot(version)`** (`internal/cli/cli.go`) —
   no new exported seam. Each case builds a fresh tree, sets `g.configPath` to the fixture's
   `jerry.yaml`, and calls `cmd.SetOut`/`cmd.SetErr` with separate buffers so stdout/stderr are
   compared independently, matching the reasoning already in `internal/cli/cli.go`'s
   `versionCmd` comment about `cmd.OutOrStdout()` vs cobra's `Print*`. One exception to "never
   `os.Chdir`": `init` (`internal/cli/init.go`) resolves its target via `os.Getwd()` directly,
   ignoring `g.configPath` entirely — an assumption this decision did not carry when written.
   Its one golden case uses `t.Chdir` (auto-restoring) instead.
4. **The clock is pinned for every case.** `now` (`internal/cli/corpus.go:26`, a package var
   for exactly this reason) is set to a fixed `time.Time` for the duration of each test, since
   `validate`, `new` and `supersede` all read it and an unpinned clock would churn dates in the
   golden files whenever a run crosses a staleness boundary.
5. **The golden fixture format is one JSON file per case**, `{stdout, stderr, failed}`,
   following the existing `-update` flag convention (`internal/rules/rules_test.go`,
   `internal/index/index_test.go`) — no new mechanism. `just test-update` (`go test ./... -update`)
   itself already fails on this repo today, on `internal/scaffold`, whose test binary does not
   register `-update` — Go passes the flag to every package's test binary in a multi-package
   invocation, so any package without the flag errors regardless of which package added it
   (pre-existing, not introduced here; reproduces identically before this ticket's changes).
   Goldens are regenerated and verified reproducible scoped to the package instead:
   `go test ./internal/cli/... -update && git status --porcelain internal/cli/testdata` (updated
   below in the acceptance test).
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
`runCLI(t *testing.T, root string, chdir bool, args ...string) goldenResult{Stdout, Stderr,
Failed}`: builds `newRoot("golden")`, sets `g.configPath` to `filepath.Join(root, "jerry.yaml")`,
`SetOut`/`SetErr` to `bytes.Buffer`s, `SetArgs(args)`, calls `.Execute()`, returns the buffers
and `err != nil`. Pins `now` for the call per decision 4 (save/restore around each case, since
`now` is package state shared with production code). `chdir` is `init`'s one exception to "never
`os.Chdir`" (decision 3, amended). Also normalises the fixture's `t.TempDir()` root to a fixed
placeholder in both buffers before returning — `hooks install`/`uninstall`/`status` print the
absolute hook path (`internal/hooks/hooks.go`'s `Path`), which is a fresh path every run and
would otherwise make those three cases' golden files unreproducible.

#### Task 4 — Golden case table and test (`internal/cli/golden_test.go`)
A table of `{name, leafPath, fixture func(*testing.T) string, chdir bool, args []string}` with
14 entries covering the tree's 13 leaves (`validate` gets two: clean and dirty): `version`,
`init --dry-run` (`chdir: true`, an empty temp dir, not a fixture), `new adr`/`new sd
--dry-run` (`--deciders`/`--authors` given explicitly — both are read straight into the printed
dry-run body, and would otherwise fall back to the machine's global `git config user.name`,
making the golden file depend on who/where it was regenerated), `validate` against *both*
fixtures, `fmt --check`, `index --check`, `supersede --dry-run`, `status` (run for real — the
one case that exercises an actual write, since `--dry-run` output never reaches the
deciders/date fields that would otherwise need pinning), `schema`, `hooks install`/`uninstall
--dry-run` and `hooks status`. For each case, run via `runCLI`, marshal `{Stdout, Stderr,
Failed}` to indented JSON, and byte-compare against `internal/cli/testdata/golden/<name>.json`,
following the `updateGolden = flag.Bool("update", …)` pattern from `internal/rules/rules_test.go`
verbatim (write-then-return under `-update`, else read-and-compare). Resolve the golden
directory's absolute path once before the table loop, not per case — the `init` case's `chdir`
changes the process's working directory for the lifetime of its subtest.

#### Task 5 — Coverage guard (`internal/cli/golden_test.go`)
`TestGoldenCoversEveryLeaf`: reuse `leaves(root)` from `write_safety_test.go` (same package) to
enumerate the real tree, and assert every non-generated leaf's `CommandPath()` appears in the
Task 4 case table — so a new leaf command ships with no golden case as a test failure, not a
silent gap. `leaves()` itself does not filter cobra's generated `help`/`completion` — that
filter is applied at the call site in `TestWriteSafety` — so repeat the same
`generated[cmd.Name()] || generated[cmd.Parent().Name()]` check here rather than assume it is
baked into the walker.

### Acceptance test

```
go test ./internal/cli/... -v
go test ./internal/cli/... -update && git status --porcelain internal/cli/testdata   # must print nothing: goldens are reproducible
just build
just test
just lint
just docs-check
```

`just test-update` (`go test ./... -update`) itself fails today on `internal/scaffold`
independent of this ticket (decision 5) — do not attempt to fix that here; it is out of scope.

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

- [x] Reviewer independence settled (step 0): the reviewing agent authored this branch in this
  same session, so audits (steps 2-4) were **delegated** to an independent, adversarially-briefed
  sub-agent with no memory of writing the code. Findings below were re-verified by the
  orchestrating reviewer before recording (the delegated agent's cobra-source claims were
  cross-checked against `$(go env GOMODCACHE)/github.com/spf13/cobra@v1.10.2/command.go` directly).
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 1, 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit (step 4a) — no user-facing surface (test-only change to `internal/cli`);
  `just docs-check` clean; no `CHANGELOG.md` entry required (this project's convention carries no
  entry for internal test infrastructure — confirmed by grep, no prior test-only ticket has one)
- [ ] Docs-readability pass (step 4b) — conscious skip: no `.adoc`/`.md` prose changed by this branch
- [x] Findings recorded with severity, class, disposition (step 5)
- [x] Ticket moved (step 6) — see below
- [ ] Other references / governing documents (step 7) — deferred to the scoped re-review's
  conclusion, once F1 lands: `review-addendum.md`'s step-2 claim that this ticket "is where [the
  stdout/stderr split] becomes mechanical" is not yet true (see F1) and should be revisited then,
  not edited now while the fix is still pending
- [ ] Remaining-tickets impact sweep (step 8) — deferred: this round does not conclude the ticket
- [ ] Summary + commit message + MR attributes for approval (step 9) — deferred: not reaching
  `6-done/` this round

**Acceptance test, re-run verbatim:** all six commands from the Implementation Plan pass —
`go test ./internal/cli/... -v` (all subtests green), `go test ./internal/cli/... -update && git
status --porcelain internal/cli/testdata` (empty — goldens reproducible), `just build`,
`just test`, `just lint`, `just docs-check`. Every task and decision in the plan is implemented
exactly where it says, with one exception: decision 3's claim that `SetOut`/`SetErr` compare
stdout/stderr "independently" is contradicted by F1 below.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | correctness | — | `runCLI` (`internal/cli/golden_test.go`) calls both `tree.SetOut` and `tree.SetErr`, but cobra v1.10.2's `Print`/`Println`/`Printf` resolve through `OutOrStderr()` → `getOut(os.Stderr)`, which returns `c.outWriter` (the `SetOut` buffer) whenever it is non-nil — never `c.errWriter`. Every `cmd.Print*` call in the tree (used by `fmt`, `new`, `hooks`, `supersede`, `status`, `validate`) therefore lands in the captured "stdout" bucket, indistinguishable from `fmt.Fprintln(cmd.OutOrStdout(), …)`. All 14 committed goldens show `"stderr": ""`. This defeats the ticket's own Outcome (stdout/stderr/exit code "byte-compared") and the exact regression class it exists to catch (`CHANGELOG.md` 0.1.0's `Print*`-breaks-the-git-hook defect, cited in this ticket's own Description) — a leaf's output moving between the two real streams produces a byte-identical golden diff. | `internal/cli/golden_test.go:57-58`; cobra `command.go:398,412,1435-1441`; confirmed with a throwaway reproduction (`SetOut`+`SetErr` set, `RunE` calls both `cmd.Println("A")` and `fmt.Fprintln(cmd.OutOrStdout(), "B")` → stdout=`"A\nB\n"`, stderr=`""`) | Capture real stream separation instead of relying on cobra's `SetOut`/`SetErr` pair for this: redirect the OS-level `os.Stdout`/`os.Stderr` (e.g. `os.Pipe`) around `tree.Execute()` while leaving `SetOut`/`SetErr` unset, so `OutOrStdout()`/`OutOrStderr()` fall through to the real `os.Stdout`/`os.Stderr` cobra uses in production (`cli.Execute` never calls `SetOut`/`SetErr` either) — then every existing golden case's expected split can be regenerated and asserted against the real behaviour it claims to pin. |
| F2 | non-blocking | test-gap | note-and-close | No golden case currently produces non-empty `stderr`, so the capture wiring for that field is itself unexercised. | all 14 files in `internal/cli/testdata/golden/*.json` show `"stderr": ""` | Expected to resolve as a natural consequence once F1 is fixed correctly (several existing cases — `fmt --check`, `validate`, `status`, `supersede`, `new` — use `cmd.Print*` in production and will populate `stderr` once the split is real); no separate task needed. |
| F3 | non-blocking | test-gap | note-and-close | `hooks status`'s other two branches ("installed by jerry", "a hook exists but jerry did not write it") have no golden case — only "not installed" is covered. | `internal/hooks/hooks.go` (`Status` callers in `hooksStatusCmd`); `internal/cli/golden_test.go`'s single `hooks-status` case | Not required by the ticket's Task 4 (leaf coverage, not branch coverage); too small on its own to pass the promotion test for a follow-up ticket. Revisit if more coverage gaps of this shape accumulate. |
| F4 | non-blocking | test-gap | note-and-close | The `--json` envelope output (`jerry.findings/1`, `jerry.index/1`) is not exercised by any golden case. | `internal/cli/golden_test.go`'s `goldenCases` table (no `--json` args) | Outside Task 4's stated scope (it names no `--json` case); marginal value on its own. Revisit alongside F3 if a themed "broaden golden coverage" ticket is ever filed. |

**Disposition summary:** 1 blocking (F1, routed to rework), 3 non-blocking — all `note-and-close`
(F2, F3, F4).

cost: estimated M, actual M

### Rework fix record — round 1 (commit ffb4eb5)

Fixed F1 only, per its suggestion column: `runCLI` (`internal/cli/golden_test.go`) no longer
calls `tree.SetOut`/`tree.SetErr`. A new `captureStreams` helper swaps the real
`os.Stdout`/`os.Stderr` package variables for a pair of `os.Pipe`s around `tree.Execute()`
(read via two goroutines into buffers, joined after the write ends close), so
`OutOrStdout()`/`OutOrStderr()` fall through to the same real streams `cli.Execute` uses in
production. All 11 affected golden files regenerated (`go test ./internal/cli/... -update`) and
confirmed stable across two consecutive `-update` runs (identical SHA-1s). Spot-checked
`validate-dirty`: the machine-readable findings (`renderFindings(cmd.OutOrStdout(), …)`) now sit
in `stdout`, and the human summary line (`cmd.Printf`) now sits in `stderr` — the exact split F1
said was missing. F2 resolved as predicted, as a side effect of the fix, with no separate change:
9 of the 14 goldens now carry non-empty `stderr`. F3 and F4 were left untouched — out of this
round's scope. Full acceptance test re-run green (`go test ./internal/cli/... -v`, reproducibility
check, `just build`, `just test`, `just lint`, `just docs-check`).

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new
- 2026-09-03 — TO DO → READY: plan complete
- 2026-09-03 — READY → IN DEVELOPMENT: picked up
- 2026-09-03 — plan amended inline: decision 2's dirty-fixture mechanism switched from a
  `status`-driven staleness path (inapplicable — the example ADR ships `status: Accepted`) to a
  direct frontmatter splice; decision 3 gained a documented `os.Chdir` exception for `init`,
  which reads `os.Getwd()` directly instead of `g.configPath`; decision 5 and the acceptance
  test were corrected after finding `go test ./... -update` already fails on `internal/scaffold`
  pre-existing this ticket, unrelated to it — verification rescoped to `internal/cli`
- 2026-09-03 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-09-03 — IN REVIEW → REWORK: F1 blocking: stdout/stderr capture collapsed by SetOut/SetErr, see Review
- 2026-09-03 — REWORK → IN REVIEW: findings fixed
