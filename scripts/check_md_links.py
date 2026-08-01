#!/usr/bin/env python3
"""Check that relative links in tracked .md files resolve to existing files.

Usage:
  python scripts/check_md_links.py          # check all tracked .md files
  python scripts/check_md_links.py --file README.md

Exit 0 = OK, 1 = broken links found.
"""

import argparse
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent

LINK_RE = re.compile(r"\[[^\]]*\]\(([^)]+)\)")


def tracked_md_files():
    r = subprocess.run(["git", "ls-files", "*.md"], capture_output=True, text=True, cwd=str(ROOT))
    out = r.stdout
    return [f for f in out.splitlines() if f]


def broken_links_in(file_path):
    text = (ROOT / file_path).read_text(encoding="utf-8", errors="replace")
    base = (ROOT / file_path).parent
    broken = []
    for m in LINK_RE.finditer(text):
        target = m.group(1).strip()
        if not target or target.startswith(("http://", "https://", "mailto:", "#")):
            continue
        path_part = target.split("#")[0]
        if not path_part:
            continue
        resolved = (base / path_part).resolve()
        if not resolved.exists():
            broken.append(target)
    return broken


def main():
    parser = argparse.ArgumentParser(description="Validate relative links in Markdown files")
    parser.add_argument("--file", help="Single file to check")
    args = parser.parse_args()

    files = [args.file] if args.file else tracked_md_files()
    all_broken = []
    for f in files:
        for target in broken_links_in(f):
            all_broken.append((f, target))

    if all_broken:
        print(f"[MD-LINKS] {len(all_broken)} broken link(s):")
        for f, t in all_broken:
            print(f"  {f} -> {t}")
        return 1
    print(f"OK: all relative links resolve ({len(files)} file(s)).")
    return 0


if __name__ == "__main__":
    sys.exit(main())
