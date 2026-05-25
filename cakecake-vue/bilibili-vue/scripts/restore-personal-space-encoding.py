# -*- coding: utf-8 -*-
"""
Restore corrupted Chinese in PersonalSpace.vue using PersonalSpace.vue.broken
(same line skeleton) + known collect-tab strings.

Run: python scripts/restore-personal-space-encoding.py
"""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PS = ROOT / "src/pages/minibili/PersonalSpace.vue"
BROKEN = ROOT / "src/pages/minibili/PersonalSpace.vue.broken"

QUOTE_RE = re.compile(r'("([^"\\]|\\.)*")|(\'([^\'\\]|\\.)*\')')


def u(*codes: int) -> str:
    return "".join(chr(c) for c in codes)


COLLECT_OVERRIDES: dict[str, str] = {
    'aria-label="????"': f'aria-label="{u(0x6536, 0x85CF, 0x5206, 0x7C7B)}"',
    "<span>???????</span>": f"<span>{u(0x6211, 0x521B, 0x5EFA, 0x7684, 0x6536, 0x85CF, 0x5939)}</span>",
    'title: "?????",': f'title: "{u(0x9ED8, 0x8BA4, 0x6536, 0x85CF, 0x5939)}",',
}


def is_corrupted_text(text: str) -> bool:
    if not text:
        return False
    if "\ufffd" in text:
        return True
    if re.fullmatch(r"\?+", text):
        return True
    if re.search(r"\?{2,}", text) and not re.search(r"https?://", text):
        return True
    return False


def line_is_corrupted(line: str) -> bool:
    if "\ufffd" in line:
        return True
    for m in QUOTE_RE.finditer(line):
        inner = m.group(0)[1:-1]
        if is_corrupted_text(inner):
            return True
    if re.search(r">\s*\?{4,}\s*<", line):
        return True
    if re.search(r">\s*\?{4,}\s*$", line.strip()):
        return True
    return False


def skeleton(line: str) -> str:
    def repl(match: re.Match[str]) -> str:
        raw = match.group(0)
        inner = raw[1:-1]
        if is_corrupted_text(inner) or re.search(r"[\u4e00-\u9fff]", inner):
            return '""' if raw[0] == '"' else "''"
        return raw

    out = QUOTE_RE.sub(repl, line)
    out = re.sub(r">\s*[^<\s][^<]{0,40}\s*<", "><", out)
    return out.strip()


def build_reference_map(ref_text: str) -> dict[str, str]:
    mapping: dict[str, str] = {}
    for line in ref_text.splitlines():
        if line_is_corrupted(line):
            continue
        sk = skeleton(line)
        if sk:
            mapping[sk] = line
    return mapping


