<p align="center">
  <strong><img src="https://img.shields.io/badge/🇨🇳中文-00a1d6?style=flat-square" alt="中文"></strong>
  <a href="README_EN.md">
    <img src="https://img.shields.io/badge/🇬🇧English-999999?style=flat-square" alt="English">
  </a>
</p>

# scripts

## 日常

| 脚本 | 用途 |
|------|------|
| `npm run check:encoding` | 检查 `src/pages/cakecake`、`src/i18n` 等是否含 `????` / 乱码 |

改 `PersonalSpace.vue` 中文文案时，优先编辑 **`src/i18n/*.zh-CN.ts`**，提交前跑 `npm run check:encoding`。详见 **本文件上方的编码说明**。

## 编码说明

`src/pages/cakecake/` 下的部分 Vue SFC 文件内联了中文文案（视频分区名、UI 标签等）。在 Windows 上使用某些编辑器修改这些文件时，可能出现乱码（`????`、`U+FFFD`）。提交前务必运行 `npm run check:encoding`。
