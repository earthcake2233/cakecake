import { describe, it, expect, vi, beforeEach } from "vitest";
import { toggleWatchLaterVideo, openLoginModal } from "@/utils/watchLaterAction";

vi.mock("@/api/minibili", () => ({
  mbToggleWatchLater: vi.fn()
}));

vi.mock("@/utils/authTokens", () => ({
  getAccessToken: vi.fn()
}));

vi.mock("element-plus", () => ({
  ElMessage: {
    success: vi.fn(),
    error: vi.fn()
  }
}));

describe("watchLaterAction", () => {
  let mockStore;
  beforeEach(() => { mockStore = { commit: vi.fn() }; vi.clearAllMocks(); });

  describe("openLoginModal", () => {
    it("commits login tab and opens modal", () => {
      openLoginModal(mockStore);
      expect(mockStore.commit).toHaveBeenCalledWith("login/SET_LOGIN_TAB", 0);
      expect(mockStore.commit).toHaveBeenCalledWith("login/OPEN_LOGIN_MODAL");
    });
  });

  describe("toggleWatchLaterVideo", () => {
    it("returns null for invalid videoId", async () => {
      const result = await toggleWatchLaterVideo(mockStore, "invalid");
      expect(result).toBeNull();
    });

    it("returns null for non-positive videoId", async () => {
      const result = await toggleWatchLaterVideo(mockStore, 0);
      expect(result).toBeNull();
    });

    it("returns null and opens login modal when not authenticated", async () => {
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue(null);
      const result = await toggleWatchLaterVideo(mockStore, 123);
      expect(result).toBeNull();
      expect(mockStore.commit).toHaveBeenCalledWith("login/SET_LOGIN_TAB", 0);
      expect(mockStore.commit).toHaveBeenCalledWith("login/OPEN_LOGIN_MODAL");
    });

    it("returns true when added to watch later", async () => {
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue("valid-token");
      const { mbToggleWatchLater } = await import("@/api/minibili");
      mbToggleWatchLater.mockResolvedValue({ in_watch_later: true });
      const { ElMessage } = await import("element-plus");
      const result = await toggleWatchLaterVideo(mockStore, 456);
      expect(result).toBe(true);
      expect(ElMessage.success).toHaveBeenCalled();
    });

    it("returns false when removed from watch later", async () => {
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue("valid-token");
      const { mbToggleWatchLater } = await import("@/api/minibili");
      mbToggleWatchLater.mockResolvedValue({ in_watch_later: false });
      const { ElMessage } = await import("element-plus");
      const result = await toggleWatchLaterVideo(mockStore, 789);
      expect(result).toBe(false);
      expect(ElMessage.success).toHaveBeenCalled();
    });

    it("returns null and shows error on API failure", async () => {
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue("valid-token");
      const { mbToggleWatchLater } = await import("@/api/minibili");
      mbToggleWatchLater.mockRejectedValue(new Error("network error"));
      const { ElMessage } = await import("element-plus");
      const result = await toggleWatchLaterVideo(mockStore, 101);
      expect(result).toBeNull();
      expect(ElMessage.error).toHaveBeenCalled();
    });

    it("falls back to default error message", async () => {
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue("valid-token");
      const { mbToggleWatchLater } = await import("@/api/minibili");
      mbToggleWatchLater.mockRejectedValue({});
      const { ElMessage } = await import("element-plus");
      const result = await toggleWatchLaterVideo(mockStore, 202);
      expect(result).toBeNull();
      expect(ElMessage.error).toHaveBeenCalled();
    });
  });
});
