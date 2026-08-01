import { describe, it, expect, vi, beforeEach } from "vitest";
import { openCakecakeLoginModal } from "@/utils/cakecakeLoginModal";

vi.mock("@/utils/authTokens", () => ({
  setCakecakePostLoginRedirect: vi.fn()
}));

const { storeCommit } = vi.hoisted(() => ({
  storeCommit: vi.fn()
}));

vi.mock("@/store/index", () => ({ default: { commit: storeCommit } }));

describe("cakecakeLoginModal", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("commits login tab 0 and opens modal with default opts", () => {
    openCakecakeLoginModal();
    expect(storeCommit).toHaveBeenCalledWith("login/SET_LOGIN_TAB", 0);
    expect(storeCommit).toHaveBeenCalledWith("login/OPEN_LOGIN_MODAL");
  });

  it("uses tab=1 when specified", () => {
    openCakecakeLoginModal({ tab: 1 });
    expect(storeCommit).toHaveBeenCalledWith("login/SET_LOGIN_TAB", 1);
  });

  it("sets redirect when valid path provided", async () => {
    openCakecakeLoginModal({ redirect: "/video/123" });
    const { setCakecakePostLoginRedirect } = await import("@/utils/authTokens");
    expect(setCakecakePostLoginRedirect).toHaveBeenCalledWith("/video/123");
  });

  it("does not set redirect for non-slash paths", async () => {
    openCakecakeLoginModal({ redirect: "video/123" });
    const { setCakecakePostLoginRedirect } = await import("@/utils/authTokens");
    expect(setCakecakePostLoginRedirect).not.toHaveBeenCalled();
  });

  it("does not set redirect for empty redirect", async () => {
    openCakecakeLoginModal({ redirect: "" });
    const { setCakecakePostLoginRedirect } = await import("@/utils/authTokens");
    expect(setCakecakePostLoginRedirect).not.toHaveBeenCalled();
  });
});