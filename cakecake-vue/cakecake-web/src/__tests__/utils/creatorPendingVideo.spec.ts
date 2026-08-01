import { describe, it, expect } from "vitest";
import {
  stashPendingVideoFile,
  takePendingVideoFile,
  clearPendingVideoFile
} from "@/utils/creatorPendingVideo";

describe("creatorPendingVideo", () => {
  it("stash and take returns the file", () => {
    const file = new File(["test"], "test.mp4");
    stashPendingVideoFile(file);
    expect(takePendingVideoFile()).toBe(file);
  });

  it("take clears the file", () => {
    stashPendingVideoFile(new File(["a"], "a.mp4"));
    takePendingVideoFile();
    expect(takePendingVideoFile()).toBeNull();
  });

  it("clearPendingVideoFile resets state", () => {
    stashPendingVideoFile(new File(["b"], "b.mp4"));
    clearPendingVideoFile();
    expect(takePendingVideoFile()).toBeNull();
  });

  it("stash null clears", () => {
    stashPendingVideoFile(new File(["c"], "c.mp4"));
    stashPendingVideoFile(null);
    expect(takePendingVideoFile()).toBeNull();
  });
});
