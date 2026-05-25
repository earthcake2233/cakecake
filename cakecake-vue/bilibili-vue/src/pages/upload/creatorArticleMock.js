import defaultCover from "@/assets/akari.jpg";

/** 创作中心「图文管理 → 图文」列表演示数据 */
export const CREATOR_ARTICLE_LIST = [
  {
    id: 101,
    title: "现在开发岗还缺人吗",
    publishedAt: "2024年06月01日 12:30:08",
    cover: defaultCover,
    kind: "column",
    kindLabel: "专栏",
    tags: ["实习交流"],
    view: 15230,
    reply: 128,
    like: 892,
    fav: 234,
    status: "passed"
  },
  {
    id: 102,
    title: "本地演示：图文列表封面与数据栏对齐参考",
    publishedAt: "2024年05月18日 09:15:22",
    cover: defaultCover,
    kind: "moment",
    kindLabel: "动态",
    tags: ["日常"],
    view: 3201,
    reply: 45,
    like: 210,
    fav: 56,
    status: "passed"
  },
  {
    id: 103,
    title: "第三条演示：审核中的图文稿件样式",
    publishedAt: "2024年05月10日 18:00:00",
    cover: defaultCover,
    kind: "post",
    kindLabel: "小站帖子",
    tags: ["讨论"],
    view: 0,
    reply: 0,
    like: 0,
    fav: 0,
    status: "processing"
  }
];
