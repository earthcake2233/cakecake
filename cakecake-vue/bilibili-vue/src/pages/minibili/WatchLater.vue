<template>
  <div class="mb-wl-page">
    <div class="bili-wrapper mb-wl-wrap">
      <div v-if="!token" class="mb-wl-login">
        <p>
          {{ wl.loginPrefix }}
          <a href="#" class="mb-wl-login__link" @click.prevent="openLoginModal">{{
            wl.loginLink
          }}</a>
          {{ wl.loginSuffix }}
        </p>
      </div>

      <div v-else class="mb-wl-card">
        <header class="mb-wl-head">
          <div class="mb-wl-head-left">
            <img
              class="mb-wl-head-ico"
              :src="watchLaterIcon"
              width="28"
              height="28"
              alt=""
            />
            <h1 class="mb-wl-head-title">
              {{ wl.title }}<span class="mb-wl-head-count"
                >{{ wl.countWrapLeft }}{{ totalDisplay }}{{ wl.countWrapRight }}</span
              >
            </h1>
          </div>
          <div class="mb-wl-head-actions">
            <button
              type="button"
              class="mb-wl-btn mb-wl-btn--ghost"
              :disabled="loading || !items.length"
              @click="onPlayAll"
            >
              {{ wl.playAll }}
            </button>
            <button
              type="button"
              class="mb-wl-btn mb-wl-btn--ghost"
              :disabled="loading || !items.length"
              @click="onClearAll"
            >
              {{ wl.clearAll }}
            </button>
            <button
              type="button"
              class="mb-wl-btn mb-wl-btn--ghost"
              :disabled="loading || !hasWatched"
              @click="onClearWatched"
            >
              {{ wl.clearWatched }}
            </button>
          </div>
        </header>

        <div v-if="loading" class="mb-wl-loading" role="status" aria-live="polite">
          <img class="mb-wl-loading-gif" :src="loadingGif" alt="" />
          <span class="mb-wl-loading-text">{{ wl.loading }}</span>
        </div>

        <div
          v-else-if="!items.length"
          class="mb-wl-empty"
          role="img"
          :aria-label="wl.emptyAria"
        >
          <img :src="emptyImg" alt="" />
          <p>{{ wl.emptyHint }}</p>
        </div>

        <ol v-else class="mb-wl-list" :aria-label="wl.listAria">
          <li
            v-for="(row, idx) in items"
            :key="'wl-' + row.id"
            class="mb-wl-row"
          >
            <span class="mb-wl-idx" aria-hidden="true">{{ idx + 1 }}</span>
            <router-link
              class="mb-wl-thumb"
              :to="minibiliVideoPlayRoute(row.id)"
            >
              <img
                class="mb-wl-thumb-img"
                :src="row.cover_url || akari"
                :alt="row.title"
              />
              <span class="mb-wl-dur">{{ formatDuration(row.duration) }}</span>
            </router-link>
            <div class="mb-wl-meta">
              <router-link
                class="mb-wl-title"
                :to="minibiliVideoPlayRoute(row.id)"
                :title="row.title"
              >
                {{ row.title }}
              </router-link>
              <router-link
                v-if="row.uploader_id"
                class="mb-wl-up"
                :to="uploaderRoute(row.uploader_id)"
              >
                <img
                  class="mb-wl-up-face"
                  :src="row.uploader_avatar_url || akari"
                  alt=""
                />
                <span class="mb-wl-up-name">{{ row.uploader }}</span>
              </router-link>
              <span v-else class="mb-wl-up">
                <img class="mb-wl-up-face" :src="akari" alt="" />
                <span class="mb-wl-up-name">{{ row.uploader }}</span>
              </span>
            </div>
            <div class="mb-wl-row-actions">
              <span v-if="row.watched" class="mb-wl-watched-tag">{{ wl.watched }}</span>
              <button
                type="button"
                class="mb-wl-del"
                :aria-label="wl.removeAria"
                :disabled="removingId === row.id"
                @click="onRemove(row)"
              >
                <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
                  <path
                    fill="currentColor"
                    d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"
                  />
                </svg>
              </button>
            </div>
          </li>
        </ol>
      </div>
    </div>
  </div>
</template>


<script>
import akari from "@/assets/akari.jpg";
import emptyImg from "@/assets/empty_2.png";
import watchLaterIcon from "@/assets/watch_later/icon.428b6a5.png";
import loadingGif from "@/assets/watch_later/loading.b91e4c8.gif";
import {
  mbListMyWatchLater,
  mbToggleWatchLater,
  mbClearMyWatchLater,
  mbClearWatchedWatchLater
} from "@/api/minibili";
import { getAccessToken } from "@/utils/authTokens";
import { openMinibiliLoginModal } from "@/utils/minibiliLoginModal";
import {
  minibiliUserSpaceRoute,
  minibiliVideoPlayRoute
} from "@/utils/minibiliRoutes";
import { ElMessage, ElMessageBox } from "element-plus";
import { watchLaterZhCN as wl } from "@/i18n/watchLater.zh-CN";

