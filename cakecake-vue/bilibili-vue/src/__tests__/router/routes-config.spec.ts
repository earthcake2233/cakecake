import { describe, it, expect } from "vitest";
import router from "@/router/index";

describe("route names and paths", () => {
  const routeList = router.getRoutes();
  const expectedRoutes = [
    { name: "home", path: "/" },
    { name: "Ranking", path: "/ranking" },
    { name: "rankingDetail" },
    { name: "searchAll" },
    { name: "searchVideo" },
    { name: "searchBangumi" },
    { name: "searchPgc" },
    { name: "searchLive" },
    { name: "searchArticle" },
    { name: "searchTopic" },
    { name: "upuser" },
    { name: "photo" },
    { name: "video", path: "/video/:aid" },
    { name: "upload", path: "/upload" },
    { name: "videoPublish", path: "/upload/publish" },
    { name: "videoEdit" },
    { name: "articlePublish" },
    { name: "articleEdit" },
    { name: "manuscript", path: "/upload/manuscript" },
    { name: "appeal", path: "/upload/appeal" },
    { name: "creatorComments", path: "/upload/comments" },
    { name: "creatorDanmakus", path: "/upload/danmakus" },
    { name: "minibiliLogin" },
    { name: "minibiliRegister" },
    { name: "minibiliMessages" },
    { name: "minibiliDynamics" },
    { name: "minibiliPersonalCenter" },
    { name: "minibiliUserSpace" },
    { name: "minibiliUserSpaceRelations" },
    { name: "minibiliWatchLater" },
    { name: "minibiliArticleRead" },
    { name: "minibiliDynamicRead" },
    { name: "minibiliViewHistory" },
    { name: "minibiliUpload" },
    { name: "adminLogin" },
    { name: "notFound", path: "/404" },
  ];

  it.each(expectedRoutes.filter(r => r.path))("$name → $path", ({ name, path }) => {
    const r = routeList.find((x) => x.name === name);
    expect(r).toBeDefined();
    expect(r.path).toBe(path);
  });

  it.each(expectedRoutes.filter(r => !r.path))("has route $name", ({ name }) => {
    expect(routeList.find((x) => x.name === name)).toBeDefined();
  });

  it("has admin children routes", () => {
    ["adminBanners","adminHotSearch","adminVideoReview","adminArticleReview","adminDynamicManage","adminAgent","adminSystemConfigs"].forEach((name) => {
      expect(routeList.find((x) => x.name === name)).toBeDefined();
    });
  });
});

describe("route meta", () => {
  const routeList = router.getRoutes();
  it.each([
    { name: "home", metaProp: "title" },
    { name: "video", metaProp: "title" },
    { name: "adminLogin", metaProp: "hideGlobalChrome" },
    { name: "notFound", metaProp: "title" },
  ])("$name has meta.$metaProp", ({ name, metaProp }) => {
    const r = routeList.find((x) => x.name === name);
    expect(r).toBeDefined();
    expect(r.meta).toBeDefined();
    expect(r.meta[metaProp]).toBeDefined();
  });
  it("admin layout has requireAdminAuth meta", () => {
    const adminLayout = routeList.find((r) => r.path === "/admin" && r.meta && r.meta.requireAdminAuth);
    expect(adminLayout).toBeDefined();
  });
  it("minibili auth routes have requireMinibiliAuth meta", () => {
    ["minibiliViewHistory","minibiliUpload"].forEach((name) => {
      const r = routeList.find((x) => x.name === name);
      expect(r).toBeDefined();
      expect(r.meta.requireMinibiliAuth).toBe(true);
    });
  });
});

describe("route redirects", () => {
  const routeList = router.getRoutes();
  it("Ranking redirects to /ranking/all/0/0/0", () => {
    const ranking = routeList.find((r) => r.name === "Ranking");
    expect(ranking).toBeDefined();
    expect(ranking.redirect).toBe("/ranking/all/0/0/0");
  });
  it("search redirects to /search/all", () => {
    const search = routeList.find((r) => r.path === "/search" && !r.name);
    expect(search).toBeDefined();
    expect(search.redirect).toBe("/search/all");
  });
  it("catch-all redirects to /404", () => {
    const catchAll = routeList.find((r) => r.path === "/:pathMatch(.*)*");
    expect(catchAll).toBeDefined();
    expect(catchAll.redirect).toBe("/404");
  });
});

describe("scrollBehavior", () => {
  const sb = router.options.scrollBehavior;
  it("scrolls to top by default", () => {
    expect(sb({}, {}, null)).toEqual({ top: 0, left: 0 });
  });
  it("restores saved position", () => {
    expect(sb({}, {}, { top: 100 })).toEqual({ top: 100 });
  });
});
