---
id: JRY-013
title: Harden validate --diff: fix corpus/git path mismatch and base-ref handling
project: jerry
depends-on: [JRY-007]
spawned-by: []
family: JRY-011
impact: high
complexity: medium
cost: M
---

# JRY-013 — Harden validate --diff: fix corpus/git path mismatch and base-ref handling

## Outcome

`jerry validate --diff` reports every finding on the changed paths it actually has, on any
repository layout, base ref, and clone depth CI actually produces — instead of silently
discarding all of them and exiting 0 whenever `jerry.yaml` is not at the git root.

## Description

**The live defect, confirmed in the code (DESIGN.md §10 item 6, and the worst-ranked
divergence there: "a validator that passes silently is worse than one that is absent"):**
`internal/cli/validate.go`'s `changedFiles` runs `git -C <root> diff --name-only
<base>...HEAD`, which always emits paths relative to the git top-level regardless of `-C`.
`onlyIn` then filters findings by `changed[finding.Path]`, where `finding.Path` is
corpus-root-relative (`internal/doc/corpus.go`'s `Load` computes it via `filepath.Rel` against
`cfg.Root`, the `jerry.yaml` directory — `internal/config/config.go`). When the corpus root is
not the git root, every finding's path fails to match every changed-file path, every finding is
filtered out, and `validate --diff` exits 0 having checked nothing. Nothing in the current code
path guards against this; it is real, not hypothetical.

Fix: make the two path spaces agree before comparing — either rewrite `finding.Path` to be
git-root-relative before filtering, or rewrite the changed-files list to be corpus-root-relative
using the corpus root's offset from the git top-level (`git rev-parse --show-toplevel`).

**Then, per PLAN.md's scope for this row:** base-ref autodetection (so `--diff` doesn't require
the caller to already know the right ref), shallow-clone behaviour (CI checkouts are commonly
`--depth 1`), and detached-HEAD behaviour (the state every CI job runs in).

**Re-verified against the current code during refinement (2026-09-07):** a missing or
unreachable base ref (no `origin/main` ref at all, or a shallow clone with no common ancestor)
does **not** silently produce an empty diff today — `git diff --name-only base...HEAD` exits
128 and `changedFiles` already wraps that into a returned error, which `validate`'s `RunE`
propagates to a non-zero process exit. The real gap is that the wrapped message is opaque
(`exec`'s bare `exit status 128`, dropping git's own `fatal: ...` reason from stderr) rather
than actually silent. Detached HEAD needs no fix either — `git diff --name-only A...HEAD`
resolves `HEAD` the same way regardless of attachment; verified locally. So this row's real,
remaining work is: the path-mismatch fix (above, still live and unverified-safe by any existing
code), a clearer failure message, and base-ref autodetection as a genuine convenience feature —
not fixing a silent-pass that turned out not to exist.

## Implementation Plan

### 0. Feature branch (mandatory)

```
git checkout main
git checkout -b feat/JRY-013-diff-hardening
```

Work happens in this repo (`layout = "in-tree"`, `project: jerry`, `path = "."`). Commit
locally as you go; publish only per the commit policy (no push / no MR without approval — tidy
WIP commits into atomic ones before presenting, since this is a root-path child).

### Prerequisite gate (hard)

JRY-007 (golden-test harness) is in `6-done/` and merged to `main` — confirmed
(`tickets/6-done/JRY-007-...md`'s History ends `merged to main`). Working tree clean, no other
ticket occupying `3-in-development/` or `4-in-review/` for `jerry` (WIP limit 1 each).

### Confirmed design decisions (do not deviate without asking)

1. **The corpus/git offset is resolved with `git -C <root> rev-parse --show-prefix`, never
   manual `filepath.Rel` arithmetic against `--show-toplevel`.** `rev-parse --show-prefix`
   computes the offset entirely inside git's own view of the tree, so it can't diverge from
   git's own path output the way manual `Abs`/`Rel` arithmetic can — confirmed locally that
   macOS aliases `/tmp` to `/private/tmp` (git resolves the symlink in `--show-toplevel`,
   `filepath.Abs` does not), which would otherwise corrupt the computed offset for exactly the
   `t.TempDir()`-backed fixtures this ticket's own tests rely on.
2. **A changed path that falls outside the corpus root's offset is dropped from the `changed`
   map, silently.** Such a path can never equal a corpus-relative `finding.Path`, so this is
   not a re-introduction of the bug — it's the correct outcome for a change to a file the
   corpus doesn't even contain.
3. **`changedFiles`'s error wraps `(*exec.ExitError).Stderr`, not just the exit status, when
   present.** `exec.Cmd.Output()` already populates it when `Stderr` was left `nil`; today's
   code discards it and reports only `exec`'s generic `exit status 128`.
4. **Base-ref autodetection only overrides the flag's default; it never overrides an explicit
   `--base`.** Detected via `cmd.Flags().Changed("base")`. When not explicitly set, and
   `$GITHUB_BASE_REF` (set by GitHub Actions on `pull_request`-triggered runs) is non-empty,
   use `origin/<GITHUB_BASE_REF>` in place of the literal `origin/main` default — the one other
   env-driven convention in this codebase (`internal/forge/github.go`'s `NewGitHubFromEnv`) is
   the precedent for reading GitHub Actions' env vars directly rather than adding a flag.
5. **`comment`'s call to the same `changedFiles` (`internal/cli/comment.go:73`) inherits the
   path-mismatch and clearer-error fixes for free, since it calls the identical function** —
   this is the intended shared-root-cause fix, not scope creep. `comment`'s own `--base` flag
   and autodetection are untouched; out of scope for this ticket.

### Tasks

#### Task 1 — Fix the corpus/git path mismatch and clarify `changedFiles`'s failure in `internal/cli/validate.go`

Replace `changedFiles` (currently lines 80–92) with a version that:
- Resolves the corpus root's offset from the git top-level via
  `exec.Command("git", "-C", root, "rev-parse", "--show-prefix").Output()`, trimmed of
  surrounding whitespace (empty string when `root` already is the git top-level).
- Runs the existing `git -C root diff --name-only base...HEAD`, and on error, uses
  `errors.As` to extract `*exec.ExitError`; when `.Stderr` is non-empty, wrap that
  (trimmed) text into the returned error instead of the bare `%w` exit-status message.
- For each non-empty output line: if the offset is `""`, keep the line as-is; otherwise use
  `strings.CutPrefix(line, offset)` and keep the ok-case result, dropping (not erroring on)
  any line CutPrefix reports `false` for (decision 2).

Add the `"errors"` import (new); `strings.CutPrefix` needs no new import (`strings` is already
imported). No change to `onlyIn` — decision 1's fix makes the map it's given already
corpus-relative.

#### Task 2 — Base-ref autodetection in `validateCmd`

Add a small helper next to `changedFiles`:

```go
// resolveDiffBase returns the base ref to diff against: the caller's
// explicit --base when given, otherwise origin/$GITHUB_BASE_REF (GitHub
// Actions' pull_request env var) when set, otherwise the flag's own
// default — never silently guessing when a human already said what they want.
func resolveDiffBase(cmd *cobra.Command, base string) string {
	if cmd.Flags().Changed("base") {
		return base
	}
	if prBase := os.Getenv("GITHUB_BASE_REF"); prBase != "" {
		return "origin/" + prBase
	}
	return base
}
```

In `validateCmd`'s `RunE`, inside the `if diffOnly` block, call
`changed, err := changedFiles(cfg.Root, resolveDiffBase(cmd, base))` instead of passing `base`
directly. Add the `"os"` import. Update the `--base` flag's help text to mention the
autodetection (`"base ref for --diff (autodetected from GITHUB_BASE_REF when not set explicitly)"`).

#### Task 3 — Test coverage in `internal/cli`

New file `internal/cli/validate_test.go`:
- `TestChangedFilesRewritesToCorpusRoot` — build a `t.TempDir()` git repo with the corpus
  nested under a `docs/` subdirectory (mirroring `nestedCorpusFixture` from Task 4), commit a
  base tagged `refs/remotes/origin/main`, commit one changed file under `docs/`, and assert
  `changedFiles(corpusRoot, "origin/main")` returns that file keyed by its corpus-relative
  path — the direct regression test for the live defect.
- `TestChangedFilesMissingBaseFailsClearly` — a repo with no `origin/main` ref; assert
  `changedFiles` returns a non-nil error whose message contains `"unknown revision"` (git's
  actual wording, confirmed by running `git diff --name-only origin/main...HEAD` locally
  against a ref-less repo during refinement) — pins decision 3's clearer message, not just a
  non-nil error.
- `TestChangedFilesDetachedHead` — same fixture pattern, `git checkout --detach` before
  diffing; assert no error and the expected changed file — locks in behaviour already correct
  today so it can't regress silently.
- `TestResolveDiffBase` — table test over: explicit `--base` wins over `GITHUB_BASE_REF`;
  `GITHUB_BASE_REF` set and `--base` not given → `origin/<value>`; neither set → the flag's
  `origin/main` default. Build a bare `*cobra.Command` with just a `base` string flag
  registered and parsed (`cmd.Flags().StringVar(&base, "base", "origin/main", "")`,
  `cmd.ParseFlags(tc.args)`) — no need to construct the full `validateCmd` tree.

In `internal/cli/fixtures_test.go`, add:
- `nestedCorpusFixture(t *testing.T) string` — scaffolds the corpus at `<tempdir>/docs` (not
  the git top-level), `gitInit`s the parent temp dir, commits the clean scaffold tagged
  `refs/remotes/origin/main` (reusing `gitCommit` + the `update-ref` line from
  `gitCommentFixture`), perturbs the example ADR exactly as `dirtyFixture` does (extract that
  perturbation into a shared `perturbExampleADR(t *testing.T, root string)` helper called by
  both `dirtyFixture` and this new fixture, since it would otherwise be duplicated verbatim),
  commits that change, and returns the `docs` corpus root.
- A `TestNestedCorpusFixtureContract` mirroring `TestDirtyFixtureContract`'s shape (exactly one
  error, one warning) — the same fixture-pins-its-own-property convention.

In `internal/cli/golden_test.go`, add to `goldenCases`:

```go
{
    name:     "validate-diff-nested-corpus-root",
    leafPath: "jerry validate",
    fixture:  nestedCorpusFixture,
    args:     []string{"validate", "--diff", "--base", "origin/main"},
},
```

Run `go test ./internal/cli/... -run TestGolden -update` once the fix is in place, then
**read** `internal/cli/testdata/golden/validate-diff-nested-corpus-root.json` before committing
it — it must show `"failed": true` and the one-error/one-warning text `dirtyFixture`'s contract
guarantees; a clean/empty result here means the fix regressed, not that the golden file is
correct by virtue of being generated.

### Acceptance test

1. `just test` — full suite green, including the three new `validate_test.go` cases, the new
   `TestNestedCorpusFixtureContract`, and the new golden case.
2. `just lint` clean (`gofmt`, `go vet`).
3. `just docs-check` clean.
4. Manual smoke, from a scratch repo, of the exact defect this ticket closes:
   ```
   mkdir -p /tmp/jry013-smoke/docs && cd /tmp/jry013-smoke
   git init -q && git -c user.email=a@b.com -c user.name=a commit -q --allow-empty -m init
   (cd docs && /path/to/jerry init --forge github --version test)   # or scaffold.Run equivalent via `jerry init`
   git add -A && git -c user.email=a@b.com -c user.name=a commit -q -m base
   git update-ref refs/remotes/origin/main HEAD
   # perturb docs/teams/example-team/adr/0001-...md the same way dirtyFixture does, then:
   git add -A && git -c user.email=a@b.com -c user.name=a commit -q -m change
   jerry validate --diff --config docs/jerry.yaml
   ```
   Expect a non-zero exit with the introduced error printed — not `0 problem(s)`. Before this
   fix, this exact layout is DESIGN.md §10 item 6's silent pass.
5. `git diff --name-only origin/main...HEAD` against a repo with no `origin/main` ref, run
   through `jerry validate --diff`, exits non-zero with the git failure reason visible in the
   printed error — not the bare `exit status 128`.

### Docs update (mandatory when user-facing)

- `DESIGN.md` §10: remove divergence row `6` from the table (the behaviour it names is now
  correct).
- `DESIGN.md` §11: add a `**Version 2.9**` entry, following the exact style of 2.1–2.8 —
  JRY-013 closed divergence 6 (§3.4/§5/§10 vs. `validate --diff`): `changedFiles` now rewrites
  git's repository-relative changed-file paths to the corpus-relative form findings use
  (`git rev-parse --show-prefix`), so a `jerry.yaml` below the git root no longer discards every
  finding; a missing/unreachable base ref's failure message now carries git's own reason; and
  `--base`, when not explicitly given, autodetects from `GITHUB_BASE_REF` on GitHub Actions
  `pull_request` runs. The resolved row was removed from §10's table.
- `CHANGELOG.md`, `[Unreleased]` → `### Fixed`: one entry stating `jerry validate --diff` no
  longer silently discards every finding when `jerry.yaml` is not at the git repository's root,
  now autodetects its base ref from `GITHUB_BASE_REF` in GitHub Actions `pull_request` runs when
  `--base` is not given, and reports git's actual failure reason when the base ref cannot be
  resolved.

