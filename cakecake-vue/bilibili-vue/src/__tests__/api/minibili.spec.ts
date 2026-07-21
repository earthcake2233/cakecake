import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";

const mockHttp = { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() };
vi.mock("@/utils/http", function() { return { default: mockHttp }; });

var mb;
beforeAll(async function() { mb = await import("@/api/minibili"); });

function ok(data) { return { code: 0, msg: "ok", data: data }; }
var A = { skipGlobalErrorToast: true };
var AJ = { skipGlobalErrorToast: true, headers: { "Content-Type": "application/json" } };

beforeEach(function() { vi.clearAllMocks(); });

// === 20 existing tests maintained ===
describe("mbLogin",function(){it("POST /auth/login",async function(){
  mockHttp.post.mockResolvedValue(ok({ access_token:"a", refresh_token:"b" }));
  var r=await mb.mbLogin("u","p");
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/auth/login",{username:"u",password:"p"},A);
  expect(r.access_token).toBe("a");
});});

describe("mbRegister",function(){it("POST /users",async function(){
  mockHttp.post.mockResolvedValue(ok({ user_id:1 }));
  await mb.mbRegister("u","p");
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/users",{username:"u",password:"p"},A);
});});

describe("mbGetVideo",function(){it("GET /videos/:id",async function(){
  mockHttp.get.mockResolvedValue(ok({ id:42 }));
  var r=await mb.mbGetVideo(42);
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/videos/42",undefined);
  expect(r.id).toBe(42);
});});

describe("mbListVideos",function(){it("GET /videos with params",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[] }));
  await mb.mbListVideos({ limit:20, zone_parent:"动画" });
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/videos",{params:{limit:20,zone_parent:"动画"}});
});});

describe("mbListComments",function(){it("GET /videos/:id/comments",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[], comments_closed:false }));
  var r=await mb.mbListComments(42);
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/videos/42/comments");
});});

describe("mbToggleVideoLike",function(){it("POST /videos/:id/like",async function(){
  mockHttp.post.mockResolvedValue(ok({ liked:true }));
  var r=await mb.mbToggleVideoLike(42);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/videos/42/like",{},AJ);
});});

describe("mbPostComment",function(){it("POST /videos/:id/comments",async function(){
  mockHttp.post.mockResolvedValue(ok({ id:1 }));
  await mb.mbPostComment(42,"nice",0);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/videos/42/comments",{content:"nice",parent_id:0},AJ);
});});

describe("mbToggleLike",function(){it("POST /comments/:id/like",async function(){
  mockHttp.post.mockResolvedValue(ok({ liked:true }));
  var r=await mb.mbToggleLike(100);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/comments/100/like",{},A);
});});

describe("mbListMyVideos",function(){it("GET /users/me/videos",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[],page:1,page_size:10,total:0,total_pages:0,counts:{} }));
  await mb.mbListMyVideos({ page:1 });
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/videos",{params:{page:1},...A});
});});

describe("mbUploadVideo",function(){it("POST /videos with FormData",async function(){
  mockHttp.post.mockResolvedValue(ok({ id:1 }));
  var fd=new FormData();fd.append("t","x");
  await mb.mbUploadVideo(fd);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/videos",fd,{timeout:6e5,skipGlobalErrorToast:true});
});});

describe("mbGetUserPublic",function(){it("GET /space/:userId",async function(){
  mockHttp.get.mockResolvedValue(ok({ user_id:1 }));
  var r=await mb.mbGetUserPublic(1);
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/space/1",undefined);
});});

describe("mbToggleUserFollow",function(){it("POST /users/:userId/follow",async function(){
  mockHttp.post.mockResolvedValue(ok({ followed:true }));
  var r=await mb.mbToggleUserFollow(42);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/users/42/follow",{},A);
});});

describe("mbToggleVideoFavorite",function(){it("POST /videos/:id/favorite",async function(){
  mockHttp.post.mockResolvedValue(ok({ favorited:true }));
  var r=await mb.mbToggleVideoFavorite(42);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/videos/42/favorite",{},AJ);
});});

describe("mbToggleWatchLater",function(){it("POST /videos/:id/watch-later",async function(){
  mockHttp.post.mockResolvedValue(ok({ in_watch_later:true }));
  var r=await mb.mbToggleWatchLater(42);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/videos/42/watch-later",{},AJ);
});});

describe("mbListMyWatchLater",function(){it("GET /users/me/watch-later",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[],total:0,max_limit:100 }));
  await mb.mbListMyWatchLater();
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/watch-later",{params:undefined,...A});
});});

describe("mbListNotifications",function(){it("GET /notifications",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[],next_cursor:"" }));
  await mb.mbListNotifications({ category:"reply" });
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/notifications",{params:{category:"reply"},...A});
});});

describe("mbUnreadSummary",function(){it("GET /notifications/unread-summary",async function(){
  mockHttp.get.mockResolvedValue(ok({ my_message:3 }));
  var r=await mb.mbUnreadSummary();
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/notifications/unread-summary",A);
});});

describe("mbListMyFavorites",function(){it("GET /users/me/favorites",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[],total:0 }));
  await mb.mbListMyFavorites();
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/favorites",{params:undefined,...A});
});});

