import { describe, it, expect, beforeEach } from "vitest";
import {
  DM_FONT_SIZE_PX, DM_SEND_FONT_PREF_KEY,
  dmLineHeightForFontPx, dmNormalizeFontSizeKey,
  dmFontPxFromKey, dmCanvasFontCss,
  dmStrokeWidthForPx, dmDensityFactor,
  dmScrollLanesForDensity, dmStackSlotsForDensity,
  dmMaxActiveBullets,
  loadDmSendFontSizePref, saveDmSendFontSizePref
} from "@/utils/danmakuDisplay";

describe("DM_FONT_SIZE_PX", () => {
  it("has sm/md/lg sizes", () => {
    expect(DM_FONT_SIZE_PX.sm).toBe(16);
    expect(DM_FONT_SIZE_PX.md).toBe(18);
    expect(DM_FONT_SIZE_PX.lg).toBe(24);
  });
});

describe("DM_SEND_FONT_PREF_KEY", () => {
  it("is the localStorage key", () => {
    expect(DM_SEND_FONT_PREF_KEY).toBe("mb_dm_send_font_size");
  });
});

describe("dmLineHeightForFontPx", () => {
  it("computes line height", () => {
    expect(dmLineHeightForFontPx(18)).toBe(26);
    expect(dmLineHeightForFontPx(16)).toBe(23);
  });
  it("floors at 20", () => {
    expect(dmLineHeightForFontPx(1)).toBe(20);
  });
  it("defaults to md size for invalid input", () => {
    expect(dmLineHeightForFontPx(null)).toBe(26);
    expect(dmLineHeightForFontPx(undefined)).toBe(26);
  });
});

describe("dmNormalizeFontSizeKey", () => {
  it("normalizes valid keys", () => {
    expect(dmNormalizeFontSizeKey("sm")).toBe("sm");
    expect(dmNormalizeFontSizeKey("md")).toBe("md");
    expect(dmNormalizeFontSizeKey("lg")).toBe("lg");
  });
  it("handles aliases", () => {
    expect(dmNormalizeFontSizeKey("small")).toBe("sm");
    expect(dmNormalizeFontSizeKey("large")).toBe("lg");
    expect(dmNormalizeFontSizeKey("SMALL")).toBe("sm");
  });
  it("defaults unknown to md", () => {
    expect(dmNormalizeFontSizeKey("xl")).toBe("md");
    expect(dmNormalizeFontSizeKey("")).toBe("md");
    expect(dmNormalizeFontSizeKey(null)).toBe("md");
  });
});

describe("dmFontPxFromKey", () => {
  it("maps keys to px", () => {
    expect(dmFontPxFromKey("sm")).toBe(16);
    expect(dmFontPxFromKey("md")).toBe(18);
    expect(dmFontPxFromKey("lg")).toBe(24);
  });
  it("falls back to md for unknown", () => {
    expect(dmFontPxFromKey("xl")).toBe(18);
  });
});

describe("dmCanvasFontCss", () => {
  it("builds from key", () => {
    expect(dmCanvasFontCss("md")).toContain("18px");
    expect(dmCanvasFontCss("sm")).toContain("16px");
  });
  it("builds from pixel", () => {
    expect(dmCanvasFontCss(20)).toContain("20px");
    expect(dmCanvasFontCss(0)).toContain("18px");
    expect(dmCanvasFontCss(-1)).toContain("18px");
  });
  it("returns bold css", () => {
    expect(dmCanvasFontCss("md")).toContain("bold");
  });
});

describe("dmStrokeWidthForPx", () => {
  it("calculates stroke", () => {
    expect(dmStrokeWidthForPx(18)).toBe(3);
    expect(dmStrokeWidthForPx(24)).toBe(4);
  });
  it("minimum 2", () => {
    expect(dmStrokeWidthForPx(1)).toBe(2);
  });
  it("handles null/undefined", () => {
    expect(dmStrokeWidthForPx(null)).toBe(3);
    expect(dmStrokeWidthForPx(undefined)).toBe(3);
  });
});

describe("dmDensityFactor", () => {
  it("maps 0-100 to 0.15-1.0", () => {
    expect(dmDensityFactor(0)).toBeCloseTo(0.15);
    expect(dmDensityFactor(50)).toBeCloseTo(0.575);
    expect(dmDensityFactor(100)).toBeCloseTo(1.0);
  });
  it("clamps out-of-range", () => {
    expect(dmDensityFactor(-10)).toBeCloseTo(0.15);
    expect(dmDensityFactor(200)).toBeCloseTo(1.0);
  });
  it("handles NaN", () => {
    expect(dmDensityFactor(NaN)).toBeCloseTo(0.9);
    expect(dmDensityFactor("abc")).toBeCloseTo(0.9);
  });
});

describe("dmScrollLanesForDensity", () => {
  it("scales lanes", () => {
    expect(dmScrollLanesForDensity(10, 100)).toBe(10);
    expect(dmScrollLanesForDensity(10, 0)).toBe(2);
  });
  it("minimum 1", () => {
    expect(dmScrollLanesForDensity(1, 0)).toBe(1);
    expect(dmScrollLanesForDensity(0, 100)).toBe(1);
  });
});

describe("dmStackSlotsForDensity", () => {
  it("scales slots", () => {
    expect(dmStackSlotsForDensity(10, 100)).toBe(10);
    expect(dmStackSlotsForDensity(10, 0)).toBe(2);
  });
  it("minimum 1", () => {
    expect(dmStackSlotsForDensity(0, 50)).toBe(1);
  });
});

describe("dmMaxActiveBullets", () => {
  it("returns at least 4", () => {
    expect(dmMaxActiveBullets(0, 80, 1)).toBeGreaterThanOrEqual(4);
  });
  it("returns at most 100", () => {
    expect(dmMaxActiveBullets(100, 2000, 50)).toBeLessThanOrEqual(100);
  });
  it("scales with density", () => {
    const low = dmMaxActiveBullets(0, 360, 10);
    const high = dmMaxActiveBullets(100, 360, 10);
    expect(low).toBeLessThanOrEqual(high);
  });
  it("handles defaults for invalid inputs", () => {
    const r = dmMaxActiveBullets(NaN, null, undefined);
    expect(r).toBeGreaterThanOrEqual(4);
    expect(r).toBeLessThanOrEqual(100);
  });
});

describe("loadDmSendFontSizePref", () => {
  beforeEach(() => localStorage.clear());
  it("returns md when nothing stored", () => {
    expect(loadDmSendFontSizePref()).toBe("md");
  });
  it("reads stored preference", () => {
    localStorage.setItem(DM_SEND_FONT_PREF_KEY, "lg");
    expect(loadDmSendFontSizePref()).toBe("lg");
  });
  it("ignores invalid stored value", () => {
    localStorage.setItem(DM_SEND_FONT_PREF_KEY, "xl");
    expect(loadDmSendFontSizePref()).toBe("md");
  });
});

describe("saveDmSendFontSizePref", () => {
  beforeEach(() => localStorage.clear());
  it("saves valid key", () => {
    saveDmSendFontSizePref("lg");
    expect(localStorage.getItem(DM_SEND_FONT_PREF_KEY)).toBe("lg");
  });
  it("normalizes before saving", () => {
    saveDmSendFontSizePref("large");
    expect(localStorage.getItem(DM_SEND_FONT_PREF_KEY)).toBe("lg");
  });
  it("defaults unknown to md", () => {
    saveDmSendFontSizePref("xl");
    expect(localStorage.getItem(DM_SEND_FONT_PREF_KEY)).toBe("md");
  });
});
