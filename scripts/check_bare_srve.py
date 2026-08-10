#!/usr/bin/env python3
"""Fail the build when handler integration tests call `srve(...)` without
checking the response (bare smoke calls).

The goal is to make coverage come from behavioral verification: every HTTP
call should either be assigned to a recorder that is then asserted, or be
followed by an assertion on status/body/DB side effects.

Usage:
    python3 scripts/check_bare_srve.py            # check against baseline
    python3 scripts/check_bare_srve.py --update   # record the current count

The baseline lives in scripts/bare_srve_baseline.txt. Cleaning up legacy files
should always lower the baseline; raising it requires a conscious decision.
"""

import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
HANDLER_TESTS = sorted((ROOT / "internal" / "handler").glob("*_test.go"))
BASELINE = ROOT / "scripts" / "bare_srve_baseline.txt"

ASSERT_MARKERS = (
    "require.",
    "assert.",
    "t.Fatal",
    "t.Error",
    "t.Skip",
    "codeFrom(",
    "switch ",
    "if ",
    "for ",
)

SRVE_RE = re.compile(r"\bsrve\(")
ASSIGN_RE = re.compile(r":=\s*srve\(")
DEF_RE = re.compile(r"func srve\(")


def is_assertion_line(line: str) -> bool:
    return any(marker in line for marker in ASSERT_MARKERS)


def count_bare_calls(path: pathlib.Path) -> tuple[int, list[tuple[int, str]]]:
    lines = path.read_text(encoding="utf-8").splitlines()
    bare = 0
    sites: list[tuple[int, str]] = []
    for i, line in enumerate(lines):
        if DEF_RE.search(line) or not SRVE_RE.search(line) or ASSIGN_RE.search(line):
            continue
        # A bare call is only tolerated when an assertion shows up within the
        # next two non-empty lines (e.g. a guard that checks the returned code).
        ok = False
        checked = 0
        for j in range(i + 1, min(i + 3, len(lines))):
            candidate = lines[j].strip()
            if not candidate:
                continue
            checked += 1
            if is_assertion_line(candidate):
                ok = True
                break
            if checked >= 2:
                break
        if not ok:
            bare += 1
            sites.append((i + 1, line.strip()))
    return bare, sites


def main() -> int:
    per_file: dict[str, int] = {}
    sites_by_file: dict[str, list[tuple[int, str]]] = {}
    total = 0
    for path in HANDLER_TESTS:
        n, sites = count_bare_calls(path)
        if n:
            per_file[str(path.relative_to(ROOT))] = n
            sites_by_file[str(path.relative_to(ROOT))] = sites
        total += n

    print(f"bare srve calls: {total}")
    for name, n in sorted(per_file.items(), key=lambda kv: (-kv[1], kv[0])):
        print(f"  {n:3d}  {name}")
        for line_no, text in sites_by_file[name][:3]:
            print(f"        L{line_no}: {text[:110]}")

    if "--update" in sys.argv:
        BASELINE.write_text(f"{total}\n", encoding="utf-8")
        print(f"baseline updated -> {total} (scripts/bare_srve_baseline.txt)")
        return 0

    if not BASELINE.exists():
        print("missing baseline; run with --update first", file=sys.stderr)
        return 1
    limit = int(BASELINE.read_text(encoding="utf-8").strip())
    if total > limit:
        print(
            f"FAIL: bare srve count {total} exceeds baseline {limit}; "
            "assert the response instead of calling srve blindly.",
            file=sys.stderr,
        )
        return 1
    print(f"ok: bare srve count {total} <= baseline {limit}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
