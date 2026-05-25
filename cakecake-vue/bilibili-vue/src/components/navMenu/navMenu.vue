<template>
  <div class="nav-menu" :class="{ 'nav-menu--solid-top': useSolidTopNav }">
    <div
      v-if="menuShow && isHomeRoute"
      class="blur-bg"
      :style="{ background: 'url(' + headBanner.pic + ')' }"
    ></div>
    <div class="nav-mask"></div>
    <div class="bili-wrapper">
      <div class="nav-con fl">
        <ul>
          <li
            class="nav-item"
            v-for="(item, index) in leftNav"
            :key="`leftNav_${index}`"
            :class="item.class"
          >
            <a class="t" :href="item.href">
              <i :class="item.icon" v-if="item.icon"></i>
              {{ item.name }}
            </a>
          </li>
        </ul>
      </div>
      <div class="up-load fr">
        <a
          v-if="minibiliUploadOpensModal"
          href="javascript:;"
          class="u-link"
          @click.prevent="onMbUploadNavClick"
          >投 稿</a
        >
        <router-link v-else class="u-link" :to="uploadNavTo">投 稿</router-link>
      </div>
      <div class="nav-con fr">
        <ul>
          <li
            class="nav-item profile-info"
            :class="{ on: signIn == 1 }"
            @mouseover="profileFadeIn"
            @mouseout="profileFadeOut"
          >
            <router-link
              v-if="signIn == 1 && isMinibiliMode && minibiliSpaceTo"
              class="t"
              :to="minibiliSpaceTo"
            >
              <div class="i-face">
                <img v-if="navFaceSrc" :src="navFaceSrc" class="face" />
                <img class="pendant" />
              </div>
            </router-link>
            <a
              class="t"
              v-else-if="signIn == 1"
              :href="profileSpaceHref"
              target="_blank"
            >
              <div class="i-face">
                <img v-if="navFaceSrc" :src="navFaceSrc" class="face" />
                <img class="pendant" />
              </div>
            </a>
            <a
              class="t"
              v-else
              @click="
                setLoginShow();
                setLoginTab(0);
              "
            >
              <div class="i-face">
                <img src="../../assets/akari.jpg" class="face" />
              </div>
            </a>
            <transition name="nav-trans">
              <div
                class="profile-m dd-bubble"
                v-if="signIn == 1"
                v-show="profileShow"
              >
                <div class="header-u-info" v-if="navProfileReady">
                  <div class="header-uname">
                    <b class="">{{ navDisplayName }}</b>
                  </div>
                  <div class="btns-profile clearfix">
                    <div class="coin fl">
                      <a
                        href="https://account.bilibili.com/site/coin"
                        target="_blank"
                        title="硬币"
                      >
                        <i class="bili-icon bi"></i>
                        <i class="bili-icon jia"></i>
                        <span class="num">{{ navCoinDisplay }}</span>
                        <span class="num-move">{{ navCoinRaw }}</span>
                        <span title="" class="num-tip">登录奖励</span>
                      </a>
                    </div>
                    <div class="currency fl">
                      <a
                        href="https://pay.bilibili.com/bb_balance.html"
                        target="_blank"
                        title="B币"
                      >
                        <i class="bili-icon"></i>
                        <span class="num">{{ navBcoinDisplay }}</span>
                      </a>
                    </div>
                    <div class="ver phone fr verified">
                      <a
                        href="https://passport.bilibili.com/site/site.html"
                        target="_blank"
                      >
                        <i class="bili-icon"></i>
                        <span class="tips">已绑定</span>
                      </a>
                    </div>
                    <div class="ver email fr verified">
                      <a
                        href="https://passport.bilibili.com/site/site.html"
                        target="_blank"
                      >
                        <i class="bili-icon"></i>
                        <span class="tips">已绑定</span>
                      </a>
                    </div>
                    <div class="link-to-bind-mobile"></div>
                  </div>
                  <div class="grade clearfix">
                    <span class="hd fl">等级</span>
                    <a
                      href="https://account.bilibili.com/site/record.html"
                      target="_blank"
                    >
                      <div class="bar fr">
                        <div class="lt" :class="level" aria-hidden="true">
                          <span class="lt-num">{{ navLevelDisplay }}</span>
                        </div>
                        <div
                          class="rate"
                          :style="{ width: navLevelFillPct + '%' }"
                        ></div>
                        <div class="num">
                          <div v-if="navLevelInfo">
                            {{ navLevelInfo.current_exp }}
                            <span>{{ "/" + navLevelInfo.next_exp }}</span>
                          </div>
                        </div>
                      </div>
                    </a>
                    <div class="desc-tips">
                      <span class="arrow-left"></span>
                      <div class="lv-row">
                        作为<strong>LV{{ navLevelDisplay }}</strong>，你可以：
                      </div>
                      <div>
                        <div
                          v-for="(line, idx) in navLevelPrivilegeLines"
                          :key="idx"
                        >
                          {{ idx + 1 }}、{{ line }}
                        </div>
                      </div>
                      <a
                        :href="userLevelHelpUrl"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="help-link"
                        >会员等级相关说明 &gt;</a
                      >
                    </div>
                  </div>
                </div>
                <div class="member-menu">
                  <ul class="clearfix">
                    <li>
                      <router-link
                        v-if="isMinibiliMode"
                        to="/minibili/account"
                        class="account"
                      >
                        <i class="bili-icon b-icon-p-account"></i>
                        个人中心
                      </router-link>
                      <a
                        v-else
                        href="https://account.bilibili.com/account/home"
                        target="_blank"
                        class="account"
                      >
                        <i class="bili-icon b-icon-p-account"></i>
                        个人中心
                      </a>
                    </li>
                    <li>
                      <router-link
                        v-if="isMinibiliMode"
                        :to="{ name: 'upload' }"
                        class="member"
                      >
                        <i class="bili-icon b-icon-p-member"></i>
                        投稿管理
                      </router-link>
                      <a
                        v-else
                        href="https://member.bilibili.com/v2#/home"
                        target="_blank"
                        class="member"
                      >
                        <i class="bili-icon b-icon-p-member"></i>
                        投稿管理
                      </a>
                    </li>
                    <li>
                      <a
                        href="https://pay.bilibili.com/paywallet-fe/bb_balance.html"
                        target="_blank"
                        class="wallet"
                      >
                        <i class="bili-icon b-icon-p-wallet"></i>
                        B币钱包
                      </a>
                    </li>
                    <li>
                      <a
                        href="https://link.bilibili.com/p/center/index#/user-center/my-info/operation"
                        target="_blank"
                        class="live"
                      >
                        <i class="bili-icon b-icon-p-live"></i>
                        直播中心
                      </a>
                    </li>
                    <li>
                      <a
                        href="https://show.bilibili.com/orderlist"
                        target="_blank"
                        class="bml"
                      >
                        <i class="bili-icon b-icon-p-ticket"></i>
                        订单中心
                      </a>
                    </li>
                    <li></li>
                  </ul>
                </div>
                <div class="member-bottom">
                  <a href="#" class="logout" @click="signOut()">退出</a>
                </div>
              </div>
            </transition>
            <div class="i_menu i_menu_login" v-if="signIn == 0">
              <p class="tip">
                登录后你可以：
              </p>
              <div class="img">
                <img src="../../assets/danmu.png" />
                <img src="../../assets/danmu.png" />
              </div>
              <a
                class="login-btn"
                @click="
                  setLoginShow();
                  setLoginTab(0);
                "
                >登录</a
              >
              <p class="reg">
                首次使用？<a
                  @click="
                    setLoginShow();
                    setLoginTab(1);
                  "
                  >点我去注册</a
                >
              </p>
            </div>
          </li>
          <template v-if="signIn == 1">
            <li class="nav-item">
              <a
                href="https://account.bilibili.com/big"
                target="_blank"
                rel="noopener noreferrer"
                class="t"
              >
                大会员
              </a>
            </li>
            <li
              class="nav-item"
              @mouseover="messageFadeIn"
              @mouseout="messageFadeOut"
            >
              <router-link
                v-if="isMinibiliMode"
                to="/minibili/messages?cat=my_message"
                class="t"
                title="消息"
                @click="messageShow = false"
              >
                <div v-if="messageUnreadTotalLabel" class="num">
                  {{ messageUnreadTotalLabel }}
                </div>
                消息
              </router-link>
              <a
                v-else
                href="#"
                target="_blank"
                title="消息"
                class="t"
              >
                <div class="num">
                  1
                </div>
                消息
              </a>
              <transition name="nav-trans">
                <div class="im-list-box" v-show="messageShow">
                  <template v-for="item in messageNavItems" :key="item.cat">
                    <router-link
                      v-if="isMinibiliMode"
                      class="im-list"
                      :to="`/minibili/messages?cat=${item.cat}`"
                      @click="messageShow = false"
                    >
                      {{ item.label }}
                      <div
                        v-if="formatMessageUnreadBadge(msgUnread[item.cat])"
                        class="im-notify im-number im-center"
                      >
                        {{ formatMessageUnreadBadge(msgUnread[item.cat]) }}
                      </div>
                    </router-link>
                    <a
                      v-else
                      class="im-list"
                      target="_blank"
                      href="#"
                    >
                      {{ item.label }}
                    </a>
                  </template>
                </div>
              </transition>
            </li>
            <li class="nav-item">
              <router-link
                v-if="isMinibiliMode && minibiliDynamicsTo"
                class="t"
                :to="minibiliDynamicsTo"
              >
                动态
              </router-link>
              <a v-else href="#" target="_blank" class="t">
                动态
              </a>
            </li>
            <li class="nav-item">
              <router-link
                v-if="isMinibiliMode"
                class="t"
                to="/minibili/watch-later"
              >
                稍后再看
              </router-link>
              <a v-else href="#" target="_blank" class="t">
                稍后再看
              </a>
            </li>
            <li class="nav-item">
              <router-link
                v-if="isMinibiliMode && minibiliCollectTo"
                class="t"
                :to="minibiliCollectTo"
              >
                收藏夹
              </router-link>
              <a v-else href="#" target="_blank" class="t">
                收藏夹
              </a>
            </li>
          </template>
          <li class="nav-item">
            <router-link
              v-if="isMinibiliMode"
              class="t"
              :to="minibiliHistoryTo"
            >
              历史
            </router-link>
            <a
              v-else
              href="//www.bilibili.com/account/history"
              target="_blank"
              class="t"
            >
              历史
            </a>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script>
