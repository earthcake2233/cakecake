import { describe, it, expect } from "vitest";
import {
  SEARCH_ORDER_OPTIONS,
  SEARCH_DURATION_OPTIONS,
  SEARCH_ZONE_OPTIONS,
  DEFAULT_VIDEO_FILTERS,
  videoFiltersToParams
} from "@/utils/searchFilters";

describe("search filter constants", () => {
  it("has all order options", () => {
    expect(SEARCH_ORDER_OPTIONS).toHaveLength(5);
    expect(SEARCH_ORDER_OPTIONS[0].id).toBe("default");
  });

  it("has all duration options", () => {
    expect(SEARCH_DURATION_OPTIONS).toHaveLength(5);
    expect(SEARCH_DURATION_OPTIONS[0].id).toBe("all");
  });

  it("has zone options", () => {
    expect(SEARCH_ZONE_OPTIONS.length).toBeGreaterThan(10);
    expect(SEARCH_ZONE_OPTIONS[0].id).toBe("");
  });

  it("has default filters", () => {
    expect(DEFAULT_VIDEO_FILTERS).toEqual({
      order: "default",
      duration: "all",
      zone: ""
    });
  });
});

describe("videoFiltersToParams", () => {
  it("returns empty params for defaults", () => {
    expect(videoFiltersToParams(DEFAULT_VIDEO_FILTERS)).toEqual({});
  });

  it("includes non-default order", () => {
    expect(
      videoFiltersToParams({ order: "click", duration: "all", zone: "" })
    ).toEqual({ order: "click" });
  });

  it("includes non-default duration", () => {
    expect(
      videoFiltersToParams({ order: "default", duration: "gt60", zone: "" })
    ).toEqual({ duration: "gt60" });
  });

  it("includes zone", () => {
    expect(
      videoFiltersToParams({ order: "default", duration: "all", zone: "动画" })
    ).toEqual({ zone: "动画" });
  });

  it("includes multiple non-default params", () => {
    expect(
      videoFiltersToParams({ order: "click", duration: "gt60", zone: "音乐" })
    ).toEqual({ order: "click", duration: "gt60", zone: "音乐" });
  });

  it("handles null/undefined filters", () => {
    expect(videoFiltersToParams(null)).toEqual({});
    expect(videoFiltersToParams(undefined)).toEqual({});
  });
});
