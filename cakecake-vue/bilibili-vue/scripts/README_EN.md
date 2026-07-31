<p align="center">
  <a href="cakecake-vue/bilibili-vue/scripts/README.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

  </a>
  </a>
</p>

  </a>
</p>

# scripts

## Daily Use

| Script | Purpose |
|--------|---------|
| `npm run check:encoding` | Check `src/pages/minibili`, `src/i18n`, etc. for `????` / garbled characters |

## Maintenance (Generally Not Needed)

| Script | Description |
|--------|-------------|
| `python scripts/rebuild-personal-space.py` | Historical: rebuild `PersonalSpace.vue` from snapshot (depends on deleted `.broken` reference files, **do NOT run casually**) |
| `python scripts/restore-personal-space-encoding.py` | Historical: line-by-line merge fix for garbled encoding (same as above, reference files removed) |
| `python scripts/patch-collect-video-menu.py` | One-time patch script, not needed for new feature development |

When editing Chinese text in `PersonalSpace.vue`, prefer editing **`src/i18n/*.zh-CN.ts`** and run `npm run check:encoding` before committing. See the encoding notes in the main project [AGENTS.md](../AGENTS.md).

---

## Encoding Notes

Some Vue SFC files in `src/pages/minibili/` contain inline Chinese text (video zone names, UI labels, etc.). When editing these files on Windows with certain editors, garbled characters (`????`, `U+FFFD`) may appear. Always run `npm run check:encoding` before committing.
