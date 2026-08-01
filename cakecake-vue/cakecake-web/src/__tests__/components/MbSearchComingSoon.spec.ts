import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import MbSearchComingSoon from "@/components/search/MbSearchComingSoon.vue";

describe("MbSearchComingSoon.vue", () => {
  it("renders coming soon text", () => {
    const wrapper = mount(MbSearchComingSoon);
    expect(wrapper.find(".mb-search-soon").exists()).toBe(true);
    expect(wrapper.text()).toContain("该功能即将开放");
  });

  it("renders image", () => {
    const wrapper = mount(MbSearchComingSoon);
    const img = wrapper.find("img");
    expect(img.exists()).toBe(true);
    expect(img.attributes("src")).toBeTruthy();
  });
});
