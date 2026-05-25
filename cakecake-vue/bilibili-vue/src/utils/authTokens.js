/** Mini-Bili JWT 本地存储（与主站 Vuex 登录弹窗独立） */
const K_ACCESS = "minibili_access_token";
const K_REFRESH = "minibili_refresh_token";
const K_UID = "minibili_user_id";
const K_DISPLAY = "minibili_display_name";
const K_POST_LOGIN = "minibili_post_login_redirect";

/** 登录成功后要跳转的路径（仅 path，如 /upload、/minibili/messages） */
export function setMinibiliPostLoginRedirect(path) {
  const p = String(path || "").trim();
  if (p.startsWith("/")) {
    sessionStorage.setItem(K_POST_LOGIN, p);
  }
}

export function consumeMinibiliPostLoginRedirect() {
  const p = sessionStorage.getItem(K_POST_LOGIN);
  sessionStorage.removeItem(K_POST_LOGIN);
  return p && p.startsWith("/") ? p : null;
}

export function clearMinibiliPostLoginRedirect() {
  sessionStorage.removeItem(K_POST_LOGIN);
}

function decodeJwtUserId(access) {
  try {
    const body = access.split(".")[1];
    const b64 = body.replace(/-/g, "+").replace(/_/g, "/");
    const pad = b64.length % 4 ? "=".repeat(4 - (b64.length % 4)) : "";
    const json = atob(b64 + pad);
    const p = JSON.parse(json);
    return p.user_id != null ? Number(p.user_id) : null;
  } catch {
    return null;
  }
}

export function getAccessToken() {
  return localStorage.getItem(K_ACCESS) || "";
}

export function getRefreshToken() {
  return localStorage.getItem(K_REFRESH) || "";
}

export function getUserId() {
  const s = localStorage.getItem(K_UID);
  return s ? Number(s) : null;
}

/** 登录表单用户名，用于顶栏展示（JWT 内无 username） */
export function getMinibiliDisplayName() {
  return localStorage.getItem(K_DISPLAY) || "";
}

export function setMinibiliDisplayName(name) {
  if (name) {
    localStorage.setItem(K_DISPLAY, String(name).trim());
  } else {
    localStorage.removeItem(K_DISPLAY);
  }
}

export function setTokens(access, refresh) {
  if (access) {
    localStorage.setItem(K_ACCESS, access);
    const uid = decodeJwtUserId(access);
    if (uid != null) localStorage.setItem(K_UID, String(uid));
  }
  if (refresh) localStorage.setItem(K_REFRESH, refresh);
}

export function clearTokens() {
  localStorage.removeItem(K_ACCESS);
  localStorage.removeItem(K_REFRESH);
  localStorage.removeItem(K_UID);
  localStorage.removeItem(K_DISPLAY);
}

export function isLoggedIn() {
  return !!getAccessToken();
}
