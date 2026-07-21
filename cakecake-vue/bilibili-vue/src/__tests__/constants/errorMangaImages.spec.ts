import { describe, it, expect } from "vitest";
import { ERROR_MANGA_IMAGES } from "@/constants/errorMangaImages";

describe("errorMangaImages", () => {
  it("exports an array of 3 images", () => {
    expect(Array.isArray(ERROR_MANGA_IMAGES)).toBe(true);
    expect(ERROR_MANGA_IMAGES).toHaveLength(3);
  });

  it("each entry is a non-empty string (mocked asset path)", () => {
    ERROR_MANGA_IMAGES.forEach(img => {
      expect(typeof img).toBe("string");
      expect(img.length).toBeGreaterThan(0);
    });
  });
});
