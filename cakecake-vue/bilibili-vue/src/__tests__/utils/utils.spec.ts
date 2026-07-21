import { describe, it, expect } from "vitest";
import { count, count2, timeChange, stringChange } from "@/utils/utils";

describe("count2", () => {
  it("formats numbers above 10000", () => {
    expect(count2(10001)).toBe("1.0万");
    expect(count2(12345)).toBe("1.2万");
    expect(count2(100000)).toBe("10.0万");
  });

  it("returns original number at or below 10000", () => {
    expect(count2(0)).toBe(0);
    expect(count2(10000)).toBe(10000);
    expect(count2(9999)).toBe(9999);
    expect(count2(100)).toBe(100);
  });
});

describe("count (seconds formatter)", () => {
  it("formats seconds under 60 with padded seconds", () => {
    expect(count(0)).toBe("00:00");
    expect(count(5)).toBe("00:05");
    expect(count(59)).toBe("00:59");
  });

  it("formats minutes", () => {
    expect(count(60)).toBe("01:00");
    expect(count(61)).toBe("01:01");
  });

  it("formats hours", () => {
    expect(count(3600)).toBe("1:00:00");
    expect(count(3661)).toBe("1:01:01");
    expect(count(7322)).toBe("2:02:02");
  });
});

describe("timeChange", () => {
  it("converts timestamp to date string without trailing space", () => {
    const result = timeChange(1672531200);
    expect(result).toBe("2023-01-01");
  });
});

describe("stringChange", () => {
  it("extracts substring from index 30", () => {
    const url = "https://example.com/videos/BV42__";
    expect(stringChange(url)).toBe("/video/2__");
  });
});
