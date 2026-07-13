<div align="center">

<img src="assets/logo.svg" alt="Nerd Fonts Installer logo" width="140" height="140" />

# Nerd Fonts Installer

**A Nerd Fonts installer for repeatable terminal and editor setup.**

<p>
  <a href="https://github.com/worxbend/nerd-fonts-installer/releases"><img alt="Release" src="https://img.shields.io/github/v/release/worxbend/nerd-fonts-installer?sort=semver&color=7C3AED&label=release"></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/license-MIT-2563EB"></a>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white">
  <img alt="Platform" src="https://img.shields.io/badge/platform-Linux%20%7C%20macOS-555">
  <a href="https://snapcraft.io/nerd-fonts-installer"><img alt="Snap" src="https://img.shields.io/badge/snap-nerd--fonts--installer-82BEA0?logo=snapcraft&logoColor=white"></a>
</p>

</div>

`nerd-fonts-installer` is a Nerd Fonts installer for people who set up
terminals, editors, dotfiles, new laptops, remote dev boxes, or fresh
Linux/macOS machines and do not want to manually download font archives every
time.

Instead of clicking through GitHub releases, unzipping files, moving fonts into
the right folder, and refreshing the font cache by hand, you keep a short YAML
file:

```yaml
release: latest
destination: ~/.local/share/fonts/NerdFonts
refresh_font_cache: true
families:
  - JetBrainsMono
  - Hack
  - FiraCode
  - Meslo
```

Then run:

```bash
nerd-fonts-installer
```

That is it.

## Why You Might Want This

Nerd Fonts are great. Installing them repeatedly is not.

This tool is useful when you:

- want terminal icons and glyphs to work in Starship, Neovim, tmux, lazygit,
  eza, yazi, WezTerm, Alacritty, Kitty, Ghostty, or VS Code
- rebuild machines often
- maintain dotfiles
- bootstrap dev environments
- want the same fonts on every workstation
- want a repeatable setup script instead of manual clicking
- want to preview exactly what will be installed before writing files

Manual install:

1. Open the Nerd Fonts release page.
2. Find the right font archive.
3. Download it.
4. Unzip it.
5. Move only the font files.
6. Put them in a font directory.
7. Refresh the font cache.
8. Repeat for every font.

With `nerd-fonts-installer`:

```bash
nerd-fonts-installer --dry-run
nerd-fonts-installer
```

## What It Does

- Downloads Nerd Font release archives from GitHub.
- Installs only font files: `.ttf`, `.otf`, and `.ttc`.
- Keeps each font family in its own folder.
- Supports `latest` or pinned releases like `v3.4.0`.
- Reads a simple YAML, JSON, or `.conf` config.
- Can discover your config automatically.
- Has an interactive picker with `--interactive`.
- Prints copy-paste-ready font family names for configs.
- Supports dry-runs.
- Refreshes `fc-cache` on Linux when requested.
- Skips `fc-cache` safely if it is not installed.
- Uses colorful CLI output and a Charm Bubble Tea TUI.

## Quick Start for Beginners

### 1. Get the Binary

Download the latest release archive for your system from the fixed `latest`
GitHub release URL.

Pick one:

| System | URL |
| --- | --- |
| Linux Intel/AMD | `https://github.com/worxbend/nerd-fonts-installer/releases/download/latest/nerd-fonts-installer_latest_linux_amd64.tar.gz` |
| Linux ARM64 | `https://github.com/worxbend/nerd-fonts-installer/releases/download/latest/nerd-fonts-installer_latest_linux_arm64.tar.gz` |
| macOS Intel | `https://github.com/worxbend/nerd-fonts-installer/releases/download/latest/nerd-fonts-installer_latest_darwin_amd64.tar.gz` |
| macOS Apple Silicon | `https://github.com/worxbend/nerd-fonts-installer/releases/download/latest/nerd-fonts-installer_latest_darwin_arm64.tar.gz` |

For example:

