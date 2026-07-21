import { describe, it, expect, vi, beforeEach } from "vitest";
import { openMinibiliLoginModal } from "@/utils/minibiliLoginModal";

vi.mock("@/utils/authTokens", () => ({
  setMinibiliPostLoginRedirect: vi.fn()
}));

const { storeCommit } = vi.hoisted(() => ({
  storeCommit: vi.fn()
}));

vi.mock("@/store/index", () => ({ default: { commit: storeCommit } }));

describe("minibiliLoginModal", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("commits login tab 0 and opens modal with default opts", () => {
    openMinibiliLoginModal();
    expect(storeCommit).toHaveBeenCalledWith("login/SET_LOGIN_TAB", 0);
    expect(storeCommit).toHaveBeenCalledWith("login/OPEN_LOGIN_MODAL");
  });

  it("uses tab=1 when specified", () => {
    openMinibiliLoginModal({ tab: 1 });
    expect(storeCommit).toHaveBeenCalledWith("login/SET_LOGIN_TAB", 1);
  });

  it("sets redirect when valid path provided", async () => {
    openMinibiliLoginModal({ redirect: "/video/123" });
    const { setMinibiliPostLoginRedirect } = await import("@/utils/authTokens");
    expect(setMinibiliPostLoginRedirect).toHaveBeenCalledWith("/video/123");
  });

  it("does not set redirect for non-slash paths", async () => {
    openMinibiliLoginModal({ redirect: "video/123" });
    const { setMinibiliPostLoginRedirect } = await import("@/utils/authTokens");
    expect(setMinibiliPostLoginRedirect).not.toHaveBeenCalled();
  });

  it("does not set redirect for empty redirect", async () => {
    openMinibiliLoginModal({ redirect: "" });
    const { setMinibiliPostLoginRedirect } = await import("@/utils/authTokens");
    expect(setMinibiliPostLoginRedirect).not.toHaveBeenCalled();
  });
});