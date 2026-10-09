"""Enforce the patched compiler policy independently of Go language directives."""
import re
import subprocess


def supported(version):
    match = re.fullmatch(r"go1\.(\d+)\.(\d+)(?:-.*)?", version)
    if not match:
        return False
    minor, patch = map(int, match.groups())
    return minor == 26 and patch >= 9 or minor == 27 and patch >= 2


if __name__ == "__main__":
    version = subprocess.check_output(["go", "env", "GOVERSION"], text=True).strip()
    if not supported(version):
        raise SystemExit("Use a supported patched Go compiler: 1.26.9+ or 1.27.2+ (within those series).")
