# jerry — solution design

**Version 1** · 2026-09-01 · Phase 1 implemented; sections 7.2 onward are intent, not code.

This file is authoritative on **intent**. Where it conflicts with a shipped
ticket decision, the ticket wins and this file is wrong and should be corrected.
Genuinely undecided points are marked **OPEN** rather than papered over. Tickets
are cut from this file.

The user manual (`docs/user-manual.adoc`) documents what has actually been
built; this file documents what it is for and why it is shaped this way.

---

## 1. Purpose

Organisations that adopt ADRs usually fail at it, and almost never because of
tooling. They fail because decisions are made in Slack threads and design
meetings and never recorded; because the records that exist go stale, so people
stop trusting them, so they stop reading them, so they stop writing them; and
because nobody has a moment in their week when they actually need the corpus.

jerry exists to make the mechanical half free — allocating ids, writing both
halves of a supersession, keeping an index honest, checking that references
resolve — so that the remaining difficulty is the part that deserves human
attention: whether the context is accurate, whether alternatives were genuinely
considered, and whether the consequences are stated honestly.

**A binary cannot fix the social half.** What it can do, from Phase 2 onward, is
surface the relevant decision at the moment somebody changes the code it
governs. That is the only mechanism in this design with a real chance of moving
the underlying behaviour, and everything before it is groundwork.

## 2. What jerry is, and is not

**Is:** a stateless CLI over a git repository of markdown files. It scaffolds
the repository, validates it, allocates identifiers, moves statuses, and renders
an index. Every output is a file.

**Is not:** a server, a database, a wiki, an approval workflow engine, or a
replacement for the forge's review process. Git is the record. Merge request
approval is the approval.

**Deliberately not built** (recorded so they stay decided):

- A server with a database. Static artifacts instead — the moment this grows a
  service, somebody owns an on-call rotation for a docs tool.
- Two-way Confluence sync. One-way publish only; two-way is a tar pit.
- An approval workflow engine. Merge request approvals already are one.
- Custom identity or auth. Use the forge's.
- WYSIWYG editing.
- Per-team "ADR maturity" dashboards. Counting ADRs produces trivial ADRs.

## 3. Charter

1. **Git is the database.** jerry holds no state of its own. Every command reads
   files and writes files, so a repository stays fully usable without jerry
   installed — just unenforced.
2. **The binary owns the rules; consuming repositories own none of them.**
   Scaffolded repos have no `scripts/` directory. This is the reason to build a
   binary at all: a shell script copied into forty repositories is forty
   divergent rule sets within a year.
3. **Stateless, single binary, no runtime dependencies.** It must run in any CI
   image and in a pre-commit hook without a toolchain behind it.
4. **Findings accumulate.** No command stops at the first problem.
5. **Hooks are UX; CI is the gate.** `--no-verify` exists. No check may exist
   only in a hook.
6. **The schema is the durable asset; the binary is replaceable.** Hence
   `schema_version` from day one, and tolerance for older documents later.

## 4. Document schema

Two kinds, deliberately different in nature.

An **ADR** is an append-only log entry. It is never deleted or rewritten; it is
superseded, and the supersession is a link in both directions.

A **Solution Design** perishes. It describes an intent that either ships or does
not, and once it no longer describes the running system it is misleading rather
than merely old. Its lifecycle therefore has somewhere to end.

### 4.1 Frontmatter

| Field | ADR | SD | Notes |
|---|---|---|---|
| `schema_version` | optional | optional | Defaults to 1. Exists so the tool can stay tolerant of old documents. |
| `id` | required | — | `ADR-NNNN`, must match the filename. SDs have no sequential id. |
| `title` | required | required | |
| `status` | required | required | See 4.2. |
| `superseded_by` | when Superseded | — | Scope-qualified references. |
| `supersedes` | optional | — | The reverse pointer. |
| `team` | required | required | Must match the folder, except cross-cutting. |
| `teams` | cross-cutting only | — | Two or more. |
| `date` | required | required | ISO. For SDs the filename prefix must agree. |
| `deciders` / `authors` | required | required | |
| `related_adrs` | — | optional | Checked. |
| `applies_to` | optional | optional | Paths or service ids. Accepted and validated now; nothing reads it until Phase 2. |

