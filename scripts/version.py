#!/usr/bin/env python3
"""Validate explicit SemVer release versions and compare promotion candidates.

Release tags omit build metadata. Version changes are chosen by maintainers;
this script never invents or increments a version.
"""
import re
import sys

PATTERN = re.compile(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?\Z")


def parse(value):
    match = PATTERN.fullmatch(value)
    if not match:
        raise ValueError(f"invalid release version: {value!r}")
    parts = match[4].split(".") if match[4] else []
    if any(part.isdigit() and len(part) > 1 and part.startswith("0") for part in parts):
        raise ValueError("numeric prerelease identifiers cannot have leading zeros")
    return tuple(int(match[i]) for i in (1, 2, 3)), parts


def compare(left, right):
    a, ap = parse(left)
    b, bp = parse(right)
    if a != b:
        return (a > b) - (a < b)
    if not ap or not bp:
        return (not ap) - (not bp)
    for x, y in zip(ap, bp):
        if x == y:
            continue
        if x.isdigit() and y.isdigit():
            return (int(x) > int(y)) - (int(x) < int(y))
        if x.isdigit() != y.isdigit():
            return -1 if x.isdigit() else 1
        return (x > y) - (x < y)
    return (len(ap) > len(bp)) - (len(ap) < len(bp))


if __name__ == "__main__":
    try:
        if len(sys.argv) == 2:
            parse(sys.argv[1])
            print(sys.argv[1])
        elif len(sys.argv) == 4 and sys.argv[1] == "--after":
            if compare(sys.argv[3], sys.argv[2]) <= 0:
                raise ValueError("production candidate VERSION must be greater than production VERSION")
        else:
            raise ValueError("usage: version.py VERSION | version.py --after OLD NEW")
    except ValueError as error:
        sys.exit(str(error))
