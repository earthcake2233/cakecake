import { describe, it, expect, vi, beforeAll } from "vitest";

const mockHttp = { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() };
vi.mock("@/utils/http", () => ({ default: mockHttp }));

let adminApi;
beforeAll(async () => {
  vi.stubEnv("VITE_MINIBILI_API", "");
  adminApi = await import("@/api/admin");
});

describe("api/admin.js - non-minibili mode", () => {
  it("getHomeBannersPublic returns empty when not minibili", async () => {
    const r = await adminApi.getHomeBannersPublic();
    expect(r).toEqual({ code: 0, data: { items: [] } });
    expect(mockHttp.get).not.toHaveBeenCalled();
  });
});