import { createNamespacedHelpers } from "vuex";
import { setMinibiliPostLoginRedirect } from "@/utils/authTokens";
import akariFace from "../../assets/akari.jpg";
import {
  minibiliUploadOpensLoginModal,
  resolveMinibiliUploadNavTo
} from "@/utils/minibiliUploadNav";
import {
  minibiliDynamicsRoute,
  minibiliViewHistoryRoute,
  minibiliUserSpaceCollectRoute,
  minibiliUserSpaceRoute,
  shouldShowMinibiliCompactHeader
} from "@/utils/minibiliRoutes";
import { formatCoinBalance, coinBalanceNumber } from "@/utils/coinBalance";
import {
  USER_LEVEL_HELP_URL,
  levelFillPct,
  levelPrivilegeLines
} from "@/utils/userLevel";
import {
  MESSAGE_CATEGORIES,
  formatMessageUnreadBadge,
  sumMessageUnread
} from "@/utils/messageCategories";
import {
  refreshMessageUnread,
  subscribeMessageUnread
} from "@/utils/messageUnread";

const { mapState, mapMutations, mapActions } = createNamespacedHelpers("login");

export default {
  props: {
    leftNav: {
      default: []
    },
    headBanner: {
      default: []
    },
    menuShow: {
      default: []
    }
  },
  data() {
    return {
      profileShow: false, //个人信息默认隐藏
      messageShow: false, //消息通知默认隐藏
      userLevelHelpUrl: USER_LEVEL_HELP_URL,
      msgUnread: {},
      _msgUnreadUnsub: null
    };
  },
  computed: {
    /** 仅首页保留顶栏毛玻璃透明叠在头图上；其余路由顶栏纯白 */
    isHomeRoute() {
      return this.$route.name === "home";
    },
    /** 消息中心 / 个人空间等：纯白顶栏，无毛玻璃头图 */
    useSolidTopNav() {
      return !this.isHomeRoute || shouldShowMinibiliCompactHeader(this.$route);
    },
    isMinibiliMode() {
      return (
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1"
      );
    },
    // 使用对象展开运算符将此对象混入到外部对象中
    ...mapState({
      //命名空间获取state
      signIn: state => state.signIn, //登录状态获取
      proInfo: state => state.proInfo //个人信息获取
    }),
    /** 顶栏用：兼容 proInfo 初始为 [] */
    navProfileRecord() {
      const p = this.proInfo;
      return p && typeof p === "object" && !Array.isArray(p) ? p : null;
    },
    navProfileReady() {
      return !!this.navProfileRecord;
    },
    navFaceSrc() {
      const p = this.navProfileRecord;
      if (p && p.face) {
        return p.face;
      }
      return this.signIn == 1 ? akariFace : "";
    },
    navDisplayName() {
      const p = this.navProfileRecord;
      return (p && p.uname) || "";
    },
    minibiliSpaceTo() {
      if (!this.isMinibiliMode || this.signIn != 1) {
        return null;
      }
      const p = this.navProfileRecord;
      if (!p || p.mid == null) {
        return null;
      }
      return minibiliUserSpaceRoute(p.mid);
    },
    minibiliCollectTo() {
      if (!this.isMinibiliMode || this.signIn != 1) {
        return null;
      }
      const p = this.navProfileRecord;
      if (!p || p.mid == null) {
        return null;
      }
      return minibiliUserSpaceCollectRoute(p.mid);
    },
    minibiliDynamicsTo() {
      if (!this.isMinibiliMode || this.signIn != 1) {
        return null;
      }
      return minibiliDynamicsRoute();
    },
    minibiliHistoryTo() {
      if (!this.isMinibiliMode) {
        return "/";
      }
      return minibiliViewHistoryRoute();
    },
    profileSpaceHref() {
      const p = this.navProfileRecord;
      if (p && p.mid != null) {
        return `https://space.bilibili.com/${p.mid}`;
      }
      return "/";
    },
    messageNavItems() {
      return MESSAGE_CATEGORIES;
    },
    messageUnreadTotal() {
      return sumMessageUnread(this.msgUnread);
    },
    messageUnreadTotalLabel() {
      return formatMessageUnreadBadge(this.messageUnreadTotal);
    },
    navCoinDisplay() {
      const p = this.navProfileRecord;
      if (p && typeof p.money === "number") {
        return formatCoinBalance(p.money);
      }
      return "0";
    },
    navCoinRaw() {
      const p = this.navProfileRecord;
      if (p && typeof p.money === "number") {
        return coinBalanceNumber(p.money);
      }
      return 0;
    },
    navBcoinDisplay() {
      const p = this.navProfileRecord;
      const w = p && p.wallet;
      if (w && typeof w.bcoin_balance === "number") {
        return w.bcoin_balance;
      }
      return 0;
    },
    navMoralPct() {
      const p = this.navProfileRecord;
      if (p && typeof p.moral === "number") {
        return p.moral;
      }
      return 0;
    },
    navLevelInfo() {
      const p = this.navProfileRecord;
      return p && p.level_info ? p.level_info : null;
    },
    navLevelDisplay() {
      const li = this.navLevelInfo;
      if (li && li.current_level != null) {
        const n = Number(li.current_level);
        if (Number.isFinite(n) && n >= 1) {
          return Math.min(6, Math.max(1, Math.floor(n)));
        }
      }
      return 1;
    },
    navLevelFillPct() {
      return levelFillPct(this.navLevelInfo);
    },
    navLevelPrivilegeLines() {
      return levelPrivilegeLines(this.navLevelDisplay);
    },
    //个人等级
    level() {
      const li = this.navLevelInfo;
      if (li && li.current_level != null) {
        return "lv" + li.current_level;
      }
      return "";
    },
    /** 最右侧「投稿」：Mini-Bili 未登录 → 弹主站登录窗 */
    minibiliUploadOpensModal() {
      void this.$route.fullPath;
      return minibiliUploadOpensLoginModal();
    },
    /** 已登录 Mini-Bili 或非 Mini：router-link 目标 */
    uploadNavTo() {
      void this.$route.fullPath;
      return resolveMinibiliUploadNavTo();
    }
  },
  methods: {
    onMbUploadNavClick() {
      setMinibiliPostLoginRedirect("/upload");
      this.$store.commit("login/SET_LOGIN_TAB", 0);
      this.$store.commit("login/OPEN_LOGIN_MODAL");
    },
    ...mapMutations({
      setLoginShow: "SET_LOGIN_SHOW", //登录弹窗显示隐藏
      setLoginTab: "SET_LOGIN_TAB" //注册登录tab状态
    }),
    ...mapActions([
      "setSignIn", //登录
      "setUserInfo", //获取个人信息
      "refreshMinibiliMe",
      "signOut" //退出登录
    ]),
    //个人信息显示隐藏
    profileFadeIn() {
      this.profileShow = true;
    },
    profileFadeOut() {
      this.profileShow = false;
    },
    formatMessageUnreadBadge,
    onMsgUnreadSummary(summary) {
      this.msgUnread = summary || {};
    },
    //消息通知显示隐藏
    messageFadeIn() {
      this.messageShow = true;
      if (this.isMinibiliMode && this.signIn == 1) {
        void refreshMessageUnread();
      }
    },
    messageFadeOut() {
      this.messageShow = false;
    }
  },
  watch: {
    signIn(v) {
      if (this.isMinibiliMode && v == 1) {
        void refreshMessageUnread();
      } else if (v != 1) {
        this.msgUnread = {};
      }
    },
    $route() {
      if (this.isMinibiliMode && this.signIn == 1) {
        void refreshMessageUnread();
      }
    }
  },
  mounted() {
    this._msgUnreadUnsub = subscribeMessageUnread(this.onMsgUnreadSummary);
    if (this.isMinibiliMode && this.signIn == 1) {
      void refreshMessageUnread();
    }
  },
  beforeUnmount() {
    if (this._msgUnreadUnsub) {
      this._msgUnreadUnsub();
      this._msgUnreadUnsub = null;
    }
  },
  async created() {
    const login = localStorage.getItem("signIn"); //读取缓存登录状态
    if (!login) {
      //无状态即未登录状态，修改state值
      this.setSignIn({
        signIn: localStorage.setItem("signIn", 0)
      });
    } else {
      //已登录状态
      //读取缓存状态
      this.setSignIn({
        signIn: localStorage.getItem("signIn")
      });
      if (this.isMinibiliMode && String(login) === "1") {
        await this.refreshMinibiliMe().catch(() => {});
      }
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style lang="scss" scoped>
@import "../../style/mixin";

//菜单偏移，透明度过渡效果
.nav-trans-enter,
.nav-trans-leave-to {
  transform: translateY(5px);
  opacity: 0;
}
.nav-trans-enter-to,
.nav-trans-leave {
  transform: translateY(0px);
  opacity: 1;
}
.nav-trans-enter-active,
.nav-trans-leave-active {
  transition: all 0.3s ease;
}
//app-header
.app-header {
  position: relative;
  background: $white;
  .bili-wrapper {
    margin: 0 auto;
    width: 1160px;
  }
  .nav-menu {
    position: relative;
    z-index: 200;
    height: 42px;
    color: $black;
    .blur-bg {
      position: absolute;
      top: 0;
      left: 0;
      @include wh(100%, 100%);
      background-position: center -10px;
      background-repeat: no-repeat;
      -webkit-filter: blur(4px);
      filter: blur(4px);
    }
    .nav-mask {
      position: absolute;
      top: 0;
      left: 0;
      @include wh(100%, 100%);
      background-color: hsla(0, 0%, 100%, 0.4);
      -webkit-box-shadow: rgba(0, 0, 0, 0.1) 0 1px 2px;
      box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
      pointer-events: none;
    }
    &.nav-menu--solid-top {
      .blur-bg {
        display: none;
      }
      .nav-mask {
        background-color: #fff;
        -webkit-box-shadow: none;
        box-shadow: none;
      }
      .bili-wrapper .nav-con .nav-item {
        background-color: #fff !important;
        &:hover {
          background-color: #fff !important;
        }
      }
    }
    .bili-wrapper {
      position: relative;
      z-index: 1;
      .nav-con {
        .nav-item {
          float: left;
          text-align: center;
          line-height: 42px;
          height: 42px;
          position: relative;
          background-color: hsla(0, 0%, 100%, 0);
          @include transition(0.3s);
          &:hover {
            background-color: hsla(0, 0%, 100%, 0.3);
          }
          a {
            &.t {
              color: $black;
              height: 100%;
              display: block;
              padding: 0 11px;
            }
          }
          &.home {
            margin-left: -10px;
            padding-left: 12px;
            a {
              padding-left: 20px;
            }
            i {
              position: absolute;
              @include wh(17px, 14px);
              left: 10px;
              top: 12px;
              background-position: -919px -88px;
            }
          }
          &.mobile {
            i {
              display: inline-block;
              vertical-align: middle;
              background-position: -1367px -1175px;
              @include wh(21px, 21px);
            }
          }
        }
      }
    }
    .nav-con {
      .nav-item {
        .t {
          .num {
            height: 12px;
            line-height: 12px;
            background-color: $pink;
            position: absolute;
            padding: 1px 2px;
            @include sc(12px, $white);
            @include borderRadius(10px);
            top: 1px;
            right: -4px;
            min-width: 16px;
            z-index: 30;
            text-align: center;
          }
        }
        .im-list-box {
          width: 110px;
          position: absolute;
          top: 100%;
          left: calc(50% - 55px);
          background: $white;
          box-shadow: rgba(0, 0, 0, 0.16) 0px 2px 4px;
          border-radius: 0 0 4px 4px;
          overflow: hidden;
          transition: all 300ms;
        }
        .reg {
          a {
            display: initial;
            cursor: pointer;
            padding: 0;
            color: $blue;
          }
        }
      }
    }
    .dd-bubble {
      position: absolute;
      z-index: 1;
    }
    .up-load {
      position: relative;
      @include wh(58px, 42px);
      .u-link {
        position: relative;
        display: block;
        @include wh(100%, 48px);
        @include sc(14px, $white);
        line-height: 42px;
        text-align: center;
        z-index: 0;
        &:after {
          position: absolute;
          left: 0;
          content: "";
          @include wh(100%, 100%);
          background: $pink;
          border-bottom-left-radius: 5px;
          border-bottom-right-radius: 5px;
          z-index: -1;
        }
      }
    }
  }
  //右侧
  .profile-info {
    width: 58px;
    .i-face {
      position: absolute;
      z-index: 20;
      @include wh(36px, 36px);
      left: 11px;
      top: 0;
      @include transition(0.3s);
      .face {
        border: 0 solid $white;
        @include wh(100%, 100%);
        @include borderRadius(50%);
      }
      .pendant {
        position: absolute;
        @include wh(84px, 84px);
        left: -11px;
        bottom: -3px;
        visibility: hidden;
        -webkit-transition-delay: 0s;
        -o-transition-delay: 0s;
        transition-delay: 0s;
      }
    }
    &.on {
      &:hover {
        .i-face {
          left: -4px;
          top: 15px;
          @include wh(64px, 64px);
          .face {
            border: 2px solid $white;
          }
        }
      }
    }
    &:hover {
      .i_menu_login {
        display: block;
        opacity: 1;
        transition: all 0.3s;
      }
    }
  }
  //个人信息开始
  .profile-m {
    left: 50%;
    margin-left: -130px;
    width: 260px;
    padding: 50px 0 0;
    top: 42px;
    background: $white;
    -webkit-box-shadow: rgba(0, 0, 0, 0.16) 0 2px 4px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.16);
    border-radius: 0 0 4px 4px;
    line-height: normal;
    .header-u-info {
      a {
        color: $black;
      }
    }
    .header-uname {
      padding-bottom: 15px;
      b {
        display: block;
        margin-bottom: 8px;
        font-weight: bold;
      }
    }
    .btns-profile {
      position: relative;
      margin: 0 20px;
      height: 18px;
      .bili-icon {
        display: inline-block;
        @include wh(18px, 18px);
        vertical-align: middle;
        background-repeat: no-repeat;
      }
      .coin {
        .bi {
          background-position: -343px -471px;
          margin-right: 2px;
          position: relative;
          z-index: 2;
        }
        .jia {
          z-index: 1;
          left: 0;
          position: absolute;
          top: 0;
          @include wh(18px, 18px);
          background-position: -279px -1495px;
        }
      }
      .num {
        vertical-align: middle;
        display: inline-block;
        @include transition(2s);
      }
      .num-move {
        position: absolute;
        @include transition(2s);
        left: 23px;
        top: -10px;
        opacity: 0;
        line-height: 14px;
      }
      .num-tip {
        color: #2cc06f;
        position: absolute;
        @include transition(0.3s);
        left: 60px;
        top: -18px;
        opacity: 0;
        background: $white;
        padding: 3px 5px;
        z-index: 10;
      }
      .currency {
        position: absolute;
        left: 58px;
        z-index: 0;
        .bili-icon {
          background-position: -407px -471px;
          margin: 0 5px 0 8px;
        }
      }
      .ver {
        position: relative;
        a {
          display: block;
        }
        .tips {
          display: none;
          padding: 0 6px;
          height: 20px;
          line-height: 20px;
          border: 1px solid #ccc;
          @include borderRadius(4px);
          position: absolute;
          right: 30px;
          top: -2px;
          white-space: nowrap;
          background-color: $white;
          color: $black;
          z-index: 10;
          &:after {
            content: "";
            position: absolute;
            @include wh(8px, 8px);
            background: url(../../assets/horn.png);
            right: -8px;
            top: 6px;
          }
        }
        &:hover {
          .tips {
            display: block;
          }
        }
      }
      .phone {
        &.verified {
          .bili-icon {
            background-position: -343px -599px;
          }
        }
      }
      .email {
        margin-right: 10px;
        &.verified {
          .bili-icon {
            background-position: -343px -534px;
          }
        }
      }
    }
    .grade {
      position: relative;
      margin: 24px 0 30px;
      height: 16px;
      padding: 0 20px;
      .bar {
        position: relative;
        top: 6px;
        @include wh(170px, 8px);
        background: #eee;
        .lt {
          @include wh(18px, 18px);
          @include borderRadius(9px);
          position: absolute;
          left: -17px;
          top: -6px;
          display: flex;
          align-items: center;
          justify-content: center;
          background-color: #f3cb85;
          background-image: none;
          z-index: 1;
          &.lv1 {
            background-color: #94def5;
          }
          &.lv2 {
            background-color: #94def5;
          }
          &.lv3 {
            background-color: #6dc781;
          }
          &.lv4 {
            background-color: #f3cb85;
          }
          &.lv5 {
            background-color: #ff9f3f;
          }
          &.lv6 {
            background-color: #ff7f24;
          }
          .lt-num {
            display: block;
            font-size: 12px;
            font-weight: 700;
            line-height: 1;
            font-family: Arial, "Helvetica Neue", Helvetica, sans-serif;
            color: #fff;
            -webkit-text-stroke: 1.5px #000;
            paint-order: stroke fill;
            text-shadow:
              1px 0 0 #000,
              -1px 0 0 #000,
              0 1px 0 #000,
              0 -1px 0 #000;
          }
        }
        .rate {
          @include wh(20%, 8px);
          background-color: #f3cb85;
        }
        .num {
          position: absolute;
          right: 0;
          bottom: -18px;
          span {
            color: #ccc;
          }
        }
      }
      .desc-tips {
        display: none;
        padding: 15px 15px 15px 20px;
        position: absolute;
        top: -16px;
        left: 260px;
        @include borderRadius(2px);
        background-color: $white;
        z-index: 100;
        width: 220px;
        line-height: 24px;
        word-break: break-word;
        word-wrap: break-word;
        min-height: 65px;
        color: #676b73;
        -webkit-box-shadow: 0 0 2px 0 rgba(0, 0, 0, 0.25);
        box-shadow: 0 0 2px 0 rgba(0, 0, 0, 0.25);
        text-align: left;
        .arrow-left {
          position: absolute;
          display: inline-block;
          top: 16px;
          left: -10px;
          @include wh(10px, 20px);
          background: transparent url(../../assets/level.png) -182px -224px
            no-repeat;
        }
        .lv-row {
          margin-bottom: 10px;
          strong {
            @include sc(14px, $black);
            padding: 0 3px;
          }
        }
        .help-link {
          margin-top: 15px;
          float: right;
          color: $blue;
        }
      }
      &:hover {
        .desc-tips {
          display: block;
        }
      }
    }
    .member-menu {
      border-top: 1px solid #e5e9ef;
      padding: 10px 20px 40px;
      overflow: hidden;
      ul {
        width: 240px;
        clear: both;
        zoom: 1;
      }
      li {
        float: left;
        width: 100px;
        margin-right: 20px;
        position: relative;
        a {
          white-space: nowrap;
          color: $black;
          text-align: left;
          margin: 0 auto;
          display: block;
          padding: 5px 0;
          line-height: 16px;
          &:hover {
            color: $blue;
            .bili-icon {
              &.b-icon-p-account {
                background-position: -536px -407px;
              }
              &.b-icon-p-member {
                background-position: -601px -1046px;
              }
              &.b-icon-p-wallet {
                background-position: -536px -472px;
              }
              &.b-icon-p-live {
                background-position: -537px -855px;
              }
              &.b-icon-p-ticket {
                background-position: -535px -2075px;
              }
            }
          }
          .bili-icon {
            @include wh(16px, 16px);
            margin-right: 10px;
            vertical-align: top;
            &.b-icon-p-account {
              background-position: -472px -407px;
            }
            &.b-icon-p-member {
              background-position: -536px -1046px;
            }
            &.b-icon-p-wallet {
              background-position: -472px -472px;
            }
            &.b-icon-p-live {
              background-position: -473px -855px;
            }
            &.b-icon-p-ticket {
              @include wh(18px, 15px);
              background-position: -471px -2075px;
            }
          }
        }
      }
    }
    .member-bottom {
      position: absolute;
      bottom: 0;
      left: 0;
      @include wh(100%, 30px);
      line-height: 30px;
      background-color: #f4f5f7;
      border-radius: 0 0 4px 4px;
      .logout {
        float: right;
        padding-right: 20px;
        color: $black;
      }
    }
  }
  //大会员开始
  &.vip-m {
    width: 260px;
    margin-left: -107px;
    position: absolute;
    border-radius: 0 0 4px 4px;
    background-color: $white;
    -webkit-box-shadow: rgba(0, 0, 0, 0.16) 0 2px 4px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.16);
    border: 1px solid #e5e9ef;
    text-align: left;
    z-index: 7000;
  }
  .bubble-traditional {
    padding: 14px;
    .recommand {
      .title {
        @include sc(14px, #212121);
        margin: 5px 0 12px;
        font-weight: 900;
        .more {
          float: right;
          -webkit-box-sizing: border-box;
          box-sizing: border-box;
          border: 1px solid $border_color;
          font-weight: 400;
          text-align: center;
          @include borderRadius(4px);
          @include wh(52px, 22px);
          @include sc(12px, #6d757a);
          line-height: 22px;
          -webkit-transition: background 0.2s;
          -o-transition: background 0.2s;
          transition: background 0.2s;
        }
      }
      .bubble-col {
        display: flex;
        margin-bottom: 7px;
        .item {
          flex: 1;
          .pic {
            display: inline-block;
          }
          .recommand-link {
            display: block;
            margin-top: 10px;
            @include sc(12px, $black);
            text-align: left;
            line-height: 18px;
            height: 36px;
            overflow: hidden;
            -o-text-overflow: ellipsis;
            text-overflow: ellipsis;
            -webkit-line-clamp: 2;
            display: -webkit-box;
            -webkit-box-orient: vertical;
            &:hover {
              color: #fb7299;
            }
          }
        }
        &.bubble-col-3 {
          img {
            @include wh(72px, 94px);
            @include borderRadius(4px);
            background: #ccc;
          }
        }
      }
    }
  }
  .b-icon {
    &.b-icon-arrow-r {
      background-position: -478px -218px;
      @include wh(6px, 12px);
      margin: -2px 0 0 5px;
    }
  }
  img {
    border: none;
    vertical-align: middle;
  }
  .i_menu_login {
    opacity: 0;
    display: none;
    background: $white;
    left: 50%;
    margin-left: -130px;
    padding-bottom: 0;
    padding-top: 50px;
    border-top: none;
    width: 320px;
    margin-left: -160px;
    padding: 12px;
    text-align: left;
    line-height: normal;
    border: 1px solid #e5e9ef;
    @include transition(0.3s);
    .tip {
      @include sc(14px, #666);
    }
    .img {
      @include wh(320px, 200px);
      margin: 12px 0;
      overflow: hidden;
      position: relative;
      background: url(../../assets/danmu_bg.png) no-repeat 50%;
      img {
        &:first-child {
          @include wh(320px, 200px);
          position: absolute;
          left: 0;
          top: 0;
          animation: one 5s linear infinite;
        }
        &:last-child {
          @include wh(320px, 200px);
          position: absolute;
          left: 320px;
          top: 0;
          animation: two 5s linear infinite;
        }
      }
    }
    .reg {
      margin-top: 8px;
      text-align: center;
      @include sc(12px, #282828);
    }
  }
}
//动态开始
.im-list {
  display: block;
  text-align: center;
  position: relative;
  line-height: 42px;
  height: 42px;
  &:hover {
    color: $blue;
    background-color: #e5e9ef;
  }
}
.im-notify {
  position: absolute;
  background-color: #fb7299;
  &.im-number {
    height: 14px;
    line-height: 15px;
    border-radius: 10px;
    padding: 1px 3px;
    @include sc(12px, $white);
    min-width: 20px;
    text-align: center;
    &.im-center {
      top: 13px;
      left: 80px;
    }
  }
}
@keyframes one {
  0% {
    left: 0;
  }

  100% {
    left: -320px;
  }
}
@keyframes two {
  0% {
    left: 320px;
  }

  100% {
    left: 0px;
  }
}
.app-header {
  .nav-menu {
    .nav-con {
      .nav-item {
        .login-btn {
          display: block;
          height: 43px;
          line-height: 43px;
          text-align: center;
          background: $blue;
          @include borderRadius(4px);
          @include sc(14px, $white);
          cursor: pointer;
        }
      }
    }
  }
}
</style>
