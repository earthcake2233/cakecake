import * as types from "./mutation-types";
import { getHomeBannersPublic } from "../api/admin";
import {
  getAdSlide,
  getLocs,
  getRankingIndex,
  getOnline,
  getDynamicRegion,
  getNewlist,
  getRankingRegion,
  getTimelineGlobal,
  getRankingGlobal3,
  getRankingGlobal7,
  getTimelineCn,
  getRankingCn3,
  getRankingCn7
} from "../api";

const isMinibiliApi =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

//广告位轮播图
export const setAdSlide = function({ commit }, data) {
  const _data = {
    id: data.id,
    rid: data.rid
  };
  const params = {
    position_id: data.position_id
  };
  getAdSlide(params).then(res => {
    _data.data = res.result;
    commit(types.SET_AD_SLIDE, _data);
  });
};
//轮播图，推广模块
export const setSlide = function({ commit }, data) {
  if (isMinibiliApi) {
    getHomeBannersPublic()
      .then(body => {
        const items = (body.data && body.data.items) || [];
        commit(types.SET_SLIDE, items);
        commit(types.SET_POPULARIZE, []);
      })
      .catch(() => {
        commit(types.SET_SLIDE, []);
        commit(types.SET_POPULARIZE, []);
      });
    return;
  }
  getLocs(data).then(res => {
    commit(types.SET_SLIDE, res.data["23"]);
    commit(types.SET_POPULARIZE, res.data["34"]);
  });
};
//推荐模块
export const setRankingIndex = function({ commit }, day) {
  getRankingIndex(day).then(res => {
    commit(types.SET_RANKING_INDEX, {
      data: res.data,
      day: day
    });
  });
};
//当前在线
export const setOnline = function({ commit }) {
  getOnline().then(res => {
    commit(types.SET_ONLINE, res.data);
  });
};
//新动态
export const setDynamicRegion = function({ commit }, data) {
  // console.log(data)
  const params = {
    ps: data.ps,
    rid: data.rid
  };
  //获取新动态
  //设置当前动态区域数据
  return getDynamicRegion(params).then(res => {
    const _data = {
      data: res.data,
      id: data.id
    };
    commit(types.SET_STOREY_DATA, _data);
  });
};
//最新投稿
export const setNewlist = function({ commit }, data) {
  const params = {
    ps: data.ps,
    rid: data.rid
  };
  //获取最新投稿
  //设置当前动态区域数据
  return getNewlist(params).then(res => {
    const _data = {
      data: res.data,
      id: data.id
    };
    commit(types.SET_STOREY_DATA, _data);
  });
};
//排行榜
export const setRankingRegion = function({ commit }, data) {
  const params = {
    rid: data.rid,
    day: data.day,
    original: data.original
  };
  const _data = {
    id: data.id,
    original: data.original,
    data: []
  };
  function jsonChange(res) {
    if (res && typeof res === "object" && res.result !== undefined) {
      return res;
    }
    if (!res || typeof res !== "string") {
      return { result: [] };
    }
    const num1 = res.indexOf("(");
    const num2 = res.length - 2;
    return JSON.parse(res.substring(num1 + 1, num2));
  }
  function getRank() {
    getRankingRegion(params).then(res => {
      _data.data = res.data;
      commit(types.SET_RANKING_DATA, _data);
    });
  }
  switch (data.rid) {
    case 13:
      if (data.day == 3 && data.tag == 1) {
        getRankingGlobal3().then(res => {
          const _res = jsonChange(res);
          _data.data = _res.result;
          commit(types.SET_RANKING_DATA, _data);
        });
      } else if (data.day == 7 && data.tag == 1) {
        getRankingGlobal7().then(res => {
          const _res = jsonChange(res);
          _data.data = _res.result;
          commit(types.SET_RANKING_DATA, _data);
        });
      } else {
        getRank();
      }
      break;
    case 168:
      if (data.day == 3 && data.tag == 1) {
        getRankingCn3().then(res => {
          const _res = jsonChange(res);
          _data.data = _res.result;
          commit(types.SET_RANKING_DATA, _data);
        });
      } else if (data.day == 7 && data.tag == 1) {
        getRankingCn7().then(res => {
          const _res = jsonChange(res);
          _data.data = _res.result;
          commit(types.SET_RANKING_DATA, _data);
        });
      } else {
        getRank();
      }
      break;
    default:
      //获取排行榜
      //设置当前排行榜数据
      getRank();
      break;
  }
};
//番剧更新时间表
export const setTimeline = function({ commit }, data) {
  const _data = {
    id: data.id,
    rid: data.rid
  };
  switch (data.rid) {
    case 13:
      getTimelineGlobal().then(res => {
        _data.data = res.result;
        commit(types.SET_TIMELINE_DATA, _data);
      });
      break;
    case 168:
      getTimelineCn().then(res => {
        _data.data = res.result;
        commit(types.SET_TIMELINE_DATA, _data);
      });
      break;
    default:
      break;
  }
};
