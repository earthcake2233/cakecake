/**
 * 个人空间「视角预览」：将 UP 主完整资料裁剪为访客/粉丝可见字段
 * @param {Record<string, unknown> | null | undefined} src
 * @param {'fan' | 'visitor'} perspective
 * @returns {Record<string, unknown> | null}
 */
export function buildSpaceViewerProfile(src, perspective) {
  if (!src || typeof src !== "object") {
    return null;
  }
  const priv =
    src.privacy && typeof src.privacy === "object" && !Array.isArray(src.privacy)
      ? src.privacy
      : {};
  const publicBirthday = priv.public_birthday !== false;
  const out = {
    ...src,
    is_owner: false,
    followed_by_me: perspective === "fan",
    birthday: publicBirthday ? String(src.birthday || "").trim() : "",
    privacy: {
      public_favorites: !!priv.public_favorites,
      public_recent_coins: !!priv.public_recent_coins,
      public_following: !!priv.public_following,
      public_fans: !!priv.public_fans,
      public_birthday: publicBirthday
    }
  };
  return out;
}

/**
 * @param {string} mode
 * @returns {mode is 'fan' | 'visitor'}
 */
export function isSpacePerspectivePreviewMode(mode) {
  return mode === "fan" || mode === "visitor";
}

export function spacePerspectiveStorageKey(userId) {
  const n = Number(userId);
  return Number.isFinite(n) && n > 0
    ? `minibili_space_perspective_${n}`
    : "";
}

/**
 * @param {number} userId
 * @returns {'self' | 'fan' | 'visitor'}
 */
export function readStoredSpacePerspective(userId) {
  const key = spacePerspectiveStorageKey(userId);
  if (!key) {
    return "self";
  }
  try {
    const p = String(sessionStorage.getItem(key) || "").trim();
    return isSpacePerspectivePreviewMode(p) ? p : "self";
  } catch {
    return "self";
  }
}

/**
 * @param {number} userId
 * @param {'self' | 'fan' | 'visitor'} mode
 */
export function writeStoredSpacePerspective(userId, mode) {
  const key = spacePerspectiveStorageKey(userId);
  if (!key) {
    return;
  }
  try {
    if (!mode || mode === "self") {
      sessionStorage.removeItem(key);
      return;
    }
    if (isSpacePerspectivePreviewMode(mode)) {
      sessionStorage.setItem(key, mode);
    }
  } catch {
    /* ignore */
  }
}

/**
 * 解析当前生效的视角：路由 query 优先，其次 session（仅本人空间）
 * @param {number} userId
 * @param {string|undefined} queryPerspective
 * @param {boolean} isRealSpaceOwner
 * @returns {'self' | 'fan' | 'visitor'}
 */
export function resolveSpacePerspective(userId, queryPerspective, isRealSpaceOwner) {
  const q = String(queryPerspective || "").trim();
  if (isSpacePerspectivePreviewMode(q)) {
    return q;
  }
  if (isRealSpaceOwner) {
    return readStoredSpacePerspective(userId);
  }
  return "self";
}
