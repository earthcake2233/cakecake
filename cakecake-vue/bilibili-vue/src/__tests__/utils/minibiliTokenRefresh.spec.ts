import { describe, it, expect, beforeEach, vi, afterAll } from "vitest";
import {
  isAccessTokenExpired,
  shouldAttemptTokenRefresh,
  ensureFreshAccessToken,
  refreshMinibiliAccessToken,
  installMinibiliTokenAutoRefresh
} from "@/utils/minibiliTokenRefresh";
import { setTokens, getAccessToken } from "@/utils/authTokens";

vi.mock("axios", () => {
  const mockPost = vi.fn();
  return { default: { create: vi.fn(), post: mockPost } };
});

function makeJwt(payload) {
  const h = btoa(JSON.stringify({ alg: "HS256" }));
  const p = btoa(JSON.stringify(payload));
  return `${h}.${p}.sig`;
}

const futureExp = Math.floor(Date.now() / 1000) + 3600;
const pastExp = Math.floor(Date.now() / 1000) - 3600;

describe("minibiliTokenRefresh", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
    vi.stubEnv("VITE_MINIBILI_API", "true");
    vi.stubEnv("VITE_REMOTE_API_BASE", "");
  });
  afterAll(() => vi.unstubAllEnvs());
  
  describe("isAccessTokenExpired", () => {
    it("returns true when no token", () => {
      expect(isAccessTokenExpired()).toBe(true);
    });
    it("returns false for valid token", () => {
      setTokens(makeJwt({ exp: futureExp }), "");
      expect(isAccessTokenExpired()).toBe(false);
    });
    it("returns true for expired token", () => {
      setTokens(makeJwt({ exp: pastExp }), "");
      expect(isAccessTokenExpired()).toBe(true);
    });
    it("handles malformed JWT", () => {
      setTokens("not-a-jwt", "");
      expect(isAccessTokenExpired()).toBe(true);
    });
  });
  
  describe("shouldAttemptTokenRefresh", () => {
    it("returns false without refresh token", () => {
      expect(shouldAttemptTokenRefresh({ response: { status: 401 } }, { url: "/api/v1/videos" })).toBe(false);
    });
    it("returns true on 401 with refresh token", () => {
      setTokens("", "rt");
      expect(shouldAttemptTokenRefresh({ response: { status: 401 } }, { url: "/api/v1/videos" })).toBe(true);
    });
    it("returns false for auth URLs", () => {
      setTokens("", "rt");
      expect(shouldAttemptTokenRefresh({ response: { status: 401 } }, { url: "/api/v1/auth/login" })).toBe(false);
    });
    it("returns false when already refreshed", () => {
      setTokens("", "rt");
      expect(shouldAttemptTokenRefresh({ response: { status: 401 } }, { url: "/api/v1/videos", _mbTokenRefresh: true })).toBe(false);
    });
    it("returns false when no config", () => {
      setTokens("", "rt");
      expect(shouldAttemptTokenRefresh({ response: { status: 401 } }, null)).toBe(false);
    });
  });

  describe("ensureFreshAccessToken", () => {
    it("returns false without refresh token", async () => {
      expect(await ensureFreshAccessToken()).toBe(false);
    });
    it("returns true when access is still fresh", async () => {
      setTokens(makeJwt({ exp: futureExp }), "rt");
      expect(await ensureFreshAccessToken()).toBe(true);
    });
    it("triggers refresh when access expired", async () => {
      setTokens(makeJwt({ exp: pastExp }), "rt");
      const axios = await import("axios");
      axios.default.post.mockResolvedValue({
        data: { code: 0, data: { access_token: "new-at", refresh_token: "new-rt" } }
      });
      expect(await ensureFreshAccessToken()).toBe(true);
    });
  });

  describe("refreshMinibiliAccessToken", () => {
    it("returns false without refresh token", async () => {
      expect(await refreshMinibiliAccessToken()).toBe(false);
    });
    it("calls refresh API and sets tokens on success", async () => {
      setTokens("old-at", "old-rt");
      const axios = await import("axios");
      axios.default.post.mockResolvedValue({
        data: { code: 0, data: { access_token: "new-at", refresh_token: "new-rt" } }
      });
      expect(await refreshMinibiliAccessToken()).toBe(true);
      expect(getAccessToken()).toBe("new-at");
    });
    it("returns false on API failure", async () => {
      setTokens("old-at", "rt");
      const axios = await import("axios");
      axios.default.post.mockRejectedValue(new Error("network"));
      expect(await refreshMinibiliAccessToken()).toBe(false);
    });
    it("returns false when API returns error code", async () => {
      setTokens("old-at", "rt");
      const axios = await import("axios");
      axios.default.post.mockResolvedValue({ data: { code: 401, msg: "bad refresh" } });
      expect(await refreshMinibiliAccessToken()).toBe(false);
    });
  });

  describe("installMinibiliTokenAutoRefresh", () => {
    it("does not throw when document is defined", () => {
      expect(() => installMinibiliTokenAutoRefresh()).not.toThrow();
    });
  });
});
