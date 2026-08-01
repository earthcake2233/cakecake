import defaultCover from "@/assets/akari.jpg";

const covers = [defaultCover];

/** 额外生成的占位条目（用于滚动与分页观感测试） */
const EXTRA_SEEDS = [
  {
    title: "「开箱」桌面麦克风套装：人声与降噪一期讲清",
    zone: "数码",
    tags: ["开箱", "评测"],
    duration: "08:42",
    status: "passed"
  },
  {
    title: "通勤一周实录｜地铁读书与晚间剪辑流水账",
    zone: "生活-日常",
    tags: ["Vlog", "通勤"],
    duration: "06:15",
    status: "passed"
  },
  {
    title: "Unity 粒子新手向：10 分钟做一个简易星空背景",
    zone: "游戏",
    tags: ["Unity", "教程"],
    duration: "11:03",
    status: "passed"
  },
  {
    title: "翻唱练习｜夏末抒情曲（干声未修）",
    zone: "音乐",
    tags: ["翻唱", "练习"],
    duration: "04:28",
    status: "processing"
  },
  {
    title: "纪录片剪辑手记：节奏点怎么卡才不拖沓",
    zone: "影视",
    tags: ["剪辑", "心得"],
    duration: "15:47",
    status: "passed"
  },
  {
    title: "料理失败合集｜同一种酱汁为何第三次才成功",
    zone: "美食",
    tags: ["料理", "翻车"],
    duration: "07:55",
    status: "passed"
  },
  {
    title: "像素风小游戏 Debug 直播回放（高能空降见简介）",
    zone: "游戏",
    tags: ["直播", "独立游戏"],
    duration: "02:18:06",
    status: "passed"
  },
  {
    title: "封面字体排版复盘：三张稿对比选最优",
    zone: "动画",
    tags: ["设计", "复盘"],
    duration: "09:12",
    status: "rejected"
  },
  {
    title: "阳台绿植月度修剪｜浇水频率笔记分享",
    zone: "生活-日常",
    tags: ["绿植", "记录"],
    duration: "05:01",
    status: "passed"
  },
  {
    title: "Blender 低多边形场景：日落湖边加速全过程",
    zone: "动画",
    tags: ["Blender", "场景"],
    duration: "22:36",
    status: "passed"
  },
  {
    title: "书评短视频｜一本关于专注力的薄册子",
    zone: "知识",
    tags: ["书评", "读书笔记"],
    duration: "06:40",
    status: "passed"
  },
  {
    title: "键盘打字音｜线性轴 vs 段落轴慢对比",
    zone: "数码",
    tags: ["键盘", "打字音"],
    duration: "03:52",
    status: "passed"
  },
  {
    title: "CityWalk 夜拍｜雨后路灯与橱窗反光",
    zone: "摄影",
    tags: ["夜景", "扫街"],
    duration: "13:09",
    status: "processing"
  },
  {
    title: "配音练习｜双人对话气息与停顿标注",
    zone: "动画",
    tags: ["配音", "练习"],
    duration: "08:24",
    status: "passed"
  },
  {
    title: "期末复习背景音｜番茄钟 + 白噪声合成一期",
    zone: "音乐",
    tags: ["学习", "背景音"],
    duration: "45:00",
    status: "passed"
  },
  {
    title: "小车模型喷漆修补｜遮盖带容易犯的错误",
    zone: "手工",
    tags: ["模型", "喷漆"],
    duration: "10:18",
    status: "passed"
  },
  {
    title: "公开课剪辑｜课堂板书放大与字幕对齐技巧",
    zone: "知识",
    tags: ["字幕", "公开课"],
    duration: "18:44",
    status: "passed"
  },
  {
    title: "骑行通勤｜新车灯光与头盔记录仪视角测试",
    zone: "运动",
    tags: ["骑行", "通勤"],
    duration: "12:31",
    status: "passed"
  },
  {
    title: "字幕组协作演示｜听译分工与时间轴交接",
    zone: "影视",
    tags: ["字幕", "协作"],
    duration: "14:02",
    status: "passed"
  },
  {
    title: "复古游戏机拆解｜清灰换导热硅脂实录",
    zone: "数码",
    tags: ["拆解", "硬件"],
    duration: "16:58",
    status: "passed"
  },
  {
    title: "水彩速写｜海边礁石与浪花（附调色思路）",
    zone: "绘画",
    tags: ["水彩", "速写"],
    duration: "07:33",
    status: "passed"
  },
  {
    title: "本地演示｜长列表最后一条：用于确认滚动到底边框",
    zone: "演示",
    tags: ["滚动测试"],
    duration: "01:00",
    status: "passed"
  }
];

