import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
vi.mock("@/api",()=>{
  var m={getSearchResult:vi.fn(),getSearchDefaultWords:vi.fn(),getSuggest:vi.fn(),getRankingIndex:vi.fn(),getOnline:vi.fn(),getDynamicRegion:vi.fn(),getAdSlide:vi.fn()};
  return m;
});
var api;
var act;
beforeAll(async()=>{
  api=await import("@/api");
  act=await import("@/store/actions");
});
beforeEach(()=>{vi.clearAllMocks()});
describe("store actions",()=>{
  it("setRankingIndex",async()=>{
    api.getRankingIndex.mockResolvedValue({code:0,data:[{aid:1}]});
    var commit=vi.fn(); await act.setRankingIndex({commit},7);
    expect(api.getRankingIndex).toHaveBeenCalledWith(7);
    expect(commit).toHaveBeenCalled();
  });
  it("setOnline",async()=>{
    api.getOnline.mockResolvedValue({data:{web_online:42}});
    var commit=vi.fn(); await act.setOnline({commit});
    expect(api.getOnline).toHaveBeenCalled();
    expect(commit).toHaveBeenCalled();
  });


});
