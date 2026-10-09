"""Review tracked/index content without displaying potential private values."""
import argparse
import os
from pathlib import Path
import re
import socket
import subprocess


PATTERNS = {
    "personal home path": re.compile(rb"/(?:home|Users)/(?!user(?:/|\b)|example(?:/|\b))[^/\s\"']+/"),
    "mounted machine path": re.compile(rb"/run/" rb"media/[^\s\"']+"),
    "private key": re.compile(rb"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
    "credential": re.compile(rb"(?:gh[pousr]_)[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{30,}|AKIA[A-Z0-9]{16}"),
    "agent execution path": re.compile(rb"/tmp/devdock-pass[0-9][A-Za-z0-9_-]*"),
}


def findings(data):
    return [(line, kind) for line, value in enumerate(data.splitlines(), 1)
            for kind, pattern in PATTERNS.items() if pattern.search(value)]


def scan(staged=False):
    args = ["git", "diff", "--cached", "--name-only", "--diff-filter=ACMR", "-z"] if staged else ["git", "ls-files", "-z"]
    names = subprocess.check_output(args).split(b"\0")
    count = 0
    for raw in filter(None, names):
        name = os.fsdecode(raw)
        if name.startswith(".devdock-work/"):
            print(f"{name!r}: local evidence must not be tracked")
            count += 1
            continue
        if not staged and not Path(name).exists():
            continue
        data = subprocess.check_output(["git", "show", ":" + name]) if staged else Path(name).read_bytes()
        for line, kind in findings(data):
            print(f"{name!r}:{line}: {kind} (value withheld)")
            count += 1
        # Match current workstation identity as a complete component, not a
        # generic English substring. Never inspect or print environment dumps.
        for value in (socket.gethostname(),):
            if len(value) >= 3 and re.search(rb"(?<![\w-])" + re.escape(value.encode()) + rb"(?![\w-])", data):
                print(f"{name!r}: workstation identity (value withheld)")
                count += 1
    return count


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--staged", action="store_true")
    group.add_argument("--tracked", action="store_true")
    raise SystemExit(bool(scan(parser.parse_args().staged)))
