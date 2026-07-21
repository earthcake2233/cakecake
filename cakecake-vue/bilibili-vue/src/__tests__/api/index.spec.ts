import { describe, it, expect, vi, beforeAll } from "vitest";
var mockHttp = { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() };
vi.mock("@/utils/http", () => ({ default: mockHttp }));
var mockLocal = {
  loginUserResponse: { code: 0, data: { uname: "test" } },
  topInfoResponse: { code: 0, data: {} },
  adSlideResponse: { code: 0, data: [{ id: 1 }] },
  locResponse: { code: 0, data: [] },
  locsResponse: { code: 0, data: [1,2,3] },
  menuGifResponse: { code: 0, data: { url: "gif" } },
  onlineResponse: { code: 0, data: { web_online: 100, all_count: 500 } },
  livelingEmptyResponse: { code: 0, data: [] },
  timelineGlobalResponse: { code: 0, data: [{ season_id: 1 }] },
  rankingGlobal3Response: { code: 0, data: [{ aid: 1 }] },
  rankingGlobal7Response: { code: 0, data: [{ aid: 2 }] },
  timelineCnResponse: { code: 0, data: [{ season_id: 3 }] },
  rankingCn3Response: { code: 0, data: [{ aid: 4 }] },
  rankingCn7Response: { code: 0, data: [{ aid: 5 }] },
  fjAdSlideResponse: { code: 0, data: [] },
  gcAdSlideResponse: { code: 0, data: [] },
  suggestEmptyResponse: { code: 0, data: { result: { tag: [] } } },
  emptySearchPayload: { code: 0, data: { result: {} } },
  emptyRankPagePayload: [],
  emptySeasonRankPayload: { code: 0, data: [] },
  emptyMoviesRankPayload: { code: 0, data: [] },
  emptySeasonDetail: { code: 0, data: {} },
  HOT_SEARCH_FALLBACK_TITLES: ["??1","??2"],
  rankingIndexResponse: vi.fn((day) => ({ code: 0, data: [{ aid: 1, title: "r"+day }] })),
  dynamicRegionResponse: vi.fn((rid) => ({ data: { archives: [{ aid: rid }], num: 5 } })),
  newlistResponse: vi.fn((rid) => ({ data: { archives: [{ aid: rid }], num: 3 } })),
  rankingRegionResponse: vi.fn((rid) => ({ code: 0, data: [{ aid: rid }] })),
};
vi.mock("@/mock/localApi", () => mockLocal);
var api;
beforeAll(async () => {
  vi.stubEnv("VITE_USE_REMOTE_API", "false");
  vi.stubEnv("VITE_MINIBILI_API", "");
  vi.stubGlobal("console", { warn: vi.fn(), log: vi.fn(), error: vi.fn() });
  api = await import("@/api/index");
});
describe("api/index.js - local mock", () => {
  it("getUserInfo", async () => { const r = await api.getUserInfo(); expect(r.data.uname).toBe("test"); });
  it("getVipInfo", async () => { const r = await api.getVipInfo(); expect(r.data).toBeDefined(); });
  it("getAdSlide", async () => { const r = await api.getAdSlide({ position_id: 5 }); expect(r.data[0].id).toBe(1); });
  it("getLoc", async () => { const r = await api.getLoc({ a: 1 }); expect(r).toEqual({ code: 0, data: [] }); });
  it("getLocs", async () => { const r = await api.getLocs({ a: 1 }); expect(r.data).toEqual([1,2,3]); });
  it("getMenuIcon", async () => { const r = await api.getMenuIcon(); expect(r.data.url).toBe("gif"); });
  it("getOnline", async () => { const r = await api.getOnline(); expect(r.data.web_online).toBe(100); });
  it("getLiveling", async () => { const r = await api.getLiveling(); expect(r.data).toEqual([]); });
  it("getSearchDefaultWords", async () => { const r = await api.getSearchDefaultWords(); expect(r.data.show_name).toBe("??1"); });
  it("getSuggest", async () => { const r = await api.getSuggest("term"); expect(r.data.result.tag).toEqual([]); });
  it("getHomeRecommendPool", async () => { const r = await api.getHomeRecommendPool(10); expect(Array.isArray(r)).toBe(true); expect(r[0].aid).toBe(1); });
  it("getRankingIndex", async () => { const r = await api.getRankingIndex(7); expect(r.data[0].title).toBe("r7"); });
  it("getDynamicRegion", async () => { const r = await api.getDynamicRegion({ rid: 1, ps: 10 }); expect(r.data.archives[0].aid).toBe(1); });
  it("getNewlist", async () => { const r = await api.getNewlist({ rid: 3, ps: 5 }); expect(r.data.num).toBe(3); });
  it("getRankingRegion", async () => { const r = await api.getRankingRegion({ rid: 129, day: 3 }); expect(r.data[0].aid).toBe(129); });
  it("getTimelineGlobal", async () => { const r = await api.getTimelineGlobal(); expect(r.data[0].season_id).toBe(1); });
  it("getRankingGlobal3", async () => { const r = await api.getRankingGlobal3(); expect(r.data[0].aid).toBe(1); });
  it("getRankingGlobal7", async () => { const r = await api.getRankingGlobal7(); expect(r.data[0].aid).toBe(2); });
  it("getTimelineCn", async () => { const r = await api.getTimelineCn(); expect(r.data[0].season_id).toBe(3); });
  it("getRankingCn3", async () => { const r = await api.getRankingCn3(); expect(r.data[0].aid).toBe(4); });
  it("getRankingCn7", async () => { const r = await api.getRankingCn7(); expect(r.data[0].aid).toBe(5); });
  it("getGlobalAdSlide", async () => { const r = await api.getGlobalAdSlide(); expect(r.data).toEqual([]); });
  it("getCnAdSlide", async () => { const r = await api.getCnAdSlide(); expect(r.data).toEqual([]); });
  it("getRanking non-type-1", async () => { const r = await api.getRanking(2, 0, 0, 3); expect(r.data).toEqual([]); });
  it("getRanking type 1", async () => { const r = await api.getRanking(1, 0, 0, 1); expect(r.data).toBeDefined(); });
  it("getSeasonRank", async () => { const r = await api.getSeasonRank(3, 1); expect(r.data).toEqual([]); });
  it("getMoviesRank", async () => { const r = await api.getMoviesRank(7, 1); expect(r.data).toEqual([]); });
  it("getSearchResult", async () => { const r = await api.getSearchResult(1, "keyword"); expect(r.data).toBeDefined(); });
  it("getSeason", async () => { const r = await api.getSeason(123); expect(r.data).toEqual({}); });
});
