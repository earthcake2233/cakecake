<template>
  <div class="creator-page">
    <header class="creator-header">
      <div class="creator-header-inner">
        <div class="creator-header-left">
          <span class="creator-logo">
            <img
              class="creator-logo-img"
              src="@/assets/cakelogo.png"
              alt="cakecake"
            />
            <span class="creator-logo-t">创作中心</span>
          </span>
          <router-link
            class="creator-home-link"
            :to="{ name: 'home' }"
            title="返回主站"
          >
            <span
              class="creator-home-ico creator-home-ico--member"
              aria-hidden="true"
            />
            主站
          </router-link>
        </div>
        <div class="creator-header-right">
          <div
            class="creator-msg"
            @mouseenter="onMessageMouseEnter"
            @mouseleave="messageShow = false"
          >
            <router-link
              v-if="minibiliEnv"
              :to="creatorMessageTo('my_message')"
              class="creator-msg-trigger"
              title="消息"
              aria-label="消息"
              @click="messageShow = false"
            >
              <svg
                class="creator-msg-env"
                viewBox="0 0 24 24"
                width="24"
                height="24"
                aria-hidden="true"
              >
                <path
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M4 6h16v12H4V6z"
                />
                <path
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M4 6l8 6 8-6"
                />
              </svg>
            </router-link>
            <a
              v-else
              href="javascript:;"
              class="creator-msg-trigger"
              title="消息"
              aria-label="消息"
            >
              <svg
                class="creator-msg-env"
                viewBox="0 0 24 24"
                width="24"
                height="24"
                aria-hidden="true"
              >
                <path
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M4 6h16v12H4V6z"
                />
                <path
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M4 6l8 6 8-6"
                />
              </svg>
            </a>
            <transition name="creator-dd">
              <div v-show="messageShow" class="creator-msg-dropdown">
                <router-link
                  v-for="item in creatorMessageItems"
                  :key="item.cat"
                  :to="creatorMessageTo(item.cat)"
                  class="creator-msg-item"
                  @click="messageShow = false"
                >
                  {{ item.label }}
                  <span
                    v-if="messageBadge(item.cat)"
                    class="creator-msg-num"
                  >{{ messageBadge(item.cat) }}</span>
                </router-link>
              </div>
            </transition>
          </div>
          <span class="creator-up-days">
            <span class="creator-up-days-txt">成为UP主的第</span>
            <span class="creator-up-days-num">{{ creatorUpDays }}</span>
            <span class="creator-up-days-txt">天</span>
          </span>
          <router-link
            v-if="creatorSpaceTo"
            class="creator-avatar-wrap"
            title="个人空间"
            :to="creatorSpaceTo"
          >
            <img class="creator-avatar" :src="creatorAvatarSrc" alt="" />
          </router-link>
          <router-link
            v-else
            :to="{ name: 'home' }"
            class="creator-avatar-wrap"
            title="返回主站"
          >
            <img class="creator-avatar" :src="creatorAvatarSrc" alt="" />
          </router-link>
        </div>
      </div>
    </header>

    <div class="creator-body">
      <aside class="creator-side">
        <a
          v-if="minibiliUploadOpensModal"
          href="javascript:;"
          class="creator-side-upload-btn"
          @click.prevent="onMbUploadNavClick"
        >
          <img
            class="creator-side-upload-img"
            src="@/assets/update.png"
            alt=""
          />
          投稿
        </a>
        <router-link
          v-else
          :to="creatorUploadNavTo"
          class="creator-side-upload-btn"
        >
          <img
            class="creator-side-upload-img"
            src="@/assets/update.png"
            alt=""
          />
          投稿
        </router-link>
        <nav class="creator-nav">
          <router-link
            class="creator-nav-item"
            :class="{ 'is-active': $route.name === 'home' }"
            :to="{ name: 'home' }"
          >
            <span class="creator-nav-leading">
              <img class="creator-nav-icon" src="@/assets/home.png" alt="" />
              首页
            </span>
          </router-link>

          <button
            type="button"
            class="creator-nav-item creator-nav-parent"
            :class="{
              'is-open': navOpen.content,
              'is-active': contentManagementActive
            }"
            @click="toggleNav('content')"
          >
            <span class="creator-nav-leading">
              <img
                class="creator-nav-icon"
                src="@/assets/contentManagement.png"
                alt=""
              />
              <span class="creator-nav-label">内容管理</span>
            </span>
            <span class="creator-nav-arrow" />
          </button>
          <div
            class="creator-nav-drawer"
            :class="{ 'is-open': navOpen.content }"
          >
            <div class="creator-nav-drawer-inner">
              <div class="creator-nav-sub">
                <router-link
                  class="creator-nav-sub-item"
                  :class="{ 'is-current': manuscriptNavActive }"
                  :to="{ name: 'manuscript' }"
                >
                  稿件管理
                </router-link>
                <router-link
                  class="creator-nav-sub-item"
                  :class="{ 'is-current': $route.name === 'appeal' }"
                  :to="{ name: 'appeal' }"
                >
                  申诉管理
                </router-link>
              </div>
            </div>
          </div>

          <a href="javascript:;" class="creator-nav-item">
            <span class="creator-nav-leading">
              <img class="creator-nav-icon" src="@/assets/DC.png" alt="" />
              数据中心
            </span>
          </a>

          <button
            type="button"
            class="creator-nav-item creator-nav-parent"
            :class="{
              'is-open': navOpen.interact,
              'is-active': interactManagementActive
            }"
            @click="toggleNav('interact')"
          >
            <span class="creator-nav-leading">
              <img
                class="creator-nav-icon creator-nav-icon--interact"
                src="@/assets/interactionManagement.png"
                alt=""
              />
              <span class="creator-nav-label">互动管理</span>
            </span>
            <span class="creator-nav-arrow" />
          </button>
          <div
            class="creator-nav-drawer"
            :class="{ 'is-open': navOpen.interact }"
          >
            <div class="creator-nav-drawer-inner">
              <div class="creator-nav-sub">
                <router-link
                  class="creator-nav-sub-item"
                  :class="{ 'is-current': $route.name === 'creatorComments' }"
                  :to="{ name: 'creatorComments' }"
                >
                  评论管理
                </router-link>
                <router-link
                  class="creator-nav-sub-item"
                  :class="{ 'is-current': $route.name === 'creatorDanmakus' }"
                  :to="{ name: 'creatorDanmakus' }"
                >
                  弹幕管理
                </router-link>
              </div>
            </div>
          </div>
        </nav>
      </aside>

      <main class="creator-main">
        <slot />
      </main>
    </div>
  </div>
