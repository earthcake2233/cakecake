import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";

var mockHttp = { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() };
vi.mock("@/utils/http", () => ({ default: mockHttp }));

let api;
beforeAll(async () => {
  vi.stubEnv("VITE_MINIBILI_API", "true");
  vi.stubEnv("VITE_USE_REMOTE_API", "false");
  vi.stubGlobal("console", { warn: vi.fn(), log: vi.fn(), error: vi.fn() });
  api = await import("@/api/index");
});
beforeEach(() => { vi.clearAllMocks(); });

describe("getSearchDefaultWords", () => {
  it("calls hot-search on success", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { items: [{ title: "t1" }, { title: "t2" }] } });
    var r = await api.getSearchDefaultWords();
    expect(mockHttp.get.mock.calls[0][0]).toBe("/api/v1/hot-search");
    expect(r.data.show_name).toBe("t1");
  });
  it("falls back on API failure", async () => {
    mockHttp.get.mockRejectedValue(new Error("fail"));
    var r = await api.getSearchDefaultWords();
    expect(r.code).toBe(0);
    expect(r.data.show_name).toBeTruthy();
  });
  it("falls back on non-zero code", async () => {
    mockHttp.get.mockResolvedValue({ code: -1, msg: "err" });
    var r = await api.getSearchDefaultWords();
    expect(r.code).toBe(0);
    expect(r.data.show_name).toBeTruthy();
  });
});

describe("getHomeRecommendPool (minibili calls /api/v1/videos)", () => {
  it("fetches pages and maps items", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { items: [{ id: 1, title: "v1", cover_url: "c", uploader: "u", play_count: 500 }], next_cursor: "" } });
    var r = await api.getHomeRecommendPool(10);
    expect(mockHttp.get.mock.calls[0][0]).toBe("/api/v1/videos");
    expect(r[0].aid).toBe(1);
    expect(r[0].author).toBe("u");
  });
  it("fetches multiple pages if needed", async () => {
    mockHttp.get
      .mockResolvedValueOnce({ code: 0, data: { items: [{ id: 1, title: "v1", cover_url: "c", uploader: "u", play_count: 500 }], next_cursor: "abc" } })
      .mockResolvedValueOnce({ code: 0, data: { items: [{ id: 2, title: "v2", cover_url: "c2", uploader: "u2", play_count: 300 }], next_cursor: "" } });
    var r = await api.getHomeRecommendPool(2);
    expect(mockHttp.get).toHaveBeenCalledTimes(2);
    expect(r.length).toBe(2);
  });
  it("falls back to local on API failure", async () => {
    mockHttp.get.mockRejectedValue(new Error("fail"));
    var r = await api.getHomeRecommendPool(10);
    // Falls back to local.rankingIndexResponse(3).data
    expect(Array.isArray(r)).toBe(true);
    expect(r.length).toBeGreaterThan(0);
  });
  it("returns [] when api data is empty", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { items: [], next_cursor: "" } });
    var r = await api.getHomeRecommendPool(5);
    expect(r).toEqual([]);
  });
});

describe("getSearchResult", () => {
  it("calls /api/v1/search", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { result: { video: [{ aid: 1 }] }, search_status: "ok" } });
    var r = await api.getSearchResult(1, "keyword");
    expect(mockHttp.get.mock.calls[0][0]).toBe("/api/v1/search");
    expect(r.data.search_status).toBe("ok");
  });
  it("passes video filters", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { result: {}, search_status: "ok" } });
    await api.getSearchResult(1, "k", { type: "video", videoFilters: { order: "click", duration: "60", zone: "1" } });
    expect(mockHttp.get.mock.calls[0][1].params.order).toBe("click");
  });
  it("infers search_status ok from results", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { result: { video: [{ aid: 1 }] } } });
    var r = await api.getSearchResult(1, "k");
    expect(r.data.search_status).toBe("ok");
  });
  it("infers search_status empty", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { result: {} } });
    var r = await api.getSearchResult(1, "k");
    expect(r.data.search_status).toBe("empty");
  });
  it("uses default type=all", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { result: {}, search_status: "ok" } });
    await api.getSearchResult(1, "k", {});
    expect(mockHttp.get.mock.calls[0][1].params.type).toBe("all");
  });
});

describe("getRanking", () => {
  it("returns empty list for non-type-1", async () => {
    var r = await api.getRanking(2, 0, 0, 3);
    expect(r.data.list).toEqual([]);
  });
  it("calls fetchMinibiliZoneVideos for type 1", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { items: [{ id: 100, title: "r", cover_url: "i", duration: 200, play_count: 9999, danmaku_count: 50, in_watch_later: false }], zone_video_count: 10 } });
    var r = await api.getRanking(1, 1, 0, 1);
    expect(mockHttp.get.mock.calls[0][0]).toBe("/api/v1/videos");
    expect(r.data.list.length).toBeGreaterThan(0);
  });
  it("handles empty response", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { items: [], zone_video_count: 0 } });
    var r = await api.getRanking(1, 0, 0, 3);
    expect(r.data.list).toEqual([]);
  });
  it("passes arc_type", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { items: [], zone_video_count: 0 } });
    await api.getRanking(1, 0, 1, 7);
    expect(mockHttp.get.mock.calls[0][1].params.arc_type).toBe(1);
  });
});
