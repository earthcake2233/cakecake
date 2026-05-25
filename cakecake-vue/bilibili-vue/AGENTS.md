# Agent / 协作说明（bilibili-vue）

## PersonalSpace.vue 编码规范

`src/pages/minibili/PersonalSpace.vue` 体积很大，在 Windows 上若用错误编码保存会出现 `????`。**禁止**对该文件用非 UTF-8 工具批量替换中文。

### 正确做法

1. **改中文文案**：编辑 `src/i18n/*.zh-CN.ts`，或在 Vue 里用 `:aria-label="t.xxx"` 引用。
2. **提交前**：`npm run check:encoding`（见 [scripts/README.md](./scripts/README.md)）。

### 禁止

- 在 PersonalSpace.vue 中硬编码大段中文后用错误编码保存
- 用只支持 ASCII 的脚本替换 UTF-8 字符串

### 历史脚本

`scripts/rebuild-personal-space.py`、`restore-personal-space-encoding.py` 为早期修复乱码时的工具，依赖的 `.broken` 参考文件已删除，**日常开发勿用**。
