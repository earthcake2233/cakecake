import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { mount } from "@vue/test-utils";
import AdSlide from "@/components/ad/adSlide.vue";

const sampleData = [
  { link: "https://example.com/1", title: "Ad 1", img: "https://example.com/ad1.jpg" },
  { link: "https://example.com/2", title: "Ad 2", img: "https://example.com/ad2.jpg" }
];

describe("AdSlide.vue", () => {
  beforeEach(() => { vi.useFakeTimers(); });
  afterEach(() => { vi.useRealTimers(); });

  it("renders with empty data", () => {
    const wrapper = mount(AdSlide, { props: { slidedata: [] } });
    expect(wrapper.find(".slide").exists()).toBe(true);
  });

  it("renders slides with data", () => {
    const wrapper = mount(AdSlide, { props: { slidedata: sampleData } });
    const imgs = wrapper.findAll("img");
    expect(imgs.length).toBeGreaterThan(0);
  });

  it("renders pagination dots", () => {
    const wrapper = mount(AdSlide, { props: { slidedata: sampleData } });
    expect(wrapper.findAll(".slide-page li").length).toBe(2);
  });

  it("first dot is active initially", () => {
    const wrapper = mount(AdSlide, { props: { slidedata: sampleData } });
    const dots = wrapper.findAll(".slide-page li");
    expect(dots[0].classes()).toContain("on");
  });

  it("navigates via goto method with fake timers", () => {
    const wrapper = mount(AdSlide, { props: { slidedata: sampleData } });
    wrapper.vm.goto(1);
    vi.advanceTimersByTime(10);
    expect(wrapper.vm.nowIndex).toBe(1);
  });

  it("computed prevIndex wraps around", () => {
    const wrapper = mount(AdSlide, { props: { slidedata: sampleData } });
    wrapper.vm.nowIndex = 0;
    expect(wrapper.vm.prevIndex).toBe(1);
  });

  it("computed nextIndex wraps around", () => {
    const wrapper = mount(AdSlide, { props: { slidedata: sampleData } });
    wrapper.vm.nowIndex = 1;
    expect(wrapper.vm.nextIndex).toBe(0);
  });

  it("starts auto-advance when slidedata has items (watch triggers)", () => {
    const wrapper = mount(AdSlide, { props: { slidedata: sampleData } });
    expect(wrapper.vm.inVld).toBeDefined();
  });

  it("pauses interval on mouseover", () => {
    const wrapper = mount(AdSlide, { props: { slidedata: sampleData, slidetimedata: 1000 } });
    wrapper.find(".slide").trigger("mouseover");
    vi.advanceTimersByTime(3000);
    expect(wrapper.vm.nowIndex).toBe(0);
  });
});
