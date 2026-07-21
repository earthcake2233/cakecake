import { describe, it, expect } from "vitest";
import {
  MB_COMMENT_CURATED_LABEL,
  MB_COMMENT_CURATED_PLACEHOLDER,
  MB_COMMENT_PENDING_TOAST
} from "@/constants/minibiliComments";

describe("minibiliComments constants", () => {
  it("exports MB_COMMENT_CURATED_LABEL", () => {
    expect(MB_COMMENT_CURATED_LABEL).toBeTruthy();
    expect(typeof MB_COMMENT_CURATED_LABEL).toBe("string");
  });

  it("exports MB_COMMENT_CURATED_PLACEHOLDER", () => {
    expect(MB_COMMENT_CURATED_PLACEHOLDER).toBeTruthy();
    expect(typeof MB_COMMENT_CURATED_PLACEHOLDER).toBe("string");
  });

  it("exports MB_COMMENT_PENDING_TOAST", () => {
    expect(MB_COMMENT_PENDING_TOAST).toBeTruthy();
    expect(typeof MB_COMMENT_PENDING_TOAST).toBe("string");
  });
});
