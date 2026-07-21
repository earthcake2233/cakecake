import { describe, it, expect, beforeEach } from "vitest";
import {
  buildSpaceViewerProfile,
  isSpacePerspectivePreviewMode,
  spacePerspectiveStorageKey,
  readStoredSpacePerspective,
  writeStoredSpacePerspective,
  resolveSpacePerspective
} from "@/utils/spacePerspective";

describe("buildSpaceViewerProfile", () => {
  const full = {
    user_id: 1,
    username: "test",
    birthday: "2000-01-01",
    privacy: { public_birthday: true }
  };

  it("builds profile for fan perspective", () => {
    const p = buildSpaceViewerProfile(full, "fan");
    expect(p.is_owner).toBe(false);
    expect(p.followed_by_me).toBe(true);
    expect(p.birthday).toBe("2000-01-01");
  });

  it("builds profile for visitor perspective", () => {
    const p = buildSpaceViewerProfile(full, "visitor");
    expect(p.followed_by_me).toBe(false);
  });

  it("hides birthday when private", () => {
    const p = buildSpaceViewerProfile(
      { ...full, privacy: { public_birthday: false } },
      "fan"
    );
    expect(p.birthday).toBe("");
  });

  it("defaults privacy when src has none", () => {
    const p = buildSpaceViewerProfile({ user_id: 1 }, "visitor");
    expect(p.privacy.public_birthday).toBe(true);
    expect(p.privacy.public_favorites).toBe(false);
  });

  it("returns null for invalid input", () => {
    expect(buildSpaceViewerProfile(null, "fan")).toBeNull();
    expect(buildSpaceViewerProfile(undefined, "fan")).toBeNull();
  });
});

describe("isSpacePerspectivePreviewMode", () => {
  it("accepts fan and visitor", () => {
    expect(isSpacePerspectivePreviewMode("fan")).toBe(true);
    expect(isSpacePerspectivePreviewMode("visitor")).toBe(true);
  });

  it("rejects other values", () => {
    expect(isSpacePerspectivePreviewMode("self")).toBe(false);
    expect(isSpacePerspectivePreviewMode("")).toBe(false);
  });
});

describe("spacePerspectiveStorageKey", () => {
  it("generates key for valid userId", () => {
    expect(spacePerspectiveStorageKey(42)).toBe("minibili_space_perspective_42");
  });

  it("returns empty for invalid", () => {
    expect(spacePerspectiveStorageKey(0)).toBe("");
    expect(spacePerspectiveStorageKey(-1)).toBe("");
  });
});

describe("readStoredSpacePerspective", () => {
  beforeEach(() => sessionStorage.clear());

  it("returns self when no stored value", () => {
    expect(readStoredSpacePerspective(42)).toBe("self");
  });

  it("reads stored fan mode", () => {
    sessionStorage.setItem("minibili_space_perspective_42", "fan");
    expect(readStoredSpacePerspective(42)).toBe("fan");
  });
});

describe("writeStoredSpacePerspective", () => {
  beforeEach(() => sessionStorage.clear());

  it("writes and reads back", () => {
    writeStoredSpacePerspective(42, "fan");
    expect(readStoredSpacePerspective(42)).toBe("fan");
  });

  it("removes key when mode is self", () => {
    writeStoredSpacePerspective(42, "visitor");
    writeStoredSpacePerspective(42, "self");
    expect(readStoredSpacePerspective(42)).toBe("self");
  });
});

describe("resolveSpacePerspective", () => {
  beforeEach(() => sessionStorage.clear());

  it("uses query param when valid", () => {
    expect(resolveSpacePerspective(42, "fan", false)).toBe("fan");
  });

  it("falls back to session for owner", () => {
    writeStoredSpacePerspective(42, "visitor");
    expect(resolveSpacePerspective(42, "", true)).toBe("visitor");
  });

  it("defaults to self for non-owner", () => {
    expect(resolveSpacePerspective(42, "", false)).toBe("self");
  });
});
