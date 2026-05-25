/**
 * Mini-Bili 全局视频号：BV + 数字 id（如 BV42）。
 * 路由 params.aid、文案展示统一使用本工具。
 */

export function formatVideoBvid(id) {
  const n = Number(id);
  if (!Number.isFinite(n) || n <= 0) {
    return "";
  }
  return `BV${Math.trunc(n)}`;
}

/** 从路由 aid（BV42 / av42 / 42）解析数字视频 id */
export function parseVideoIdFromRoute(aid) {
  const raw = String(aid ?? "").trim();
  if (!raw) {
    return null;
  }
  let m = /^bv(\d+)$/i.exec(raw);
  if (m) {
    return Number(m[1]);
  }
  m = /^av(\d+)$/i.exec(raw);
  if (m) {
    return Number(m[1]);
  }
  if (/^\d+$/.test(raw)) {
    return Number(raw);
  }
  return null;
}

/** 写入 vue-router params.aid */
export function videoPlayRouteAid(id) {
  return formatVideoBvid(id) || String(id ?? "");
}