```bash
curl -LO https://github.com/worxbend/nerd-fonts-installer/releases/download/latest/nerd-fonts-installer_latest_linux_amd64.tar.gz
curl -LO https://github.com/worxbend/nerd-fonts-installer/releases/download/latest/checksums.txt
sha256sum --check --ignore-missing checksums.txt
```

Extract it:

```bash
tar -xzf nerd-fonts-installer_latest_linux_amd64.tar.gz
cd nerd-fonts-installer_latest_linux_amd64
```

Move it somewhere on your `PATH`:

```bash
chmod +x nerd-fonts-installer
mkdir -p ~/.local/bin
mv nerd-fonts-installer ~/.local/bin/
```

Check it works:

```bash
nerd-fonts-installer --version
```

If `~/.local/bin` is not on your `PATH`, add this to your shell config:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

### Snap Package

On Linux, after the Snap Store package is published, you can install it with:

```bash
sudo snap install nerd-fonts-installer --classic
```

The snap uses classic confinement because the CLI installs fonts into the real
user font directory and refreshes fontconfig. Classic confinement requires Snap
Store approval before the package can be released publicly.

### 2. Create a Config

Create the config directory:

```bash
mkdir -p ~/.config/nerd-fonts-installer
```

Create the config file:

```bash
nano ~/.config/nerd-fonts-installer/config.yaml
```

Paste this:

```yaml
release: latest
destination: ~/.local/share/fonts/NerdFonts
refresh_font_cache: true
families:
  - JetBrainsMono
  - Hack
  - FiraCode
```

Save the file.

### 3. Preview First

Before installing, run a dry-run:

```bash
nerd-fonts-installer --dry-run
```

This prints what would be downloaded and where it would go. It does not write
font files.

### 4. Install

Run:

```bash
nerd-fonts-installer
```

Now select the installed Nerd Font in your terminal or editor settings.

## The Easiest Mode: Interactive Picker

If you do not want to write YAML yet, just run:

```bash
nerd-fonts-installer --interactive
```

When no config file is found and `--interactive` is set, the app opens an
interactive picker:

1. Pick a Nerd Fonts release.
2. Pick one or more font families.
3. Press `enter`.
4. The selected fonts install.

Controls:

| Key | Action |
| --- | --- |
| `up` / `down` | Move through releases or fonts |
| `/` | Search/filter |
| `enter` | Choose a release or confirm selected fonts |
| `space` | Select or unselect a font |
| `a` | Select all or clear all |
| `b` / `esc` | Go back |
| `q` / `ctrl+c` | Quit |

## Copy-Paste Font Names Into Your Config

Not sure what the exact family names are? Ask the tool:

```bash
nerd-fonts-installer --font-names
```

It prints YAML you can paste directly into your config:

```yaml
# v3.4.0
families:
  - 0xProto
  - 3270
  - AdwaitaMono
  - Agave
  - AnonymousPro
```

For a pinned release, put the release in your config and run:

```bash
nerd-fonts-installer --config ~/.config/nerd-fonts-installer/config.yaml --font-names
```

## Common Recipes

### Install One Font

```yaml
release: latest
destination: ~/.local/share/fonts/NerdFonts
refresh_font_cache: true
families:
  - JetBrainsMono
```

Run:

```bash
nerd-fonts-installer --config fonts.yaml --dry-run
nerd-fonts-installer --config fonts.yaml
```

### Install a Good Terminal Font Set

```yaml
release: latest
destination: ~/.local/share/fonts/NerdFonts
refresh_font_cache: true
families:
  - JetBrainsMono
  - Hack
  - FiraCode
  - Meslo
  - SymbolsOnly
```

### Pin a Release for Dotfiles

Use this when you want the same result every time your setup script runs.

```yaml
release: v3.4.0
destination: ~/.local/share/fonts/NerdFonts
refresh_font_cache: true
families:
  - JetBrainsMono
  - Hack
```

### Install Into a Test Folder

Use this if you want to inspect the files before touching your real font
directory.

