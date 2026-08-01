import { describe, it, expect } from "vitest";
import { DYNAMICS_HOT_SEARCH } from "@/utils/dynamicsFeedSeed";

describe("dynamicsFeedSeed", () => {
  it("exports 10 hot search items", () => {
    expect(DYNAMICS_HOT_SEARCH).toHaveLength(10);
  });

  it("each item has rank, title, and badge fields", () => {
    DYNAMICS_HOT_SEARCH.forEach((item, i) => {
      expect(item).toHaveProperty("rank");
      expect(item).toHaveProperty("title");
      expect(item).toHaveProperty("badge");
      expect(typeof item.rank).toBe("number");
      expect(typeof item.title).toBe("string");
      expect(typeof item.badge).toBe("string");
    });
  });

  it("ranks are sequential from 1 to 10", () => {
    DYNAMICS_HOT_SEARCH.forEach((item, i) => {
      expect(item.rank).toBe(i + 1);
    });
  });

  it("titles are non-empty strings", () => {
    DYNAMICS_HOT_SEARCH.forEach(item => {
      expect(item.title.length).toBeGreaterThan(0);
    });
  });
});
