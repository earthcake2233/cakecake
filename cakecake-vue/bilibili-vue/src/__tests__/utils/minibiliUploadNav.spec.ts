import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  isMinibiliApiEnv,
  minibiliUploadOpensLoginModal,
  resolveMinibiliUploadNavTo
} from "@/utils/minibiliUploadNav";

vi.mock("@/utils/authTokens", () => ({
  getAccessToken: vi.fn()
}));

describe("minibiliUploadNav", () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
  });

  describe("isMinibiliApiEnv", () => {
    it("returns false when env is not set", () => {
      vi.stubEnv("VITE_MINIBILI_API", "false");
      expect(isMinibiliApiEnv()).toBe(false);
    });

    it("returns true when env is 'true'", () => {
      vi.stubEnv("VITE_MINIBILI_API", "true");
      expect(isMinibiliApiEnv()).toBe(true);
    });

    it("returns true when env is '1'", () => {
      vi.stubEnv("VITE_MINIBILI_API", "1");
      expect(isMinibiliApiEnv()).toBe(true);
    });
  });

  describe("minibiliUploadOpensLoginModal", () => {
    it("returns false when not minibili api env", () => {
      vi.stubEnv("VITE_MINIBILI_API", "false");
      expect(minibiliUploadOpensLoginModal()).toBe(false);
    });

    it("returns false when minibili api and has token", async () => {
      vi.stubEnv("VITE_MINIBILI_API", "true");
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue("some-token");
      expect(minibiliUploadOpensLoginModal()).toBe(false);
    });

    it("returns true when minibili api and no token", async () => {
      vi.stubEnv("VITE_MINIBILI_API", "true");
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue(null);
      expect(minibiliUploadOpensLoginModal()).toBe(true);
    });
  });

  describe("resolveMinibiliUploadNavTo", () => {
    it("returns { name: 'upload' } when not minibili api env", () => {
      vi.stubEnv("VITE_MINIBILI_API", "false");
      expect(resolveMinibiliUploadNavTo()).toEqual({ name: "upload" });
    });

    it("returns { name: 'upload' } when minibili api and has token", async () => {
      vi.stubEnv("VITE_MINIBILI_API", "true");
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue("some-token");
      expect(resolveMinibiliUploadNavTo()).toEqual({ name: "upload" });
    });

    it("returns { name: 'home' } when minibili api and no token", async () => {
      vi.stubEnv("VITE_MINIBILI_API", "true");
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue(null);
      expect(resolveMinibiliUploadNavTo()).toEqual({ name: "home" });
    });
  });
});
