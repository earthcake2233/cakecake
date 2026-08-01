import { describe, it, expect } from "vitest";
import {
  MESSAGE_CATEGORIES,
  MESSAGE_CAT_LABELS,
  formatMessageUnreadBadge,
  sumMessageUnread
} from "@/utils/messageCategories";

describe("MESSAGE_CATEGORIES", () => {
  it("has 5 categories", () => {
    expect(MESSAGE_CATEGORIES).toHaveLength(5);
    expect(MESSAGE_CATEGORIES[0].cat).toBe("my_message");
  });
});

describe("MESSAGE_CAT_LABELS", () => {
  it("maps category to label", () => {
    expect(MESSAGE_CAT_LABELS.my_message).toBe("我的消息");
    expect(MESSAGE_CAT_LABELS.reply_received).toBe("回复我的");
  });
});

describe("formatMessageUnreadBadge", () => {
  it("formats single digit", () => {
    expect(formatMessageUnreadBadge(5)).toBe("5");
  });

  it("caps at 99+", () => {
    expect(formatMessageUnreadBadge(100)).toBe("99+");
    expect(formatMessageUnreadBadge(999)).toBe("99+");
  });

  it("returns empty for zero or negative", () => {
    expect(formatMessageUnreadBadge(0)).toBe("");
    expect(formatMessageUnreadBadge(-1)).toBe("");
  });
});

describe("sumMessageUnread", () => {
  it("sums all category counts", () => {
    const summary = {
      my_message: 3,
      reply_received: 5,
      at_me: 1,
      like_aggregation: 2,
      system_notice: 0
    };
    expect(sumMessageUnread(summary)).toBe(11);
  });

  it("handles null/undefined", () => {
    expect(sumMessageUnread(null)).toBe(0);
    expect(sumMessageUnread({})).toBe(0);
  });
});