### 4.2 Status lifecycles

ADR: `Proposed` → `Accepted` \| `Rejected`; `Accepted` → `Deprecated`; any →
`Superseded` **only** via `jerry supersede`.

`Rejected` is first-class. A decision that was considered and turned down is
precisely the record that stops the same proposal returning every year, and
deleting it destroys the most reusable reasoning in the corpus.

`Superseded` is unreachable through `jerry status` on purpose: it requires a
successor to point at. Allowing it there would permit a `Superseded` document
with no successor, which is exactly the dangling state the field exists to
prevent.

SD: `Draft` → `In Review` → `Approved` → `Implemented` → `Archived`, with
`Archived` reachable from anywhere (abandonment is a legitimate ending) and
`Superseded` for replacement.

### 4.3 References

IDs are unique **per folder**, not globally — the price of short, readable
numbering. A bare `ADR-0007` therefore means "in this document's own folder";
anything else must be scope-qualified: `payments/ADR-0007`,
`cross-cutting/ADR-0003`.

This is a trade-off, not a design triumph. See section 9.

## 5. Rule catalogue

Every rule is a function over the corpus returning findings. Severity decides
the exit code; it never decides whether a finding is printed.

**Errors:** filename format · `id`/filename agreement · duplicate id within a
folder · unknown status · `superseded_by` present when Superseded, absent
otherwise · every reference resolves · `team` matches folder · cross-cutting
names two or more teams and uses `teams:` not `team:` · ISO date · SD filename
prefix matches its date · required sections present · **required sections
non-empty** · **no unfilled template placeholders**.

**Warnings** (never fail CI): an ADR left `Proposed` past `proposed-stale-days`.

Two of these earn their place beyond the obvious structural checks:

- **Empty sections.** A heading with nothing under it passes every structural
  check while reading as though the question was answered. HTML comments do not
  count as content, which is why `jerry new` writes its prompts as comments — a
  document created and never filled in must fail.
- **Placeholders.** A copied, half-filled template is the single most common
  real defect. A test asserts every placeholder in the default list actually
  appears in a shipped template, so the list cannot rot into decoration.

Staleness is a warning rather than an error deliberately: failing CI on the
calendar teaches people to falsify dates, and the judgement — accept it, reject
it, or close it — is not one a validator can make.

## 6. Scaffold contract

`jerry init` writes a complete repository and nothing that needs a network.

- The index is **generated, never embedded**. A committed template index is
  stale the moment the example ADR changes, and a freshly scaffolded repo would
  fail its own first pipeline. (It did, until this was fixed; a test now pins
  it.)
- CI **verifies** the index rather than pushing it. A job that regenerated it
  and pushed to `main` needs a protected-branch bypass, adds a commit per merge,
  and fails on a non-fast-forward whenever two merge requests land together.
- CODEOWNERS puts the **catch-all first** (both forges apply the last matching
  pattern). Without it a newly created `teams/<new-team>/` matches no rule at
  all: no required reviewer, while the documentation claims review is enforced.
- Emitted CI **pins the version of jerry that wrote it**, so a repository is
  checked against the rules it was created with. Rule changes are not
  retroactive across the estate; a new check must not turn every repository red
  the day it merges.

## 7. Roadmap

### 7.1 Phase 1 — Foundation *(implemented)*

`init` plus the Tier 0 command set — `new`, `validate`, `fmt`, `index`,
`supersede`, `status`, `schema`, `hooks`, `version` — operating on one
repository on local disk.

`jerry fmt` canonicalises frontmatter and whitespace only. It does **not**
reflow prose: deciding what is a paragraph and what is a list item, table row or
fenced block is exactly where a markdown formatter corrupts authored text, and a
formatter that damages content is worse than no formatter. Frontmatter is
structured data with an unambiguous canonical form, so that is what is
normalised — preserving keys jerry has never heard of, in their authored order.

### 7.2 Phase 2 — The read side

**This phase is the point of the project, and it is deliberately sequenced
before validation is hardened into a blocking gate.** Validation is the fun,
bounded, testable part, so the predictable failure mode is an excellent linter
and no reader ever served. A linter alone is net negative: every author pays
friction and no reader gets value, and the tool acquires a reputation as
compliance theatre in its first quarter.

