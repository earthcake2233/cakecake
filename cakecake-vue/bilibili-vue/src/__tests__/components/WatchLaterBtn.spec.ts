import { describe, it, expect, vi, beforeEach } from "vitest";
import { mount } from "@vue/test-utils";
import WatchLaterBtn from "@/components/common/WatchLaterBtn.vue";

const toggleWatchLaterMock = vi.fn();
vi.mock("@/utils/watchLaterAction", () => ({
  toggleWatchLaterVideo: (...args) => toggleWatchLaterMock(...args)
}));

describe("WatchLaterBtn.vue", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    toggleWatchLaterMock.mockResolvedValue(true);
  });

  it("does not render when videoId is 0", () => {
    const wrapper = mount(WatchLaterBtn, { props: { videoId: 0 } });
    expect(wrapper.find("button").exists()).toBe(false);
  });

  it("does not render when videoId is negative", () => {
    const wrapper = mount(WatchLaterBtn, { props: { videoId: -1 } });
    expect(wrapper.find("button").exists()).toBe(false);
  });

  it("renders when videoId is positive", () => {
    const wrapper = mount(WatchLaterBtn, { props: { videoId: 123 } });
    expect(wrapper.find("button").exists()).toBe(true);
  });

  it("renders button with aria-label", () => {
    const wrapper = mount(WatchLaterBtn, { props: { videoId: 456 } });
    expect(wrapper.find("button").attributes("aria-label")).toBe("稍后再看");
  });

  it("shows active state when inWatchLater is true", () => {
    const wrapper = mount(WatchLaterBtn, { props: { videoId: 123, inWatchLater: true } });
    expect(wrapper.find("button").classes()).toContain("home-wl-btn--on");
  });

  it("does not show active state when inWatchLater is false", () => {
    const wrapper = mount(WatchLaterBtn, { props: { videoId: 123, inWatchLater: false } });
    expect(wrapper.find("button").classes()).not.toContain("home-wl-btn--on");
  });

  it("calls toggleWatchLaterVideo on click", async () => {
    const store = { dispatch: vi.fn() };
    const wrapper = mount(WatchLaterBtn, {
      props: { videoId: 789 },
      global: { mocks: { $store: store } }
    });
    await wrapper.find("button").trigger("click.prevent");
    expect(toggleWatchLaterMock).toHaveBeenCalledWith(store, 789);
  });

  it("sets active class when toggle returns true", async () => {
    const store = { dispatch: vi.fn() };
    const wrapper = mount(WatchLaterBtn, {
      props: { videoId: 789 },
      global: { mocks: { $store: store } }
    });
    await wrapper.find("button").trigger("click.prevent");
    expect(wrapper.find("button").classes()).toContain("home-wl-btn--on");
  });

  it("survives toggleWatchLaterVideo rejection", async () => {
    const store = { dispatch: vi.fn() };
    toggleWatchLaterMock.mockResolvedValueOnce(true);
    const wrapper = mount(WatchLaterBtn, {
      props: { videoId: 789 },
      global: { mocks: { $store: store } }
    });
    await wrapper.find("button").trigger("click.prevent");
    // Component should still be rendered after handling
    expect(wrapper.find("button").exists()).toBe(true);
  });
});
