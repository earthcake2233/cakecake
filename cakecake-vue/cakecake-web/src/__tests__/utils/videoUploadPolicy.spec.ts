import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  isVideoUploadDisabled,
  VIDEO_UPLOAD_DISABLED_TITLE,
  VIDEO_UPLOAD_DISABLED_MESSAGE,
  STORAGE_METADATA_ONLY,
  guardVideoFileUploadDisabled,
  guardVideoUploadDisabled
} from "@/utils/videoUploadPolicy";

describe("videoUploadPolicy", () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
  });

  describe("constants", () => {
    it("exports STORAGE_METADATA_ONLY", () => {
      expect(STORAGE_METADATA_ONLY).toBe("creator_center_metadata_only");
    });

    it("VIDEO_UPLOAD_DISABLED_TITLE has fallback default", () => {
      expect(typeof VIDEO_UPLOAD_DISABLED_TITLE).toBe("string");
      expect(VIDEO_UPLOAD_DISABLED_TITLE.length).toBeGreaterThan(0);
    });

    it("VIDEO_UPLOAD_DISABLED_MESSAGE has fallback default", () => {
      expect(typeof VIDEO_UPLOAD_DISABLED_MESSAGE).toBe("string");
      expect(VIDEO_UPLOAD_DISABLED_MESSAGE.length).toBeGreaterThan(0);
    });
  });

  describe("isVideoUploadDisabled", () => {
    it("returns false when env is 'false'", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "false");
      expect(isVideoUploadDisabled()).toBe(false);
    });

    it("returns true when env is 'true'", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "true");
      expect(isVideoUploadDisabled()).toBe(true);
    });

    it("returns true when env is '1'", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "1");
      expect(isVideoUploadDisabled()).toBe(true);
    });

    it("returns true when env is 'yes'", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "yes");
      expect(isVideoUploadDisabled()).toBe(true);
    });

    it("returns true when env is 'on'", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "on");
      expect(isVideoUploadDisabled()).toBe(true);
    });

    it("returns false for unknown value", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "random");
      expect(isVideoUploadDisabled()).toBe(false);
    });

    it("returns false when env is empty string", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "");
      expect(isVideoUploadDisabled()).toBe(false);
    });

    it("handles whitespace in value", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "  true  ");
      expect(isVideoUploadDisabled()).toBe(true);
    });
  });

  describe("guardVideoFileUploadDisabled", () => {
    it("returns false when upload is enabled", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "false");
      expect(guardVideoFileUploadDisabled()).toBe(false);
    });

    it("returns true when upload is disabled", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "true");
      expect(guardVideoFileUploadDisabled()).toBe(true);
    });

    it("calls notify function when upload is disabled", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "true");
      const notify = vi.fn();
      expect(guardVideoFileUploadDisabled(notify)).toBe(true);
      expect(notify).toHaveBeenCalledTimes(1);
      expect(notify).toHaveBeenCalledWith(
        expect.stringContaining(VIDEO_UPLOAD_DISABLED_TITLE)
      );
    });

    it("does not call notify when upload is enabled", () => {
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "false");
      const notify = vi.fn();
      expect(guardVideoFileUploadDisabled(notify)).toBe(false);
      expect(notify).not.toHaveBeenCalled();
    });
  });

  describe("guardVideoUploadDisabled (deprecated)", () => {
    it("is same function as guardVideoFileUploadDisabled", () => {
      // They should have the same behavior
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "true");
      expect(guardVideoUploadDisabled()).toBe(true);
      vi.stubEnv("VITE_VIDEO_UPLOAD_DISABLED", "false");
      expect(guardVideoUploadDisabled()).toBe(false);
    });
  });
});
