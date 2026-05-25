/**
 * 本地接口数据：封面 / 轮播等图片均来自 src/assets，便于离线开发与演示。
 * 若需走远程 axios，在项目根目录设置 VITE_USE_REMOTE_API=true（见 .env.example）。
 */
import navBg from "../assets/nav-bg.png";
import headLogo from "../assets/head-logo.png";
import faceAkari from "../assets/akari.jpg";
import lbGif from "../assets/lb.gif";

const posters = [faceAkari];

function archive(aid, title, idx) {
  const poster = posters[idx % posters.length];
  return {
    aid,
    title,
    pic: poster,
    duration: 180 + (aid % 200),
    stat: {
      view: 8000 + aid * 17,
      danmaku: 88 + (aid % 500)
    }
  };
}

function archivesBlock(baseAid, n) {
  return Array.from({ length: n }, (_, i) =>
    archive(baseAid + i, `示例投稿 ${baseAid + i}`, i)
  );
}

export const locResponse = {
  code: 0,
  data: [
    {
      pic: navBg,
      litpic: headLogo
    }
  ]
};

export const locsResponse = {
  code: 0,
  data: {
    "23": Array.from({ length: 5 }, (_, i) => ({
      url: "/",
      name: `本地轮播 ${i + 1}`,
      pic: posters[i % posters.length]
    })),
    "34": Array.from({ length: 6 }, (_, i) => ({
      url: "//www.bilibili.com/video/av170001",
      name: `本地推广 ${i + 1}`,
      pic: posters[(i + 2) % posters.length],
      archive: { duration: 520 + i * 40, aid: 35899404 + i }
    }))
  }
};

/** 无后端热搜数据时的占位词（与 dynamicsFeedSeed 一致） */
export const HOT_SEARCH_FALLBACK_TITLES = [
  "春季新番开播一览",
  "UP 主创作激励计划更新",
  "Mini-Bili 联调实录",
  "本周热门游戏实况",
  "开源播放器性能对比",
  "弹幕礼仪小课堂",
  "二创剪辑技巧分享",
  "直播回放剪辑教程",
  "社区公约解读",
  "投稿封面设计灵感"
];

export const searchDefaultResponse = {
  code: 0,
  data: {
    show_name: HOT_SEARCH_FALLBACK_TITLES[0],
    word: HOT_SEARCH_FALLBACK_TITLES[0],
    hot_list: HOT_SEARCH_FALLBACK_TITLES
  }
};

export const suggestEmptyResponse = {
  code: 0,
  result: { tag: [] }
};

export const menuGifResponse = {
  code: 0,
  data: {
    links: ["/"],
    title: "本地活动",
    icon: lbGif
  }
};

export const rankingIndexResponse = day => ({
  code: 0,
  data: Array.from({ length: 16 }, (_, i) => ({
    aid: 300000 + i,
    title: `推荐内容 ${day}日-${i + 1}`,
    pic: posters[i % posters.length],
    author: "本地 UP",
    play: 12000 + i * 888
  }))
});

export const onlineResponse = {
  code: 0,
  data: {
    web_online: 1234567,
    all_count: 987654
  }
};

export const livelingEmptyResponse = {
  code: 0,
  data: []
};

const dynamicByRid = {
  1: archivesBlock(400100, 12),
  13: archivesBlock(500100, 12),
  168: archivesBlock(600100, 12),
  3: archivesBlock(700100, 12),
  129: archivesBlock(800100, 12)
};

export function dynamicRegionResponse(rid) {
  const list = dynamicByRid[rid] || archivesBlock(900100 + Number(rid) * 10, 10);
  return {
    code: 0,
    data: {
      archives: list,
      num: list.length
    }
  };
}

export function newlistResponse(rid) {
  const base = 200000 + Number(rid) * 1000;
  const list = dynamicByRid[rid] || archivesBlock(base, 12);
  return {
    code: 0,
    data: {
      archives: archivesBlock(base, 12),
      num: list.length
    }
  };
}

function rankVideoItem(aid, i) {
  return {
    aid,
    title: `排行榜条目 ${i + 1}`,
    pic: posters[i % posters.length],
    pts: 120 - i * 3
  };
}

export function rankingRegionResponse(rid) {
  const r = Number(rid);
  const items = Array.from({ length: 10 }, (_, i) =>
    rankVideoItem(900001 + r * 100 + i, i)
  );
  return {
    code: 0,
    data: items
  };
}

function bangumiRankEntry(i) {
  return {
    aid: 700001 + i,
    title: `番剧排行 ${i + 1}`,
    pic: posters[i % posters.length],
    newest_ep_index: 12 - (i % 8)
  };
}

