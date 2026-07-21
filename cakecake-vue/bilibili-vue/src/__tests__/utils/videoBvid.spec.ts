import { describe, it, expect } from "vitest";
import {
  formatVideoBvid,
  parseVideoIdFromRoute,
  videoPlayRouteAid
} from "@/utils/videoBvid";

describe("formatVideoBvid", () => {
  it("formats numeric id to BV string", () => {
    expect(formatVideoBvid(42)).toBe("BV42");
    expect(formatVideoBvid(100)).toBe("BV100");
  });

  it("returns empty for invalid inputs", () => {
    expect(formatVideoBvid(0)).toBe("");
    expect(formatVideoBvid(-1)).toBe("");
    expect(formatVideoBvid(NaN)).toBe("");
    expect(formatVideoBvid(Infinity)).toBe("");
    expect(formatVideoBvid(null)).toBe("");
    expect(formatVideoBvid(undefined)).toBe("");
    expect(formatVideoBvid("abc")).toBe("");
  });
});

describe("parseVideoIdFromRoute", () => {
  it("parses BV prefix", () => {
    expect(parseVideoIdFromRoute("BV42")).toBe(42);
    expect(parseVideoIdFromRoute("bv100")).toBe(100);
  });

  it("parses av prefix", () => {
    expect(parseVideoIdFromRoute("av42")).toBe(42);
    expect(parseVideoIdFromRoute("AV100")).toBe(100);
  });

  it("parses bare numeric", () => {
    expect(parseVideoIdFromRoute("42")).toBe(42);
  });

  it("returns null for invalid inputs", () => {
    expect(parseVideoIdFromRoute("")).toBeNull();
    expect(parseVideoIdFromRoute(null)).toBeNull();
    expect(parseVideoIdFromRoute(undefined)).toBeNull();
    expect(parseVideoIdFromRoute("abc")).toBeNull();
    expect(parseVideoIdFromRoute("BVabc")).toBeNull();
  });
});

describe("videoPlayRouteAid", () => {
  it("returns BV format for valid id", () => {
    expect(videoPlayRouteAid(42)).toBe("BV42");
  });

  it("returns original string for invalid id", () => {
    expect(videoPlayRouteAid(0)).toBe("0");
    expect(videoPlayRouteAid("abc")).toBe("abc");
  });
});
