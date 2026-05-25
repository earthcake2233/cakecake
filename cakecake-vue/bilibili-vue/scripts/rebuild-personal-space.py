# -*- coding: utf-8 -*-
"""
Rebuild PersonalSpace.vue from PersonalSpace.vue.broken + collect tab snapshot.

Run: python scripts/rebuild-personal-space.py
"""
from __future__ import annotations

import re
import shutil
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PS = ROOT / "src/pages/minibili/PersonalSpace.vue"
BROKEN = ROOT / "src/pages/minibili/PersonalSpace.vue.broken"
SNAPSHOT = ROOT / "scripts/snapshots/personal-space-collect-block.vue"
BACKUP = ROOT / "src/pages/minibili/PersonalSpace.vue.before-rebuild"

COLLECT_START = 'v-else-if="activeNav === \'collect\'"'
COLLECT_END = 'v-else-if="activeNav === \'settings\'"'


def extract_collect_block(text: str) -> str:
    i0 = text.find(COLLECT_START)
    i1 = text.find(COLLECT_END, i0)
    if i0 < 0 or i1 < 0:
        raise SystemExit("collect block markers not found in source")
    start = text.rfind("<div", 0, i0)
    if start < 0:
        start = i0 - 20
    return text[start:i1]


def save_snapshot(block: str) -> None:
    SNAPSHOT.parent.mkdir(parents=True, exist_ok=True)
    SNAPSHOT.write_text(block, encoding="utf-8", newline="\n")
    print("saved collect snapshot:", SNAPSHOT)


def load_collect_block(current: str) -> str:
    if SNAPSHOT.is_file():
        snap = SNAPSHOT.read_text(encoding="utf-8")
        if "collect-outer" in snap:
            return snap.rstrip() + "\n\n          "
    block = extract_collect_block(current)
    save_snapshot(block)
    return block.rstrip() + "\n\n          "


def replace_collect_in_broken(broken: str, collect_block: str) -> str:
    pattern = re.compile(
        r"<div\s+v-else-if=\"activeNav === 'collect'\"[\s\S]*?"
        r"(?=\s*<div v-else-if=\"activeNav === 'settings'\")"
    )
    m = pattern.search(broken)
    if not m:
        raise SystemExit("collect placeholder not found in .broken")
    return broken[: m.start()] + collect_block + broken[m.end() :]


def main() -> None:
    if not BROKEN.is_file():
        raise SystemExit(f"missing {BROKEN}")
    if not PS.is_file():
        raise SystemExit(f"missing {PS}")

    current = PS.read_text(encoding="utf-8")
    collect_block = load_collect_block(current)

    shutil.copy2(PS, BACKUP)
    print("backup:", BACKUP)

    broken = BROKEN.read_text(encoding="utf-8")
    merged = replace_collect_in_broken(broken, collect_block)
    PS.write_text(merged, encoding="utf-8", newline="\n")
    print("rebuild complete:", PS)
    print("next: npm run check:encoding")


if __name__ == "__main__":
    main()
