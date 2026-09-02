---
id: JRY-003
title: Emitted CI installs a checksum-verified release binary instead of go install
project: jerry
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# JRY-003 — Emitted CI installs a checksum-verified release binary instead of go install

## Outcome

A repository scaffolded by `jerry init` validates its documents in CI without a Go toolchain
and without reaching the module proxy: the pipeline downloads the pinned jerry release binary
for the runner's platform, verifies it against the published checksums file, and runs it. A
runner image with nothing but `curl` and `sh` is enough.

## Description

DESIGN.md §3.3 states jerry "must run in any CI image and in a pre-commit hook without a
toolchain behind it". The CI `jerry init` actually emits contradicts that: both templates
(`internal/scaffold/templates/github/.github/workflows/docs.yml`,
`internal/scaffold/templates/gitlab/.gitlab-ci.yml`) run

    go install github.com/codcod/jerry/cmd/jerry@__JERRY_VERSION__

inside a `golang:1.26` image, on every pipeline run. That needs a Go toolchain present and the
module proxy reachable, compiles jerry from source on each run, and discards the single-binary
property at the one place it was supposed to pay for itself. Recorded as divergence 2 in
DESIGN.md §10.

JRY-001 already verified that goreleaser publishes per-platform archives **and** a checksums
file, so the artifacts this needs exist and are pinned by tag. The change is to the two emitted
CI templates plus whatever token substitution `internal/scaffold/scaffold.go` needs (it already
substitutes `__JERRY_VERSION__` in `replaceTokens`); the `go install` line stays in the emitted
`CONTRIBUTING.md` as a documented fallback for anyone who wants it.

Two things to settle in refinement: how the runner's platform is detected in a shell one-liner
that works on both forges, and what the failure message says when the pinned tag has no release
asset — a scaffolded repo must fail loudly there, not silently fall back to `@latest`.

**Sequencing.** Should land before JRY-004 (real-forge proof), because it deletes two of the
failure modes JRY-004 exists to detect rather than proving them. Not recorded in `depends-on:`
pending approval — JRY-004's own text says it can still run, just less usefully, if this slips,
which is a soft coupling, not a hard one.

