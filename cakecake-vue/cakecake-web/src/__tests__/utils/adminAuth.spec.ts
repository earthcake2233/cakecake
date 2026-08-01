import { describe, it, expect, beforeEach } from "vitest";
import {
  getAdminAccessToken,
  getAdminRefreshToken,
  setAdminTokens,
  clearAdminTokens,
  isAdminLoggedIn
} from "@/utils/adminAuth";

describe("adminAuth", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("stores and retrieves tokens", () => {
    setAdminTokens("admin_access", "admin_refresh");
    expect(getAdminAccessToken()).toBe("admin_access");
    expect(getAdminRefreshToken()).toBe("admin_refresh");
    expect(isAdminLoggedIn()).toBe(true);
  });

  it("clearTokens removes everything", () => {
    setAdminTokens("admin_access", "admin_refresh");
    clearAdminTokens();
    expect(getAdminAccessToken()).toBe("");
    expect(isAdminLoggedIn()).toBe(false);
  });

  it("is not logged in without tokens", () => {
    expect(isAdminLoggedIn()).toBe(false);
  });
});
