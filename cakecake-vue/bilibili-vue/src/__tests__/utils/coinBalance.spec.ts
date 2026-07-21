import { describe, it, expect } from "vitest";
import { formatCoinBalance, coinBalanceNumber } from "@/utils/coinBalance";

describe("formatCoinBalance", () => {
  it("formats integer values", () => {
    expect(formatCoinBalance(100)).toBe("100");
    expect(formatCoinBalance(0)).toBe("0");
  });

  it("formats one decimal place", () => {
    expect(formatCoinBalance(10.5)).toBe("10.5");
    expect(formatCoinBalance(1.1)).toBe("1.1");
  });

  it("rounds to one decimal", () => {
    expect(formatCoinBalance(10.55)).toBe("10.6");
    expect(formatCoinBalance(10.54)).toBe("10.5");
  });

  it("drops .0 when not needed", () => {
    expect(formatCoinBalance(10.0)).toBe("10");
    expect(formatCoinBalance(100.04)).toBe("100");
  });

  it("handles invalid values", () => {
    expect(formatCoinBalance(-1)).toBe("0");
    expect(formatCoinBalance(null)).toBe("0");
    expect(formatCoinBalance("abc")).toBe("0");
  });
});

describe("coinBalanceNumber", () => {
  it("returns rounded number", () => {
    expect(coinBalanceNumber(10.55)).toBe(10.6);
    expect(coinBalanceNumber(100)).toBe(100);
  });

  it("handles invalid values", () => {
    expect(coinBalanceNumber(-1)).toBe(0);
    expect(coinBalanceNumber(null)).toBe(0);
  });
});