**Refinement update (2026-09-02).** The two open questions are settled below: platform is
detected with a `uname -s`/`uname -m` case/esac (linux/darwin × amd64/arm64, matching
goreleaser's published matrix); a missing release asset for the pinned tag/platform fails the
job loudly and points at the documented `go install`/`brew` fallback, never retrying `@latest`.
A third question surfaced during refinement — the GitLab template's default image is
`golang:1.26`, which defeats the point once `go install` is gone — is also settled: it becomes
`alpine:3.20` (user-confirmed 2026-09-02).

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd /Users/nicos.karagieorgopulus/Projects/private/jerry
git checkout main
git checkout -b feat/JRY-003-checksum-verified-ci-install
```

Root-path child (`path = "."`): tidy WIP commits into atomic ones before presenting, per the
project's commit policy.

### Prerequisite gate (hard)

None. The released binaries and `checksums.txt` this ticket downloads already exist — JRY-001
(`6-done/`, merged to `main`) cut `v0.1.0`/`v0.1.1` and confirmed goreleaser publishes both. No
ticket needs to land first.

### Confirmed design decisions (do not deviate without asking)

1. **The GitHub template drops `actions/setup-go@v6` entirely.** No Go toolchain is used by
   this job once `go install` is gone.
2. **The GitLab template's default image changes from `golang:1.26` to `alpine:3.20`,** with
   `apk add --no-cache curl` added to `before_script` (user-confirmed 2026-09-02). Alpine's
   busybox already provides `sh`, `tar`, `sha256sum`, `grep`, `sed` and `mktemp`; only `curl` is
   missing.
3. **Both forges carry the identical install script, duplicated verbatim** (differing only in
   the final line that puts the extracted binary on `PATH`). `internal/scaffold/scaffold.go`'s
   package doc is explicit that the emitted repo ships no `scripts/` directory jerry's rules
   could drift from, so the script cannot be factored into a shared file inside the scaffold —
   the duplication mirrors the `go install` line's status quo before this ticket.
4. **The binary installs to a per-run temp directory added to `PATH`,** not `/usr/local/bin` —
   `echo "$TMP" >> "$GITHUB_PATH"` on GitHub, `export PATH="$TMP:$PATH"` on GitLab (GitLab
   concatenates `before_script` and `script` into one shell invocation per job, so the export
   persists). This avoids assuming the runner is root or has passwordless `sudo`.
5. **Platform detection is `uname -s` (`Linux`→`linux`, `Darwin`→`darwin`) × `uname -m`
   (`x86_64`/`amd64`→`amd64`, `aarch64`/`arm64`→`arm64`),** matching `.goreleaser.yaml`'s
   `builds.goos`/`goarch`. Anything else exits 1 with a message naming the unsupported
   platform and pointing at `CONTRIBUTING.md`'s `go install`/`brew` fallback.
6. **`__JERRY_VERSION__`'s substitution is unchanged** (`internal/scaffold/scaffold.go:173`
   `replaceTokens` still emits the bare pin — `vX.Y.Z`, or the literal string `latest` only when
   jerry itself was a dev/dirty build with nothing to pin to). Only the template text around the
   token changes, from a `go install …@__JERRY_VERSION__` suffix to a shell variable assignment
   `JERRY_VERSION=__JERRY_VERSION__`. When the pin is literally `latest`, the script first
   resolves the real tag via
   `curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/codcod/jerry/releases/latest`
   and strips the `/tag/` prefix with `sed` — GitHub's asset filenames are versioned, so a bare
   `latest` alias cannot address one directly.
7. **A missing release asset (curl failure fetching the archive) fails the job immediately** with
   a message naming the pinned tag and detected platform and pointing at the documented
   fallback — it must never silently retry `@latest`.
8. **Checksum verification is `grep " ${ARCHIVE}\$" checksums.txt | sha256sum -c -`** — present
   in both Ubuntu's coreutils and Alpine's busybox; no macOS `shasum` branch, since neither
   emitted template targets a macOS-hosted runner.
9. **`go install github.com/codcod/jerry/cmd/jerry@latest` stays, unchanged, in the emitted
   `CONTRIBUTING.md`** as the documented fallback for anyone without network access to GitHub
   Releases. This ticket touches only the CI install step.

The shared install script body (identical in both templates except the last line):

```sh
set -eu
JERRY_VERSION=__JERRY_VERSION__
case "$(uname -s)" in
  Linux) JERRY_OS=linux ;;
  Darwin) JERRY_OS=darwin ;;
  *)
    echo "jerry install: unsupported OS $(uname -s) -- released binaries cover linux and darwin. See CONTRIBUTING.md for a source install." >&2
    exit 1
    ;;
esac
case "$(uname -m)" in
  x86_64|amd64) JERRY_ARCH=amd64 ;;
  aarch64|arm64) JERRY_ARCH=arm64 ;;
  *)
    echo "jerry install: unsupported architecture $(uname -m) -- released binaries cover amd64 and arm64. See CONTRIBUTING.md for a source install." >&2
    exit 1
    ;;
esac
if [ "$JERRY_VERSION" = "latest" ]; then
  JERRY_VERSION=$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/codcod/jerry/releases/latest | sed 's#.*/tag/##')
