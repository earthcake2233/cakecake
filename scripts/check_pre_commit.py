#!/usr/bin/env python3
"""Pre-commit gate: BOM check + go vet + sensitive data scan.

Hard blocks:
  - UTF-8 BOM in any .go/.md/.yaml/.json/.py file
  - go vet failures in staged packages
  - Sensitive tokens in staged files

Usage: python scripts/check_pre_commit.py [--no-vet] [--yes]
Exit 0 = OK, 1 = blocked.
"""
import pathlib, re, subprocess, sys

REPO = pathlib.Path(__file__).resolve().parent.parent


def run(cmd):
    r = subprocess.run(cmd, capture_output=True, text=True, cwd=str(REPO))
    return r.stdout.strip(), r.returncode


def get_staged_go_files():
    """Return list of staged .go file paths."""
    out, _ = run(["git", "diff", "--cached", "--name-only", "--diff-filter=ACM"])
    return [f for f in out.splitlines() if f.endswith(".go")]


def get_staged_packages():
    """Return unique Go package paths from staged .go files."""
    files = get_staged_go_files()
    pkgs = set()
    for f in files:
        p = pathlib.Path(f)
        if p.parts and p.parts[0] in ("internal", "cmd"):
            parts = list(p.parts[:-1])
            if not parts:
                continue
            pkgs.add("/".join(parts))
    return sorted(pkgs)


def check_bom():
    """Run check_bom.py. Returns True if clean."""
    bom_script = REPO / "scripts" / "check_bom.py"
    out, code = run([sys.executable, str(bom_script)])
    if code != 0:
        print("[FAIL] BOM check failed:")
        print(out)
        print("  Run: python scripts/check_bom.py --fix")
        return False
    print("[OK]   BOM check")
    return True


def check_vet():
    """Run go vet on staged packages. Returns True if clean."""
    pkgs = get_staged_packages()
    if not pkgs:
        print("[SKIP] go vet (no staged Go files)")
        return True

    all_ok = True
    for pkg in pkgs:
        pkg_path = f"./{pkg}"
        out, code = run(["go", "vet", pkg_path])
        if code != 0:
            print(f"[FAIL] go vet {pkg}:")
            for line in out.splitlines():
                if line.strip():
                    print(f"  {line}")
            all_ok = False

    if all_ok:
        print(f"[OK]   go vet ({len(pkgs)} package(s))")
    return all_ok


def check_sensitive():
    """Scan staged files for secrets. Returns True if clean."""
    patterns = [
        (r"CODECOV_TOKEN=", "Codecov token"),
        (r'password\s*=\s*"[^"]+"', "hardcoded password"),
        (r'secret\s*=\s*"[^"]+"', "hardcoded secret"),
        (r"PRIVATE_KEY", "private key"),
        (r'ACCESS_KEY_ID\s*=\s*"[^"]+"', "access key ID"),
        (r'ACCESS_KEY_SECRET\s*=\s*"[^"]+"', "access key secret"),
        (r'DEEPSEEK_API_KEY\s*=\s*"[^"]+"', "DeepSeek API key"),
    ]
    files, _ = run(["git", "diff", "--cached", "--name-only"])
    ok = True
    for f in files.splitlines():
        if not f:
            continue
        try:
            text = (REPO / f).read_text(encoding="utf-8", errors="replace")
            for pat, label in patterns:
                if re.search(pat, text, re.IGNORECASE):
                    print(f"[FAIL] Sensitive data in {f}: {label}")
                    ok = False
        except Exception:
            pass

    if ok:
        print("[OK]   Sensitive data scan")
    return ok


def main():
    import argparse
    parser = argparse.ArgumentParser(description="Pre-commit gate")
    parser.add_argument("--no-vet", action="store_true", help="Skip go vet (for quick doc-only commits)")
    parser.add_argument("--yes", "-y", action="store_true", help="Skip confirmation prompt")
    args = parser.parse_args()

    status, _ = run(["git", "status", "--short"])
    if not status:
        print("Nothing to commit.")
        sys.exit(0)

    staged = [l for l in status.splitlines() if l[:1] in ("A", "M", " ")]
    print(f"\nStaged files ({len(staged)}):")
    for l in staged[:30]:
        print(f"  {l}")
    if len(staged) > 30:
        print(f"  ... and {len(staged) - 30} more")

    print()
    all_pass = True

    if not check_bom():
        all_pass = False
    if not args.no_vet:
        if not check_vet():
            all_pass = False
    if not check_sensitive():
        all_pass = False

    print()

    if not all_pass:
        print("=" * 50)
        print("BLOCKED: Fix the failures above before committing.")
        print("=" * 50)
        sys.exit(1)

    print("=" * 50)
    if args.yes:
        print("All checks passed. Proceeding.")
    else:
        ans = input("  All checks passed. Confirm commit? (y/N): ").strip().lower()
        if ans != "y":
            print("Cancelled.")
            sys.exit(1)
    print("OK, proceed with commit.")


if __name__ == "__main__":
    main()
