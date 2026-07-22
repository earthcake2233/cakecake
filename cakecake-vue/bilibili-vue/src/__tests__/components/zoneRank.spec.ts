import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import ZoneRank from "@/components/zoneRank/zoneRank.vue";

describe("ZoneRank.vue", () => {
  it("renders with empty zoneRank prop (no data)", () => {
    const wrapper = mount(ZoneRank, {
      props: { zoneRank: {}, scrollTop: 0, tag: 0, bangumiRankLists: 0 },
      global: { stubs: { Dropdown: true, AdSlide: true } }
    });
    expect(wrapper.find(".zone-rank").exists()).toBe(true);
  });

  it("renders rank head", () => {
    const wrapper = mount(ZoneRank, {
      props: { zoneRank: { rid: 1, rankAllData: [{ aid: 100, title: "Rank 1" }], offsetTop: 100, ranktab: [{name:"全部"}] },
        scrollTop: 200, tag: 0, bangumiRankLists: 0
      },
      global: { stubs: { Dropdown: true, AdSlide: true } }
    });
    expect(wrapper.find(".rank-head").exists()).toBe(true);
    expect(wrapper.text()).toContain("排行");
  });
});
