import { describe, it, expect, vi, beforeEach } from "vitest";
import headerModule from "@/store/modules/header";
var mGL=vi.fn(),mGSDW=vi.fn(),mGS=vi.fn(),mGMI=vi.fn();
vi.mock("@/api",()=>({getLoc:(...a)=>mGL(...a),getSearchDefaultWords:(...a)=>mGSDW(...a),getSuggest:(...a)=>mGS(...a),getMenuIcon:(...a)=>mGMI(...a)}));
describe("header state",()=>{
  it("leftNav 7",()=>expect(headerModule.state.leftNav).toHaveLength(7));
  it("menuLeft[0]",()=>expect(headerModule.state.menuLeft[0].name).toBe("首页"));
});
describe("header mutations",()=>{
  var{mutations}=headerModule;var s;
  beforeEach(()=>{s=JSON.parse(JSON.stringify(headerModule.state));});
  it("SET_HEAD_BANNER",()=>{mutations.SET_HEAD_BANNER(s,[{title:"b1"}]);expect(s.headBanner.title).toBe("b1");});
  it("SET_SEARCH_DEFAULT_WORDS",()=>{mutations.SET_SEARCH_DEFAULT_WORDS(s,{show_name:"热搜"});expect(s.searchWord.show_name).toBe("热搜");});
  it("SET_MENUICON",()=>{mutations.SET_MENUICON(s,{icon:"x"});expect(s.menuIcon.icon).toBe("x");});
  it("SET_SUGGEST",()=>{mutations.SET_SUGGEST(s,{tag:[{name:"t"}]});expect(s.suggest.tag[0].name).toBe("t");});
});
describe("header actions",()=>{
  beforeEach(()=>{vi.clearAllMocks();});
  it("setHeadBanner",async()=>{mGL.mockResolvedValue({data:[{title:"b1"}]});var c=vi.fn();await headerModule.actions.setHeadBanner({commit:c},{id:1});expect(c).toHaveBeenCalledWith("SET_HEAD_BANNER",[{title:"b1"}]);});
  it("setSearchDefaultWords",async()=>{mGSDW.mockResolvedValue({data:{show_name:"热搜"}});var c=vi.fn();await headerModule.actions.setSearchDefaultWords({commit:c});expect(c).toHaveBeenCalledWith("SET_SEARCH_DEFAULT_WORDS",{show_name:"热搜"});});
  it("setMenuIcon",async()=>{mGMI.mockResolvedValue({data:{icon:"x"}});var c=vi.fn();await headerModule.actions.setMenuIcon({commit:c});expect(c).toHaveBeenCalledWith("SET_MENUICON",{icon:"x"});});
  it("setSuggest empty",async()=>{var c=vi.fn();await headerModule.actions.setSuggest({commit:c,state:{searchValue:""}});expect(c).toHaveBeenCalledWith("SET_SUGGEST",{tag:[]});expect(mGS).not.toHaveBeenCalled();});
  it("setSuggest term",async()=>{mGS.mockResolvedValue({result:{tag:[{name:"t"}]}});var c=vi.fn();await headerModule.actions.setSuggest({commit:c,state:{searchValue:"动漫"}});expect(c).toHaveBeenCalledWith("SET_SUGGEST",{tag:[{name:"t"}]});});
  it("setSuggest array",async()=>{mGS.mockResolvedValue({result:["a","b"]});var c=vi.fn();await headerModule.actions.setSuggest({commit:c,state:{searchValue:"x"}});expect(c).toHaveBeenCalledWith("SET_SUGGEST",{tag:["a","b"]});});
  it("setSuggest null",async()=>{mGS.mockResolvedValue({});var c=vi.fn();await headerModule.actions.setSuggest({commit:c,state:{searchValue:"x"}});expect(c).toHaveBeenCalledWith("SET_SUGGEST",{tag:[]});});
});