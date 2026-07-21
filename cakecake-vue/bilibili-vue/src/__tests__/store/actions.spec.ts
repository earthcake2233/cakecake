import { describe, it, expect, vi, beforeEach } from "vitest";

vi.stubEnv("VITE_MINIBILI_API", "false");

var mGAS=vi.fn(),mGL=vi.fn(),mGRI=vi.fn(),mGO=vi.fn(),mGDR=vi.fn(),mGN=vi.fn(),mGRR=vi.fn(),mGTG=vi.fn(),mRGG3=vi.fn(),mRGG7=vi.fn(),mGTC=vi.fn(),mRGC3=vi.fn(),mRGC7=vi.fn();

vi.mock("@/api",()=>({
  getAdSlide:(...a)=>mGAS(...a),getLocs:(...a)=>mGL(...a),getRankingIndex:(...a)=>mGRI(...a),
  getOnline:(...a)=>mGO(...a),getDynamicRegion:(...a)=>mGDR(...a),getNewlist:(...a)=>mGN(...a),
  getRankingRegion:(...a)=>mGRR(...a),getTimelineGlobal:(...a)=>mGTG(...a),getRankingGlobal3:(...a)=>mRGG3(...a),
  getRankingGlobal7:(...a)=>mRGG7(...a),getTimelineCn:(...a)=>mGTC(...a),getRankingCn3:(...a)=>mRGC3(...a),
  getRankingCn7:(...a)=>mRGC7(...a)
}));

