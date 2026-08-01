import { describe, it, expect } from "vitest";
import {
  levelInfoFromExperience,
  levelFillPct,
  clampUserLevel,
  commentUserLevel,
  USER_LEVEL_MAX
} from "@/utils/userLevel";

describe("levelInfoFromExperience", () => {
  it("returns Lv1 for 0 exp", () => {
    const info = levelInfoFromExperience(0);
    expect(info.current_level).toBe(1);
    expect(info.current_min).toBe(0);
  });

  it("returns correct level based on thresholds", () => {
    expect(levelInfoFromExperience(19).current_level).toBe(1);
    expect(levelInfoFromExperience(20).current_level).toBe(2);
    expect(levelInfoFromExperience(149).current_level).toBe(2);
    expect(levelInfoFromExperience(150).current_level).toBe(3);
    expect(levelInfoFromExperience(2880).current_level).toBe(6);
  });

  it("caps at Lv6", () => {
    const info = levelInfoFromExperience(99999);
    expect(info.current_level).toBe(USER_LEVEL_MAX);
    expect(info.next_exp).toBe(2880);
  });

  it("handles negative values", () => {
    const info = levelInfoFromExperience(-10);
    expect(info.current_level).toBe(1);
  });
});

describe("levelFillPct", () => {
  it("returns 0-100 percentage", () => {
    const info = levelInfoFromExperience(20);
    const pct = levelFillPct(info);
    expect(pct).toBe(0);
  });

  it("returns 100 for max level", () => {
    const info = levelInfoFromExperience(99999);
    expect(levelFillPct(info)).toBe(100);
  });

  it("handles invalid input", () => {
    expect(levelFillPct(null)).toBe(0);
    expect(levelFillPct({})).toBe(100);
  });
});

describe("clampUserLevel", () => {
  it("clamps to 1-6 range", () => {
    expect(clampUserLevel(0)).toBe(1);
    expect(clampUserLevel(1)).toBe(1);
    expect(clampUserLevel(3)).toBe(3);
    expect(clampUserLevel(6)).toBe(6);
    expect(clampUserLevel(10)).toBe(6);
  });

  it("handles invalid input", () => {
    expect(clampUserLevel(null)).toBe(1);
    expect(clampUserLevel("abc")).toBe(1);
  });
});

describe("commentUserLevel", () => {
  it("extracts user_level from row", () => {
    expect(commentUserLevel({ user_level: 5 })).toBe(5);
  });

  it("extracts level_info.current_level fallback", () => {
    expect(
      commentUserLevel({ level_info: { current_level: 3 } })
    ).toBe(3);
  });

  it("defaults to 1 when no level info", () => {
    expect(commentUserLevel({})).toBe(1);
    expect(commentUserLevel(null)).toBe(1);
  });
});