def apply_collect_overrides(text: str) -> str:
    i0 = text.find('v-else-if="activeNav === \'collect\'"')
    i1 = text.find('v-else-if="activeNav === \'settings\'"', i0)
    if i0 < 0 or i1 < 0:
        return text
    head, block, tail = text[:i0], text[i0:i1], text[i1:]
    for old, new in COLLECT_OVERRIDES.items():
        block = block.replace(old, new)
    collect = {
        "sidenav_aria": u(0x6536, 0x85CF, 0x5206, 0x7C7B),
        "folders_nav": u(0x6211, 0x521B, 0x5EFA, 0x7684, 0x6536, 0x85CF, 0x5939),
        "watch_later": u(0x7A0D, 0x540E, 0x518D, 0x770B),
        "public": u(0x516C, 0x5F00),
        "private": u(0x79C1, 0x5BC6),
        "play_all": u(0x64AD, 0x653E, 0x5168, 0x90E8),
        "batch": u(0x6279, 0x91CF, 0x64CD, 0x4F5C),
        "sort_aria": u(0x6536, 0x85CF, 0x6392, 0x5E8F),
        "sort_recent": u(0x6700, 0x8FD1, 0x6536, 0x85CF),
        "sort_play": u(0x6700, 0x591A, 0x64AD, 0x653E),
        "sort_submit": u(0x6700, 0x8FD1, 0x6295, 0x7A3F),
        "opt_current": u(0x5F53, 0x524D, 0x6536, 0x85CF, 0x5939),
        "opt_all": u(0x5168, 0x90E8, 0x6536, 0x85CF, 0x5939),
        "kw": u(0x8BF7, 0x8F93, 0x5165, 0x5173, 0x952E, 0x8BCD),
        "search": u(0x641C, 0x7D22),
        "feed_aria": u(0x6536, 0x85CF, 0x89C6, 0x9891, 0x5217, 0x8868),
        "loading": u(0x52A0, 0x8F7D, 0x4E2D, 0x2026),
        "empty_fav": u(0x6682, 0x65E0, 0x6536, 0x85CF, 0x89C6, 0x9891),
        "later_list": u(0x7A0D, 0x540E, 0x518D, 0x770B, 0x89C6, 0x9891, 0x5217, 0x8868),
        "later_empty_aria": u(0x7A0D, 0x540E, 0x518D, 0x770B, 0x6682, 0x65E0, 0x5185, 0x5BB9),
        "later_empty_own": u(0x8FD8, 0x6CA1, 0x6709, 0x7A0D, 0x540E, 0x518D, 0x770B, 0x7684, 0x89C6, 0x9891),
        "later_empty_guest": u(
            0x4EC5, 0x53EF, 0x5728, 0x81EA, 0x5DF1, 0x7684, 0x7A7A, 0x95F4, 0x67E5, 0x770B, 0x7A0D, 0x540E, 0x518D, 0x770B
        ),
        "default_folder": u(0x9ED8, 0x8BA4, 0x6536, 0x85CF, 0x5939),
        "video_count": u(0x89C6, 0x9891, 0x6570, 0x003A),
        "fav_at": u(0x6536, 0x85CF, 0x4E8E),
        "dot": u(0x00B7),
    }
    block = re.sub(
        r'class="mb-space__collect-sidenav" aria-label="[^"]*"',
        f'class="mb-space__collect-sidenav" aria-label="{collect["sidenav_aria"]}"',
        block,
        count=1,
    )
    block = re.sub(
        r"<span>[^<]{3,12}</span>\s*<svg\s+class=\"mb-space__collect-nav-chev\"",
        f'<span>{collect["folders_nav"]}</span>\n                  <svg\n                    class="mb-space__collect-nav-chev"',
        block,
        count=1,
    )
    block = re.sub(
        r'activeCollectFolder\.isPublic \? "[^"]*" : "[^"]*"',
        f'activeCollectFolder.isPublic ? "{collect["public"]}" : "{collect["private"]}"',
        block,
        count=1,
    )
    block = re.sub(
        r'<span class="mb-space__collect-folder-meta-sep">[^<]*</span>',
        f'<span class="mb-space__collect-folder-meta-sep">{collect["dot"]}</span>',
        block,
        count=1,
    )
    block = re.sub(
        r"<span>[^<]*\{\{ activeCollectFolder\.videoCount \}\}</span>",
        f'<span>{collect["video_count"]} {{{{ activeCollectFolder.videoCount }}}}</span>',
        block,
        count=1,
    )
    block = re.sub(
        r'(class="mb-space__play-all mb-space__play-all--collect"[\s\S]*?</svg>\s*)\S+',
        rf"\1{collect['play_all']}",
        block,
        count=2,
    )
    block = re.sub(
        r'class="mb-space__collect-batch"[^>]*>\s*\S+',
        f'class="mb-space__collect-batch"\n                    disabled\n                  >\n                    {collect["batch"]}',
        block,
        count=1,
    )
    block = re.sub(
        r'class="mb-space__subtabs mb-space__collect-subtabs"[\s\S]{0,80}aria-label="[^"]*"',
        f'class="mb-space__subtabs mb-space__collect-subtabs"\n                    role="group"\n                    aria-label="{collect["sort_aria"]}"',
        block,
        count=1,
    )
    for key, sort_key in (
        ("sort_recent", "recent"),
        ("sort_play", "play"),
        ("sort_submit", "submit"),
    ):
        block = re.sub(
            rf"@click=\"collectSort = '{sort_key}'\"\s*>\s*[^\n<]+",
            f"@click=\"collectSort = '{sort_key}'\"\n                    >\n                      {collect[key]}",
            block,
            count=1,
        )
    block = re.sub(
        r'<option value="current">[^<]+</option>',
        f'<option value="current">{collect["opt_current"]}</option>',
        block,
        count=1,
    )
    block = re.sub(
        r'<option value="all">[^<]+</option>',
        f'<option value="all">{collect["opt_all"]}</option>',
        block,
        count=1,
    )
    block = re.sub(
        r'placeholder="[^"]*"\s+autocomplete="off"\s*/>\s*<button\s+type="button"\s+class="mb-space__collect-search-btn"',
        f'placeholder="{collect["kw"]}"\n                        autocomplete="off"\n                      />\n                      <button\n                        type="button"\n                        class="mb-space__collect-search-btn"',
        block,
        count=1,
    )
    block = re.sub(
        r'class="mb-space__collect-search-btn"\s+aria-label="[^"]*"',
        f'class="mb-space__collect-search-btn"\n                        aria-label="{collect["search"]}"',
        block,
        count=1,
    )
    block = re.sub(
        r'class="mb-space__collect-feed" aria-label="[^"]*"',
        f'class="mb-space__collect-feed" aria-label="{collect["feed_aria"]}"',
        block,
        count=1,
    )
    block = re.sub(
        r'collectLoading" class="mb-space__hint">[^<]+',
        f'collectLoading" class="mb-space__hint">{collect["loading"]}',
        block,
        count=1,
    )
    block = re.sub(
        r'class="mb-space__collect-fav-at"[^>]*>[^<{]+',
        f'class="mb-space__collect-fav-at"\n                            >{collect["fav_at"]}{{{{ formatFavoritedMD(v.favorited_at) }}}}',
        block,
        count=1,
    )
    block = re.sub(
        r'role="img"\s+aria-label="[^"]*"\s*>\s*<img :src="dynEmptyImg" alt="" />\s*</motion>\s*</template>',
        f'role="img"\n                    aria-label="{collect["empty_fav"]}"\n                  >\n                    <img :src="dynEmptyImg" alt="" />\n                  </motion>\n              </template>'.replace(
            "motion", "motion"
        ),
        block,
        count=0,
    )
    block = block.replace(
        'role="img"\n                    aria-label="??????"',
        f'role="img"\n                    aria-label="{collect["empty_fav"]}"',
        1,
    )
    block = re.sub(
        r"<h2 class=\"mb-space__collect-later-title\">[^<]+</h2>",
        f'<h2 class="mb-space__collect-later-title">{collect["watch_later"]}</h2>',
        block,
        count=1,
    )
    block = re.sub(
        r'watchLaterLoading" class="mb-space__hint">[^<]+',
        f'watchLaterLoading" class="mb-space__hint">{collect["loading"]}',
        block,
        count=1,
    )
    block = re.sub(
        r'aria-label="[^"]*"\s*>\s*<li\s+v-for="v in watchLaterVideos"',
        f'aria-label="{collect["later_list"]}"\n                >\n                  <li\n                    v-for="v in watchLaterVideos"',
        block,
        count=1,
    )
    block = block.replace(
        'mb-space__collect-later-empty"\n                  role="img"\n                  aria-label="????????"',
        f'mb-space__collect-later-empty"\n                  role="img"\n                  aria-label="{collect["later_empty_aria"]}"',
        1,
    )
    block = re.sub(
        r'\{\{ isOwnSpace \? "[^"]*" : "[^"]*" \}\}',
        f'{{{{ isOwnSpace ? "{collect["later_empty_own"]}" : "{collect["later_empty_guest"]}" }}}}',
        block,
        count=1,
    )
    block = re.sub(
        r'(@click="selectCollectLater"\s*>\s*)[^\n<]+',
        rf"\1{collect['watch_later']}",
        block,
        count=1,
    )
    return head + block + tail


