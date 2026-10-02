"""Check that terminal output never chooses an explicit background/inverse fill."""

import re

SGR = re.compile(rb"\x1b\[([0-9;:]*)m")


def decorative_background(output):
    for sequence in SGR.finditer(output):
        values = [int(v) for v in re.split(rb"[;:]", sequence[1]) if v]
        index = 0
        while index < len(values):
            value = values[index]
            if value == 7 or 40 <= value <= 47 or 100 <= value <= 107 or value == 48:
                return sequence[0]
            if value == 38 and index + 1 < len(values):
                mode = values[index + 1]
                index += 2 if mode == 5 else 4 if mode == 2 else 0
            index += 1
    return None
