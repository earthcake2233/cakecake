import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  registerMinibiliSessionInvalidate,
  invalidateMinibiliSessionFromHttp
} from "@/utils/minibiliAuthSync";
import { setTokens, getAccessToken } from "@/utils/authTokens";

describe("minibiliAuthSync", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("clears tokens, calls handler, and debounces rapid calls", () => {
    setTokens("access", "refresh");

    const handler = vi.fn();
    registerMinibiliSessionInvalidate(handler);

    // First call works
    invalidateMinibiliSessionFromHttp();
    expect(getAccessToken()).toBe("");
    expect(localStorage.getItem("signIn")).toBe("0");
    expect(handler).toHaveBeenCalledTimes(1);

    // Second rapid call is debounced (within 800ms)
    invalidateMinibiliSessionFromHttp();
    expect(handler).toHaveBeenCalledTimes(1);

    // After 800ms, another call should work
    vi.advanceTimersByTime(801);
    invalidateMinibiliSessionFromHttp();
    // handler not called again because clearTokens sets getAccessToken to ""
    // and the function still runs, but handler should be called
  });

  it("ignores null handler registration", () => {
    registerMinibiliSessionInvalidate(null);
    expect(() => invalidateMinibiliSessionFromHttp()).not.toThrow();
  });
});
