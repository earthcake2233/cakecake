import http from "../utils/http";
import * as local from "../mock/localApi";
import {
  mapVideoToRankListItem,
  normalizeRankDays,
  RANK_PAGE_NOTE,
  RANKING_RID_ZONE,
  sortRankPool
} from "../utils/rankingFeeds";

/** 设为 true 时使用远程接口（需在 http.js 配置 baseURL）；默认使用本地 mock + assets 图片 */
const useLocalMock = import.meta.env.VITE_USE_REMOTE_API !== "true";

/** Mini-Bili 后端：首页推荐等走真实 REST（需后端 CORS 与 VITE_REMOTE_API_BASE） */
const isMinibili =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

function pick(fnRemote, fnLocal) {
  if (isMinibili) {
    return Promise.resolve(fnLocal());
  }
  return useLocalMock ? Promise.resolve(fnLocal()) : fnRemote();
}

/** B 站首页分区 rid → Mini-Bili zone_parent（与 videoZones.js 一致） */
const HOME_ZONE_RID_PARENT = {
  1: "动画",
  3: "音乐",
  129: "舞蹈"
};

function minibiliItemToArchive(v) {
  return {
    aid: v.id,
    title: v.title,
    pic: v.cover_url,
    duration: v.duration,
    in_watch_later: !!v.in_watch_later,
    stat: {
      view: v.play_count,
      danmaku: v.danmaku_count
    }
  };
}

function minibiliItemToRank(v) {
  return {
    aid: v.id,
    title: v.title,
    pic: v.cover_url,
    pts: v.play_count
  };
}

function fetchMinibiliZoneVideos({
  zoneParent,
  limit,
  sort,
  days,
  arc_type
}) {
  const params = {
    limit: limit || 10,
    sort: sort || "hot"
  };
  if (zoneParent) {
    params.zone_parent = zoneParent;
  }
  if (days != null && days > 0) {
    params.days = days;
  }
  if (arc_type === 0 || arc_type === 1) {
    params.arc_type = arc_type;
  }
  return http
    .get("/api/v1/videos", {
      params
    })
    .then(body => {
      if (!body || body.code !== 0) {
        throw new Error((body && body.msg) || "加载失败");
      }
      const data = body.data || {};
      return {
        items: data.items || [],
        zoneVideoCount:
          typeof data.zone_video_count === "number"
            ? data.zone_video_count
            : typeof data.new_dynamic_count === "number"
              ? data.new_dynamic_count
              : 0
      };
    });
}

//登录获取个人信息
export function getUserInfo() {
  return pick(() => Promise.resolve(local.loginUserResponse), () => local.loginUserResponse);
}
//获取大会员推荐信息
export function getVipInfo() {
  return pick(() => Promise.resolve(local.topInfoResponse), () => local.topInfoResponse);
}
//头部背景图
export function getAdSlide(data) {
  return pick(
    () =>
      http.get("/ad_slide", {
        params: {
          position_id: data.position_id
        }
      }),
    () => local.adSlideResponse
  );
}
//头部背景图
export function getLoc(data) {
  return pick(() => http.get("/loc", { params: data }), () => local.locResponse);
}
//轮播图+广告位
export function getLocs(data) {
  return pick(() => http.get("/locs", { params: data }), () => local.locsResponse);
}
function buildSearchDefaultFromHotItems(items) {
  const titles = (items || [])
    .map(it => String((it && it.title) || "").trim())
    .filter(Boolean);
  const hotList = titles.length ? titles : local.HOT_SEARCH_FALLBACK_TITLES;
  const first = hotList[0] || "搜索";
  return {
    code: 0,
    data: {
      show_name: first,
      word: first,
      hot_list: hotList
    }
  };
}

//搜索框默认关键词（Mini-Bili：热搜榜；其余：原站 mock）
export function getSearchDefaultWords() {
  if (isMinibili) {
    return http
      .get("/api/v1/hot-search", { params: { limit: 10 } })
      .then(body => {
        if (!body || body.code !== 0) {
          throw new Error((body && body.msg) || "加载热搜失败");
        }
        const items = (body.data && body.data.items) || [];
        return buildSearchDefaultFromHotItems(items);
      })
      .catch(err => {
        console.warn("Mini-Bili 热搜占位失败，回退本地热搜", err);
        return buildSearchDefaultFromHotItems(
          local.HOT_SEARCH_FALLBACK_TITLES.map((title, i) => ({
            rank: i + 1,
            title
          }))
        );
      });
  }
  return pick(
    () => http.get("/search_value"),
    () => buildSearchDefaultFromHotItems(local.HOT_SEARCH_FALLBACK_TITLES.map((title, i) => ({ rank: i + 1, title })))
  );
}
function suggestTagsFromHotSearchItems(items) {
  return (items || [])
    .map(it => {
      const value = String((it && it.title) || "").trim();
      if (!value) {
        return null;
      }
      return { name: value, value };
    })
    .filter(Boolean);
}

