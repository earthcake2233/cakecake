import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  searchKeywordNorm,
  loadSearchHistory,
  addSearchHistory,
  removeSearchHistoryAt
} from "@/utils/searchHistory";

vi.mock("@/api/minibili", () => ({
  mbAddMySearchHistory: vi.fn().mockResolvedValue({ keywords: ["mock1", "mock2"] }),
  mbGetMySearchHistory: vi.fn().mockResolvedValue({ keywords: [] }),
  mbPutMySearchHistory: vi.fn().mockResolvedValue(undefined)
}));

const STORAGE_KEY = "minibili_search_history";

describe("searchKeywordNorm", () => {
  it("lowercases and trims", () => {
    expect(searchKeywordNorm("  Hello  ")).toBe("hello");
    expect(searchKeywordNorm("ABC")).toBe("abc");
  });
  it("removes internal spaces", () => {
    expect(searchKeywordNorm("hello world")).toBe("helloworld");
  });
  it("handles empty and null", () => {
    expect(searchKeywordNorm("")).toBe("");
    expect(searchKeywordNorm(null)).toBe("");
    expect(searchKeywordNorm(undefined)).toBe("");
  });
});

describe("loadSearchHistory", () => {
  beforeEach(() => localStorage.clear());
  it("returns empty array when nothing stored", () => {
    expect(loadSearchHistory()).toEqual([]);
  });
  it("loads stored keywords", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(["vue", "react"]));
    expect(loadSearchHistory()).toEqual(["vue", "react"]);
  });
  it("handles corrupted JSON", () => {
    localStorage.setItem(STORAGE_KEY, "not-json");
    expect(loadSearchHistory()).toEqual([]);
  });
  it("deduplicates and normalizes", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(["Vue", "vue", "  VUE  "]));
    expect(loadSearchHistory()).toEqual(["Vue"]);
  });
  it("respects MAX_ITEMS limit (20)", () => {
    const many = Array.from({ length: 30 }, (_, i) => `key-${i}`);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(many));
    expect(loadSearchHistory().length).toBe(20);
  });
  it("filters empty strings", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(["a", "", "b", "  "]));
    expect(loadSearchHistory()).toEqual(["a", "b"]);
  });
});

describe("addSearchHistory", () => {
  beforeEach(() => { localStorage.clear(); vi.clearAllMocks(); });
  it("adds keyword to empty history", () => {
    expect(addSearchHistory("hello")).toEqual(["hello"]);
  });
  it("prepends new keyword", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(["old"]));
    expect(addSearchHistory("new")).toEqual(["new", "old"]);
  });
  it("deduplicates by normalized form", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(["Vue"]));
    expect(addSearchHistory("vue")).toEqual(["vue"]);
  });
  it("returns current history for empty keyword", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(["a"]));
    expect(addSearchHistory("")).toEqual(["a"]);
  });
  it("trims whitespace from keyword", () => {
    expect(addSearchHistory("  spaced  ")).toEqual(["spaced"]);
  });
  it("does not exceed 20 items", () => {
    const items = Array.from({ length: 20 }, (_, i) => `key-${i}`);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
    expect(addSearchHistory("newest")[0]).toBe("newest");
  });
  it("calls mbAddMySearchHistory when authenticated", async () => {
    const { setTokens } = await import("@/utils/authTokens");
    setTokens("valid-token", "rt");
    addSearchHistory("term");
    const { mbAddMySearchHistory } = await import("@/api/minibili");
    expect(mbAddMySearchHistory).toHaveBeenCalledWith("term");
  });
});

describe("removeSearchHistoryAt", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
    localStorage.setItem(STORAGE_KEY, JSON.stringify(["a", "b", "c"]));
  });
  it("removes item at valid index", () => {
    expect(removeSearchHistoryAt(1)).toEqual(["a", "c"]);
  });
  it("returns unchanged for negative index", () => {
    expect(removeSearchHistoryAt(-1)).toEqual(["a", "b", "c"]);
  });
  it("returns unchanged for out-of-bounds index", () => {
    expect(removeSearchHistoryAt(99)).toEqual(["a", "b", "c"]);
  });
  it("handles non-numeric index", () => {
    expect(removeSearchHistoryAt("invalid")).toEqual(["a", "b", "c"]);
  });
  it("persists after removal", () => {
    removeSearchHistoryAt(0);
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY))).toEqual(["b", "c"]);
  });
});
