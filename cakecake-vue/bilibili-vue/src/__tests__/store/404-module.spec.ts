import { describe, it, expect } from "vitest";
import module from "@/store/modules/404";

describe("store/modules/404", () => {
  describe("state", () => {
    it("has initial error array", () => {
      expect(module.state.error).toEqual([]);
    });

    it("has initial randomNum 0", () => {
      expect(module.state.randomNum).toBe(0);
    });
  });

  describe("mutations", () => {
    it("error() sets error data", () => {
      const state = { error: [], randomNum: 0 };
      module.mutations.error(state, { message: "not found" });
      expect(state.error).toEqual({ message: "not found" });
    });

    it("randomNum() sets random number", () => {
      const state = { error: [], randomNum: 0 };
      module.mutations.randomNum(state, 42);
      expect(state.randomNum).toBe(42);
    });
  });

  describe("actions", () => {
    it("setError commits error mutation", () => {
      const commit = vi.fn();
      module.actions.setError({ commit }, { msg: "test error" });
      expect(commit).toHaveBeenCalledWith("error", { msg: "test error" });
    });

    it("setRandomNum commits randomNum mutation", () => {
      const commit = vi.fn();
      module.actions.setRandomNum({ commit }, 99);
      expect(commit).toHaveBeenCalledWith("randomNum", 99);
    });
  });

  describe("namespaced", () => {
    it("is namespaced", () => {
      expect(module.namespaced).toBe(true);
    });
  });
});