function fetchSuggestHotFallback(limit = 10) {
  return http
    .get("/api/v1/hot-search", { params: { limit } })
    .then(body => {
      if (!body || body.code !== 0) {
        return [];
      }
      return suggestTagsFromHotSearchItems(
        (body.data && body.data.items) || []
      );
    })
    .catch(() => []);
}

//搜索框搜索建议（Mini-Bili：GET /api/v1/search/suggest）
export function getSuggest(term) {
  if (isMinibili) {
    const q = String(term || "").trim();
    if (!q) {
      return Promise.resolve({ result: { tag: [] } });
    }
    return http
      .get("/api/v1/search/suggest", { params: { term: q, limit: 10 } })
      .then(body => {
        if (!body || body.code !== 0) {
          return fetchSuggestHotFallback(10).then(tag => ({ result: { tag } }));
        }
        const tags = (body.data && body.data.tag) || [];
        if (tags.length) {
          return { result: { tag: tags } };
        }
        return fetchSuggestHotFallback(10).then(tag => ({ result: { tag } }));
      })
      .catch(err => {
        console.warn("搜索建议加载失败", err);
        return fetchSuggestHotFallback(10).then(tag => ({ result: { tag } }));
      });
  }
  return pick(
    () =>
      http.get("/suggest", {
        params: {
          term
        }
      }),
    () => local.suggestEmptyResponse
  );
}
//菜单右侧gif
export function getMenuIcon() {
  return pick(() => http.get("/menu_gif"), () => local.menuGifResponse);
}
function mapMinibiliRecommendItem(v) {
  return {
    aid: v.id,
    title: v.title,
    pic: v.cover_url,
    author: v.uploader,
    play: v.play_count
  };
}

/** 首页推荐区视频池（用于刷新轮换） */
export function getHomeRecommendPool(limit = 48) {
  if (isMinibili) {
    const fetchPage = cursor => {
      const params = { limit: Math.min(50, limit) };
      if (cursor) {
        params.cursor = cursor;
      }
      return http.get("/api/v1/videos", { params }).then(body => {
        if (!body || body.code !== 0) {
          throw new Error((body && body.msg) || "加载失败");
        }
        const data = body.data || {};
        const items = (data.items || []).map(mapMinibiliRecommendItem);
        return { items, next: data.next_cursor || "" };
      });
    };
    const mergeUnique = (acc, batch) => {
      const seen = new Set(acc.map(v => Number(v.aid)));
      for (const it of batch) {
        const id = Number(it.aid);
        if (!seen.has(id)) {
          seen.add(id);
          acc.push(it);
        }
      }
      return acc;
    };
    return fetchPage("")
      .then(async first => {
        let all = [...first.items];
        let next = first.next;
        let guard = 0;
        while (all.length < limit && next && guard < 6) {
          guard += 1;
          const page = await fetchPage(next);
          mergeUnique(all, page.items);
          next = page.next;
        }
        return all.slice(0, limit);
      })
      .catch(err => {
        console.warn("Mini-Bili 首页推荐池失败，回退本地 mock", err);
        return local.rankingIndexResponse(3).data;
      });
  }
  return pick(
    () =>
      http.get("/ranking/index", {
        params: { day: 3 }
      }),
    () => local.rankingIndexResponse(3)
  ).then(res => {
    const data =
      res && res.data != null
        ? res.data
        : Array.isArray(res)
          ? res
          : [];
    return Array.isArray(data) ? data : [];
  });
}

