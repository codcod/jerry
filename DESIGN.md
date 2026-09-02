# jerry — solution design

**Version 2.2** · 2026-09-02 · Phase 1 implemented; sections 7.2 onward are intent, not code.

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
2. **The binary owns the rule *set*; consuming repositories tune it, and cannot
   add to or remove from it.** Scaffolded repos have no `scripts/` directory.
   This is the reason to build a binary at all: a shell script copied into forty
   repositories is forty divergent rule sets within a year. `jerry.yaml` is the
   deliberate seam — it sets layout (`adr-dir`, `solution-design-dir`,
   `skip-dirs`), thresholds (`proposed-stale-days`) and the section and
   placeholder lists a rule reads. What it must never do is switch a rule off,
   because that is the same forty-divergent-rule-sets outcome reached by config
   instead of by copied script. **The list-valued keys are therefore additive:
   a repository extends `placeholders` and the required-section lists; it does
   not replace them.** (Today they replace — see §10.)
3. **Stateless, single binary, no runtime dependencies.** It must run in any CI
   image and in a pre-commit hook without a toolchain behind it — which is a
   claim about the *binary*, and one the emitted CI now honours: it downloads
   and checksum-verifies the release binary instead of running `go install`.
   §6 states the mechanism.
4. **Findings accumulate.** No command stops at the first problem.
5. **Hooks are UX; CI is the gate.** `--no-verify` exists. No check may exist
   only in a hook.
6. **The schema is the durable asset; the binary is replaceable.** Hence
   `schema_version` from day one, and tolerance for older documents later.

   A bump means: **a change no `v(n-1)` reader can be trusted to interpret** —
   a field whose meaning changed, or one that became load-bearing. Adding an
   optional field is not a bump; adding a rule is not a bump. The tolerance
   rule follows from the charter, and is directional:

   - A document with `schema_version` **at or below** what the binary knows is
     read with that version's rules.
   - A document **above** it produces a *warning*, never an error, and is
     otherwise checked with the newest rules the binary has. An old binary must
     not turn a repository red for being newer than itself: that would make
     upgrading jerry a prerequisite for merging anything, which is the opposite
     of "the binary is replaceable".
   - `jerry schema` must therefore publish a *floor*, not an equality. Today it
     emits `const: 1` and so will reject every v2 document the day v2 exists
     (§10).

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
| `schema_version` | optional | optional | Defaults to 1. Exists so the tool can stay tolerant of old documents. Nothing reads it yet (§10). |
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
| `applies_to` | optional | optional | Paths. Validated: rejects empty/whitespace-only, absolute, and `..`-traversal entries. Nothing yet *reads* the field for drift or ownership — that's still Phase 2. |

**Unknown keys are preserved, not rejected.** `jerry fmt` keeps keys jerry has
never heard of, in their authored order, because a tool that silently deletes
what it does not understand cannot be trusted to write files at all. The cost
is that a misspelling is invisible: `applies-to:` parses clean, round-trips
intact, and governs nothing — permanently and silently, on the one field the
whole of §7.2 depends on. Preservation stays; the **warning that a key is not
one jerry knows** is what is missing.

**`applies_to` holds paths, not service ids.** Version 1 of this document said
"paths or service ids"; nothing resolves a service id until a catalogue exists
(§7.3, `jerry owners`), so specifying the form now would be specifying it
against nothing. It is deferred to the ticket that builds the catalogue lookup.

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

**Transition legality is a guarantee of the *command*, not an invariant of the
corpus.** `jerry status` refuses an illegal move; `jerry validate` checks only
that a status is one of the allowed values. A hand-edited `Accepted` →
`Proposed` therefore validates clean, and CI — which runs `validate`, per §3.5
— will not catch it.

This looks like the failure §3.5 forbids, and it is worth being explicit about
why it is not: a transition is a statement about two states over time, and §3.1
gives the validator one state on disk. Checking it needs git history, which no
other rule needs and which is unavailable in the shallow clones CI produces.
The honest position is that the arrow diagram above describes what `jerry
status` will do for you, not what the corpus is guaranteed to satisfy. A
`validate --history` mode could close the gap; it is unbuilt and unrejected.

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

  The rule is a substring scan of the whole file, and that has an unmanaged
  cost: a decision that *documents* the id convention, or quotes a template in
  a fenced block, fails on a phrase it is legitimately discussing. The only
  escape today is deleting the phrase from `jerry.yaml`, which switches the
  check off for every document in the repository. For the rule this section
  calls the most valuable one, the path of least resistance must not be
  "turn it off": **fenced code blocks are excluded from the scan, and a
  document opts out of one phrase with an inline `<!-- jerry:allow
  placeholder -->` marker.** Neither is built yet (§10).