fi
ARCHIVE="jerry_${JERRY_VERSION#v}_${JERRY_OS}_${JERRY_ARCH}.tar.gz"
BASE_URL="https://github.com/codcod/jerry/releases/download/${JERRY_VERSION}"
TMP=$(mktemp -d)
curl -fsSL -o "$TMP/$ARCHIVE" "$BASE_URL/$ARCHIVE" || {
  echo "jerry install: no release asset $ARCHIVE at tag $JERRY_VERSION ($JERRY_OS/$JERRY_ARCH). See CONTRIBUTING.md for a source install." >&2
  exit 1
}
curl -fsSL -o "$TMP/checksums.txt" "$BASE_URL/checksums.txt"
( cd "$TMP" && grep " ${ARCHIVE}\$" checksums.txt | sha256sum -c - )
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
```

### Tasks

#### Task 1 — Rewrite the GitHub install step

`internal/scaffold/templates/github/.github/workflows/docs.yml`: delete the
`actions/setup-go@v6` step. Replace the `install jerry` step's `run:` with the shared script
above (decision 6) followed by `echo "$TMP" >> "$GITHUB_PATH"`. Rewrite the explanatory comment
above it to describe checksum-verified download instead of `go install`; the old aside
recommending `brew` on a macOS runner can go — the new script handles `darwin` itself
(decision 5).

#### Task 2 — Rewrite the GitLab install step and default image

`internal/scaffold/templates/gitlab/.gitlab-ci.yml`: change `default.image` from `golang:1.26`
to `alpine:3.20` (decision 2). Replace the single `go install …` line in `before_script` with
`apk add --no-cache curl` followed by the shared script above (decision 6) and
`export PATH="$TMP:$PATH"`. Update the comment above `before_script` to match; drop the macOS
aside for the same reason as Task 1.

#### Task 3 — Confirm `scaffold.go` needs no change

`internal/scaffold/scaffold.go`: read `replaceTokens` (line 173) and confirm it still substitutes
the bare pin (`vX.Y.Z` or `latest`) with no code change needed — decision 6 depends on this. If
it turns out something does need to change, treat that as a plan deviation and stop to ask
before proceeding.

#### Task 4 — Update the version-pinning test to the new template shape

`internal/scaffold/scaffold_test.go`'s `TestVersionPinning` (~line 111) currently asserts
`github.com/codcod/jerry/cmd/jerry@vX.Y.Z` appears in `docs.yml`. Change the assertion to look
for `JERRY_VERSION=vX.Y.Z` (and `JERRY_VERSION=latest` for the two fallback cases) instead,
keeping all four existing cases and the existing "version token was left unreplaced" check.

#### Task 5 — Reconcile DESIGN.md

`DESIGN.md` §6's paragraph starting "The pin is to a **released artifact, verified by
checksum**" currently describes the `go install` behaviour as the present state contradicting
the charter — rewrite it to describe the checksum-verified download as shipped. In §10's
divergence table, remove row 2 (`jerry needs no toolchain behind it`) — it is no longer a
divergence — and renumber nothing else (the table has no ordinal cross-references, so removing
one row is safe). Add a `## 11. Revision history` line for this correction, following the
existing entries' style.

### Acceptance test

From the `jerry` repo root, on the feature branch:

1. `just test` — includes the updated `TestVersionPinning` and the still-passing
   `TestScaffoldValidatesClean`/`TestNoScriptsDirectoryIsEmitted`.
2. `just lint` and `just docs-check` clean.
3. Real download-and-verify, GitHub shape: extract the shared script (Task 1's version, minus
   the `$GITHUB_PATH` line) into a local file, run
   `JERRY_VERSION=v0.1.1 sh ./install.sh` in a plain POSIX shell, and confirm `$TMP/jerry
   version` prints `0.1.1`.
4. Real download-and-verify, GitLab shape, inside the actual target image:
   `docker run --rm alpine:3.20 sh -c "apk add --no-cache curl && <script>"` with
   `JERRY_VERSION=v0.1.1`, confirming the extracted `jerry version` prints `0.1.1` inside the
   container.
5. Failure path: re-run step 3 with `JERRY_VERSION=v0.0.1-does-not-exist` and confirm it exits
   non-zero with the "no release asset" message — not a silent `@latest` retry.
6. `jerry init --forge github` and `jerry init --forge gitlab` into two scratch directories from
   a binary built at a real tag (e.g. `just build` after `git checkout v0.1.1` in a worktree, or
   patch `Version` via the `just build` ldflags) — confirm the emitted `docs.yml`/`.gitlab-ci.yml`
   contain the new script with the tag substituted and no `__JERRY_VERSION__` token left.

### Docs update (mandatory when user-facing)

`DESIGN.md` §6 and §10 (Task 5, above) — no other user-facing docs describe the emitted CI's
install mechanism (`README.md` and `docs/user-manual/introduction.adoc`'s `go install` lines are
about installing jerry itself for local development, unaffected).

### Finish (mandatory)

1. Acceptance test green; `just build`/`test`/`lint`/`docs-check` clean.
2. `DESIGN.md` updated per Task 5.
3. Write a summary (files touched: the two templates, `scaffold_test.go`, `DESIGN.md`; decisions
   made; anything deferred).
4. Suggested commit message: `fix(scaffold): pin emitted CI to a checksum-verified release
   binary (JRY-003)`.
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; publish only per the project's commit policy (no push/MR without approval).
   `pickle ticket move JRY-003 in-review --reason "acceptance green"` and hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new
- 2026-09-02 — TO DO → READY: plan complete
- 2026-09-02 — READY → IN DEVELOPMENT: picked up
