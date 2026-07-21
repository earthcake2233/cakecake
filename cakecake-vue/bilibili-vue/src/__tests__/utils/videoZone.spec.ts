import { describe, it, expect } from "vitest";
import { parseVideoZone, videoZoneCrumbs } from "@/utils/videoZone";

describe("parseVideoZone", () => {
  it("parses zone with parent-child", () => {
    const result = parseVideoZone("生活-日常");
    expect(result.zone).toBe("生活-日常");
    expect(result.parent).toBe("生活");
    expect(result.child).toBe("日常");
    expect(result.category).toBe("生活 > 日常");
  });

  it("parses parent-only zone", () => {
    const result = parseVideoZone("动画");
    expect(result.zone).toBe("动画");
    expect(result.parent).toBe("动画");
    expect(result.child).toBe("");
    expect(result.category).toBe("动画");
  });

  it("handles empty input", () => {
    const result = parseVideoZone("");
    expect(result.zone).toBe("");
    expect(result.parent).toBe("");
    expect(result.child).toBe("");
    expect(result.category).toBe("");
  });

  it("normalizes arrow separators", () => {
    const result = parseVideoZone("动画 → MAD·AMV");
    expect(result.parent).toBe("动画");
    expect(result.child).toBe("MAD·AMV");
  });

  it("trims whitespace around separator", () => {
    const result = parseVideoZone(" 音乐 - 翻唱 ");
    expect(result.parent).toBe("音乐");
    expect(result.child).toBe("翻唱");
  });
});

describe("videoZoneCrumbs", () => {
  it("builds breadcrumbs for parent-child zone", () => {
    const crumbs = videoZoneCrumbs("生活-日常");
    expect(crumbs).toHaveLength(3);
    expect(crumbs[0].key).toBe("home");
    expect(crumbs[1].label).toBe("生活");
    expect(crumbs[2].label).toBe("日常");
    expect(crumbs[2].last).toBe(true);
  });

  it("builds breadcrumbs for parent-only zone", () => {
    const crumbs = videoZoneCrumbs("动画");
    expect(crumbs).toHaveLength(2);
    expect(crumbs[1].label).toBe("动画");
    expect(crumbs[1].last).toBe(true);
  });

  it("returns empty for empty input", () => {
    expect(videoZoneCrumbs("")).toEqual([]);
  });
});
