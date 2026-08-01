import { describe, it, expect } from "vitest";
import {
  SITE_BRAND,
  SITE_HOME_TITLE,
  buildDocumentTitle
} from "@/constants/siteTitle";

describe("SITE_BRAND", () => {
  it("is cakecake", () => {
    expect(SITE_BRAND).toBe("cakecake");
  });
});

describe("buildDocumentTitle", () => {
  it("returns home title for empty input", () => {
    expect(buildDocumentTitle("")).toBe(SITE_HOME_TITLE);
  });

  it("formats page title with suffix", () => {
    const result = buildDocumentTitle("首页");
    expect(result).toContain("首页");
    expect(result).toContain(SITE_HOME_TITLE);
  });
});
