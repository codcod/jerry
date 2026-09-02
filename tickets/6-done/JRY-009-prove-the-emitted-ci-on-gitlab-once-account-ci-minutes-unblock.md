---
id: JRY-009
title: Prove the emitted CI on GitLab once account CI minutes unblock
project: jerry
depends-on: []
spawned-by: [JRY-004]
impact: medium
complexity: low
cost: S
---

# JRY-009 — Prove the emitted CI on GitLab once account CI minutes unblock

## Outcome

We know, rather than believe, that `jerry init --forge gitlab`'s emitted pipeline actually runs
and passes on a real GitLab.com runner, and fails legibly on a deliberately broken document —
closing the half of JRY-004's real-forge proof that its own execution couldn't complete.

## Description

JRY-004 proved the GitHub half of DESIGN.md §6's scaffold contract end to end: an unmodified
`jerry init --forge github` scaffold passed its Actions run, and a copied/unfilled
`templates/adr-template.md` failed it with the exact defects named
(`id-format`/`placeholder`/`team-mismatch`/`date`). The GitLab half was attempted with the same
throwaway-repo approach (`nicos.ka/jerry-forge-proof-gitlab`, scaffolded, pushed, unmodified)
but its pipeline never left `pending` — no runner ever picked up either job, despite shared
runners showing enabled and online on the project. The pipeline page showed GitLab's identity-
verification banner; after the user completed identity verification, the pipeline (and a
follow-up push-triggered retry) still stayed `pending`, and `glab api .../pipelines` showed
`status: "pending"` with no assigned runner even after the `data-identity-verification-required`
flag flipped to `false`. This is an account-level CI-minutes/quota or review-hold issue, not a
defect in the emitted `.gitlab-ci.yml` template — the template's shape is identical in kind to
the GitHub one JRY-004 already proved (same install script, same `jerry validate`/`jerry index
--check` steps), so this ticket is a re-run of JRY-004's GitLab tasks once the account-level
blocker is actually gone, not new design work.

Scope: repeat JRY-004's Task 3/4 shape — scaffold a fresh throwaway GitLab repo (the old one,
`nicos.ka/jerry-forge-proof-gitlab`, is already marked for deletion — confirmed gone, `glab api
projects/nicos.ka%2Fjerry-forge-proof-gitlab` now 404s), push unmodified, confirm the pipeline
passes; add the same unfilled `adr-template.md` copy, push, confirm it fails legibly; record the
evidence in this ticket's Review; delete the repo. Before starting, confirm in the GitLab web UI
(logged in) that the account no longer shows a CI-minutes/verification hold — `glab`/`curl`
against the API could not surface that banner reliably during JRY-004's own attempt, so don't
trust a bare "verification: false" flag alone; watch a real pipeline run to completion before
concluding the blocker is gone.

