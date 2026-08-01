import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
vi.mock("@/utils/adminAuth",()=>({getAdminAccessToken:vi.fn(),getAdminRefreshToken:vi.fn(),setAdminTokens:vi.fn(),clearAdminTokens:vi.fn()}));
var m;var mc=vi.fn(()=>{m={defaults:{},interceptors:{request:{use:vi.fn()},response:{use:vi.fn()}},post:vi.fn()};return m;});
vi.mock("axios",()=>({default:{create:mc,post:vi.fn()}}));
vi.mock("element-plus",()=>({ElMessage:{error:vi.fn()}}));
var onFulfilled;var onRejected;var reqInt;
beforeAll(async()=>{
  vi.stubEnv("VITE_REMOTE_API_BASE","http://localhost:8080");
  await import("@/utils/adminHttp");
  // Capture interceptors before any clearAllMocks
  onFulfilled=m.interceptors.response.use.mock.calls[0][0];
  onRejected=m.interceptors.response.use.mock.calls[0][1];
  reqInt=m.interceptors.request.use.mock.calls[0][0];
});
beforeEach(()=>{vi.clearAllMocks()});
describe("res success",()=>{
  it("code 0",()=>{expect(onFulfilled({data:{code:0,data:{id:1}}})).toEqual({code:0,data:{id:1}});});
  it("non-zero",async()=>{await expect(onFulfilled({data:{code:1001,msg:"e"}})).rejects.toThrow("e");});
  it("fallback",async()=>{await expect(onFulfilled({data:{code:1001}})).rejects.toThrow("请求失败");});
});
describe("res error",()=>{
  it("400",async()=>{var{ElMessage}=await import("element-plus");await onRejected({response:{status:400,data:{msg:"e"}},config:{}}).catch(()=>{});expect(ElMessage.error).toHaveBeenCalledWith("e");});
  it("fallback",async()=>{var{ElMessage}=await import("element-plus");await onRejected({response:{status:500,data:{}},config:{},message:"srv"}).catch(()=>{});expect(ElMessage.error).toHaveBeenCalledWith("srv");});
  it("skip toast",async()=>{var{ElMessage}=await import("element-plus");await onRejected({response:{status:400,data:{msg:"skip"}},config:{skipGlobalErrorToast:true}}).catch(()=>{});expect(ElMessage.error).not.toHaveBeenCalled();});
  it("adminRetry",async()=>{var{ElMessage}=await import("element-plus");await onRejected({response:{status:401,data:{msg:"unauth"}},config:{_adminRetry:true}}).catch(()=>{});expect(ElMessage.error).toHaveBeenCalledWith("unauth");});
});
describe("request interceptor",()=>{
  it("adds Bearer token",async()=>{
    var{getAdminAccessToken}=await import("@/utils/adminAuth");
    getAdminAccessToken.mockReturnValue("tok123");
    var cfg=reqInt({headers:{}});
    expect(cfg.headers.Authorization).toBe("Bearer tok123");
  });
  it("skips Authorization when no token",async()=>{
    var{getAdminAccessToken}=await import("@/utils/adminAuth");
    getAdminAccessToken.mockReturnValue(null);
    var cfg=reqInt({headers:{}});
    expect(cfg.headers.Authorization).toBeUndefined();
  });
  it("deletes Content-Type for FormData",async()=>{
    var cfg=reqInt({headers:{"Content-Type":"multipart","content-type":"multipart"},data:new FormData()});
    expect(cfg.headers["Content-Type"]).toBeUndefined();
    expect(cfg.headers["content-type"]).toBeUndefined();
  });
});
describe("401 retry",()=>{
  it("retries with new token on 401",async()=>{
    var{getAdminRefreshToken,setAdminTokens,getAdminAccessToken}=await import("@/utils/adminAuth");
    var{default:axios}=await import("axios");
    getAdminRefreshToken.mockReturnValue("rt123");
    getAdminAccessToken.mockReturnValue("new_at");
    axios.post.mockResolvedValue({data:{code:0,data:{access_token:"new_at",refresh_token:"new_rt"}}});
    vi.stubGlobal("window",{location:{hash:""}});
    var cfg={url:"/admin/api",headers:{}};
    try{await onRejected({response:{status:401,data:{msg:"unauth"}},config:cfg});}catch(e){}
    expect(axios.post).toHaveBeenCalledWith("http://localhost:8080/api/v1/admin/auth/refresh",{refresh_token:"rt123"});
    expect(setAdminTokens).toHaveBeenCalledWith("new_at","new_rt");
  });
  it("redirects login on 401 when no refresh token",async()=>{
    var{getAdminRefreshToken,clearAdminTokens}=await import("@/utils/adminAuth");
    getAdminRefreshToken.mockReturnValue(null);
    var loc={hash:""};
    vi.stubGlobal("window",{location:loc});
    try{await onRejected({response:{status:401,data:{msg:"unauth"}},config:{}});}catch(e){}
    expect(clearAdminTokens).toHaveBeenCalled();
    expect(loc.hash).toBe("#/admin/login");
  });
});