export default {
  name: "MinibiliWatchLater",
  data() {
    return {
      wl,
      akari,
      emptyImg,
      watchLaterIcon,
      loadingGif,
      loading: false,
      items: [],
      total: 0,
      maxLimit: 100,
      removingId: null
    };
  },
  computed: {
    token() {
      return getAccessToken();
    },
    totalDisplay() {
      return `${this.total}/${this.maxLimit}`;
    },
    hasWatched() {
      return this.items.some(r => r.watched);
    }
  },
  watch: {
    token(tok) {
      if (!tok) {
        this.items = [];
        this.total = 0;
        this.loading = false;
      }
    }
  },
  mounted() {
    this._wlSkipActivatedOnce = true;
    this.onPageEnter();
  },
  activated() {
    if (this._wlSkipActivatedOnce) {
      this._wlSkipActivatedOnce = false;
      return;
    }
    this.onPageEnter();
  },
  methods: {
    minibiliVideoPlayRoute,
    openLoginModal() {
      openMinibiliLoginModal({ tab: 0, redirect: "/minibili/watch-later" });
    },
    onPageEnter() {
      document.title = wl.pageTitle;
      if (!this.token) {
        this.items = [];
        this.total = 0;
        this.loading = false;
        return;
      }
      void this.loadList();
    },
    uploaderRoute(uid) {
      const r = minibiliUserSpaceRoute(uid);
      return r || { name: "home" };
    },
    formatDuration(sec) {
      const s = Math.max(0, Math.floor(Number(sec) || 0));
      const h = Math.floor(s / 3600);
      const m = Math.floor((s % 3600) / 60);
      const ss = s % 60;
      const pad = n => String(n).padStart(2, "0");
      if (h > 0) {
        return `${h}:${pad(m)}:${pad(ss)}`;
      }
      return `${m}:${pad(ss)}`;
    },
    async loadList() {
      if (!this.token) return;
      this.loading = true;
      this.items = [];
      this.total = 0;
      try {
        const res = await mbListMyWatchLater({ limit: 100 });
        this.items = Array.isArray(res.items) ? res.items : [];
        this.total =
          typeof res.total === "number" ? res.total : this.items.length;
        this.maxLimit =
          typeof res.max_limit === "number" ? res.max_limit : 100;
      } catch (e) {
        this.items = [];
        ElMessage.error((e && e.message) || wl.loadFailed);
      } finally {
        this.loading = false;
      }
    },
    onPlayAll() {
      if (!this.items.length) return;
      const first = this.items[0];
      this.$router
        .push(minibiliVideoPlayRoute(first.id))
        .catch(() => {});
    },
    async onClearAll() {
      if (!this.items.length) return;
      try {
        await ElMessageBox.confirm(wl.clearConfirmMessage, wl.clearConfirmTitle, {
          confirmButtonText: wl.confirm,
          cancelButtonText: wl.cancel,
          center: true,
          showClose: true,
          customClass: "mb-wl-clear-msgbox",
          confirmButtonClass: "mb-wl-clear-msgbox__ok",
          cancelButtonClass: "mb-wl-clear-msgbox__cancel"
        });
      } catch {
        return;
      }
      try {
        await mbClearMyWatchLater();
        this.items = [];
        this.total = 0;
        ElMessage.success(wl.cleared);
      } catch (e) {
        ElMessage.error((e && e.message) || wl.loadFailed);
      }
    },
    async onClearWatched() {
      if (!this.hasWatched) return;
      try {
        await mbClearWatchedWatchLater();
        await this.loadList();
        ElMessage.success(wl.clearWatchedDone);
      } catch (e) {
        ElMessage.error((e && e.message) || wl.loadFailed);
      }
    },
    async onRemove(row) {
      const id = Number(row.id);
      if (!Number.isFinite(id) || id <= 0) return;
      this.removingId = id;
      try {
        await mbToggleWatchLater(id);
        this.items = this.items.filter(r => Number(r.id) !== id);
        this.total = Math.max(0, this.total - 1);
      } catch (e) {
        ElMessage.error((e && e.message) || wl.loadFailed);
      } finally {
        this.removingId = null;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
$text1: #18191c;
$text2: #61666d;
$text3: #9499a0;
$line: #e3e5e7;
$bili-blue: #00aeec;

.mb-wl-page {
  min-height: calc(100vh - 64px);
  padding: 0 0 40px;
  box-sizing: border-box;
  background: #fff;
}

.mb-wl-wrap {
  box-sizing: border-box;
}

.mb-wl-login {
  padding: 48px 24px;
  text-align: center;
  background: #fff;
  font-size: 14px;
  color: $text2;
}

.mb-wl-login__link {
  color: #00aeec;
  text-decoration: none;
  cursor: pointer;
  &:hover {
    color: #00b5e5;
  }
}

.mb-wl-card {
  background: #fff;
  overflow: hidden;
}

.mb-wl-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 56px;
  padding: 16px 20px;
  border-bottom: 1px solid $line;
  box-sizing: border-box;
}

.mb-wl-head-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex-shrink: 0;
}

.mb-wl-head-ico {
  display: block;
  width: 28px;
  height: 28px;
  flex-shrink: 0;
}

.mb-wl-head-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  line-height: 28px;
  color: $text1;
  white-space: nowrap;
}

.mb-wl-head-count {
  font-weight: 600;
  color: $text1;
}

.mb-wl-head-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.mb-wl-btn {
  height: 28px;
  padding: 0 14px;
  border: 1px solid $bili-blue;
  border-radius: 4px;
  background: #fff;
  color: $bili-blue;
  font-size: 13px;
  line-height: 26px;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s ease, color 0.15s ease;
  &:hover:not(:disabled) {
    background: rgba(0, 174, 236, 0.06);
  }
  &:disabled {
    opacity: 0.45;
    cursor: not-allowed;
    border-color: #c9ccd0;
    color: $text3;
  }
}

.mb-wl-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 420px;
  padding: 48px 20px;
}

