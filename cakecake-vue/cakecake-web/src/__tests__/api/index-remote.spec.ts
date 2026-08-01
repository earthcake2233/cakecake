import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";

vi.hoisted(() => {
  process.env.VITE_USE_REMOTE_API = "true";
  process.env.VITE_MINIBILI_API = "";
});

const mockHttp = { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() };
vi.mock("@/utils/http", () => ({ default: mockHttp }));

const mockLocal = {
  loginUserResponse: { code: 0, data: { uname: "should_not_be_called" } },
  topInfoResponse: { code: 0, data: {} },
  adSlideResponse: { code: 0, data: [] },
  locResponse: { code: 0, data: [] },
  locsResponse: { code: 0, data: [] },
  menuGifResponse: { code: 0, data: { url: "gif" } },
  onlineResponse: { code: 0, data: { web_online: 100 } },
  livelingEmptyResponse: { code: 0, data: [] },
  timelineGlobalResponse: { code: 0, data: [] },
  rankingGlobal3Response: { code: 0, data: [] },
  rankingGlobal7Response: { code: 0, data: [] },
  timelineCnResponse: { code: 0, data: [] },
  rankingCn3Response: { code: 0, data: [] },
  rankingCn7Response: { code: 0, data: [] },
  fjAdSlideResponse: { code: 0, data: [] },
  gcAdSlideResponse: { code: 0, data: [] },
  suggestEmptyResponse: { code: 0, result: { tag: [] } },
  emptySearchPayload: { code: 0, data: { result: {} } },
  emptyRankPagePayload: [],
  emptySeasonRankPayload: { code: 0, data: [] },
  emptyMoviesRankPayload: { code: 0, data: [] },
  emptySeasonDetail: { code: 0, data: {} },
  HOT_SEARCH_FALLBACK_TITLES: ["fallback1", "fallback2"],
  rankingIndexResponse: vi.fn(() => ({ data: [{ aid: 1, title: "local" }] })),
  dynamicRegionResponse: vi.fn(() => ({ data: { archives: [], num: 0 } })),
  newlistResponse: vi.fn(() => ({ data: { archives: [], num: 0 } })),
  rankingRegionResponse: vi.fn(() => ({ data: [] })),
};
vi.mock("@/mock/localApi", () => mockLocal);

let api;
beforeAll(async () => {
  api = await import("@/api/index");
});
beforeEach(() => { vi.clearAllMocks(); });

