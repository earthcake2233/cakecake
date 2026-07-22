import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import AppFooter from "@/components/foot/footer.vue";

describe("AppFooter.vue", () => {
  it("renders copyright with current year", () => {
    const wrapper = mount(AppFooter);
    const year = new Date().getFullYear();
    expect(wrapper.text()).toContain("Copyright");
    expect(wrapper.text()).toContain(String(year));
  });

  it("renders all postcard sections", () => {
    const wrapper = mount(AppFooter);
    expect(wrapper.text()).toContain("cakecake");
    expect(wrapper.text()).toContain("传送门");
  });

  it("renders postcard items", () => {
    const wrapper = mount(AppFooter);
    expect(wrapper.text()).toContain("关于我们");
    expect(wrapper.text()).toContain("帮助中心");
  });

  it("renders mobile app download link", () => {
    const wrapper = mount(AppFooter);
    expect(wrapper.text()).toContain("手机端下载");
  });

  it("renders social media links", () => {
    const wrapper = mount(AppFooter);
    expect(wrapper.text()).toContain("新浪微博");
    expect(wrapper.text()).toContain("官方微信");
  });
});
