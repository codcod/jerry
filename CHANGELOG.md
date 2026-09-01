# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the version is below `1.0.0`, breaking changes may land in a minor release.

## [Unreleased]

### Fixed

- `jerry init` run from a binary built between tags (e.g. `just build`'s own pseudo-version,
  `v0.1.1-3-g<sha>`) now falls back to `@latest` in scaffolded CI instead of pinning to an
  unresolvable commit-describe string.

## [0.1.1] - 2026-09-01

### Fixed

- `go install github.com/codcod/jerry@vX.Y.Z` never resolved: the module root has no `main`
  package, only `cmd/jerry` does. Every documented and scaffold-emitted install command now
  reads `go install github.com/codcod/jerry/cmd/jerry@vX.Y.Z`.

## [0.1.0] - 2026-09-01

### Added

- Phase 1: the foundation. `jerry init` scaffolds a complete architecture-docs
  repository for GitHub or GitLab — templates, an example ADR, CODEOWNERS with a
  catch-all first so a new team folder is never left unreviewed, CI that
  verifies rather than bot-pushes, and a generated index. The emitted CI pins
  the version of jerry that wrote it, so a repository is always checked against
  the rules it was created with.
- The Tier 0 command set: `new adr` / `new sd` (next-free-id allocation,
  slugging, git identity, date stamping), `validate`, `fmt`, `index --check`,
  `supersede`, `status`, `schema`, `hooks install`, `version`.
- `jerry supersede` writes **both** halves of a supersession — the predecessor's
  `status` and `superseded_by`, the successor's `supersedes`, and the note under
  Consequences. Doing it by hand is how one side of the link ends up missing.
- `jerry status` refuses to set `Superseded` directly: it needs a successor to
  point at, so it is reachable only through `supersede`.
- Rule catalogue covering filenames, `id`/filename agreement, duplicate ids
  within a folder, allowed statuses, reference resolution (scope-qualified
  across folders), `team`/folder agreement, cross-cutting `teams:` lists, ISO
  dates, required sections, **empty sections**, and **unfilled template
  placeholders**. Findings accumulate: a validator that reports one error per
  run turns a five-minute fix into five CI round-trips.
- `--format text|json|sarif|junit` on `validate`. SARIF puts findings inline on
  the diff in GitHub and GitLab rather than in a log nobody opens.
- `jerry schema` emits JSON Schema for the frontmatter, so any editor with a
  YAML language server gets completion and inline validation for free.
- `applies_to` is accepted and validated now though nothing reads it until
  Phase 2, so `jerry related` will not need a migration.

### Fixed

- Index links are written relative to the index's own directory. The
  shell-script original wrote root-relative paths into `index/index.md`, where
  they resolved to `index/teams/...` — every link in the generated index was
  dead. Pinned by a test that stats each link from `index/`.
