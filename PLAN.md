# PLAN.md — provisional ticket list for building DESIGN.md

Scratch artifact. Discard once the tickets are filed.

**What this is.** A rough decomposition of `DESIGN.md` into tickets, ordered by and
cross-referenced to the **roadmap (§7)**. Every row cites the design sections it implements.
Sizes and splits are guesses — expect churn as tickets are refined and as implementation
reveals what actually belongs together.

**Rows are keyed by slug, not by ticket id.** Version 1 of this file numbered the rows
`JRY-001`…`JRY-030` on the assumption they would be filed in order, and warned that an
unplanned ticket landing mid-sequence would force a re-map. That happened immediately:
`JRY-002` was filed as a defect spawned by `JRY-001`'s review, not as this file's second row,
so every `depends-on` after row 1 pointed at a ticket that no longer meant what it said. The
board owns the id space and assigns ids at filing time; a scratch document should not pretend
to own it too. **Slugs here are permanent handles for a unit of work; the id it gets is
whatever `pickle ticket new` allocates.** Filed rows record their real id.

**Phase 1 (§7.1) is already built and shipped without tickets.** It was written in one pass
against the design rather than through the board, so there is no `JRY-000` and no history to
reconcile. That is a one-off: everything in this file goes through the board. Three defects
found during that pass are already fixed and recorded in `CHANGELOG.md` (the scaffold shipping
a repo that failed its own CI, cobra's `Print*` writing to stderr and breaking the git hook,
and a placeholder that matched no template) — they are mentioned here only so nobody files
them again.

**Filed so far:**

| slug | id | status |
|---|---|---|
| `release` | `JRY-001` | done, merged (`v0.1.0`, then `v0.1.1`) |
| `pin-fallback` | `JRY-002` | done, publish-gated — unplanned, spawned by `JRY-001`'s review |
| `ci-binary-install` | `JRY-003` | done |
| `forge-proof` | `JRY-004` | done — GitHub proven end to end; GitLab half deferred to `JRY-009` |
| `applies-to-validate` | `JRY-005` | to do |
| `placeholder-escapes` | `JRY-006` | done, publish-gated |
| `golden-tests` | `JRY-007` | to do |
| `schema-tolerance` | `JRY-008` | to do |
| `forge-proof-gitlab` | `JRY-009` | done — GitLab proven end to end with the current (checksum-verified) template |

`pin-fallback` is not a row in the tables below and never was; it is recorded here because it
consumed the id this file's second row expected.

None of the six new tickets carries `depends-on:` frontmatter yet — hard dependencies are
human-approved, so each states its sequencing in prose for refinement to promote. The intended
order is the one above.

**Sizing calibration.** Still weak, but no longer absent. `release` was sized `S` and behaved
like one, though it found two real defects (the `go install` module path, and the pin fallback
that became `JRY-002`) — so *verification* tickets earn their keep and should not be trimmed on
the grounds that they build nothing. The useful baseline is that **all of Phase 1 is ~4,500
lines of Go**; any row below sized `L` is claiming a third of Phase 1's volume, and three of
them are claiming it for work with an open-ended surface (a static site, a crawler, diff
inference). Treat `L` here as "not yet decomposed" rather than as an estimate. The rows most
likely to be under-sized are still the ones that touch a forge API (`forge-comment`, `crawl`,
`owners`): token handling, pagination and rate limits are never in the one-line description.

**Documentation is part of every row, priced nowhere.** `AGENTS.md` requires the user manual to
track any change to a command's behaviour or flags, so every row that adds or changes a command
carries manual work on top of its size. Stated once here rather than repeated per row;
`manual-restructure` is the *structural* change to the manual, not its content.

**Sequencing rules that constrain this list** (§7, restated so they are not lost in a re-order):

- **Build step 0 comes first and is not optional.** Until jerry is released, the CI that
  `jerry init` emits pins `@latest` rather than a version (§6), so no scaffolded repository is
  actually reproducible. Everything downstream assumes a released binary. *(`release`: done.)*
- **One forge at a time from Phase 2 onward** (§7, new in DESIGN v2). Every integration row
  below reads "both forges" in its Phase-4 form; each is built and proven on one forge first.
  Two-forge parity doubles the cost of every remaining step against the bus factor §9 names as
  the binding constraint.
