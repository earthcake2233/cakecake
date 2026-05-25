import {
  getLoc,
  getSearchDefaultWords,
  getSuggest,
  getMenuIcon
} from "../../api";
import { buildHeaderMenuLeftZones } from "@/constants/videoZones";
import square01 from "../../assets/square_01.jpg";
import square02 from "../../assets/square_02.jpg";
import live01 from "../../assets/live_01.png";
import live02 from "../../assets/live_02.png";

const state = {
  leftNav: [
    //顶部左侧导航栏
    {
      name: "主站",
      class: "home",
      icon: "bili-icon",
      href: "/"
    },
    {
      name: "画友",
      class: "hbili",
      href: "/huayou"
    },
    {
      name: "游戏中心",
      class: "game",
      href: "/"
    },
    {
      name: "直播",
      class: "live",
      href: "/"
    },
    {
      name: "会员购",
      class: "buy",
      href: "/"
    },
    {
      name: "BML",
      href: "/"
    },
    {
      name: "下载APP",
      class: "mobile",
      icon: "bili-icon",
      href: "/"
    }
  ],
  headBanner: [], //顶部背景、LOGO
  searchValue: "", //搜索框输入值
  searchWord: [], //默认搜索关键字
  suggest: { tag: [] }, //建议搜索
  menuLeft: [
    {
      name: "首页",
      class: "home",
      href: ""
    },
    ...buildHeaderMenuLeftZones()
  ], //主要菜单左侧（分区与 constants/videoZones.js 一致）
  menuRight: [
    {
      name: "专栏",
      class: "zl",
      icon: "zhuanlan",
      href: "/",
      fieldClass: "",
      fields: []
    },
    {
      name: "广场",
      class: "nav-square",
      icon: "square",
      href: "/",
      fieldClass: "square-wrap",
      fields: [
        {
          name: "会员购",
          icon: "icon-vip-buy",
          href: ""
        },
        {
          name: "活动中心",
          icon: "icon-activity",
          href: ""
        },
        {
          name: "游戏中心",
          icon: "icon-game",
          href: ""
        },
        {
          name: "新闻中心",
          icon: "icon-news",
          href: ""
        },
        {
          name: "画友",
          icon: "icon-hy",
          href: ""
        },
        {
          name: "芒果TV",
          icon: "icon-mango",
          href: ""
        }
      ],
      fieldImgClass: "square-field",
      fieldImg: [
        {
          title: "bilibili 活动",
          href: "",
          src: square01
        },
        {
          title: "话题列表",
          href: "",
          src: square02
        }
      ]
    },
    {
      name: "直播",
      class: "",
      icon: "live",
      href: "/",
      fieldClass: "nav-live",
      fields: [
        {
          name: "推荐主播",
          href: ""
        },
        {
          name: "生活娱乐",
          href: ""
        },
        {
          name: "绘画专区",
          href: ""
        },
        {
          name: "唱见舞见",
          href: ""
        },
        {
          name: "御宅文化",
          href: ""
        },
        {
          name: "单机联机",
          href: ""
        },
        {
          name: "网络游戏",
          href: ""
        },
        {
          name: "电子竞技",
          href: ""
        },
        {
          name: "手游直播",
          href: ""
        }
      ],
      fieldImgClass: "live-field",
      fieldImg: [
        {
          title: "有文画",
          href: "",
          imgclass: "pic",
          src: live01
        },
        {
          title: "小视频",
          href: "",
          imgclass: "pic",
          src: live02
        }
      ]
    },
    {
      name: "小黑屋",
      class: "",
      icon: "blackroom",
      href: "/",
      fieldClass: "",
      fields: []
    }
  ], //主要菜单右侧
  menuIcon: [] //主要菜单右侧icon
};

const getters = {};

const mutations = {
  SET_HEAD_BANNER: (state, data) => {
    state.headBanner = Object.assign({}, data[0]);
  },
  SET_SEARCH_DEFAULT_WORDS: (state, data) => {
    state.searchWord = Object.assign({}, data);
  },
  SET_MENUICON: (state, data) => {
    state.menuIcon = Object.assign({}, data);
  },
  SET_SEARCH_WORD: (state, data) => {
    state.searchValue = data;
  },
  SET_SUGGEST: (state, data) => {
    state.suggest = data;
  }
};

const actions = {
  setHeadBanner({ commit }, data) {
    getLoc(data).then(res => {
      commit("SET_HEAD_BANNER", res.data);
    });
  },
  setSearchDefaultWords({ commit }) {
    getSearchDefaultWords().then(res => {
      commit("SET_SEARCH_DEFAULT_WORDS", res.data);
    });
  },
  setSuggest({ commit, state }) {
    const term = String(state.searchValue || "").trim();
    if (!term) {
      commit("SET_SUGGEST", { tag: [] });
      return;
    }
    getSuggest(term).then(res => {
      const payload = res && res.result;
      if (payload && Array.isArray(payload.tag)) {
        commit("SET_SUGGEST", payload);
      } else if (Array.isArray(payload)) {
        commit("SET_SUGGEST", { tag: payload });
      } else {
        commit("SET_SUGGEST", { tag: [] });
      }
    });
  },
  setMenuIcon({ commit }) {
    getMenuIcon().then(res => {
      commit("SET_MENUICON", res.data);
    });
  }
};

export default {
  namespaced: true, //注册header空间模块
  state,
  getters,
  actions,
  mutations
};
