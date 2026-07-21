import { describe, it, expect, vi, beforeEach } from "vitest";

vi.stubEnv("VITE_MINIBILI_API", "true");

var mGHBP = vi.fn();

vi.mock("@/api/admin",()=>({getHomeBannersPublic:(...a)=>mGHBP(...a)}));

describe("actions setSlide (minibili API)",()=>{
  beforeEach(()=>{vi.clearAllMocks();});

  it("returns items from body.data",async()=>{
    mGHBP.mockResolvedValue({data:{items:[{img:"b1"},{img:"b2"}]}});
    var c=vi.fn();var{setSlide}=await import("@/store/actions");
    await setSlide({commit:c},{});
    expect(c).toHaveBeenCalledWith("SET_SLIDE",[{img:"b1"},{img:"b2"}]);
    expect(c).toHaveBeenCalledWith("SET_POPULARIZE",[]);
  });

  it("handles null data",async()=>{
    mGHBP.mockResolvedValue({data:null});
    var c=vi.fn();var{setSlide}=await import("@/store/actions");
    await setSlide({commit:c},{});
    expect(c).toHaveBeenCalledWith("SET_SLIDE",[]);
    expect(c).toHaveBeenCalledWith("SET_POPULARIZE",[]);
  });

  it("rejects gracefully",async()=>{
    mGHBP.mockRejectedValue(new Error("fail"));
    var c=vi.fn();var{setSlide}=await import("@/store/actions");
    setSlide({commit:c},{});
    // Flush microtasks so the .catch() handler runs
    await new Promise(resolve => setTimeout(resolve, 0));
    expect(c).toHaveBeenCalledWith("SET_SLIDE",[]);
    expect(c).toHaveBeenCalledWith("SET_POPULARIZE",[]);
  });
});