</template>

<script>
import { createNamespacedHelpers } from "vuex";
import akari from "@/assets/akari.jpg";
import { getUserId, setMinibiliPostLoginRedirect } from "@/utils/authTokens";
import {
  minibiliUploadOpensLoginModal,
  resolveMinibiliUploadNavTo
} from "@/utils/minibiliUploadNav";
import { minibiliUserSpaceRoute } from "@/utils/minibiliRoutes";
import {
  CREATOR_MESSAGE_DROPDOWN,
  formatMessageUnreadBadge
} from "@/utils/messageCategories";
import {
  refreshMessageUnread,
  subscribeMessageUnread
} from "@/utils/messageUnread";
import { clearStuckPageOverlays } from "@/utils/clearPageOverlays";

const { mapState } = createNamespacedHelpers("login");

const minibiliEnv =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

export default {
  name: "CreatorShell",
  computed: {
    ...mapState({
      minibiliMe: (s) => s.minibiliMe,
      proInfo: (s) => s.proInfo
    }),
    creatorAvatarSrc() {
      void this.minibiliMe;
      void this.proInfo;
      const m = this.minibiliMe;
      if (m && typeof m === "object") {
        const u = String(m.avatar_url || "").trim();
        if (u) return u;
      }
      const p = this.proInfo;
      if (p && typeof p === "object" && !Array.isArray(p) && p.face) {
        return p.face;
      }
      return akari;
    },
    creatorUpDays() {
      const m = this.minibiliMe;
      if (!m || typeof m !== "object") return 0;
      const n = Number(m.creator_up_days);
      return Number.isFinite(n) && n >= 0 ? n : 0;
    },
    minibiliUploadOpensModal() {
      void this.$route.fullPath;
      return minibiliUploadOpensLoginModal();
    },
    creatorUploadNavTo() {
      void this.$route.fullPath;
      return resolveMinibiliUploadNavTo();
    },
    creatorSpaceTo() {
      if (!minibiliEnv) {
        return null;
      }
      const uid = this.creatorMyUserId;
      return uid != null ? minibiliUserSpaceRoute(uid) : null;
    },
    creatorMyUserId() {
      const m = this.minibiliMe;
      if (m && m.user_id != null) {
        const n = Number(m.user_id);
        if (Number.isFinite(n) && n > 0) {
          return n;
        }
      }
      const fromStore = getUserId();
      if (fromStore != null && Number.isFinite(fromStore) && fromStore > 0) {
        return fromStore;
      }
      const p = this.proInfo;
      if (p && typeof p === "object" && !Array.isArray(p) && p.mid != null) {
        const n = Number(p.mid);
        if (Number.isFinite(n) && n > 0) {
          return n;
        }
      }
      return null;
    },
    manuscriptNavActive() {
      const n = this.$route.name;
      return n === "manuscript" || n === "videoEdit";
    },
    contentManagementActive() {
      const n = this.$route.name;
      return n === "manuscript" || n === "videoEdit" || n === "appeal";
    },
    interactManagementActive() {
      return (
        this.$route.name === "creatorComments" ||
        this.$route.name === "creatorDanmakus"
      );
    },
    creatorMessageItems() {
      return CREATOR_MESSAGE_DROPDOWN;
    },
    minibiliEnv() {
      return minibiliEnv;
    }
  },
  data() {
    return {
      messageShow: false,
      msgUnread: {},
      navOpen: {
        content: false,
        interact: false
      }
    };
  },
  created() {
    if (minibiliEnv) {
      void this.$store.dispatch("login/refreshMinibiliMe");
    }
  },
  mounted() {
    clearStuckPageOverlays();
    this._msgUnreadUnsub = subscribeMessageUnread(this.onMsgUnreadSummary);
    if (minibiliEnv) {
      void refreshMessageUnread();
    }
  },
  beforeUnmount() {
    if (this._msgUnreadUnsub) {
      this._msgUnreadUnsub();
      this._msgUnreadUnsub = null;
    }
  },
  activated() {
    if (minibiliEnv) {
      void this.$store.dispatch("login/refreshMinibiliMe");
      void refreshMessageUnread();
    }
  },
  watch: {
    $route: {
      immediate: true,
      handler(route) {
        if (
          route.name === "manuscript" ||
          route.name === "appeal" ||
          route.name === "upload" ||
          route.name === "videoPublish" ||
          route.name === "videoEdit" ||
          route.name === "articlePublish" ||
          route.name === "articleEdit"
        ) {
          this.navOpen.content = true;
        }
        if (route.name === "creatorComments" || route.name === "creatorDanmakus") {
          this.navOpen.interact = true;
        }
        if (minibiliEnv) {
          void refreshMessageUnread();
        }
      }
    }
  },
  methods: {
    creatorMessageTo(cat) {
      return {
        path: "/minibili/messages",
        query: { cat }
      };
    },
    messageBadge(cat) {
      return formatMessageUnreadBadge(this.msgUnread[cat]);
    },
    onMsgUnreadSummary(summary) {
      this.msgUnread = summary || {};
    },
    onMessageMouseEnter() {
      this.messageShow = true;
      if (minibiliEnv) {
        void refreshMessageUnread();
      }
    },
    onMbUploadNavClick() {
      setMinibiliPostLoginRedirect("/upload");
      this.$store.commit("login/SET_LOGIN_TAB", 0);
      this.$store.commit("login/OPEN_LOGIN_MODAL");
    },
    toggleNav(key) {
      this.navOpen[key] = !this.navOpen[key];
    }
  }
};
</script>

