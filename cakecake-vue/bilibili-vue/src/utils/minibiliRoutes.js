import { videoPlayRouteAid } from "./videoBvid";

/** 仅保留顶栏 nav-menu，隐藏头图搜索与分区导航（历史/稍后再看等与首页一致展示完整顶栏） */
export const MINIBILI_COMPACT_HEADER_ROUTES = new Set([
  "notFound",
  "minibiliPersonalCenter",
  "minibiliMessages",
  "minibiliDynamics",
  "minibiliArticleRead",
  "minibiliDynamicRead",
  "minibiliUserSpace",
  "minibiliUserSpaceRelations"
]);

export function isMinibiliCompactHeaderRoute(routeName) {
  return MINIBILI_COMPACT_HEADER_ROUTES.has(routeName);
}

/** 个人空间 / 消息中心等：仅顶栏 nav-menu，不展示首页头图与分区导航 */
export function shouldShowMinibiliCompactHeader(route) {
  if (!route || !route.name) {
    return false;
  }
  return MINIBILI_COMPACT_HEADER_ROUTES.has(route.name);
}

/** 搜索页：仅顶栏，不展示首页头图与分区导航 */
export function isSearchLayoutRoute(route) {
  if (!route || !Array.isArray(route.matched)) {
    return false;
  }
  return route.matched.some(r => r.path === "/search");
}

/** 首页头图、分区栏、顶栏毛玻璃是否展示（随路由变化，避免后退后 Vuex menuShow 残留） */
export function shouldShowHomeHeaderChrome(route) {
  return (
    !shouldShowMinibiliCompactHeader(route) && !isSearchLayoutRoute(route)
  );
}

/**
 * Mini-Bili 前端路由片段（个人空间等），供顶栏 / 播放页 / 评论等复用。
 * @param {string|number|null|undefined} userId
 * @returns {{ name: string, params: { userId: string } }|null}
 */
export function minibiliUserSpaceRoute(userId) {
  const raw = userId != null ? String(userId).trim() : "";
  const n = parseInt(raw, 10);
  if (!Number.isFinite(n) || n <= 0) {
    return null;
  }
  return { name: "minibiliUserSpace", params: { userId: String(n) } };
}

/** 稍后再看独立页 */
export function minibiliWatchLaterRoute() {
  return { name: "minibiliWatchLater" };
}

/** 顶栏「动态」→ 独立关注流页 /minibili/dynamics */
export function minibiliDynamicsRoute() {
  return { name: "minibiliDynamics" };
}

/** 个人空间内「动态」Tab */
export function minibiliUserSpaceDynamicRoute(userId) {
  const base = minibiliUserSpaceRoute(userId);
  if (!base) {
    return null;
  }
  return { ...base, query: { nav: "dynamic" } };
}

/** 个人中心（可选 tab：home / info / coin / record 等） */
export function minibiliPersonalCenterRoute(tab) {
  const t = tab != null ? String(tab).trim() : "";
  if (!t) {
    return { name: "minibiliPersonalCenter" };
  }
  return { name: "minibiliPersonalCenter", query: { tab: t } };
}

/** 观看历史独立页（顶栏「历史」） */
export function minibiliViewHistoryRoute() {
  return { name: "minibiliViewHistory" };
}

/** 播放页路由（params.aid 为 BV{id}） */
export function minibiliVideoPlayRoute(videoId) {
  const aid = videoPlayRouteAid(videoId);
  if (!aid) {
    return null;
  }
  return { name: "video", params: { aid } };
}

/** 专栏阅读页 */
export function minibiliArticleReadRoute(articleId) {
  const n = parseInt(String(articleId ?? ""), 10);
  if (!Number.isFinite(n) || n <= 0) {
    return null;
  }
  return { name: "minibiliArticleRead", params: { id: String(n) } };
}

/** 动态图文阅读页（复用 ArticleRead 布局）；query 如 { edit: "1" } 打开编辑弹窗 */
export function minibiliDynamicReadRoute(dynamicId, query) {
  const n = parseInt(String(dynamicId ?? ""), 10);
  if (!Number.isFinite(n) || n <= 0) {
    return null;
  }
  const q =
    query && typeof query === "object" && !Array.isArray(query) ? query : {};
  return { name: "minibiliDynamicRead", params: { id: String(n) }, query: q };
}

/**
 * 个人空间 · 收藏 tab（顶栏「收藏夹」等）
 * @param {string|number|null|undefined} userId
 * @returns {{ name: string, params: { userId: string }, query: { nav: string } }|null}
 */
export function minibiliUserSpaceCollectRoute(userId) {
  const base = minibiliUserSpaceRoute(userId);
  if (!base) {
    return null;
  }
  return { ...base, query: { nav: "collect" } };
}

/** 个人空间 · 投稿 tab · 视频子栏 */
export function minibiliUserSpaceContributeVideoRoute(userId) {
  const base = minibiliUserSpaceRoute(userId);
  if (!base) {
    return null;
  }
  return { ...base, query: { nav: "contribute", side: "video" } };
}

/**
 * 个人空间 · 关注/粉丝列表
 * @param {string|number|null|undefined} userId
 * @param {"following"|"followers"} tab
 */
export function minibiliUserSpaceRelationsRoute(
  userId,
  tab = "following",
  extraQuery = {}
) {
  const raw = userId != null ? String(userId).trim() : "";
  const n = parseInt(raw, 10);
  if (!Number.isFinite(n) || n <= 0) {
    return null;
  }
  const t = tab === "followers" ? "followers" : "following";
  const query = { tab: t, ...(extraQuery && typeof extraQuery === "object" ? extraQuery : {}) };
  return {
    name: "minibiliUserSpaceRelations",
    params: { userId: String(n) },
    query
  };
}