def line_signature(line: str) -> str:
    """Stable key from tags/classes — avoids wrong skeleton collisions."""
    parts = re.findall(
        r'(?:class|aria-label|placeholder|role|type|@click|v-else-if|v-if)=["\'][^"\']*["\']',
        line,
    )
    return "|".join(parts)


def merge_from_broken(current: str, ref_map: dict[str, str]) -> tuple[str, int]:
    lines = current.splitlines()
    fixed = 0
    for i, line in enumerate(lines):
        if not line_has_garbled_markers(line):
            continue
        sk = skeleton(line)
        ref = ref_map.get(sk)
        if ref is not None:
            lines[i] = ref
            fixed += 1
    return "\n".join(lines) + ("\n" if current.endswith("\n") else ""), fixed


def graft_by_signature(current_lines: list[str], broken_lines: list[str]) -> int:
    from collections import defaultdict

    buckets: dict[str, list[str]] = defaultdict(list)
    for line in broken_lines:
        if line_is_corrupted(line):
            continue
        sig = line_signature(line)
        if sig:
            buckets[sig].append(line)
    ref_by_sig = {k: v[0] for k, v in buckets.items() if len(v) == 1}

    fixed = 0
    for i, line in enumerate(current_lines):
        if not line_has_garbled_markers(line):
            continue
        sig = line_signature(line)
        ref = ref_by_sig.get(sig)
        if ref is not None:
            current_lines[i] = ref
            fixed += 1
    return fixed


