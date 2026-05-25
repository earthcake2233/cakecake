<template>
  <div
    v-if="showActions"
    class="mb-space-actions"
    :class="{ 'is-preview-readonly': previewOnly }"
  >
    <div class="mb-space-actions__follow-wrap">
      <span
        v-show="follow.hint"
        class="mb-space-actions__hint"
        role="status"
      >{{ follow.hint }}</span>
      <button
        type="button"
        class="mb-space-actions__btn mb-space-actions__btn--follow"
        :class="{ 'is-followed': followed }"
        :disabled="follow.pending || previewOnly"
        @mouseenter="onFollowEnter"
        @mouseleave="onFollowLeave"
        @click="onFollowClick"
      >
        {{ followButtonLabel }}
      </button>
    </div>
    <button
      type="button"
      class="mb-space-actions__btn mb-space-actions__btn--msg"
      :disabled="previewOnly"
      @click="onMessageClick"
    >
      发消息
    </button>
    <div
      class="mb-space-actions__more-wrap"
      @mouseenter="onMoreEnter"
      @mouseleave="onMoreLeave"
    >
      <button
        type="button"
        class="mb-space-actions__btn mb-space-actions__btn--more"
        aria-label="更多"
        :disabled="previewOnly"
        @click.stop="onMoreToggle"
      >
        <span class="mb-space-actions__more-dots" aria-hidden="true">
          <i /><i /><i />
        </span>
      </button>
      <ul
        v-show="moreOpen"
        class="mb-space-actions__menu"
        role="menu"
        @mouseenter="onMoreEnter"
        @mouseleave="onMoreLeave"
        @click.stop
      >
        <li role="none">
          <button type="button" role="menuitem" @click="onBlacklistClick">
            加入黑名单
          </button>
        </li>
        <li role="none">
          <button type="button" role="menuitem" @click="onReportClick">
            举报
          </button>
        </li>
      </ul>
    </div>
  </div>
</template>

<script>
import { getAccessToken } from "@/utils/authTokens";
import { mbBlockUser, mbToggleUserFollow } from "@/api/minibili";

