# jerry

A single binary that scaffolds a repository of ADRs and Solution Designs and
owns every rule that governs it. The design of record is
[`DESIGN.md`](DESIGN.md); it is authoritative on intent, and where a shipped
ticket decision contradicts it, the ticket wins and DESIGN.md should be fixed.

## Governing documents

| document | what it is |
|---|---|
| [`DESIGN.md`](DESIGN.md) | the design of record. §3 charter · §4 schema · §5 rules · §6 scaffold contract · **§10 known divergences between this document and the code** |
| [`PLAN.md`](PLAN.md) | provisional decomposition of DESIGN.md into tickets, keyed by slug; its *Filed so far* table maps slug → ticket id |
| [`CHANGELOG.md`](CHANGELOG.md) | what shipped, per release |
| [`review-addendum.md`](review-addendum.md) | jerry's project-specific review rules, layered on top of the brine review protocol via `review_addendum` in `pickle.toml`; no overarching addendum, since jerry is the only child-project |

## Conventions that reviews must enforce

These are the rules that are cheap to break and expensive to notice. They are the *conventions*;
the audit procedure for them — which are enforced by the test suite and which need a reviewer's
attention — is in [`review-addendum.md`](review-addendum.md), which deliberately does not
restate them.

- **Dependency policy, as a grep — not as a belief.** `go.mod` stays at cobra +
  pflag + yaml.v3 and the standard library. Verify it mechanically rather than
  trusting recollection. jerry runs in CI images and pre-commit hooks; every
  dependency is weight in both.
- **Golden fixtures are rendered from the production path, never hand-edited.**
  Regenerate with `just test-update` (or `go test ./... -update`). A fixture
  patched by hand asserts what somebody believed, not what the code does. Every
  golden test is paired with a `…FixtureContract` test asserting the properties
  the fixture exists to demonstrate, so regenerating can never quietly drop one.
- **Every leaf command carries a `jerry/kind` annotation** (`read`, `write` or
  `other`), and every `write` leaf has `--dry-run`.
  `internal/cli/write_safety_test.go` enforces both, including a negative test
  proving the checker fires. Adding a command without an annotation is a test
  failure, not a review comment.
- **No package-level root command and no `init()` registration.** Commands are
  constructors taking `*globals`, so two trees in one process never share flag
  state. There is a test for that too.
- **Output envelopes are versioned and additive only** —
  `jerry.findings/1`, `jerry.index/1`. Consumers pin them.
- **The scaffold must validate clean the day it is written.**
  `internal/scaffold/scaffold_test.go` runs the real rules over a freshly
  emitted repository. If `init` ships something that fails `jerry validate` or
  `jerry index --check`, that is the worst possible first impression.
- **The placeholder list must match the shipped templates.** A phrase in the
  list that appears in no template catches nothing; a template phrase missing
  from the list means a copied, unfilled document validates clean. Tested.
- **`README.md` is deliberately a short pointer to the manual — do not grow it
  back.** Everything beyond "what is it and how do I install it" belongs in
  `docs/user-manual/`, and a change to a command's behaviour or flags has two
  doc surfaces to reconcile: the manual is the one reviews keep missing.
- **Comments explain why, not what**, and cite tickets and design sections
  inline (`JRY-004 decision 2`, `DESIGN.md §4.2`) so the reasoning stays
  reachable from the code.

## Commit and branch conventions

- Code: Conventional Commits with the ticket id in parentheses at the end —
  `feat(cli): add supersede (JRY-004)`.
- Ticket and board bookkeeping: `board: JRY-004 <verb phrase>`.
- Branches: `feat/JRY-NNN-<slug>`.
- Stage with explicit pathspecs. Never `git add -A` or `git add .`.

## Build

`just build test lint docs-check` — and `just dist-check` before a tag. Lint is
`gofmt -l` plus `go vet`, matching CI exactly; there is no `.golangci.yml`.

<!-- pickle:begin -->
## Brine (start here)

**Start at [`tickets/BOARD.md`](tickets/BOARD.md)** — the generated index of every ticket by
status. No feature is built directly from a chat message or a raw idea — work enters only as a
ticket whose Implementation Plan has met the READY gate. A *review finding* is different: it
earns a **disposition** (rules §5), and most are resolved without a new ticket.

- The flow engine is the **brine skill** at `.agents/skills/brine/`. It holds
  the rules (`resources/tickets-README.md`), the ticket template
  (`resources/TEMPLATE.md`), and the review protocol
  (`resources/review-protocol.md`). Agents that read `.agents/skills/` find it there
  directly; `pickle install --agent claude` adds a `.claude/skills/brine` view for
  Claude Code. The directory is pickle-owned — `pickle install` and `pickle upgrade`
  both replace it wholesale, so keep hand-written notes outside it.
- Triggers: "make it a ticket", "refine ticket T-NNN", "implement ticket T-NNN", "rework ticket
  T-NNN", "validate ticket T-NNN" (or "review ticket T-NNN"), "audit the board".

### Project configuration

- **Build target.** Every ticket targets one registered child-project via `project:`
  frontmatter (`pickle project list`). Registered child-projects: `jerry`.
- **Commands** (each child's, from `pickle.toml`):
  - `jerry`: build `just build` · test `just test` · lint `just lint` · docs `just docs-check`
- **Branch & commit.** Conventional Commits with the **ticket id in brackets at the end of
  the subject** (e.g. `feat(cli): add board audit (T-2)`) for child-project code. Ticket/board
  bookkeeping uses its own `board: T-NNN <verb phrase>` form instead — grammar and scope in
  the rules §0. Branch per child:
  - `jerry`: `feat/JRY-NNN-<slug>`
- **WIP limits** (per child):
  - `jerry`: `3-in-development/` ≤ 1 · `4-in-review/` ≤ 1
- **Commit policy.** Child-projects are **publish-gated**: local WIP commits are encouraged;
  **no push / no merge request without explicit user approval**; after approval, finalize
  (squash or keep history) + push + open the MR — **merging is always the human's**.
  Overarching bookkeeping (tickets, board, docs) may be committed automatically,
  always with **explicit pathspecs** (`git add <paths>`, never `git add -A`/`.`).
- **Where commits land.** Code goes on the child's feature branch; **ticket and board
  bookkeeping is committed on the base branch**, never on a feature branch — a squash-merge
  folds or drops it and the board then disagrees with the tickets it indexes. This covers a
  review's own moves too, and it is why a reviewer on a feature branch reads the ticket from
  the base branch. This project uses the `in-tree` layout, where the board and the code share
  one repository, which is what makes the rule load-bearing here.
  `pickle hooks install` enforces it locally, once per clone: a `pre-commit`
  hook refuses the commit, and a `pre-push` hook refuses the push if it still slipped through
  (bypass either with `--no-verify`).

### Board rule

`tickets/BOARD.md` is **generated** — regenerated wholesale from the ticket files by
`pickle ticket new`, `pickle ticket move` and `pickle board sync`. **Never edit it by
hand**; hand-written planning notes go in `tickets/NOTES.md`. Every ticket move = move
the file + one dated `## History` line, and the board regenerates. Prefer
`pickle ticket move` — it does all of it atomically.
<!-- pickle:end -->
