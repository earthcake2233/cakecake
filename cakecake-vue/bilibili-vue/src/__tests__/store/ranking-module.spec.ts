import { describe, it, expect, vi, beforeEach } from "vitest";

vi.stubEnv("VITE_MINIBILI_API", "false");

import rankingModule from "@/store/modules/ranking";

var mGR=vi.fn(),mGSR=vi.fn(),mGMR=vi.fn();
vi.mock("@/api",()=>({getRanking:(...a)=>mGR(...a),getSeasonRank:(...a)=>mGSR(...a),getMoviesRank:(...a)=>mGMR(...a)}));

describe("ranking state",()=>{
  it("5 menus",()=>expect(rankingModule.state.rankMenu).toHaveLength(5));
  it("loading true",()=>expect(rankingModule.state.loading).toBe(true));
});

describe("ranking mutations",()=>{
  var s;
  beforeEach(()=>{s=JSON.parse(JSON.stringify(rankingModule.state));});
  it("SET_RANK_ALL",()=>{rankingModule.mutations.SET_RANK_ALL(s,{note:"t",list:[{id:1}]});expect(s.rankAll).toEqual({note:"t",list:[{id:1}]});});
  it("SET_RANK_ALL null",()=>{rankingModule.mutations.SET_RANK_ALL(s,null);expect(s.rankAll).toEqual({note:"",list:[]});});
  it("SET_LOADING toggle",()=>{rankingModule.mutations.SET_LOADING(s);expect(s.loading).toBe(false);});
});

describe("ranking actions",()=>{
  beforeEach(()=>{vi.clearAllMocks();});

  it("case 0 (full) calls getRanking",async()=>{
    mGR.mockResolvedValue({data:{list:[{id:1}]}});
    var c=vi.fn();var st={...rankingModule.state,firstMenuActive:0};
    await rankingModule.actions.setRankData({commit:c,state:st},{rid:1,arc_type:0,day:3});
    expect(mGR).toHaveBeenCalledWith(1,1,0,3);
  });

  it("case 2 (bangumi) calls getSeasonRank",async()=>{
    mGSR.mockResolvedValue({result:{note:"s",list:[{id:3}]}});
    var c=vi.fn();var st={...rankingModule.state,firstMenuActive:2};
    await rankingModule.actions.setRankData({commit:c,state:st},{rid:1,day:3});
    expect(mGSR).toHaveBeenCalledWith(3,1);
  });

  it("case 3 (cinema) calls getMoviesRank",async()=>{
    mGMR.mockResolvedValue({rank:{note:"m",list:[{id:4}]}});
    var c=vi.fn();var st={...rankingModule.state,firstMenuActive:3};
    await rankingModule.actions.setRankData({commit:c,state:st},{rid:23,day:7});
    expect(mGMR).toHaveBeenCalledWith(7,23);
  });

  it("commits SET_LOADING on success",async()=>{
    mGR.mockResolvedValue({data:{list:[{id:1}]}});
    var c=vi.fn();var st={...rankingModule.state,firstMenuActive:0};
    await rankingModule.actions.setRankData({commit:c,state:st},{rid:1,arc_type:0,day:3});
    expect(c).toHaveBeenCalledWith("SET_LOADING");
  });

  it("comingSoon via minibili=true non-zero menu",async()=>{
    vi.stubEnv("VITE_MINIBILI_API","true");
    var c=vi.fn();var st={...rankingModule.state,firstMenuActive:1};
    await rankingModule.actions.setRankData({commit:c,state:st},{rid:1,arc_type:0,day:3});
    expect(c).toHaveBeenCalledWith("SET_RANK_ALL",{note:"",list:[],comingSoon:true});
    vi.stubEnv("VITE_MINIBILI_API","false");
  });
});