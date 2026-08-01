import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import SearchFilter from "@/components/searchFilter/searchFilter.vue";

describe("SearchFilter.vue", () => {
  it("renders with default viewMode=grid", () => {
    const wrapper = mount(SearchFilter);
    expect(wrapper.find(".filter-wrap").exists()).toBe(true);
    expect(wrapper.vm.viewMode).toBe("grid");
  });

  it("initially folded (fold=true)", () => {
    const wrapper = mount(SearchFilter);
    expect(wrapper.vm.fold).toBe(true);
  });

  it("renders order options", () => {
    const wrapper = mount(SearchFilter);
    expect(wrapper.findAll(".filter-item").length).toBeGreaterThan(0);
  });

  it("applies view-only class when viewOnly prop is true", () => {
    const wrapper = mount(SearchFilter, { props: { viewOnly: true } });
    expect(wrapper.find(".filter-wrap--view-only").exists()).toBe(true);
  });

  it("renders fold/unfold link", () => {
    const wrapper = mount(SearchFilter);
    expect(wrapper.find(".fold").exists()).toBe(true);
  });
});
