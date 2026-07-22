import { describe, it, expect, vi, beforeAll, beforeEach, afterAll } from "vitest";

var mockAxiosInstance;
var mockCreate = vi.fn(function(cfg) {
  mockAxiosInstance = Object.assign(vi.fn().mockResolvedValue({data:{code:0}}), {
    interceptors: { request: { use: vi.fn() }, response: { use: vi.fn() } },
    get: vi.fn(), post: vi.fn()
  });
  return mockAxiosInstance;
});

vi.mock("axios", function() { return { default: { create: mockCreate } }; });
vi.mock("element-plus", function() { return { ElMessage: vi.fn() }; });
vi.mock("@/utils/minibiliTokenRefresh", function() {
  return { ensureFreshAccessToken: vi.fn(), isAccessTokenExpired: vi.fn(), refreshMinibiliAccessToken: vi.fn(), shouldAttemptTokenRefresh: vi.fn() };
});
vi.mock("@/utils/minibiliAuthSync", function() { return { invalidateMinibiliSessionFromHttp: vi.fn() }; });
vi.mock("@/utils/authTokens", function() { return { getAccessToken: vi.fn(), getRefreshToken: vi.fn() }; });

var reqHandler, resSuccess, resError;

beforeAll(async function() {
  vi.stubEnv("VITE_MINIBILI_API", "true");
  vi.stubEnv("VITE_REMOTE_API_BASE", "");
  await import("@/utils/http");
  reqHandler = mockAxiosInstance.interceptors.request.use.mock.calls[0][0];
  resSuccess = mockAxiosInstance.interceptors.response.use.mock.calls[0][0];
  resError = mockAxiosInstance.interceptors.response.use.mock.calls[0][1];
});

afterAll(() => vi.unstubAllEnvs());

it("creates axios instance with correct config", function() {
  expect(mockCreate).toHaveBeenCalledWith({ withCredentials: false, baseURL: "", timeout: 15000 });
});

describe("request interceptor", function() {
  beforeEach(function() { vi.clearAllMocks(); });
  it("strips Content-Type for FormData", async function() {
    var fd = new FormData();fd.append("k","v");
    var cfg = { url:"/api/users",data:fd,headers:{"Content-Type":"json","content-type":"json"} };
    await reqHandler(cfg);
    expect(cfg.headers["Content-Type"]).toBeUndefined();
  });
  it("adds Bearer token", async function() {
    var at=await import("@/utils/authTokens");at.getAccessToken.mockReturnValue("jwt");at.getRefreshToken.mockReturnValue("rt");
    var tr=await import("@/utils/minibiliTokenRefresh");tr.isAccessTokenExpired.mockReturnValue(false);
    var cfg = { url:"/api/v1/videos",headers:{} };
    var r=await reqHandler(cfg);
    expect(r.headers.Authorization).toBe("Bearer jwt");
  });
  it("auto-refreshes when token expired", async function() {
    var at=await import("@/utils/authTokens");at.getRefreshToken.mockReturnValue("rt");
    var tr=await import("@/utils/minibiliTokenRefresh");tr.isAccessTokenExpired.mockReturnValue(true);tr.ensureFreshAccessToken.mockResolvedValue();
    await reqHandler({ url:"/api/v1/videos",headers:{} });
    expect(tr.ensureFreshAccessToken).toHaveBeenCalled();
  });
  it("skips auth for auth API URLs", async function() {
    var at=await import("@/utils/authTokens");at.getRefreshToken.mockReturnValue("rt");
    var tr=await import("@/utils/minibiliTokenRefresh");tr.isAccessTokenExpired.mockReturnValue(true);
    await reqHandler({ url:"/api/v1/auth/refresh",headers:{} });
    expect(tr.ensureFreshAccessToken).not.toHaveBeenCalled();
  });
});

describe("response success handler", function() {
  beforeEach(function() { vi.clearAllMocks(); });
  it("returns data on code 0", async function() {
    var r=await resSuccess({data:{code:0,data:{id:1}}});
    expect(r).toEqual({code:0,data:{id:1}});
  });
  it("rejects with fallback msg", async function() {
    await expect(resSuccess({data:{code:500}})).rejects.toThrow("请求失败");
  });
  it("invalidates session when refresh fails", async function() {
    var tr=await import("@/utils/minibiliTokenRefresh");tr.shouldAttemptTokenRefresh.mockReturnValue(true);tr.refreshMinibiliAccessToken.mockResolvedValue(false);
    var as=await import("@/utils/minibiliAuthSync");
    await expect(resSuccess({data:{code:40100,msg:"err"},config:{url:"/api/users"}})).rejects.toThrow("err");
    expect(as.invalidateMinibiliSessionFromHttp).toHaveBeenCalled();
  });
});

describe("response error handler", function() {
  beforeEach(function() { vi.clearAllMocks(); });
  it("shows ElMessage for errors", async function() {
    var{ElMessage}=await import("element-plus");
    await resError({response:{status:400,data:{msg:"err"}},config:{url:"/api/test"}}).catch(function(){});
    expect(ElMessage).toHaveBeenCalled();
  });
  it("skips ElMessage when skipGlobalErrorToast", async function() {
    var{ElMessage}=await import("element-plus");
    await resError({response:{status:400,data:{msg:"skip"}},config:{url:"/api/test",skipGlobalErrorToast:true}}).catch(function(){});
    expect(ElMessage).not.toHaveBeenCalled();
  });
  it("invalidates session on 401", async function() {
    var as=await import("@/utils/minibiliAuthSync");
    await resError({response:{status:401,data:{code:40100}},config:{url:"/api/videos"}}).catch(function(){});
    expect(as.invalidateMinibiliSessionFromHttp).toHaveBeenCalled();
  });
  it("propagates minibiliApiCode", async function() {
    try { await resError({response:{status:500,data:{msg:"fail"}},config:{url:"/api/v1"},minibiliApiCode:50001}); }
    catch(e) { expect(e.minibiliApiCode).toBe(50001); }
  });
});
