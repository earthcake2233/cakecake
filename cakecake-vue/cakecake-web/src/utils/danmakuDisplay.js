/** 弹幕字号：与播放器发送面板、后端 font_size 字段一致 */
export const DM_FONT_SIZE_PX = {
  sm: 16,
  md: 18,
  lg: 24
};

export const DM_SEND_FONT_PREF_KEY = "mb_dm_send_font_size";

/** 行高（顶/底堆叠、滚动轨道间距） */
export function dmLineHeightForFontPx(px) {
  const n = Number(px) || DM_FONT_SIZE_PX.md;
  return Math.max(20, Math.round(n * 1.44));
}

export function dmNormalizeFontSizeKey(raw) {
  const k = String(raw || "md")
    .trim()
    .toLowerCase();
  if (k === "sm" || k === "small") return "sm";
  if (k === "lg" || k === "large") return "lg";
  return "md";
}

export function dmFontPxFromKey(key) {
  const k = dmNormalizeFontSizeKey(key);
  return DM_FONT_SIZE_PX[k] || DM_FONT_SIZE_PX.md;
}

export function dmCanvasFontCss(pxOrKey) {
  const size =
    typeof pxOrKey === "number" && pxOrKey > 0
      ? pxOrKey
      : dmFontPxFromKey(pxOrKey);
  return `bold ${size}px system-ui, 'Segoe UI', 'Microsoft YaHei', sans-serif`;
}

export function dmStrokeWidthForPx(px) {
  return Math.max(2, Math.round((Number(px) || 18) * 0.17));
}

export function loadDmSendFontSizePref() {
  try {
    const v = localStorage.getItem(DM_SEND_FONT_PREF_KEY);
    if (v === "sm" || v === "md" || v === "lg") return v;
  } catch {
    /* noop */
  }
  return "md";
}

export function saveDmSendFontSizePref(key) {
  try {
    localStorage.setItem(DM_SEND_FONT_PREF_KEY, dmNormalizeFontSizeKey(key));
  } catch {
    /* noop */
  }
}

/** 弹幕密度滑条 0–100 → 同屏容量系数（约 0.15–1） */
export function dmDensityFactor(densityRaw) {
  const n = Math.min(100, Math.max(0, Number(densityRaw)));
  if (!Number.isFinite(n)) {
    return 0.9;
  }
  return 0.15 + (n / 100) * 0.85;
}

/** 在显示区域算出的基础轨道数上按密度缩放 */
export function dmScrollLanesForDensity(baseLanes, densityRaw) {
  const base = Math.max(1, Math.floor(Number(baseLanes) || 1));
  const lanes = Math.round(base * dmDensityFactor(densityRaw));
  return Math.max(1, lanes);
}

/** 顶/底固定弹幕最大行数按密度缩放 */
export function dmStackSlotsForDensity(baseSlots, densityRaw) {
  const base = Math.max(1, Math.floor(Number(baseSlots) || 1));
  const slots = Math.round(base * dmDensityFactor(densityRaw));
  return Math.max(1, slots);
}

/**
 * 同屏弹幕总量上限（滚动为主；密度越低越少）。
 * @param {number} densityRaw 0–100
 * @param {number} canvasH 画布 CSS 高度
 * @param {number} scrollLanes 当前滚动轨道数
 */
export function dmMaxActiveBullets(densityRaw, canvasH, scrollLanes) {
  const lanes = Math.max(1, Math.floor(Number(scrollLanes) || 1));
  const h = Math.max(80, Number(canvasH) || 360);
  const f = dmDensityFactor(densityRaw);
  const byLanes = Math.round(lanes * 2.2 + 4);
  const byHeight = Math.round((h / 28) * f * 1.2);
  return Math.max(4, Math.min(100, Math.max(byLanes, byHeight)));
}
