import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import MbSearchEmpty from "@/components/search/MbSearchEmpty.vue";

describe("MbSearchEmpty.vue", () => {
  it("renders default empty mode message", () => {
    const wrapper = mount(MbSearchEmpty);
    expect(wrapper.find(".mb-search-empty").exists()).toBe(true);
    expect(wrapper.text()).toContain("没有找到相关结果");
  });

  it("renders empty-user mode message", () => {
    const wrapper = mount(MbSearchEmpty, {
      props: { mode: "empty-user" }
    });
    expect(wrapper.text()).toContain("没有找到相关用户");
  });

  it("renders unavailable mode message", () => {
    const wrapper = mount(MbSearchEmpty, {
      props: { mode: "unavailable" }
    });
    expect(wrapper.text()).toContain("搜索服务暂未就绪");
  });

  it("renders empty image", () => {
    const wrapper = mount(MbSearchEmpty);
    const img = wrapper.find("img");
    expect(img.exists()).toBe(true);
    expect(img.attributes("src")).toBeTruthy();
  });

  it("renders hint text for default mode", () => {
    const wrapper = mount(MbSearchEmpty);
    expect(wrapper.text()).toContain("换个关键词试试吧");
  });

  it("renders hint for unavailable mode", () => {
    const wrapper = mount(MbSearchEmpty, {
      props: { mode: "unavailable" }
    });
    expect(wrapper.text()).toContain("Elasticsearch");
  });
});
