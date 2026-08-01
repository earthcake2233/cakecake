/**
 * 与顶栏分区导航（store/modules/header.js menuLeft）一致的可投稿分区。
 * 创作中心、播放页面包屑、后端 zone 校验均以此为准。
 */
export const VIDEO_ZONE_CATEGORIES = [
  {
    name: "动画",
    items: ["MAD·AMV", "MMD·3D", "短片·手书·配音", "综合"]
  },
  {
    name: "番剧",
    items: ["连载动画", "完结动画", "资讯", "官方延伸", "新番时间表", "番剧索引"]
  },
  {
    name: "国创",
    items: [
      "国产动画",
      "国产原创相关",
      "布袋戏",
      "资讯",
      "新番时间表",
      "国产动画索引"
    ]
  },
  {
    name: "音乐",
    items: [
      "原创音乐",
      "翻唱",
      "VOCALOID·UTAU",
      "演奏",
      "三次元音乐",
      "OP/ED/OST",
      "音乐选集"
    ]
  },
  {
    name: "舞蹈",
    items: ["宅舞", "三次元舞蹈", "舞蹈教程"]
  },
  {
    name: "游戏",
    items: [
      "单机游戏",
      "电子竞技",
      "手机游戏",
      "网络游戏",
      "桌游棋牌",
      "GMV",
      "音游",
      "Mugen"
    ]
  },
  {
    name: "科技",
    items: [
      "趣味科普人文",
      "野生技术协会",
      "演讲·公开课",
      "星海",
      "数码",
      "机械",
      "汽车"
    ]
  },
  {
    name: "生活",
    items: [
      "搞笑",
      "日常",
      "美食圈",
      "动物圈",
      "手工",
      "绘画",
      "ASMR",
      "运动",
      "其他"
    ]
  },
  {
    name: "鬼畜",
    items: ["鬼畜调教", "音MAD", "人力VOCALOID", "教程演示"]
  },
  {
    name: "时尚",
    items: ["美妆", "服饰", "健身", "资讯"]
  },
  { name: "广告", items: [] },
  {
    name: "娱乐",
    items: ["综艺", "明星", "Korea相关"]
  },
  {
    name: "影视",
    items: ["影视杂谈", "影视剪辑", "短片", "预告·资讯", "特摄"]
  },
  {
    name: "放映厅",
    items: ["纪录片", "电影", "电视剧"]
  }
];

/** 顶栏展示用投稿量角标（与原版 menuLeft.num 一致） */
export const VIDEO_ZONE_MENU_NUMS = {
  动画: 884,
  番剧: 105,
  国创: 103,
  音乐: 12389,
  舞蹈: 199,
  游戏: 23890,
  科技: 27893,
  生活: 12678,
  鬼畜: 68,
  时尚: 859,
  广告: 205,
  娱乐: 1280,
  影视: 3450,
  放映厅: 147
};

/** @returns {{ name: string, num: number, href: string, items: { name: string, href: string }[] }[]}} */
export function buildHeaderMenuLeftZones() {
  return VIDEO_ZONE_CATEGORIES.map(cat => ({
    name: cat.name,
    num: VIDEO_ZONE_MENU_NUMS[cat.name] ?? 0,
    href: "",
    items: (cat.items || []).map(sub => ({ name: sub, href: "" }))
  }));
}

/** @returns {string} 创作中心展示用，如「动画 / MAD·AMV」 */
export function formatVideoZoneLabel(value) {
  const z = String(value || "").trim();
  if (!z) {
    return "";
  }
  const idx = z.indexOf("-");
  if (idx > 0) {
    return `${z.slice(0, idx)} / ${z.slice(idx + 1)}`;
  }
  return z;
}

/** @returns {{ group: string, value: string, label: string }[]}} */
export function buildVideoZoneSelectOptions() {
  const options = [];
  for (const cat of VIDEO_ZONE_CATEGORIES) {
    options.push({
      group: cat.name,
      value: cat.name,
      label: cat.name
    });
    for (const sub of cat.items || []) {
      options.push({
        group: cat.name,
        value: `${cat.name}-${sub}`,
        label: `${cat.name} → ${sub}`
      });
    }
  }
  return options;
}

/** @returns {{ name: string, options: { value: string, label: string }[] }[]}} */
export function buildVideoZoneSelectGroups() {
  const groups = [];
  for (const cat of VIDEO_ZONE_CATEGORIES) {
    const options = [{ value: cat.name, label: cat.name }];
    for (const sub of cat.items || []) {
      options.push({
        value: `${cat.name}-${sub}`,
        label: `${cat.name} → ${sub}`
      });
    }
    groups.push({ name: cat.name, options });
  }
  return groups;
}

const _optionValues = new Set(
  buildVideoZoneSelectOptions().map(o => o.value)
);

/** @param {string} value */
export function isKnownVideoZone(value) {
  return _optionValues.has(String(value || "").trim());
}

/** @param {string} value */
export function normalizeVideoZoneValue(value) {
  let z = String(value || "").trim();
  if (!z) {
    return "";
  }
  z = z.replace(/ → /g, "-").replace(/→/g, "-").replace(/—/g, "-");
  if (isKnownVideoZone(z)) {
    return z;
  }
  return "";
}