Staleness is a warning rather than an error deliberately: failing CI on the
calendar teaches people to falsify dates, and the judgement — accept it, reject
it, or close it — is not one a validator can make.

**A warning still needs somewhere to go.** Section 7.3 argues that a check with
no routing target produces a list nobody actions, and the argument applies here
first: the stale-proposal warning is currently printed into a green pipeline
log, which nobody opens. Until owner routing exists (§7.3), `jerry validate`
must at least report a warning count on success, so a clean run is visibly
distinct from a clean-with-loose-ends run.

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
- The pin is to a **released artifact, verified by checksum** — not to a source
  install. The emitted CI downloads the platform-matched release archive and
  verifies it against the published `checksums.txt`, which needs nothing but
  `curl`/`sh` in the runner image and honours the single-binary property of
  §3.3 at the one place it is meant to pay for itself.
  `go install github.com/codcod/jerry/cmd/jerry@latest` stays documented in the
  emitted `CONTRIBUTING.md` as a source-install fallback.
- **Pinning needs an upgrade path, and it is not the same thing as retroactive
  rules.** Freezing a version means a *bugfix* cannot reach a scaffolded
  repository either — the `0.1.1` fix to the install path is the live example,
  and no repository scaffolded by `0.1.0` will ever pick it up. So: a command
  that bumps a repository's pin, run **per repository, by that repository's
  owners**, landing as an ordinary merge request they review. What stays
  forbidden is an estate-wide sweep that bumps everyone at once, because that is
  retroactive rule enforcement wearing a version number.

## 7. Roadmap

**Forge strategy: one forge at a time, from Phase 2 onward.** Every integration
below reads "GitHub and GitLab", and Phase 1 already carries two CI variants and
two CODEOWNERS files for it. That symmetry was cheap when the artifact was a
template; it is not cheap once the artifact is an API client with tokens,
pagination and rate limits, and it doubles the cost of every remaining phase
against the bus factor §9 already flags as the binding constraint. So: **build
each integration on one forge, prove it is used, then port.** Two-forge parity
is a goal, not a definition of done — a working bot on one forge beats two
half-built ones, and the port is a known quantity once the interface has one
real consumer.

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

  It **counts what it surfaces from its first commit** — one appended line per
  post, to a file in the repository. This is a few lines of code inside the bot,
  and it is the difference between §9's adoption question being answerable and
  being an opinion. Deferring the counter to its own later ticket means the
  measurement arrives after everything it was supposed to inform has been built.

  It also needs a **token with write scope on merge requests** — the first thing
  in this design that does. The rules: the token comes from the CI environment
  and nowhere else, it is the forge's own bot or job token wherever one exists,
  the scaffold *documents* the requirement rather than provisioning it, and the
  bot degrades to a silent no-op when the token is absent rather than failing the
  pipeline. A docs tool must never be the reason a merge request cannot merge.
- `jerry search`, `jerry show` — offline query over the corpus.
- `jerry site` — a static searchable site. Static, so nobody is on call for it.
  Sequenced **after** the bot's numbers are in, not before: it is the largest
  piece of work in this phase and the one whose value is most dependent on the
  answer to §9.

### 7.3 Phase 3 — Company scale

**Where the on-call boundary actually is.** "No server, so nobody is on call"
(§2) is a claim about architecture, and it stops being a claim about operational
burden here. A crawl holds forge credentials, paginates, respects rate limits
and runs on a schedule; that is a thing that breaks, and the honest statement is
what happens when it does rather than that it cannot. So, explicitly: the failure
modes are an expired token, a scheduled job that silently stopped, and a forge
API that changed under us. Nobody is paged for any of them — **every one
degrades to a stale artifact, never to an empty or wrong one**, the artifact
carries the timestamp of its last successful crawl, and a consumer reading a
stale corpus can tell that it is stale. The cost of the whole thing failing
unnoticed is bounded at "the aggregate index is out of date", which is the
pre-jerry status quo. That bound is the design property; the absence of a daemon
is only how it is achieved.

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
  ids would remove an entire rule from section 5. This is the open question
  whose cost compounds fastest — every phase from 3 onward assumes references
  resolve estate-wide, and the note above is right that changing it later means
  rewriting references across the estate. **Decide it before the aggregate
  corpus exists, not after.**

  A third option this document previously did not consider: keep the short
  per-folder id as the *human* reference and add an immutable key that never
  changes. Renumbering then becomes a data migration of one field rather than a
  rewrite of every reference in every repository, and it is also what makes a
  folder rename (the `teams/` question above) survivable. It costs one more
  frontmatter field and buys the way out of two of the three open questions
  here.
