import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import ZoneModule from "@/components/zoneModule/zoneModule.vue";

describe("ZoneModule.vue", () => {
  it("renders wrapper with empty data", () => {
    const wrapper = mount(ZoneModule, {
      props: { moduledata: {} },
      global: {
        stubs: { "storey-box": true, "zone-rank": true }
      }
    });
    expect(wrapper.find(".zone-wrap-module").exists()).toBe(true);
  });

  it("renders zone-module inner div", () => {
    const wrapper = mount(ZoneModule, {
      props: { moduledata: [] },
      global: {
        stubs: { "storey-box": true, "zone-rank": true }
      }
    });
    expect(wrapper.find(".zone-module").exists()).toBe(true);
  });

  it("renders stubs for child components", () => {
    const wrapper = mount(ZoneModule, {
      props: { moduledata: { id: 1 } },
      global: {
        stubs: { "storey-box": true, "zone-rank": true }
      }
    });
    expect(wrapper.find(".zone-wrap-module").exists()).toBe(true);
  });
});
