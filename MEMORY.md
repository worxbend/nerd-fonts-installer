# Project Memory — nerd-fonts-installer

> **Protocol for all agents (Claude, Codex, others):** Read this file at the
> start of every session *before* touching code. After any change that alters
> architecture, a package boundary, a public contract, an invariant, or the
> dev/build workflow, **update this file in the same change**. Keep it short and
> current — it is a map, not a changelog. See [AGENTS.md](AGENTS.md).

## What this is

`nerd-fonts-installer` is a small, scriptable Go CLI that installs
[Nerd Fonts](https://github.com/ryanoasis/nerd-fonts) from a config file or an
explicit interactive TUI: it resolves a release, downloads per-family zip
archives from GitHub, extracts the font files into the user's font directory,
and refreshes the font cache. Audience: dotfiles / fresh-machine /
dev-container setup.

## Architecture (package map)

- `cmd/nerd-fonts-installer` — entrypoint & orchestration.
  - `run(ctx, args, streams, deps) int` is the real, testable main; `main()`
    only wires os values into it. The command package bundles stdin/stdout/stderr
    into `ioStreams` and passes small option structs through helpers to keep CLI
    plumbing testable without high-parameter helper signatures.
  - Dependency injection via the `dependencies` struct of **function-typed
    fields** (`loadConfig`, `discoverConfig`, `defaultConfigPaths`,
    `listReleases`, `runTUI`, `installFonts`, `isTerminal`) + `withDefaults()`.
    This is deliberately NOT an interface seam — keep it that way for a CLI
    this size.
  - CLI shape: bare `nerd-fonts-installer` still runs the install flow, but the
    command now also exposes `install`, `list`, `info`, `version`, and
    `completion` subcommands. Dispatch checks the first positional token after
    any leading root flags; legacy `--font-names` and `--version` remain
    supported as backward-compatible aliases.
  - Exit-code contract (`exitCodeFor`): `0` success/cancelled, `2`
    user-correctable input (missing config, unknown/absent release), `1`
    runtime failure (network/fs/install).
  - `version/commit/date` are injected via `-ldflags -X main.*`.
- `internal/config` — YAML/JSON-compatible config loading. YAML and `.conf`
  use the maintained `go.yaml.in/yaml/v3` module with strict `KnownFields(true)`
  decoding; JSON uses `DisallowUnknownFields`.
  `ApplyDefaults` → `Normalize` → `Validate`. `Discover`/`DefaultPaths` search
  app-named files in CWD first, then `$XDG_CONFIG_HOME` when absolute, otherwise
  `$HOME/.config`: `nerd-fonts-installer.{yaml,yml,json,conf}` and
  `nerd-fonts-installer/config.{yaml,yml,json,conf}`.
- `internal/nerdfonts` — GitHub releases API client. Zero-value-with-defaults
  `Client` (configurable `HTTPClient`/`BaseURL`/`MaxPages`). Owns `Latest`
  const and typed errors `ErrNoReleases`, `ReleaseNotFoundError`. Pagination
  stops on an empty *raw* API page, not a filtered-empty one.
- `internal/fonts` — the install engine. Per family: download → extract → atomic
  replace → optional `fc-cache`. Families install **concurrently** (errgroup,
  limit `maxConcurrentInstalls`); a `syncWriter` keeps progress lines from
  interleaving. Size caps (`maxDownloadBytes`/`maxFontFileBytes`/
  `maxArchiveBytes`) bound zip bombs.
- `internal/fontname` — the single shared path-traversal validator
  (`Validate`), used at both trust boundaries (`config` and `fonts`). Security
  boundary — do not duplicate it.
- `internal/tui` — Charm v2 (`charm.land/{bubbletea,bubbles,lipgloss}/v2`)
  picker: release step → families step, with
  `IconMode` (auto/nerd/unicode/ascii) icon sets. The frame is **height
  budgeted** (`chromeHeight`/`compactChromeHeight`/`minListHeight`): a view even
  one row taller than the terminal makes Bubble Tea truncate the top of the
  frame. `screenBody` drops the side panel when it will not fit beside the list;
  `compactBanner` drops the banner's subtitle and badges on short terminals.
  `TestViewFitsTerminalHeight` guards the arithmetic. Lip Gloss v2 widths
  include borders and padding; subtract `GetHorizontalFrameSize()` when
  sizing inner content. Bubble Tea uses `tea.KeyPressMsg` and `tea.View`, with
  alternate-screen mode declared on the picker view. Installer progress uses
  per-output colorprofile writers before `syncWriter` serializes worker output.
- `docs/` — the GitHub Pages microsite (plain HTML/CSS/JS, no build step). It
  references shared repository assets, so `.github/workflows/pages.yml` publishes
  `docs/` **plus** `assets/` rather than duplicating files.
- `assets/screenshots/` — SVG terminal screenshots used by the README and the
  site, generated from real runs by `scripts/screenshots/` (pty capture with
  `pyte` → SVG render; `bwrap` shadows `$HOME` so a genuine install lands in a
  throwaway directory). Regenerate with `scripts/screenshots/refresh.sh`.
- `snap/snapcraft.yaml` — Snapcraft packaging for the CLI. It builds the Go
  command with the same ldflags contract and uses classic confinement so the
  tool can write real user font directories and refresh fontconfig.

## Invariants & design tenets (do not break)

1. **Family-name trust boundary:** every family name passes
   `fontname.Validate` before it is joined onto a path or URL. One
   implementation, used everywhere.
2. **Atomic per-family install:** stage into a unique temp dir, `rename` into
   place, keep a `.old` backup. Disjoint per-family paths are what make
   concurrency safe — preserve that if you touch the install loop.
3. **Best-effort cleanup must never fail a committed operation** (e.g. removing
   the `.old` backup after a successful swap).
4. **Surface write-close/flush errors** on written font files (`Sync` + explicit
   `Close`) — a swallowed close can promote a truncated font.
5. **Testability:** `run` takes explicit I/O; HTTP goes through an injectable
   `*http.Client` (tests use `roundTripFunc` + in-memory zips, or `httptest`).
6. **Security defaults:** strict config decoding, `url.PathEscape` on URL
   segments, `exec.CommandContext` (no shell) for `fc-cache`, size caps on all
   copies.
7. **Download integrity:** each zip is verified against the release's
   `SHA-256.txt` manifest (`fetchChecksums` → per-family digest). Verification
   is *best-effort*: a missing manifest warns and proceeds; a digest **mismatch
   aborts** the install. The manifest fetch itself is timeout-bounded and any
   rollback failure during the atomic swap must be surfaced with recovery
   context. Do not weaken the mismatch-is-fatal rule.

## Dev / build workflow

- Go **1.27.2**. Module: `github.com/worxbend/nerd-fonts-installer`.
- `make verify` runs module/format checks, vet, golangci-lint (v2.14.0), unit
  and race tests, build, govulncheck (v1.9.0), and actionlint (v1.7.12).
  `golangci-lint` is a required check for every code change; `make lint` is the
  focused lint target. **Tests must pass under `-race`** (the install path is
  concurrent).
- CI runs vet, lint, test matrix (ubuntu+macOS), race, coverage, govulncheck,
  actionlint. All Go CI jobs read the toolchain version from `go.mod`; local
  and CI vulnerability/workflow checks use the same Makefile tool pins.
  Mirror that locally before pushing.
- Dependabot checks Go modules, GitHub Actions, and the screenshot harness's
  `requirements.txt` weekly. Update the Makefile tool pins and Snap Go channel
  manually when upgrading tooling; keep the CI lint version in sync.
- Release CI publishes versioned GitHub releases for `v*` tags and refreshes a
  moving `latest` release with stable asset names for fixed download URLs.
- Pages CI (`.github/workflows/pages.yml`) deploys the microsite on pushes to
  `main` touching `docs/` or `assets/`. Repo metadata (description, homepage,
  topics) and the wiki are maintained out-of-tree via `gh`.
- Snap CI builds snaps on PRs/main/tags/manual runs. Non-PR runs publish to the
  Snap Store (`main` → `edge`, `v*` tags → `stable`, manual → chosen channel)
  using the `SNAPCRAFT_STORE_CREDENTIALS` repository secret. The snap name must
  be registered and classic confinement approved before public stable release.
- Commits: Conventional Commits + `Co-Authored-By` trailer. Branch off `main`.
- Reusable Go skills live under `.agents/skills/` (golang-*, testing, security,
  review). The review→refactor methodology is the
  `principal-review-and-refactor` skill.

## Open / deferred decisions (need a human call)

- **Release tooling duplication:** `.goreleaser.yaml` and the hand-rolled bash
  in `.github/workflows/release.yml` are two sources of truth — pick one.
