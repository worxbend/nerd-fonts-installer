#!/usr/bin/env python3
"""Render a captured pyte screen JSON into a standalone terminal-window SVG."""
import json
import sys
from xml.sax.saxutils import escape

CW = 8.4          # cell width
LH = 18.0         # line height
FS = 14.0         # font size
BASE = 13.6       # baseline offset within the line box
PAD_X = 18.0
PAD_TOP = 44.0    # room for the title bar
PAD_BOT = 16.0
RADIUS = 12.0

BG = "#12121C"
CHROME = "#1C1B2A"
BORDER = "#2C2A44"
FG_DEFAULT = "#EDEDF7"

ANSI = {
    "black": "#22212F", "red": "#FF5C7A", "green": "#54E08A", "brown": "#FFC857",
    "yellow": "#FFC857", "blue": "#5BA8FF", "magenta": "#C75CFF", "cyan": "#46E5E0",
    "white": "#EDEDF7", "brightblack": "#595972", "brightred": "#FF8098",
    "brightgreen": "#7BE8A6", "brightbrown": "#FFD98A", "brightyellow": "#FFD98A",
    "brightblue": "#8AC2FF", "brightmagenta": "#D98BFF", "brightcyan": "#7BEDE9",
    "brightwhite": "#FFFFFF",
}


# Box-drawing and block characters are drawn as vectors rather than text. A
# monospace font's box glyphs only fill their own em box, so at a terminal line
# height they leave visible gaps between rows — borders come out dashed. Drawing
# them geometrically makes every rule continuous and pixel-crisp at any zoom.
STROKE = 1.3      # border line thickness
CORNER = 3.0      # rounded-corner radius on ╭ ╮ ╰ ╯

BOX_LINES = {
    #      up     down   left   right
    "│": (True, True, False, False),
    "─": (False, False, True, True),
    "╭": (False, True, False, True),
    "╮": (False, True, True, False),
    "╰": (True, False, False, True),
    "╯": (True, False, True, False),
    "├": (True, True, False, True),
    "┤": (True, True, True, False),
    "┬": (False, True, True, True),
    "┴": (True, False, True, True),
    "┼": (True, True, True, True),
}

# Block elements: (x offset, width, opacity) as fractions of the cell.
BOX_BLOCKS = {
    "█": (0.0, 1.0, 1.0),
    "▌": (0.0, 0.5, 1.0),
    "▐": (0.5, 0.5, 1.0),
    "░": (0.0, 1.0, 0.30),
    "▒": (0.0, 1.0, 0.50),
    "▓": (0.0, 1.0, 0.75),
}

BOX_GLYPHS = set(BOX_LINES) | set(BOX_BLOCKS)


def box_glyph(ch, x, y, fill, span=1):
    """Return SVG for a run of `span` identical box-drawing/block cells.

    (x, y) is the top-left corner of the first cell. Only full-width blocks are
    ever passed a span greater than one.
    """
    if ch in BOX_BLOCKS:
        dx, w, opacity = BOX_BLOCKS[ch]
        alpha = "" if opacity == 1.0 else f' fill-opacity="{opacity}"'
        return (
            f'<rect x="{x + dx * CW:.2f}" y="{y:.2f}" width="{(w + span - 1) * CW:.2f}" '
            f'height="{LH:.2f}" fill="{fill}"{alpha} shape-rendering="crispEdges"/>'
        )

    up, down, left, right = BOX_LINES[ch]
    cx, cy = x + CW / 2, y + LH / 2
    # A corner (exactly one vertical and one horizontal arm) gets a rounded joint;
    # anything else is drawn as independent straight arms.
    corner = (up != down) and (left != right)
    if corner:
        vy = y if up else y + LH
        hx = x if left else x + CW
        r = CORNER
        ry = cy + r if up else cy - r
        rx = cx - r if left else cx + r
        d = (f"M{cx:.2f} {vy:.2f} L{cx:.2f} {ry:.2f} "
             f"Q{cx:.2f} {cy:.2f} {rx:.2f} {cy:.2f} L{hx:.2f} {cy:.2f}")
        return (f'<path d="{d}" fill="none" stroke="{fill}" stroke-width="{STROKE}" '
                f'stroke-linecap="butt"/>')

    arms = []
    if up or down:
        y0 = y if up else cy
        y1 = y + LH if down else cy
        arms.append(f'<rect x="{cx - STROKE / 2:.2f}" y="{y0:.2f}" width="{STROKE}" '
                    f'height="{y1 - y0:.2f}" fill="{fill}"/>')
    if left or right:
        x0 = x if left else cx
        x1 = x + CW if right else cx
        arms.append(f'<rect x="{x0:.2f}" y="{cy - STROKE / 2:.2f}" width="{x1 - x0:.2f}" '
                    f'height="{STROKE}" fill="{fill}"/>')
    return "".join(arms)


def color(name, default):
    if not name or name == "default":
        return default
    if name in ANSI:
        return ANSI[name]
    if len(name) == 6:
        try:
            int(name, 16)
            return "#" + name
        except ValueError:
            pass
    return default


