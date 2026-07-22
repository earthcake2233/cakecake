import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import Popularize from "@/components/popularize/popularize.vue";

describe("Popularize.vue", () => {
  it("renders default state", () => {
    const wrapper = mount(Popularize);
    expect(wrapper.find("#popularize_module").exists()).toBe(true);
  });

  it("renders r-con when online has default (empty array is truthy)", () => {
    const wrapper = mount(Popularize);
    // Default prop is [], which is truthy, so .r-con renders
    expect(wrapper.find(".r-con").exists()).toBe(true);
  });

  it("renders online data when provided as object", () => {
    const wrapper = mount(Popularize, {
      props: { online: { web_online: 12345, all_count: 67890 } }
    });
    expect(wrapper.text()).toContain("12345");
    expect(wrapper.text()).toContain("67890");
  });

  it("renders online text when values provided", () => {
    const wrapper = mount(Popularize, {
      props: { online: { web_online: 1000, all_count: 5000 } }
    });
    expect(wrapper.find(".online").exists()).toBe(true);
  });

  it("renders online links", () => {
    const wrapper = mount(Popularize, {
      props: { online: { web_online: 999, all_count: 8888 } }
    });
    expect(wrapper.text()).toContain("999");
    expect(wrapper.text()).toContain("8888");
  });
});