//首页推荐排行
export function getRankingIndex(day) {
  if (isMinibili) {
    return http
      .get("/api/v1/videos", { params: { limit: 48 } })
      .then(body => {
        if (!body || body.code !== 0) {
          throw new Error((body && body.msg) || "加载失败");
        }
        const items = (body.data && body.data.items) || [];
        return {
          code: 0,
          data: items.map(v => ({
            aid: v.id,
            title: v.title,
            pic: v.cover_url,
            author: v.uploader,
            play: v.play_count,
            in_watch_later: !!v.in_watch_later
          }))
        };
      })
      .catch(err => {
        console.warn("Mini-Bili 首页列表失败，回退本地 mock", err);
        return local.rankingIndexResponse(day);
      });
  }
  return pick(
    () =>
      http.get("/ranking/index", {
        params: {
          day: day //1,3,7
        }
      }),
    () => local.rankingIndexResponse(day)
  );
}
//当前在线数
export function getOnline() {
  if (isMinibili) {
    return http.get("/api/v1/stats/home").then(res => {
      const body = res && res.data;
      if (body && typeof body.code === "number" && body.code !== 0) {
        throw new Error(body.msg || "请求失败");
      }
      const data =
        body && body.data != null ? body.data : body;
      return {
        data: data || { web_online: 0, all_count: 0 }
      };
    });
  }
  return pick(() => http.get("/online"), () => local.onlineResponse);
}
//正在直播
export function getLiveling() {
  return pick(() => http.get("/liveling"), () => local.livelingEmptyResponse);
}
//模块新动态
export function getDynamicRegion(data) {
  const zoneParent = HOME_ZONE_RID_PARENT[data.rid];
  if (isMinibili && zoneParent) {
    return fetchMinibiliZoneVideos({
      zoneParent,
      limit: data.ps,
      sort: "hot"
    })
      .then(({ items, zoneVideoCount }) => ({
        data: {
          archives: items.map(minibiliItemToArchive),
          num: zoneVideoCount
        }
      }))
      .catch(err => {
        console.warn(
          `Mini-Bili 分区「${zoneParent}」新动态失败，回退本地 mock`,
          err
        );
        return local.dynamicRegionResponse(data.rid);
      });
  }
  return pick(
    () =>
      http.get("/dynamic/region", {
        params: {
          ps: data.ps,
          rid: data.rid
        }
      }),
    () => local.dynamicRegionResponse(data.rid)
  );
}
//模块最新投稿
export function getNewlist(data) {
  const zoneParent = HOME_ZONE_RID_PARENT[data.rid];
  if (isMinibili && zoneParent) {
    return fetchMinibiliZoneVideos({
      zoneParent,
      limit: data.ps,
      sort: "time"
    })
      .then(({ items, zoneVideoCount }) => ({
        data: {
          archives: items.map(minibiliItemToArchive),
          num: zoneVideoCount
        }
      }))
      .catch(err => {
        console.warn(
          `Mini-Bili 分区「${zoneParent}」最新投稿失败，回退本地 mock`,
          err
        );
        return local.newlistResponse(data.rid);
      });
  }
  return pick(
    () =>
      http.get("/newlist", {
        params: {
          ps: data.ps,
          rid: data.rid
        }
      }),
    () => local.newlistResponse(data.rid)
  );
}
//模块 三日/一周 排行 全部/原创
export function getRankingRegion(data) {
  const zoneParent = HOME_ZONE_RID_PARENT[data.rid];
  if (isMinibili && zoneParent) {
    return fetchMinibiliZoneVideos({
      zoneParent,
      limit: 10,
      sort: "hot"
    })
      .then(({ items }) => ({
        data: items.map(minibiliItemToRank)
      }))
      .catch(err => {
        console.warn(
          `Mini-Bili 分区「${zoneParent}」排行失败，回退本地 mock`,
          err
        );
        return local.rankingRegionResponse(data.rid);
      });
  }
  return pick(
    () =>
      http.get("/ranking/region", {
        params: {
          rid: data.rid,
          day: data.day,
          original: data.original
        }
      }),
    () => local.rankingRegionResponse(data.rid)
  );
}
//番剧更新时间表
export function getTimelineGlobal() {
  return pick(() => http.get("/timeline_v2_global"), () => local.timelineGlobalResponse);
}
//番剧更新三日排行
export function getRankingGlobal3() {
  return pick(() => http.get("/ranking/global_3"), () => local.rankingGlobal3Response);
}
//番剧更新七日排行
export function getRankingGlobal7() {
  return pick(() => http.get("/ranking/global_7"), () => local.rankingGlobal7Response);
}
//国创更新时间表
export function getTimelineCn() {
  return pick(() => http.get("/timeline_v2_cn"), () => local.timelineCnResponse);
}
//国创更新三日排行
export function getRankingCn3() {
  return pick(() => http.get("/ranking/cn_3"), () => local.rankingCn3Response);
}
//国创更新七日排行
export function getRankingCn7() {
  return pick(
    () =>
      http.get("/ranking/cn_7", {
        headers: { "Content-Type": "application/json" }
      }),
    () => local.rankingCn7Response
  );
}

