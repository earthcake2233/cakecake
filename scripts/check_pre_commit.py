#!/usr/bin/env python3
"""Pre-commit check: R-CLEAN-4, R-CLEAN-3, R-DOC-6.
Usage: python scripts/check_pre_commit.py
Exit 0 = OK, 1 = blocked.
"""
import pathlib, re, subprocess, sys

REPO = pathlib.Path(__file__).resolve().parent.parent

def run(cmd):
    r = subprocess.run(cmd, capture_output=True, text=True, cwd=REPO)
    return r.stdout.strip(), r.returncode

def main():
    # 1. Show status
    status, _ = run(["git", "status", "--short"])
    if not status:
        print("  Nothing to commit.")
        sys.exit(0)

    staged = [l for l in status.splitlines() if l[:1] in ("A", "M")]
    print(f"\n  Staged files ({len(staged)}):")
    for l in staged:
        print(f"    {l}")

    diffstat, _ = run(["git", "diff", "--cached", "--stat"])
    if diffstat:
        print(f"\n  Diff stat:\n{diffstat}")

    # 2. Check sensitive content
    sensitive_ok = True
    sensitive_patterns = [
        r"CODECOV_TOKEN=", r"password\s*=", r"secret\s*=",
        r"PRIVATE_KEY", r"ACCESS_KEY_ID", r"ACCESS_KEY_SECRET",
    ]
    files, _ = run(["git", "diff", "--cached", "--name-only"])
    for f in files.splitlines():
        if not f:
            continue
        try:
            text = (REPO / f).read_text(encoding="utf-8", errors="replace")
            for pat in sensitive_patterns:
                if re.search(pat, text, re.IGNORECASE):
                    print(f"  [!] {f} contains sensitive data ({pat})")
                    sensitive_ok = False
        except Exception:
            pass

    # 3. Confirm
    print("\n" + "=" * 50)
    if not sensitive_ok:
        print("  [!] Sensitive content detected. Fix before commit.")
        sys.exit(1)

    ans = input("  Confirm commit? (y/N): ").strip().lower()
    if ans != "y":
        print("  Cancelled.")
        sys.exit(1)
    print("  OK, proceed with commit.")

if __name__ == "__main__":
    main()