- **OPEN: what metric governs Phase 2?** The proposal is *share of merge
  requests touching governed paths where a relevant decision was surfaced*, and
  *share of design reviews citing an existing ADR*. Explicitly **not** ADR count,
  which rewards writing trivial ADRs. Nothing measures either yet.
- **OPEN: bus factor.** A bespoke internal binary maintained by one person is
  worse than a readable script anybody can patch. Phase 1 is small enough that
  abandonment is survivable; each later phase makes that less true.

  Listing this as a risk and then planning nine more build steps is not a
  response to it, so here is the stop condition. **Phase 2 must show the read
  side being used before Phase 3 starts** — that is what the surfacing count in
  §7.2 is for, and "stop" is a legitimate answer to it. If it is used, the
  question becomes a second maintainer rather than a smaller roadmap, because at
  that point the corpus is load-bearing and abandonment is no longer
  survivable: **a second maintainer is a precondition for Phase 3, not a nice to
  have.** Two things bound the damage in the meantime — every output is a file
  that outlives the binary (§3.1), and the schema, not the tool, is the durable
  asset (§3.6). Neither helps if the estate depends on a crawl only one person
  can fix.

## 10. Known divergences between this document and the code

Version 1 asserted several things that Phase 1 does not do. They are listed
here rather than quietly corrected above, because a design document that has
been wrong once should show where it was wrong; each has a ticket or is named as
needing one.

| # | This document says | The code does | Where |
|---|---|---|---|
| 2 | `schema_version` keeps the tool tolerant of old documents (§3.6) | Nothing reads it, and `jerry schema` publishes `const: 1`, which will reject v2 documents outright | `internal/cli/schema.go` |
| 3 | Repositories own none of the rules (v1 §3.2) | `jerry.yaml` replaces the placeholder and required-section lists, so a repository can switch a rule off | `internal/config/config.go` |
| 4 | Status lifecycles are enforced (§4.2) | `jerry status` enforces transitions; `validate` checks only membership, so a hand edit passes CI | `internal/cli/status.go` vs `internal/rules/rules.go` |
| 5 | The placeholder rule is the highest-value check (§5) | It is a substring scan of the whole file, fenced blocks included, with no per-document escape — so the cheapest fix for a false positive is disabling it | `internal/rules/rules.go` |
| 6 | Findings accumulate and are always printed (§3.4, §5) | True — except `validate --diff`, which filters findings by a corpus-relative path against git's repo-relative output. With `jerry.yaml` below the git root it discards every finding and exits 0 | `internal/cli/validate.go` |

Item 6 is the worst of these: a validator that passes silently is worse than one
that is absent, because the green tick is taken as evidence.

## 11. Revision history

- **Version 2** (2026-09-02) — corrected after challenging v1 against the
  shipped code. §3.2 narrowed to what `jerry.yaml` actually permits; §3.3 and §6
  reconciled over the emitted CI's install mechanism; §3.6 given a bump
  definition and a tolerance rule; §4.1 corrected on `applies_to` and on service
  ids; §4.2 states that transition legality is a command guarantee, not a corpus
  invariant; §5 handles placeholder false positives and warning delivery; §6
  adds the per-repository pin bump; §7 states a one-forge-at-a-time strategy;
  §7.2 moves adoption counting into the bot and states the token model; §7.3
  states the on-call boundary; §9 gains a stop condition and a stable-key option
  for ids; §10 is new.
- **Version 2.1** (2026-09-02) — JRY-003 closed divergence 2 (§3.3/§6 vs. the
  emitted CI's install mechanism): the emitted CI now downloads a
  checksum-verified release binary instead of running `go install`, so §6's
  paragraph on the pin mechanism was rewritten to describe the shipped
  behaviour and the now-resolved row was removed from §10's table.
- **Version 2.2** (2026-09-02) — JRY-005 closed divergence 1 (§4.1/§10 vs. `applies_to`
  validation): the field is now validated for path shape, unknown frontmatter keys warn, and
  the resolved row was removed from §10's table.
- **Version 1** (2026-09-01) — initial design, written alongside the Phase 1
  implementation.
