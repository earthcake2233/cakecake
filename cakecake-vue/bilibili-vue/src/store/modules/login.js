import { getUserInfo, getVipInfo } from "../../api";
import { mbGetMe } from "../../api/minibili";
import defaultAvatar from "../../assets/akari.jpg";
import {
  clearMinibiliPostLoginRedirect,
  clearTokens,
  getAccessToken,
  getRefreshToken
} from "../../utils/authTokens";
import { refreshMinibiliAccessToken } from "../../utils/minibiliTokenRefresh";
import { coinBalanceNumber } from "../../utils/coinBalance";
import { resolveUserAvatarUrl } from "../../utils/imageCacheBust";
import { levelInfoFromExperience } from "../../utils/userLevel";

const minibiliEnv =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

/** 将 /users/me 同步为顶栏等沿用的 proInfo 形状（等级/硬币占位至后端提供字段） */
function mapMinibiliMeToProInfo(me, avatarCacheBust = 0) {
  if (!me || typeof me !== "object") {
    return [];
  }
  const nick = String(me.nickname || "").trim();
  const uname = String(me.username || "").trim();
  const rawAvatar = me.avatar_url && String(me.avatar_url).trim();
  const face = rawAvatar
    ? resolveUserAvatarUrl(rawAvatar, avatarCacheBust) || defaultAvatar
    : defaultAvatar;
  const mid =
    me.user_id != null && !Number.isNaN(Number(me.user_id))
      ? Number(me.user_id)
      : 0;
  return {
    isLogin: true,
    face,
    uname: nick || uname || "用户",
    mid,
    money: coinBalanceNumber(me.coin_balance),
    moral: 0,
    level_info:
      me.level_info && typeof me.level_info.current_level === "number"
        ? me.level_info
        : levelInfoFromExperience(
            typeof me.experience === "number" ? me.experience : 0
          ),
    mobile_verified: 0,
    email_verified: 0,
    officialVerify: { type: -1, desc: "" },
    pendant: { pid: 0, name: "", image: "", expire: 0 },
    wallet: {
      mid,
      bcoin_balance: 0,
      coupon_balance: 0,
      coupon_due_time: 0
    }
  };
}

const state = {
  loginShow: false, //登录弹窗，默认隐藏
  userName: "", //用户名
  password: "", //密码
  signIn: "", //0为未登录，1为已登录
  proInfo: [], //个人信息
  /** Mini-Bili GET /users/me 原始数据（个人中心表单等） */
  minibiliMe: null,
  /** 头像 OSS 路径固定，换图后需 bust 顶栏/个人中心缓存 */
  avatarCacheBust: 0,
  topInfo: { picAndWords: [] }, //会员推荐（与接口 data 结构一致，避免模板 filter 报错）
  nowindex: 0 //登录框tab
};

const getters = {};

const mutations = {
  //登录弹窗显示隐藏
  SET_LOGIN_SHOW: state => {
    state.loginShow = state.loginShow ? false : true;
  },
  /** 强制打开（不 toggle），供「投稿」等入口在已关闭时直接弹出 */
  OPEN_LOGIN_MODAL: state => {
    state.loginShow = true;
  },
  /** 登录成功等场景：明确关闭，避免误用 SET_LOGIN_SHOW toggle */
  CLOSE_LOGIN_MODAL: state => {
    state.loginShow = false;
  },
  SET_LOGIN_TAB: (state, data) => {
    state.nowindex = data;
  },
  //登录状态
  SET_SIGNIN: (state, data) => {
    state.signIn = data.signIn;
  },
  //个人信息
  SET_USER_INFO: (state, data) => {
    state.proInfo = data.proInfo;
  },

  //会员推荐信息
  SET_VIP_INFO: (state, data) => {
    const t = data.topInfo;
    state.topInfo =
      t && typeof t === "object" && Array.isArray(t.picAndWords)
        ? t
        : { picAndWords: [] };
  },

  //用户名
  SET_USERNAME: (state, data) => {
    state.userName = data;
  },

  //用户密码
  SET_PASSWORD: (state, data) => {
    state.password = data;
  },

  /** Mini-Bili：写入 users/me，并同步顶栏用的 proInfo */
  SYNC_MINIBILI_ME(state, me) {
    state.minibiliMe = me == null ? null : me;
    state.proInfo = mapMinibiliMeToProInfo(me, state.avatarCacheBust);
  },
  BUMP_AVATAR_BUST(state) {
    state.avatarCacheBust = Date.now();
    if (state.minibiliMe) {
      state.proInfo = mapMinibiliMeToProInfo(
        state.minibiliMe,
        state.avatarCacheBust
      );
    }
  }
};

const actions = {
  setSignIn({ commit }, signin) {
    commit("SET_SIGNIN", signin);
  },
  setUserInfo({ commit }) {
    getUserInfo().then(res => {
      commit("SET_USER_INFO", {
        proInfo: res.data //state传入个人信息
      });
    });
  },
  setVipInfo({ commit }) {
    getVipInfo().then(res => {
      commit("SET_VIP_INFO", {
        topInfo: res.data //state传入大会员推荐信息
      });
    });
  },
  /** Mini-Bili：拉取当前用户并写入 state（顶栏 + 个人中心共用） */
  async refreshMinibiliMe({ commit }) {
    if (!minibiliEnv) {
      return;
    }
    if (!getAccessToken()) {
      if (getRefreshToken()) {
        const ok = await refreshMinibiliAccessToken();
        if (!ok) {
          commit("SYNC_MINIBILI_ME", null);
          localStorage.setItem("signIn", "0");
          commit("SET_SIGNIN", { signIn: "0" });
          return;
        }
      } else {
        commit("SYNC_MINIBILI_ME", null);
        /** 顶栏 signIn=1 但 JWT 已丢失时，与未登录态对齐，避免「假登录」壳子 */
        localStorage.setItem("signIn", "0");
        commit("SET_SIGNIN", { signIn: "0" });
        return;
      }
    }
    try {
      const me = await mbGetMe();
      commit("SYNC_MINIBILI_ME", me);
    } catch {
      /* 保留上次成功数据，避免网络抖动清空顶栏与个人中心 */
    }
  },
  //退出登录（含 Mini-Bili：必须清 JWT，否则路由守卫仍放行创作中心）
  signOut({ commit }) {
    if (minibiliEnv) {
      clearTokens();
      clearMinibiliPostLoginRedirect();
    }
    localStorage.setItem("signIn", "0");
    commit("SET_SIGNIN", { signIn: "0" });
    commit("SYNC_MINIBILI_ME", null);
    commit("SET_USER_INFO", {
      proInfo: []
    });
    commit("SET_VIP_INFO", {
      topInfo: { picAndWords: [] }
    });
    window.location.reload();
  }
};

export default {
  namespaced: true, //注册login空间模块
  state,
  getters,
  actions,
  mutations
};