def line_has_garbled_markers(line: str) -> bool:
    if line_is_corrupted(line):
        return True
    if re.search(r">\s*\?{2,}\s*<", line):
        return True
    if re.search(r">\s*\?{2,}\s*$", line.rstrip()):
        return True
    if re.search(r'"\?{2,}"', line):
        return True
    return False


def post_fix_known(text: str) -> str:
    text = text.replace(
        'placeholder="开启评论精选"',
        f'placeholder="{u(0x641C, 0x7D22, 0x89C6, 0x9891, 0x3001, 0x52A8, 0x6001)}"',
    )
    stat_labels = [
        u(0x5173, 0x6CE8),
        u(0x7C89, 0x4E1D),
        u(0x83B7, 0x8D5E),
        u(0x64AD, 0x653E),
    ]
    for label in stat_labels:
        text = text.replace(
            '<span class="mb-space__stat-k">??</span>',
            f'<span class="mb-space__stat-k">{label}</span>',
            1,
        )
    text = re.sub(
        r'class="mb-space__stats" aria-label="[^"]*"',
        f'class="mb-space__stats" aria-label="{u(0x7A7A, 0x95F4, 0x6570, 0x636E)}"',
        text,
        count=1,
    )
    text = text.replace(
        "开启评论精选????",
        u(0x6682, 0x6CA1, 0x6709, 0x66F4, 0x591A, 0x52A8, 0x6001, 0x4E86, 0xFF5E),
    )
    return text


def patch_dyn_station(text: str, broken: str) -> str:
    m = re.search(
        r"const DYN_MB_STATION = \{[\s\S]*?\n\};",
        broken,
    )
    if not m:
        return text
    return re.sub(
        r"const DYN_MB_STATION = \{[\s\S]*?\n\};",
        m.group(0),
        text,
        count=1,
    )


def patch_nav_tabs(text: str) -> str:
    nav = {
        "home": u(0x4E3B, 0x9875),
        "dynamic": u(0x52A8, 0x6001),
        "contribute": u(0x6295, 0x7A3F),
        "collect": u(0x6536, 0x85CF),
        "settings": u(0x8BBE, 0x7F6E),
    }
    for key, label in nav.items():
        text = re.sub(
            rf'(key: "{key}",\s*label: )"[^"]*"',
            rf'\1"{label}"',
            text,
            count=1,
        )
    return text


def patch_default_folder(text: str) -> str:
    title = u(0x9ED8, 0x8BA4, 0x6536, 0x85CF, 0x5939)
    return re.sub(r'title: "[^"]*",\s*isPublic: true,\s*videoCount:', f'title: "{title}",\n          isPublic: true,\n          videoCount:', text)


def count_corruption(text: str) -> int:
    return sum(1 for line in text.splitlines() if line_has_garbled_markers(line))


def main() -> None:
    if not PS.exists():
        raise SystemExit(f"missing {PS}")
    if not BROKEN.exists():
        raise SystemExit(f"missing {BROKEN}")

    current = PS.read_bytes().decode("utf-8", errors="replace")
    broken = BROKEN.read_text(encoding="utf-8")
    ref_map = build_reference_map(broken)

    before = count_corruption(current)
    current_lines = current.splitlines()
    n_sig = graft_by_signature(current_lines, broken.splitlines())
    current = "\n".join(current_lines) + ("\n" if current.endswith("\n") else "")
    current, n_fixed = merge_from_broken(current, ref_map)
    current = post_fix_known(current)
    current = patch_dyn_station(current, broken)
    current = patch_nav_tabs(current)
    current = patch_default_folder(current)
    current = apply_collect_overrides(current)
    after = count_corruption(current)

    print(
        f"restored: signature_lines={n_sig}, skeleton_lines={n_fixed}, "
        f"corrupted_lines {before} -> {after}"
    )
    PS.write_text(current, encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()
