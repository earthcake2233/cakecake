import { count2 } from "@/utils/utils";
import { minibiliVideoPlayRoute } from "@/utils/minibiliRoutes";

/** 播放页推荐池：过滤当前稿件并去重 */
export function filterRecommendPool(items, excludeVideoId) {
  const ex = Number(excludeVideoId);
  const seen = new Set();
  const out = [];
  for (const it of items || []) {
    const id = Number(it && it.id);
    if (!Number.isFinite(id) || id <= 0) continue;
    if (Number.isFinite(ex) && ex > 0 && id === ex) continue;
    if (seen.has(id)) continue;
    seen.add(id);
    out.push(it);
  }
  return out;
}

/** 推荐视频 / 还喜欢 两路稿件交替合并为一条列表 */
export function interleaveRecommendPools(poolA, poolB) {
  const a = poolA || [];
  const b = poolB || [];
  const max = Math.max(a.length, b.length);
  const out = [];
  const seen = new Set();
  for (let i = 0; i < max; i++) {
    for (const it of [a[i], b[i]]) {
      if (!it) continue;
      const id = Number(it.id);
      if (!Number.isFinite(id) || id <= 0 || seen.has(id)) continue;
      seen.add(id);
      out.push(it);
    }
  }
  return out;
}

export function formatRecommendDuration(sec) {
  const n = Math.max(0, Math.floor(Number(sec) || 0));
  const h = Math.floor(n / 3600);
  const m = Math.floor((n % 3600) / 60);
  const s = n % 60;
  const pad = x => String(x).padStart(2, "0");
  if (h > 0) return `${h}:${pad(m)}:${pad(s)}`;
  return `${m}:${pad(s)}`;
}

function playCountLabel(n) {
  const v = Number(n);
  if (!Number.isFinite(v) || v < 0) return "0";
  return count2(v);
}

function alsoBadgeFromItem(item) {
  const cat = String(item.category || "").trim();
  if (cat) {
    const part = cat.split(">")[0].trim();
    if (part) return part.length > 8 ? `${part.slice(0, 8)}…` : part;
  }
  const zp = String(item.zone_parent || "").trim();
  if (zp) return zp.length > 8 ? `${zp.slice(0, 8)}…` : zp;
  const z = String(item.zone || "").trim();
  if (z) {
    const head = z.split("-")[0];
    return head.length > 8 ? `${head.slice(0, 8)}…` : head;
  }
  return formatRecommendDuration(item.duration);
}

/** 侧栏 related-list 行（不改 class 结构） */
export function mapToRelatedVideoRow(item, fallbackCover) {
  const cover = String(item.cover_url || "").trim() || fallbackCover;
  return {
    id: item.id,
    title: String(item.title || "").trim() || "未命名稿件",
    duration: formatRecommendDuration(item.duration),
    playShort: playCountLabel(item.play_count),
    dm: playCountLabel(item.danmaku_count),
    inWatchLater: !!item.in_watch_later,
    cover,
    playRoute: minibiliVideoPlayRoute(item.id)
  };
}

/** 下方 vd-also 行 */
export function mapToAlsoLikedRow(item, fallbackCover) {
  const cover = String(item.cover_url || "").trim() || fallbackCover;
  return {
    id: item.id,
    title: String(item.title || "").trim() || "未命名稿件",
    duration: formatRecommendDuration(item.duration),
    inWatchLater: !!item.in_watch_later,
    badge: alsoBadgeFromItem(item),
    cover,
    playRoute: minibiliVideoPlayRoute(item.id)
  };
}

const HOME_RECOMMEND_PAGE_SIZE = 8;

function recommendItemId(item) {
  return Number(item && (item.aid != null ? item.aid : item.id));
}

/** 从池中按偏移循环取满 pageSize 条（总数不足时重复稿件，保证条数） */
export function fillHomeRecommendSlots(pool, offset = 0, pageSize = HOME_RECOMMEND_PAGE_SIZE) {
  const list = (Array.isArray(pool) ? pool : []).filter(v => {
    const id = recommendItemId(v);
    return Number.isFinite(id) && id > 0;
  });
  if (!list.length) {
    return [];
  }
  const len = list.length;
  let off = ((Number(offset) || 0) % len + len) % len;
  const out = [];
  for (let i = 0; i < pageSize; i++) {
    out.push(list[(off + i) % len]);
  }
  return out;
}

/** 首页推荐区：在池中取下一批 8 条（优先未展示的稿件，不足则新旧组合，始终满 8 条） */
export function nextHomeRecommendBatch(pool, currentItems, batchOffset, direction = 1) {
  const PAGE = HOME_RECOMMEND_PAGE_SIZE;
  const list = Array.isArray(pool) ? pool : [];
  const cur = Array.isArray(currentItems) ? currentItems : [];
  if (!list.length) {
    return { items: [], nextOffset: 0 };
  }

  const currentIds = new Set(
    cur.map(recommendItemId).filter(id => Number.isFinite(id) && id > 0)
  );
  const fresh = list.filter(v => {
    const id = recommendItemId(v);
    return Number.isFinite(id) && id > 0 && !currentIds.has(id);
  });

  let source;
  if (fresh.length >= PAGE) {
    source = fresh;
  } else {
    const seen = new Set(
      fresh.map(recommendItemId).filter(id => Number.isFinite(id) && id > 0)
    );
    source = [...fresh];
    for (const v of list) {
      const id = recommendItemId(v);
      if (!Number.isFinite(id) || id <= 0 || seen.has(id)) continue;
      seen.add(id);
      source.push(v);
    }
  }

  const len = source.length;
  const step = PAGE;
  const dir = direction < 0 ? -1 : 1;
  let offset = ((Number(batchOffset) || 0) + dir * step) % len;
  if (offset < 0) offset += len;

  return {
    items: fillHomeRecommendSlots(source, offset, PAGE),
    nextOffset: offset
  };
}

export { HOME_RECOMMEND_PAGE_SIZE };

export function zoneParentFromDetail(detail) {
  if (!detail || typeof detail !== "object") return "";
  const zp = String(detail.zone_parent || "").trim();
  if (zp) return zp;
  const z = String(detail.zone || "").trim();
  if (!z) return "";
  const i = z.indexOf("-");
  return i > 0 ? z.slice(0, i) : z;
}