**Refinement-time discovery (2026-09-02):** the account hold is confirmed gone — a throwaway
repo the user had already scaffolded, `nicos.ka/jerry-test-gitlab`
(`~/temp/arch-docs/jerry-test-gitlab`), ran its pipeline
([2814449017](https://gitlab.com/nicos.ka/jerry-test-gitlab/-/pipelines/2814449017)) to a clean
`success` — no `pending` hang. But that scaffold was cut with a stale `jerry` binary: its
`.gitlab-ci.yml` still does `go install github.com/codcod/jerry/cmd/jerry@v0.1.1`, not the
checksum-verified release-binary download JRY-003 shipped
(`internal/scaffold/templates/gitlab/.gitlab-ci.yml`), so that pass evidence proves the old
template, not the current one. The Implementation Plan below reuses that same repo/directory
(user-confirmed 2026-09-02) but re-runs `jerry init --forge gitlab` with a binary built off
current HEAD before pushing again. A sibling throwaway, `codcod/jerry-test-github`
(`~/temp/arch-docs/jerry-test-github`), was made in the same exploratory session and has the
same staleness, but GitHub is already proven current in JRY-004 (its evidence run post-dates
JRY-003) — that leftover directory/repo is out of scope for this ticket and is left untouched.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd /Users/nicos.karagieorgopulus/Projects/private/jerry
git checkout main
git checkout -b feat/JRY-009-gitlab-forge-proof
```

Root-path child (`path = "."`): tidy WIP commits into atomic ones before presenting, per the
project's commit policy. The only in-repo diff this ticket produces is a `PLAN.md` bookkeeping
edit (Task 5) — every other task acts on the throwaway GitLab repo, not on this repo.

### Prerequisite gate (hard)

`depends-on: []` — no other ticket gates pickup. Operationally: `glab auth status` must show an
active GitLab.com account (confirmed 2026-09-02: `nicos.ka`). The account-level CI-minutes/
identity-verification hold that blocked JRY-004 is confirmed cleared (2026-09-02: pipeline
[2814449017](https://gitlab.com/nicos.ka/jerry-test-gitlab/-/pipelines/2814449017) ran to
`success`, not `pending`) — re-check at pickup time only if that evidence is stale by then (a new
`pending`-forever pipeline would mean the hold is back).

### Confirmed design decisions (do not deviate without asking)

1. **Reuse the existing throwaway directory/repo** — `~/temp/arch-docs/jerry-test-gitlab`,
   GitLab project `nicos.ka/jerry-test-gitlab` — rather than creating a new one (user-confirmed
   2026-09-02).
2. **Re-run current, unmodified `jerry init --forge gitlab` in that directory before pushing
   again.** The existing pass evidence there used a stale scaffold (`go install ...@v0.1.1`, pre-
   JRY-003) and does not count toward this ticket's acceptance test — see the Description's
   refinement-time discovery note. Clear the directory's tracked files (keep `.git`) before
   re-running `init` so no stale files linger alongside the fresh output.
3. **The version pin is left as whatever the locally-built binary emits, not forced to a tag**
   (same as JRY-004 decision 3) — building at current HEAD gives a dirty/pseudo-version, which
   `scaffold.go`'s `replaceTokens` pins to `latest`, resolved to the real published tag at
   pipeline run time.
4. **The broken-document case copies jerry's own shipped template verbatim, unedited** (same as
   JRY-004 decision 5):
   `internal/scaffold/templates/common/templates/adr-template.md` into
   `teams/example-team/adr/0002-broken.md` with no field filled in.
5. **Scope is GitLab only.** The sibling leftover `~/temp/arch-docs/jerry-test-github` /
   `codcod/jerry-test-github` from the same exploratory session is untouched by this ticket —
   GitHub is already proven current in JRY-004.
6. **The throwaway GitLab repo is deleted once evidence is captured** (same disposal discipline
   as JRY-004): both the local directory and the `nicos.ka/jerry-test-gitlab` remote.
7. **No release is tagged and no jerry source code changes as part of this ticket.** The only
   diff landing in this repo is Task 5's `PLAN.md` row update.

### Tasks

#### Task 1 — Build the binary under test

At the repo root, on this feature branch: `just build`. Confirm `./jerry version` runs and
reports a dirty/pseudo-version off current HEAD (decision 3). This binary performs the `init`
call below — do not reuse the stale binary that produced the existing scaffold.

#### Task 2 — Re-scaffold and push the GitLab throwaway repo

```
cd ~/temp/arch-docs/jerry-test-gitlab
git rm -rf . && git clean -fdx        # clear tracked+untracked, keep .git
/path/to/jerry init --forge gitlab    # unmodified — no hand edits to the output
git add -A && git commit -m "jerry init (current template)"
git push origin main
glab ci status                        # watch to completion — success, not pending
```

#### Task 3 — Break the repo deliberately and re-confirm failure

Copy `internal/scaffold/templates/common/templates/adr-template.md` (from the jerry repo) to
`teams/example-team/adr/0002-broken.md` in the throwaway repo, unedited (decision 4). Commit and
push to `main`. Confirm the pipeline now fails, and capture the failing job's own output (it must
name the actual defects — placeholder/id mismatches — not fail opaquely).

#### Task 4 — Record the evidence

In this ticket's `## Review` (added when the ticket reaches `4-in-review/`, not now): the
pipeline URLs from Task 2 (pass) and Task 3 (fail), plus the relevant snippet of the failing
job's log output. Then delete the throwaway repo (decision 6):
`rm -rf ~/temp/arch-docs/jerry-test-gitlab` and `glab repo delete nicos.ka/jerry-test-gitlab
--yes` (or the web UI equivalent, if the token lacks delete scope — JRY-004 hit that on the
GitHub side).

#### Task 5 — Reconcile PLAN.md

`PLAN.md`'s "Filed so far" table, `forge-proof-gitlab` row: change `status` from `to do` to
`done`, matching the style of the `forge-proof` row's own update for JRY-004.

### Acceptance test

1. Task 2's GitLab pipeline reports `success`, watched to completion (not just non-`pending`),
   from an unmodified, current `jerry init --forge gitlab` output with no hand edits.
2. Task 3's re-run fails, and the failing job's log names the actual defects (not a generic
   non-zero exit with no explanation).
3. `just build`/`test`/`lint`/`docs-check` stay clean in this repo (only `PLAN.md` changed here).

### Docs update (mandatory when user-facing)

No user-facing surface changes — no template or code diff ships. `PLAN.md`'s
`forge-proof-gitlab` row (Task 5) is the only doc touched, and it is scratch/status bookkeeping,
not user-facing.

### Finish (mandatory)

1. Acceptance test green; `just build`/`test`/`lint`/`docs-check` clean.
2. `PLAN.md` updated per Task 5.
3. Write a summary (pipeline URLs, what was observed, confirmation the account hold is gone).
4. Suggested commit message: `docs(plan): mark forge-proof-gitlab verified end-to-end (JRY-009)`.
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; publish only per the project's commit policy (no push/MR without approval).
   `pickle ticket move JRY-009 in-review --reason "acceptance green"` and hand back.

## Review

### Evidence (GitLab — proven, current template)

- Re-scaffold + push (Task 2): reused `nicos.ka/jerry-test-gitlab`
  (`~/temp/arch-docs/jerry-test-gitlab`); cleared its stale (pre-JRY-003, `go install`) scaffold
  and re-ran unmodified `jerry init --forge gitlab` off current HEAD
  (`v0.1.1-68-gc9174b7`) — the emitted `.gitlab-ci.yml` grew from 981 to 2757 bytes and now
  installs via the checksum-verified release-archive download, not `go install`. Pushed
  (`ac27efe`). Pipeline
  [2814501697](https://gitlab.com/nicos.ka/jerry-test-gitlab/-/pipelines/2814501697) —
  **success**.
- Broken doc (Task 3): copied `internal/scaffold/templates/common/templates/adr-template.md`
  verbatim into `teams/example-team/adr/0002-broken.md`, unedited, committed (`005122f`, local
  pre-commit hook from an earlier `jerry hooks install` in that throwaway repo bypassed with
  `--no-verify` so the failure surfaces in the remote pipeline instead of blocking locally),
  pushed. Pipeline
  [2814503528](https://gitlab.com/nicos.ka/jerry-test-gitlab/-/pipelines/2814503528) —
  **failed**. The runner installs jerry via the checksum-verified download
  (`jerry_0.1.1_linux_amd64.tar.gz: OK`), then `jerry validate`'s own log names the real defects:
  ```
  teams/example-team/adr/0002-broken.md:3: error: frontmatter id must be ADR-NNNN, got "ADR-NNNN" (id-format)
  teams/example-team/adr/0002-broken.md:3: error: template placeholder "ADR-NNNN" was never filled in (placeholder)
  teams/example-team/adr/0002-broken.md:4: error: template placeholder "Short title of the decision" was never filled in (placeholder)
  teams/example-team/adr/0002-broken.md:9: error: frontmatter team "your-team" does not match folder "example-team" (team-mismatch)
  teams/example-team/adr/0002-broken.md:9: error: template placeholder "your-team" was never filled in (placeholder)
  teams/example-team/adr/0002-broken.md:11: error: date "YYYY-MM-DD" is not a valid ISO date (YYYY-MM-DD) (date)
  teams/example-team/adr/0002-broken.md:11: error: template placeholder "YYYY-MM-DD" was never filled in (placeholder)
  teams/example-team/adr/0002-broken.md:12: error: template placeholder "[alice, bob]" was never filled in (placeholder)
  teams/example-team/adr/0002-broken.md:37: error: template placeholder "**Option A** — why it was rejected" was never filled in (placeholder)
  ```
  Both jobs (`check-index`, `validate-docs`) failed for the same reason.
- Cleanup (Task 4, decision 6): remote `nicos.ka/jerry-test-gitlab` deleted via `glab repo delete
  nicos.ka/jerry-test-gitlab --yes`, confirmed gone (`glab api projects/nicos.ka%2Fjerry-test-gitlab`
  now 404s); local clone `~/temp/arch-docs/jerry-test-gitlab` removed (`rm -rf`).

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | docs-gap | fixed inline | Evidence section recorded the pass/fail pipelines but never stated the mandatory repo deletion (decision 6/Task 4) actually happened | independent audit of this ticket's own Review section (prior to this fix) | added the cleanup bullet above, confirming the remote 404 and local dir removal |

**Disposition summary:** 1 non-blocking (F1 → fixed inline). No blocking findings.

cost: estimated S, actual S — matched estimate; the account-hold risk that inflated JRY-004's
GitLab attempt did not recur.

## History

- 2026-09-02 — created (TO DO). source: review: JRY-004's own execution hit an account-level
  GitLab CI-minutes/identity-verification blocker that never resolved within that ticket's
  session; scoped down to GitHub-only proof and this ticket carries the deferred GitLab half
- 2026-09-02 — TO DO → READY: plan complete
- 2026-09-02 — READY → IN DEVELOPMENT: picked up
- 2026-09-02 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-09-02 — IN REVIEW → DONE: review clean; 1 non-blocking finding fixed inline
- 2026-09-02 — [PR #10](https://github.com/codcod/jerry/pull/10) opened against `main`; awaiting human merge
- 2026-09-02 — merged to main (PR #10, `af6ccf2`)