- `applies_to` matching (step 1) lands before the read side (step 2) and before drift detection
  (`drift`). Phase 1 already *accepts* the field, so this is semantics, not schema — but Phase 1
  does **not** validate it (DESIGN §10.1), so `applies-to-validate` is a prerequisite, not a
  bonus.
- **The read side lands before validation is hardened into a blocking gate.** This is the
  design's own sequencing rule (§7.2) and the easiest one to quietly violate, because adding a
  rule is a pleasant afternoon and building a reader is not.
- **Step 4 gates step 5.** If the read side is not measurably used, company-scale aggregation
  is scaling something nobody reads. The counting happens inside `bot` from its first commit
  (§7.2, DESIGN v2); `adoption-report` reads it and is allowed to conclude "stop".
- The aggregate envelope (`corpus-artifact`) lands before cross-repo references (`cross-repo-refs`)
  and deletion detection (`deletion-detect`), both of which consume it.
- The forge client is written once and reused. It is **not** written before its first consumer:
  `forge-comment` builds only what a comment bot needs, and `crawl` extends it with the
  pagination and rate-limit handling it is the first thing to actually exercise. A second
  hand-rolled HTTP client for a forge is a review failure; a speculative general one built
  against no consumer is the other failure, and the `L` on a single up-front client row was
  hiding a five-row critical path.
- Nothing in steps 5+ may introduce a server or a database (§3.1, §8). Every output stays a file.

Each build step below carries an **Outcome** line: what concretely exists and works once that
step's tickets are done, stated as capability rather than as code. It is there so the value of
stopping — or pausing — at any given step is legible without reading the ticket list.

WIP limit is 1 in development and 1 in review for `jerry`, so this is a queue, not a parallel
plan.

---

## Build step 0 — Release, so the scaffold is real

**Outcome.** jerry is installable at a pinned version, and a repository it scaffolds is
reproducible *and* checkable without a Go toolchain: its CI fetches a checksum-verified release
binary rather than building one from source on every run. This is the difference between a tool
that works on the author's machine and one another team can adopt. Almost nothing new is built —
the deliverable is that Phase 1 is actually usable by someone else (§6, §3.3, RELEASING.md).

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| ~~`release`~~ | Cut `v0.1.0`: reconcile `CHANGELOG.md`, tag, verify goreleaser publishes binaries + checksums, verify the tap formula installs, verify `go install` resolves — **`JRY-001`, done** | §6, RELEASING.md | — | S |
| `ci-binary-install` | Emitted CI installs a **checksum-verified release binary** instead of `go install`: fetch the asset for the runner's platform, verify against the published checksums file, keep the source install documented as a fallback | §3.3, §6 | `release` | M |
| `forge-proof` | Prove the emitted CI on a real forge: scaffold a throwaway repository on each of GitHub and GitLab, push, and confirm both pipelines pass unmodified — including a deliberately broken document failing them | §6, §5 | `ci-binary-install` | M |
| `golden-tests` | Command-level golden tests: execute the cobra tree in-process against fixture repositories and byte-compare stdout, stderr and exit codes. Phase 1 has structural tests (`write_safety_test.go`) and package tests, but no test asserts what a user actually sees | §5 | `release` | M |

`forge-proof` is the only thing that can tell us the scaffolded CI works, and it went missing
in v1 of this file: its number was taken by an unplanned defect and the row was never refiled.
The local tests assert the workflow file's *content* and that `actionlint` accepts it; neither
runs it. A `golang:1.26` image that cannot reach the module proxy, or a GitLab runner without
`go`, fails only here — which is also the argument for `ci-binary-install` landing first, since
it deletes both of those failure modes rather than proving them.

`golden-tests` is pulled forward from the cross-cutting section. It is buildable today, every
row after it benefits, and nothing about it gets easier by waiting. See the note on scheduling
at the end of this file.

## Build step 1 — `applies_to` acquires meaning

**Outcome.** A decision can state which code it governs — and jerry checks that the statement
is well-formed — and can answer the reverse question: given a set of changed paths, which
decisions apply. Nothing surfaces it to anyone yet; this is the index that step 2 queries.

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| `applies-to-validate` | Validate `applies_to`, and **warn on unknown frontmatter keys**. DESIGN §4.1 claimed the field was validated; nothing reads it. Unknown keys are preserved by design and never reported, so `applies-to:` parses clean and governs nothing, silently | §4.1, §5, §10 | `release` | S |
| `applies-to-match` | `applies_to` matching semantics: **gitignore-style globs**, directory-prefix rules, and precedence when several decisions match one path. Specified in DESIGN.md §4.1 first, then implemented with table-driven tests | §4.1, §7.2 | `applies-to-validate` | M |
| `related` | `jerry related --paths <files>`: resolve changed paths to governing decisions, `--format text\|json`, versioned `jerry.related/1` envelope | §7.2 | `applies-to-match` | M |

