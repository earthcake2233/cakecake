# scripts

## 日常

| 脚本 | 用途 |
|------|------|
| `npm run check:encoding` | 检查 `src/pages/minibili`、`src/i18n` 等是否含 `????` / 乱码 |

## 维护（一般不需要）

| 脚本 | 说明 |
|------|------|
| `python scripts/rebuild-personal-space.py` | 历史：从快照重建 `PersonalSpace.vue`（依赖已删除的 `.broken` 参考文件，**勿随意运行**） |
| `python scripts/restore-personal-space-encoding.py` | 历史：按行合并修复乱码（同上，参考文件已移除） |
| `python scripts/patch-collect-video-menu.py` | 一次性补丁脚本，新功能开发不必使用 |

改 `PersonalSpace.vue` 中文文案时，优先编辑 **`src/i18n/*.zh-CN.ts`**，提交前跑 `npm run check:encoding`。详见 **[AGENTS.md](../AGENTS.md)**。
