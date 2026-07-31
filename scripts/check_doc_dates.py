#!/usr/bin/env python3
"""Hard gate: every modified .md file that declares a "last updated" header
field must have that field set to today's date.

Enforces R-DOC-7 / R-DOC-14: any edited doc must bump its header date.

Usage: python scripts/check_doc_dates.py
Exit 0 = OK, 1 = blocked.
"""
import datetime
import pathlib
import re
import subprocess
import sys

REPO = pathlib.Path(__file__).resolve().parent.parent

# Matches CN (`**最后更新**：2026-07-31`) and EN (`**Last Updated**: 2026-07-31`)
# header lines. The date MUST be the only value on the line.
DATE_LINE_RE = re.compile(
    r"^\*\*(最后更新|Last [Uu]pdated)\*\*\s*[:：]\s*(\d{4}-\d{2}-\d{2})\s*$"
)


def get_staged_md_files():
    """Return staged .md file paths (added/modified/copied/renamed)."""
    r = subprocess.run(
        ["git", "diff", "--cached", "--name-only", "--diff-filter=ACM"],
        capture_output=True,
        text=True,
        cwd=str(REPO),
    )
    out = r.stdout
    return [f for f in out.splitlines() if f.endswith(".md")]


def main():
    files = get_staged_md_files()
    if not files:
        print("[SKIP] doc date check (no staged .md files)")
        return 0

    today = datetime.date.today().isoformat()
    failed = False
    for f in sorted(files):
        path = REPO / f
        if not path.exists():
            continue  # deleted file: no header date to check
        text = path.read_text(encoding="utf-8", errors="replace")
        dates = [m.group(2) for m in DATE_LINE_RE.finditer(text)]
        if not dates:
            continue  # file does not declare a last-updated field
        if today not in dates:
            print(
                f"[FAIL] {f}: header last-updated {dates} != today {today}; "
                "edited docs must bump the date to today"
            )
            failed = True

    if failed:
        print("  Fix: update **最后更新** / **Last Updated** in the edited docs to today's date.")
        return 1

    print(f"[OK]   doc date check ({len(files)} file(s))")
    return 0


if __name__ == "__main__":
    sys.exit(main())