<style lang="scss" scoped>
$c-blue: #00a1d6;
$c-text: #18191c;
$c-sub: #9499a0;
$c-line: #e3e5e7;
$c-side-bg: #fff;

/* 使用视口高度：父级 router-view / keep-alive 通常无明确高度，height:100% 会失效导致主区域无法内部滚动 */
.creator-page {
  height: 100vh;
  max-height: 100vh;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: #fff;
  color: $c-text;
  font-family: Helvetica Neue, Helvetica, Hiragino Sans GB, Microsoft YaHei,
    sans-serif;
}
@supports (height: 100dvh) {
  .creator-page {
    height: 100dvh;
    max-height: 100dvh;
  }
}

.creator-header {
  flex-shrink: 0;
  background: #fff;
  border-bottom: 1px solid $c-line;
  position: relative;
  z-index: 300;
}

.creator-header-inner {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 24px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-sizing: border-box;
}

.creator-header-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.creator-logo {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: 600;
}

.creator-logo-img {
  display: block;
  height: 26px;
  width: auto;
  max-width: min(160px, 38vw);
  object-fit: contain;
  flex-shrink: 0;
}

.creator-logo-t {
  color: #00aeec;
  font-weight: 600;
}

.creator-home-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: $c-sub;
  text-decoration: none;
  &:hover {
    color: $c-blue;
  }
}

.creator-home-ico--member {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  background: url("@/assets/icons.png") -536px -1046px no-repeat;
}

.creator-home-link:hover .creator-home-ico--member {
  background-position: -601px -1046px;
}