def render(data, title, out_path):
    cols, rows = data["cols"], data["rows"]
    cells = data["cells"]

    # trim trailing blank rows
    def blank_row(row):
        return all(c["c"] in (" ", "") and c["bg"] == "default" for c in row)

    while rows > 1 and blank_row(cells[rows - 1]):
        rows -= 1
    cells = cells[:rows]

    # trim trailing blank columns so wide captures shrink to their content
    def blank_col(x):
        return all(row[x]["c"] in (" ", "") and row[x]["bg"] == "default" for row in cells)

    while cols > 1 and blank_col(cols - 1):
        cols -= 1
    cells = [row[:cols] for row in cells]

    width = cols * CW + PAD_X * 2
    height = rows * LH + PAD_TOP + PAD_BOT

    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width:.0f}" height="{height:.0f}" '
        f'viewBox="0 0 {width:.2f} {height:.2f}" font-family="ui-monospace, SFMono-Regular, '
        f'Menlo, Consolas, &quot;DejaVu Sans Mono&quot;, monospace" font-size="{FS}" '
        f'role="img" aria-label="{escape(title)}">',
        "<defs>"
        '<linearGradient id="chrome" x1="0" y1="0" x2="0" y2="1">'
        f'<stop offset="0" stop-color="#221F35"/><stop offset="1" stop-color="{CHROME}"/>'
        "</linearGradient>"
        f'<clipPath id="win"><rect x="0.5" y="0.5" width="{width-1:.2f}" height="{height-1:.2f}" rx="{RADIUS}"/></clipPath>'
        "</defs>",
        f'<rect x="0.5" y="0.5" width="{width-1:.2f}" height="{height-1:.2f}" rx="{RADIUS}" fill="{BG}" stroke="{BORDER}"/>',
        f'<g clip-path="url(#win)"><rect x="0" y="0" width="{width:.2f}" height="30" fill="url(#chrome)"/>'
        f'<rect x="0" y="29.5" width="{width:.2f}" height="1" fill="{BORDER}"/></g>',
        '<circle cx="20" cy="15" r="5.5" fill="#FF5F57"/>'
        '<circle cx="39" cy="15" r="5.5" fill="#FEBC2E"/>'
        '<circle cx="58" cy="15" r="5.5" fill="#28C840"/>',
        f'<text x="{width/2:.2f}" y="19.5" font-size="12" fill="#8A87A8" text-anchor="middle">{escape(title)}</text>',
    ]

    # background rects first (merged horizontal runs)
    bg_parts = []
    for y, row in enumerate(cells):
        x = 0
        while x < cols:
            bg = row[x]["bg"]
            rev = row[x]["r"]
            key = (bg, rev)
            if bg == "default" and not rev:
                x += 1
                continue
            run = x
            while run < cols and (cells[y][run]["bg"], cells[y][run]["r"]) == key:
                run += 1
            fill = color(row[x]["fg"], FG_DEFAULT) if rev else color(bg, BG)
            bg_parts.append(
                f'<rect x="{PAD_X + x*CW:.2f}" y="{PAD_TOP + y*LH - 13.5:.2f}" '
                f'width="{(run-x)*CW:.2f}" height="{LH:.2f}" fill="{fill}"/>'
            )
            x = run
    parts.extend(bg_parts)

    # box-drawing and block glyphs, drawn as vectors. Identical neighbours merge
    # into one shape: abutting translucent rects would otherwise double-paint
    # their shared edge and leave visible seams down a progress-bar track.
    for y, row in enumerate(cells):
        x = 0
        while x < cols:
            ch = row[x]
            if ch["c"] not in BOX_GLYPHS:
                x += 1
                continue
            fill = color(ch["bg"], BG) if ch["r"] else color(ch["fg"], FG_DEFAULT)
            run = x + 1
            if ch["c"] in BOX_BLOCKS and BOX_BLOCKS[ch["c"]][:2] == (0.0, 1.0):
                while run < cols:
                    nxt = row[run]
                    if nxt["c"] != ch["c"] or nxt["r"] != ch["r"]:
                        break
                    if (color(nxt["bg"], BG) if nxt["r"] else color(nxt["fg"], FG_DEFAULT)) != fill:
                        break
                    run += 1
            parts.append(box_glyph(
                ch["c"], PAD_X + x * CW, PAD_TOP + y * LH - 13.5, fill, span=run - x))
            x = run

    # foreground text runs
    for y, row in enumerate(cells):
        x = 0
        yy = PAD_TOP + y * LH + BASE - 13.5
        while x < cols:
            ch = row[x]
            if ch["c"] in (" ", "") or ch["c"] in BOX_GLYPHS:
                x += 1
                continue
            style = (ch["fg"], ch["b"], ch["i"], ch["u"], ch["r"], ch["bg"])
            run = x
            text = []
            while run < cols:
                c2 = cells[y][run]
                if (c2["fg"], c2["b"], c2["i"], c2["u"], c2["r"], c2["bg"]) != style:
                    break
                if c2["c"] in BOX_GLYPHS:  # already drawn as a vector
                    break
                text.append(c2["c"] or " ")
                run += 1
            s = "".join(text).rstrip()
            if s:
                fill = color(ch["bg"], BG) if ch["r"] else color(ch["fg"], FG_DEFAULT)
                attrs = [
                    f'x="{PAD_X + x*CW:.2f}"',
                    f'y="{yy:.2f}"',
                    f'fill="{fill}"',
                    f'textLength="{len(s)*CW:.2f}"',
                    'lengthAdjust="spacingAndGlyphs"',
                    'xml:space="preserve"',
                ]
                if ch["b"]:
                    attrs.append('font-weight="700"')
                if ch["i"]:
                    attrs.append('font-style="italic"')
                if ch["u"]:
                    attrs.append('text-decoration="underline"')
                parts.append(f"<text {' '.join(attrs)}>{escape(s)}</text>")
            x = run

    parts.append("</svg>")
    with open(out_path, "w") as fh:
        fh.write("\n".join(parts))
    print(f"rendered -> {out_path} ({width:.0f}x{height:.0f})")


if __name__ == "__main__":
    render(json.load(open(sys.argv[1])), sys.argv[3], sys.argv[2])
