#!/usr/bin/env python3
"""Run a command in a pty, drive it with keystrokes, dump the final screen as JSON."""
import codecs
import json
import os
import pty
import select
import signal
import sys
import time

import pyte

TIMEOUT = 60


def capture(cmd, env, cols, rows, script, screen_rows=None):
    """script: list of (delay_seconds, bytes_to_send).

    screen_rows lets the emulated screen be taller than the pty so a full-height
    TUI frame ending in a newline does not scroll its own top border away.
    """
    screen_rows = screen_rows or rows
    screen = pyte.Screen(cols, screen_rows)
    stream = pyte.Stream(screen)
    # incremental so a multi-byte glyph split across two reads is not mangled
    decoder = codecs.getincrementaldecoder("utf-8")("replace")

    pid, fd = pty.fork()
    if pid == 0:
        os.environ.clear()
        os.environ.update(env)
        os.execvp(cmd[0], cmd)
        os._exit(127)

    import fcntl
    import struct
    import termios

    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))

    deadline = time.time() + TIMEOUT
    steps = list(script)
    next_send = time.time() + (steps[0][0] if steps else 0)

    try:
        while time.time() < deadline:
            r, _, _ = select.select([fd], [], [], 0.05)
            if r:
                try:
                    data = os.read(fd, 65536)
                except OSError:
                    break
                if not data:
                    break
                stream.feed(decoder.decode(data))
            now = time.time()
            if steps and now >= next_send:
                _, payload = steps.pop(0)
                if payload is None:
                    break
                os.write(fd, payload)
                next_send = now + (steps[0][0] if steps else 0)
            if not steps and not r:
                break
    finally:
        try:
            os.kill(pid, signal.SIGKILL)
        except OSError:
            pass
        try:
            os.close(fd)
        except OSError:
            pass
        try:
            os.waitpid(pid, 0)
        except OSError:
            pass

    out = []
    for y in range(screen_rows):
        line = screen.buffer[y]
        cells = []
        for x in range(cols):
            ch = line[x]
            cells.append(
                {
                    "c": ch.data,
                    "fg": ch.fg,
                    "bg": ch.bg,
                    "b": ch.bold,
                    "i": ch.italics,
                    "u": ch.underscore,
                    "r": ch.reverse,
                }
            )
        out.append(cells)
    return {"cols": cols, "rows": screen_rows, "cells": out}


if __name__ == "__main__":
    spec = json.load(open(sys.argv[1]))
    TIMEOUT = spec.get("timeout", 60)
    env = dict(os.environ)
    env.update(spec.get("env", {}))
    for k in spec.get("unset", []):
        env.pop(k, None)
    script = [(s.get("wait", 0.4), s["send"].encode() if s.get("send") is not None else None)
              for s in spec.get("script", [])]
    result = capture(spec["cmd"], env, spec.get("cols", 108), spec.get("rows", 34), script,
                     spec.get("screen_rows"))
    json.dump(result, open(sys.argv[2], "w"))
    print(f"captured -> {sys.argv[2]}")
