import { describe, it, expect, vi } from "vitest";
import { showMbDarkToast, MB_DARK_TOAST_CLASS } from "@/utils/mbToast";

vi.mock("element-plus", () => ({
  ElMessage: vi.fn()
}));

describe("mbToast", () => {
  it("exports MB_DARK_TOAST_CLASS constant", () => {
    expect(MB_DARK_TOAST_CLASS).toBe("mb-dark-toast");
  });

  it("showMbDarkToast calls ElMessage with default duration", async () => {
    const { ElMessage } = await import("element-plus");
    showMbDarkToast("test message");
    expect(ElMessage).toHaveBeenCalledWith({
      message: "test message",
      duration: 2000,
      customClass: "mb-dark-toast"
    });
  });

  it("showMbDarkToast accepts custom duration", async () => {
    const { ElMessage } = await import("element-plus");
    showMbDarkToast("custom duration", 5000);
    expect(ElMessage).toHaveBeenCalledWith({
      message: "custom duration",
      duration: 5000,
      customClass: "mb-dark-toast"
    });
  });
});
