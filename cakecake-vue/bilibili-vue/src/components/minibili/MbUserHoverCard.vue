<template>
  <span
    ref="root"
    class="mb-user-hover"
    @mouseenter="onTriggerEnter"
    @mouseleave="onTriggerLeave"
  >
    <slot />
    <Teleport to="body">
      <div
        v-show="panelVisible"
        ref="panel"
        class="mb-user-hover-card"
        :style="panelStyle"
        role="dialog"
        aria-label="用户资料"
        @mouseenter="onPanelEnter"
        @mouseleave="onPanelLeave"
      >
        <div v-if="loading" class="mb-user-hover-card__loading">
          加载中…
        </div>
        <template v-else-if="profile">
          <div
            class="mb-user-hover-card__banner"
            :style="{ backgroundImage: 'url(' + bannerUrl + ')' }"
            aria-hidden="true"
          />
          <div class="mb-user-hover-card__body">
            <router-link
              v-if="spaceRoute"
              class="mb-user-hover-card__avatar-hit"
              :to="spaceRoute"
              @click="hidePanel"
            >
              <img
                class="mb-user-hover-card__avatar"
                :src="avatarSrc"
                alt=""
              />
            </router-link>
            <img
              v-else
              class="mb-user-hover-card__avatar"
              :src="avatarSrc"
              alt=""
            />
            <div class="mb-user-hover-card__main">
              <div class="mb-user-hover-card__name-row">
                <router-link
                  v-if="spaceRoute"
                  class="mb-user-hover-card__name"
                  :class="'tone-' + nameTone"
                  :to="spaceRoute"
                  @click="hidePanel"
                >
                  {{ displayName }}
                </router-link>
                <span
                  v-else
                  class="mb-user-hover-card__name"
                  :class="'tone-' + nameTone"
                >
                  {{ displayName }}
                </span>
                <img
                  class="mb-user-hover-card__level"
                  :src="levelIconUrl(levelDisplay)"
                  width="30"
                  height="30"
                  alt=""
                  :title="'LV' + levelDisplay"
                />
              </div>
              <div class="mb-user-hover-card__stats">
                <span
                  ><em>{{ statFollowing }}</em> 关注</span
                >
                <span
                  ><em>{{ statFollowers }}</em> 粉丝</span
                >
                <span
                  ><em>{{ statPublished }}</em> 投稿</span
                >
              </div>
              <p class="mb-user-hover-card__sign">{{ displaySign }}</p>
              <div v-if="!isSelf" class="mb-user-hover-card__actions">
                <button
                  type="button"
                  class="mb-user-hover-card__btn mb-user-hover-card__btn--primary"
                  :class="{ 'is-followed': followed }"
                  :disabled="followPending"
                  @click.stop="onFollowClick"
                >
                  {{ followLabel }}
                </button>
                <button
                  type="button"
                  class="mb-user-hover-card__btn mb-user-hover-card__btn--ghost"
                  @click.stop="onMessageClick"
                >
                  发消息
                </button>
              </div>
            </div>
          </div>
        </template>
        <div v-else-if="loadError" class="mb-user-hover-card__loading">
          {{ loadError }}
        </div>
      </div>
    </Teleport>
  </span>
</template>

<script>
import defaultFace from "@/assets/akari.jpg";
import defaultBanner from "@/assets/personal_space/1sz3p8w2Sk.png@3840w_400h_1c_100q.avif";
import { mbGetUserPublic, mbToggleUserFollow } from "@/api/minibili";
import { getAccessToken, getUserId } from "@/utils/authTokens";
import { openMinibiliLoginModal } from "@/utils/minibiliLoginModal";
import { minibiliUserSpaceRoute } from "@/utils/minibiliRoutes";
import { levelIconUrl } from "@/utils/userLevel";
import { ElMessage } from "element-plus";

export const userHoverProfileCache = new Map();

export function invalidateUserHoverProfileCache(userId) {
  const id = Number(userId) || 0;
  if (id > 0) userHoverProfileCache.delete(id);
}
const SHOW_DELAY_MS = 280;
const HIDE_DELAY_MS = 220;

function formatStat(n) {
  const v = Number(n) || 0;
  if (v >= 10000) {
    return (Math.round(v / 1000) / 10).toFixed(1) + "万";
  }
  return String(v);
}

