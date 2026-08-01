#!/usr/bin/env python3
"""Write the capture specs consumed by capture.py.

Kept in Python rather than hand-written JSON so escape sequences stay readable.
Each spec drives one screenshot in assets/screenshots/.
"""
import json
import os

HERE = os.path.dirname(os.path.abspath(__file__))
WORK = os.environ.get("NFI_SHOT_WORK", os.path.join(HERE, ".work"))
HOST_BIN = os.environ.get("NFI_SHOT_BIN", os.path.join(WORK, "nerd-fonts-installer"))
# The home directory path bwrap invents. Screenshots show it verbatim, so it has
# to be short and ordinary-looking rather than a real temp path.
FAKE_HOME = os.environ.get("NFI_SHOT_HOME", "/home/dev")
# Where the built binary is mounted inside the sandbox. It lives next to the fake
# home because /home is a tmpfs there, and only a tmpfs lets bwrap create new
# mount points on the fly.
BIN = "/home/.nerd-fonts-installer"

CONFIGURED = os.path.join(WORK, "home-configured")  # has a config -> normal runs
BARE = os.path.join(WORK, "home-bare")  # has none -> --interactive opens the picker

DOWN = "\x1b[B"


def sandbox(home, extra=()):
    """bwrap args that replace the real /home with a throwaway one.

    This is what keeps the capture honest: the tool really downloads, extracts
    and installs fonts, but every byte lands under WORK instead of the real home,
    and the paths on screen still read like an ordinary machine.
    """
    return [
        "bwrap", "--dev-bind", "/", "/",
        "--tmpfs", "/home",
        "--bind", home, FAKE_HOME,
        "--ro-bind", HOST_BIN, BIN,
        "--setenv", "HOME", FAKE_HOME,
        "--chdir", FAKE_HOME,
        *extra,
    ]


def jail(home):
    """Run one command in the sandbox via `sh -c`."""
    return sandbox(home) + ["sh", "-c"]


def shell(home):
    """Run a real interactive bash in the sandbox.

    Used for the session shot: a genuine shell means genuine prompts and genuine
    output, rather than a prompt string pasted in to look like one. refresh.sh
    puts the binary on PATH inside the sandbox so the commands on screen read
    exactly as a reader would type them.
    """
    return sandbox(home, [
        "--setenv", "PS1", "$ ",
        "--setenv", "PATH", f"{FAKE_HOME}/.local/bin:/usr/local/bin:/usr/bin:/bin",
    ]) + ["bash", "--norc", "--noprofile", "-i"]


ENV = {"TERM": "xterm-256color", "COLORTERM": "truecolor"}
UNSET = ["NERD_FONTS_INSTALLER_CONFIG", "XDG_CONFIG_HOME"]

# 2>/dev/null on the interactive runs hides the pre-TUI "no config found" notice
# so the capture starts on a clean full-screen frame.
SPECS = {
    "tui-releases": {
        "cmd": jail(BARE) + [f"exec {BIN} --interactive --icons unicode 2>/dev/null"],
        "title": "nerd-fonts-installer --interactive",
        "cols": 112, "rows": 34, "screen_rows": 36, "timeout": 60,
        "script": [{"wait": 25.0, "send": ""}],
    },
    "tui-families": {
        "cmd": jail(BARE) + [f"exec {BIN} --interactive --icons unicode 2>/dev/null"],
        "title": "nerd-fonts-installer --interactive",
        "cols": 112, "rows": 34, "screen_rows": 36, "timeout": 60,
        "script": [
            {"wait": 25.0, "send": "\r"},   # choose the newest release
            {"wait": 1.0, "send": " "},     # then tick the first three families
            {"wait": 0.4, "send": DOWN},
            {"wait": 0.3, "send": " "},
            {"wait": 0.4, "send": DOWN},
            {"wait": 0.3, "send": " "},
            {"wait": 1.2, "send": ""},
        ],
    },
    # A typed session rather than one command's output: --font-names is only
    # interesting in a pipeline, and a lone column of family names renders as a
    # tall sliver that looks wrong next to the other shots.
    "cli-font-names": {
        "cmd": shell(CONFIGURED),
        "title": "nerd-fonts-installer --font-names",
        "cols": 100, "rows": 30, "timeout": 240,
        "script": [
            {"wait": 1.0, "send": "nerd-fonts-installer --font-names | head -8\r"},
            {"wait": 30.0,
             "send": "nerd-fonts-installer --font-names | grep -iE 'jetbrains|hack|fira|meslo'\r"},
            {"wait": 30.0,
             "send": "nerd-fonts-installer --font-names > fonts.yaml && wc -l fonts.yaml\r"},
            {"wait": 30.0, "send": ""},
        ],
    },
    "cli-dry-run": {
        "cmd": jail(CONFIGURED) + [f"exec {BIN} --dry-run"],
        "title": "nerd-fonts-installer --dry-run",
        "cols": 200, "rows": 14, "timeout": 60,
        "script": [{"wait": 12.0, "send": ""}],
    },
    "cli-install": {
        "cmd": jail(CONFIGURED) + [f"exec {BIN}"],
        "title": "nerd-fonts-installer",
        "cols": 200, "rows": 18, "timeout": 400,
        "script": [{"wait": 360.0, "send": ""}],
    },
}


if __name__ == "__main__":
    for name, spec in SPECS.items():
        spec = dict(spec, env=ENV, unset=UNSET)
        path = os.path.join(WORK, f"spec-{name}.json")
        with open(path, "w") as fh:
            json.dump(spec, fh, indent=2)
        print("wrote", path)