describe("mbPostDanmaku",function(){it("POST /videos/:id/danmaku",async function(){
  mockHttp.post.mockResolvedValue(ok({ id:1 }));
  var b={content:"h",color:"#fff",type:"scroll",video_time:1.5,font_size:"md"};
  await mb.mbPostDanmaku(42,b);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/videos/42/danmaku",b,AJ);
});});

describe("mbGetHotSearch",function(){it("GET /hot-search",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[] }));
  await mb.mbGetHotSearch(10);
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/hot-search",{params:{limit:10}});
});});

describe("error handling",function(){it("rejects non-zero code",async function(){
  mockHttp.get.mockResolvedValue({ code:40006,msg:"err",data:null });
  await expect(mb.mbGetVideo(42)).rejects.toThrow("err");
});});

// === NEW tests (30+ functions) ===
describe("mbGetMe",function(){it("GET /users/me",async function(){
  mockHttp.get.mockResolvedValue(ok({ user_id:1,nickname:"T" }));
  var r=await mb.mbGetMe();
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me");
  expect(r.nickname).toBe("T");
});});

describe("mbGetSearchSuggest",function(){it("GET /search/suggest",async function(){
  mockHttp.get.mockResolvedValue(ok({ tag:[{name:"t"}] }));
  var r=await mb.mbGetSearchSuggest("test");
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/search/suggest",{params:{term:"test",limit:10}});
});});

describe("mbGetSearchSuggest empty",function(){it("returns empty for blank term",async function(){
  var r=await mb.mbGetSearchSuggest("");
  expect(r).toEqual({tag:[]});
  expect(mockHttp.get).not.toHaveBeenCalled();
});});

describe("mbListMyFollowGroups",function(){it("GET /users/me/follow-groups",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[] }));
  await mb.mbListMyFollowGroups();
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/follow-groups",A);
});});

describe("mbDeleteMyVideo",function(){it("DELETE /videos/:id",async function(){
  mockHttp.delete.mockResolvedValue(ok({ ok:true }));
  var r=await mb.mbDeleteMyVideo(42);
  expect(mockHttp.delete).toHaveBeenCalledWith("/api/v1/videos/42",A);
  expect(r.ok).toBe(true);
});});

describe("mbDeleteDanmaku",function(){it("DELETE /danmakus/:id",async function(){
  mockHttp.delete.mockResolvedValue(ok({ id:1 }));
  var r=await mb.mbDeleteDanmaku(1);
  expect(mockHttp.delete).toHaveBeenCalledWith("/api/v1/danmakus/1",{...A});
  expect(r.id).toBe(1);
});});

describe("mbToggleDanmakuLike",function(){it("POST /danmakus/:id/like",async function(){
  mockHttp.post.mockResolvedValue(ok({ liked:true }));
  var r=await mb.mbToggleDanmakuLike(42);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/danmakus/42/like",undefined,{...A});
});});

describe("mbGetMeDailyRewards",function(){it("GET /users/me/daily-rewards",async function(){
  mockHttp.get.mockResolvedValue(ok({ watched_for_today:false }));
  var r=await mb.mbGetMeDailyRewards();
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/daily-rewards",A);
});});

describe("mbPostMeDailyRewardWatch",function(){it("POST /users/me/daily-rewards/watch",async function(){
  mockHttp.post.mockResolvedValue(ok({ watched_for_today:true }));
  var r=await mb.mbPostMeDailyRewardWatch();
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/users/me/daily-rewards/watch",{},A);
  expect(r.watched_for_today).toBe(true);
});});

describe("mbGetMeSpacePrivacy",function(){it("GET /users/me/space-privacy",async function(){
  mockHttp.get.mockResolvedValue(ok({ show_favorites:true }));
  await mb.mbGetMeSpacePrivacy();
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/space-privacy",A);
});});

describe("mbGetMySearchHistory",function(){it("GET /users/me/search-history",async function(){
  mockHttp.get.mockResolvedValue(ok({ keywords:["a","b"] }));
  var r=await mb.mbGetMySearchHistory();
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/search-history",A);
  expect(r.keywords).toEqual(["a","b"]);
});});

describe("mbLogout",function(){it("clears localStorage",function(){
  localStorage.setItem("minibili_access_token","tok");
  localStorage.setItem("minibili_refresh_token","rt");
  mb.mbLogout();
  expect(localStorage.getItem("minibili_access_token")).toBeNull();
  expect(localStorage.getItem("minibili_refresh_token")).toBeNull();
});});

describe("mbGetArticle",function(){it("GET /articles/:id",async function(){
  mockHttp.get.mockResolvedValue(ok({ id:1,title:"A" }));
  var r=await mb.mbGetArticle(1);
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/articles/1");
  expect(r.title).toBe("A");
});});

describe("mbGetMyArticle",function(){it("GET /users/me/articles/:id",async function(){
  mockHttp.get.mockResolvedValue(ok({ id:1 }));
  await mb.mbGetMyArticle(1);
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/articles/1",A);
});});

