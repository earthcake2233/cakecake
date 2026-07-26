"""
Check for UTF-8 BOM (Byte Order Mark) in project files.

Scans for:
  - File-level BOM (\xef\xbb\xbf at start) — in .go, .md, .yaml, .json, .py, .toml, .env*
  - Embedded BOM (\ufeff in middle of content) — in all text files

Usage:
  python scripts/check_bom.py                        # scan all files
  python scripts/check_bom.py --fix                   # remove BOM from all files
  python scripts/check_bom.py --path xxx.md           # check/fix single file
  python scripts/check_bom.py --path internal/        # check/fix a directory
"""

import sys
import os
import pathlib
import re

FILE_BOM = b"\xef\xbb\xbf"
TEXT_EXTENSIONS = {".go", ".md", ".yaml", ".yml", ".json", ".py", ".toml", ".env", ".env.example", ".gitignore"}
SCAN_DIRS = ["internal", "cmd", "deploy", "scripts", "docs"]


def has_file_bom(path):
    """Check for BOM at start of file."""
    with open(path, "rb") as f:
        return f.read(3) == FILE_BOM


def has_embedded_bom(path):
    """Check for embedded U+FEFF in text content."""
    try:
        content = pathlib.Path(path).read_text("utf-8")
        return "\ufeff" in content
    except (UnicodeDecodeError, Exception):
        return False


def strip_file_bom(path):
    """Remove BOM from start of file."""
    with open(path, "rb") as f:
        data = f.read()
    if data.startswith(FILE_BOM):
        with open(path, "wb") as f:
            f.write(data[len(FILE_BOM):])
        return True
    return False


def strip_embedded_bom(path):
    """Remove embedded U+FEFF from text content and rewrite without BOM."""
    try:
        content = pathlib.Path(path).read_text("utf-8-sig")
        if "\ufeff" in content:
            content = content.replace("\ufeff", "")
            pathlib.Path(path).write_text(content, "utf-8")
            return True
        return False
    except Exception:
        return False


def find_bom_files(root):
    found = []
    root_path = pathlib.Path(root)
    for ext in TEXT_EXTENSIONS:
        for f in root_path.glob("**/*" + ext):
            if has_file_bom(f) or has_embedded_bom(f):
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
            files = [p] if (has_file_bom(p) or has_embedded_bom(p)) else []
    else:
        repo = pathlib.Path(__file__).resolve().parent.parent
        files = []
        for d in SCAN_DIRS:
            if (repo / d).exists():
                files.extend(find_bom_files(repo / d))
        # Also scan root-level text files
        for f in repo.glob("*.md"):
            if has_file_bom(f) or has_embedded_bom(f):
                files.append(f)
        for f in repo.glob(".*"):
            if f.suffix in TEXT_EXTENSIONS or f.name in TEXT_EXTENSIONS:
                if has_file_bom(f) or has_embedded_bom(f):
                    files.append(f)

    if not files:
        print("OK: no BOM found")
        sys.exit(0)

    print(f"Found {len(files)} file(s) with BOM:")
    for f in files:
        issues = []
        if has_file_bom(f):
            issues.append("file-level BOM")
        if has_embedded_bom(f):
            issues.append("embedded U+FEFF")
        print(f"  {f}  ({", ".join(issues)})")
        if fix:
            file_fixed = strip_file_bom(f)
            embed_fixed = strip_embedded_bom(f)
            if file_fixed or embed_fixed:
                print(f"    -> fixed")

    if fix:
        print("All fixed.")
    else:
        print("Run with --fix to remove BOM.")
        sys.exit(1)


if __name__ == "__main__":
    main()
