import { describe, it, expect, vi, beforeEach } from "vitest";
import { mount } from "@vue/test-utils";

vi.mock("@/utils/utils", () => ({
  count2: vi.fn((n) => String(n)),
  timeChange: vi.fn((t) => "2025-01-01")
}));

vi.mock("@/utils/userLevel", () => ({
  levelIconUrl: vi.fn((lv) => "https://example.com/level_" + lv + ".png"),
  clampUserLevel: vi.fn((lv) => Math.min(Math.max(Number(lv) || 1, 1), 6))
}));

vi.mock("@/utils/minibiliRoutes", () => ({
  minibiliUserSpaceContributeVideoRoute: vi.fn((userId) => ({ name: "minibiliUserSpace", params: { userId } }))
}));

vi.mock("@/api/minibili", () => ({
  mbToggleUserFollow: vi.fn()
}));

vi.mock("@/utils/authTokens", () => ({
  getAccessToken: vi.fn(() => "mock-token"),
  getUserId: vi.fn(() => 999)
}));

import MbUserSearchList from "@/components/search/MbUserSearchList.vue";

const sampleUsers = [
  {
    mid: 100, uname: "User One", face: "https://example.com/face1.jpg", level: 5,
    usign: "Hello world", followed_by_me: false, archives: 10, fans: 500,
    archive_list: [{ aid: 1, title: "Video 1", pic: "", pubdate: 1700000000, rtype: "video" }]
  },
  {
    mid: 200, uname: "User Two", face: "", level: 3,
    usign: "", followed_by_me: true, archives: 5, fans: 100,
    archive_list: []
  }
];

describe("MbUserSearchList.vue", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("renders with empty items", () => {
    const wrapper = mount(MbUserSearchList, {
      props: { items: [], numResults: 0 },
      global: { stubs: { "router-link": true } }
    });
    expect(wrapper.find(".mb-user-search").exists()).toBe(true);
  });

  it("renders with user items", () => {
    const wrapper = mount(MbUserSearchList, {
      props: { items: sampleUsers, numResults: 2 },
      global: { stubs: { "router-link": true } }
    });
    expect(wrapper.vm.items.length).toBe(2);
  });

  it("computed totalLabel for numResults <= 1000", () => {
    const wrapper = mount(MbUserSearchList, {
      props: { items: sampleUsers, numResults: 42 },
      global: { stubs: { "router-link": true } }
    });
    expect(wrapper.vm.totalLabel).toContain("42");
  });

  it("computed totalLabel for numResults > 1000", () => {
    const wrapper = mount(MbUserSearchList, {
      props: { items: sampleUsers, numResults: 5000 },
      global: { stubs: { "router-link": true } }
    });
    expect(wrapper.vm.totalLabel).toContain("1000+");
  });

  it("applies compact class when compact prop is true", () => {
    const wrapper = mount(MbUserSearchList, {
      props: { items: sampleUsers, numResults: 2, compact: true },
      global: { stubs: { "router-link": true } }
    });
    expect(wrapper.find(".mb-user-search--compact").exists()).toBe(true);
  });

  it("applies user tab count when isUserTab", () => {
    const wrapper = mount(MbUserSearchList, {
      props: { items: sampleUsers, numResults: 2, isUserTab: true },
      global: { stubs: { "router-link": true } }
    });
    expect(wrapper.find(".mb-user-search__count").exists()).toBe(true);
  });

  it("calls levelBadgeSrc with clamped level", () => {
    const wrapper = mount(MbUserSearchList, {
      props: { items: sampleUsers, numResults: 2 },
      global: { stubs: { "router-link": true } }
    });
    const src = wrapper.vm.levelBadgeSrc(5);
    expect(src).toContain("5");
  });

  it("uses count2 for userCount", () => {
    const wrapper = mount(MbUserSearchList, {
      props: { items: sampleUsers, numResults: 2 },
      global: { stubs: { "router-link": true } }
    });
    expect(wrapper.vm.userCount(500)).toBe("500");
  });

  it("plainSign strips HTML tags", () => {
    const wrapper = mount(MbUserSearchList, {
      props: { items: sampleUsers, numResults: 2 },
      global: { stubs: { "router-link": true } }
    });
    expect(wrapper.vm.plainSign({ usign: "<b>test</b>" })).toBe("test");
  });

  it("formatDate uses timeChange", () => {
    const wrapper = mount(MbUserSearchList, {
      props: { items: sampleUsers, numResults: 2 },
      global: { stubs: { "router-link": true } }
    });
    expect(wrapper.vm.formatDate(1700000000)).toBe("2025-01-01");
  });
});
