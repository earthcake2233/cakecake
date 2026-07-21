import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  getLastMessageUnreadSummary,
  subscribeMessageUnread,
  refreshMessageUnread
} from "@/utils/messageUnread";

vi.mock("@/api/minibili", () => ({
  mbUnreadSummary: vi.fn()
}));

vi.mock("@/utils/authTokens", () => ({
  getAccessToken: vi.fn()
}));

describe("messageUnread", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("getLastMessageUnreadSummary returns null initially", () => {
    expect(getLastMessageUnreadSummary()).toBeNull();
  });

  it("subscribeMessageUnread returns an unsubscribe function", () => {
    const fn = vi.fn();
    const unsub = subscribeMessageUnread(fn);
    expect(typeof unsub).toBe("function");
  });

  it("subscribeMessageUnread calls callback immediately if lastSummary exists", async () => {
    const { getAccessToken } = await import("@/utils/authTokens");
    getAccessToken.mockReturnValue("token");
    const { mbUnreadSummary } = await import("@/api/minibili");
    mbUnreadSummary.mockResolvedValue({ total: 5 });
    await refreshMessageUnread();

    const fn = vi.fn();
    subscribeMessageUnread(fn);
    expect(fn).toHaveBeenCalledWith({ total: 5 });
  });

  it("refreshMessageUnread returns null when not authenticated", async () => {
    const { getAccessToken } = await import("@/utils/authTokens");
    getAccessToken.mockReturnValue(null);

    const result = await refreshMessageUnread();
    expect(result).toBeNull();
    expect(getLastMessageUnreadSummary()).toBeNull();
  });

  it("refreshMessageUnread fetches and caches summary", async () => {
    const { getAccessToken } = await import("@/utils/authTokens");
    getAccessToken.mockReturnValue("token");
    const { mbUnreadSummary } = await import("@/api/minibili");
    mbUnreadSummary.mockResolvedValue({ total: 3, chat: 1 });

    const result = await refreshMessageUnread();
    expect(result).toEqual({ total: 3, chat: 1 });
    expect(getLastMessageUnreadSummary()).toEqual({ total: 3, chat: 1 });
  });

  it("refreshMessageUnread notifies subscribers", async () => {
    const fn = vi.fn();
    subscribeMessageUnread(fn);

    const { getAccessToken } = await import("@/utils/authTokens");
    getAccessToken.mockReturnValue("token");
    const { mbUnreadSummary } = await import("@/api/minibili");
    mbUnreadSummary.mockResolvedValue({ total: 7 });

    await refreshMessageUnread();
    expect(fn).toHaveBeenCalledWith({ total: 7 });
  });

  it("refreshMessageUnread returns previous cached value on fetch error", async () => {
    const { getAccessToken } = await import("@/utils/authTokens");
    getAccessToken.mockReturnValue("token");
    const { mbUnreadSummary } = await import("@/api/minibili");

    // First call succeeds
    mbUnreadSummary.mockResolvedValueOnce({ total: 99 });
    await refreshMessageUnread();

    // Second call fails
    mbUnreadSummary.mockRejectedValueOnce(new Error("network"));
    const result = await refreshMessageUnread();
    expect(result).toEqual({ total: 99 });
  });

  it("subscriber errors are silently caught", async () => {
    const { getAccessToken } = await import("@/utils/authTokens");
    getAccessToken.mockReturnValue("token");
    const { mbUnreadSummary } = await import("@/api/minibili");
    mbUnreadSummary.mockResolvedValue({ total: 1 });

    const fn = vi.fn(() => { throw new Error("subscriber error"); });
    subscribeMessageUnread(fn);

    await expect(refreshMessageUnread()).resolves.toEqual({ total: 1 });
  });

  it("unsubscribe removes listener and does not call on next refresh", async () => {
    const fn = vi.fn();
    const unsub = subscribeMessageUnread(fn);

    const { getAccessToken } = await import("@/utils/authTokens");
    getAccessToken.mockReturnValue("token");
    const { mbUnreadSummary } = await import("@/api/minibili");
    mbUnreadSummary.mockResolvedValue({ total: 2 });

    unsub();
    vi.clearAllMocks(); // also clear the subscribe call
    await refreshMessageUnread();
    expect(fn).toHaveBeenCalledTimes(0);
  });
});