Decide the glob dialect once, in `applies-to-match`, and write it into the design before coding
it. The recommendation is gitignore syntax, because `.gitignore` and CODEOWNERS are the two
other path-matching files in every one of these repositories and the audience is already
trained on them. Path matching that is "whatever the library did" is the kind of thing nobody
can answer questions about a year later.

**Service ids are out of scope here.** v1 of this row required specifying "the service-id form
and how it resolves", but nothing resolves a service id until `owners` builds the catalogue
lookup in step 6 — so the row carried an undeclared dependency on a step five places later.
DESIGN v2 §4.1 narrows the field to paths; service ids come back with `owners` or not at all.

`applies-to-validate` is small and it is the one row here that should not slip: every day of
Phase 1 use accumulates documents whose `applies_to` was never checked, and step 2 under-reports
on all of them without ever saying so.

## Build step 2 — Decisions arrive where the work happens

**Outcome.** The point of the whole project. An engineer changing a governed path sees the
decisions that govern it, in the merge request, without going looking. Authors see their
records surface where they matter, which is the only argument for writing them that has ever
worked. Everything before this step is groundwork and everything after it is scale (§1, §7.2).

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| `forge-comment` | Forge client, comment scope only: one interface, **one forge**, token from the CI environment only, comment create-or-**update** (never a second comment on the same MR). No pagination or crawl-scale concerns — they arrive with `crawl`, which is the first thing that exercises them | §7.2, §7.4 | `release` | M |
| `bot` | `jerry comment`: run `related` over the merge request's changed files and post the governing decisions, or update the existing comment; silent no-op when nothing matches **and when no token is present**; appends one JSONL line per post so surfacing is counted from day one | §7.2, §9 | `related`, `forge-comment` | M |
| `bot-scaffold` | Scaffold emits the bot: `jerry init` adds the comment job to the CI variant for the chosen forge, documented in the emitted `CONTRIBUTING.md`, with the token and its required scope stated | §6, §7.2 | `bot` | M |
| `forge-port` | Port `forge-comment` and `bot-scaffold` to the second forge, once the first is in use | §7, §7.2 | `bot-scaffold`, `adoption-report` | M |

A bot that posts a fresh comment per push is a bot people mute. Create-or-update in
`forge-comment` is a correctness requirement, not a nicety — worth stating in the ticket body so
a refinement does not "simplify" it into an append.

Two things in `bot` are easy to drop in refinement and must not be. **The counter**: a few lines
that make §9's adoption question answerable instead of an opinion; deferring it to its own later
ticket means the measurement arrives after everything it was meant to inform. **The no-token
no-op**: a docs tool must never be the reason a merge request cannot merge, so an absent or
unscoped token degrades to silence, not to a red pipeline.

The token itself is the first thing in this design needing write scope on merge requests, and it
is the first row here that a security review can block. `bot-scaffold` documents the requirement;
it does not provision anything.

## Build step 3 — Reading the corpus without opening it

**Outcome.** The corpus is queryable from a terminal, offline, with no service to operate.
Someone onboarding onto a system can read its accepted decisions in one command; someone writing
a design can check for a contradicting decision before writing it — which is the review trigger
the scaffolded README already promises (§1, §7.2).

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| `search` | `jerry search <query>`: full-text plus field filters (`--status`, `--scope`, `--team`, `--since`), ranked, offline | §7.2 | `release` | M |
| `show` | `jerry show <ref>`: render one decision in the terminal, resolve and display its supersession chain | §4.3, §7.2 | `release` | S |

## Build step 4 — Find out whether any of it is read

**Outcome.** An answer to the design's own open question (§9): is the read side used? The
number is *share of merge requests touching governed paths where a decision was surfaced*, and
*share of design reviews citing an existing ADR* — explicitly not ADR count, which rewards
writing trivial ADRs. This step is allowed to conclude that the honest next move is to stop,
and that conclusion is worth more than the features it would cancel.

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| `adoption-report` | `jerry stats` over the log `bot` already writes: coverage and trend, and DESIGN.md §9 updated with the answer and a keep/stop recommendation | §9, §7.2 | `bot` | S |

