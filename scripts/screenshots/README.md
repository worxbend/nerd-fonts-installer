# Screenshot harness

The terminal screenshots in [`assets/screenshots/`](../../assets/screenshots) are
generated from **real runs** of `nerd-fonts-installer`, not mocked up by hand.

```bash
pip install pyte          # one-off
./scripts/screenshots/refresh.sh
```

## How it works

| Step | File | What it does |
| --- | --- | --- |
| 1 | `specs.py` | Describes each shot: command, terminal size, and the keystrokes to send. |
| 2 | `capture.py` | Runs the command in a real pty, drives it, and dumps the final screen (with colors) as JSON via `pyte`. |
| 3 | `render.py` | Turns that screen into a self-contained SVG with terminal-window chrome. |

`refresh.sh` wires the three together and drops the results straight into
`assets/screenshots/`.

## Why bubblewrap

The install shot is a genuine install — it downloads release archives from GitHub
and writes font files. `bwrap` shadows `$HOME` with a throwaway directory under
`.work/`, so:

- nothing is written to the real font directory, and
- the paths visible on screen still read like an ordinary machine
  (`/home/dev/.local/share/fonts/NerdFonts`) instead of a temp path.

## Notes

- The interactive shots use `--icons unicode` so the SVG renders correctly for
  readers who do not have a Nerd Font installed.
- SVG output means the screenshots stay sharp at any zoom, respond to the
  reader's display, and diff as text.
- Captures hit the public GitHub API unauthenticated (60 requests/hour). If a run
  produces an empty screen, check your rate limit.
