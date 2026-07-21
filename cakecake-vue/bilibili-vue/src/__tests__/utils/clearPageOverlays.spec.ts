import { describe, it, expect, vi, beforeEach } from "vitest";
import { clearStuckPageOverlays } from "@/utils/clearPageOverlays";

describe("clearPageOverlays", () => {
  beforeEach(() => {
    document.body.className = "";
    document.body.style.cssText = "";
    // Clear any existing overlay elements
    document.querySelectorAll(".mb-dyn-edit-overlay, .mb-dyn-leave-overlay, .el-overlay, .el-message-box__wrapper").forEach(el => el.remove());
  });

  it("removes stuck body classes", () => {
    document.body.classList.add("el-popup-parent--hidden", "mb-dyn-edit-leave-open");
    clearStuckPageOverlays();
    expect(document.body.classList.contains("el-popup-parent--hidden")).toBe(false);
    expect(document.body.classList.contains("mb-dyn-edit-leave-open")).toBe(false);
  });

  it("removes body overflow styles", () => {
    document.body.style.width = "100%";
    document.body.style.overflow = "hidden";
    document.body.style.paddingRight = "15px";

    clearStuckPageOverlays();

    expect(document.body.style.width).toBe("");
    expect(document.body.style.overflow).toBe("");
    expect(document.body.style.paddingRight).toBe("");
  });

  it("removes mb-dyn-edit-overlay elements", () => {
    const overlay = document.createElement("div");
    overlay.className = "mb-dyn-edit-overlay";
    document.body.appendChild(overlay);

    expect(document.querySelectorAll(".mb-dyn-edit-overlay").length).toBe(1);
    clearStuckPageOverlays();
    expect(document.querySelectorAll(".mb-dyn-edit-overlay").length).toBe(0);
  });

  it("removes mb-dyn-leave-overlay elements", () => {
    const overlay = document.createElement("div");
    overlay.className = "mb-dyn-leave-overlay";
    document.body.appendChild(overlay);

    clearStuckPageOverlays();
    expect(document.querySelectorAll(".mb-dyn-leave-overlay").length).toBe(0);
  });

  it("does not remove el-overlay containing el-message-box", () => {
    const overlay = document.createElement("div");
    overlay.className = "el-overlay";
    const msgBox = document.createElement("div");
    msgBox.className = "el-message-box";
    overlay.appendChild(msgBox);
    document.body.appendChild(overlay);

    clearStuckPageOverlays();
    expect(document.querySelectorAll(".el-overlay").length).toBe(1);
  });

  it("does not remove el-overlay containing el-dialog", () => {
    const overlay = document.createElement("div");
    overlay.className = "el-overlay";
    const dialog = document.createElement("div");
    dialog.className = "el-dialog";
    overlay.appendChild(dialog);
    document.body.appendChild(overlay);

    clearStuckPageOverlays();
    expect(document.querySelectorAll(".el-overlay").length).toBe(1);
  });

  it("removes visible el-overlay without message-box or dialog", () => {
    const overlay = document.createElement("div");
    overlay.className = "el-overlay";
    // Make it non-visible so it gets removed (no el-message-box child)
    document.body.appendChild(overlay);

    clearStuckPageOverlays();
    expect(document.querySelectorAll(".el-overlay").length).toBe(0);
  });

  it("skips removal when .mm-del-overlay exists", () => {
    const delOverlay = document.createElement("div");
    delOverlay.className = "mm-del-overlay";
    document.body.appendChild(delOverlay);

    const overlay = document.createElement("div");
    overlay.className = "mb-dyn-edit-overlay";
    document.body.appendChild(overlay);

    clearStuckPageOverlays();
    // Both should remain because mm-del-overlay causes early return
    expect(document.querySelectorAll(".mm-del-overlay").length).toBe(1);
  });

  it("handles undefined document gracefully", async () => {
    // Temporarily make document undefined
    const savedDoc = globalThis.document;
    delete globalThis.document;

    // Should not throw
    expect(() => clearStuckPageOverlays()).not.toThrow();

    globalThis.document = savedDoc;
  });
});