The log is a file in the repository, not a metrics backend (§3.1). If measuring adoption
requires standing up a service, the measurement is more expensive than the thing it measures.

This row shrank from `M` to `S` because the instrumentation moved into `bot`, where it costs
almost nothing, leaving this row to do only the reading and the deciding. It now also sits
**before** the two largest read-side rows rather than after them.

## Build step 4a — The browsable corpus *(gated on step 4)*

**Outcome.** The corpus is browsable in a browser, offline, with nothing to operate. Deliberately
behind the measurement: this is the largest piece of read-side work in the design and the one
whose value depends most on the answer to §9 (DESIGN v2 §7.2).

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| `site` | `jerry site`: static searchable HTML — per-scope views, no server, deployable to GitLab/GitHub Pages | §3.1, §7.2 | `search`, `adoption-report` | L |
| `site-graph` | The decision graph as a view in the site: supersession chains rendered, orphans and dangling pointers visible | §4.2, §4.3, §7.2 | `site`, `graph` | M |

Split out of v1's single `L` row. A static searchable site and a rendered decision graph share a
deployment target and nothing else; bundling them is what made the row unestimatable. `site-graph`
also wants `graph`'s traversal logic rather than a second implementation of it.

## Build step 5 — More than one repository

**Outcome.** A single index across an estate. Decisions can live wherever they are authored and
still be found in one place, with a repository that is unreachable degrading to its last known
state rather than silently vanishing from the index. This is also, deliberately, the mechanism
that would make the "per-service `docs/decisions/`" reversal in §9 possible without rewriting
anything (§7.3).

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| `crawl` | Repository registry (`jerry.repos.yaml`) + `jerry crawl`: extend the forge client with pagination and rate-limit handling, fetch document directories through the API without cloning; a failed repository keeps its previous entries, flagged stale with the timestamp of its last successful crawl, and never empties the index | §7.3 | `forge-comment`, `adoption-report` | L |
| `corpus-artifact` | Aggregate corpus artifact: versioned `jerry.corpus/1` JSON, `jerry index` over the aggregate, per-scope and per-repository views | §7.3 | `crawl` | M |
| `cross-repo-refs` | Cross-repo reference resolution: `jerry validate --corpus <artifact>` resolves references into other repositories, **advisory not blocking** — a link can break through no fault of the merge request in front of you | §4.3, §7.3 | `corpus-artifact` | M |
| `deletion-detect` | Deletion detection: diff successive crawls; a vanished `Accepted` decision is an alert, not a silent gap | §7.3 | `corpus-artifact` | S |

An explicit registry is preferred over topic-based discovery because it can tell you a service
is *missing* decisions, which discovery by definition cannot (§7.3).

`crawl` is where the stale-not-empty guarantee of DESIGN §7.3's on-call boundary is actually
implemented. The timestamp is not decoration: a consumer that cannot tell a stale corpus from a
current one has no way to bound the damage of a crawl that quietly stopped running.

## Build step 6 — Every finding gets an owner

