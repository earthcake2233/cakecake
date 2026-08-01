import { describe, it, expect, vi } from "vitest";
import module from "@/store/modules/video";

describe("store/modules/video", () => {
  describe("state", () => {
    it("has initial aid empty string", () => {
      expect(module.state.aid).toBe("");
    });

    it("has initial cid empty string", () => {
      expect(module.state.cid).toBe("");
    });
  });

  describe("mutations", () => {
    it("setAid converts to number", () => {
      const state = { aid: "", cid: "" };
      module.mutations.setAid(state, "12345");
      expect(state.aid).toBe(12345);
    });

    it("setCid stores as-is", () => {
      const state = { aid: "", cid: "" };
      module.mutations.setCid(state, "abc-def");
      expect(state.cid).toBe("abc-def");
    });
  });

  describe("actions", () => {
    it("getAid commits setAid with message", () => {
      const commit = vi.fn();
      module.actions.getAid({ commit }, "67890");
      expect(commit).toHaveBeenCalledWith("setAid", "67890");
    });

    it("getCid commits setCid with message", () => {
      const commit = vi.fn();
      module.actions.getCid({ commit }, "xyz-123");
      expect(commit).toHaveBeenCalledWith("setCid", "xyz-123");
    });
  });

  describe("namespaced", () => {
    it("is namespaced", () => {
      expect(module.namespaced).toBe(true);
    });
  });
});
