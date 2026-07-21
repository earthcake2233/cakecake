import { describe, it, expect } from "vitest";
import {
  RANKING_RID_ZONE,
  normalizeRankDays,
  rankCompositeScore,
  rankSortScore,
  mapVideoToRankListItem
} from "@/utils/rankingFeeds";

describe("RANKING_RID_ZONE", () => {
  it("maps rid 0 to empty string", () => {
    expect(RANKING_RID_ZONE["0"]).toBe("");
  });

  it("maps known rids", () => {
    expect(RANKING_RID_ZONE["1"]).toBe("动画");
    expect(RANKING_RID_ZONE["3"]).toBe("音乐");
    expect(RANKING_RID_ZONE["4"]).toBe("游戏");
  });
});

describe("normalizeRankDays", () => {
  it("passes valid days through", () => {
    expect(normalizeRankDays(1)).toBe(1);
    expect(normalizeRankDays(3)).toBe(3);
    expect(normalizeRankDays(7)).toBe(7);
    expect(normalizeRankDays(30)).toBe(30);
  });

  it("defaults invalid days to 3", () => {
    expect(normalizeRankDays(0)).toBe(3);
    expect(normalizeRankDays(2)).toBe(3);
    expect(normalizeRankDays("abc")).toBe(3);
  });
});

describe("rankCompositeScore", () => {
  it("computes weighted score", () => {
    const v = {
      play_count: 1000,
      danmaku_count: 50,
      like_count: 200,
      fav_count: 100,
      coin_count: 50,
      comment_count: 30
    };
    const expected = Math.round(
      1000 * 1.2 + 50 * 85 + 200 * 120 + 100 * 90 + 50 * 200 + 30 * 60
    );
    expect(rankCompositeScore(v)).toBe(expected);
  });

  it("handles missing fields as 0", () => {
    expect(rankCompositeScore({})).toBe(0);
  });
});

describe("rankSortScore", () => {
  it("applies time decay for old videos", () => {
    const v = { play_count: 100, created_at: "2020-01-01" };
    const score = rankSortScore(v, { arcType: 0, days: 3 });
    expect(score).toBeLessThan(120);
  });

  it("no decay for arcType 1", () => {
    const v = { play_count: 100, created_at: "2020-01-01" };
    const score = rankSortScore(v, { arcType: 1, days: 3 });
    expect(score).toBe(120);
  });
});

describe("mapVideoToRankListItem", () => {
  it("maps video fields correctly", () => {
    const v = {
      id: 42,
      title: "测试视频",
      cover_url: "https://example.com/cover.jpg",
      play_count: 1000,
      danmaku_count: 50,
      uploader: "测试UP",
      user_id: 1
    };
    const item = mapVideoToRankListItem(v, null);
    expect(item.aid).toBe(42);
    expect(item.title).toBe("测试视频");
    expect(item.author).toBe("测试UP");
    expect(item.play).toBe(1000);
    expect(item.pts).toBeGreaterThan(0);
  });

  it("handles missing fields with defaults", () => {
    const item = mapVideoToRankListItem({ id: 1 }, null);
    expect(item.title).toBe("未命名稿件");
    expect(item.author).toBe("未知UP主");
    expect(item.play).toBe(0);
  });
});
