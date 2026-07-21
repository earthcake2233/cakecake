import adminHttp from "@/utils/adminHttp";
import http from "@/utils/http";

const isMinibili =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

export function adminLogin(username, password) {
  return adminHttp.post(
    "/api/v1/admin/auth/login",
    { username, password },
    { skipGlobalErrorToast: true }
  );
}

export function adminMe() {
  return adminHttp.get("/api/v1/admin/me");
}

export function adminListBanners() {
  return adminHttp.get("/api/v1/admin/home-banners");
}

export function adminCreateBanner(payload) {
  return adminHttp.post("/api/v1/admin/home-banners", payload);
}

export function adminUpdateBanner(id, payload) {
  return adminHttp.put(`/api/v1/admin/home-banners/${id}`, payload);
}

export function adminDeleteBanner(id) {
  return adminHttp.delete(`/api/v1/admin/home-banners/${id}`);
}

/** 轮播图上传 OSS；新建用 upload-image，编辑已有轮播可传 bannerId 直接写库 */
export function adminUploadBannerImage(file, bannerId) {
  const fd = new FormData();
  fd.append("image", file);
  const opts = { timeout: 120000, skipGlobalErrorToast: false };
  if (bannerId) {
    return adminHttp.post(`/api/v1/admin/home-banners/${bannerId}/image`, fd, opts);
  }
  return adminHttp.post("/api/v1/admin/home-banners/upload-image", fd, opts);
}

export function adminListHotSearchOps() {
  return adminHttp.get("/api/v1/admin/hot-search/ops");
}

export function adminCreateHotSearchOp(payload) {
  return adminHttp.post("/api/v1/admin/hot-search/ops", payload);
}

export function adminUpdateHotSearchOp(id, payload) {
  return adminHttp.put(`/api/v1/admin/hot-search/ops/${id}`, payload);
}

export function adminDeleteHotSearchOp(id) {
  return adminHttp.delete(`/api/v1/admin/hot-search/ops/${id}`);
}

export function adminPreviewHotSearch(limit = 10) {
  return adminHttp.get("/api/v1/admin/hot-search/preview", {
    params: { limit }
  });
}

export function adminHotSearchDashboard(limit = 10, redisLimit = 30) {
  return adminHttp.get("/api/v1/admin/hot-search/dashboard", {
    params: { limit, redis_limit: redisLimit }
  });
}

export function adminRemoveHotSearchRedis(keyword) {
  return adminHttp.post("/api/v1/admin/hot-search/redis/remove", { keyword });
}

export function adminBoostHotSearchRedis(keyword, delta = 5) {
  return adminHttp.post("/api/v1/admin/hot-search/redis/boost", { keyword, delta });
}

export function adminQuickHotSearchOp(payload) {
  return adminHttp.post("/api/v1/admin/hot-search/quick-op", payload);
}

export function adminReorderHotSearch(items) {
  return adminHttp.post("/api/v1/admin/hot-search/reorder", { items });
}

export function adminResetHotSearchDisplayOrder() {
  return adminHttp.post("/api/v1/admin/hot-search/display-order/reset");
}

export function adminListVideos(params = {}) {
  return adminHttp.get("/api/v1/admin/videos", { params });
}

export function adminGetVideo(id) {
  return adminHttp.get(`/api/v1/admin/videos/${id}`);
}

export function adminApproveVideo(id) {
  return adminHttp.post(`/api/v1/admin/videos/${id}/approve`);
}

export function adminRejectVideo(id, reason) {
  return adminHttp.post(`/api/v1/admin/videos/${id}/reject`, { reason });
}

export function adminDeleteVideo(id) {
  return adminHttp.post(`/api/v1/admin/videos/${id}/delete`);
}

export function adminListArticles(params = {}) {
  return adminHttp.get("/api/v1/admin/articles", { params });
}

export function adminGetArticle(id) {
  return adminHttp.get(`/api/v1/admin/articles/${id}`);
}

export function adminApproveArticle(id) {
  return adminHttp.post(`/api/v1/admin/articles/${id}/approve`);
}

export function adminRejectArticle(id, reason) {
  return adminHttp.post(`/api/v1/admin/articles/${id}/reject`, { reason });
}

export function adminDeleteArticle(id) {
  return adminHttp.post(`/api/v1/admin/articles/${id}/delete`);
}

export function adminListDynamics(params = {}) {
  return adminHttp.get("/api/v1/admin/dynamics", { params });
}

export function adminGetDynamic(id) {
  return adminHttp.get(`/api/v1/admin/dynamics/${id}`);
}

export function adminDeleteDynamic(id) {
  return adminHttp.post(`/api/v1/admin/dynamics/${id}/delete`);
}

export function adminGetAgentSettings() {
  return adminHttp.get("/api/v1/admin/agent-settings");
}

export function adminPutAgentSettings(payload) {
  return adminHttp.put("/api/v1/admin/agent-settings", payload);
}

export function adminUploadAgentAvatar(file) {
  const fd = new FormData();
  fd.append("image", file);
  return adminHttp.post("/api/v1/admin/agent-settings/avatar", fd);
}

export function adminListAgentProfiles() {
  return adminHttp.get("/api/v1/admin/agent-profiles");
}

export function adminCreateAgentProfile(payload) {
  return adminHttp.post("/api/v1/admin/agent-profiles", payload);
}

export function adminUpdateAgentProfile(id, payload) {
  return adminHttp.put(`/api/v1/admin/agent-profiles/${id}`, payload);
}

export function adminDeleteAgentProfile(id) {
  return adminHttp.delete(`/api/v1/admin/agent-profiles/${id}`);
}

export function adminUploadAgentProfileAvatar(id, file) {
  const fd = new FormData();
  fd.append("image", file);
  return adminHttp.post(`/api/v1/admin/agent-profiles/${id}/avatar`, fd);
}

/** 主站首页轮播（公开接口） */
export function adminListSystemConfigs() {
  return adminHttp.get("/api/v1/admin/system-configs");
}

export function adminUpdateSystemConfigs(configs) {
  return adminHttp.put("/api/v1/admin/system-configs", { configs });
}

export function getHomeBannersPublic() {
  if (!isMinibili) {
    return Promise.resolve({ code: 0, data: { items: [] } });
  }
  return http.get("/api/v1/home-banners", { skipGlobalErrorToast: true });
}
