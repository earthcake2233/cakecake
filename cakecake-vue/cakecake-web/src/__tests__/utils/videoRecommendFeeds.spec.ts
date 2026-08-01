import { describe, it, expect } from "vitest";
import {
  filterRecommendPool,
  interleaveRecommendPools,
  formatRecommendDuration,
  fillHomeRecommendSlots,
  nextHomeRecommendBatch,
  zoneParentFromDetail,
  HOME_RECOMMEND_PAGE_SIZE
} from "@/utils/videoRecommendFeeds";

describe("filterRecommendPool", () => {
  it("removes excludeVideoId and duplicates", () => {
    const items = [
      { id: 1 }, { id: 2 }, { id: 1 }, { id: 2 }, { id: 3 }
    ];
    expect(filterRecommendPool(items, 2)).toEqual([
      { id: 1 }, { id: 3 }
    ]);
  });

  it("filters invalid ids", () => {
    expect(filterRecommendPool([{ id: 0 }, { id: -1 }, {}], 0)).toEqual([]);
  });

  it("handles empty input", () => {
    expect(filterRecommendPool([], 1)).toEqual([]);
    expect(filterRecommendPool(null, 1)).toEqual([]);
  });
});

describe("interleaveRecommendPools", () => {
  it("interleaves two pools without duplicates", () => {
    const a = [{ id: 1 }, { id: 3 }];
    const b = [{ id: 2 }, { id: 1 }];
    expect(interleaveRecommendPools(a, b)).toEqual([
      { id: 1 }, { id: 2 }, { id: 3 }
    ]);
  });

  it("handles empty pools", () => {
    expect(interleaveRecommendPools([{ id: 1 }], [])).toEqual([{ id: 1 }]);
    expect(interleaveRecommendPools(null, null)).toEqual([]);
  });
});

describe("formatRecommendDuration", () => {
  it("formats seconds", () => {
    expect(formatRecommendDuration(0)).toBe("0:00");
    expect(formatRecommendDuration(61)).toBe("1:01");
    expect(formatRecommendDuration(3661)).toBe("1:01:01");
  });

  it("handles invalid input", () => {
    expect(formatRecommendDuration(-1)).toBe("0:00");
    expect(formatRecommendDuration(null)).toBe("0:00");
  });
});

describe("fillHomeRecommendSlots", () => {
  it("fills exactly HOME_RECOMMEND_PAGE_SIZE items", () => {
    const pool = [{ id: 1 }, { id: 2 }, { id: 3 }];
    const slots = fillHomeRecommendSlots(pool, 0, 8);
    expect(slots).toHaveLength(8);
  });

  it("cycles through pool when not enough items", () => {
    const pool = [{ id: 1 }, { id: 2 }];
    const slots = fillHomeRecommendSlots(pool, 1, 3);
    expect(slots).toHaveLength(3);
    expect(slots[0].id).toBe(2);
    expect(slots[2].id).toBe(2);
  });

  it("returns empty for empty pool", () => {
    expect(fillHomeRecommendSlots([], 0, 8)).toEqual([]);
  });
});

describe("nextHomeRecommendBatch", () => {
  it("returns next batch with offset", () => {
    const pool = [{ id: 1 }, { id: 2 }, { id: 3 }, { id: 4 }, { id: 5 },
      { id: 6 }, { id: 7 }, { id: 8 }, { id: 9 }, { id: 10 }];
    const result = nextHomeRecommendBatch(pool, [], 0, 1);
    expect(result.items).toHaveLength(8);
    expect(result.nextOffset).toBe(8);
  });

  it("handles empty pool", () => {
    const result = nextHomeRecommendBatch([], [], 0, 1);
    expect(result.items).toEqual([]);
    expect(result.nextOffset).toBe(0);
  });
});

describe("zoneParentFromDetail", () => {
  it("extracts parent from zone_parent", () => {
    expect(zoneParentFromDetail({ zone_parent: "动画" })).toBe("动画");
  });

  it("parses parent from zone string", () => {
    expect(zoneParentFromDetail({ zone: "生活-日常" })).toBe("生活");
  });

  it("returns empty for null", () => {
    expect(zoneParentFromDetail(null)).toBe("");
    expect(zoneParentFromDetail({})).toBe("");
  });
});
