import { describe, it, expect, vi } from "vitest";
import { mount } from "@vue/test-utils";

vi.mock("@/utils/utils", () => ({
  count2: vi.fn((n) => String(n))
}));

const RouterLinkStub = {
  props: ["to", "custom", "v-slot"],
  template: '<a class="router-link-stub" :href="to"><slot :href="to" :navigate="()=>{}" /></a>'
};

import MbArticleSearchList from "@/components/search/MbArticleSearchList.vue";

const sampleItems = [
  { id: 1, title: "<b>Article 1</b>", desc: "Description 1", mid: 100, face: "https://example.com/face1.jpg", category_name: "Tech", stats: { view: 500 } },
  { id: 2, title: "Article 2", desc: "Description 2", mid: 200, face: "", category_name: "Life", stats: { view: 300 } }
];

describe("MbArticleSearchList.vue", () => {
  it("renders with empty items", () => {
    const wrapper = mount(MbArticleSearchList, {
      props: { items: [], numResults: 0 },
      global: { stubs: { "router-link": RouterLinkStub, MbSearchEmpty: true } }
    });
    expect(wrapper.find(".mb-article-search").exists()).toBe(true);
  });

  it("renders total label for numResults <= 1000", () => {
    const wrapper = mount(MbArticleSearchList, {
      props: { items: sampleItems, numResults: 42 },
      global: { stubs: { "router-link": RouterLinkStub, MbSearchEmpty: true } }
    });
    expect(wrapper.text()).toContain("42");
  });

  it("renders total label for numResults > 1000", () => {
    const wrapper = mount(MbArticleSearchList, {
      props: { items: sampleItems, numResults: 5000 },
      global: { stubs: { "router-link": RouterLinkStub, MbSearchEmpty: true } }
    });
    expect(wrapper.text()).toContain("1000+");
  });

  it("renders sort options", () => {
    const wrapper = mount(MbArticleSearchList, {
      props: { items: sampleItems, numResults: 10, sort: "default" },
      global: { stubs: { "router-link": RouterLinkStub, MbSearchEmpty: true } }
    });
    expect(wrapper.findAll(".mb-article-search__sort-item").length).toBeGreaterThan(0);
  });

  it("emits sort-change on sort click", async () => {
    const wrapper = mount(MbArticleSearchList, {
      props: { items: sampleItems, numResults: 10, sort: "default" },
      global: { stubs: { "router-link": RouterLinkStub, MbSearchEmpty: true } }
    });
    const sortItems = wrapper.findAll(".mb-article-search__sort-item a");
    if (sortItems.length > 0) {
      await sortItems[0].trigger("click.prevent");
      expect(wrapper.emitted("sort-change")).toBeTruthy();
    }
  });

  it("renders article rows", () => {
    const wrapper = mount(MbArticleSearchList, {
      props: { items: sampleItems, numResults: 2 },
      global: { stubs: { "router-link": RouterLinkStub, MbSearchEmpty: true } }
    });
    expect(wrapper.findAll(".mb-article-search__row").length).toBe(2);
  });

  it("renders article category", () => {
    const wrapper = mount(MbArticleSearchList, {
      props: { items: sampleItems, numResults: 2 },
      global: { stubs: { "router-link": RouterLinkStub, MbSearchEmpty: true } }
    });
    expect(wrapper.text()).toContain("Tech");
    expect(wrapper.text()).toContain("Life");
  });
});
