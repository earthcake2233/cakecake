import { clearTokens, clearMinibiliPostLoginRedirect } from "./authTokens";

let sessionInvalidateHandler = null;

/** 在 main 中注册：同步 Vuex（signIn / proInfo 等），避免 http 直接依赖 store 产生循环引用 */
export function registerMinibiliSessionInvalidate(handler) {
  sessionInvalidateHandler = typeof handler === "function" ? handler : null;
}

let lastInvalidateAt = 0;

/** Token 失效或鉴权失败：清本地 JWT、主站 signIn 缓存，并通知 store 回到未登录顶栏 */
export function invalidateMinibiliSessionFromHttp() {
  const now = Date.now();
  if (now - lastInvalidateAt < 800) {
    return;
  }
  lastInvalidateAt = now;
  clearTokens();
  clearMinibiliPostLoginRedirect();
  try {
    localStorage.setItem("signIn", "0");
  } catch {
    /* ignore private mode */
  }
  if (sessionInvalidateHandler) {
    try {
      sessionInvalidateHandler();
    } catch {
      /* ignore */
    }
  }
}