//番剧广告位
export function getGlobalAdSlide() {
  return pick(() => http.get("/public/fj_ad_slide.json"), () => local.fjAdSlideResponse);
}

//国创广告位
export function getCnAdSlide() {
  return pick(() => http.get("/public/gc_ad_slide.json"), () => local.gcAdSlideResponse);
}

//排行榜数据
//全站、原创、新人排行榜
export function getRanking(type, rid, arc_type = 0, day) {
  if (isMinibili) {
    const rankType = Number(type);
    const ridStr = String(rid ?? "0");
    /** 当前实现：全站榜各分区 */
    if (rankType !== 1) {
      return Promise.resolve({
        data: {
          note: RANK_PAGE_NOTE,
          list: []
        }
      });
    }
    const zoneParent = RANKING_RID_ZONE[ridStr] ?? "";
    const days = normalizeRankDays(day);
    const arcType = Number(arc_type) === 1 ? 1 : 0;
    return fetchMinibiliZoneVideos({
      limit: 100,
      sort: "hot",
      zoneParent: zoneParent || undefined,
      days,
      arc_type: arcType
    })
      .then(({ items }) => {
        const ranked = sortRankPool(items, { arcType, days });
        return {
          data: {
            note: RANK_PAGE_NOTE,
            list: ranked.map(({ v, score }) => mapVideoToRankListItem(v, score))
          }
        };
      })
      .catch(err => {
        console.warn("Mini-Bili 全站排行榜失败", err);
        return {
          data: {
            note: RANK_PAGE_NOTE,
            list: []
          }
        };
      });
  }
  return pick(
    () =>
      http.get("/ranking", {
        params: {
          rid,
          day,
          type,
          arc_type
        }
      }),
    () => ({ data: local.emptyRankPagePayload })
  );
}
//新番排行榜
export function getSeasonRank(day, season_type) {
  return pick(
    () =>
      http.get("/season/rank/list", {
        params: {
          day: day,
          season_type: season_type
        }
      }),
    () => local.emptySeasonRankPayload
  );
}
//影视排行榜
export function getMoviesRank(day, rid) {
  return pick(
    () => http.get(`/ranking/movies/all-${day}-${rid}.json`),
    () => local.emptyMoviesRankPayload
  );
}

//搜索结果
export function getSearchResult(highlight, keyword, opts = {}) {
  if (isMinibili) {
    const type = opts.type || "all";
    const sort = opts.sort || "";
    const vf = opts.videoFilters || {};
    const params = {
      highlight,
      keyword,
      type,
      sort: sort || undefined,
      page: opts.page || 1,
      page_size: opts.page_size || 20
    };
    if (vf.order && vf.order !== "default") {
      params.order = vf.order;
    }
    if (vf.duration && vf.duration !== "all") {
      params.duration = vf.duration;
    }
    if (vf.zone) {
      params.zone = vf.zone;
    }
    return http
      .get("/api/v1/search", {
        params,
        skipGlobalErrorToast: true
      })
      .then(body => {
        if (!body || body.code !== 0) {
          const e = new Error((body && body.msg) || "搜索失败");
          e.minibiliApiCode = body && body.code;
          throw e;
        }
        const data = body.data || emptySearchPayload.data;
        if (!data.search_status) {
          const r = data.result || {};
          const has =
            (r.video && r.video.length) ||
            (r.article && r.article.length) ||
            (r.bili_user && r.bili_user.length);
          data.search_status = has ? "ok" : "empty";
        }
        return { code: 0, data };
      })
      .catch(err => {
        console.warn("Mini-Bili 搜索失败", err);
        const base = { ...(emptySearchPayload.data || {}) };
        if (err && err.minibiliApiCode === 50301) {
          base.search_status = "unavailable";
        } else {
          base.search_status = base.search_status || "unavailable";
        }
        return { code: 0, data: base };
      });
  }
  return pick(
    () =>
      http.get("/search/all", {
        params: {
          highlight,
          keyword
        }
      }),
    () => local.emptySearchPayload
  );
}
//根据media_id获取详细信息
export function getSeason(id) {
  return pick(
    () =>
      http.get("/search/season", {
        params: {
          media_id: id
        }
      }),
    () => local.emptySeasonDetail
  );
}
