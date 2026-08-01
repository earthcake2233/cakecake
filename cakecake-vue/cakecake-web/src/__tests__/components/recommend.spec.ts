import { describe, it, expect, vi, beforeEach } from "vitest";
import { mount } from "@vue/test-utils";
import { nextTick } from "vue";

vi.mock("@/api", () => ({
  getHomeRecommendPool: vi.fn().mockResolvedValue([
    { aid: 1, title: "Video 1", pic: "https://example.com/1.jpg", author: "Author 1", play: 1000 }
  ])
}));

vi.mock("@/utils/videoRecommendFeeds", () => ({
  fillHomeRecommendSlots: (pool) => pool,
  HOME_RECOMMEND_PAGE_SIZE: 8,
  nextHomeRecommendBatch: (pool, display, offset, direction) => {
    const newOffset = offset + direction;
    const items = pool.slice(newOffset, newOffset + 8);
    return items.length ? { items, nextOffset: newOffset } : { items: [], nextOffset: offset };
  }
}));

vi.mock("@/utils/utils", () => ({ count2: (n) => String(n) }));
vi.mock("@/utils/formatDuration", () => ({ formatDuration: (s) => String(s) + "s" }));

import Recommend from "@/components/recommend/recommend.vue";

describe("Recommend.vue", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("renders with empty recommend prop", async () => {
    const wrapper = mount(Recommend, {
      props: { recommend: { rec: [], day: 3 } },
      global: { stubs: { "router-link": true, WatchLaterBtn: true } }
    });
    await nextTick(); await nextTick();
    expect(wrapper.find(".recommend-module").exists()).toBe(true);
  });

  it("renders refresh buttons", () => {
    const wrapper = mount(Recommend, {
      props: { recommend: { rec: [], day: 3 } },
      global: { stubs: { "router-link": true, WatchLaterBtn: true } }
    });
    expect(wrapper.find(".rec-left").exists()).toBe(true);
    expect(wrapper.find(".rec-right").exists()).toBe(true);
  });

  it("updates displayItems from recommend.rec prop", async () => {
    const wrapper = mount(Recommend, {
      props: {
        recommend: {
          rec: [{ aid: 10, title: "Rec 1", pic: "", author: "A", play: 500 }],
          day: 3
        }
      },
      global: { stubs: { "router-link": true, WatchLaterBtn: true } }
    });
    await nextTick(); await nextTick();
    expect(wrapper.vm.pool.length).toBeGreaterThan(0);
  });
});