export default {
  name: "MbUserHoverCard",
  props: {
    userId: { type: Number, required: true }
  },
  emits: ["follow-change"],
  data() {
    return {
      panelVisible: false,
      loading: false,
      loadError: "",
      profile: null,
      followed: false,
      followPending: false,
      panelStyle: { top: "0px", left: "0px", visibility: "hidden" },
      bannerUrl: defaultBanner,
      _showTimer: null,
      _hideTimer: null,
      _inPanel: false
    };
  },
  computed: {
    uid() {
      const n = Number(this.userId) || 0;
      return n > 0 ? n : 0;
    },
    isSelf() {
      const me = getUserId();
      return me != null && Number(me) === this.uid;
    },
    spaceRoute() {
      return minibiliUserSpaceRoute(this.uid);
    },
    displayName() {
      if (!this.profile) return "用户";
      const n = String(this.profile.nickname || "").trim();
      return n || "用户" + this.uid;
    },
    avatarSrc() {
      const u = this.profile && String(this.profile.avatar_url || "").trim();
      return u || defaultFace;
    },
    displaySign() {
      const s = this.profile && String(this.profile.sign || "").trim();
      return s || "这个家伙很懒，什么都没有写";
    },
    levelDisplay() {
      const li = this.profile && this.profile.level_info;
      const lv = li && Number(li.current_level);
      if (Number.isFinite(lv) && lv >= 1) {
        return Math.min(6, Math.max(1, Math.floor(lv)));
      }
      return 1;
    },
    nameTone() {
      const tones = ["blue", "pink", "black"];
      return tones[Math.abs(this.uid) % 3];
    },
    statFollowing() {
      return formatStat(this.profile && this.profile.following_count);
    },
    statFollowers() {
      return formatStat(this.profile && this.profile.follower_count);
    },
    statPublished() {
      return formatStat(this.profile && this.profile.published_count);
    },
    followLabel() {
      if (this.followPending) return "…";
      return this.followed ? "已关注" : "+ 关注";
    }
  },
  beforeUnmount() {
    this.clearTimers();
  },
  methods: {
    levelIconUrl,
    clearTimers() {
      if (this._showTimer) {
        clearTimeout(this._showTimer);
        this._showTimer = null;
      }
      if (this._hideTimer) {
        clearTimeout(this._hideTimer);
        this._hideTimer = null;
      }
    },
    onTriggerEnter() {
      if (!this.uid) return;
      this.clearTimers();
      this._hideTimer = null;
      this._showTimer = setTimeout(() => {
        this._showTimer = null;
        void this.openPanel();
      }, SHOW_DELAY_MS);
    },
    onTriggerLeave() {
      this.clearTimers();
      this._showTimer = null;
      this._hideTimer = setTimeout(() => {
        if (!this._inPanel) this.hidePanel();
      }, HIDE_DELAY_MS);
    },
    onPanelEnter() {
      this._inPanel = true;
      if (this._hideTimer) {
        clearTimeout(this._hideTimer);
        this._hideTimer = null;
      }
    },
    onPanelLeave() {
      this._inPanel = false;
      this._hideTimer = setTimeout(() => this.hidePanel(), HIDE_DELAY_MS);
    },
    hidePanel() {
      this.panelVisible = false;
      this.panelStyle = {
        ...this.panelStyle,
        visibility: "hidden"
      };
    },
    async openPanel() {
      if (!this.uid) return;
      this.panelVisible = true;
      this.$nextTick(() => this.updatePosition());
      const cached = userHoverProfileCache.get(this.uid);
      if (cached) {
        this.profile = cached;
        this.followed = !!cached.followed_by_me;
        this.loadError = "";
        this.loading = false;
        return;
      }
      this.loading = true;
      this.loadError = "";
      this.profile = null;
      try {
        const p = await mbGetUserPublic(this.uid, { skipGlobalErrorToast: true });
        userHoverProfileCache.set(this.uid, p);
        this.profile = p;
        this.followed = !!p.followed_by_me;
      } catch (e) {
        this.loadError = (e && e.message) || "加载失败";
        this.profile = null;
      } finally {
        this.loading = false;
        this.$nextTick(() => this.updatePosition());
      }
    },
    updatePosition() {
      const root = this.$refs.root;
      const panel = this.$refs.panel;
      if (!root || !panel) return;
      const rect = root.getBoundingClientRect();
      const pw = panel.offsetWidth || 376;
      const ph = panel.offsetHeight || 280;
      const gap = 8;
      let top = rect.bottom + gap;
      let left = rect.left;
      if (top + ph > window.innerHeight - 8) {
        top = Math.max(8, rect.top - ph - gap);
      }
      if (left + pw > window.innerWidth - 8) {
        left = Math.max(8, window.innerWidth - pw - 8);
      }
      this.panelStyle = {
        top: `${Math.round(top)}px`,
        left: `${Math.round(left)}px`,
        visibility: "visible"
      };
    },
    openLogin() {
      openMinibiliLoginModal({ tab: 0 });
    },
    async onFollowClick() {
      if (!getAccessToken()) {
        this.openLogin();
        return;
      }
      if (this.isSelf) return;
      if (this.followPending) return;
      this.followPending = true;
      try {
        const res = await mbToggleUserFollow(this.uid);
        this.followed = !!res.followed;
        if (this.profile) {
          const next = {
            ...this.profile,
            followed_by_me: this.followed,
            follower_count: res.follower_count
          };
          this.profile = next;
          userHoverProfileCache.set(this.uid, next);
        }
        this.$emit("follow-change", {
          userId: this.uid,
          followed: this.followed,
          follower_count: res.follower_count
        });
      } catch (e) {
        ElMessage.error((e && e.message) || "关注失败");
      } finally {
        this.followPending = false;
      }
    },
    onMessageClick() {
      if (!getAccessToken()) {
        this.openLogin();
        return;
      }
      if (this.isSelf) return;
      this.hidePanel();
      void this.$router
        .push({
          path: "/minibili/messages",
          query: { cat: "my_message", peer_id: String(this.uid) }
        })
        .catch(() => {});
    }
  }
};
</script>

