# Contributing an ADR or Solution Design

Install jerry once, then let it do the mechanical parts:

```sh
brew install codcod/tap/jerry     # or: go install github.com/codcod/jerry/cmd/jerry@latest
jerry hooks install               # validates and regenerates the index on commit
```

## Adding an ADR

```sh
jerry new adr "Use event sourcing for the ledger" --team payments
```

This allocates the next free number in the folder, slugs the filename, stamps
today's date, and fills `deciders` from your git identity. Then:

1. Fill in Context, Decision, Consequences and Alternatives Considered. An
   empty section fails validation — a heading with nothing under it reads as
   though the question was answered.
2. Choose the status you're asking reviewers to agree to:
   - `Accepted` — the normal case. Merging the MR *is* the acceptance, so don't
     merge as `Proposed` planning to "update it later"; that follow-up never
     happens.
   - `Proposed` — only when the MR is deliberately open for discussion. Jerry
     warns once it is over 90 days old.
   - `Rejected` — considered and turned down. Worth merging: it stops the idea
     being re-proposed from scratch.
3. Open a merge request. CODEOWNERS routes it to your team's reviewers.

Run `jerry validate` and `jerry index` before pushing — CI runs the same two
commands, and the hook does it for you.

## Adding a Solution Design

```sh
jerry new sd "Ledger rewrite" --team payments
```

Solution designs have no sequential ID — the filename is `YYYY-MM-short-title.md`
and the prefix must match the frontmatter `date`. Link any ADRs the design
produced in `related_adrs`; every reference is checked.

Solution designs go stale in a way ADRs don't, so they carry an ongoing
obligation on the authoring team:

- `jerry status <path> implemented` when the design ships.
- `jerry status <path> archived` once it no longer describes the running system,
  or if it was abandoned. Don't leave it `Approved`.

## Superseding an ADR

Never delete or edit history away:

```sh
jerry supersede payments/ADR-0007 --with "Use CockroachDB for the ledger"
```

That creates the successor, sets `status: Superseded` plus `superseded_by:` on
the old one, writes the reverse `supersedes:` pointer on the new one, and adds
the note under Consequences. Doing it by hand is how one side of the link ends
up missing.

References to an ADR in a different folder must be scope-qualified:
`payments/ADR-0007`, `cross-cutting/ADR-0003`. Bare IDs are only unique within
one folder.

## Cross-cutting decisions

If a decision affects more than one team it goes under `cross-cutting/`, uses
`teams: [a, b]` instead of `team:`, and needs a reviewer from each affected team.

```sh
jerry new adr "Standardise on OpenTelemetry" --cross-cutting --teams payments,platform
```

That's a heavier review than a team folder, which creates an incentive to file
multi-team decisions under your own team instead. Don't. Reviewers: if an ADR in
a team folder commits other teams to anything, ask for it to be moved.

## Review expectations

- At least one reviewer from the owning team (routed via CODEOWNERS; new team
  folders fall back to the catch-all until a CODEOWNERS line is added).
- For cross-cutting ADRs, at least one reviewer from each affected team.
- Reviews focus on: is the context accurate, were alternatives genuinely
  considered, are consequences (including negative ones) honest, and does this
  contradict an existing `Accepted` ADR.

## What CI checks

- `jerry validate` — filenames, `id`/filename agreement, duplicate IDs within a
  folder, allowed statuses, `superseded_by` presence and resolution,
  `team`/folder agreement, cross-cutting `teams:` lists, ISO dates, required and
  non-empty sections, unfilled template placeholders, and a warning for ADRs
  left `Proposed` for over 90 days.
- `jerry index --check` — that `index/index.md` matches the documents.

Run `jerry schema --kind adr` to get a JSON Schema your editor can use for
frontmatter autocompletion and inline validation.
