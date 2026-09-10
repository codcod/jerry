# Review addendum — jerry project-specific rules

**Version 1** · written 2026-09-02 against `main` at `e8ac0c6` (JRY-001 and JRY-002 done,
DESIGN.md at v2)

Applies **on top of** the brine review protocol
(`.agents/skills/brine/resources/review-protocol.md`), keyed to that procedure's step numbers.
It never replaces it.

There is no overarching addendum layer: jerry is a single child-project at `path = "."` under
`layout = "in-tree"`, so the board and the code share one repository and one addendum covers
both surfaces. If a second child is ever registered, the bookkeeping rules under steps 7 and 9
below move up to a top-level `review_addendum` and the rest stays here.

**This file does not restate `AGENTS.md`.** Its "Conventions that reviews must enforce" section
is the source of truth for the conventions themselves, and a second copy would drift the first
time either is refined. What this file adds is the *audit procedure*: which conventions the test
suite already enforces (so the reviewer's remaining job is the residue), which have no
mechanical enforcement at all, and where jerry has actually been bitten.

## Step 1 — Load context (additions)

Read [`DESIGN.md`](DESIGN.md) — §3 (the charter), §4 (the schema and its lifecycles), §5 (the
rule catalogue), and §6 (the scaffold contract).

**Read §10 before anything else.** It is the list of claims this document has made that the code
does not honour, and it exists because v1 carried seven of them at once. A review must not
"verify" a claim §10 already retracts. If the branch **closes** a §10 row, deleting that row is
part of the ticket, not a follow-up — see step 7.

`layout = "in-tree"` with `path = "."` means the ticket and the code are in the same repository,
so the protocol's stale-ticket hazard is live on every review: read the ticket with
`git show main:tickets/4-in-review/JRY-NNN-*.md`, never from the feature branch's worktree.

## Step 2 — Implementation audit (additions)

Configured commands: `just build`, `just test`, `just lint`, `just docs-check`. Lint is
`gofmt -l` plus `go vet` and matches CI exactly — there is no `.golangci.yml`, so a review that
wants more than `vet` is asking for a ticket, not a finding.

1. **The dependency policy, as a grep.** `go.mod` stays at `cobra` + `pflag` + `yaml.v3` and the
   standard library (`AGENTS.md`). Read the file; do not recall it. jerry runs in CI images and
   pre-commit hooks, where every dependency is weight — and §3.3 promises it runs with no
   toolchain behind it at all.
2. **Which stream does it print to?** Nothing enforces this and the tree is currently
   inconsistent, which makes it the rule most worth a reviewer's attention. cobra's `cmd.Print*`
   helpers write to **stderr**; only `fmt.Fprintln(cmd.OutOrStdout(), …)` reaches stdout. This
   has already shipped as a defect once — `CHANGELOG.md` 0.1.0 records `Print*` breaking the git
   hook — and the fix was applied at the two call sites that failed (`internal/cli/cli.go`'s
   `version`, `internal/cli/index.go`'s path echo, both carrying a comment saying why), not as a
   rule. Everything else still uses `cmd.Print*`: `fmt --check`'s list of files needing
   formatting, `validate`'s summary line, `status`'s transition line, `new`'s created path.

   The reviewer's test: **anything a script, a hook or a pipe would consume goes to stdout.** A
   new leaf emitting machine-readable output through `cmd.Print*` is **blocking**. Progress and
   advice on stderr is defensible; the file list from a `--check` flag is not. The split itself
   is not yet decided project-wide — say so in the finding rather than inventing the rule, and
   note that `JRY-007` (command-level golden tests) is where it becomes mechanical.
3. **`write_safety_test.go` checks presence, not correctness.** It enforces that every leaf
   carries a `jerry/kind` annotation and that every `write` leaf has `--dry-run`, with a negative
   test proving the checker fires — so a missing annotation is a test failure, never a review
   comment. What it cannot check is whether a *new* leaf's classification is the **right** one. A
   command that mutates files is `write`; anything classed `read` or `other` that touches the
   disk is **blocking**.
4. **Findings accumulate** (§3.4). A new rule that returns on its first problem, or a command
   that stops at the first bad document, is **blocking** — the charter's reason is that a
   validator reporting one error per run turns a five-minute fix into five CI round-trips.
5. **A new rule's severity is a design decision, not a default.** §5's principle: a judgement a
   validator cannot make is a *warning*. Staleness is the worked example — failing CI on the
   calendar teaches people to falsify dates. A new check that makes a human judgement an error
   is **blocking**; a new check that makes a structural defect a warning is a finding.
6. **A new rule needs a false-positive story.** The placeholder rule shipped without one and the
   cheapest fix for a false positive became "delete the phrase from `jerry.yaml`", which disables
   the check repository-wide (§10.6, `JRY-006`). Ask of any new rule: what does an author do when
   it fires wrongly, and does that answer switch the rule off for everyone?
7. **The scaffold must validate clean the day it is written** — `internal/scaffold/scaffold_test.go`
   runs the real rules over a freshly emitted repository, so this is enforced. The residue for
   the reviewer: a ticket that adds a file to the scaffold must confirm the new file is inside
   what that test actually walks, and `jerry init` still needs no network (§6).

## Step 4 — Consistency audit (additions)

1. **Envelopes are versioned and additive only** (`AGENTS.md`) — `jerry.findings/1`
   (`internal/cli/output.go`), `jerry.index/1` (`internal/index/index.go`). The addition here:
   check the literal **and** its consumers together, and hold any new envelope
   (`jerry.related/1`, `jerry.corpus/1` are planned) to the same shape. Nothing mechanical
   catches a consumer, because jerry's consumers are other people's pipelines.
2. **Three surfaces assert what jerry does, and they have disagreed.** `DESIGN.md`, the code, and
   `CHANGELOG.md`. The `applies_to` claim was wrong in DESIGN §4.1 *and* in the 0.1.0 changelog
   entry, and the changelog is the one that outlives the review, because a released entry is not
   normally rewritten. Check the entry asserts only what shipped, in the tense it shipped in.
3. **A cited section that no longer says what the comment claims is `stale-xref`.** `AGENTS.md`
   requires comments to cite tickets and design sections inline (`DESIGN.md §4.2`,
   `JRY-004 decision 2`); those citations are references like any other and go stale the same
   way. Grep the branch for `DESIGN.md §` and confirm each still points at the claim it names —
   DESIGN.md v2 renumbered nothing, but it added §10 and moved revision history to §11.

## Step 4a — Documentation audit (jerry specifics)

The docs command is **`just docs-check`** (snowball; a broken include or xref fails it). The
manual under `docs/user-manual/` is real content, not a stub: a new command or flag that does not
appear there is **blocking** coverage.

- `README.md` is deliberately a short pointer to the manual — do not grow it back.
- `CHANGELOG.md`'s `## [Unreleased]` section is part of coverage, class `docs-gap`.
- The manual is still one page (`introduction.adoc`). Splitting it is `manual-restructure` in
  `PLAN.md` and belongs to that ticket — a review should not let an unrelated ticket restructure
  it opportunistically, and should not treat the single page as a reason to skip coverage.

## Step 5 — Findings (additions)

**A pre-existing defect found during a review is filed, not fixed inline.** JRY-002 is the
precedent: JRY-001's review found that the scaffold's version-pin fallback missed non-dirty
intermediate builds — real, adjacent, and outside that ticket's stated scope — so it became its
own ticket rather than a quiet extra commit on a release-verification branch. Follow it. The
`new ticket` disposition's promotion test still applies, and findings of one theme batch into one
ticket.

## Step 7 — Governing documents (additions)

jerry's governing documents are **`DESIGN.md`** (the design of record), **`AGENTS.md`**
(conventions), **`PLAN.md`** (the provisional ticket decomposition) and **`CHANGELOG.md`**.

**`DESIGN.md` carries a version stamp on line 3.** If the review changed what it asserts: correct
the section, bump the stamp, and add a line to §11 Revision history. If the branch closed a row
in **§10 Known divergences**, delete that row in the same commit — a divergence table that still
lists something fixed is the same defect it was written to prevent, one layer up.

This rule is not precautionary here. DESIGN.md v1 asserted seven things Phase 1 did not do —
`applies_to` validated, no toolchain needed in CI, `schema_version` read, repositories owning no
rules, lifecycles enforced, the placeholder rule sound, findings always reported — and one of
them propagated into a released changelog entry. They are §10 now precisely so that a review has
a list to check against instead of a document to re-read.

**`PLAN.md` is keyed by slug, not by ticket id**, and its *Filed so far* table maps slug → id →
status. A review that concludes a ticket updates that row. This is the failure that produced the
convention: v1 of the file numbered its rows `JRY-001`…`JRY-030`, JRY-002 was then filed as a
spawned defect rather than as the file's second row, and every `depends-on` after row 1 silently
pointed at the wrong work.

## Step 8 — Impact sweep (addition)

The sweep reads `PLAN.md`'s build-step tables as well as `tickets/2-ready/` and
`tickets/1-to-do/`. Most downstream work is described only there — the filed tickets carry no
`depends-on:` frontmatter yet, deliberately, since hard dependencies are human-approved — so a
sweep restricted to ticket frontmatter would find almost nothing to check.

## Step 9 — Finish (additions)

- **Two commit vocabularies.** Code on `feat/JRY-NNN-<slug>` takes Conventional Commits with the
  ticket id **in brackets at the end of the subject** (`feat(cli): add supersede (JRY-004)`).
  Ticket and board bookkeeping takes `board: JRY-NNN <verb phrase>` with no trailing id, and
  lands on `main`, never on the feature branch. Using the code form for bookkeeping is the
  common error.
- Root-path child: tidy the WIP commits into a few atomic, correctly typed commits and default to
  **keeping** that history rather than squashing.
- `pickle hooks install` is per-clone and bypassable, so run the protocol's
  `origin/main...HEAD` check for leaked `tickets/` paths by hand regardless of whether the hook
  is armed.
- **`just dist-check` before a tag**, and note that a release is currently a manual procedure
  against `RELEASING.md` — `release-automation` in `PLAN.md` is the ticket for that.

## Revision history

- **v1** (2026-09-02) — Written after DESIGN.md v2 recorded seven divergences between the design
  of record and the shipped code. Step 2 rule 2 (output stream) and step 7 (governing documents,
  §10 row deletion) are the two rules earned from real failures rather than from principle.