<style lang="scss" scoped>
.mb-user-hover {
  display: inline-flex;
  vertical-align: inherit;
  max-width: 100%;
}

.mb-user-hover-card {
  position: fixed;
  z-index: 10050;
  width: 376px;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.14);
  border: 1px solid #e3e5e7;
  box-sizing: border-box;
}

.mb-user-hover-card__loading {
  padding: 48px 16px;
  text-align: center;
  font-size: 14px;
  color: #9499a0;
}

.mb-user-hover-card__banner {
  height: 118px;
  background: #c9ccd0 center / cover no-repeat;
}

.mb-user-hover-card__body {
  display: flex;
  gap: 12px;
  padding: 0 16px 16px;
  margin-top: -24px;
  position: relative;
}

.mb-user-hover-card__avatar-hit {
  flex-shrink: 0;
  display: block;
  line-height: 0;
}

.mb-user-hover-card__avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  border: 2px solid #fff;
  object-fit: cover;
  background: #f1f2f3;
  display: block;
}

.mb-user-hover-card__main {
  min-width: 0;
  flex: 1;
  padding-top: 28px;
}

.mb-user-hover-card__name-row {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.mb-user-hover-card__name {
  font-size: 15px;
  font-weight: 600;
  line-height: 22px;
  color: #18191c;
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
  &.tone-blue {
    color: #00aeec;
  }
  &.tone-pink {
    color: #fb7299;
  }
  &.tone-black {
    color: #18191c;
  }
  &:hover {
    opacity: 0.92;
  }
}

.mb-user-hover-card__level {
  flex-shrink: 0;
  display: block;
}

.mb-user-hover-card__stats {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 6px;
  font-size: 12px;
  line-height: 18px;
  color: #9499a0;
  em {
    font-style: normal;
    color: #18191c;
    font-weight: 600;
    margin-right: 2px;
  }
}

.mb-user-hover-card__sign {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 18px;
  color: #61666d;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  overflow: hidden;
  word-break: break-word;
}

.mb-user-hover-card__actions {
  display: flex;
  gap: 10px;
  margin-top: 12px;
}

.mb-user-hover-card__btn {
  flex: 1;
  height: 32px;
  border-radius: 4px;
  font-size: 14px;
  line-height: 30px;
  cursor: pointer;
  box-sizing: border-box;
  &--primary {
    border: 1px solid #00aeec;
    background: #00aeec;
    color: #fff;
    &.is-followed {
      background: #fff;
      color: #61666d;
      border-color: #e3e5e7;
    }
    &:hover:not(:disabled) {
      filter: brightness(1.03);
    }
  }
  &--ghost {
    border: 1px solid #e3e5e7;
    background: #fff;
    color: #18191c;
    &:hover:not(:disabled) {
      border-color: #c9ccd0;
      background: #fafbfd;
    }
  }
  &:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }
}
</style>
