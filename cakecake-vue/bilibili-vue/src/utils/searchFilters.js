/** 与 searchFilter.vue / 后端 search.ParseVideoFilter 一致 */

export const SEARCH_ORDER_OPTIONS = [
  { id: "default", name: "综合排序" },
  { id: "click", name: "最多点击" },
  { id: "pubdate", name: "最新发布" },
  { id: "dm", name: "最多弹幕" },
  { id: "fav", name: "最多收藏" }
];

export const SEARCH_DURATION_OPTIONS = [
  { id: "all", name: "全部时长" },
  { id: "lt10", name: "10分钟以下" },
  { id: "m10_30", name: "10-30分钟" },
  { id: "m30_60", name: "30-60分钟" },
  { id: "gt60", name: "60分钟以上" }
];

export const SEARCH_ZONE_OPTIONS = [
  { id: "", name: "全部分区" },
  { id: "动画", name: "动画" },
  { id: "番剧", name: "番剧相关" },
  { id: "国创", name: "国创" },
  { id: "音乐", name: "音乐" },
  { id: "舞蹈", name: "舞蹈" },
  { id: "游戏", name: "游戏" },
  { id: "科技", name: "科技" },
  { id: "生活", name: "生活" },
  { id: "鬼畜", name: "鬼畜" },
  { id: "时尚", name: "时尚" },
  { id: "广告", name: "广告" },
  { id: "娱乐", name: "娱乐" },
  { id: "影视", name: "影视" },
  { id: "纪录片", name: "纪录片" },
  { id: "电影", name: "电影" },
  { id: "电视剧", name: "电视剧" }
];

export const DEFAULT_VIDEO_FILTERS = {
  order: "default",
  duration: "all",
  zone: ""
};

/** grid = 卡片矩阵；list = 行详情（缩略图左 + 文案右） */
export const SEARCH_VIDEO_VIEW_GRID = "grid";
export const SEARCH_VIDEO_VIEW_LIST = "list";
export const DEFAULT_SEARCH_VIDEO_VIEW = SEARCH_VIDEO_VIEW_LIST;

export function videoFiltersToParams(filters) {
  const f = filters || DEFAULT_VIDEO_FILTERS;
  const params = {};
  if (f.order && f.order !== "default") {
    params.order = f.order;
  }
  if (f.duration && f.duration !== "all") {
    params.duration = f.duration;
  }
  if (f.zone) {
    params.zone = f.zone;
  }
  return params;
}
