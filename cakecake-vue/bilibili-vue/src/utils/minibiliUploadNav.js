import { getAccessToken } from "./authTokens";

export function isMinibiliApiEnv() {
  return (
    import.meta.env.VITE_MINIBILI_API === "true" ||
    import.meta.env.VITE_MINIBILI_API === "1"
  );
}

/**
 * Mini-Bili 未登录：顶栏/侧栏「投稿」不跳独立登录页，改为打开主站登录弹窗（见各组件 @click）。
 * 在组件 computed 中务必读取 `this.$route.fullPath`，否则仅改 localStorage 时状态不刷新。
 */
export function minibiliUploadOpensLoginModal() {
  return isMinibiliApiEnv() && !getAccessToken();
}

/**
 * 主站顶栏 / 创作中心侧栏「投稿」的 :to（仅当已登录 Mini-Bili 或非 Mini 模式时使用）。
 * 已登录 Mini-Bili 仍走主站创作中心路由（CreatorShell），与 bilibili-vue README 一致。
 */
export function resolveMinibiliUploadNavTo() {
  if (!isMinibiliApiEnv()) {
    return { name: "upload" };
  }
  if (getAccessToken()) {
    return { name: "upload" };
  }
  return { name: "home" };
}