.mb-wl-loading-gif {
  width: 24px;
  height: 24px;
  display: block;
  flex-shrink: 0;
}

.mb-wl-loading-text {
  font-size: 14px;
  color: $text3;
  line-height: 1.5;
}

.mb-wl-empty {
  padding: 56px 20px 72px;
  text-align: center;
  img {
    width: 280px;
    max-width: 100%;
    height: auto;
    display: block;
    margin: 0 auto 12px;
  }
  p {
    margin: 0;
    font-size: 14px;
    color: $text3;
  }
}

.mb-wl-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.mb-wl-row {
  display: grid;
  grid-template-columns: 36px 160px minmax(0, 1fr) auto;
  align-items: center;
  column-gap: 16px;
  min-height: 100px;
  padding: 16px 20px;
  border-bottom: 1px solid $line;
  box-sizing: border-box;
  &:last-child {
    border-bottom: none;
  }
}

.mb-wl-idx {
  justify-self: center;
  font-size: 14px;
  line-height: 1;
  color: $text3;
  font-variant-numeric: tabular-nums;
}

.mb-wl-thumb {
  position: relative;
  display: block;
  width: 160px;
  height: 90px;
  border-radius: 4px;
  overflow: hidden;
  background: #f1f2f3;
  flex-shrink: 0;
}

.mb-wl-thumb-img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.mb-wl-dur {
  position: absolute;
  right: 6px;
  bottom: 6px;
  padding: 0 4px;
  border-radius: 2px;
  background: rgba(0, 0, 0, 0.75);
  color: #fff;
  font-size: 12px;
  line-height: 18px;
  font-variant-numeric: tabular-nums;
}

.mb-wl-meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 10px;
  padding-right: 8px;
}

.mb-wl-title {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  font-size: 15px;
  font-weight: 500;
  line-height: 22px;
  color: $text1;
  text-decoration: none;
  &:hover {
    color: $bili-blue;
  }
}

.mb-wl-up {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  max-width: 100%;
  text-decoration: none;
  color: $text2;
  &:hover .mb-wl-up-name {
    color: $bili-blue;
  }
}

.mb-wl-up-face {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  background: #f1f2f3;
}

.mb-wl-up-name {
  font-size: 13px;
  line-height: 22px;
  color: $text2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mb-wl-row-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 16px;
  flex-shrink: 0;
  min-width: 88px;
}

.mb-wl-watched-tag {
  font-size: 13px;
  line-height: 20px;
  color: $text3;
  white-space: nowrap;
}

.mb-wl-del {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: $text3;
  cursor: pointer;
  transition: color 0.15s ease, background 0.15s ease;
  &:hover:not(:disabled) {
    color: $text2;
    background: rgba(0, 0, 0, 0.04);
  }
  &:disabled {
    opacity: 0.5;
    cursor: wait;
  }
}
</style>

<style lang="scss">
@import "../../styles/mb-wl-clear-msgbox.scss";
</style>
