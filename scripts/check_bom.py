#!/usr/bin/env python3
"""
Check for UTF-8 BOM (Byte Order Mark) in Go source files.

Usage:
  python scripts/check_bom.py                    # list all .go files with BOM
  python scripts/check_bom.py --fix              # remove BOM from all .go files
  python scripts/check_bom.py --path xxx.go      # check/fix single file
  python scripts/check_bom.py --path internal/   # check/fix a directory
"""

import sys
import os
import pathlib

BOM = b"\xef\xbb\xbf"


def has_bom(path):
    with open(path, "rb") as f:
        return f.read(3) == BOM


def strip_bom(path):
    with open(path, "rb") as f:
        data = f.read()
    if data.startswith(BOM):
        with open(path, "wb") as f:
            f.write(data[len(BOM):])
        return True
    return False


def find_bom_files(root):
    found = []
    root_path = pathlib.Path(root)
    pattern = "**/*.go"
    for f in root_path.glob(pattern):
        if has_bom(f):
            found.append(f)
    return found


def main():
    args = sys.argv[1:]

    fix = "--fix" in args
    path_arg = None
    for a in args:
        if not a.startswith("-"):
            path_arg = a
            break

    if path_arg:
        p = pathlib.Path(path_arg)
        if p.is_dir():
            files = find_bom_files(p)
        else:
            files = [p] if has_bom(p) else []
    else:
        # Default: scan internal/ + cmd/
        repo = pathlib.Path(__file__).resolve().parent.parent
        files = []
        for d in ["internal", "cmd"]:
            files.extend(find_bom_files(repo / d))

    if not files:
        print("OK: no BOM found")
        sys.exit(0)

    print(f"Found {len(files)} file(s) with BOM:")
    for f in files:
        print(f"  {f}")
        if fix:
            strip_bom(f)
            print(f"    -> fixed")

    if fix:
        print("All fixed.")
    else:
        print("Run with --fix to remove BOM.")
        sys.exit(1)


if __name__ == "__main__":
    main()
