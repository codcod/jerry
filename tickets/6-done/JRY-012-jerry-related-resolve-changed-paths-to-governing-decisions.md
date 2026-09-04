---
id: JRY-012
title: jerry related: resolve changed paths to governing decisions
project: jerry
depends-on: [JRY-011]
spawned-by: []
family: JRY-011
impact: high
complexity: medium
cost: M
---

# JRY-012 — jerry related: resolve changed paths to governing decisions

## Outcome

`jerry related --paths <files>` answers "which decisions govern the code I am about to
touch," offline, from the terminal. This is the first command that reads the `applies_to`
index rather than just validating it — the landing point DESIGN.md §1/§7.2 calls the actual
point of the project, and it is what `bot` (step 2, unfiled) will call in a merge request.

## Description

DESIGN.md §7.2: "`jerry related --paths <changed files>` — which decisions govern the code I
am about to touch. Requires `applies_to`, which Phase 1 already accepts." Phase 1 accepts and
(as of JRY-005) validates the field's shape; JRY-011 builds the matching engine. This ticket is
the first consumer: given a set of changed paths, resolve them against every decision's
`applies_to` (via JRY-011's matcher) and return the governing decisions, applying JRY-011's
precedence rule when several decisions match one path.

No command named `related` exists yet (confirmed against the full `internal/cli/*.go` command
list: `version`, `fmt`, `hooks`, `index`, `schema`, `init`, `validate`, `status`, `new`,
`supersede` — no `related.go`). Follow the conventions the existing commands already set:
`--format text|json` (mirroring `validate.go`'s `--format text,json,sarif,junit` and its
`--json` shorthand), and a versioned JSON envelope on the model of
`internal/cli/output.go`'s `FindingsEnvelopeSchema = "jerry.findings/1"` — this command's
envelope is `jerry.related/1`.

**Hard dependency on JRY-011** (applies_to matching): this command has nothing to query until
the matcher exists. PLAN.md states this as a sequencing rule for build step 1 ("`related` lands
after applies-to-match"), which is the prior sign-off for the `depends-on:` below; flagging it
here rather than treating it as silently pre-approved.

Out of scope: posting anywhere (that's `bot`, step 2, unfiled — a soft coupling, not a
dependency) and service-id resolution (still deferred to `owners`).

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/JRY-012-related
```

WIP commits locally as you go. Publish only per the project's commit policy (no push/MR
without explicit user approval); tidy WIP into atomic commits before presenting (root-path
child, rules §0).

### Prerequisite gate (hard)

JRY-011 (`internal/match`) must be in `6-done/` and merged to `main` before this starts —
`related` has nothing to query otherwise. Currently `2-ready/`, not yet picked up: **stop and
confirm JRY-011 shipped and merged before starting this ticket's tasks.**

### Confirmed design decisions (do not deviate without asking)

1. **`--paths` is a repeatable flag (`StringSliceVar`), required, not positional args.**
   Matches DESIGN.md §7.2's literal `jerry related --paths <changed files>` and the existing
   repeated-value convention (`new adr --deciders`, `new sd --authors`). Empty `--paths` is a
   usage error, not "no matches."
2. **`--format text|json`**, mirroring `validate.go`'s `-f` shorthand and the `g.json` global
   override (`if g.json { format = "json" }`, same as `validateCmd`). JSON envelope constant
   `RelatedEnvelopeSchema = "jerry.related/1"`, shaped on `FindingsEnvelopeSchema`
   (`internal/cli/output.go`).
3. **No pass/fail semantics.** `related` is a query, not a gate (unlike `validate`): exit 0
   whenever the corpus loads and `--paths` is non-empty. A queried path with no matching
   decision renders no line in `text` and an empty `decisions` array in `json` — silence is
   the expected shape for "nothing governs this," not an error. Only bad input (empty
   `--paths`, a corpus load failure) exits non-zero.
4. **Renders `internal/match.Resolve`'s order verbatim, one call per queried path.** `related`
   does not re-rank or collapse to a single winner — JRY-011's `Resolve` already returns
   most-specific-first; this command is a thin render layer over it.
5. **One new file, `internal/cli/related.go`.** No separate `related_output.go`: `output.go`
   exists to share SARIF/JUnit machinery across `validate`'s four formats, and two formats
   sharing only a JSON encoder don't earn a second file (mirrors `validate.go` itself, which
   keeps its own flag parsing and `RunE` in one place).

### Tasks

#### Task 1 — `internal/cli/related.go`
`relatedCmd(g *globals) *cobra.Command`: `Use: "related"`, `Args: cobra.NoArgs`,
`Annotations: map[string]string{kindKey: kindRead}`. Flags: `--paths` (StringSliceVar, nil
default), `--format`/`-f` (default `"text"`). `RunE`: reject empty `--paths`; `openCorpus(g)`;
for each path, `match.Resolve(corpus, path)`; assemble into:
```go
type relatedEnvelope struct {
    Schema  string          `json:"schema"`
    Results []relatedResult `json:"results"`
}
type relatedResult struct {
    Path      string            `json:"path"`
    Decisions []relatedDecision `json:"decisions"`
}
type relatedDecision struct {
    ID      string `json:"id,omitempty"`
    Title   string `json:"title"`
    Path    string `json:"path"`
    Pattern string `json:"pattern"`
}
```
Text renderer: for each `relatedResult` with a non-empty `Decisions`, print the queried path
then one indented line per decision (id/title + doc path); a result with zero decisions prints
nothing (decision 3). JSON renderer: the full envelope, one path entry per queried path
(including empty-`decisions` ones), `json.Encoder` with `SetIndent`/`SetEscapeHTML(false)`
matching `output.go`'s existing encoders.

#### Task 2 — Wire the command in
Add `relatedCmd(g)` to the `root.AddCommand(...)` list in `internal/cli/cli.go` (currently
lines 61–71).

#### Task 3 — Fixture + golden case
`internal/cli/fixtures_test.go`: add `relatedFixture(t *testing.T) string`, following
`dirtyFixture`'s exact technique (string-replace the example ADR's frontmatter marker) to set
`applies_to: ["teams/example-team/"]` on the scaffolded example ADR — a directory-prefix entry
(JRY-011 decision 2), so the golden case exercises a real match, not just validation of the
"no matches" path.
`internal/cli/golden_test.go`: add a `goldenCases` entry, `name: "related-match"`,
`leafPath: "jerry related"`, `fixture: relatedFixture`,
`args: []string{"related", "--paths", "teams/example-team/src/db.go", "--paths", "docs/readme.md"}`
(one path under the `applies_to` prefix, one outside it — exercises both the match and the
silent-no-match branches in one golden file). `TestGoldenCoversEveryLeaf` requires this case to
exist once `related` is wired in, or that test fails.

### Acceptance test

```
go test ./internal/cli/... -run TestGolden -v
go test ./internal/cli/... -run TestGoldenCoversEveryLeaf -v
just test
just lint
```
`related-match` golden case passes on first run only after `-update` regenerates its fixture
(`just test-update`, then re-run `just test` clean) — same convention every other golden case
follows.

### Docs update (mandatory when user-facing)

Add a `CHANGELOG.md` `[Unreleased] / Added` line for `jerry related --paths <files>
[--format text|json]` — the first user-facing command DESIGN.md §7.2 describes, so this is the
first entry that makes Phase 2 observable. No user-manual page yet: `docs/user-manual.adoc`
has only `introduction.adoc` so far — adding a per-command reference structure is out of scope
for this ticket (nothing else has one either).

### Finish (mandatory)

1. `go test ./...`, `just test`, `just lint` all clean.
2. CHANGELOG.md updated.
3. Write the summary (files touched, decisions made, anything deferred) and suggest a
   Conventional Commit message, e.g. `feat(cli): add jerry related (JRY-012)`.
4. Tidy WIP commits into atomic ones (root-path child). Commit locally; do not push or open an
   MR without user approval. Verify `git diff --name-only origin/main...HEAD | grep '^tickets/'`
   prints nothing before pushing. Hand back.

## Review

Reviewer independence (step 0): fresh session, no memory of writing this branch — the natural
handoff the protocol allows when an independent reviewer cannot be spawned. Recorded as this
review's own audits (independent by way of a memory-free session), not a self-review.

Implementation audit: all three tasks done as specified. `related.go` matches the plan's flag
shapes, envelope, exit-code rule (query not gate), and render-verbatim-order rule. Golden case
`related-match` and `TestRelatedFixtureContract` both present and pass on first run.
`go test ./...`, `just test`, `just lint`, `just docs-check` all clean.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | docs-gap | — | `jerry related` is not documented in `docs/user-manual/introduction.adoc`; the existing text also still asserts applies_to matching "is separate, larger work and not yet built," which JRY-011/012 made false | `docs/user-manual/introduction.adoc` (no `related` section; stale sentence under "References between decisions") | Add a section documenting `jerry related --paths <files> [--format text\|json]` (model it on the existing "Check it" section's style) and correct or remove the now-false "not yet built" sentence. `review-addendum.md` §4a is explicit that the manual being one page is not a reason to skip coverage for a new command. |
| F2 | non-blocking | stale-xref | fixed inline | DESIGN.md line 3 asserted "sections 7.2 onward are intent, not code," which this branch made false for §7.2's `related` bullet; PLAN.md's filed-so-far table had no row for `related`/JRY-012 | `DESIGN.md` (version stamp + §11), `PLAN.md` (filed-so-far table) | Fixed inline this review, on `feat/JRY-012-related` (commit `8d2d1c2`): version bumped to 2.6 with a corrected opening line and a new §11 entry; PLAN.md's `related` row added. |

Disposition summary: 1 blocking (F1, docs-gap), 1 non-blocking fixed inline (F2, stale-xref).

cost: estimated M, actual M.

Impact sweep (step 8): JRY-015 depends on JRY-012 and reuses `relatedFixture`; it calls
`internal/match.Resolve` in-process rather than the CLI envelope, so nothing this branch shipped
invalidates its assumptions. No patch needed.

### Rework fix record — round 1 (commit b1f9c7b)

Fixed F1: added a "Which decisions govern this code?" section to
`docs/user-manual/introduction.adoc` documenting `jerry related --paths <files> [--format
text|json]` (query semantics, silent-no-match, the `jerry.related/1` envelope), and replaced the
stale "matching ... is separate, larger work and not yet built" sentence with a short description
of the shipped dialect. `just docs-check`, `just test`, `just lint` all clean after the fix.

### Scoped re-review (round 1)

Reviewer independence (step 0): **delegated** — the reviewing agent authored round 1's fix
commit in this same session, so the audit of that diff (steps 2–4a) was handed to a fresh,
memory-free agent briefed adversarially, per an independent reviewer's brief. Its four findings
were each re-verified by hand before recording below (grep for section headings, read
`internal/cli/related.go`/`internal/match/match.go` directly, confirmed `just docs-check` clean)
— all four held up.

F1 confirmed resolved: `jerry related` now has a manual section, and every factual claim in it
(envelope schema, exit-code rule, ordering, `cmd.OutOrStdout()`) checks out against the code.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F3 | non-blocking | other | fixed inline | New section heading was the only question-phrased one in the file; every sibling is a noun/imperative phrase | `docs/user-manual/introduction.adoc:154` (as of b1f9c7b) | Renamed to "Resolve a path to its governing decisions", matching sibling style. |
| F4 | non-blocking | docs-gap | fixed inline | Example `--paths` value used jerry's own source path (`internal/cli/related.go`) rather than a fictional ADR-repo path like every other example and the golden-test precedent, which could read as introspecting on jerry itself | `docs/user-manual/introduction.adoc:158` (as of b1f9c7b) | Swapped for `teams/payments/src/ledger.go`. |
| F5 | non-blocking | correctness | fixed inline | "A queried path with no governing decision prints nothing" was stated before the `--format json` paragraph and read as universal, but JSON mode always emits an entry (empty `decisions: []`) — never literal silence | `docs/user-manual/introduction.adoc:163` (as of b1f9c7b) vs `internal/cli/related.go` (JSON path always appends to `envelope.Results`) | Qualified as "In the default text format, ...". |
| F6 | non-blocking | other | noted | The new cross-reference ("described below under ...") is a plain quoted string, not an asciidoc `xref:`/`<<>>` anchor — but the file has no xref convention anywhere, so this isn't a new deviation; a future heading rename won't be caught by `docs-check`'s xref checker | `docs/user-manual/introduction.adoc` (grep for `xref:`/`<<` returns nothing repo-wide) | No action — matches existing (lack of) convention; flagged for awareness only. |

Disposition summary: 3 fixed inline (F3, F4, F5), 1 noted (F6). Fixed in
`feat/JRY-012-related` commit `e6ae97f`; `just docs-check`, `just test`, `just lint` re-verified
clean after.

cost: estimated M, actual M.

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-1 row `related`,
  the read-side landing command that queries JRY-011's matcher.
- 2026-09-04 — TO DO → READY: plan complete
- 2026-09-04 — READY → IN DEVELOPMENT: picked up
- 2026-09-04 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-09-04 — IN REVIEW → REWORK: blocking: user-manual coverage for jerry related missing (F1)
- 2026-09-04 — REWORK → IN REVIEW: findings fixed
- 2026-09-04 — IN REVIEW → DONE: scoped re-review clean, 3 non-blocking fixed inline, 1 noted
- 2026-09-04 — MR opened (PR #15)
- 2026-09-04 — merged to main (PR #15, `4e4b321`)
