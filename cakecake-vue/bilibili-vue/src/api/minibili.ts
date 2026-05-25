/**
 * Mini-Bili 后端 REST 封装（Rule R-FE-5：显式类型，禁止 any）
 */
import type { AxiosRequestConfig } from "axios";
import http from "../utils/http";
import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  setMinibiliDisplayName,
  setTokens
} from "../utils/authTokens";

const authAxiosOpts: AxiosRequestConfig = {
  skipGlobalErrorToast: true
};

export interface ApiEnvelope<T> {
  code: number;
  msg: string;
  data: T;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface VideoListItem {
  id: number;
  user_id?: number;
  title: string;
  /** 列表接口与 videoCard 一致时返回，用于空间动态等 */
  description?: string;
  cover_url: string;
  play_count: number;
  danmaku_count: number;
  comment_count?: number;
  /** 视频获赞数（POST /videos/:id/like 维护） */
  like_count?: number;
  fav_count?: number;
  coin_count?: number;
  /** 当前登录用户是否已赞该视频（OptionalJWT 列表） */
  liked_by_me?: boolean;
  favorited_by_me?: boolean;
  coined_by_me?: boolean;
  /** 当前用户已给该视频投币数量（0–2） */
  my_coin_amount?: number;
  daily_coin_exp_progress?: number;
  daily_coin_exp_max?: number;
  in_watch_later?: boolean;
  duration: number;
  uploader: string;
  created_at: string;
  zone_parent?: string;
  zone_child?: string;
  category?: string;
  /** UP 关闭评论区 */
  comments_closed?: boolean;
  /** UP 开启评论精选 */
  comments_curated?: boolean;
  /** UP 关闭弹幕发送 */
  danmaku_closed?: boolean;
}

export interface VideoListPayload {
  items: VideoListItem[];
  next_cursor: string;
}

/** GET /users/me/videos 列表项（字段少于详情接口） */
export interface MyVideoListItem {
  id: number;
  title: string;
  status: string;
  fail_reason: string;
  cover_url: string;
  duration: number;
  play_count: number;
  danmaku_count: number;
  comment_count?: number;
  fav_count?: number;
  coin_count?: number;
  /** 标签（与详情接口一致，JSON 来自后端） */
  tags?: string[];
  created_at: string;
}

export interface MyVideoListCounts {
  draft: number;
  processing: number;
  passed: number;
  rejected: number;
}

export interface MyVideoListPayload {
  items: MyVideoListItem[];
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
  counts: MyVideoListCounts;
}

export interface VideoDetail {
  id: number;
  /** 稿件作者用户 ID（用于评论区 UP 标识与菜单权限） */
  user_id?: number;
  title: string;
  description: string;
  play_count: number;
  danmaku_count: number;
  comment_count: number;
  like_count?: number;
  fav_count?: number;
  coin_count?: number;
  liked_by_me?: boolean;
  favorited_by_me?: boolean;
  coined_by_me?: boolean;
  /** 当前用户已给该视频投币数量（0–2） */
  my_coin_amount?: number;
  daily_coin_exp_progress?: number;
  daily_coin_exp_max?: number;
  in_watch_later?: boolean;
  /** 当前已连接弹幕 WebSocket 的人数（与侧栏「正在看」一致） */
  watching_count?: number;
  duration: number;
  uploader: string;
  /** 作者头像（OSS URL，可能为空） */
  uploader_avatar_url?: string;
  /** 当前登录用户是否已关注 UP 主 */
  followed_by_me?: boolean;
  uploader_sign?: string;
  uploader_follower_count?: number;
  uploader_published_count?: number;
  created_at: string;
  video_url: string;
  cover_url: string;
  status: string;
  fail_reason: string;
  /** 草稿是否仍有本地源文件可预览（仅作者拉取 draft-source） */
  draft_has_source?: boolean;
  /** 展示用标签（上传/编辑时写入） */
  tags?: string[];
  /** 投稿分区，如「动画」「生活-日常」 */
  zone?: string;
  zone_parent?: string;
  zone_child?: string;
  /** 分区展示文案，如「生活 > 日常」 */
  category?: string;
  comments_closed?: boolean;
  comments_curated?: boolean;
  danmaku_closed?: boolean;
}

export interface CommentItem {
  id: number;
  user_id: number;
  username: string;
  /** 评论者头像（与 users.avatar_url 一致） */
  avatar_url?: string;
  parent_id: number;
  /** 楼中楼层级深度 1–3，非用户账号等级 */
  level: number;
  /** 评论者账号等级 Lv1–Lv6 */
  user_level?: number;
  content: string;
  like_count: number;
  created_at: string;
  /** 当前请求用户是否已赞（未带 Token 时为 false） */
  liked_by_me?: boolean;
  /** 当前请求用户是否已踩 */
  disliked_by_me?: boolean;
  /** 是否为置顶根评论 */
  pinned?: boolean;
  /** 评论者是否为该视频作者 */
  is_by_uploader?: boolean;
  /** 发帖时解析的 IP 属地（省级简称），如「广东」 */
  ip_location?: string;
}

export interface NotificationRow {
  id: number;
  type: string;
  message: string;
  comment_preview: string;
  is_read: boolean;
  total_likes: number;
  created_at: string;
  /** reply_to_comment | video_comment */
  inbox_kind?: string;
  sender_username?: string;
  sender_avatar_url?: string;
  reply_content?: string;
  parent_content_preview?: string;
  video_id?: number;
  article_id?: number;
  article_cover_url?: string;
  reply_comment_id?: number;
  parent_comment_id?: number;
  video_title?: string;
  video_cover_url?: string;
  /** 当前用户对「对方这条评论」是否已赞（仅 reply_received 列表） */
  liked_by_me?: boolean;
  /** like_aggregation：点赞者用户名列表 */
  sender_names?: string[];
  /** like_aggregation：与 sender_names 顺序对应的前两个头像 URL */
  sender_avatar_urls?: string[];
  /** like_aggregation：与 sender_names 顺序对应的前两个用户 ID（用于跳转个人空间） */
  sender_user_ids?: number[];
  /** like_aggregation：「评论」或「弹幕」 */
  like_target?: string;
  /** like_aggregation：用户已选「不再通知」，新点赞不再聚合更新 */
  likes_muted?: boolean;
  /** like_aggregation：被点赞的评论 ID（用于跳转视频内定位） */
  liked_comment_id?: number;
  /** like_aggregation：被点赞评论全文（详情页标题） */
  comment_full_text?: string;
}

export interface LikeNotifLikerRow {
  id: number;
  user_id: number;
  username: string;
  avatar_url: string;
  created_at: string;
  followed_by_me?: boolean;
}

function unwrap<T>(r: unknown): T {
  const o = r as ApiEnvelope<T | null>;
  if (!o || typeof o.code !== "number") {
    throw new Error("无效响应");
  }
  if (o.code !== 0) {
    throw new Error(o.msg || "请求失败");
  }
  return o.data as T;
}

export async function mbListVideos(params?: {
  limit?: number;
  cursor?: string;
  zone_parent?: string;
  sort?: "hot" | "time";
  /** 排行榜：1|3|7|30 天 */
  days?: number;
  /** 0=全部投稿 1=近期投稿（仅 days 天内发布） */
  arc_type?: number;
}): Promise<VideoListPayload> {
  const r = await http.get("/api/v1/videos", { params });
  return unwrap<VideoListPayload>(r);
}

/** GET /hot-search — 动态页侧栏热搜（Redis 统计用户搜索词） */
export interface HotSearchItem {
  rank: number;
  title: string;
  badge: string;
  video_id?: number;
  play_count?: number;
}

export async function mbGetHotSearch(
  limit = 10
): Promise<{ items: HotSearchItem[] }> {
  const r = await http.get("/api/v1/hot-search", { params: { limit } });
  return unwrap<{ items: HotSearchItem[] }>(r);
}

/** GET /search/suggest — 搜索框联想词（B 站 suggest.tag 结构） */
export interface SearchSuggestTag {
  name: string;
  value: string;
}

export async function mbGetSearchSuggest(
  term: string,
  limit = 10
): Promise<{ tag: SearchSuggestTag[] }> {
  const q = String(term || "").trim();
  if (!q) {
    return { tag: [] };
  }
  const r = await http.get("/api/v1/search/suggest", {
    params: { term: q, limit }
  });
  return unwrap<{ tag: SearchSuggestTag[] }>(r);
}

/** Bilibili-style account level block (GET /users/me, GET /space/:userId). */
export interface UserLevelInfo {
  current_level: number;
  current_min: number;
  current_exp: number;
  next_exp: number;
}

/** GET /space/:userId — 个人空间顶栏用公开资料（无需登录） */
export interface UserPublicProfile {
  user_id: number;
  nickname: string;
  cake_id: string;
  avatar_url: string;
  sign: string;
  /** 个人空间侧栏公告（≤150 字，与个性签名独立） */
  announcement: string;
  /** 若后端在公开空间接口中返回，侧栏可展示他人生日 */
  birthday?: string;
  /** male | female | secret，与 /users/me 一致 */
  gender?: string;
  privacy?: SpacePrivacySettings;
  /** 当前登录用户是否为该空间主人（OptionalJWT） */
  is_owner?: boolean;
  following_count?: number;
  follower_count?: number;
  published_count?: number;
  followed_by_me?: boolean;
  level_info?: UserLevelInfo;
}

export interface SpaceRelationUser {
  user_id: number;
  nickname: string;
  sign: string;
  avatar_url: string;
  followed_at?: string;
  mutual?: boolean;
}

export interface FollowGroup {
  id: number;
  name: string;
  member_count: number;
  created_at?: string;
}

export async function mbListMyFollowGroups(): Promise<{ items: FollowGroup[] }> {
  const r = await http.get("/api/v1/users/me/follow-groups", authAxiosOpts);
  return unwrap(r);
}

export async function mbCreateFollowGroup(body: {
  name: string;
}): Promise<FollowGroup> {
  const r = await http.post("/api/v1/users/me/follow-groups", body, authAxiosOpts);
  return unwrap(r);
}

export async function mbUpdateFollowGroup(
  groupId: number,
  body: { name: string }
): Promise<FollowGroup> {
  const r = await http.put(
    `/api/v1/users/me/follow-groups/${groupId}`,
    body,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbDeleteFollowGroup(groupId: number): Promise<{ deleted: boolean }> {
  const r = await http.delete(
    `/api/v1/users/me/follow-groups/${groupId}`,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbListFolloweeGroupIds(
  followeeId: number
): Promise<{ group_ids: number[] }> {
  const r = await http.get(
    `/api/v1/users/me/following/${followeeId}/groups`,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbAddFollowGroupMember(
  groupId: number,
  followeeId: number
): Promise<{ added: boolean; group_id: number; followee_id: number }> {
  const r = await http.post(
    `/api/v1/users/me/follow-groups/${groupId}/members`,
    { followee_id: followeeId },
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbRemoveFollowGroupMember(
  groupId: number,
  followeeId: number
): Promise<{ removed: boolean }> {
  const r = await http.delete(
    `/api/v1/users/me/follow-groups/${groupId}/members/${followeeId}`,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbListUserFollowing(
  userId: number,
  params?: { limit?: number; groupId?: number },
  config?: AxiosRequestConfig
): Promise<{ items: SpaceRelationUser[]; total: number; group_id?: number }> {
  const r = await http.get(`/api/v1/space/${userId}/following`, {
    ...config,
    params,
    skipGlobalErrorToast: true
  });
  return unwrap(r);
}

export async function mbListUserFollowers(
  userId: number,
  params?: { limit?: number },
  config?: AxiosRequestConfig
): Promise<{ items: SpaceRelationUser[]; total: number }> {
  const r = await http.get(`/api/v1/space/${userId}/followers`, {
    ...config,
    params,
    skipGlobalErrorToast: true
  });
  return unwrap(r);
}

export async function mbToggleUserFollow(userId: number): Promise<{
  followed: boolean;
  follower_count: number;
}> {
  const r = await http.post(
    `/api/v1/users/${userId}/follow`,
    {},
    authAxiosOpts
  );
  return unwrap(r);
}

export interface SpacePrivacySettings {
  public_favorites: boolean;
  public_recent_coins: boolean;
  public_following: boolean;
  public_fans: boolean;
  public_birthday: boolean;
}

export async function mbGetUserPublic(
  userId: number,
  axiosConfig?: AxiosRequestConfig
): Promise<UserPublicProfile> {
  const r = await http.get(`/api/v1/space/${userId}`, axiosConfig);
  return unwrap<UserPublicProfile>(r);
}

/** GET /space/:userId/videos — 该用户已上架稿件（与首页列表项结构一致） */
export async function mbListUserPublishedVideos(
  userId: number,
  params?: { limit?: number; cursor?: string }
): Promise<VideoListPayload> {
  const r = await http.get(`/api/v1/space/${userId}/videos`, { params });
  return unwrap<VideoListPayload>(r);
}

export async function mbGetVideo(
  id: number,
  axiosConfig?: AxiosRequestConfig
): Promise<VideoDetail> {
  const r = await http.get(`/api/v1/videos/${id}`, axiosConfig);
  return unwrap<VideoDetail>(r);
}

export async function mbUpdateMyVideo(
  id: number,
  body: {
    title: string;
    description: string;
    tags?: string[];
    zone?: string;
  }
): Promise<{ ok: boolean }> {
  const r = await http.put(`/api/v1/videos/${id}`, body, {
    ...authAxiosOpts,
    headers: { "Content-Type": "application/json" }
  });
  return unwrap(r);
}

export async function mbUpdateVideoCover(
  videoId: number,
  file: File
): Promise<{ cover_url: string }> {
  const fd = new FormData();
  fd.append("cover", file);
  const r = await http.put(`/api/v1/videos/${videoId}/cover`, fd, {
    ...authAxiosOpts,
    timeout: 120000
  });
  return unwrap(r);
}

export async function mbLogin(
  username: string,
  password: string
): Promise<TokenPair> {
  const u = String(username || "").trim();
  const r = await http.post(
    "/api/v1/auth/login",
    { username: u, password },
    { ...authAxiosOpts }
  );
  const data = unwrap<TokenPair>(r);
  setTokens(data.access_token, data.refresh_token);
  setMinibiliDisplayName(u);
  return data;
}

export async function mbRegister(
  username: string,
  password: string
): Promise<{ user_id: number; username: string }> {
  const u = String(username || "").trim();
  const r = await http.post(
    "/api/v1/users",
    { username: u, password },
    { ...authAxiosOpts }
  );
  return unwrap(r);
}

/** 注册成功后用同一凭据登录（失败时由调用方处理提示） */
export async function mbRegisterThenLogin(
  username: string,
  password: string
): Promise<TokenPair> {
  await mbRegister(username, password);
  return mbLogin(username, password);
}

export async function mbRefresh(): Promise<TokenPair> {
  const { refreshMinibiliAccessToken } = await import(
    "../utils/minibiliTokenRefresh"
  );
  const ok = await refreshMinibiliAccessToken();
  if (!ok) {
    throw new Error("登录已过期，请重新登录");
  }
  return {
    access_token: getAccessToken(),
    refresh_token: getRefreshToken()
  };
}

export function mbLogout(): void {
  clearTokens();
}

export interface UserMe {
  user_id: number;
  username: string;
  /** 公开展示的固定账号 id（个人中心「用户名」） */
  cake_id: string;
  nickname: string;
  sign: string;
  /** 个人空间侧栏公告 */
  announcement?: string;
  gender: string;
  birthday: string;
  avatar_url: string;
  created_at: string;
  /** 首条成功上架视频创建时间（RFC3339）；删稿后仍保留，供「成为 UP 主」天数 */
  first_published_at?: string | null;
  /** 服务端按 first_published_at 计算的累计天数（第 1 天起算）；无已上架稿时为 0 */
  creator_up_days?: number;
  /** 冷静期内：已提交注销申请且尚未到期 */
  pending_deletion?: boolean;
  /** RFC3339，期满后将执行永久匿名化 */
  deletion_effective_at?: string | null;
  space_privacy?: SpacePrivacySettings;
  level_info?: UserLevelInfo;
  /** 当前用户硬币余额（可为小数，如 UP 分成 0.1） */
  coin_balance?: number;
}

export async function mbGetMeSpacePrivacy(): Promise<SpacePrivacySettings> {
  const r = await http.get("/api/v1/users/me/space-privacy", authAxiosOpts);
  return unwrap<SpacePrivacySettings>(r);
}

export async function mbPutMeSpacePrivacy(
  body: Partial<SpacePrivacySettings>
): Promise<SpacePrivacySettings> {
  const r = await http.put("/api/v1/users/me/space-privacy", body, authAxiosOpts);
  return unwrap<SpacePrivacySettings>(r);
}

export async function mbGetMe(): Promise<UserMe> {
  const r = await http.get("/api/v1/users/me");
  return unwrap<UserMe>(r);
}

export interface DailyRewardTaskItem {
  exp: number;
  done: boolean;
}

export interface DailyRewardCoinTask extends DailyRewardTaskItem {
  progress: number;
  max: number;
}

export interface DailyRewardsSnapshot {
  login: DailyRewardTaskItem;
  watch: DailyRewardTaskItem;
  coin: DailyRewardCoinTask;
  share: DailyRewardTaskItem;
}

export async function mbGetMeDailyRewards(): Promise<DailyRewardsSnapshot> {
  const r = await http.get("/api/v1/users/me/daily-rewards", authAxiosOpts);
  return unwrap<DailyRewardsSnapshot>(r);
}

export async function mbPostMeDailyRewardWatch(): Promise<DailyRewardsSnapshot> {
  const r = await http.post(
    "/api/v1/users/me/daily-rewards/watch",
    {},
    authAxiosOpts
  );
  return unwrap<DailyRewardsSnapshot>(r);
}

export interface CoinLedgerItem {
  created_at: string;
  change: number;
  reason: string;
}

export interface CoinLedgerPage {
  range: "month" | "week";
  total: number;
  has_more: boolean;
  items: CoinLedgerItem[];
}

export async function mbGetMeCoinLedger(params?: {
  range?: "month" | "week";
  limit?: number;
  offset?: number;
}): Promise<CoinLedgerPage> {
  const r = await http.get("/api/v1/users/me/coin-ledger", {
    ...authAxiosOpts,
    params: {
      range: params?.range || "month",
      limit: params?.limit ?? 10,
      offset: params?.offset ?? 0
    }
  });
  return unwrap<CoinLedgerPage>(r);
}

export interface ViewHistoryItem {
  media_type?: "video" | "article";
  video_id: number;
  article_id?: number;
  title: string;
  cover_url: string;
  duration_sec: number;
  progress_sec: number;
  device: "web" | "mobile" | string;
  viewed_at: string;
  viewed_time: string;
  uploader_id: number;
  uploader_name: string;
  uploader_avatar_url: string;
  category: string;
}

export interface ViewHistoryPage {
  items: ViewHistoryItem[];
  total: number;
  paused: boolean;
}

export async function mbGetMeViewHistory(keyword?: string): Promise<ViewHistoryPage> {
  const r = await http.get("/api/v1/users/me/view-history", {
    ...authAxiosOpts,
    params: keyword ? { keyword } : undefined
  });
  return unwrap<ViewHistoryPage>(r);
}

export async function mbDeleteMeViewHistoryEntry(videoId: number): Promise<void> {
  await http.delete(`/api/v1/users/me/view-history/${videoId}`, authAxiosOpts);
}

export async function mbDeleteMeArticleViewHistoryEntry(
  articleId: number
): Promise<void> {
  await http.delete(
    `/api/v1/users/me/view-history/articles/${articleId}`,
    authAxiosOpts
  );
}

export async function mbClearMeViewHistory(): Promise<void> {
  await http.delete("/api/v1/users/me/view-history", authAxiosOpts);
}

export async function mbPutMeViewHistorySettings(paused: boolean): Promise<{ paused: boolean }> {
  const r = await http.put(
    "/api/v1/users/me/view-history/settings",
    { paused },
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap<{ paused: boolean }>(r);
}

export async function mbPostViewHistory(
  videoId: number,
  body: { progress_sec?: number; duration_sec?: number; device?: string }
): Promise<void> {
  await http.post(`/api/v1/videos/${videoId}/view-history`, body, {
    ...authAxiosOpts,
    headers: { "Content-Type": "application/json" }
  });
}

export async function mbPutMeProfile(body: {
  nickname: string;
  sign: string;
  gender: string;
  birthday: string;
}): Promise<UserMe> {
  const r = await http.put("/api/v1/users/me/profile", body, { ...authAxiosOpts });
  return unwrap<UserMe>(r);
}

export async function mbPutMeAnnouncement(
  announcement: string
): Promise<{ user_id: number; announcement: string }> {
  const r = await http.put(
    "/api/v1/users/me/announcement",
    { announcement: String(announcement ?? "") },
    {
      ...authAxiosOpts,
      headers: { "Content-Type": "application/json" }
    }
  );
  return unwrap(r);
}

export async function mbPutMeUsername(
  username: string
): Promise<{ user_id: number; username: string }> {
  const r = await http.put("/api/v1/users/me", {
    username: String(username || "").trim()
  });
  return unwrap(r);
}

export async function mbPutMePassword(
  oldPassword: string,
  newPassword: string
): Promise<{ ok: boolean }> {
  const r = await http.put(
    "/api/v1/users/me/password",
    {
      old_password: oldPassword,
      new_password: newPassword
    },
    { ...authAxiosOpts }
  );
  return unwrap(r);
}

export async function mbRequestAccountDeletion(password: string): Promise<{
  ok: boolean;
  pending?: boolean;
  deletion_effective_at?: string;
  cooling_days?: number;
}> {
  const r = await http.post(
    "/api/v1/users/me/deletion/request",
    { password: String(password || "") },
    { ...authAxiosOpts }
  );
  return unwrap(r);
}

export async function mbRevokeAccountDeletion(): Promise<{ ok: boolean }> {
  const r = await http.post(
    "/api/v1/users/me/deletion/revoke",
    {},
    { ...authAxiosOpts }
  );
  return unwrap(r);
}

export async function mbPostMeAvatar(file: File): Promise<{ avatar_url: string }> {
  const fd = new FormData();
  fd.append("avatar", file);
  const r = await http.post("/api/v1/users/me/avatar", fd, { ...authAxiosOpts });
  return unwrap(r);
}

export async function mbListMyVideos(params?: {
  page?: number;
  page_size?: number;
  /** time | view | fav | danmu | reply */
  sort?: string;
  /** all | draft | processing | passed | rejected */
  status?: string;
  q?: string;
}): Promise<MyVideoListPayload> {
  const r = await http.get("/api/v1/users/me/videos", {
    params,
    ...authAxiosOpts
  });
  return unwrap(r);
}

export interface CreatorCommentParent {
  id: number;
  user_id: number;
  username: string;
  content: string;
}

export interface CreatorCommentVideo {
  id: number;
  title: string;
  cover_url: string;
}

export interface CreatorCommentArticle {
  id: number;
  title: string;
  cover_url: string;
}

export interface CreatorCommentRow {
  id: number;
  video_id?: number;
  article_id?: number;
  dynamic_id?: number;
  user_id: number;
  username: string;
  avatar_url: string;
  parent_id: number;
  content: string;
  like_count: number;
  reply_count: number;
  liked_by_me?: boolean;
  created_at: string;
  approved?: boolean;
  curated_ignored?: boolean;
  video?: CreatorCommentVideo;
  article?: CreatorCommentArticle;
  dynamic?: CreatorCommentVideo;
  parent?: CreatorCommentParent;
}

export async function mbListCreatorComments(params?: {
  page?: number;
  page_size?: number;
  sort?: "recent" | "earliest" | "likes" | "replies";
  q?: string;
  video_id?: number;
  article_id?: number;
  /** video（默认）| article | dynamic */
  media?: "video" | "article" | "dynamic";
  dynamic_id?: number;
  /** 1 = 待精选（仅开启评论精选的稿件） */
  pending?: 0 | 1;
  pending_status?: "unprocessed" | "ignored" | "all";
  scope?: "all" | "root" | "reply";
}): Promise<{
  items: CreatorCommentRow[];
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}> {
  const r = await http.get("/api/v1/users/me/creator/comments", {
    params,
    ...authAxiosOpts
  });
  return unwrap(r);
}

export interface CreatorDanmakuRow {
  id: number;
  video_id: number;
  user_id: number;
  username: string;
  content: string;
  color: string;
  type: string;
  type_label: string;
  video_time: number;
  play_time: string;
  like_count: number;
  liked_by_me?: boolean;
  created_at: string;
  video: CreatorCommentVideo;
}

export async function mbListCreatorDanmakus(params?: {
  limit?: number;
  q?: string;
  video_id?: number;
  type?: "" | "scroll" | "top" | "bottom";
}): Promise<{ items: CreatorDanmakuRow[]; total: number; limit: number }> {
  const r = await http.get("/api/v1/users/me/creator/danmakus", {
    params,
    ...authAxiosOpts
  });
  return unwrap(r);
}

export async function mbDeleteDanmaku(id: number): Promise<{ id: number }> {
  const r = await http.delete(`/api/v1/danmakus/${id}`, { ...authAxiosOpts });
  return unwrap(r);
}

export async function mbToggleDanmakuLike(
  danmakuId: number
): Promise<{ liked: boolean; like_count: number }> {
  const r = await http.post(`/api/v1/danmakus/${danmakuId}/like`, undefined, {
    ...authAxiosOpts
  });
  return unwrap(r);
}

export async function mbDeleteMyVideo(id: number): Promise<{ ok: boolean }> {
  const r = await http.delete(`/api/v1/videos/${id}`, { ...authAxiosOpts });
  return unwrap(r);
}

/** POST /videos/:id/danmaku 成功后的 data（与 WS 实时包字段一致，便于乐观更新） */
export interface DanmakuCommittedRow {
  id: number;
  content: string;
  color: string;
  type: string;
  /** sm | md | lg */
  font_size?: string;
  video_time: number;
  user: string;
  created_at: string;
}

export async function mbPostDanmaku(
  videoId: number,
  body: {
    content: string;
    color: string;
    type: string;
    font_size?: string;
    video_time: number;
  }
): Promise<DanmakuCommittedRow> {
  const r = await http.post(
    `/api/v1/videos/${videoId}/danmaku`,
    body,
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbListComments(
  videoId: number
): Promise<{
  items: CommentItem[];
  comments_closed?: boolean;
  comments_curated?: boolean;
}> {
  const r = await http.get(`/api/v1/videos/${videoId}/comments`);
  return unwrap(r);
}

export async function mbPatchVideoPlayback(
  videoId: number,
  body: {
    comments_closed?: boolean;
    comments_curated?: boolean;
    danmaku_closed?: boolean;
  }
): Promise<{
  comments_closed: boolean;
  comments_curated: boolean;
  danmaku_closed: boolean;
}> {
  const r = await http.patch(`/api/v1/videos/${videoId}/playback`, body, {
    ...authAxiosOpts,
    headers: { "Content-Type": "application/json" }
  });
  return unwrap(r);
}

export async function mbApproveComment(
  commentId: number
): Promise<{ id: number; approved: boolean }> {
  const r = await http.post(`/api/v1/comments/${commentId}/approve`, null, authAxiosOpts);
  return unwrap(r);
}

export async function mbIgnoreCuratedComment(
  commentId: number
): Promise<{ id: number; curated_ignored: boolean }> {
  const r = await http.post(
    `/api/v1/comments/${commentId}/ignore-curated`,
    null,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbPostComment(
  videoId: number,
  content: string,
  parentId = 0
): Promise<{ id: number; approved?: boolean }> {
  const r = await http.post(
    `/api/v1/videos/${videoId}/comments`,
    {
      content,
      parent_id: parentId
    },
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbToggleVideoLike(
  videoId: number
): Promise<{ liked: boolean }> {
  const r = await http.post(
    `/api/v1/videos/${videoId}/like`,
    {},
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbToggleVideoFavorite(
  videoId: number
): Promise<{ favorited: boolean; fav_count: number }> {
  const r = await http.post(
    `/api/v1/videos/${videoId}/favorite`,
    {},
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export interface VideoFavoritePickerItem {
  id: number;
  title: string;
  is_default: boolean;
  video_count: number;
  count_label: string;
  selected: boolean;
}

export async function mbGetVideoFavoritePicker(videoId: number): Promise<{
  favorited: boolean;
  fav_count: number;
  folder_ids: number[];
  items: VideoFavoritePickerItem[];
}> {
  const r = await http.get(`/api/v1/videos/${videoId}/favorite-picker`, {
    ...authAxiosOpts
  });
  return unwrap(r);
}

export async function mbSetVideoFavoriteFolders(
  videoId: number,
  folderIds: number[]
): Promise<{
  favorited: boolean;
  fav_count: number;
  folder_ids: number[];
}> {
  const r = await http.put(
    `/api/v1/videos/${videoId}/favorite-folders`,
    { folder_ids: folderIds },
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbRemoveVideoFromFavoriteFolder(
  videoId: number,
  folderId: number
): Promise<{ ok: boolean; removed: boolean }> {
  const r = await http.delete(
    `/api/v1/videos/${videoId}/favorite-folders/${folderId}`,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbCopyVideoToFavoriteFolder(
  videoId: number,
  folderId: number
): Promise<{ ok: boolean; copied: boolean }> {
  const r = await http.post(
    `/api/v1/videos/${videoId}/favorite-folders/${folderId}`,
    {},
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbMoveVideoFavoriteFolder(
  videoId: number,
  fromFolderId: number,
  toFolderId: number
): Promise<{ ok: boolean; moved: boolean }> {
  const r = await http.put(
    `/api/v1/videos/${videoId}/favorite-folders/move`,
    { from_folder_id: fromFolderId, to_folder_id: toFolderId },
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbPostVideoCoin(
  videoId: number,
  amount: 1 | 2 = 1
): Promise<{
  coined: boolean;
  coin_count: number;
  amount: number;
  my_coin_amount: number;
  coined_by_me: boolean;
  coin_balance: number;
  daily_coin_exp_progress: number;
  daily_coin_exp_max: number;
}> {
  const r = await http.post(
    `/api/v1/videos/${videoId}/coin`,
    { amount },
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbGetMySearchHistory(): Promise<{ keywords: string[] }> {
  const r = await http.get("/api/v1/users/me/search-history", authAxiosOpts);
  return unwrap(r);
}

export async function mbPutMySearchHistory(
  keywords: string[]
): Promise<{ keywords: string[] }> {
  const r = await http.put(
    "/api/v1/users/me/search-history",
    { keywords },
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbAddMySearchHistory(
  keyword: string
): Promise<{ keywords: string[] }> {
  const r = await http.post(
    "/api/v1/users/me/search-history",
    { keyword },
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbToggleWatchLater(
  videoId: number
): Promise<{ in_watch_later: boolean }> {
  const r = await http.post(
    `/api/v1/videos/${videoId}/watch-later`,
    {},
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export interface WatchLaterListItem {
  id: number;
  title: string;
  cover_url: string;
  play_count: number;
  danmaku_count: number;
  duration: number;
  uploader: string;
  uploader_id?: number;
  uploader_avatar_url?: string;
  watched?: boolean;
  created_at: string;
  added_at: string;
}

export async function mbListMyWatchLater(params?: {
  limit?: number;
}): Promise<{
  items: WatchLaterListItem[];
  total: number;
  max_limit: number;
}> {
  const r = await http.get("/api/v1/users/me/watch-later", {
    params,
    ...authAxiosOpts
  });
  return unwrap(r);
}

export async function mbClearMyWatchLater(): Promise<{ ok: boolean }> {
  const r = await http.delete("/api/v1/users/me/watch-later", authAxiosOpts);
  return unwrap(r);
}

export async function mbClearWatchedWatchLater(): Promise<{ ok: boolean }> {
  const r = await http.delete(
    "/api/v1/users/me/watch-later/watched",
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbMarkWatchLaterWatched(
  videoId: number
): Promise<{ watched: boolean }> {
  const r = await http.post(
    `/api/v1/users/me/watch-later/${videoId}/watched`,
    {},
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbDeleteComment(commentId: number): Promise<void> {
  const r = await http.delete(`/api/v1/comments/${commentId}`);
  unwrap(r);
}

export async function mbPinComment(
  commentId: number
): Promise<{ pinned: boolean }> {
  const r = await http.post(`/api/v1/comments/${commentId}/pin`);
  return unwrap(r);
}

export async function mbToggleLike(commentId: number): Promise<{ liked: boolean }> {
  const r = await http.post(
    `/api/v1/comments/${commentId}/like`,
    {},
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbToggleDislike(
  commentId: number
): Promise<{ disliked: boolean }> {
  const r = await http.post(`/api/v1/comments/${commentId}/dislike`);
  return unwrap(r);
}

export async function mbUnreadSummary(): Promise<Record<string, number>> {
  const r = await http.get("/api/v1/notifications/unread-summary", authAxiosOpts);
  return unwrap(r);
}

export async function mbListNotifications(params: {
  category: string;
  cursor?: string;
}): Promise<{ items: NotificationRow[]; next_cursor: string }> {
  const r = await http.get("/api/v1/notifications", { params, ...authAxiosOpts });
  return unwrap(r);
}

/** 「回复我的」：对通知关联的评论点赞（后端按通知解析 video/article） */
export async function mbToggleNotificationCommentLike(
  notificationId: number
): Promise<{ liked: boolean }> {
  const r = await http.post(
    `/api/v1/notifications/${notificationId}/comment-like`,
    {},
    authAxiosOpts
  );
  return unwrap(r);
}

/** 「回复我的」：对通知关联的评论回复（后端按通知解析 video/article） */
export async function mbPostNotificationCommentReply(
  notificationId: number,
  content: string
): Promise<{ id: number; approved?: boolean }> {
  const r = await http.post(
    `/api/v1/notifications/${notificationId}/comment-reply`,
    { content },
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

/** GET /notifications/:id/like-likers — 某条「收到的赞」聚合通知下，点赞用户列表（新在前） */
export async function mbListNotificationLikeLikers(
  id: number,
  params?: { cursor?: string }
): Promise<{ items: LikeNotifLikerRow[]; next_cursor: string }> {
  const r = await http.get(`/api/v1/notifications/${id}/like-likers`, {
    params
  });
  return unwrap(r);
}

export async function mbMarkNotificationRead(id: number): Promise<void> {
  const r = await http.patch(`/api/v1/notifications/${id}/read`, null, {
    ...authAxiosOpts
  });
  unwrap(r);
}

/** 进入某分类页时将该分类下全部未读标为已读（SPEC F9） */
export async function mbMarkNotificationCategoryRead(
  category: string
): Promise<void> {
  const cat = String(category || "").trim();
  if (!cat) return;
  const r = await http.patch(
    "/api/v1/notifications/read-by-category",
    null,
    { params: { category: cat }, ...authAxiosOpts }
  );
  unwrap(r);
}

/** 消息中心列表内已展示的通知批量标已读（SPEC F9） */
export async function mbMarkNotificationsReadBatch(
  ids: number[]
): Promise<void> {
  const clean = (ids || []).map(n => Number(n) || 0).filter(n => n > 0);
  if (!clean.length) return;
  const r = await http.patch(
    "/api/v1/notifications/read-batch",
    { ids: clean },
    { ...authAxiosOpts }
  );
  unwrap(r);
}

export async function mbDeleteNotification(
  id: number
): Promise<{ ok: boolean }> {
  const r = await http.delete(`/api/v1/notifications/${id}`, {
    ...authAxiosOpts
  });
  return unwrap(r);
}

export async function mbMuteLikeNotification(
  id: number
): Promise<{ likes_muted: boolean }> {
  const r = await http.post(
    `/api/v1/notifications/${id}/mute-likes`,
    {},
    { ...authAxiosOpts }
  );
  return unwrap(r);
}

export interface UploadVideoResult {
  id: number;
  status: string;
  title: string;
  duration: number;
  created_at: string;
}

export async function mbUploadVideo(
  form: FormData,
  opts?: Pick<AxiosRequestConfig, "onUploadProgress" | "signal">
): Promise<UploadVideoResult> {
  const r = await http.post("/api/v1/videos", form, {
    timeout: 600000,
    skipGlobalErrorToast: true,
    ...opts
  });
  return unwrap<UploadVideoResult>(r);
}

export interface SaveVideoDraftResult {
  id: number;
  status: string;
  title: string;
  cover_url?: string;
  duration?: number;
  created_at: string;
}

/** POST /videos/draft — 存草稿（multipart，须含 file） */
export async function mbSaveVideoDraft(
  form: FormData,
  opts?: Pick<AxiosRequestConfig, "onUploadProgress" | "signal">
): Promise<SaveVideoDraftResult> {
  const r = await http.post("/api/v1/videos/draft", form, {
    timeout: 600000,
    skipGlobalErrorToast: true,
    ...opts
  });
  return unwrap<SaveVideoDraftResult>(r);
}

/** PUT /videos/:id/draft — 更新草稿（multipart 或 JSON） */
export async function mbUpdateVideoDraft(
  videoId: number,
  body:
    | FormData
    | { title: string; description: string; tags?: string[]; zone?: string },
  opts?: Pick<AxiosRequestConfig, "onUploadProgress" | "signal">
): Promise<{ id: number; status: string; title: string; cover_url?: string }> {
  if (body instanceof FormData) {
    const r = await http.post(`/api/v1/videos/${videoId}/draft`, body, {
      timeout: 600000,
      skipGlobalErrorToast: true,
      ...opts
    });
    return unwrap(r);
  }
  const r = await http.put(`/api/v1/videos/${videoId}/draft`, body, {
    ...authAxiosOpts,
    headers: { "Content-Type": "application/json" }
  });
  return unwrap(r);
}

/** POST /videos/:id/publish — 草稿投稿（进入转码） */
export async function mbPublishVideoDraft(
  videoId: number
): Promise<{ id: number; status: string }> {
  const r = await http.post(
    `/api/v1/videos/${videoId}/publish`,
    {},
    authAxiosOpts
  );
  return unwrap(r);
}

/** POST /videos/:id/replace-media — 审核/转码未通过时更换视频并重新转码 */
export async function mbReplaceVideoMedia(
  videoId: number,
  form: FormData,
  opts?: Pick<AxiosRequestConfig, "onUploadProgress" | "signal">
): Promise<{ id: number; status: string }> {
  const r = await http.post(`/api/v1/videos/${videoId}/replace-media`, form, {
    timeout: 600000,
    skipGlobalErrorToast: true,
    ...opts
  });
  return unwrap(r);
}

/** 拉取草稿源文件用于编辑页预览（需登录） */
export async function mbFetchDraftVideoObjectUrl(
  videoId: number
): Promise<string> {
  const remote = import.meta.env.VITE_REMOTE_API_BASE;
  const base =
    remote && String(remote).trim()
      ? String(remote).replace(/\/$/, "")
      : "";
  const path = `/api/v1/users/me/videos/${videoId}/draft-source`;
  const url = base ? `${base}${path}` : path;
  const token = getAccessToken();
  const headers: Record<string, string> = {};
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  const res = await fetch(url, { headers });
  if (!res.ok) {
    throw new Error("无法加载草稿视频");
  }
  const blob = await res.blob();
  return URL.createObjectURL(blob);
}

export interface DmConversationItem {
  id: number;
  peer_id: number;
  peer_name: string;
  peer_avatar: string;
  last_preview: string;
  last_message_at: string;
  unread_count: number;
  pinned?: boolean;
  muted?: boolean;
}

export interface DmMessageItem {
  id: number;
  conversation_id: number;
  sender_id: number;
  sender_name: string;
  sender_avatar: string;
  content: string;
  created_at: string;
}

export async function mbListDmConversations(): Promise<{
  items: DmConversationItem[];
}> {
  const r = await http.get("/api/v1/dm/conversations", authAxiosOpts);
  return unwrap(r);
}

export async function mbCreateDmConversation(
  peerId: number
): Promise<DmConversationItem> {
  const r = await http.post(
    "/api/v1/dm/conversations",
    { peer_id: peerId },
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbListDmMessages(
  conversationId: number,
  params?: { cursor?: string; limit?: number }
): Promise<{
  items: DmMessageItem[];
  next_cursor: string;
  peer_id: number;
  peer_name: string;
  peer_avatar: string;
}> {
  const r = await http.get(
    `/api/v1/dm/conversations/${conversationId}/messages`,
    { params, ...authAxiosOpts }
  );
  return unwrap(r);
}

export async function mbPostDmMessage(
  conversationId: number,
  content: string
): Promise<DmMessageItem> {
  const r = await http.post(
    `/api/v1/dm/conversations/${conversationId}/messages`,
    { content },
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbPatchDmConversationSettings(
  conversationId: number,
  body: { pinned?: boolean; muted?: boolean }
): Promise<DmConversationItem> {
  const r = await http.patch(
    `/api/v1/dm/conversations/${conversationId}/settings`,
    body,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbDeleteDmConversation(
  conversationId: number
): Promise<{ deleted: boolean; conversation_id: number }> {
  const r = await http.delete(
    `/api/v1/dm/conversations/${conversationId}`,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbResetDmAgentConversation(conversationId: number): Promise<{
  conversation: DmConversationItem;
  welcome_message: DmMessageItem;
}> {
  const r = await http.post(
    `/api/v1/dm/conversations/${conversationId}/reset`,
    {},
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbBlockUser(userId: number): Promise<{
  blocked: boolean;
  user_id: number;
}> {
  const r = await http.post(
    `/api/v1/users/${userId}/block`,
    {},
    authAxiosOpts
  );
  return unwrap(r);
}

export function mbWsChatUrl(accessToken?: string | null): string {
  const t = accessToken != null ? String(accessToken).trim() : "";
  if (!t) {
    return "";
  }
  const path = `/api/v1/ws/chat?token=${encodeURIComponent(t)}`;
  const remote = import.meta.env.VITE_REMOTE_API_BASE;
  if (remote && String(remote).trim()) {
    const u = String(remote).replace(/\/$/, "");
    const ws = u.replace(/^http/, "ws");
    return `${ws}${path}`;
  }
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}${path}`;
}

export function mbWsDanmakuUrl(
  videoId: number,
  accessToken?: string | null
): string {
  const t = accessToken != null ? String(accessToken).trim() : "";
  const path =
    t.length > 0
      ? `/api/v1/ws/danmaku?token=${encodeURIComponent(t)}&video_id=${videoId}`
      : `/api/v1/ws/danmaku?video_id=${videoId}`;
  const remote = import.meta.env.VITE_REMOTE_API_BASE;
  if (remote && String(remote).trim()) {
    const u = String(remote).replace(/\/$/, "");
    const ws = u.replace(/^http/, "ws");
    return `${ws}${path}`;
  }
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}${path}`;
}

export interface FavoriteListItem {
  id: number;
  title: string;
  cover_url: string;
  play_count: number;
  danmaku_count: number;
  duration: number;
  uploader: string;
  uploader_id?: number;
  uploader_avatar_url?: string;
  created_at: string;
  favorited_at: string;
  folder_id?: number;
}

export interface FavoriteFolderItem {
  id: number;
  title: string;
  description: string;
  is_public: boolean;
  is_default: boolean;
  video_count: number;
  cover_url?: string | null;
}

export async function mbListMyFavoriteFolders(): Promise<{
  items: FavoriteFolderItem[];
}> {
  const r = await http.get("/api/v1/users/me/favorite-folders", authAxiosOpts);
  return unwrap(r);
}

export async function mbCreateFavoriteFolder(input: {
  title: string;
  description?: string;
  is_public?: boolean;
  cover?: File | null;
}): Promise<FavoriteFolderItem> {
  const fd = new FormData();
  fd.append("title", input.title);
  fd.append("description", input.description || "");
  fd.append("is_public", input.is_public === false ? "false" : "true");
  if (input.cover) {
    fd.append("cover", input.cover);
  }
  const r = await http.post("/api/v1/users/me/favorite-folders", fd, {
    ...authAxiosOpts
  });
  return unwrap(r);
}

export async function mbUpdateFavoriteFolder(
  folderId: number,
  input: {
    title: string;
    description?: string;
    is_public?: boolean;
    cover?: File | null;
  }
): Promise<FavoriteFolderItem> {
  const fd = new FormData();
  fd.append("title", input.title);
  fd.append("description", input.description || "");
  fd.append("is_public", input.is_public === false ? "false" : "true");
  if (input.cover) {
    fd.append("cover", input.cover);
  }
  const r = await http.put(
    `/api/v1/users/me/favorite-folders/${folderId}`,
    fd,
    { ...authAxiosOpts }
  );
  return unwrap(r);
}

export async function mbDeleteFavoriteFolder(
  folderId: number
): Promise<{ deleted: boolean }> {
  const r = await http.delete(
    `/api/v1/users/me/favorite-folders/${folderId}`,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbClearInvalidFavoritesInFolder(
  folderId: number
): Promise<{ cleared: number }> {
  const r = await http.delete(
    `/api/v1/users/me/favorite-folders/${folderId}/invalid-favorites`,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbBatchRemoveVideosFromFavoriteFolder(
  folderId: number,
  videoIds: number[]
): Promise<{ removed: number }> {
  const r = await http.post(
    `/api/v1/users/me/favorite-folders/${folderId}/batch-remove`,
    { video_ids: videoIds },
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbListUserFavoriteFolders(
  userId: number
): Promise<{
  items: FavoriteFolderItem[];
  total?: number;
  hidden_count?: number;
}> {
  const r = await http.get(`/api/v1/space/${userId}/favorite-folders`);
  return unwrap(r);
}

export async function mbListUserRecentCoinVideos(
  userId: number,
  params?: { limit?: number }
): Promise<{ items: VideoListItem[]; total: number }> {
  const r = await http.get(`/api/v1/space/${userId}/recent-coins`, { params });
  return unwrap(r);
}

export async function mbListMyFavorites(params?: {
  limit?: number;
  folder_id?: number;
}): Promise<{ items: FavoriteListItem[]; total: number }> {
  const r = await http.get("/api/v1/users/me/favorites", {
    params,
    ...authAxiosOpts
  });
  return unwrap(r);
}

export async function mbListUserFavorites(
  userId: number,
  params?: { limit?: number; folder_id?: number }
): Promise<{ items: FavoriteListItem[]; total: number }> {
  const r = await http.get(`/api/v1/space/${userId}/favorites`, { params });
  return unwrap(r);
}

export interface ArticleTocEntry {
  id: string;
  level: number;
  text: string;
}

export interface ArticleDetail {
  id: number;
  cv_id: number;
  user_id: number;
  title: string;
  cover_url: string;
  body_md: string;
  body_html: string;
  toc: ArticleTocEntry[];
  tags: string[];
  status: string;
  fail_reason?: string;
  view_count: number;
  comment_count: number;
  coin_count: number;
  fav_count: number;
  forward_count: number;
  published_at: string;
  created_at: string;
  author_name: string;
  author_avatar: string;
  favorited_by_me: boolean;
  coined_by_me: boolean;
  my_coin_amount: number;
  is_author: boolean;
  comments_closed?: boolean;
  comments_curated?: boolean;
}

export interface ArticleListItem {
  id: number;
  title: string;
  cover_url: string;
  status: string;
  fail_reason?: string;
  view_count: number;
  comment_count: number;
  coin_count: number;
  fav_count: number;
  forward_count: number;
  published_at: string;
  created_at: string;
  author_name?: string;
  favorited_at?: string;
  favorited_by_me?: boolean;
  comments_closed?: boolean;
  /** 作者开启评论精选 */
  comments_curated?: boolean;
}

export async function mbGetArticle(id: number): Promise<ArticleDetail> {
  const r = await http.get(`/api/v1/articles/${id}`);
  return unwrap<ArticleDetail>(r);
}

export async function mbGetMyArticle(id: number): Promise<ArticleDetail> {
  const r = await http.get(`/api/v1/users/me/articles/${id}`, authAxiosOpts);
  return unwrap<ArticleDetail>(r);
}

export async function mbPostArticle(body: {
  title: string;
  body_md: string;
  cover_url?: string;
  tags?: string[];
  publish?: boolean;
}): Promise<{ id: number; status: string }> {
  const r = await http.post(`/api/v1/articles`, body, {
    ...authAxiosOpts,
    headers: { "Content-Type": "application/json" }
  });
  return unwrap(r);
}

export async function mbPutMyArticle(
  id: number,
  body: {
    title?: string;
    body_md?: string;
    cover_url?: string;
    tags?: string[];
    publish?: boolean;
  }
): Promise<{ id: number; status: string }> {
  const r = await http.put(`/api/v1/users/me/articles/${id}`, body, {
    ...authAxiosOpts,
    headers: { "Content-Type": "application/json" }
  });
  return unwrap(r);
}

export async function mbPatchArticlePlayback(
  articleId: number,
  body: { comments_closed?: boolean; comments_curated?: boolean }
): Promise<{ comments_closed: boolean; comments_curated: boolean }> {
  const r = await http.patch(
    `/api/v1/users/me/articles/${articleId}/playback`,
    body,
    {
      ...authAxiosOpts,
      headers: { "Content-Type": "application/json" }
    }
  );
  return unwrap(r);
}

export async function mbApproveArticleComment(
  commentId: number
): Promise<{ id: number; approved: boolean }> {
  const r = await http.post(
    `/api/v1/article-comments/${commentId}/approve`,
    null,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbIgnoreCuratedArticleComment(
  commentId: number
): Promise<{ id: number; curated_ignored: boolean }> {
  const r = await http.post(
    `/api/v1/article-comments/${commentId}/ignore-curated`,
    null,
    authAxiosOpts
  );
  return unwrap(r);
}

/** PUT cover image to OSS (same as video cover upload). */
export async function mbUpdateArticleCover(
  articleId: number,
  file: File
): Promise<{ cover_url: string }> {
  const fd = new FormData();
  fd.append("cover", file);
  const r = await http.put(`/api/v1/users/me/articles/${articleId}/cover`, fd, {
    ...authAxiosOpts,
    timeout: 120000
  });
  return unwrap(r);
}

export async function mbDeleteMyArticle(id: number): Promise<{ ok: boolean }> {
  const r = await http.delete(`/api/v1/users/me/articles/${id}`, authAxiosOpts);
  return unwrap(r);
}

export interface MyArticleListCounts {
  draft: number;
  passed: number;
  processing: number;
  rejected: number;
  dynamics?: number;
}

export interface MyArticleListPayload {
  items: ArticleListItem[];
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
  counts: MyArticleListCounts;
}

export async function mbListMyArticles(params?: {
  page?: number;
  page_size?: number;
  /** time | view | reply | like | fav */
  sort?: string;
  /** all | draft | passed | processing | rejected */
  status?: string;
  q?: string;
}): Promise<MyArticleListPayload> {
  const r = await http.get(`/api/v1/users/me/articles`, {
    params,
    ...authAxiosOpts
  });
  return unwrap(r);
}

export interface MyDynamicListPayload {
  items: UserDynamicItem[];
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

export async function mbListMyDynamics(params?: {
  page?: number;
  page_size?: number;
  /** time | reply | like */
  sort?: string;
  q?: string;
}): Promise<MyDynamicListPayload> {
  const r = await http.get(`/api/v1/users/me/dynamics`, {
    params,
    ...authAxiosOpts
  });
  return unwrap(r);
}

export async function mbListUserPublishedArticles(
  userId: number,
  params?: { limit?: number; cursor?: string }
): Promise<{ items: ArticleListItem[]; next_cursor: string }> {
  const r = await http.get(`/api/v1/space/${userId}/articles`, { params });
  return unwrap(r);
}

export async function mbPostArticleView(
  id: number
): Promise<{ view_count: number }> {
  const r = await http.post(`/api/v1/articles/${id}/view`, null, authAxiosOpts);
  return unwrap(r);
}

export async function mbToggleArticleFavorite(
  id: number
): Promise<{ favorited: boolean; fav_count: number }> {
  const r = await http.post(`/api/v1/articles/${id}/favorite`, null, authAxiosOpts);
  return unwrap(r);
}

export async function mbPostArticleCoin(
  id: number,
  amount: 1 | 2 = 1
): Promise<{
  coined: boolean;
  coin_count: number;
  amount: number;
  my_coin_amount: number;
  coined_by_me: boolean;
  coin_balance: number;
  daily_coin_exp_progress: number;
  daily_coin_exp_max: number;
}> {
  const r = await http.post(
    `/api/v1/articles/${id}/coin`,
    { amount },
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbListMyArticleFavorites(params?: {
  limit?: number;
  cursor?: string;
}): Promise<{ items: ArticleListItem[]; next_cursor: string; total: number }> {
  const r = await http.get(`/api/v1/users/me/article-favorites`, {
    params,
    ...authAxiosOpts
  });
  return unwrap(r);
}

export async function mbListUserArticleFavorites(
  userId: number,
  params?: { limit?: number; cursor?: string }
): Promise<{ items: ArticleListItem[]; next_cursor: string; total: number }> {
  const r = await http.get(`/api/v1/space/${userId}/article-favorites`, { params });
  return unwrap(r);
}

export async function mbListArticleComments(
  articleId: number
): Promise<{
  items: CommentItem[];
  comments_closed?: boolean;
  comments_curated?: boolean;
}> {
  const r = await http.get(`/api/v1/articles/${articleId}/comments`);
  return unwrap(r);
}

export async function mbPostArticleComment(
  articleId: number,
  content: string,
  parentId = 0
): Promise<{ id: number; approved?: boolean }> {
  const r = await http.post(
    `/api/v1/articles/${articleId}/comments`,
    { content, parent_id: parentId },
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbToggleArticleCommentLike(
  commentId: number
): Promise<{ liked: boolean }> {
  const r = await http.post(
    `/api/v1/article-comments/${commentId}/like`,
    null,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbToggleArticleCommentDislike(
  commentId: number
): Promise<{ disliked: boolean }> {
  const r = await http.post(
    `/api/v1/article-comments/${commentId}/dislike`,
    null,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbDeleteArticleComment(
  commentId: number
): Promise<void> {
  const r = await http.delete(
    `/api/v1/article-comments/${commentId}`,
    authAxiosOpts
  );
  unwrap(r);
}

export async function mbPinArticleComment(
  commentId: number
): Promise<{ pinned: boolean }> {
  const r = await http.post(
    `/api/v1/article-comments/${commentId}/pin`,
    null,
    authAxiosOpts
  );
  return unwrap(r);
}

export interface UserDynamicItem {
  id: number;
  title: string;
  content: string;
  images: string[];
  like_count?: number;
  comment_count?: number;
  liked_by_me?: boolean;
  comments_closed?: boolean;
  comments_curated?: boolean;
  created_at: string;
}

export async function mbListUserPublishedDynamics(
  userId: number,
  params?: { limit?: number; cursor?: string }
): Promise<{ items: UserDynamicItem[]; next_cursor: string }> {
  const r = await http.get(`/api/v1/space/${userId}/dynamics`, { params });
  return unwrap(r);
}

export interface UserDynamicDetail extends UserDynamicItem {
  user_id: number;
  author_name: string;
  author_avatar: string;
  is_author?: boolean;
}

export async function mbGetUserDynamic(id: number): Promise<UserDynamicDetail> {
  const r = await http.get(`/api/v1/user-dynamics/${id}`);
  return unwrap<UserDynamicDetail>(r);
}

export async function mbListDynamicComments(
  dynamicId: number
): Promise<{
  items: CommentItem[];
  comments_closed?: boolean;
  comments_curated?: boolean;
}> {
  const r = await http.get(`/api/v1/user-dynamics/${dynamicId}/comments`);
  return unwrap(r);
}

export async function mbPatchDynamicPlayback(
  dynamicId: number,
  body: { comments_closed?: boolean; comments_curated?: boolean }
): Promise<{ comments_closed: boolean; comments_curated: boolean }> {
  const r = await http.patch(
    `/api/v1/users/me/dynamics/${dynamicId}/playback`,
    body,
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbApproveDynamicComment(
  commentId: number
): Promise<{ id: number; approved: boolean }> {
  const r = await http.post(
    `/api/v1/dynamic-comments/${commentId}/approve`,
    null,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbIgnoreCuratedDynamicComment(
  commentId: number
): Promise<{ id: number; curated_ignored: boolean }> {
  const r = await http.post(
    `/api/v1/dynamic-comments/${commentId}/ignore-curated`,
    null,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbPostDynamicComment(
  dynamicId: number,
  content: string,
  parentId = 0
): Promise<{ id: number; approved?: boolean }> {
  const r = await http.post(
    `/api/v1/user-dynamics/${dynamicId}/comments`,
    { content, parent_id: parentId },
    { ...authAxiosOpts, headers: { "Content-Type": "application/json" } }
  );
  return unwrap(r);
}

export async function mbToggleDynamicLike(
  dynamicId: number
): Promise<{ liked: boolean; like_count_delta?: number }> {
  const r = await http.post(
    `/api/v1/user-dynamics/${dynamicId}/like`,
    null,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbDeleteMyDynamic(id: number): Promise<{ ok: boolean }> {
  const r = await http.delete(`/api/v1/users/me/dynamics/${id}`, authAxiosOpts);
  return unwrap(r);
}

export async function mbDeleteDynamicComment(commentId: number): Promise<void> {
  const r = await http.delete(
    `/api/v1/dynamic-comments/${commentId}`,
    authAxiosOpts
  );
  unwrap(r);
}

export async function mbToggleDynamicCommentLike(
  commentId: number
): Promise<{ liked: boolean }> {
  const r = await http.post(
    `/api/v1/dynamic-comments/${commentId}/like`,
    null,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbToggleDynamicCommentDislike(
  commentId: number
): Promise<{ disliked: boolean }> {
  const r = await http.post(
    `/api/v1/dynamic-comments/${commentId}/dislike`,
    null,
    authAxiosOpts
  );
  return unwrap(r);
}

export async function mbPostUserDynamic(input: {
  title?: string;
  content?: string;
  images?: File[];
}): Promise<UserDynamicItem> {
  const fd = new FormData();
  fd.append("title", String(input.title || "").trim());
  fd.append("content", String(input.content || "").trim());
  for (const file of input.images || []) {
    fd.append("images", file);
  }
  const r = await http.post("/api/v1/users/me/dynamics", fd, {
    ...authAxiosOpts,
    timeout: 120000
  });
  return unwrap<UserDynamicItem>(r);
}

export async function mbPutMyUserDynamic(
  id: number,
  input: {
    title?: string;
    content?: string;
    keepImages?: string[];
    images?: File[];
  }
): Promise<UserDynamicItem> {
  const fd = new FormData();
  fd.append("title", String(input.title || "").trim());
  fd.append("content", String(input.content || "").trim());
  const keep = Array.isArray(input.keepImages) ? input.keepImages : [];
  fd.append("keep_images", JSON.stringify(keep));
  for (const file of input.images || []) {
    fd.append("images", file);
  }
  const r = await http.put(`/api/v1/users/me/dynamics/${id}`, fd, {
    ...authAxiosOpts,
    timeout: 120000
  });
  return unwrap<UserDynamicItem>(r);
}
