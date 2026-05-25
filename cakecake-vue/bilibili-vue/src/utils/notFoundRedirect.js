import { parseVideoIdFromRoute } from "./videoBvid";

/** 播放页 aid 是否无法解析为有效视频 id（如 BV7wewqewqe、BV0） */
export function isInvalidVideoRouteAid(aid) {
  const raw = String(aid ?? "").trim();
  if (!raw) {
    return true;
  }
  const id = parseVideoIdFromRoute(raw);
  return id == null || id <= 0;
}

/** 当前路由是否为无效播放页，应跳转 404 */
export function shouldRedirectVideoToNotFound(route) {
  return route?.name === "video" && isInvalidVideoRouteAid(route.params?.aid);
}
