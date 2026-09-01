# jerry

A single binary that scaffolds a repository of Architecture Decision Records
and Solution Designs, then owns every rule that governs it.

```sh
brew install codcod/tap/jerry     # or: go install github.com/codcod/jerry/cmd/jerry@latest

mkdir architecture-docs && cd architecture-docs && git init
jerry init --forge github         # or --forge gitlab
jerry hooks install
```

Scaffolded repositories carry no scripts of their own: the binary is the single
source of truth for the rules, so they cannot drift between the repositories
that share them.

The design of record is [DESIGN.md](DESIGN.md). Everything beyond "what is it
and how do I install it" is in the user manual: `docs/user-manual.adoc`,
rendered to PDF and EPUB on each release.
