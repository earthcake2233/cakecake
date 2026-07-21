import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
vi.mock("@/utils/clearPageOverlays",()=>({clearStuckPageOverlays:vi.fn()}));
vi.mock("@/utils/authTokens",()=>({getAccessToken:vi.fn(),getRefreshToken:vi.fn()}));
vi.mock("@/utils/adminAuth",()=>({isAdminLoggedIn:vi.fn()}));
vi.mock("@/utils/minibiliTokenRefresh",()=>({isAccessTokenExpired:vi.fn(),refreshMinibiliAccessToken:vi.fn()}));
vi.mock("@/utils/notFoundRedirect",()=>({shouldRedirectVideoToNotFound:vi.fn()}));
vi.mock("element-plus",()=>({ElMessageBox:{close:vi.fn()}}));
var guard;
vi.mock("vue-router",()=>{
  var g;
  var router={beforeEach:vi.fn((fn)=>{g=fn;return router;}),afterEach:vi.fn()};
  return {createRouter:vi.fn(()=>router),createWebHashHistory:vi.fn()};
});
beforeAll(async()=>{
  vi.stubEnv("VITE_MINIBILI_API","true");
  // The mock createRouter is called with router options, and beforeEach captures the guard
  var router=await import("@/router/index");
  // Extract the guard from the beforeEach mock
  var call=router.default.beforeEach.mock.calls[0];
  if(call&&call[0])guard=call[0];
});
beforeEach(()=>{vi.clearAllMocks()});
describe("router guards",()=>{
  it("redirects to adminLogin when requireAdminAuth not met",async()=>{
    var{isAdminLoggedIn}=await import("@/utils/adminAuth");
    isAdminLoggedIn.mockReturnValue(false);
    var to={matched:[{meta:{requireAdminAuth:true}}],name:"admin"};
    var next=vi.fn(); await guard(to,{},next);
    expect(next).toHaveBeenCalledWith({name:"adminLogin",replace:true});
  });
  it("redirects admin login to banners when already logged in",async()=>{
    var{isAdminLoggedIn}=await import("@/utils/adminAuth");
    isAdminLoggedIn.mockReturnValue(true);
    var to={matched:[],name:"adminLogin"}; var next=vi.fn(); await guard(to,{},next);
    expect(next).toHaveBeenCalledWith({name:"adminBanners",replace:true});
  });
  it("redirects to notFound via shouldRedirectVideoToNotFound",async()=>{
    var{isAdminLoggedIn}=await import("@/utils/adminAuth");
    isAdminLoggedIn.mockReturnValue(false);
    var{shouldRedirectVideoToNotFound}=await import("@/utils/notFoundRedirect");
    shouldRedirectVideoToNotFound.mockReturnValue(true);
    var to={matched:[],name:"video"}; var next=vi.fn(); await guard(to,{},next);
    expect(next).toHaveBeenCalledWith({name:"notFound",replace:true});
  });
  it("calls next() when no guards triggered",async()=>{
    var{isAdminLoggedIn}=await import("@/utils/adminAuth");
    isAdminLoggedIn.mockReturnValue(false);
    var{shouldRedirectVideoToNotFound}=await import("@/utils/notFoundRedirect");
    shouldRedirectVideoToNotFound.mockReturnValue(false);
    var to={matched:[]}; var next=vi.fn(); await guard(to,{},next);
    expect(next).toHaveBeenCalledWith();
  });
  it("refreshes token for requireMinibiliAuth",async()=>{
    var{isAdminLoggedIn}=await import("@/utils/adminAuth");
    isAdminLoggedIn.mockReturnValue(false);
    var{shouldRedirectVideoToNotFound}=await import("@/utils/notFoundRedirect");
    shouldRedirectVideoToNotFound.mockReturnValue(false);
    var{getAccessToken,getRefreshToken}=await import("@/utils/authTokens");
    var{isAccessTokenExpired,refreshMinibiliAccessToken}=await import("@/utils/minibiliTokenRefresh");
    getAccessToken.mockReturnValue("expired");
    isAccessTokenExpired.mockReturnValue(true);
    getRefreshToken.mockReturnValue("rt");
    refreshMinibiliAccessToken.mockResolvedValue(true);
    var to={matched:[{meta:{requireMinibiliAuth:true}}]}; var next=vi.fn(); await guard(to,{},next);
    expect(refreshMinibiliAccessToken).toHaveBeenCalled(); expect(next).toHaveBeenCalledWith();
  });
  it("redirects home when requireMinibiliAuth fails",async()=>{
    var{isAdminLoggedIn}=await import("@/utils/adminAuth");
    isAdminLoggedIn.mockReturnValue(false);
    var{shouldRedirectVideoToNotFound}=await import("@/utils/notFoundRedirect");
    shouldRedirectVideoToNotFound.mockReturnValue(false);
    var{getAccessToken,getRefreshToken}=await import("@/utils/authTokens");
    var{isAccessTokenExpired,refreshMinibiliAccessToken}=await import("@/utils/minibiliTokenRefresh");
    getAccessToken.mockReturnValue("expired");
    isAccessTokenExpired.mockReturnValue(true);
    getRefreshToken.mockReturnValue("rt");
    refreshMinibiliAccessToken.mockResolvedValue(false);
    var to={matched:[{meta:{requireMinibiliAuth:true}}]}; var next=vi.fn(); await guard(to,{},next);
    expect(next).toHaveBeenCalledWith({path:"/",replace:true});
  });
});
