import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";

const mockAdminHttp = { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() };
vi.mock("@/utils/adminHttp", () => ({ default: mockAdminHttp }));

const mockHttp = { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() };
vi.mock("@/utils/http", () => ({ default: mockHttp }));

let adminApi;
beforeAll(async () => {
  vi.stubEnv("VITE_MINIBILI_API", "true");
  vi.stubGlobal("console", { warn: vi.fn(), log: vi.fn(), error: vi.fn() });
  adminApi = await import("@/api/admin");
});
beforeEach(() => { vi.clearAllMocks(); });

describe("api/admin.js", () => {
  // --- Auth ---
  describe("adminLogin", () => {
    it("posts to /api/v1/admin/auth/login", async () => {
      mockAdminHttp.post.mockResolvedValue({ code: 0, data: { token: "abc" } });
      const r = await adminApi.adminLogin("admin", "pass");
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/auth/login", { username: "admin", password: "pass" }, { skipGlobalErrorToast: true });
      expect(r.data.token).toBe("abc");
    });
  });
  describe("adminMe", () => {
    it("gets /api/v1/admin/me", async () => {
      mockAdminHttp.get.mockResolvedValue({ code: 0, data: { username: "admin" } });
      const r = await adminApi.adminMe();
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/me");
      expect(r.data.username).toBe("admin");
    });
  });
  // --- Banners ---
  describe("banner CRUD", () => {
    it("adminListBanners", async () => {
      mockAdminHttp.get.mockResolvedValue({ code: 0, data: { items: [{ id: 1 }] } });
      const r = await adminApi.adminListBanners();
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/home-banners");
      expect(r.data.items[0].id).toBe(1);
    });
    it("adminCreateBanner", async () => {
      mockAdminHttp.post.mockResolvedValue({ code: 0, data: { id: 2 } });
      const r = await adminApi.adminCreateBanner({ title: "new banner" });
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/home-banners", { title: "new banner" });
      expect(r.data.id).toBe(2);
    });
    it("adminUpdateBanner", async () => {
      mockAdminHttp.put.mockResolvedValue({ code: 0 });
      await adminApi.adminUpdateBanner(1, { title: "updated" });
      expect(mockAdminHttp.put).toHaveBeenCalledWith("/api/v1/admin/home-banners/1", { title: "updated" });
    });
    it("adminDeleteBanner", async () => {
      mockAdminHttp.delete.mockResolvedValue({ code: 0 });
      await adminApi.adminDeleteBanner(5);
      expect(mockAdminHttp.delete).toHaveBeenCalledWith("/api/v1/admin/home-banners/5");
    });
  });
  describe("adminUploadBannerImage", () => {
    it("with bannerId", async () => {
      const file = new File(["test"], "test.png", { type: "image/png" });
      mockAdminHttp.post.mockResolvedValue({ code: 0, data: { url: "img.jpg" } });
      const r = await adminApi.adminUploadBannerImage(file, 3);
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/home-banners/3/image", expect.any(FormData), { timeout: 120000, skipGlobalErrorToast: false });
      expect(r.data.url).toBe("img.jpg");
    });
    it("without bannerId", async () => {
      const file = new File(["test"], "test.png", { type: "image/png" });
      mockAdminHttp.post.mockResolvedValue({ code: 0 });
      await adminApi.adminUploadBannerImage(file);
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/home-banners/upload-image", expect.any(FormData), { timeout: 120000, skipGlobalErrorToast: false });
    });
  });
  // --- Hot Search ---
  describe("hot search ops", () => {
    it("adminListHotSearchOps", async () => {
      mockAdminHttp.get.mockResolvedValue({ code: 0, data: { items: [{ keyword: "test" }] } });
      const r = await adminApi.adminListHotSearchOps();
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/hot-search/ops");
      expect(r.data.items[0].keyword).toBe("test");
    });
    it("adminCreateHotSearchOp", async () => {
      await adminApi.adminCreateHotSearchOp({ keyword: "new" });
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/hot-search/ops", { keyword: "new" });
    });
    it("adminUpdateHotSearchOp", async () => {
      await adminApi.adminUpdateHotSearchOp(1, { keyword: "updated" });
      expect(mockAdminHttp.put).toHaveBeenCalledWith("/api/v1/admin/hot-search/ops/1", { keyword: "updated" });
    });
    it("adminDeleteHotSearchOp", async () => {
      await adminApi.adminDeleteHotSearchOp(2);
      expect(mockAdminHttp.delete).toHaveBeenCalledWith("/api/v1/admin/hot-search/ops/2");
    });
    it("adminPreviewHotSearch", async () => {
      await adminApi.adminPreviewHotSearch(10);
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/hot-search/preview", { params: { limit: 10 } });
    });
    it("adminHotSearchDashboard", async () => {
      await adminApi.adminHotSearchDashboard(10, 30);
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/hot-search/dashboard", { params: { limit: 10, redis_limit: 30 } });
    });
    it("adminRemoveHotSearchRedis", async () => {
      await adminApi.adminRemoveHotSearchRedis("badword");
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/hot-search/redis/remove", { keyword: "badword" });
    });
    it("adminBoostHotSearchRedis", async () => {
      await adminApi.adminBoostHotSearchRedis("hotword", 5);
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/hot-search/redis/boost", { keyword: "hotword", delta: 5 });
    });
    it("adminQuickHotSearchOp", async () => {
      await adminApi.adminQuickHotSearchOp({ keyword: "quick" });
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/hot-search/quick-op", { keyword: "quick" });
    });
    it("adminReorderHotSearch", async () => {
      await adminApi.adminReorderHotSearch([{ id: 1 }, { id: 2 }]);
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/hot-search/reorder", { items: [{ id: 1 }, { id: 2 }] });
    });
    it("adminResetHotSearchDisplayOrder", async () => {
      await adminApi.adminResetHotSearchDisplayOrder();
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/hot-search/display-order/reset");
    });
  });
  // --- Videos ---
  describe("video review", () => {
    it("adminListVideos", async () => {
      await adminApi.adminListVideos({ status: "pending" });
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/videos", { params: { status: "pending" } });
    });
    it("adminGetVideo", async () => {
      mockAdminHttp.get.mockResolvedValue({ code: 0, data: { id: 5 } });
      const r = await adminApi.adminGetVideo(5);
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/videos/5");
      expect(r.data.id).toBe(5);
    });
    it("adminApproveVideo", async () => {
      await adminApi.adminApproveVideo(1);
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/videos/1/approve");
    });
    it("adminRejectVideo", async () => {
      await adminApi.adminRejectVideo(1, "spam");
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/videos/1/reject", { reason: "spam" });
    });
    it("adminDeleteVideo", async () => {
      await adminApi.adminDeleteVideo(1);
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/videos/1/delete");
    });
  });
  // --- Articles ---
  describe("article review", () => {
    it("adminListArticles", async () => {
      await adminApi.adminListArticles({ status: "pending" });
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/articles", { params: { status: "pending" } });
    });
    it("adminGetArticle", async () => {
      mockAdminHttp.get.mockResolvedValue({ code: 0, data: { id: 3 } });
      const r = await adminApi.adminGetArticle(3);
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/articles/3");
      expect(r.data.id).toBe(3);
    });
    it("adminApproveArticle", async () => {
      await adminApi.adminApproveArticle(1);
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/articles/1/approve");
    });
    it("adminRejectArticle", async () => {
      await adminApi.adminRejectArticle(1, "inappropriate");
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/articles/1/reject", { reason: "inappropriate" });
    });
    it("adminDeleteArticle", async () => {
      await adminApi.adminDeleteArticle(1);
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/articles/1/delete");
    });
  });
  // --- Dynamics ---
  describe("dynamics", () => {
    it("adminListDynamics", async () => {
      await adminApi.adminListDynamics({ page: 1 });
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/dynamics", { params: { page: 1 } });
    });
    it("adminGetDynamic", async () => {
      mockAdminHttp.get.mockResolvedValue({ code: 0, data: { id: 7 } });
      const r = await adminApi.adminGetDynamic(7);
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/dynamics/7");
      expect(r.data.id).toBe(7);
    });
    it("adminDeleteDynamic", async () => {
      await adminApi.adminDeleteDynamic(7);
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/dynamics/7/delete");
    });
  });
  // --- Agent ---
  describe("agent settings", () => {
    it("adminGetAgentSettings", async () => {
      mockAdminHttp.get.mockResolvedValue({ code: 0, data: { name: "agent" } });
      const r = await adminApi.adminGetAgentSettings();
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/agent-settings");
      expect(r.data.name).toBe("agent");
    });
    it("adminPutAgentSettings", async () => {
      await adminApi.adminPutAgentSettings({ name: "new agent" });
      expect(mockAdminHttp.put).toHaveBeenCalledWith("/api/v1/admin/agent-settings", { name: "new agent" });
    });
    it("adminUploadAgentAvatar", async () => {
      const file = new File(["data"], "avatar.png", { type: "image/png" });
      await adminApi.adminUploadAgentAvatar(file);
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/agent-settings/avatar", expect.any(FormData));
    });
  });
  // --- Agent Profiles ---
  describe("agent profiles", () => {
    it("adminListAgentProfiles", async () => {
      await adminApi.adminListAgentProfiles();
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/agent-profiles");
    });
    it("adminCreateAgentProfile", async () => {
      await adminApi.adminCreateAgentProfile({ name: "profile1" });
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/agent-profiles", { name: "profile1" });
    });
    it("adminUpdateAgentProfile", async () => {
      await adminApi.adminUpdateAgentProfile(1, { name: "updated" });
      expect(mockAdminHttp.put).toHaveBeenCalledWith("/api/v1/admin/agent-profiles/1", { name: "updated" });
    });
    it("adminDeleteAgentProfile", async () => {
      await adminApi.adminDeleteAgentProfile(2);
      expect(mockAdminHttp.delete).toHaveBeenCalledWith("/api/v1/admin/agent-profiles/2");
    });
    it("adminUploadAgentProfileAvatar", async () => {
      const file = new File(["data"], "prof.png", { type: "image/png" });
      await adminApi.adminUploadAgentProfileAvatar(1, file);
      expect(mockAdminHttp.post).toHaveBeenCalledWith("/api/v1/admin/agent-profiles/1/avatar", expect.any(FormData));
    });
  });
  // --- System Configs ---
  describe("system configs", () => {
    it("adminListSystemConfigs", async () => {
      await adminApi.adminListSystemConfigs();
      expect(mockAdminHttp.get).toHaveBeenCalledWith("/api/v1/admin/system-configs");
    });
    it("adminUpdateSystemConfigs", async () => {
      await adminApi.adminUpdateSystemConfigs({ key: "value" });
      expect(mockAdminHttp.put).toHaveBeenCalledWith("/api/v1/admin/system-configs", { configs: { key: "value" } });
    });
  });
  // --- Public Banners ---
  describe("getHomeBannersPublic", () => {
    it("calls http.get when VITE_MINIBILI_API=true", async () => {
      mockHttp.get.mockResolvedValue({ code: 0, data: { items: [{ id: 1 }] } });
      const r = await adminApi.getHomeBannersPublic();
      expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/home-banners", { skipGlobalErrorToast: true });
      expect(r.data.items[0].id).toBe(1);
    });
  });
});
