import { describe, it, expect, vi, beforeEach } from "vitest";

vi.stubEnv("VITE_MINIBILI_API","true");

var mMB=vi.fn();
vi.mock("@/api/minibili",()=>({mbGetMe:(...a)=>mMB(...a)}));
vi.mock("@/utils/authTokens",()=>({getAccessToken:vi.fn(),getRefreshToken:vi.fn(),clearTokens:vi.fn(),clearMinibiliPostLoginRedirect:vi.fn()}));
vi.mock("@/utils/minibiliTokenRefresh",()=>({refreshMinibiliAccessToken:vi.fn()}));
vi.mock("@/utils/coinBalance",()=>({coinBalanceNumber:vi.fn((n)=>n||0)}));
vi.mock("@/utils/imageCacheBust",()=>({resolveUserAvatarUrl:vi.fn((u)=>u)}));
vi.mock("@/utils/userLevel",()=>({levelInfoFromExperience:vi.fn(()=>({current_level:0}))}));

describe("login actions (minibili env)",()=>{
  beforeEach(()=>{vi.clearAllMocks();});

  it("refreshMinibiliMe no access token clears",async()=>{
    var at=await import("@/utils/authTokens");at.getAccessToken.mockReturnValue(null);at.getRefreshToken.mockReturnValue(null);
    var c=vi.fn();var{default:mod}=await import("@/store/modules/login");
    await mod.actions.refreshMinibiliMe({commit:c});
    expect(c).toHaveBeenCalledWith("SYNC_MINIBILI_ME",null);
  });

  it("refreshMinibiliMe refreshes and fetches",async()=>{
    var at=await import("@/utils/authTokens");var tr=await import("@/utils/minibiliTokenRefresh");
    at.getAccessToken.mockReturnValue(null);at.getRefreshToken.mockReturnValue("rt");
    tr.refreshMinibiliAccessToken.mockResolvedValue(true);
    mMB.mockResolvedValue({user_id:1,nickname:"Test"});
    var c=vi.fn();var{default:mod}=await import("@/store/modules/login");
    await mod.actions.refreshMinibiliMe({commit:c});
    expect(tr.refreshMinibiliAccessToken).toHaveBeenCalled();
    expect(c).toHaveBeenCalledWith("SYNC_MINIBILI_ME",{user_id:1,nickname:"Test"});
  });

  it("refreshMinibiliMe refresh fails clears",async()=>{
    var at=await import("@/utils/authTokens");var tr=await import("@/utils/minibiliTokenRefresh");
    at.getAccessToken.mockReturnValue(null);at.getRefreshToken.mockReturnValue("rt");
    tr.refreshMinibiliAccessToken.mockResolvedValue(false);
    var c=vi.fn();var{default:mod}=await import("@/store/modules/login");
    await mod.actions.refreshMinibiliMe({commit:c});
    expect(c).toHaveBeenCalledWith("SYNC_MINIBILI_ME",null);
  });

  it("refreshMinibiliMe refresh fails sets signIn=0",async()=>{
    var at=await import("@/utils/authTokens");var tr=await import("@/utils/minibiliTokenRefresh");
    at.getAccessToken.mockReturnValue(null);at.getRefreshToken.mockReturnValue("rt");
    tr.refreshMinibiliAccessToken.mockResolvedValue(false);
    var c=vi.fn();var{default:mod}=await import("@/store/modules/login");
    await mod.actions.refreshMinibiliMe({commit:c});
    expect(c).toHaveBeenCalledWith("SET_SIGNIN",{signIn:"0"});
  });

  it("refreshMinibiliMe with access token fetches directly",async()=>{
    var at=await import("@/utils/authTokens");at.getAccessToken.mockReturnValue("atok");
    mMB.mockResolvedValue({user_id:1,nickname:"Direct"});
    var c=vi.fn();var{default:mod}=await import("@/store/modules/login");
    await mod.actions.refreshMinibiliMe({commit:c});
    expect(mMB).toHaveBeenCalled();
    expect(c).toHaveBeenCalledWith("SYNC_MINIBILI_ME",{user_id:1,nickname:"Direct"});
  });

  it("refreshMinibiliMe with no access and no refresh sets signIn=0",async()=>{
    var at=await import("@/utils/authTokens");at.getAccessToken.mockReturnValue(null);at.getRefreshToken.mockReturnValue(null);
    var c=vi.fn();var{default:mod}=await import("@/store/modules/login");
    await mod.actions.refreshMinibiliMe({commit:c});
    expect(c).toHaveBeenCalledWith("SET_SIGNIN",{signIn:"0"});
  });
});