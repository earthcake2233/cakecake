import { describe, it, expect } from "vitest";
import {
  stripImageCacheBust,
  withImageCacheBust,
  resolveUserAvatarUrl
} from "@/utils/imageCacheBust";

describe("stripImageCacheBust", () => {
  it("removes v= parameter", () => {
    expect(stripImageCacheBust("https://example.com/img.jpg?v=123")).toBe(
      "https://example.com/img.jpg"
    );
  });

  it("removes only v= parameter among others", () => {
    expect(
      stripImageCacheBust("https://example.com/img.jpg?w=100&v=123&h=200")
    ).toBe("https://example.com/img.jpg?w=100&h=200");
  });

  it("returns unchanged if no v= parameter", () => {
    expect(stripImageCacheBust("https://example.com/img.jpg")).toBe(
      "https://example.com/img.jpg"
    );
  });

  it("handles empty input", () => {
    expect(stripImageCacheBust("")).toBe("");
    expect(stripImageCacheBust(null)).toBe("");
    expect(stripImageCacheBust(undefined)).toBe("");
  });
});

describe("withImageCacheBust", () => {
  it("appends v= parameter", () => {
    expect(withImageCacheBust("https://example.com/img.jpg", 456)).toBe(
      "https://example.com/img.jpg?v=456"
    );
  });

  it("appends to existing query string", () => {
    expect(withImageCacheBust("https://example.com/img.jpg?w=100", 456)).toBe(
      "https://example.com/img.jpg?w=100&v=456"
    );
  });

  it("returns empty for empty input", () => {
    expect(withImageCacheBust("", 123)).toBe("");
  });

  it("returns base without bust when bust is falsy", () => {
    expect(withImageCacheBust("https://example.com/img.jpg", 0)).toBe(
      "https://example.com/img.jpg"
    );
    expect(withImageCacheBust("https://example.com/img.jpg", null)).toBe(
      "https://example.com/img.jpg"
    );
  });
});

describe("resolveUserAvatarUrl", () => {
  it("uses existing v= parameter", () => {
    expect(resolveUserAvatarUrl("https://example.com/avatar.jpg?v=999")).toBe(
      "https://example.com/avatar.jpg?v=999"
    );
  });

  it("adds bust when no v= parameter exists", () => {
    expect(resolveUserAvatarUrl("https://example.com/avatar.jpg", 777)).toBe(
      "https://example.com/avatar.jpg?v=777"
    );
  });

  it("returns empty for empty input", () => {
    expect(resolveUserAvatarUrl("")).toBe("");
  });
});