describe("actions",()=>{
  beforeEach(()=>{vi.clearAllMocks();});

  it("setAdSlide",async()=>{
    mGAS.mockResolvedValue({result:{items:[1,2]}});
    var c=vi.fn();var{setAdSlide}=await import("@/store/actions");
    await setAdSlide({commit:c},{id:0,rid:1,position_id:5});
    expect(mGAS).toHaveBeenCalledWith({position_id:5});
    expect(c).toHaveBeenCalledWith("SET_AD_SLIDE",{id:0,rid:1,data:{items:[1,2]}});
  });

  it("setSlide non-minibili",async()=>{
    mGL.mockResolvedValue({data:{"23":[{"img":"a"}],"34":[{"img":"b"}]}});
    var c=vi.fn();var{setSlide}=await import("@/store/actions");
    await setSlide({commit:c},{});
    expect(c).toHaveBeenCalledWith("SET_SLIDE",[{"img":"a"}]);
    expect(c).toHaveBeenCalledWith("SET_POPULARIZE",[{"img":"b"}]);
  });

  it("setRankingIndex",async()=>{
    mGRI.mockResolvedValue({data:[{id:1}]});
    var c=vi.fn();var{setRankingIndex}=await import("@/store/actions");
    await setRankingIndex({commit:c},3);
    expect(c).toHaveBeenCalledWith("SET_RANKING_INDEX",{data:[{id:1}],day:3});
  });

  it("setOnline",async()=>{
    mGO.mockResolvedValue({data:12345});
    var c=vi.fn();var{setOnline}=await import("@/store/actions");
    await setOnline({commit:c});
    expect(c).toHaveBeenCalledWith("SET_ONLINE",12345);
  });

  it("setDynamicRegion",async()=>{
    mGDR.mockResolvedValue({data:[{num:5}]});
    var c=vi.fn();var{setDynamicRegion}=await import("@/store/actions");
    await setDynamicRegion({commit:c},{ps:20,rid:1,id:0});
    expect(mGDR).toHaveBeenCalledWith({ps:20,rid:1});
    expect(c).toHaveBeenCalledWith("SET_STOREY_DATA",{data:[{num:5}],id:0});
  });

  it("setNewlist",async()=>{
    mGN.mockResolvedValue({data:[{num:3}]});
    var c=vi.fn();var{setNewlist}=await import("@/store/actions");
    await setNewlist({commit:c},{ps:10,rid:3,id:1});
    expect(mGN).toHaveBeenCalledWith({ps:10,rid:3});
  });

  it("setRankingRegion default",async()=>{
    mGRR.mockResolvedValue({data:[{id:1}]});
    var c=vi.fn();var{setRankingRegion}=await import("@/store/actions");
    await setRankingRegion({commit:c},{rid:1,day:3,original:0,id:0});
    expect(mGRR).toHaveBeenCalledWith({rid:1,day:3,original:0});
  });

  it("setRankingRegion rid=13 d=3",async()=>{
    mRGG3.mockResolvedValue({result:[{id:"g3"}]});
    var c=vi.fn();var{setRankingRegion}=await import("@/store/actions");
    await setRankingRegion({commit:c},{rid:13,day:3,tag:1,original:0,id:0});
    expect(mRGG3).toHaveBeenCalled();
  });

  it("setRankingRegion rid=13 d=7",async()=>{
    mRGG7.mockResolvedValue({result:[{id:"g7"}]});
    var c=vi.fn();var{setRankingRegion}=await import("@/store/actions");
    await setRankingRegion({commit:c},{rid:13,day:7,tag:1,original:0,id:0});
    expect(mRGG7).toHaveBeenCalled();
  });

  it("setRankingRegion rid=168 d=3",async()=>{
    mRGC3.mockResolvedValue({result:[{id:"cn3"}]});
    var c=vi.fn();var{setRankingRegion}=await import("@/store/actions");
    await setRankingRegion({commit:c},{rid:168,day:3,tag:1,original:0,id:1});
    expect(mRGC3).toHaveBeenCalled();
  });

  it("setRankingRegion rid=168 d=7",async()=>{
    mRGC7.mockResolvedValue({result:[{id:"cn7"}]});
    var c=vi.fn();var{setRankingRegion}=await import("@/store/actions");
    await setRankingRegion({commit:c},{rid:168,day:7,tag:1,original:0,id:1});
    expect(mRGC7).toHaveBeenCalled();
  });

  it("setRankingRegion JSONP parse",async()=>{
    mRGG3.mockResolvedValue('callback({"result":[{"id":"g3"}]});');
    var c=vi.fn();var{setRankingRegion}=await import("@/store/actions");
    await setRankingRegion({commit:c},{rid:13,day:3,tag:1,original:0,id:0});
    expect(c).toHaveBeenCalledWith("SET_RANKING_DATA",{id:0,original:0,data:[{id:"g3"}]});
  });

  it("setRankingRegion non-string jsonChange",async()=>{
    mRGG3.mockResolvedValue(123);
    var c=vi.fn();var{setRankingRegion}=await import("@/store/actions");
    await setRankingRegion({commit:c},{rid:13,day:3,tag:1,original:0,id:0});
    expect(c).toHaveBeenCalledWith("SET_RANKING_DATA",{id:0,original:0,data:[]});
  });

  it("setTimeline rid=13",async()=>{
    mGTG.mockResolvedValue({result:[{ep:1}]});
    var c=vi.fn();var{setTimeline}=await import("@/store/actions");
    await setTimeline({commit:c},{id:0,rid:13});
    expect(c).toHaveBeenCalledWith("SET_TIMELINE_DATA",{id:0,rid:13,data:[{ep:1}]});
  });

  it("setTimeline rid=168",async()=>{
    mGTC.mockResolvedValue({result:[{ep:2}]});
    var c=vi.fn();var{setTimeline}=await import("@/store/actions");
    await setTimeline({commit:c},{id:1,rid:168});
    expect(c).toHaveBeenCalledWith("SET_TIMELINE_DATA",{id:1,rid:168,data:[{ep:2}]});
  });

  it("setTimeline unknown rid",async()=>{
    var c=vi.fn();var{setTimeline}=await import("@/store/actions");
    await setTimeline({commit:c},{id:0,rid:999});
    expect(c).not.toHaveBeenCalled();
  });

  it("setRankingRegion default-rid calls getRank with commit",async()=>{
    mGRR.mockResolvedValue({data:[{id:99}]});
    var c=vi.fn();var{setRankingRegion}=await import("@/store/actions");
    await setRankingRegion({commit:c},{rid:5,day:1,original:0,id:0});
    expect(c).toHaveBeenCalledWith("SET_RANKING_DATA",{id:0,original:0,data:[{id:99}]});
  });
});