import { describe, it, expect } from "vitest";
import {
  isInvalidVideoRouteAid,
  shouldRedirectVideoToNotFound
} from "@/utils/notFoundRedirect";

describe("isInvalidVideoRouteAid", () => {
  it("rejects empty values", () => {
    expect(isInvalidVideoRouteAid("")).toBe(true);
    expect(isInvalidVideoRouteAid(null)).toBe(true);
    expect(isInvalidVideoRouteAid(undefined)).toBe(true);
  });

  it("rejects invalid formats", () => {
    expect(isInvalidVideoRouteAid("abc")).toBe(true);
    expect(isInvalidVideoRouteAid("BVabc")).toBe(true);
  });

  it("rejects zero or negative", () => {
    expect(isInvalidVideoRouteAid("BV0")).toBe(true);
    expect(isInvalidVideoRouteAid("av0")).toBe(true);
    expect(isInvalidVideoRouteAid("0")).toBe(true);
  });

  it("accepts valid BV/av ids", () => {
    expect(isInvalidVideoRouteAid("BV42")).toBe(false);
    expect(isInvalidVideoRouteAid("av42")).toBe(false);
    expect(isInvalidVideoRouteAid("42")).toBe(false);
  });
});

describe("shouldRedirectVideoToNotFound", () => {
  it("returns true for invalid video routes", () => {
    expect(
      shouldRedirectVideoToNotFound({ name: "video", params: { aid: "BVabc" } })
    ).toBe(true);
  });

  it("returns false for valid video routes", () => {
    expect(
      shouldRedirectVideoToNotFound({ name: "video", params: { aid: "BV42" } })
    ).toBe(false);
  });

  it("returns false for non-video routes", () => {
    expect(
      shouldRedirectVideoToNotFound({ name: "home", params: {} })
    ).toBe(false);
  });

  it("handles null route", () => {
    expect(shouldRedirectVideoToNotFound(null)).toBe(false);
  });
});
