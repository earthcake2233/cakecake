import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import MbCommentCuratedNotice from "@/components/minibili/MbCommentCuratedNotice.vue";

describe("MbCommentCuratedNotice.vue", () => {
  it("renders default text", () => {
    const wrapper = mount(MbCommentCuratedNotice);
    expect(wrapper.text()).toContain("up主精选");
  });

  it("renders custom text prop", () => {
    const wrapper = mount(MbCommentCuratedNotice, {
      props: { text: "自定义提示" }
    });
    expect(wrapper.text()).toContain("自定义提示");
  });

  it("uses provided avatar src", () => {
    const wrapper = mount(MbCommentCuratedNotice, {
      props: { avatarSrc: "https://example.com/av.jpg" }
    });
    const img = wrapper.find("img");
    expect(img.attributes("src")).toBe("https://example.com/av.jpg");
  });

  it("falls back to default avatar", () => {
    const wrapper = mount(MbCommentCuratedNotice);
    const img = wrapper.find("img");
    expect(img.attributes("src")).toBeTruthy();
  });
});
