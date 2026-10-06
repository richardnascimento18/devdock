"""Check that terminal output never chooses an explicit background/inverse fill."""

import re

SGR = re.compile(rb"\x1b\[([0-9;:]*)m")


def decorative_background(output):
    for sequence in SGR.finditer(output):
        values = sequence[1].split(b";")
        index = 0
        while index < len(values):
            compound = values[index].split(b":")
            value = int(compound[0] or b"0")
            if value == 7 or 40 <= value <= 47 or 100 <= value <= 107 or value == 48:
                return sequence[0]
            if value == 38 and len(compound) == 1 and index + 1 < len(values):
                mode = int(values[index + 1] or b"0")
                index += 2 if mode == 5 else 4 if mode == 2 else 0
            index += 1
    return None


# PTY streams include cursor/erase commands as well as SGR. Strip complete
# control sequences before looking for leaked renderer parameters. Test workspaces
# use known plain text; optional literal text is explicitly exempted.
CSI = re.compile(rb"\x1b\[[0-?]*[ -/]*[@-~]")
OSC = re.compile(rb"\x1b\][^\x1b\x07]*(?:\x07|\x1b\\)")
LEAKED_SGR = re.compile(rb"\[38;(?:2|5);")


def malformed_ansi(output, literal_user_text=()):
    plain = OSC.sub(b"", output)
    plain = CSI.sub(b"", plain)
    # ANSI cursor save/restore, keypad and character-set selection.
    plain = re.sub(rb"\x1b(?:[=>78]|[()][0-2A-Z])", b"", plain)
    if b"\x1b" in plain:
        return b"incomplete escape"
    for literal in literal_user_text:
        plain = plain.replace(literal, b"")
    match = LEAKED_SGR.search(plain)
    return match[0] if match else None
