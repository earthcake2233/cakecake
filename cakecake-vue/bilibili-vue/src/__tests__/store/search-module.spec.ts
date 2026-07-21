import { describe, it, expect, vi, beforeEach } from "vitest";
import searchModule from "@/store/modules/search";
const mockGetSearchResult = vi.fn(), mockGetSuggest = vi.fn(), mockGetSeason = vi.fn();
vi.mock("@/api", () => ({ getSearchResult: (...a) => mockGetSearchResult(...a), getSuggest: (...a) => mockGetSuggest(...a), getSeason: (...a) => mockGetSeason(...a) }));
describe("search state", () => {
  it("9 menus", () => expect(searchModule.state.searchMenu).toHaveLength(9));
  it("first title", () => expect(searchModule.state.searchMenu[0].title).toBe("综合"));
  it("searchWord empty", () => expect(searchModule.state.searchWord).toBe(""));
});
describe("search mutations", () => {
  const { mutations } = searchModule;
  let st;
  beforeEach(() => { st = JSON.parse(JSON.stringify(searchModule.state)); });
  it("SET_SEARCH_VALUE", () => { mutations.SET_SEARCH_VALUE(st, "t"); expect(st.searchWord).toBe("t"); });
  it("SET_MENU_SHOW", () => { mutations.SET_MENU_SHOW(st, false); expect(st.menuShow).toBe(false); });
  it("SET_TOP_NUM", () => { mutations.SET_TOP_NUM(st, {video:100, media_bangumi:50, movie:30}); expect(st.searchMenu[1].resultNum).toBe(100); });
  it("SET_ALL_RESULT", () => { mutations.SET_ALL_RESULT(st, {result:{video:[]}}); expect(st.allResult.result.video).toEqual([]); });
  it("SET_SEASON", () => { st.allResult = {result:{media_bangumi:[{season:"winter"}]}}; mutations.SET_SEASON(st, {id:0, result:"spring"}); expect(st.allResult.result.media_bangumi[0].season).toBe("spring"); });
});
describe("search actions", () => {
  beforeEach(() => { vi.clearAllMocks(); });
  it("setSuggest empty", async () => { const c = vi.fn(); await searchModule.actions.setSuggest({commit:c, state:{searchWord:""}}); expect(c).toHaveBeenCalledWith("SET_SUGGEST", {tag:[]}); expect(mockGetSuggest).not.toHaveBeenCalled(); });
  it("setSuggest term", async () => { mockGetSuggest.mockResolvedValue({result:{tag:[{name:"t"}]}}); const c = vi.fn(); await searchModule.actions.setSuggest({commit:c, state:{searchWord:"动漫"}}); expect(c).toHaveBeenCalledWith("SET_SUGGEST", {tag:[{name:"t"}]}); });
  it("setSuggest array", async () => { mockGetSuggest.mockResolvedValue({result:["a","b"]}); const c = vi.fn(); await searchModule.actions.setSuggest({commit:c, state:{searchWord:"x"}}); expect(c).toHaveBeenCalledWith("SET_SUGGEST", {tag:["a","b"]}); });
  it("setSuggest null", async () => { mockGetSuggest.mockResolvedValue({}); const c = vi.fn(); await searchModule.actions.setSuggest({commit:c, state:{searchWord:"x"}}); expect(c).toHaveBeenCalledWith("SET_SUGGEST", {tag:[]}); });
  it("setAllResult", async () => { mockGetSearchResult.mockResolvedValue({data:{top_tlist:{video:100}, result:{media_bangumi:[{media_id:123}]}}}); mockGetSeason.mockResolvedValue({result:{season:"spring"}}); const c = vi.fn(); await searchModule.actions.setAllResult({commit:c, state:{...searchModule.state, allResult:{result:{media_bangumi:[{media_id:123}]}}}}, {highlight:1, keyword:"test", type:"all"}); expect(mockGetSearchResult).toHaveBeenCalledWith(1,"test",{type:"all",sort:"",videoFilters:null}); });
  it("setSeason", () => { const c = vi.fn(); searchModule.actions.setSeason({commit:c}, {id:0, result:"fall"}); expect(c).toHaveBeenCalledWith("SET_SEASON", {id:0, result:"fall"}); });
});