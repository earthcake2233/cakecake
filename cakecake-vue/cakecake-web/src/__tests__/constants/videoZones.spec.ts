import { describe, it, expect } from "vitest";
import {
  VIDEO_ZONE_CATEGORIES,
  VIDEO_ZONE_MENU_NUMS,
  buildHeaderMenuLeftZones,
  formatVideoZoneLabel,
  buildVideoZoneSelectOptions,
  buildVideoZoneSelectGroups,
  isKnownVideoZone,
  normalizeVideoZoneValue
} from "@/constants/videoZones";

describe("VIDEO_ZONE_CATEGORIES", () => {
  it("has all expected categories", () => {
    const names = VIDEO_ZONE_CATEGORIES.map(c => c.name);
    expect(names).toContain("动画");
    expect(names).toContain("音乐");
    expect(names).toContain("游戏");
    expect(names).toContain("科技");
    expect(names).toContain("生活");
    expect(names).toContain("放映厅");
  });

  it("each category has items array", () => {
    for (const cat of VIDEO_ZONE_CATEGORIES) {
      expect(Array.isArray(cat.items)).toBe(true);
    }
  });

  it("广告 category has empty items", () => {
    const ad = VIDEO_ZONE_CATEGORIES.find(c => c.name === "广告");
    expect(ad.items).toEqual([]);
  });
});

describe("VIDEO_ZONE_MENU_NUMS", () => {
  it("has positive numbers for active zones", () => {
    expect(VIDEO_ZONE_MENU_NUMS["动画"]).toBeGreaterThan(0);
    expect(VIDEO_ZONE_MENU_NUMS["音乐"]).toBeGreaterThan(0);
  });
});

describe("buildHeaderMenuLeftZones", () => {
  it("builds menu structure with correct shape", () => {
    const menu = buildHeaderMenuLeftZones();
    expect(menu.length).toBe(VIDEO_ZONE_CATEGORIES.length);
    expect(menu[0].name).toBe("动画");
    expect(menu[0].num).toBe(VIDEO_ZONE_MENU_NUMS["动画"]);
    expect(Array.isArray(menu[0].items)).toBe(true);
  });
});

describe("formatVideoZoneLabel", () => {
  it("formats parent-child zone", () => {
    expect(formatVideoZoneLabel("生活-日常")).toBe("生活 / 日常");
  });

  it("returns raw value for parent-only", () => {
    expect(formatVideoZoneLabel("动画")).toBe("动画");
  });

  it("handles empty input", () => {
    expect(formatVideoZoneLabel("")).toBe("");
    expect(formatVideoZoneLabel(null)).toBe("");
  });
});

describe("buildVideoZoneSelectOptions", () => {
  it("includes all categories and sub-items", () => {
    const options = buildVideoZoneSelectOptions();
    expect(options.length).toBeGreaterThan(VIDEO_ZONE_CATEGORIES.length);
    expect(options.some(o => o.value === "动画")).toBe(true);
    expect(options.some(o => o.value === "动画-MAD·AMV")).toBe(true);
  });
});

describe("buildVideoZoneSelectGroups", () => {
  it("groups options by category", () => {
    const groups = buildVideoZoneSelectGroups();
    expect(groups.length).toBe(VIDEO_ZONE_CATEGORIES.length);
    expect(groups[0].options[0].value).toBe("动画");
  });
});

describe("isKnownVideoZone", () => {
  it("recognizes valid zones", () => {
    expect(isKnownVideoZone("动画")).toBe(true);
    expect(isKnownVideoZone("生活-日常")).toBe(true);
    expect(isKnownVideoZone("音乐-翻唱")).toBe(true);
  });

  it("rejects invalid zones", () => {
    expect(isKnownVideoZone("")).toBe(false);
    expect(isKnownVideoZone("不存在的分区")).toBe(false);
  });
});

describe("normalizeVideoZoneValue", () => {
  it("normalizes arrow separators", () => {
    expect(normalizeVideoZoneValue("动画 → MAD·AMV")).toBe("动画-MAD·AMV");
  });

  it("returns empty for invalid value", () => {
    expect(normalizeVideoZoneValue("不存在")).toBe("");
  });
});
