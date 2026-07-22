import { describe, it, expect, vi, beforeEach } from "vitest";
import { mount } from "@vue/test-utils";
import { nextTick } from "vue";

vi.mock("@/utils/utils", () => ({
  count2: vi.fn((n) => n >= 10000 ? (n / 10000).toFixed(1) + "万" : String(n))
}));

vi.mock("@/utils/formatDuration", () => ({
  formatDuration: vi.fn((sec) => {
    const m = Math.floor(sec / 60);
    const s = sec % 60;
    return m + ":" + String(s).padStart(2, "0");
  })
}));

const storeDispatchMock = vi.fn();
const mockStore = { dispatch: storeDispatchMock };

const RouterLinkStub = {
  props: ["to", "title"],
  template: '<a class="router-link-stub"><slot /></a>'
};

import StoreyBox from "@/components/storeyBox/storeyBox.vue";

const sampleStoreydata = {
  id: 1, rid: 13, icon: "icon-donghua",
  title: "动画", title2: "动画区",
  tab: [{ name: "动态" }, { name: "最新" }],
  moreUrl: "//www.bilibili.com/v/donghua/",
  dynamic: 42, offsetTop: 500,
  data: {
    archives: [
      { aid: 1001, title: "Test Video 1", pic: "https://example.com/1.jpg", duration: 360, stat: { view: 50000, danmaku: 1200 }, in_watch_later: false },
      { aid: 1002, title: "Test Video 2", pic: "https://example.com/2.jpg", duration: 180, stat: { view: 30000, danmaku: 800 }, in_watch_later: true },
      { aid: 1003, title: "Test Video 3", pic: "https://example.com/3.jpg", duration: 60, stat: { view: 1500, danmaku: 50 }, in_watch_later: false }
    ]
  }
};

describe("StoreyBox.vue", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    storeDispatchMock.mockResolvedValue([]);
  });

  it("renders with empty storeydata (no data archive)", () => {
    const wrapper = mount(StoreyBox, {
      props: { storeydata: { rid: 1 }, scrollTop: 0 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    expect(wrapper.find(".new-comers-module").exists()).toBe(true);
  });

  it("renders zone title", () => {
    const wrapper = mount(StoreyBox, {
      props: { storeydata: sampleStoreydata, scrollTop: 600 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    expect(wrapper.text()).toContain("动画区");
  });

  it("renders tab items", () => {
    const wrapper = mount(StoreyBox, {
      props: { storeydata: sampleStoreydata, scrollTop: 600 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    expect(wrapper.text()).toContain("动态");
    expect(wrapper.text()).toContain("最新");
  });

  it("renders 3 spread-module items for 3 archives", () => {
    const wrapper = mount(StoreyBox, {
      props: { storeydata: sampleStoreydata, scrollTop: 600 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    expect(wrapper.findAll(".spread-module").length).toBe(3);
  });

  it("renders dynamic count badge", () => {
    const wrapper = mount(StoreyBox, {
      props: { storeydata: sampleStoreydata, scrollTop: 600 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    expect(wrapper.text()).toContain("42");
  });

  it("renders more link", () => {
    const wrapper = mount(StoreyBox, {
      props: { storeydata: sampleStoreydata, scrollTop: 600 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    expect(wrapper.find(".link-more").exists()).toBe(true);
  });

  it("changes tab on click", async () => {
    const wrapper = mount(StoreyBox, {
      props: { storeydata: sampleStoreydata, scrollTop: 600 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    const tabs = wrapper.findAll(".bili-tab-item");
    await tabs[1].trigger("click");
    expect(wrapper.vm.nowtab).toBe(1);
  });

  it("emits setDynamicRegion when scrollTop passes threshold", async () => {
    const wrapper = mount(StoreyBox, {
      props: { storeydata: sampleStoreydata, scrollTop: 0 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    await wrapper.setProps({ scrollTop: 2000 });
    await nextTick();
    expect(wrapper.emitted("setDynamicRegion")).toBeTruthy();
  });

  it("calls store.dispatch on refreshZone", async () => {
    const wrapper = mount(StoreyBox, {
      props: { storeydata: sampleStoreydata, scrollTop: 600 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    await wrapper.find(".read-push").trigger("click");
    expect(storeDispatchMock).toHaveBeenCalled();
  });

  it("adds fj class for rid=13", () => {
    const wrapper = mount(StoreyBox, {
      props: { storeydata: sampleStoreydata, scrollTop: 600 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    expect(wrapper.find(".fj").exists()).toBe(true);
  });

  it("prevents double refresh while refreshing", async () => {
    storeDispatchMock.mockImplementation(() => new Promise(() => {}));
    const wrapper = mount(StoreyBox, {
      props: { storeydata: sampleStoreydata, scrollTop: 600 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    await wrapper.find(".read-push").trigger("click");
    expect(wrapper.vm.refreshing).toBe(true);
    await wrapper.find(".read-push").trigger("click");
    expect(storeDispatchMock).toHaveBeenCalledTimes(1);
  });

  it("handles refreshZone store error gracefully", async () => {
    storeDispatchMock.mockRejectedValue(new Error("store error"));
    const wrapper = mount(StoreyBox, {
      props: { storeydata: sampleStoreydata, scrollTop: 600 },
      global: { stubs: { "router-link": RouterLinkStub, WatchLaterBtn: true }, mocks: { $store: mockStore } }
    });
    await wrapper.find(".read-push").trigger("click");
    expect(wrapper.vm.refreshing).toBe(false);
  });
});