**Outcome.** Staleness stops being a report nobody reads. Each stale or drifting decision
resolves to an owner and lands in their queue as an issue, and drift is evidenced by what
actually changed in the code rather than by the calendar. A check with no routing target
produces a list nobody actions (§7.3).

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| `owners` | `jerry owners`: resolve a document to its owners from CODEOWNERS (both forges' matching rules, last-match-wins) with a pluggable service-catalogue lookup — and, if service ids return to `applies_to`, the resolution rule for them | §4.1, §6, §7.3 | `corpus-artifact` | M |
| `stale-assign` | `jerry stale --assign`: open or update one issue per stale item against the resolved owner, idempotently; never a second issue for the same document | §7.3 | `forge-comment`, `owners` | M |
| `drift` | Evidence-based drift: compare git churn in a decision's `applies_to` paths since its date; report "N commits since this was recorded" as a ranked signal | §7.3 | `applies-to-match`, `crawl` | L |

`drift` is the one genuinely novel check in the design. It is also the one most likely to be
noisy on first contact with a real repository — expect the ticket to spend most of its effort
on the threshold, not the churn count.

This step is where the warning-with-nowhere-to-go problem from DESIGN §5 is finally closed. Until
then the interim is a warning count on a successful `validate` run, which belongs with
`golden-tests` in step 0 rather than waiting for this step.

## Build step 7 — Corpus integrity at estate scale

**Outcome.** The shape of the decision record is inspectable: supersession chains resolve,
cycles and orphans are found, and a repository being decommissioned does not take its decisions
with it. Dead services' decisions are the ones most worth keeping (§7.3).

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| `graph` | `jerry graph`: supersession chains, cycle and orphan detection, dangling pointers, DOT and mermaid export | §4.2, §4.3, §7.3 | `corpus-artifact` | M |
| `archive` | `jerry archive <repo>`: extract a decommissioning repository's decisions into the central archive with provenance preserved | §7.3 | `crawl` | M |

## Build step 8 — Integrations

**Outcome.** jerry's findings appear in the tools people already have open: inline on the diff,
in the service catalogue, on a Confluence page for readers who will never open a repository,
and in a Slack digest. None of it is new capability — it is distribution for capability that
already exists (§7.4).

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| `sarif` | Forge annotations: upload SARIF from `jerry validate` so findings appear inline on the diff | §5, §7.4 | `forge-comment` | M |
| `backstage` | Backstage: emit catalog entities and annotate services with their governing decisions | §7.4 | `corpus-artifact` | M |
| `confluence` | Confluence one-way publish: render decisions to pages for non-engineer readers, with a stated no-two-way-sync boundary in the ticket body | §7.4, §8 | `corpus-artifact` | M |
| `slack` | Slack: new-decision digest and owner nudges for stale items | §7.4 | `stale-assign` | S |

`confluence` is the ticket most likely to attract a "while we're here, sync it back" suggestion.
Two-way sync is a stated non-goal (§8, §2).

Revisit the bus factor before this step (§9). Four integrations are four more things one person
maintains, and none of them adds capability.

## Build step 9 — Lowering the cost of writing one

**Outcome.** Recording a decision stops being a blank page, and decisions that are being made
without a record get caught at the moment they are made. This is the other half of the §1
argument: step 2 makes records worth reading, this makes them cheap to write (§7.5).

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| `draft` | `jerry draft --from-mr <id>`: skeleton from the diff and description — prefill Context, prompt for Alternatives, and **never** generate Decision or Consequences | §7.5 | `forge-comment` | L |
| `undocumented` | Undocumented-decision detection: flag merge requests adding a dependency, a datastore, an external integration, or changing a base image, and ask whether an ADR is warranted | §7.5 | `bot` | L |

A machine-written Consequences section is worse than an empty one, because it looks like
thinking happened (§7.5). That constraint belongs in `draft`'s body, not only here.

## Cross-cutting — one per build step, not a parking lot

**Outcome.** The difference between a tool that works and one that stays working: a `--diff`
path that survives the shallow clones CI actually produces, a schema that can change without
invalidating every existing document, an upgrade path for a pinned version, and a manual that
keeps pace with the commands. None of it is a feature; without it the steps above decay.

v1 of this file parked four rows here with no scheduling rule, which is where infrastructure
tickets go to be permanently outranked by features. **The rule is one cross-cutting row per
build step**, taken in the order below. `golden-tests` has already been pulled into step 0
under it.

| slug | title | design refs | depends-on | size |
|---|---|---|---|---|
| `diff-hardening` | Harden `validate --diff`. **First, the live defect:** findings carry corpus-root-relative paths while `git diff --name-only` emits repository-root-relative ones, so when `jerry.yaml` is not at the git root every finding is filtered out and `validate --diff` exits 0 reporting nothing. Then base-ref autodetection, shallow-clone and detached-HEAD behaviour, and a clear failure when the base ref is absent | §5, §10 | `golden-tests` | M |
| `schema-tolerance` | `schema_version` tolerance: implement DESIGN §3.6's rule — read at-or-below as that version, warn (never error) above — and fix `jerry schema`, which publishes `const: 1` and will reject every v2 document the day v2 exists | §3.6, §4.1, §10 | `release` | M |
| `upgrade-ci` | `jerry upgrade-ci`: bump one repository's pinned jerry version in its emitted CI, run per repository by its owners as an ordinary reviewed merge request. Without it a *bugfix* can never reach a scaffolded repo — no repo scaffolded by `0.1.0` will ever pick up `0.1.1`'s install-path fix | §6 | `ci-binary-install` | S |
| `release-automation` | Turn `RELEASING.md` into a checked procedure: `release` was a manual verification exercise, and every subsequent release currently repeats it by hand | RELEASING.md | `release` | S |
| `placeholder-escapes` | Placeholder rule false positives: exclude fenced code blocks from the scan and add an inline `<!-- jerry:allow placeholder -->` opt-out, so the cheapest fix for a false positive stops being "delete the phrase from `jerry.yaml`", which disables the check repository-wide | §5, §10 | `golden-tests` | S |
| `config-additive` | Make the list-valued `jerry.yaml` keys additive: a repository extends `placeholders` and the required-section lists rather than replacing them, so no repository can switch a rule off (DESIGN v2 §3.2) | §3.2, §10 | `release` | S |
| `manual-restructure` | Manual restructure: split `docs/user-manual/introduction.adoc` into per-capability chapters as the command surface grows past one page, and add the chapter-per-phase discipline to AGENTS.md | §7 | `site` | S |

`diff-hardening` and `placeholder-escapes` are the two rows here that are already costing
something. The first is a silent pass — a green tick taken as evidence when nothing was checked
— and the second is a standing invitation to disable the design's highest-value rule.

---

## Deliberately not tickets

Recorded so they are not reintroduced by a well-meaning refinement:

- **A server, a database, or any hosted component** — cut (§2, §3.1, §8). Every output is a
  file. The moment this grows a service, somebody owns an on-call rotation for a docs tool.
- **Two-way Confluence sync** — cut (§8). One-way publish only; `confluence` states the boundary.
- **An approval workflow engine** — cut (§8). Merge request approval already is one.
- **Custom identity or auth** — cut (§8). Use the forge's.
- **Prose reflow in `jerry fmt`** — cut (§7.1). Deciding what is a paragraph versus a list item,
  table row or fenced block is where markdown formatters corrupt authored text. Frontmatter is
  structured data with a canonical form; prose is not.
- **Per-team ADR maturity dashboards or ADR counts as a metric** — cut (§8, §9). Counting ADRs
  produces trivial ADRs. `adoption-report` measures reading, not writing.
- **Semantic conflict detection between decisions** — cut (§8). jerry checks that references
  resolve, not that two decisions agree.
- **An estate-wide sweep that bumps every repository's pinned version at once** — cut (§6). That
  is retroactive rule enforcement wearing a version number. Note the change from v1 of this
  file, which cut version bumping altogether and thereby cut the only route by which a bugfix
  reaches a scaffolded repository: the *per-repository, owner-reviewed* bump is `upgrade-ci`
  above, and it is needed. Retroactivity is the thing being refused, not upgrading.
- **Rejecting unknown frontmatter keys.** `jerry fmt` preserves keys it does not know, and a
  tool that silently deletes what it does not understand cannot be trusted to write files at
  all. `applies-to-validate` adds a *warning*, never an error.

## Open questions that gate specific rows

From **§9 Open questions**. These need a decision, not design work — and two of them can
invalidate whole build steps, which is why they are listed rather than assumed.

| Open item | Blocks / shapes |
|---|---|
| Does `teams/` survive a reorganisation? | Gates an unfiled `layout-flat` row (§7.1 shipped `teams/` only). Shapes `crawl`'s registry, which is where a layout change would have to be absorbed. Answer before step 5, or the registry encodes a layout we are about to abandon |
| Is central authoring right at all? | Shapes `crawl`–`cross-repo-refs`. `crawl` is the per-service reversal in everything but name; if the answer is "per-service", step 5 becomes the main line rather than an aggregation feature, and §6's scaffold contract changes with it |
| Per-folder vs global ids | Would delete the duplicate-id rule and the scope-qualification rule from §5 outright. Changing it after step 5 means rewriting references across the estate, so **decide before `corpus-artifact`**. DESIGN v2 §9 adds a third option — keep the short id as the human reference and add an immutable key — which turns both this question and the `teams/` one into a field migration rather than an estate-wide reference rewrite. Worth filing as a row in its own right if it is chosen |
| What metric governs Phase 2? | `adoption-report` — that row exists to answer it, and is allowed to answer "stop". The instrumentation is no longer gated on it: `bot` counts from its first commit |
| Bus factor | **Answered in DESIGN v2 §9, and it now gates something.** Phase 2 must show the read side being used before step 5 starts, and a second maintainer is a precondition for step 5 rather than a nice to have. Revisit again before step 8, which is four integrations and no new capability |

Two items that were open in v1 are now decided in DESIGN v2 and are no longer listed: the forge
strategy (one at a time, port after adoption — §7) and the meaning of a `schema_version` bump
(§3.6, implemented by `schema-tolerance`).
