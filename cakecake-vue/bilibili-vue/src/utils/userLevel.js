/** Cumulative EXP thresholds to reach each level (Lv1–Lv6). */
export const USER_LEVEL_THRESHOLDS = [0, 20, 150, 450, 1080, 2880];

export const USER_LEVEL_MAX = 6;

/** Bilibili 会员等级说明（顶栏气泡「会员等级相关说明」） */
export const USER_LEVEL_HELP_URL =
  "https://www.bilibili.com/blackboard/help.html#/?qid=295";

/**
 * 顶栏等级气泡说明（Lv1–Lv6 统一文案，仅展示；不绑定真实权限校验）。
 * 与 B 站「作为 LVx，你可以」样式一致。
 */
export const USER_LEVEL_PRIVILEGE_LINES = [
  "购买邀请码（2个/月）",
  "发射个性弹幕（彩色、高级、顶部、底部）",
  "参与视频互动（评论、添加tag）",
  "投稿成为偶像"
];

/**
 * @param {number} exp Total experience
 * @returns {{ current_level: number, current_min: number, current_exp: number, next_exp: number }}
 */
export function levelInfoFromExperience(exp) {
  const total = Math.max(0, Math.floor(Number(exp) || 0));
  let lv = 1;
  for (let i = USER_LEVEL_THRESHOLDS.length - 1; i >= 0; i--) {
    if (total >= USER_LEVEL_THRESHOLDS[i]) {
      lv = i + 1;
      break;
    }
  }
  const currentMin = USER_LEVEL_THRESHOLDS[lv - 1];
  const nextExp =
    lv >= USER_LEVEL_MAX
      ? USER_LEVEL_THRESHOLDS[USER_LEVEL_MAX - 1]
      : USER_LEVEL_THRESHOLDS[lv];
  return {
    current_level: lv,
    current_min: currentMin,
    current_exp: total,
    next_exp: nextExp
  };
}

/** Progress within current level, 0–100. */
export function levelFillPct(levelInfo) {
  if (!levelInfo || typeof levelInfo !== "object") {
    return 0;
  }
  const min = Number(levelInfo.current_min) || 0;
  const cur = Number(levelInfo.current_exp) || 0;
  const next = Number(levelInfo.next_exp) || 0;
  const lv = Number(levelInfo.current_level) || 1;
  if (lv >= USER_LEVEL_MAX) {
    return 100;
  }
  if (next <= min) {
    return 100;
  }
  return Math.min(100, Math.round(((cur - min) / (next - min)) * 1000) / 10);
}

/** 等级气泡权益列表（各等级相同，level 参数保留供调用方统一接口） */
export function levelPrivilegeLines(_level) {
  void _level;
  return USER_LEVEL_PRIVILEGE_LINES;
}

/** Clamp account level to Lv1–Lv6 for display. */
export function clampUserLevel(lv) {
  const n = Math.floor(Number(lv) || 1);
  if (!Number.isFinite(n) || n < 1) {
    return 1;
  }
  return Math.min(USER_LEVEL_MAX, n);
}

/** `public/user-profile/level_0～level_6.svg` badge URL. */
export function levelIconUrl(lv) {
  const n = Math.min(Math.max(Math.floor(Number(lv) || 0), 0), USER_LEVEL_MAX);
  const base = import.meta.env.BASE_URL;
  return `${base}user-profile/level_${n}.svg`;
}

/**
 * Comment / user row account level (not comment thread depth `level`).
 * @param {{ user_level?: number, level_info?: { current_level?: number } } | null | undefined} row
 */
export function commentUserLevel(row) {
  if (!row || typeof row !== "object") {
    return 1;
  }
  if (row.user_level != null) {
    return clampUserLevel(row.user_level);
  }
  const li = row.level_info;
  if (li && li.current_level != null) {
    return clampUserLevel(li.current_level);
  }
  return 1;
}