function buildExtraVideos() {
  const baseMonth = 10;
  return EXTRA_SEEDS.map((seed, i) => {
    const id = 4 + i;
    const day = 28 - ((i * 3) % 27);
    const hour = 9 + ((i * 5) % 12);
    const min = (i * 7) % 60;
    const sec = (i * 11) % 60;
    const mult = 1 + (i % 7) * 0.37;
    const cover = covers[i % covers.length];
    return {
      id,
      title: seed.title,
      publishedAt: `2024年${String(baseMonth).padStart(2, "0")}月${String(day).padStart(2, "0")}日 ${String(hour).padStart(2, "0")}:${String(min).padStart(2, "0")}:${String(sec).padStart(2, "0")}`,
      duration: seed.duration,
      cover,
      view: Math.floor(1200 * mult * mult),
      danmu: Math.floor(80 * mult),
      reply: Math.floor(40 * mult),
      coin: Math.floor(60 * mult),
      fav: Math.floor(200 * mult),
      share: Math.floor(18 * mult),
      status: seed.status,
      fileName: `demo_${id}_${seed.zone}.mp4`,
      videoType: "自制",
      zone: seed.zone,
      tags: seed.tags,
      intro: i % 4 === 0 ? "滚动测试用占位简介。" : ""
    };
  });
}

/** 稿件列表 / 编辑页共用的本地演示数据 */
export const CREATOR_VIDEO_LIST = [
  {
    id: 1,
    title:
      "【本地演示】稿件标题示例：用于对齐创作中心列表排版与数据图标展示",
    publishedAt: "2023年12月30日 16:26:10",
    duration: "00:00:50",
    cover: defaultCover,
    view: 125680,
    danmu: 3421,
    reply: 856,
    coin: 1203,
    fav: 8932,
    share: 412,
    status: "passed",
    fileName: "本地演示稿件源文件.mp4",
    videoType: "自制",
    zone: "动画",
    tags: ["录屏", "演示", "创作中心"],
    intro: ""
  },
  {
    id: 2,
    title: "第二条演示视频：统计数据与封面时长角标样式参考",
    publishedAt: "2023年11月02日 09:15:33",
    duration: "00:12:07",
    cover: defaultCover,
    view: 8942,
    danmu: 210,
    reply: 88,
    coin: 156,
    fav: 620,
    share: 34,
    status: "passed",
    fileName: "第二条演示视频_h264.mp4",
    videoType: "自制",
    zone: "生活-日常",
    tags: ["Vlog", "日常"],
    intro: "演示简介占位。"
  },
  {
    id: 3,
    title: "第三条占位稿件：播放、弹幕、评论、投币、收藏、转发图标顺序固定",
    publishedAt: "2023年08月18日 21:40:00",
    duration: "00:03:22",
    cover: defaultCover,
    view: 502341,
    danmu: 12056,
    reply: 4302,
    coin: 8901,
    fav: 21098,
    share: 1203,
    status: "processing",
    fileName: "第三条占位稿件_4K演示.mp4",
    videoType: "自制",
    zone: "动画",
    tags: ["手游", "直播"],
    intro: ""
  },
  ...buildExtraVideos()
];

export function findCreatorVideo(id) {
  const n = Number(id);
  return CREATOR_VIDEO_LIST.find((v) => v.id === n);
}