describe("mbDeleteMyArticle",function(){it("DELETE /users/me/articles/:id",async function(){
  mockHttp.delete.mockResolvedValue(ok({ ok:true }));
  var r=await mb.mbDeleteMyArticle(42);
  expect(mockHttp.delete).toHaveBeenCalledWith("/api/v1/users/me/articles/42",A);
  expect(r.ok).toBe(true);
});});

describe("mbListMyArticles",function(){it("GET /users/me/articles",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[],total:0 }));
  await mb.mbListMyArticles({ page:1 });
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/articles",{params:{page:1},...A});
});});

describe("mbListMyDynamics",function(){it("GET /users/me/dynamics",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[],next_cursor:"" }));
  await mb.mbListMyDynamics({ page:1 });
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/dynamics",{params:{page:1},...A});
});});

describe("mbDeleteMyDynamic",function(){it("DELETE /users/me/dynamics/:id",async function(){
  mockHttp.delete.mockResolvedValue(ok({ ok:true }));
  var r=await mb.mbDeleteMyDynamic(42);
  expect(mockHttp.delete).toHaveBeenCalledWith("/api/v1/users/me/dynamics/42",A);
  expect(r.ok).toBe(true);
});});

describe("mbGetUserDynamic",function(){it("GET /user-dynamics/:id",async function(){
  mockHttp.get.mockResolvedValue(ok({ id:1,content:"Hello" }));
  var r=await mb.mbGetUserDynamic(1);
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/user-dynamics/1");
  expect(r.content).toBe("Hello");
});});

describe("mbListMyFavoriteFolders",function(){it("GET /users/me/favorite-folders",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[] }));
  await mb.mbListMyFavoriteFolders();
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/favorite-folders",A);
});});

describe("mbGetVideoFavoritePicker",function(){it("GET /videos/:id/favorite-picker",async function(){
  mockHttp.get.mockResolvedValue(ok({ folders:[],max_count:10 }));
  var r=await mb.mbGetVideoFavoritePicker(42);
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/videos/42/favorite-picker",A);
  expect(r.max_count).toBe(10);
});});

describe("mbGetMeViewHistory",function(){it("GET /users/me/view-history",async function(){
  mockHttp.get.mockResolvedValue(ok({ items:[],total:0 }));
  await mb.mbGetMeViewHistory();
  expect(mockHttp.get).toHaveBeenCalledWith("/api/v1/users/me/view-history",A);
});});

describe("mbDeleteMeViewHistoryEntry",function(){it("DELETE /users/me/view-history/:id",async function(){
  mockHttp.delete.mockResolvedValue(ok({}));
  await mb.mbDeleteMeViewHistoryEntry(42);
  expect(mockHttp.delete).toHaveBeenCalledWith("/api/v1/users/me/view-history/42",A);
});});

describe("mbPostArticleView",function(){it("POST /articles/:id/view",async function(){
  mockHttp.post.mockResolvedValue(ok({ view_count:5 }));
  var r=await mb.mbPostArticleView(42);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/articles/42/view",null,A);
  expect(r.view_count).toBe(5);
});});

describe("mbToggleArticleFavorite",function(){it("POST /articles/:id/favorite",async function(){
  mockHttp.post.mockResolvedValue(ok({ favorited:true }));
  var r=await mb.mbToggleArticleFavorite(42);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/articles/42/favorite",null,A);
  expect(r.favorited).toBe(true);
});});

describe("mbApproveComment",function(){it("POST /comments/:id/approve",async function(){
  mockHttp.post.mockResolvedValue(ok({ approved:true }));
  await mb.mbApproveComment(100);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/comments/100/approve",null,A);
});});

describe("mbIgnoreCuratedComment",function(){it("POST /comments/:id/ignore-curated",async function(){
  mockHttp.post.mockResolvedValue(ok({ curated_ignored:true }));
  await mb.mbIgnoreCuratedComment(100);
  expect(mockHttp.post).toHaveBeenCalledWith("/api/v1/comments/100/ignore-curated",null,A);
});});

describe("mbPutMeViewHistorySettings",function(){it("PUT /users/me/view-history/settings",async function(){
  mockHttp.put.mockResolvedValue(ok({ paused:true }));
  var r=await mb.mbPutMeViewHistorySettings(true);
  expect(mockHttp.put).toHaveBeenCalledWith("/api/v1/users/me/view-history/settings",{paused:true},AJ);
  expect(r.paused).toBe(true);
});});

describe("mbClearMeViewHistory",function(){it("DELETE /users/me/view-history",async function(){
  mockHttp.delete.mockResolvedValue(ok({}));
  await mb.mbClearMeViewHistory();
  expect(mockHttp.delete).toHaveBeenCalledWith("/api/v1/users/me/view-history",A);
});});

describe("mbDeleteMeArticleViewHistoryEntry",function(){it("DELETE /users/me/view-history/articles/:id",async function(){
  mockHttp.delete.mockResolvedValue(ok({}));
  await mb.mbDeleteMeArticleViewHistoryEntry(99);
  expect(mockHttp.delete).toHaveBeenCalledWith("/api/v1/users/me/view-history/articles/99",A);
});});
