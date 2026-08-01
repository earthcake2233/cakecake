import { describe, it, expect } from "vitest";
import { formatDuration } from "@/utils/formatDuration";

describe("formatDuration", () => {
  it("formats seconds under 1 minute", () => {
    expect(formatDuration(0)).toBe("0:00");
    expect(formatDuration(5)).toBe("0:05");
    expect(formatDuration(59)).toBe("0:59");
  });

  it("formats minutes and seconds", () => {
    expect(formatDuration(60)).toBe("1:00");
    expect(formatDuration(61)).toBe("1:01");
    expect(formatDuration(3661)).toBe("1:01:01");
  });

  it("pads single-digit minutes/hours", () => {
    expect(formatDuration(600)).toBe("10:00");
    expect(formatDuration(3630)).toBe("1:00:30");
  });

  it("handles edge cases", () => {
    expect(formatDuration(-1)).toBe("0:00");
    expect(formatDuration(null)).toBe("0:00");
    expect(formatDuration(undefined)).toBe("0:00");
    expect(formatDuration("abc")).toBe("0:00");
  });

  it("formats large durations", () => {
    expect(formatDuration(3600)).toBe("1:00:00");
    expect(formatDuration(86399)).toBe("23:59:59");
  });
});
