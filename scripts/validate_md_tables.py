#!/usr/bin/env python3
"""Validate Markdown tables - check for continuity breaks and column mismatches.

Usage:
  python scripts/validate_md_tables.py              # scan project .md files
  python scripts/validate_md_tables.py --file Rule.md # single file
  python scripts/validate_md_tables.py --ci          # exit 1 on errors

Checks:
  - Table rows split by --- separator (R-DOC-9-type bug)
  - Column count mismatch between rows and header
  - Rows not starting/ending with pipe
"""

import argparse, pathlib, re, sys

IGNORE_DIRS = {"node_modules", ".git", ".github", "dist", "build", "vendor"}

def scan_errors(text, filepath):
    errors = []
    lines = text.split("\n")
    
    # Find all table blocks
    tables = []
    i = 0
    while i < len(lines):
        # Look for header row: starts with | and has |
        if lines[i].strip().startswith("|") and "|" in lines[i][1:]:
            # Next line should be separator
            if i + 1 < len(lines) and re.match(r"^\|[\s:]*-+\|", lines[i + 1].strip()):
                header_line = i
                sep_line = i + 1
                header_cols = lines[i].count("|")
                row_start = i + 2
                row_end = row_start
                while row_end < len(lines) and lines[row_end].strip().startswith("|"):
                    row_end += 1
                
                # Check for continuity: no blank lines or --- between rows
                for j in range(row_start, row_end):
                    actual = j + 1
                    stripped = lines[j].strip()
                    if stripped == "---":
                        errors.append((filepath, actual, "CONTINUITY: --- separator inside table, breaks rendering"))
                    elif not stripped.startswith("|"):
                        errors.append((filepath, actual, f"CONTINUITY: blank/non-pipe line inside table"))
                    elif stripped[0] != "|" or stripped[-1] != "|":
                        errors.append((filepath, actual, "PIPE: row must start AND end with |"))
                    else:
                        cols = stripped.count("|")
                        if cols != header_cols:
                            errors.append((filepath, actual, f"COLUMNS: has {cols} pipes, header has {header_cols}"))
                
                i = row_end
                continue
        i += 1
    
    return errors


def main():
    parser = argparse.ArgumentParser(description="Validate MD table formatting")
    parser.add_argument("--file", help="Single file to check")
    parser.add_argument("--ci", action="store_true", help="Non-zero exit on errors")
    args = parser.parse_args()
    
    if args.file:
        files = [pathlib.Path(args.file)]
    else:
        files = sorted(pathlib.Path(".").rglob("*.md"))
        files = [f for f in files if not any(p in f.parts for p in IGNORE_DIRS) and not f.parent.name.startswith(".")]
    
    all_errors = []
    for f in files:
        all_errors += scan_errors(f.read_text("utf-8"), f)
    
    if not all_errors:
        print("OK: all tables look clean")
        return 0
    
    print(f"[TABLE CHECK] {len(all_errors)} issue(s) found:")
    print(f"{'File':<40} {'Line':>5}  {'Issue'}")
    print("-" * 100)
    for f, ln, desc in all_errors:
        print(f"{str(f):<40} {ln:>5}  {desc}")
    return 1 if args.ci else 0


if __name__ == "__main__":
    sys.exit(main())