/** B 站排行榜 rid → Mini-Bili zone_parent */
export const RANKING_RID_ZONE = {
  "0": "",
  "1": "动画",
  "168": "国创",
  "3": "音乐",
  "129": "舞蹈",
  "4": "游戏",
  "36": "科技",
  "160": "生活",
  "119": "鬼畜",
  "155": "时尚",
  "5": "娱乐",
  "181": "影视"
};

export function normalizeRankDays(day) {
  const n = parseInt(String(day), 10);
  if (n === 1 || n === 3 || n === 7 || n === 30) {
    return n;
  }
  return 3;
}

/** 全部投稿：按榜单周期对老稿降权；近期投稿：服务端已按 created_at 过滤 */
export function rankSortScore(v, { arcType, days }) {
  const base = rankCompositeScore(v);
  const dayN = normalizeRankDays(days);
  if (Number(arcType) === 1) {
    return base;
  }
  const raw = String(v.created_at || "").trim();
  const created = raw ? Date.parse(raw.replace(/-/g, "/")) : NaN;
  if (!Number.isFinite(created)) {
    return base;
  }
  const ageDays = (Date.now() - created) / 86400000;
  if (ageDays > dayN * 2) {
    return base * 0.35;
  }
  const boost = Math.max(0.45, 1.55 - ageDays / dayN);
  return base * boost;
}

export function sortRankPool(items, { arcType, days }) {
  const pool = Array.isArray(items) ? [...items] : [];
  return pool
    .map(v => ({
      v,
      score: rankSortScore(v, { arcType, days })
    }))
    .sort((a, b) => b.score - a.score)
    .map(({ v, score }) => ({ v, score }));
}

/** 排行榜页「综合得分」：由播放/弹幕/互动加权（仅展示用） */
export function rankCompositeScore(v) {
  const play = Number(v.play_count) || 0;
  const dm = Number(v.danmaku_count) || 0;
  const like = Number(v.like_count) || 0;
  const fav = Number(v.fav_count) || 0;
  const coin = Number(v.coin_count) || 0;
  const comment = Number(v.comment_count) || 0;
  return Math.round(
    play * 1.2 + dm * 85 + like * 120 + fav * 90 + coin * 200 + comment * 60
  );
}

/** 全站榜列表行 → allList.vue 所需字段 */
export function mapVideoToRankListItem(v, displayScore) {
  const pts =
    displayScore != null && Number.isFinite(Number(displayScore))
      ? Math.round(Number(displayScore))
      : rankCompositeScore(v);
  return {
    aid: v.id,
    title: String(v.title || "").trim() || "未命名稿件",
    pic: String(v.cover_url || "").trim(),
    play: Number(v.play_count) || 0,
    video_review: Number(v.danmaku_count) || 0,
    author: String(v.uploader || "").trim() || "未知UP主",
    userId: Number(v.user_id) > 0 ? Number(v.user_id) : null,
    mid: "javascript:;",
    pts
  };
}

export const RANK_PAGE_NOTE =
  "根据稿件内容质量、近期的数据综合展示，动态更新";