export default {
  name: "MbSpaceHeaderActions",
  props: {
    userId: { type: Number, default: 0 },
    isOwner: { type: Boolean, default: false },
    followedByMe: { type: Boolean, default: false },
    previewOnly: { type: Boolean, default: false }
  },
  emits: ["update:followedByMe", "follower-count", "login"],
  data() {
    return {
      follow: { pending: false, hover: false, hint: "" },
      moreOpen: false,
      _followHintTimer: null,
      _moreCloseTimer: null
    };
  },
  computed: {
    showActions() {
      return this.userId > 0 && !this.isOwner;
    },
    followed: {
      get() {
        return !!this.followedByMe;
      },
      set(v) {
        this.$emit("update:followedByMe", !!v);
      }
    },
    followButtonLabel() {
      if (this.follow.pending) {
        return "…";
      }
      if (!this.followed) {
        return "+ 关注";
      }
      if (this.follow.hover) {
        return "取消关注";
      }
      return "已关注";
    }
  },
  beforeUnmount() {
    this.clearFollowHintTimer();
    this.clearMoreCloseTimer();
  },
  methods: {
    clearFollowHintTimer() {
      if (this._followHintTimer) {
        clearTimeout(this._followHintTimer);
        this._followHintTimer = null;
      }
    },
    clearMoreCloseTimer() {
      if (this._moreCloseTimer) {
        clearTimeout(this._moreCloseTimer);
        this._moreCloseTimer = null;
      }
    },
    showFollowHint(message) {
      this.follow.hint = String(message || "");
      this.clearFollowHintTimer();
      if (!this.follow.hint) {
        return;
      }
      this._followHintTimer = setTimeout(() => {
        this.follow.hint = "";
        this._followHintTimer = null;
      }, 2000);
    },
    onFollowEnter() {
      if (this.previewOnly) {
        return;
      }
      if (this.followed && !this.follow.pending) {
        this.follow.hover = true;
      }
    },
    onFollowLeave() {
      this.follow.hover = false;
    },
    onMoreEnter() {
      if (this.previewOnly) {
        return;
      }
      this.clearMoreCloseTimer();
      this.moreOpen = true;
    },
    onMoreLeave() {
      this.clearMoreCloseTimer();
      this._moreCloseTimer = setTimeout(() => {
        this.moreOpen = false;
        this._moreCloseTimer = null;
      }, 120);
    },
    onMoreToggle() {
      if (this.previewOnly) {
        return;
      }
      this.moreOpen = !this.moreOpen;
    },
    onMessageClick() {
      if (this.previewOnly) {
        return;
      }
      if (!getAccessToken()) {
        this.$emit("login");
        return;
      }
      if (!this.userId) {
        return;
      }
      this.$router.push({
        path: "/minibili/messages",
        query: { cat: "my_message", peer_id: String(this.userId) }
      });
    },
    async onBlacklistClick() {
      this.moreOpen = false;
      if (!getAccessToken()) {
        this.$emit("login");
        return;
      }
      if (!this.userId) return;
      try {
        await mbBlockUser(this.userId);
        this.followed = false;
        this.showFollowHint("已加入黑名单");
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败";
        this.showFollowHint(msg);
      }
    },
    onReportClick() {
      this.moreOpen = false;
      this.showFollowHint("举报功能开发中");
    },
    async onFollowClick() {
      if (this.previewOnly || !this.userId || this.follow.pending) {
        return;
      }
      if (!getAccessToken()) {
        this.$emit("login");
        return;
      }
      this.follow.pending = true;
      try {
        const res = await mbToggleUserFollow(this.userId);
        this.followed = !!res.followed;
        const fans = Number(res.follower_count);
        if (Number.isFinite(fans) && fans >= 0) {
          this.$emit("follower-count", fans);
        }
        this.follow.hover = false;
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败";
        this.showFollowHint(msg);
      } finally {
        this.follow.pending = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
.mb-space-actions {
  position: relative;
  z-index: 12;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.mb-space-actions.is-preview-readonly {
  pointer-events: none;
  user-select: none;

  .mb-space-actions__btn {
    cursor: default;
    opacity: 0.92;
  }
}

.mb-space-actions__follow-wrap {
  position: relative;
}

.mb-space-actions__hint {
  position: absolute;
  left: 50%;
  bottom: calc(100% + 8px);
  transform: translateX(-50%);
  z-index: 6;
  padding: 6px 10px;
  border-radius: 4px;
  background: #f85a54;
  color: #fff;
  font-size: 12px;
  line-height: 1.2;
  white-space: nowrap;
  pointer-events: none;
  box-shadow: 0 2px 8px rgba(248, 90, 84, 0.35);
}

.mb-space-actions__btn {
  box-sizing: border-box;
  border-radius: 6px;
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
  transition:
    background 0.15s ease,
    border-color 0.15s ease,
    color 0.15s ease;

  &:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }
}

.mb-space-actions__btn--follow {
  min-width: 128px;
  height: 34px;
  padding: 0 26px;
  border: none;
  background: #00a1d6;
  color: #fff;
  font-weight: 500;

  &:hover:not(:disabled) {
    background: #00b5e5;
  }

  &.is-followed {
    background: rgba(255, 255, 255, 0.22);
    color: #fff;
    border: 1px solid rgba(255, 255, 255, 0.55);

    &:hover:not(:disabled) {
      background: rgba(255, 255, 255, 0.32);
    }
  }
}

.mb-space-actions__btn--msg {
  min-width: 116px;
  height: 34px;
  padding: 0 26px;
  border: 1px solid rgba(255, 255, 255, 0.55);
  background: rgba(0, 0, 0, 0.18);
  color: #fff;

  &:hover:not(:disabled) {
    background: rgba(0, 0, 0, 0.28);
  }
}

.mb-space-actions__more-wrap {
  position: relative;
  z-index: 13;
}

.mb-space-actions__btn--more {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  padding: 0;
  border: 1px solid rgba(255, 255, 255, 0.55);
  background: rgba(0, 0, 0, 0.18);
  color: #fff;

  &:hover {
    background: rgba(0, 0, 0, 0.28);
  }
}

.mb-space-actions__more-dots {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  width: 14px;
  height: 18px;

  i {
    display: block;
    width: 3px;
    height: 3px;
    border-radius: 50%;
    background: #fff;
  }
}

.mb-space-actions__menu {
  position: absolute;
  right: 0;
  bottom: calc(100% + 8px);
  top: auto;
  z-index: 30;
  min-width: 120px;
  margin: 0;
  padding: 6px 0;
  list-style: none;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);

  button {
    display: block;
    width: 100%;
    padding: 9px 16px;
    border: none;
    background: transparent;
    color: #61666d;
    font-size: 14px;
    line-height: 1.3;
    text-align: center;
    cursor: pointer;
    white-space: nowrap;

    &:hover {
      background: #f6f7f8;
      color: #18191c;
    }
  }
}
</style>