```yaml
release: latest
destination: ./tmp/fonts
refresh_font_cache: false
families:
  - Hack
```

Run:

```bash
nerd-fonts-installer --config fonts.yaml
```

Files will appear under:

```text
./tmp/fonts/Hack/
```

### Use It in a Bootstrap Script

```bash
#!/usr/bin/env bash
set -euo pipefail

mkdir -p ~/.config/nerd-fonts-installer

cat > ~/.config/nerd-fonts-installer/config.yaml <<'YAML'
release: latest
destination: ~/.local/share/fonts/NerdFonts
refresh_font_cache: true
families:
  - JetBrainsMono
  - Hack
  - FiraCode
YAML

nerd-fonts-installer --dry-run
nerd-fonts-installer
```

This is useful in dotfiles, Ansible roles, install scripts, or fresh-machine
setup scripts.

## Configuration Reference

Config is YAML.

```yaml
release: latest
destination: ~/.local/share/fonts/NerdFonts
refresh_font_cache: true
families:
  - JetBrainsMono
  - Hack
```

| Field | Required | Default | Meaning |
| --- | --- | --- | --- |
| `release` | No | `latest` | Nerd Fonts release to use. Use `latest` or a tag like `v3.4.0`. |
| `destination` | No | `~/.local/share/fonts/NerdFonts` | Root folder where fonts are installed. |
| `refresh_font_cache` | No | `false` | Run `fc-cache -f <destination>` after installing. |
| `families` | Yes | none | Font archive names without `.zip`. |

Family names must be exact Nerd Font archive names. Use `--font-names` when in
doubt.

## Where Config Files Are Found

When `--config` is not provided, the tool checks the `NERD_FONTS_INSTALLER_CONFIG`
environment variable, then the following locations in order:

1. `./nerd-fonts-installer.yaml`
2. `./nerd-fonts-installer.yml`
3. `./nerd-fonts-installer.json`
4. `./nerd-fonts-installer.conf`
5. `./nerd-fonts-installer/config.yaml`
6. `./nerd-fonts-installer/config.yml`
7. `./nerd-fonts-installer/config.json`
8. `./nerd-fonts-installer/config.conf`
9. The same app-named files under `$XDG_CONFIG_HOME`, when it is set to an
   absolute path.
10. The same app-named files under `~/.config` when `$XDG_CONFIG_HOME` is
    unset, empty, or relative.

Set `NERD_FONTS_INSTALLER_CONFIG=/path/to/fonts.yaml` to point at a config
without passing `--config` every time — handy in dotfiles, CI, and containers.
It is honored by `--font-names` as well.

The recommended location is:

```text
~/.config/nerd-fonts-installer/config.yaml
```

## Command Reference

```text
nerd-fonts-installer [flags]
```

| Flag | What it does |
| --- | --- |
| `--config <path>` | Use a specific config file. |
| `--dry-run` | Show what would happen without installing. |
| `--font-names` | Print YAML-ready font family names and exit. |
| `--interactive` | Open the terminal picker when no config is found. |
| `--icons <mode>` | Set interactive TUI icons: `auto`, `nerd`, `unicode`, or `ascii`. Defaults to `auto`, which avoids requiring Nerd Font glyphs. |
| `--version` | Print version info and exit. |

Examples:

```bash
nerd-fonts-installer --dry-run
nerd-fonts-installer --config fonts.yaml
nerd-fonts-installer --config fonts.yaml --dry-run
nerd-fonts-installer --font-names
nerd-fonts-installer --interactive --icons nerd
nerd-fonts-installer --interactive --icons ascii
nerd-fonts-installer --version
```

## Install Layout

Given:

```yaml
destination: ~/.local/share/fonts/NerdFonts
families:
  - JetBrainsMono
  - Hack
```

The tool writes:

```text
~/.local/share/fonts/NerdFonts/
  JetBrainsMono/
    JetBrainsMonoNerdFont-Regular.ttf
    ...
  Hack/
    HackNerdFont-Regular.ttf
    ...
```

