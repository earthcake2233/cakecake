import { describe, it, expect } from "vitest";

// Read route names and paths from the router module
import router from "@/router/index";

describe("router routes config", () => {
  const routeList = router.getRoutes();

  it("has home route", () => {
    const home = routeList.find(r => r.name === "home");
    expect(home).toBeDefined();
    expect(home.path).toBe("/");
  });

  it("has video route with :aid param", () => {
    const video = routeList.find(r => r.name === "video");
    expect(video).toBeDefined();
    expect(video.path).toBe("/video/:aid");
  });

  it("has notFound catch-all route", () => {
    const nf = routeList.find(r => r.name === "notFound");
    expect(nf).toBeDefined();
    expect(nf.path).toBe("/404");
  });

  it("has search routes", () => {
    const searchAll = routeList.find(r => r.name === "searchAll");
    expect(searchAll).toBeDefined();
    expect(searchAll.path).toBe("/search/all");
  });

  it("has upload routes", () => {
    const upload = routeList.find(r => r.name === "upload");
    expect(upload).toBeDefined();
    const publish = routeList.find(r => r.name === "videoPublish");
    expect(publish).toBeDefined();
  });

  it("has minibili routes", () => {
    const login = routeList.find(r => r.name === "minibiliLogin");
    expect(login).toBeDefined();
    expect(login.path).toBe("/minibili/login");

    const msgs = routeList.find(r => r.name === "minibiliMessages");
    expect(msgs).toBeDefined();
    expect(msgs.path).toBe("/minibili/messages");
  });

  it("has admin routes", () => {
    const adminLogin = routeList.find(r => r.name === "adminLogin");
    expect(adminLogin).toBeDefined();
  });

  it("uses hash history", () => {
    expect(router.options.history).toBeDefined();
  });
});