.creator-header-right {
  display: flex;
  align-items: center;
  gap: 20px;
}

.creator-msg {
  position: relative;
}

.creator-msg-trigger {
  position: relative;
  display: inline-flex;
  text-decoration: none;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  color: $c-text;
  text-decoration: none;
  cursor: pointer;
  border-radius: 4px;
  &:hover {
    color: $c-blue;
    background: rgba(0, 0, 0, 0.04);
  }
}

.creator-msg-env {
  flex-shrink: 0;
  color: currentColor;
  opacity: 0.78;
}

.creator-msg-trigger:hover .creator-msg-env {
  opacity: 1;
}

.creator-msg-dropdown {
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  width: 110px;
  margin-top: 0;
  padding: 0;
  background: #fff;
  box-shadow: rgba(0, 0, 0, 0.16) 0 2px 4px;
  border-radius: 0 0 4px 4px;
  overflow: hidden;
}

.creator-msg-item {
  display: block;
  position: relative;
  text-align: center;
  line-height: 42px;
  height: 42px;
  font-size: 12px;
  color: $c-text;
  text-decoration: none;
  &:hover {
    color: $c-blue;
    background: #e5e9ef;
  }
}

.creator-msg-num {
  position: absolute;
  top: 13px;
  left: 80px;
  height: 14px;
  line-height: 15px;
  border-radius: 10px;
  padding: 1px 3px;
  min-width: 20px;
  text-align: center;
  font-size: 12px;
  color: #fff;
  background: #fb7299;
  display: inline-block;
  box-sizing: border-box;
}

.creator-dd-enter-from,
.creator-dd-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(4px);
}
.creator-dd-enter-active,
.creator-dd-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.creator-up-days {
  font-size: 13px;
  line-height: 1;
  white-space: nowrap;
}

.creator-up-days-txt {
  color: #99a2aa;
}

.creator-up-days-num {
  margin: 0 4px;
  color: #fa9600;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.creator-avatar-wrap {
  display: block;
}
.creator-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  vertical-align: middle;
}

.creator-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  max-width: 1400px;
  width: 100%;
  margin: 0 auto;
}

.creator-side {
  width: 200px;
  flex-shrink: 0;
  align-self: stretch;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  background: $c-side-bg;
  padding: 16px 12px;
  box-sizing: border-box;
  border-right: 1px solid $c-line;
  -webkit-overflow-scrolling: touch;
}

.creator-side-upload-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  height: 44px;
  margin-bottom: 16px;
  border-radius: 4px;
  background: $c-blue;
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  text-decoration: none;
  box-sizing: border-box;
}

.creator-side-upload-img {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  object-fit: contain;
  filter: brightness(0) invert(1);
}

.creator-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.creator-nav-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 10px 12px;
  border: none;
  border-radius: 4px;
  background: transparent;
  font-size: 14px;
  color: $c-text;
  text-align: left;
  text-decoration: none;
  cursor: pointer;
  box-sizing: border-box;
  &:hover {
    background: rgba(0, 0, 0, 0.04);
  }
}

.creator-nav-item.is-active {
  color: $c-blue;
  font-weight: 600;
}

.creator-nav-parent.is-open .creator-nav-arrow {
  transform: rotate(180deg);
}

.creator-nav-parent.is-active .creator-nav-arrow {
  border-top-color: $c-blue;
}

.creator-nav-leading {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.creator-nav-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  object-fit: contain;
}

.creator-nav-icon--interact {
  width: 22px;
  height: 22px;
}

.creator-nav-label {
  flex: 1;
  text-align: left;
}

.creator-nav-arrow {
  width: 0;
  height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 5px solid #999;
  transition: transform 0.28s ease;
}

.creator-nav-drawer {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 0.28s ease;
}

.creator-nav-drawer.is-open {
  grid-template-rows: 1fr;
}

.creator-nav-drawer-inner {
  overflow: hidden;
  min-height: 0;
}

.creator-nav-sub {
  padding: 4px 0 8px 8px;
}

.creator-nav-sub-item {
  display: block;
  padding: 8px 12px 8px 16px;
  font-size: 13px;
  color: $c-sub;
  text-decoration: none;
  border-radius: 4px;
  &:hover {
    color: $c-blue;
    background: rgba(0, 161, 214, 0.06);
  }
}

.creator-nav-sub-item.is-current {
  color: $c-blue;
  font-weight: 600;
}

.creator-main {
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 24px 40px 40px;
  box-sizing: border-box;
  -webkit-overflow-scrolling: touch;
}
</style>
