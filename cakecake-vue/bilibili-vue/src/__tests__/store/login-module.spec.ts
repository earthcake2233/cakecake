import { describe, it, expect, vi, beforeEach } from "vitest";
import loginModule from "@/store/modules/login";

var mGUS=vi.fn(),mGV=vi.fn(),mMB=vi.fn();
vi.mock("@/api",()=>({getUserInfo:(...a)=>mGUS(...a),getVipInfo:(...a)=>mGV(...a)}));
vi.mock("@/api/minibili",()=>({mbGetMe:(...a)=>mMB(...a)}));
vi.mock("@/utils/authTokens",()=>({getAccessToken:vi.fn(),getRefreshToken:vi.fn(),clearTokens:vi.fn(),clearMinibiliPostLoginRedirect:vi.fn()}));
vi.mock("@/utils/minibiliTokenRefresh",()=>({refreshMinibiliAccessToken:vi.fn()}));
vi.mock("@/utils/coinBalance",()=>({coinBalanceNumber:vi.fn((n)=>n||0)}));
vi.mock("@/utils/imageCacheBust",()=>({resolveUserAvatarUrl:vi.fn((u)=>u)}));
vi.mock("@/utils/userLevel",()=>({levelInfoFromExperience:vi.fn(()=>({current_level:0}))}));

describe("login state",()=>{
  it("loginShow false",()=>expect(loginModule.state.loginShow).toBe(false));
  it("signIn empty",()=>expect(loginModule.state.signIn).toBe(""));
  it("minibiliMe null",()=>expect(loginModule.state.minibiliMe).toBeNull());
});

describe("login mutations",()=>{
  var{mutations}=loginModule;var s;
  beforeEach(()=>{s=JSON.parse(JSON.stringify(loginModule.state));s.minibiliMe=null;s.avatarCacheBust=0;});
  it("SET_LOGIN_SHOW toggle",()=>{mutations.SET_LOGIN_SHOW(s);expect(s.loginShow).toBe(true);mutations.SET_LOGIN_SHOW(s);expect(s.loginShow).toBe(false);});
  it("OPEN_LOGIN_MODAL",()=>{mutations.OPEN_LOGIN_MODAL(s);expect(s.loginShow).toBe(true);});
  it("CLOSE_LOGIN_MODAL",()=>{mutations.CLOSE_LOGIN_MODAL(s);expect(s.loginShow).toBe(false);});
  it("SET_SIGNIN",()=>{mutations.SET_SIGNIN(s,{signIn:"1"});expect(s.signIn).toBe("1");});
  it("SYNC_MINIBILI_ME",()=>{mutations.SYNC_MINIBILI_ME(s,{user_id:1,nickname:"T",coin_balance:100});expect(s.minibiliMe).toBeTruthy();expect(s.proInfo.isLogin).toBe(true);expect(s.proInfo.money).toBe(100);});
  it("SYNC_MINIBILI_ME null",()=>{mutations.SYNC_MINIBILI_ME(s,null);expect(s.minibiliMe).toBeNull();expect(s.proInfo).toEqual([]);});
  it("BUMP_AVATAR_BUST",()=>{mutations.BUMP_AVATAR_BUST(s);expect(s.avatarCacheBust).toBeGreaterThan(0);});
  it("BUMP_AVATAR_BUST with me",()=>{mutations.SYNC_MINIBILI_ME(s,{user_id:2});mutations.BUMP_AVATAR_BUST(s);expect(s.avatarCacheBust).toBeGreaterThan(0);});
});

describe("login actions",()=>{
  beforeEach(()=>{vi.clearAllMocks();});
  it("setSignIn",()=>{var c=vi.fn();loginModule.actions.setSignIn({commit:c},{signIn:"1"});expect(c).toHaveBeenCalledWith("SET_SIGNIN",{signIn:"1"});});
  it("setUserInfo",async()=>{mGUS.mockResolvedValue({data:{isLogin:true}});var c=vi.fn();await loginModule.actions.setUserInfo({commit:c});expect(c).toHaveBeenCalledWith("SET_USER_INFO",{proInfo:{isLogin:true}});});
  it("setVipInfo",async()=>{mGV.mockResolvedValue({data:{picAndWords:["v"]}});var c=vi.fn();await loginModule.actions.setVipInfo({commit:c});expect(c).toHaveBeenCalledWith("SET_VIP_INFO",{topInfo:{picAndWords:["v"]}});});
});