# Releasing jerry

This process mirrors pickle's own `RELEASING.md`, the golden sample for every Go
project in this family.

## One-time setup

- A `codcod/homebrew-tap` repository.
- A repository secret `HOMEBREW_TAP_GITHUB_TOKEN`: a PAT with `repo` scope on
  the tap, so goreleaser can push the updated formula.

## Cutting a release

1. Retitle `[Unreleased]` in `CHANGELOG.md` to `[X.Y.Z] - YYYY-MM-DD`, add a
   fresh empty `[Unreleased]`, and update the link references at the bottom.
   Reconcile by hand against `tickets/6-done/` — jerry has no
   `pickle changelog check` equivalent.
2. Commit. The tag should include the changelog.
3. `just dist-check` and `just test` locally.
4. Tag and push:

   ```sh
   git tag v0.1.0 && git push origin v0.1.0
   ```

`release.yml` runs goreleaser: binaries and checksums for darwin/linux/windows
on amd64 and arm64, a GitHub release, and the tap formula.

`docs-release.yml` then attaches the user manual as PDF and EPUB. It is
deliberately `continue-on-error`: a broken manual must never block a release.

## What a release means for scaffolded repositories

`jerry init` stamps the version of the binary that ran it into the CI it emits
(`go install github.com/codcod/jerry@vX.Y.Z`). A repository therefore keeps
being checked against the rules it was created with until someone deliberately
bumps the pin. A development build (`dev`, or a `-dirty` describe) falls back to
`@latest`, because there is no released version for it to pin.

Rule changes are therefore **not** retroactive across the estate, which is the
intended behaviour: a new check must not turn every existing repository red the
day it merges.