Each family gets its own directory. Existing files for that family are replaced
after the new archive extracts successfully.

## Troubleshooting

### `no config found`

You have two choices:

1. Create a config:

   ```bash
   mkdir -p ~/.config/nerd-fonts-installer
   nano ~/.config/nerd-fonts-installer/config.yaml
   ```

2. Or pass a config explicitly:

   ```bash
   nerd-fonts-installer --config /path/to/fonts.yaml
   ```

If you want to use the terminal picker instead, run:

```bash
nerd-fonts-installer --interactive
```

### `duplicate font family "JetBrainsMono"`

The same font appears twice in `families`.

Remove the duplicate:

```yaml
families:
  - JetBrainsMono
```

### `download ... 404 Not Found`

Usually this means the family name or release tag is wrong.

Run:

```bash
nerd-fonts-installer --font-names
```

Then copy the exact family name from the output.

### Font Installed but Not Visible

Try refreshing the font cache:

```bash
fc-cache -f ~/.local/share/fonts/NerdFonts
```

Also restart the application where you select fonts. Terminals and editors often
need a restart before new fonts appear.

### Icons Still Look Broken

Install a Nerd Font and then choose that exact Nerd Font in your terminal or
editor preferences. Installing the font is only half of the job; your app still
needs to use it.

For example, choose something like:

```text
JetBrainsMono Nerd Font
Hack Nerd Font
FiraCode Nerd Font
```

## Notes for Linux and macOS

Linux:

- Recommended destination: `~/.local/share/fonts/NerdFonts`
- Set `refresh_font_cache: true`
- The tool will run `fc-cache` when available

macOS:

- You can use a custom destination if you manage fonts manually
- `fc-cache` is usually not installed and will be skipped
- You may prefer installing into a local folder first, then importing fonts with
  Font Book or another font manager

## Build From Source

Requirements:

- Go 1.26 or newer

Build:

```bash
go build -trimpath -o bin/nerd-fonts-installer ./cmd/nerd-fonts-installer
```

Smoke test:

```bash
./bin/nerd-fonts-installer --version
./bin/nerd-fonts-installer --config config.example.yaml --dry-run
```

## Development

Run the full local validation suite:

```bash
make verify
```

Run tests:

```bash
make test
```

Run vet:

```bash
make vet
```

Run lint:

```bash
make lint
```

Format Go code:

```bash
make fmt
```

## Release and Snap Publishing

The release workflow publishes versioned GitHub releases for `v*` tags and also
refreshes a moving `latest` release. Use the fixed download form:

```text
https://github.com/worxbend/nerd-fonts-installer/releases/download/latest/<asset>
```

The Snap workflow builds snaps on pull requests, `main`, tags, and manual runs.
It publishes only on non-PR runs:

- `main` publishes to `edge`
- `v*` tags publish to `stable`
- manual runs publish to the selected channel

Before publishing snaps, register the `nerd-fonts-installer` snap name and add a
repository secret named `SNAPCRAFT_STORE_CREDENTIALS`. Generate it with:

```bash
snapcraft export-login --snaps=nerd-fonts-installer \
  --acls package_access,package_push,package_update,package_release \
  /tmp/snapcraft-login.txt
```

Paste the contents of the generated file into the `SNAPCRAFT_STORE_CREDENTIALS`
secret, then delete the local file. Never write it inside the repository —
it is a live credential and must not be committed.

## License

MIT. See [LICENSE](LICENSE).

## Design

The project is intentionally small:

- `cmd/nerd-fonts-installer` owns flags, command flow, and exit codes.
- `internal/config` owns config loading, defaults, validation, and discovery.
- `internal/fonts` owns downloads, extraction, atomic replacement, and cache refresh.
- `internal/nerdfonts` owns GitHub release discovery.
- `internal/tui` owns the Charm Bubble Tea interactive UI.

The goal is not to be a full font manager. The goal is to make Nerd Font
installation boring, repeatable, and scriptable.
