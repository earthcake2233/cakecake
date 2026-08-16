#!/usr/bin/env python3
"""Check that all paired .md files are in sync (CN <-> EN).

Usage:
  python scripts/check_en_sync.py          # Check all files
  python scripts/check_en_sync.py --fix    # Print fix suggestions
  python scripts/check_en_sync.py --check-sync  # Existence + content/structure sync
"""

import os
import sys
import hashlib
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

# Files excluded from EN sync check (personal docs, incident reports, etc.)
# These are gitignored and do not need EN counterparts
SKIP_PATTERNS = [
    "incident-*.md",
    "service-refactor-*.md",
    "docker-config*.md",
    "ci-checks*.md",
    # 个人面试准备材料，不要求英文对照，也不提交仓库
    "interview-testing*.md",
    "interview-core-chains*.md",
    "INTERVIEW_SPRINT*.md",
    "resume-optimized-go-intern*.md",
    "skill_frequency*.md",
    "onsite_shortlist*.md",
]

def extract_heading_levels(text):
    """Extract the heading-level sequence of a markdown doc (structure only)."""
    levels = []
    in_block = False
    for line in text.splitlines():
        if line.strip().startswith("```"):
            in_block = not in_block
            continue
        if in_block:
            continue
        m = re.match(r"^(#{1,6})\s+\S", line)
        if m:
            levels.append(len(m.group(1)))
    return levels


def extract_code_blocks(text):
    """Extract fenced code blocks, normalized (rstrip lines, strip outer blank lines)."""
    blocks = []
    buf = []
    in_block = False
    for line in text.splitlines():
        if line.strip().startswith("```"):
            if in_block:
                blocks.append("\n".join(buf).strip())
                buf = []
                in_block = False
            else:
                in_block = True
        elif in_block:
            buf.append(line.rstrip())
    if in_block:
        blocks.append("\n".join(buf).strip())
    return blocks


def is_gitignored(rel_path):
    """Best-effort gitignore check: exact path or basename listed in .gitignore."""
    gitignore_path = ROOT / ".gitignore"
    if not gitignore_path.exists():
        return False
    text = gitignore_path.read_text(encoding="utf-8", errors="replace")
    basename = os.path.basename(rel_path)
    lines = [ln.strip() for ln in text.splitlines() if ln.strip() and not ln.lstrip().startswith("#")]
    if basename in lines or rel_path in lines:
        return True
    for pat in lines:
        if pat.endswith("/") and rel_path.startswith(pat):
            return True
        if "*" in pat:
            import fnmatch
            if fnmatch.fnmatch(rel_path, pat) or fnmatch.fnmatch(basename, pat):
                return True
    return False


def check_pair_content(cn_path, en_path):
    """Compare the heading structure (level sequence) and code-block count of a CN/EN pair.
    Only structure is compared — translated titles legitimately differ (R-DOC-3).
    Returns a list of problem strings (empty = in sync)."""
    problems = []
    cn = (ROOT / cn_path).read_text(encoding="utf-8", errors="replace")
    en = (ROOT / en_path).read_text(encoding="utf-8", errors="replace")

    cn_levels = extract_heading_levels(cn)
    en_levels = extract_heading_levels(en)
    if cn_levels != en_levels:
        from collections import Counter
        cn_cnt = dict(sorted(Counter(cn_levels).items()))
        en_cnt = dict(sorted(Counter(en_levels).items()))
        problems.append(
            "  章节层级结构不一致: CN %d 个标题 %s vs EN %d 个标题 %s"
            % (len(cn_levels), cn_cnt, len(en_levels), en_cnt)
        )

    cn_code_count = len(extract_code_blocks(cn))
    en_code_count = len(extract_code_blocks(en))
    if cn_code_count != en_code_count:
        problems.append("  代码块数量不一致: CN %d vs EN %d" % (cn_code_count, en_code_count))

    return problems

def find_md_files():
    """Find all .md files in project (excluding node_modules, .git)."""
    md_files = []
    for root, dirs, files in os.walk(ROOT):
        dirs[:] = [d for d in dirs if d not in ("node_modules", ".git", ".gocache", ".gotmp", ".gopath", "bin", "tmp", "dist")]
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
    import argparse
    parser = argparse.ArgumentParser(description="CN/EN .md pairing & sync check")
    parser.add_argument("--fix", action="store_true", help="Print fix suggestions")
    parser.add_argument("--check-sync", action="store_true", help="Also verify content/structure sync of each pair")
    args = parser.parse_args()

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
    
    sync_issues = []
    if args.check_sync:
        for cn in sorted(cn_files):
            if should_skip(cn):
                continue
            en = cn.replace(".md", "_EN.md")
            if en in en_files:
                if is_gitignored(cn) or is_gitignored(en):
                    continue
                problems = check_pair_content(cn, en)
                if problems:
                    sync_issues.append((cn, en, problems))

    if issues or sync_issues:
        print(f"Found {len(issues) + len(sync_issues)} issue(s):")
        for issue in issues:
            print(f"  - {issue}")
        for cn, en, problems in sync_issues:
            print(f"  - [SYNC] {cn} <-> {en}")
            for p in problems:
                print(p)
        print(f"\nRun with --fix for suggestions.")
        return 1
    else:
        extra = " and content/structure sync" if args.check_sync else ""
        print(f"OK: all .md files have EN counterparts, are properly gitignored{extra}.")
        return 0

if __name__ == "__main__":
    sys.exit(main())
