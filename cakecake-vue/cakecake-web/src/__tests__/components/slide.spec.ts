import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { mount } from "@vue/test-utils";
import Slide from "@/components/slide/slide.vue";

const sampleData = [
  { url: "https://example.com/1", name: "Slide 1", pic: "https://example.com/pic1.jpg" },
  { url: "https://example.com/2", name: "Slide 2", pic: "https://example.com/pic2.jpg" },
  { url: "https://example.com/3", name: "Slide 3", pic: "https://example.com/pic3.jpg" }
];

describe("Slide.vue", () => {
  beforeEach(() => { vi.useFakeTimers(); });
  afterEach(() => { vi.useRealTimers(); });

  it("renders with empty data", () => {
    const wrapper = mount(Slide, { props: { slidedata: [] } });
    expect(wrapper.find(".slide").exists()).toBe(true);
  });

  it("renders first slide image on mount", () => {
    const wrapper = mount(Slide, { props: { slidedata: sampleData } });
    const imgs = wrapper.findAll("img");
    expect(imgs.length).toBeGreaterThan(0);
  });

  it("renders pagination dots for each item", () => {
    const wrapper = mount(Slide, { props: { slidedata: sampleData } });
    expect(wrapper.findAll(".slide-page li").length).toBe(3);
  });

  it("first dot is active initially", () => {
    const wrapper = mount(Slide, { props: { slidedata: sampleData } });
    const dots = wrapper.findAll(".slide-page li");
    expect(dots[0].classes()).toContain("on");
  });

  it("navigates via goto method with fake timers", () => {
    const wrapper = mount(Slide, { props: { slidedata: sampleData } });
    wrapper.vm.goto(2);
    vi.advanceTimersByTime(10);
    expect(wrapper.vm.nowIndex).toBe(2);
  });

  it("computed prevIndex wraps around", () => {
    const wrapper = mount(Slide, { props: { slidedata: sampleData } });
    wrapper.vm.nowIndex = 0;
    expect(wrapper.vm.prevIndex).toBe(2);
    wrapper.vm.nowIndex = 1;
    expect(wrapper.vm.prevIndex).toBe(0);
  });

  it("computed nextIndex wraps around", () => {
    const wrapper = mount(Slide, { props: { slidedata: sampleData } });
    wrapper.vm.nowIndex = 2;
    expect(wrapper.vm.nextIndex).toBe(0);
    wrapper.vm.nowIndex = 0;
    expect(wrapper.vm.nextIndex).toBe(1);
  });

  it("sets up interval on mount (runInv)", () => {
    const wrapper = mount(Slide, { props: { slidedata: sampleData, slidetimedata: 1000 } });
    expect(wrapper.vm.inVld).toBeDefined();
  });

  it("pauses interval on mouseover (clearInv)", () => {
    const wrapper = mount(Slide, { props: { slidedata: sampleData, slidetimedata: 1000 } });
    const intervalId = wrapper.vm.inVld;
    wrapper.find(".slide").trigger("mouseover");
    // After clearInv, the interval should no longer fire
    vi.advanceTimersByTime(2000);
    expect(wrapper.vm.nowIndex).toBe(0);
  });

  it("renders prev/next buttons when pagation is true", () => {
    const wrapper = mount(Slide, { props: { slidedata: sampleData, pagation: true } });
    expect(wrapper.find(".slide-prev-button").exists()).toBe(true);
    expect(wrapper.find(".slide-next-button").exists()).toBe(true);
  });

  it("navigates to next via method and fake timers", () => {
    const wrapper = mount(Slide, { props: { slidedata: sampleData, pagation: true } });
    wrapper.vm.goto(wrapper.vm.nextIndex);
    vi.advanceTimersByTime(10);
    expect(wrapper.vm.nowIndex).toBe(1);
  });

  it("navigates to prev via method and fake timers", () => {
    const wrapper = mount(Slide, { props: { slidedata: sampleData, pagation: true } });
    wrapper.vm.nowIndex = 1;
    wrapper.vm.goto(wrapper.vm.prevIndex);
    vi.advanceTimersByTime(10);
    expect(wrapper.vm.nowIndex).toBe(0);
  });
});
