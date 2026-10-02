"""Prepare a dedicated release directory without deleting unrelated files."""

import argparse
import pathlib
import re


def clean(directory):
    directory = pathlib.Path(directory)
    directory.mkdir(parents=True, exist_ok=True)
    entries = list(directory.iterdir())
    generated = re.compile(r"devdock_.+_linux_[a-z0-9]+\Z")
    for entry in entries:
        if entry.name != "SHA256SUMS" and not generated.fullmatch(entry.name):
            raise ValueError(f"unexpected release output {entry}; use a dedicated output directory")
        if entry.is_dir():
            raise ValueError(f"release output is a directory: {entry}")
    for entry in entries:
        entry.unlink()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=pathlib.Path)
    args = parser.parse_args()
    try:
        clean(args.directory)
    except (OSError, ValueError) as error:
        parser.exit(1, f"release output preparation failed: {error}\n")