describe("api/index.js - remote API mode", () => {
  it("getUserInfo returns mock value", async () => {
    const r = await api.getUserInfo();
    expect(r.data.uname).toBe("should_not_be_called");
  });
  it("getVipInfo returns mock value", async () => {
    const r = await api.getVipInfo();
    expect(r.data).toEqual({});
  });
  it("getAdSlide calls /ad_slide", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: [{ id: 2 }] });
    const r = await api.getAdSlide({ position_id: 5 });
    expect(mockHttp.get).toHaveBeenCalledWith("/ad_slide", { params: { position_id: 5 } });
    expect(r.data[0].id).toBe(2);
  });
  it("getLoc calls /loc", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: ["remote"] });
    const r = await api.getLoc({ a: 1 });
    expect(mockHttp.get).toHaveBeenCalledWith("/loc", { params: { a: 1 } });
    expect(r.data[0]).toBe("remote");
  });
  it("getLocs calls /locs", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: ["a","b"] });
    const r = await api.getLocs({ a: 1 });
    expect(mockHttp.get).toHaveBeenCalledWith("/locs", { params: { a: 1 } });
    expect(r.data[0]).toBe("a");
  });
  it("getMenuIcon calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { url: "remote_gif" } });
    const r = await api.getMenuIcon();
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data.url).toBe("remote_gif");
  });
  it("getOnline calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { web_online: 999 } });
    const r = await api.getOnline();
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data.web_online).toBe(999);
  });
  it("getLiveling calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: [{ roomid: 1 }] });
    const r = await api.getLiveling();
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data[0].roomid).toBe(1);
  });
  it("getSearchDefaultWords calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { show_name: "remote_word" } });
    const r = await api.getSearchDefaultWords();
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data.show_name).toBe("remote_word");
  });
  it("getSuggest calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: { result: { tag: [{ name: "remote_tag" }] } } });
    const r = await api.getSuggest("term");
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data.result.tag[0].name).toBe("remote_tag");
  });
  it("getHomeRecommendPool calls /ranking/index in remote mode", async () => {
    mockHttp.get.mockResolvedValue({ data: [{ aid: 42, title: "remote_rec" }] });
    const r = await api.getHomeRecommendPool(10);
    expect(mockHttp.get).toHaveBeenCalledWith("/ranking/index", { params: { day: 3 } });
    expect(r[0].aid).toBe(42);
  });
  it("getRankingIndex calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: [{ aid: 7, title: "rank_day" }] });
    const r = await api.getRankingIndex(7);
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data[0].aid).toBe(7);
  });
  it("getDynamicRegion calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ data: { archives: [{ aid: 88 }], num: 1 } });
    const r = await api.getDynamicRegion({ rid: 1, ps: 10 });
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data.archives[0].aid).toBe(88);
  });
  it("getNewlist calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ data: { archives: [{ aid: 77 }], num: 5 } });
    const r = await api.getNewlist({ rid: 3, ps: 5 });
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data.num).toBe(5);
  });
  it("getRankingRegion calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ data: [{ aid: 129 }] });
    const r = await api.getRankingRegion({ rid: 129, day: 3 });
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data[0].aid).toBe(129);
  });
  it("getTimelineGlobal calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: [{ season_id: 1 }] });
    const r = await api.getTimelineGlobal();
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data[0].season_id).toBe(1);
  });
  it("getRankingGlobal3 calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: [{ aid: 11 }] });
    const r = await api.getRankingGlobal3();
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data[0].aid).toBe(11);
  });
  it("getRankingGlobal7 calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: [{ aid: 22 }] });
    const r = await api.getRankingGlobal7();
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data[0].aid).toBe(22);
  });
  it("getTimelineCn calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: [{ season_id: 33 }] });
    const r = await api.getTimelineCn();
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data[0].season_id).toBe(33);
  });
  it("getRankingCn3 calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: [{ aid: 44 }] });
    const r = await api.getRankingCn3();
    expect(mockHttp.get).toHaveBeenCalled();
    expect(r.data[0].aid).toBe(44);
  });
  it("getRankingCn7 calls http.get with JSON headers", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: [{ aid: 55 }] });
    const r = await api.getRankingCn7();
    expect(mockHttp.get).toHaveBeenCalledWith("/ranking/cn_7", { headers: { "Content-Type": "application/json" } });
    expect(r.data[0].aid).toBe(55);
  });
  it("getGlobalAdSlide calls /public/fj_ad_slide.json", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: [{ link: "/fj" }] });
    const r = await api.getGlobalAdSlide();
    expect(mockHttp.get).toHaveBeenCalledWith("/public/fj_ad_slide.json");
    expect(r.data[0].link).toBe("/fj");
  });
  it("getCnAdSlide calls /public/gc_ad_slide.json", async () => {
    mockHttp.get.mockResolvedValue({ code: 0, data: [{ link: "/gc" }] });
    const r = await api.getCnAdSlide();
    expect(mockHttp.get).toHaveBeenCalledWith("/public/gc_ad_slide.json");
    expect(r.data[0].link).toBe("/gc");
  });
  it("getRanking non-type-1 calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ data: [{ aid: 66 }] });
    const r = await api.getRanking(2, 0, 0, 3);
    expect(mockHttp.get).toHaveBeenCalledWith("/ranking", { params: { rid: 0, day: 3, type: 2, arc_type: 0 } });
  });
  it("getRanking type-1 calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ data: [{ aid: 77 }] });
    const r = await api.getRanking(1, 0, 0, 1);
    expect(mockHttp.get).toHaveBeenCalledWith("/ranking", { params: { rid: 0, day: 1, type: 1, arc_type: 0 } });
  });
  it("getSeasonRank calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ data: [{ season_id: 88 }] });
    const r = await api.getSeasonRank(3, 1);
    expect(mockHttp.get).toHaveBeenCalledWith("/season/rank/list", { params: { day: 3, season_type: 1 } });
  });
  it("getMoviesRank calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ data: [{ movie_id: 99 }] });
    const r = await api.getMoviesRank(7, 1);
    expect(mockHttp.get).toHaveBeenCalledWith("/ranking/movies/all-7-1.json");
  });
  it("getSearchResult calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ data: { result: { video: [] } } });
    const r = await api.getSearchResult(1, "keyword");
    expect(mockHttp.get).toHaveBeenCalled();
  });
  it("getSeason calls http.get", async () => {
    mockHttp.get.mockResolvedValue({ data: { media_id: 123 } });
    const r = await api.getSeason(123);
    expect(mockHttp.get).toHaveBeenCalledWith("/search/season", { params: { media_id: 123 } });
  });
});
