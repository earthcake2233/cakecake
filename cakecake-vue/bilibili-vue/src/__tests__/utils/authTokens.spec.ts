import { describe, it, expect, beforeEach } from "vitest";
import {
  setTokens,
  getAccessToken,
  getRefreshToken,
  getUserId,
  clearTokens,
  isLoggedIn,
  setMinibiliDisplayName,
  getMinibiliDisplayName,
  setMinibiliPostLoginRedirect,
  consumeMinibiliPostLoginRedirect,
  clearMinibiliPostLoginRedirect
} from "@/utils/authTokens";

describe("authTokens", () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  it("stores and retrieves tokens", () => {
    setTokens("access123", "refresh456");
    expect(getAccessToken()).toBe("access123");
    expect(getRefreshToken()).toBe("refresh456");
  });

  it("extracts user_id from JWT access token", () => {
    const header = btoa('{"alg":"HS256"}');
    const payload = btoa('{"user_id":42}');
    const token = `${header}.${payload}.signature`;
    setTokens(token, "refresh");
    expect(getUserId()).toBe(42);
  });

  it("handles JWT without user_id", () => {
    const header = btoa('{"alg":"HS256"}');
    const payload = btoa('{"sub":"test"}');
    const token = `${header}.${payload}.signature`;
    setTokens(token, "refresh");
    expect(getUserId()).toBeNull();
  });

  it("handles malformed JWT gracefully", () => {
    setTokens("invalid-token", "refresh");
    expect(getUserId()).toBeNull();
  });

  it("isLoggedIn returns true with token", () => {
    expect(isLoggedIn()).toBe(false);
    setTokens("access", "refresh");
    expect(isLoggedIn()).toBe(true);
  });

  it("clearTokens removes everything", () => {
    setTokens("access", "refresh");
    setMinibiliDisplayName("testuser");
    clearTokens();
    expect(getAccessToken()).toBe("");
    expect(getRefreshToken()).toBe("");
    expect(getUserId()).toBeNull();
    expect(getMinibiliDisplayName()).toBe("");
    expect(isLoggedIn()).toBe(false);
  });

  it("handles display name", () => {
    expect(getMinibiliDisplayName()).toBe("");
    setMinibiliDisplayName("测试用户");
    expect(getMinibiliDisplayName()).toBe("测试用户");
    setMinibiliDisplayName("");
    expect(getMinibiliDisplayName()).toBe("");
  });

  it("manages post-login redirect", () => {
    expect(consumeMinibiliPostLoginRedirect()).toBeNull();
    setMinibiliPostLoginRedirect("/upload");
    expect(consumeMinibiliPostLoginRedirect()).toBe("/upload");
    expect(consumeMinibiliPostLoginRedirect()).toBeNull();
  });

  it("ignores non-absolute redirect paths", () => {
    setMinibiliPostLoginRedirect("relative");
    expect(consumeMinibiliPostLoginRedirect()).toBeNull();
    clearMinibiliPostLoginRedirect();
    expect(consumeMinibiliPostLoginRedirect()).toBeNull();
  });
});
