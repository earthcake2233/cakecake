#!/usr/bin/env python3
"""Check that all paired .md files are in sync (CN <-> EN).

Usage:
  python scripts/check_en_sync.py          # Check all files
  python scripts/check_en_sync.py --fix    # Print fix suggestions
"""

import os
import sys
import hashlib
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

# Files excluded from EN sync check (personal docs, incident reports, etc.)
# These are gitignored and do not need EN counterparts
SKIP_PATTERNS = [
    "incident-*.md",
]

def find_md_files():
    """Find all .md files in project (excluding node_modules, .git)."""
    md_files = []
    for root, dirs, files in os.walk(ROOT):
        dirs[:] = [d for d in dirs if d not in ("node_modules", ".git", ".gocache", ".gotmp", "bin", "tmp", "dist")]
        for f in files:
            if f.endswith(".md"):
                rel = os.path.relpath(os.path.join(root, f), ROOT).replace("\\", "/")
                md_files.append(rel)
    return md_files

def should_skip(rel_path):
    """Check if file matches skip patterns."""
    import fnmatch
    basename = os.path.basename(rel_path)
    for pattern in SKIP_PATTERNS:
        if fnmatch.fnmatch(basename, pattern):
            return True
    return False

def main():
    md_files = find_md_files()
    
    # Group: CN files and their EN counterparts
    cn_files = {}
    en_files = {}
    for f in md_files:
        if f.endswith("_EN.md"):
            en_files[f] = True
        else:
            cn_files[f] = True
    
    issues = []
    
    # Check 1: Every CN file (that's not skipped) should have an EN counterpart
    for cn in sorted(cn_files):
        if should_skip(cn):
            continue
        en = cn.replace(".md", "_EN.md")
        if en not in en_files:
            issues.append(f"MISSING: {en} (counterpart of {cn})")
    
    # Check 2: Every EN file should have a CN counterpart
    for en in sorted(en_files):
        cn = en.replace("_EN.md", ".md")
        if cn not in cn_files:
            # OK only if CN is gitignored/skipped
            if not should_skip(cn):
                issues.append(f"ORPHAN: {en} (no counterpart {cn})")
    
    # Check 3: Verify gitignore sync for paired private files
    gitignore_path = ROOT / ".gitignore"
    if gitignore_path.exists():
        with open(gitignore_path, "r", encoding="utf-8") as f:
            gitignore = f.read()
        
        for cn in sorted(cn_files):
            if should_skip(cn):
                continue
            basename = os.path.basename(cn)
            en_basename = os.path.basename(cn.replace(".md", "_EN.md"))
            
            # Check if CN file is gitignored
            cn_ignored = basename in gitignore or any(
                line.strip() == cn for line in gitignore.split("\n")
            )
            if cn_ignored:
                # EN counterpart must also be gitignored
                en_ignored = en_basename in gitignore or any(
                    line.strip() == cn.replace(".md", "_EN.md") for line in gitignore.split("\n")
                )
                if not en_ignored:
                    issues.append(f"GITIGNORE: {cn} is in .gitignore but {cn.replace('.md', '_EN.md')} is NOT")
    
    if issues:
        print(f"Found {len(issues)} issue(s):")
        for issue in issues:
            print(f"  - {issue}")
        print(f"\nRun with --fix for suggestions.")
        return 1
    else:
        print("OK: all .md files have EN counterparts and are properly gitignored.")
        return 0

if __name__ == "__main__":
    sys.exit(main())
