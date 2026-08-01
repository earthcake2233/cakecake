import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  isCakecakeApiEnv,
  cakecakeUploadOpensLoginModal,
  resolveCakecakeUploadNavTo
} from "@/utils/cakecakeUploadNav";

vi.mock("@/utils/authTokens", () => ({
  getAccessToken: vi.fn()
}));

describe("cakecakeUploadNav", () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
  });

  describe("isCakecakeApiEnv", () => {
    it("returns false when env is not set", () => {
      vi.stubEnv("VITE_MINIBILI_API", "false");
      expect(isCakecakeApiEnv()).toBe(false);
    });

    it("returns true when env is 'true'", () => {
      vi.stubEnv("VITE_MINIBILI_API", "true");
      expect(isCakecakeApiEnv()).toBe(true);
    });

    it("returns true when env is '1'", () => {
      vi.stubEnv("VITE_MINIBILI_API", "1");
      expect(isCakecakeApiEnv()).toBe(true);
    });
  });

  describe("cakecakeUploadOpensLoginModal", () => {
    it("returns false when not cakecake api env", () => {
      vi.stubEnv("VITE_MINIBILI_API", "false");
      expect(cakecakeUploadOpensLoginModal()).toBe(false);
    });

    it("returns false when cakecake api and has token", async () => {
      vi.stubEnv("VITE_MINIBILI_API", "true");
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue("some-token");
      expect(cakecakeUploadOpensLoginModal()).toBe(false);
    });

    it("returns true when cakecake api and no token", async () => {
      vi.stubEnv("VITE_MINIBILI_API", "true");
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue(null);
      expect(cakecakeUploadOpensLoginModal()).toBe(true);
    });
  });

  describe("resolveCakecakeUploadNavTo", () => {
    it("returns { name: 'upload' } when not cakecake api env", () => {
      vi.stubEnv("VITE_MINIBILI_API", "false");
      expect(resolveCakecakeUploadNavTo()).toEqual({ name: "upload" });
    });

    it("returns { name: 'upload' } when cakecake api and has token", async () => {
      vi.stubEnv("VITE_MINIBILI_API", "true");
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue("some-token");
      expect(resolveCakecakeUploadNavTo()).toEqual({ name: "upload" });
    });

    it("returns { name: 'home' } when cakecake api and no token", async () => {
      vi.stubEnv("VITE_MINIBILI_API", "true");
      const { getAccessToken } = await import("@/utils/authTokens");
      getAccessToken.mockReturnValue(null);
      expect(resolveCakecakeUploadNavTo()).toEqual({ name: "home" });
    });
  });
});
