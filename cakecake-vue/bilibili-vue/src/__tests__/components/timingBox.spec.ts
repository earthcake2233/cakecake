import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import TimingBox from "@/components/timingBox/timingBox.vue";

const sampleData = [
  { season_id: 1, title: "Anime A", square_cover: "https://example.com/a.jpg", weekday: 1, new: true, bgmcount: 5, ep_id: 10, favorites: 100 },
  { season_id: 2, title: "Anime B", square_cover: "https://example.com/b.jpg", weekday: 2, new: false, bgmcount: 12, ep_id: 20, favorites: 200 },
  { season_id: 3, title: "Anime C", square_cover: "https://example.com/c.jpg", weekday: 3, new: true, bgmcount: -1, ep_id: 30, favorites: 150 },
  { season_id: 4, title: "Anime D", square_cover: "https://example.com/d.jpg", weekday: 0, new: false, bgmcount: 8, ep_id: 40, favorites: 300 }
];

describe("TimingBox.vue", () => {
  it("renders with empty data", () => {
    const wrapper = mount(TimingBox, {
      props: { timelineData: [], activetab: 0 }
    });
    expect(wrapper.find(".timing-box").exists()).toBe(true);
  });

  it("renders timeline items for activetab=0 (new episodes)", () => {
    const wrapper = mount(TimingBox, {
      props: { timelineData: sampleData, activetab: 0 }
    });
    // Should show new episodes (new=true) sorted by favorites
    const items = wrapper.findAll(".card-timing");
    expect(items.length).toBeGreaterThanOrEqual(1);
  });

  it("renders items for a specific weekday tab", () => {
    const wrapper = mount(TimingBox, {
      props: { timelineData: sampleData, activetab: 1 }
    });
    // weekday=1 should show
    expect(wrapper.text()).toContain("Anime A");
  });

  it("does not render items for weekday with no matching data", () => {
    const wrapper = mount(TimingBox, {
      props: { timelineData: sampleData, activetab: 5 }
    });
    // No items with weekday=5
    const items = wrapper.findAll(".card-timing");
    expect(items.length).toBe(0);
  });

  it("renders items for activetab=7 (weekday=0 items)", () => {
    const wrapper = mount(TimingBox, {
      props: { timelineData: sampleData, activetab: 7 }
    });
    // weekday=0 items should show
    expect(wrapper.text()).toContain("Anime D");
  });

  it("renders update status text for bgmcount > 0", () => {
    const wrapper = mount(TimingBox, {
      props: { timelineData: [sampleData[0]], activetab: 0 }
    });
    expect(wrapper.text()).toContain("更新至");
  });

  it("renders 'not yet updated' for bgmcount = -1", () => {
    const wrapper = mount(TimingBox, {
      props: { timelineData: [sampleData[2]], activetab: 0 }
    });
    expect(wrapper.text()).toContain("尚未更新");
  });

  it("renders ep count with 'ep' suffix", () => {
    const wrapper = mount(TimingBox, {
      props: { timelineData: [sampleData[0]], activetab: 0 }
    });
    expect(wrapper.text()).toContain("5话");
  });

  it("renders images with v-lazy attribute (transformed by vue-lazyload)", () => {
    const wrapper = mount(TimingBox, {
      props: { timelineData: [sampleData[0]], activetab: 0 }
    });
    const img = wrapper.find("img");
    expect(img.exists()).toBe(true);
  });
});
