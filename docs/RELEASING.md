---
summary: 'Release checklist for sag (GitHub release + Homebrew tap)'
---

# Releasing sag

Follow these steps for each release. Title GitHub releases as `sag <version>`.

## Checklist
- Update CLI version in `cmd/root.go` (`Version` field).
- Update `CHANGELOG.md` with a section for the new version.
- Run the gates: `pnpm format && pnpm lint && pnpm test && pnpm build`.
- Tag the release: `git tag -a v<version> -m "Release <version>"` after commits; push tags with `git push origin main --tags`.
- Create source archive (used by Homebrew):
  - `git archive --format=tar.gz --output /tmp/sag-<version>.tar.gz v<version>`
- Update Homebrew tap formula (`../homebrew-tap/Formula/sag.rb`):
  1. Set `version "<version>"`.
  2. Set `url "https://github.com/steipete/sag/archive/refs/tags/v<version>.tar.gz"`.
  3. Update `sha256` using `shasum -a 256 /tmp/sag-<version>.tar.gz`.
  4. Ensure build step uses `system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/sag"`.
  5. Commit and push in tap repo: `git commit -am "sag v<version>" && git push origin main`.
- Verify Homebrew install from tap:
  - `brew update && brew reinstall steipete/tap/sag`
  - `brew test steipete/tap/sag`
  - `sag --version`
- Create GitHub release for `v<version>`:
  - Title: `sag <version>`
  - Body: bullets from `CHANGELOG.md` for that version
  - Assets: source tarball `/tmp/sag-<version>.tar.gz` (mention SHA256 in body)
- Smoke-test CLI locally: `sag --help`, `sag voices --limit 3`, `sag -v Roger "hello"`.
- Announce: optional note with `brew update && brew upgrade steipete/tap/sag`.
