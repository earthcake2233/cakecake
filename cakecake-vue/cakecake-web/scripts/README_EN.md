<p align="center">
  <a href="README.md">
    <img src="https://img.shields.io/badge/🇨🇳中文-999999?style=flat-square" alt="中文">
  </a>
  <strong><img src="https://img.shields.io/badge/🇬🇧English-00a1d6?style=flat-square" alt="English"></strong>
</p>

# scripts

## Daily Use

| Script | Purpose |
|--------|---------|
| `npm run check:encoding` | Check `src/pages/cakecake`, `src/i18n`, etc. for `????` / garbled characters |

When editing Chinese text in `PersonalSpace.vue`, prefer editing **`src/i18n/*.zh-CN.ts`** and run `npm run check:encoding` before committing. See the **Encoding Notes** section above.

---

## Encoding Notes

Some Vue SFC files in `src/pages/cakecake/` contain inline Chinese text (video zone names, UI labels, etc.). When editing these files on Windows with certain editors, garbled characters (`????`, `U+FFFD`) may appear. Always run `npm run check:encoding` before committing.
