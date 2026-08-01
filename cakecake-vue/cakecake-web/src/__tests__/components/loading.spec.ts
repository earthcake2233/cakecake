import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import Loading from "@/components/loading/loading.vue";

describe("Loading.vue", () => {
  it("renders loading text", () => {
    const wrapper = mount(Loading);
    expect(wrapper.text()).toContain("加载中");
  });

  it("renders loading image", () => {
    const wrapper = mount(Loading);
    const img = wrapper.find("img");
    expect(img.exists()).toBe(true);
    expect(img.attributes("src")).toBeTruthy();
  });
});
