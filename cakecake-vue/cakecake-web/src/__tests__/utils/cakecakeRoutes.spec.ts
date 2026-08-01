import { describe, it, expect } from "vitest";
import {
  MINIBILI_COMPACT_HEADER_ROUTES,
  isCakecakeCompactHeaderRoute,
  shouldShowCakecakeCompactHeader,
  shouldShowHomeHeaderChrome,
  cakecakeUserSpaceRoute,
  cakecakeWatchLaterRoute,
  cakecakeDynamicsRoute,
  cakecakePersonalCenterRoute,
  cakecakeVideoPlayRoute,
  cakecakeArticleReadRoute,
  cakecakeDynamicReadRoute,
  cakecakeUserSpaceRelationsRoute
} from "@/utils/cakecakeRoutes";

describe("MINIBILI_COMPACT_HEADER_ROUTES", () => {
  it("contains known routes", () => {
    expect(MINIBILI_COMPACT_HEADER_ROUTES.has("notFound")).toBe(true);
    expect(MINIBILI_COMPACT_HEADER_ROUTES.has("cakecakePersonalCenter")).toBe(true);
  });
});

describe("isCakecakeCompactHeaderRoute", () => {
  it("returns true for compact routes", () => {
    expect(isCakecakeCompactHeaderRoute("notFound")).toBe(true);
  });

  it("returns false for non-compact routes", () => {
    expect(isCakecakeCompactHeaderRoute("home")).toBe(false);
  });
});

describe("shouldShowCakecakeCompactHeader", () => {
  it("checks route name", () => {
    expect(shouldShowCakecakeCompactHeader({ name: "notFound" })).toBe(true);
    expect(shouldShowCakecakeCompactHeader({})).toBe(false);
    expect(shouldShowCakecakeCompactHeader(null)).toBe(false);
  });
});

describe("shouldShowHomeHeaderChrome", () => {
  it("shows chrome for home route", () => {
    expect(shouldShowHomeHeaderChrome({ name: "home" })).toBe(true);
  });

  it("hides chrome for compact routes", () => {
    expect(shouldShowHomeHeaderChrome({ name: "notFound" })).toBe(false);
  });
});

describe("cakecakeUserSpaceRoute", () => {
  it("builds route for valid userId", () => {
    const r = cakecakeUserSpaceRoute(42);
    expect(r.name).toBe("cakecakeUserSpace");
    expect(r.params.userId).toBe("42");
  });

  it("returns null for invalid userId", () => {
    expect(cakecakeUserSpaceRoute(0)).toBeNull();
    expect(cakecakeUserSpaceRoute(-1)).toBeNull();
  });
});

describe("cakecakeWatchLaterRoute", () => {
  it("returns watch-later route", () => {
    expect(cakecakeWatchLaterRoute().name).toBe("cakecakeWatchLater");
  });
});

describe("cakecakeDynamicsRoute", () => {
  it("returns dynamics route", () => {
    expect(cakecakeDynamicsRoute().name).toBe("cakecakeDynamics");
  });
});

describe("cakecakePersonalCenterRoute", () => {
  it("builds route with tab query", () => {
    const r = cakecakePersonalCenterRoute("coin");
    expect(r.name).toBe("cakecakePersonalCenter");
    expect(r.query.tab).toBe("coin");
  });

  it("builds route without tab", () => {
    const r = cakecakePersonalCenterRoute();
    expect(r.name).toBe("cakecakePersonalCenter");
    expect(r.query).toBeUndefined();
  });
});

describe("cakecakeVideoPlayRoute", () => {
  it("builds route with BV id", () => {
    const r = cakecakeVideoPlayRoute(42);
    expect(r.name).toBe("video");
    expect(r.params.aid).toBe("BV42");
  });

  it("handles 0 by falling back to string id", () => {
    const r = cakecakeVideoPlayRoute(0);
    expect(r.name).toBe("video");
    expect(r.params.aid).toBe("0");
  });
});

describe("cakecakeArticleReadRoute", () => {
  it("builds route for valid article id", () => {
    const r = cakecakeArticleReadRoute(100);
    expect(r.name).toBe("cakecakeArticleRead");
    expect(r.params.id).toBe("100");
  });

  it("returns null for invalid id", () => {
    expect(cakecakeArticleReadRoute(0)).toBeNull();
    expect(cakecakeArticleReadRoute("abc")).toBeNull();
  });
});

describe("cakecakeDynamicReadRoute", () => {
  it("builds route with query", () => {
    const r = cakecakeDynamicReadRoute(5, { edit: "1" });
    expect(r.name).toBe("cakecakeDynamicRead");
    expect(r.params.id).toBe("5");
    expect(r.query.edit).toBe("1");
  });

  it("handles invalid id", () => {
    expect(cakecakeDynamicReadRoute(0)).toBeNull();
  });
});

describe("cakecakeUserSpaceRelationsRoute", () => {
  it("builds followers route", () => {
    const r = cakecakeUserSpaceRelationsRoute(42, "followers");
    expect(r.name).toBe("cakecakeUserSpaceRelations");
    expect(r.query.tab).toBe("followers");
  });

  it("defaults to following", () => {
    const r = cakecakeUserSpaceRelationsRoute(42);
    expect(r.query.tab).toBe("following");
  });

  it("returns null for invalid id", () => {
    expect(cakecakeUserSpaceRelationsRoute(0)).toBeNull();
  });
});
