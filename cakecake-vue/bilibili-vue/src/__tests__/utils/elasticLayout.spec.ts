import { describe, it, expect, vi, beforeEach } from "vitest";
import { installElasticLayout } from "@/utils/elasticLayout";

describe("elasticLayout", () => {
  beforeEach(() => {
    // Clean up any injected style between tests
    document.head.querySelectorAll("#mb-elastic-layout").forEach(el => el.remove());
    vi.restoreAllMocks();
  });

  it("injects a style element into document head", () => {
    installElasticLayout();
    const style = document.getElementById("mb-elastic-layout");
    expect(style).not.toBeNull();
    expect(style.tagName).toBe("STYLE");
  });

  it("only injects style once (idempotent)", () => {
    installElasticLayout();
    installElasticLayout();
    const styles = document.head.querySelectorAll("#mb-elastic-layout");
    expect(styles.length).toBe(1);
  });

  it("sets data-source attribute on the style element", () => {
    installElasticLayout();
    const style = document.getElementById("mb-elastic-layout");
    expect(style.getAttribute("data-source")).toBe("mb-elastic-layout");
  });

  it("style element contains CSS content", () => {
    installElasticLayout();
    const style = document.getElementById("mb-elastic-layout");
    expect(style.textContent.length).toBeGreaterThan(0);
    expect(style.textContent).toContain("--mb-e-gutter");
    expect(style.textContent).toContain("--mb-e-main");
  });

  it("adds resize event listener", () => {
    const addEventListenerSpy = vi.spyOn(window, "addEventListener");
    installElasticLayout();
    expect(addEventListenerSpy).toHaveBeenCalledWith("resize", expect.any(Function), { passive: true });
  });

  it("sets mb-e-narrow class when window width < 960", () => {
    Object.defineProperty(window, "innerWidth", { value: 800, configurable: true });
    installElasticLayout();
    expect(document.documentElement.classList.contains("mb-e-narrow")).toBe(true);
  });

  it("sets mb-e-compact class when window width < 1200", () => {
    Object.defineProperty(window, "innerWidth", { value: 1100, configurable: true });
    installElasticLayout();
    expect(document.documentElement.classList.contains("mb-e-compact")).toBe(true);
  });

  it("does not set narrow class when window width >= 960", () => {
    Object.defineProperty(window, "innerWidth", { value: 960, configurable: true });
    installElasticLayout();
    expect(document.documentElement.classList.contains("mb-e-narrow")).toBe(false);
  });

  it("handles undefined document gracefully", async () => {
    const savedDoc = globalThis.document;
    delete globalThis.document;

    expect(() => installElasticLayout()).not.toThrow();

    globalThis.document = savedDoc;
  });
});
