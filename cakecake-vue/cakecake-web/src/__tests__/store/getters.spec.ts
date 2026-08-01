import { describe, it, expect } from "vitest";
import * as getters from "@/store/getters";

const mockState = {
  scrollTop: 100,
  slide: { time: 4000, pagation: true, data: [] },
  recommend: { rec: [], day: 3 },
  popularize: [],
  online: [],
  module: [
    { id: 0, title: "动画" },
    { id: 1, title: "番剧" },
    { id: 2, title: "国创" }
  ]
};

describe("store getters", () => {
  it("scrollTop returns state.scrollTop", () => {
    expect(getters.scrollTop(mockState)).toBe(100);
  });

  it("slide returns state.slide", () => {
    expect(getters.slide(mockState)).toBe(mockState.slide);
  });

  it("recommend returns state.recommend", () => {
    expect(getters.recommend(mockState)).toBe(mockState.recommend);
  });

  it("popularize returns state.popularize", () => {
    expect(getters.popularize(mockState)).toBe(mockState.popularize);
  });

  it("online returns state.online", () => {
    expect(getters.online(mockState)).toBe(mockState.online);
  });

  it("donghua returns module[0]", () => {
    expect(getters.donghua(mockState)).toBe(mockState.module[0]);
  });

  it("bangumi returns module[1]", () => {
    expect(getters.bangumi(mockState)).toBe(mockState.module[1]);
  });

  it("guochuang returns module[2]", () => {
    expect(getters.guochuang(mockState)).toBe(mockState.module[2]);
  });

  it("module returns state.module", () => {
    expect(getters.module(mockState)).toBe(mockState.module);
  });
});