### Finish (mandatory)

1. Acceptance test green (all 5 steps above); `just build`, `just test`, `just lint`,
   `just docs-check` clean.
2. Docs updated per the section above and registered (`DESIGN.md` §11 entry, `CHANGELOG.md`
   line).
3. Write the summary (files touched, decisions made, anything deferred) and hand back.
4. Suggested commit message:

   ```
   fix(cli): rewrite validate --diff's changed-file paths to the corpus root (JRY-013)

   changedFiles compared git-root-relative paths against corpus-root-relative
   finding paths, so any repo with jerry.yaml below the git root silently
   discarded every finding under --diff. Rewrite via `git rev-parse
   --show-prefix`, surface git's actual failure reason instead of a bare exit
   status, and autodetect --base from GITHUB_BASE_REF on GitHub Actions
   pull_request runs when not given explicitly.
   ```

5. Tidy WIP commits into a small number of atomic commits before presenting (root-path child).
6. Commit locally; present the commit message; publish only after user approval (fetch-check
   `origin/main` is not behind, push, open the MR) — merging is always the human's.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's cross-cutting row
  `diff-hardening`, next in the cross-cutting queue since its dependency (`golden-tests`,
  JRY-007) is done and the underlying defect is already live and silent.
- 2026-09-07 — TO DO → READY: plan complete
- 2026-09-07 — READY → IN DEVELOPMENT: picked up
