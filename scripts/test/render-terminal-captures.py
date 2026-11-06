#!/usr/bin/env python3
"""Render already-captured tmux ANSI cells as SVG; does not invent UI content."""

import html
import pathlib
import re
import sys

source, destination = map(pathlib.Path, sys.argv[1:3])
destination.mkdir(parents=True, exist_ok=True)
escape = re.compile(r"\x1b\[([0-9;]*)m")
for capture in sorted(source.glob("*.ansi")):
    lines = capture.read_text().splitlines()
    columns, rows = 110, 32
    size_file = capture.with_suffix(".size")
    if size_file.exists():
        match = re.fullmatch(r"(\d+)x(\d+)", size_file.read_text().strip())
        if not match:
            raise ValueError(f"Invalid captured pane size: {size_file}")
        columns, rows = map(int, match.groups())
        if not 20 <= columns <= 300 or not 10 <= rows <= 150:
            raise ValueError(f"Unsupported captured pane size: {size_file}")
    canvas_width, canvas_height = columns * 9 + 24, rows * 19 + 20
    fg, bg, bold = "#c8c8c0", "#1a1f1a", False
    backgrounds = ['<rect width="100%" height="100%" fill="#1a1f1a"/>']
    foregrounds = []
    for row, line in enumerate(lines):
        column = 0
        position = 0
        for match in list(escape.finditer(line)) + [None]:
            end = match.start() if match else len(line)
            text = line[position:end]
            if text:
                x, y, width = 12 + column * 9, 10 + row * 19, len(text) * 9
                backgrounds.append(f'<rect x="{x}" y="{y}" width="{width}" height="19" fill="{bg}"/>')
                weight = "bold" if bold else "normal"
                foregrounds.append(f'<text x="{x}" y="{y+14}" fill="{fg}" font-weight="{weight}">{html.escape(text)}</text>')
                column += len(text)
            if match:
                values = [int(value or 0) for value in match.group(1).split(";")]
                index = 0
                while index < len(values):
                    code = values[index]
                    if code == 0:
                        fg, bg, bold = "#c8c8c0", "#1a1f1a", False
                    elif code == 1:
                        bold = True
                    elif code == 22:
                        bold = False
                    elif code == 39:
                        fg = "#c8c8c0"
                    elif code == 49:
                        bg = "#1a1f1a"
                    elif code in (38, 48) and values[index+1:index+2] == [2] and len(values) >= index+5:
                        color = "#" + "".join(f"{value:02x}" for value in values[index+2:index+5])
                        if code == 38:
                            fg = color
                        else:
                            bg = color
                        index += 4
                    index += 1
                position = match.end()
    # Draw all cell backgrounds first: a later ANSI style segment must not paint
    # over the final glyph of the preceding segment (bold glyphs can overhang).
    svg = f'<svg xmlns="http://www.w3.org/2000/svg" width="{canvas_width}" height="{canvas_height}" viewBox="0 0 {canvas_width} {canvas_height}"><g font-family="DejaVu Sans Mono,monospace" font-size="15" xml:space="preserve">' + "".join(backgrounds + foregrounds) + "</g></svg>"
    (destination / (capture.stem + ".svg")).write_text(svg)