- `jerry related --paths <changed files>` — which decisions govern the code I am
  about to touch. Requires `applies_to`, which Phase 1 already accepts.
- **A merge request comment bot** running that same query in CI, posting the
  governing decisions on any MR touching covered paths. Highest adoption impact
  of anything in this document: it delivers decisions at the moment of
  relevance, and it makes writing them visibly worthwhile because authors see
  their work surface where it matters.
- `jerry search`, `jerry show` — offline query over the corpus.
- `jerry site` — a static searchable site. Static, so nobody is on call for it.

### 7.3 Phase 3 — Company scale

- `jerry crawl` — registry-driven multi-repo discovery, fetching document
  directories through the forge API without cloning, degrading to last-known
  state rather than emptying the index when one repository is unreachable.
- Cross-repo reference resolution: a single repository cannot resolve
  `payments/ADR-0007` because it does not have that repository. Local validation
  covers syntax and same-repo links (blocking); the aggregator covers cross-repo
  links (advisory — a link can break through no fault of the MR in front of you).
- `jerry owners` — resolve owners from CODEOWNERS or a service catalogue, so
  every finding has somewhere to go. A check with no routing target produces a
  list nobody actions.
- `jerry stale --assign` — opens issues against resolved owners.
- **Evidence-based drift**: compare git churn in an ADR's `applies_to` paths
  since its date. "This area has changed 200 commits since the decision was
  recorded" is a far better staleness signal than a calendar.
- `jerry graph` — supersession chains, cycles, orphans, dangling pointers.
- `jerry archive <repo>` — pull decisions out of a repository being
  decommissioned. Easy to forget, and dead services' decisions are the ones most
  worth keeping.
- Deletion detection by diffing successive crawls: an `Accepted` ADR that
  vanished is an alert, not a silent gap.

### 7.4 Phase 4 — Integrations

Forge comments, SARIF and issue creation; Backstage catalog entities; one-way
Confluence publish for non-engineer readers; a Slack digest and owner nudges.

### 7.5 Phase 5 — Lowering the write cost

- `jerry draft --from-mr <id>` — a skeleton from the diff and description.
  Prefill Context and prompt for Alternatives; **never** auto-generate Decision
  or Consequences. A machine-written consequences section is worse than an empty
  one, because it looks like thinking happened.
- Undocumented-decision detection: flag merge requests that add a dependency,
  introduce a datastore, add an external integration, or change a base image,
  and ask whether an ADR is warranted. This turns "we never record decisions"
  from a culture problem into a prompt at the right moment.

## 8. Non-goals

See section 2. Additionally: jerry does not lint prose, does not grade decision
quality, and does not attempt to detect semantic conflicts between decisions.

## 9. Open questions

- **OPEN: does `teams/` survive a reorganisation?** Teams reorganise and
  decisions do not. On the first reorg the choice is to move files — breaking
  every permalink and cross-reference — or to live with a lying path. A layout
  partitioned by system or service would be more durable and makes CODEOWNERS
  routing harder. Phase 1 ships `teams/` because it was the layout being
  replaced; `--layout flat` remains unbuilt and unrejected.
- **OPEN: is central authoring right at all?** ADRs decay when they do not live
  beside the code they govern, because nobody edits a document in a repository
  they never open. The alternative — `docs/decisions/` per service plus a
  central aggregating index — is Phase 3's `crawl` in everything but name, and
  would make the central repository index-only. That would be a reversal of
  section 6, and it should be taken deliberately if taken at all.
- **OPEN: per-folder ids.** They buy short references and cost scope
  qualification plus a duplicate-id check. Date-prefixed or globally monotonic
  ids would remove an entire rule from section 5.
- **OPEN: what metric governs Phase 2?** The proposal is *share of merge
  requests touching governed paths where a relevant decision was surfaced*, and
  *share of design reviews citing an existing ADR*. Explicitly **not** ADR count,
  which rewards writing trivial ADRs. Nothing measures either yet.
- **OPEN: bus factor.** A bespoke internal binary maintained by one person is
  worse than a readable script anybody can patch. Phase 1 is small enough that
  abandonment is survivable; each later phase makes that less true.

## 10. Revision history

- **Version 1** (2026-09-01) — initial design, written alongside the Phase 1
  implementation.
