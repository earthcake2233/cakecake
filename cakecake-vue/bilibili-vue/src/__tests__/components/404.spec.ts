import { describe, it, expect, vi, beforeEach } from "vitest";
import { mount } from "@vue/test-utils";
import NotFoundPage from "@/pages/notFound/404.vue";

describe("NotFoundPage.vue", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    Object.defineProperty(window, "history", {
      writable: true,
      value: { length: 2, back: vi.fn() }
    });
  });

  it("renders error container", () => {
    const wrapper = mount(NotFoundPage);
    expect(wrapper.find(".error-container").exists()).toBe(true);
  });

  it("renders 'very sorry' image", () => {
    const wrapper = mount(NotFoundPage);
    const img = wrapper.find(".error-sign");
    expect(img.exists()).toBe(true);
    expect(img.attributes("alt")).toBeTruthy();
  });

  it("renders go back button", () => {
    const wrapper = mount(NotFoundPage);
    expect(wrapper.text()).toContain("返回上一页");
  });

  it("renders change manga button", () => {
    const wrapper = mount(NotFoundPage);
    expect(wrapper.find(".change-img-btn").exists()).toBe(true);
  });

  it("calls window.history.back() when goBack is invoked and history.length > 1", async () => {
    const backSpy = vi.fn();
    Object.defineProperty(window, "history", {
      writable: true,
      value: { length: 2, back: backSpy }
    });
    const wrapper = mount(NotFoundPage);
    await wrapper.find(".rollback-btn").trigger("click.prevent");
    expect(backSpy).toHaveBeenCalled();
  });

  it("calls $router.push when goBack is invoked and history.length <= 1", async () => {
    const routerPush = vi.fn();
    Object.defineProperty(window, "history", {
      writable: true,
      value: { length: 1, back: vi.fn() }
    });
    const wrapper = mount(NotFoundPage, {
      global: {
        mocks: {
          $router: { push: routerPush }
        }
      }
    });
    await wrapper.find(".rollback-btn").trigger("click.prevent");
    expect(routerPush).toHaveBeenCalledWith({ name: "home" });
  });

  it("changeManga changes the manga image", async () => {
    const wrapper = mount(NotFoundPage);
    const newSrc = wrapper.find(".error-manga__img").attributes("src");
    expect(newSrc).toBeTruthy();
    await wrapper.find(".change-img-btn").trigger("click.prevent");
  });
});
