import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import Dropdown from "@/components/dropdown/dropdown.vue";

const items = [
  { name: "三天" },
  { name: "一周" },
  { name: "一个月" }
];

describe("Dropdown.vue", () => {
  it("renders with data", () => {
    const wrapper = mount(Dropdown, {
      props: { dropdownData: items, selected: 0 }
    });
    expect(wrapper.find(".bili-dropdown").exists()).toBe(true);
  });

  it("shows selected item name", () => {
    const wrapper = mount(Dropdown, {
      props: { dropdownData: items, selected: 1 }
    });
    expect(wrapper.text()).toContain("一周");
  });

  it("emits selectClick on item click", async () => {
    const wrapper = mount(Dropdown, {
      props: { dropdownData: items, selected: 0 }
    });
    const listItems = wrapper.findAll(".dropdown-item");
    await listItems[0].trigger("click");
    expect(wrapper.emitted("selectClick")).toBeTruthy();
  });

  it("has correct number of dropdown data items", () => {
    const wrapper = mount(Dropdown, {
      props: { dropdownData: items, selected: 0 }
    });
    expect(wrapper.vm.dropdownData.length).toBe(3);
  });
});
