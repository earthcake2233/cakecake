import { describe, it, expect } from "vitest";
import {
  MINIBILI_COMPACT_HEADER_ROUTES,
  isMinibiliCompactHeaderRoute,
  shouldShowMinibiliCompactHeader,
  shouldShowHomeHeaderChrome,
  minibiliUserSpaceRoute,
  minibiliWatchLaterRoute,
  minibiliDynamicsRoute,
  minibiliPersonalCenterRoute,
  minibiliVideoPlayRoute,
  minibiliArticleReadRoute,
  minibiliDynamicReadRoute,
  minibiliUserSpaceRelationsRoute
} from "@/utils/minibiliRoutes";

describe("MINIBILI_COMPACT_HEADER_ROUTES", () => {
  it("contains known routes", () => {
    expect(MINIBILI_COMPACT_HEADER_ROUTES.has("notFound")).toBe(true);
    expect(MINIBILI_COMPACT_HEADER_ROUTES.has("minibiliPersonalCenter")).toBe(true);
  });
});

describe("isMinibiliCompactHeaderRoute", () => {
  it("returns true for compact routes", () => {
    expect(isMinibiliCompactHeaderRoute("notFound")).toBe(true);
  });

  it("returns false for non-compact routes", () => {
    expect(isMinibiliCompactHeaderRoute("home")).toBe(false);
  });
});

describe("shouldShowMinibiliCompactHeader", () => {
  it("checks route name", () => {
    expect(shouldShowMinibiliCompactHeader({ name: "notFound" })).toBe(true);
    expect(shouldShowMinibiliCompactHeader({})).toBe(false);
    expect(shouldShowMinibiliCompactHeader(null)).toBe(false);
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

describe("minibiliUserSpaceRoute", () => {
  it("builds route for valid userId", () => {
    const r = minibiliUserSpaceRoute(42);
    expect(r.name).toBe("minibiliUserSpace");
    expect(r.params.userId).toBe("42");
  });

  it("returns null for invalid userId", () => {
    expect(minibiliUserSpaceRoute(0)).toBeNull();
    expect(minibiliUserSpaceRoute(-1)).toBeNull();
  });
});

describe("minibiliWatchLaterRoute", () => {
  it("returns watch-later route", () => {
    expect(minibiliWatchLaterRoute().name).toBe("minibiliWatchLater");
  });
});

describe("minibiliDynamicsRoute", () => {
  it("returns dynamics route", () => {
    expect(minibiliDynamicsRoute().name).toBe("minibiliDynamics");
  });
});

describe("minibiliPersonalCenterRoute", () => {
  it("builds route with tab query", () => {
    const r = minibiliPersonalCenterRoute("coin");
    expect(r.name).toBe("minibiliPersonalCenter");
    expect(r.query.tab).toBe("coin");
  });

  it("builds route without tab", () => {
    const r = minibiliPersonalCenterRoute();
    expect(r.name).toBe("minibiliPersonalCenter");
    expect(r.query).toBeUndefined();
  });
});

describe("minibiliVideoPlayRoute", () => {
  it("builds route with BV id", () => {
    const r = minibiliVideoPlayRoute(42);
    expect(r.name).toBe("video");
    expect(r.params.aid).toBe("BV42");
  });

  it("handles 0 by falling back to string id", () => {
    const r = minibiliVideoPlayRoute(0);
    expect(r.name).toBe("video");
    expect(r.params.aid).toBe("0");
  });
});

describe("minibiliArticleReadRoute", () => {
  it("builds route for valid article id", () => {
    const r = minibiliArticleReadRoute(100);
    expect(r.name).toBe("minibiliArticleRead");
    expect(r.params.id).toBe("100");
  });

  it("returns null for invalid id", () => {
    expect(minibiliArticleReadRoute(0)).toBeNull();
    expect(minibiliArticleReadRoute("abc")).toBeNull();
  });
});

describe("minibiliDynamicReadRoute", () => {
  it("builds route with query", () => {
    const r = minibiliDynamicReadRoute(5, { edit: "1" });
    expect(r.name).toBe("minibiliDynamicRead");
    expect(r.params.id).toBe("5");
    expect(r.query.edit).toBe("1");
  });

  it("handles invalid id", () => {
    expect(minibiliDynamicReadRoute(0)).toBeNull();
  });
});

describe("minibiliUserSpaceRelationsRoute", () => {
  it("builds followers route", () => {
    const r = minibiliUserSpaceRelationsRoute(42, "followers");
    expect(r.name).toBe("minibiliUserSpaceRelations");
    expect(r.query.tab).toBe("followers");
  });

  it("defaults to following", () => {
    const r = minibiliUserSpaceRelationsRoute(42);
    expect(r.query.tab).toBe("following");
  });

  it("returns null for invalid id", () => {
    expect(minibiliUserSpaceRelationsRoute(0)).toBeNull();
  });
});
