---
id: JRY-004
title: Prove the emitted CI on a real forge with an unmodified scaffold
project: jerry
depends-on: [JRY-003]
spawned-by: []
impact: high
complexity: medium
cost: M
---

# JRY-004 — Prove the emitted CI on a real forge with an unmodified scaffold

## Outcome

We know — rather than believe — that `jerry init`'s output works: a throwaway repository
scaffolded on GitHub and one on GitLab both pass their emitted pipelines unmodified, and both
fail when a deliberately broken document is added, with the failure legible in the job output.
Both throwaway repos are deleted once that evidence is captured in this ticket's `## Review` —
the deliverable is the recorded evidence, not any lasting repository or code change.

## Description

This was `PLAN.md`'s original second row and went missing: the id it expected was taken by
JRY-002 (an unplanned defect spawned by JRY-001's review) and the row was never refiled. It is
the only thing that can tell us the scaffolded CI works.

The existing tests assert the workflow file's *content* and that `actionlint` accepts it.
Neither executes it. Everything that only fails at runtime is therefore unverified: a runner
image that cannot reach the network, a GitLab runner without the tooling the pipeline assumes,
a `rules:`/`on:` block that never triggers, a permissions default that blocks a step,
`jerry index --check` disagreeing with the index `init` just generated on a real checkout.

Scope:

1. Scaffold a throwaway repo for each forge with an unmodified `jerry init`, push, confirm both
   pipelines pass with no hand edits.
2. Add a deliberately broken document — the useful case is a copied, unfilled template, since
   that is the defect DESIGN.md §5 calls the most common — and confirm each pipeline fails, with
   the finding legible in the job output.
3. Record what was actually run and observed in the ticket's Review section. The deliverable is
   evidence, not code.

**Sequencing.** JRY-003 (checksum-verified binary download, replacing `go install`) is now
`6-done/` and merged to `main` (PR #6, `4593316`), which removes the toolchain and module-proxy
failure modes before this ticket goes looking for them. Recorded as a hard `depends-on:`
(user-confirmed 2026-09-02) — already satisfied, so it gates nothing at pickup; it documents the
real coupling for the board/audit rather than leaving it in prose.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd /Users/nicos.karagieorgopulus/Projects/private/jerry
git checkout main
git checkout -b feat/JRY-004-real-forge-ci-proof
```

Root-path child (`path = "."`): tidy WIP commits into atomic ones before presenting, per the
project's commit policy. The only in-repo diff this ticket produces is a `PLAN.md` bookkeeping
edit (Task 6) — every other task acts on the two throwaway repos, not on this repo.

### Prerequisite gate (hard)

`depends-on: [JRY-003]` — already satisfied: JRY-003 is in `tickets/6-done/` and merged to
`main` (PR #6, `4593316`). No other ticket needs to land first. Operationally, both forge CLIs
must be authenticated before Task 1/2 start — confirmed already true in this environment:
`gh auth status` shows an active `codcod` GitHub account, `glab auth status` shows an active
`nicos.ka` GitLab.com account. Re-check both at pickup time in case the session changed; if
either is missing, stop and ask the human to authenticate rather than improvising a workaround.

### Confirmed design decisions (do not deviate without asking)

1. **Both throwaway repos are disposable and their namespace is arbitrary** (user-confirmed
   2026-09-02: "does it matter? should this be random"). Use whichever account each CLI is
   already authenticated as — `codcod` on GitHub, `nicos.ka` on GitLab.com — with a
   throwaway-looking name (e.g. `jerry-forge-proof-github`, `jerry-forge-proof-gitlab`); no new
   one-time setup is needed on either forge.
2. **Both repos are deleted once the evidence is captured** in this ticket's `## Review`
   (user-confirmed 2026-09-02). Nothing durable is left behind on either forge.
3. **The version pin is left as whatever the locally-built binary emits, not forced to a tag.**
   Building at `main` HEAD (past `v0.1.1`, untagged) gives a dirty/pseudo-version, which
   `scaffold.go`'s `replaceTokens` (JRY-002) pins to `latest` — the emitted script then resolves
   `latest` to the real published tag (`v0.1.1`) at run time. This ticket proves the
   install/verify *mechanics* work on a real runner; it does not require cutting a new release
   first, and cutting one is a separate, human-approved publish action out of scope here.
4. **The trigger path exercised is a direct push to each repo's default branch (`main`).** Both
   templates already run their validate job on that event (`on: push: branches: [main]` on
   GitHub; `rules: - if: '$CI_COMMIT_BRANCH == "main"'` on GitLab) — a pull/merge-request run is
   a different trigger path and is not needed to prove the scaffold contract.
5. **The broken-document case copies jerry's own shipped template verbatim, unedited**:
   `internal/scaffold/templates/common/templates/adr-template.md` into
   `teams/example-team/adr/0002-broken.md` with no field filled in. This is the exact "copied,
   unfilled template" defect DESIGN.md §5 names as the most common real one. Expect it to trip
   more than one rule at once (filename/id disagreement, the placeholder scan) — either failure
   surfacing legibly in the job output satisfies this ticket; no need to isolate a single rule.
6. **No release is tagged and no jerry source code changes as part of this ticket.** The only
   diff landing in this repo is Task 6's `PLAN.md` row update.

### Tasks

#### Task 1 — Build the binary under test

At the repo root, on this feature branch (HEAD includes JRY-003): `just build`. Confirm
`./jerry version` runs and reports a dirty/pseudo-version (decision 3). This binary performs
every `init` call below — do not use any previously built binary.

#### Task 2 — Scaffold and push the GitHub throwaway repo

```
gh repo create codcod/jerry-forge-proof-github --public --clone
cd jerry-forge-proof-github
/path/to/jerry init --forge github        # unmodified — no hand edits to the output
git add -A && git commit -m "jerry init"
git push origin main
gh run watch                              # or: gh run list; confirm the docs workflow succeeds
```

#### Task 3 — Scaffold and push the GitLab throwaway repo

```
glab repo create jerry-forge-proof-gitlab --public --clone
cd jerry-forge-proof-gitlab
/path/to/jerry init --forge gitlab        # unmodified — no hand edits to the output
git add -A && git commit -m "jerry init"
git push origin main
glab ci status                            # or the web UI; confirm the pipeline succeeds
```

#### Task 4 — Break each repo deliberately and re-confirm failure

In each of the two clones from Task 2/3: copy
`internal/scaffold/templates/common/templates/adr-template.md` (from the jerry repo) to
`teams/example-team/adr/0002-broken.md`, unedited (decision 5). Commit and push to `main` in
each. Confirm both pipelines now fail, and capture the failing job's own output (it must name
the actual defect — a placeholder/id mismatch — not fail opaquely).

#### Task 5 — Record the evidence

In this ticket's `## Review` (added when the ticket reaches `4-in-review/`, not now): the two
run/pipeline URLs from Task 2/3 (pass) and Task 4 (fail) for each forge, plus the relevant
snippet of each failing job's log output. Then delete both repos (decision 2):
`gh repo delete codcod/jerry-forge-proof-github --yes` and
`glab repo delete jerry-forge-proof-gitlab --yes` (or the web UI equivalent).

#### Task 6 — Reconcile PLAN.md

`PLAN.md`'s "Filed so far" table, `forge-proof` row: change `status` from `to do` to `done`,
matching the style of the `ci-binary-install` row's own update for JRY-003.

### Acceptance test

This ticket's acceptance test *is* Tasks 1–5, run for real against the live GitHub and GitLab
APIs — there is no separate check beyond re-reading the captured run URLs and confirming:

1. Task 2's GitHub Actions run and Task 3's GitLab pipeline both report success, from an
   unmodified `jerry init` output, with no hand edits.
2. Task 4's re-run of both fails, and each failing job's log names the actual defect (not a
   generic non-zero exit with no explanation).
3. `just build`/`test`/`lint`/`docs-check` stay clean in this repo (only `PLAN.md` changed here).

### Docs update (mandatory when user-facing)

No user-facing surface changes — no template or code diff ships. `PLAN.md`'s `forge-proof` row
(Task 6) is the only doc touched, and it is scratch/status bookkeeping, not user-facing.

### Finish (mandatory)

1. Acceptance test green (evidence captured for both forges, both pass and fail cases); `just
   build`/`test`/`lint`/`docs-check` clean.
2. `PLAN.md` updated per Task 6.
3. Write a summary (both throwaway repo run URLs before deletion, what was observed, anything
   deferred).
4. Suggested commit message: `docs(plan): mark forge-proof verified end-to-end (JRY-004)`.
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; publish only per the project's commit policy (no push/MR without approval).
   `pickle ticket move JRY-004 in-review --reason "acceptance green"` and hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new
- 2026-09-02 — TO DO → READY: plan complete
- 2026-09-02 — READY → IN DEVELOPMENT: picked up
- 2026-09-02 — Task 2 done: `codcod/jerry-forge-proof-github` scaffolded (unmodified `jerry
  init --forge github`), pushed, GitHub Actions run
  [33621310129](https://github.com/codcod/jerry-forge-proof-github/actions/runs/33621310129)
  succeeded — no hand edits
- 2026-09-02 — Task 3 blocked: `nicos.ka/jerry-forge-proof-gitlab` scaffolded and pushed
  (unmodified `jerry init --forge gitlab`), but its pipeline
  ([2812901714](https://gitlab.com/nicos.ka/jerry-forge-proof-gitlab/-/pipelines/2812901714))
  stays `pending` indefinitely — no runner ever picks up either job, despite shared runners
  showing enabled and online on the project. The pipeline page itself shows GitLab's identity-
  verification banner: this GitLab.com account needs identity verification (payment method/
  phone) before shared-runner CI minutes unlock. Not something this session can resolve.
  User decision: wait and retry later rather than verify now or drop GitLab from scope. Pausing
  here — both throwaway repos left live (GitHub evidence captured, GitLab not yet), feature
  branch `feat/JRY-004-real-forge-ci-proof` untouched, ticket stays in `3-in-development/`
