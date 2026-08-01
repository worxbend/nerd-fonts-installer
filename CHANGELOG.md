# Changelog

## Unreleased

### Added

- A GitHub Pages microsite in `docs/`, deployed by the `Pages` workflow, with
  screenshots, a quick start, a config and flag reference, and an FAQ.
- Terminal screenshots in `assets/screenshots/`, generated from real runs of the
  tool by the harness in `scripts/screenshots/`.
- A social preview card (`assets/social-card.svg` / `.png`).

### Fixed

- The interactive picker rendered one row taller than the terminal, so Bubble
  Tea truncated the top of the view and the banner's top border never appeared.
  The layout now fits at every terminal size: the side panel is dropped when it
  cannot fit beside the list, and the banner goes compact on short terminals.

### Changed

- Rewrote the README around the new screenshots, with collapsible sections for
  the long-form material that now lives in the wiki.
- Bumped the `govulncheck` toolchain pin to `go1.26.5`, which carries the fix for
  GO-2026-5856 in `crypto/tls`. `make verify` failed on the stale pin; CI already
  resolved `1.26.x` to a patched toolchain.

## v1.0.6 - 2026-06-29

### Fixed

- Allow installing large Nerd Font families such as Noto by raising the
  bounded download and extraction size caps while preserving oversized archive
  protections.