export const rankingGlobal3Response = {
  code: 0,
  result: {
    list: Array.from({ length: 10 }, (_, i) => bangumiRankEntry(i))
  }
};

export const rankingGlobal7Response = {
  code: 0,
  result: {
    list: Array.from({ length: 10 }, (_, i) => bangumiRankEntry(i + 3))
  }
};

export const rankingCn3Response = rankingGlobal3Response;
export const rankingCn7Response = rankingGlobal7Response;

function timelineItem(i, weekday, isNew) {
  return {
    season_id: 10000 + i,
    title: `本地番剧 ${weekday}-${i}`,
    square_cover: posters[i % posters.length],
    bgmcount: 10 + (i % 5),
    ep_id: 200000 + i,
    weekday,
    new: isNew,
    favorites: 5000 - i * 100
  };
}

export const timelineGlobalResponse = {
  code: 0,
  result: [
    timelineItem(0, new Date().getDay() || 7, true),
    timelineItem(1, 1, false),
    timelineItem(2, 2, false),
    timelineItem(3, 3, true),
    timelineItem(4, 4, false),
    timelineItem(5, 5, false),
    timelineItem(6, 6, false),
    timelineItem(7, 7, false)
  ]
};

export const timelineCnResponse = {
  code: 0,
  result: timelineGlobalResponse.result.map(e => ({
    ...e,
    title: `国创 · ${e.title}`,
    season_id: e.season_id + 5000
  }))
};

export const adSlideResponse = {
  code: 0,
  result: [
    { link: "/", title: "本地推荐 1", img: faceAkari },
    { link: "/", title: "本地推荐 2", img: faceAkari }
  ]
};

export const fjAdSlideResponse = {
  code: 0,
  result: [
    { link: "/", title: "番剧侧栏", img: faceAkari },
    { link: "/", title: "番剧侧栏 2", img: faceAkari }
  ]
};

export const gcAdSlideResponse = {
  code: 0,
  result: [
    { link: "/", title: "国创侧栏", img: faceAkari },
    { link: "/", title: "国创侧栏 2", img: faceAkari }
  ]
};

export const loginUserResponse = {
  code: 0,
  message: "0",
  ttl: 1,
  data: {
    isLogin: true,
    email_verified: 1,
    face: faceAkari,
    level_info: {
      current_level: 4,
      current_min: 4500,
      current_exp: 7567,
      next_exp: 10800
    },
    mid: 6700464,
    mobile_verified: 1,
    money: 0,
    moral: 70,
    officialVerify: { type: -1, desc: "" },
    pendant: { pid: 0, name: "", image: "", expire: 0 },
    scores: 0,
    uname: "本地用户",
    vipDueDate: 1893465600000,
    vipStatus: 0,
    vipType: 1,
    wallet: {
      mid: 6700464,
      bcoin_balance: 0,
      coupon_balance: 0,
      coupon_due_time: 0
    },
    has_shop: false
  }
};

export const topInfoResponse = {
  code: 0,
  data: {
    picAndWords: [
      {
        id: "181",
        content: "本地大会员示例",
        imageUrl: faceAkari,
        image1Url: faceAkari,
        image2Url: faceAkari,
        image3Url: "",
        linkUrl: "/",
        tagType: -1,
        iosImage: faceAkari,
        androidImage: faceAkari,
        ipadImage: faceAkari,
        mobileLink: "/",
        ipadLink: "/"
      }
    ]
  }
};

export const emptySearchPayload = {
  code: 0,
  data: {
    result: {
      video: [],
      media_bangumi: [],
      media_ft: [],
      live: [],
      article: [],
      topic: [],
      bili_user: [],
      photo: []
    },
    top_tlist: {
      video: 0,
      media_bangumi: 0,
      movie: 0,
      live: 0,
      article: 0,
      topic: 0,
      bili_user: 0,
      photo: 0
    }
  }
};

export const emptySeasonDetail = {
  code: 0,
  result: {
    actors: "",
    alias: "",
    areas: "",
    cover: faceAkari,
    evaluate: "",
    jp_title: "",
    link: "",
    media_id: 0,
    pub_time: "",
    rating: { score: 0, count: 0 },
    share_url: "",
    square_cover: faceAkari,
    subtitle: "",
    title: "",
    type_name: ""
  }
};

export const emptyRankPagePayload = {
  note: "根据稿件内容质量、近期的数据综合展示，动态更新",
  list: []
};

export const emptySeasonRankPayload = {
  code: 0,
  result: {
    list: []
  }
};

export const emptyMoviesRankPayload = {
  code: 0,
  rank: {
    note: "",
    list: []
  }
};
