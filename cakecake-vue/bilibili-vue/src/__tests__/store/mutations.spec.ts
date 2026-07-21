import { describe, it, expect, beforeEach } from "vitest";
import mutations from "@/store/mutations";

// State structure matching store/state.js
function freshState() {
  return {
    scrollTop: 0,
    slide: { time: 4000, pagation: true, data: [] },
    recommend: { rec: [], day: 3 },
    popularize: [],
    online: [],
    elevator: [],
    module: [
      { id: 0, data: [], dynamic: 0, offsetTop: 0, ad: { data: [] },
        rankAllData: [], rankOriginalData: [], rankBangumiData: [],
        timeline: { data: [] } },
      { id: 1, data: [], dynamic: 0, offsetTop: 0, ad: { data: [] },
        rankAllData: [], rankOriginalData: [], rankBangumiData: [],
        timeline: { data: [] } }
    ]
  };
}

describe("store mutations", () => {
  let state;
  beforeEach(() => { state = freshState(); });
  
  it("SET_SCROLL_TOP updates scrollTop", () => {
    mutations.SET_SCROLL_TOP(state, 200);
    expect(state.scrollTop).toBe(200);
  });
  
  it("SET_AD_SLIDE updates module ad data", () => {
    mutations.SET_AD_SLIDE(state, { id: 0, data: [{ img: "slide1" }] });
    expect(state.module[0].ad.data).toEqual([{ img: "slide1" }]);
  });
  
  it("SET_SLIDE updates slide data", () => {
    mutations.SET_SLIDE(state, [{ img: "hero" }]);
    expect(state.slide.data).toEqual([{ img: "hero" }]);
  });
  
  it("SET_POPULARIZE updates popularize", () => {
    mutations.SET_POPULARIZE(state, [{ title: "promo" }]);
    expect(state.popularize).toEqual([{ title: "promo" }]);
  });
  
  it("SET_RANKING_INDEX updates recommend", () => {
    mutations.SET_RANKING_INDEX(state, { data: [{ id: 1 }], day: 7 });
    expect(state.recommend.rec).toEqual([{ id: 1 }]);
    expect(state.recommend.day).toBe(7);
  });
  
  it("SET_ONLINE updates online", () => {
    mutations.SET_ONLINE(state, 12345);
    expect(state.online).toBe(12345);
  });
  
  it("SET_STOREY_DATA updates module data and dynamic", () => {
    mutations.SET_STOREY_DATA(state, { id: 0, data: { num: 5, items: [] } });
    expect(state.module[0].data).toEqual({ num: 5, items: [] });
    expect(state.module[0].dynamic).toBe(5);
  });
  
  it("SET_STOREY_DATA ignores dynamic when num is negative", () => {
    mutations.SET_STOREY_DATA(state, { id: 0, data: { num: -1 } });
    expect(state.module[0].dynamic).toBe(0);
  });
  
  it("SET_STOREY_DATA ignores dynamic when num is not a number", () => {
    mutations.SET_STOREY_DATA(state, { id: 0, data: { num: "abc" } });
    expect(state.module[0].dynamic).toBe(0);
  });
  
  it("SET_RANKING_DATA with original=0 sets rankAllData", () => {
    mutations.SET_RANKING_DATA(state, { id: 0, original: 0, data: [1, 2, 3] });
    expect(state.module[0].rankAllData).toEqual([1, 2, 3]);
  });
  
  it("SET_RANKING_DATA with original=1 sets rankOriginalData", () => {
    mutations.SET_RANKING_DATA(state, { id: 1, original: 1, data: [4, 5] });
    expect(state.module[1].rankOriginalData).toEqual([4, 5]);
  });
  
  it("SET_RANKING_DATA with undefined original sets rankBangumiData", () => {
    mutations.SET_RANKING_DATA(state, { id: 0, data: [6] });
    expect(state.module[0].rankBangumiData).toEqual([6]);
  });
  
  it("SET_TIMELINE_DATA updates timeline data by module id", () => {
    mutations.SET_TIMELINE_DATA(state, { id: 1, data: [{ episode: 1 }] });
    expect(state.module[1].timeline.data).toEqual([{ episode: 1 }]);
  });
  
  it("SET_ELE_OFFSETTOP updates offsetTop by index", () => {
    mutations.SET_ELE_OFFSETTOP(state, { index: 0, data: 150 });
    expect(state.module[0].offsetTop).toBe(150);
  });
});
