# -*- coding: utf-8 -*-
import json
from pathlib import Path

TARGET = Path(__file__).resolve().parents[1] / "cakecake-vue/bilibili-vue/src/pages/upload/commentManage.vue"

S = {
    "tab_visible": "\u7528\u6237\u53ef\u89c1\u8bc4\u8bba",
    "tab_pending": "\u5f85\u7cbe\u9009\u8bc4\u8bba",
    "sub_video": "\u89c6\u9891\u8bc4\u8bba",
    "sub_article": "\u4e13\u680f\u8bc4\u8bba",
    "sub_dynamic": "\u52a8\u6001\u8bc4\u8bba",
    "select_all": "\u5168\u9009",
    "report": "\u4e3e\u62a5",
    "pick": "\u7cbe\u9009",
    "ignore": "\u5ffd\u7565",
    "deleting": "\u5220\u9664\u4e2d\u2026",
    "delete": "\u5220\u9664",
    "search_ph": "\u641c\u7d22\u89c6\u9891\u8bc4\u8bba",
    "search_aria": "\u641c\u7d22",
    "all_videos": "\u5168\u90e8\u89c6\u9891",
    "pending_unprocessed": "\u672a\u5904\u7406",
    "pending_ignored": "\u5df2\u5ffd\u7565",
    "pending_all": "\u5168\u90e8",
    "scope_root": "\u6839\u8bc4\u8bba",
    "scope_reply": "\u56de\u590d\u8bc4\u8bba",
    "loading": "\u52a0\u8f7d\u4e2d\u2026",
    "user": "\u7528\u6237",
    "reply": "\u56de\u590d",
    "empty": "\u6682\u65e0\u8bc4\u8bba",
    "empty_pending": "\u8fd8\u6ca1\u6709\u8bc4\u8bba\u54e6~",
    "empty_hint": "\u8bf7\u5f00\u542f Mini-Bili \u6a21\u5f0f\u5e76\u767b\u5f55\u540e\u67e5\u770b",
    "foot_note": "\u4ec5\u5c55\u793a\u6700\u8fd1\u768450000\u6761\u8bc4\u8bba",
    "prev": "\u4e0a\u4e00\u9875",
    "next": "\u4e0b\u4e00\u9875",
    "sort_recent": "\u6700\u8fd1\u53d1\u5e03",
    "sort_earliest": "\u6700\u65e9\u53d1\u5e03",
    "sort_likes": "\u70b9\u8d5e\u6700\u591a",
    "sort_replies": "\u56de\u590d\u6700\u591a",
    "load_fail": "\u52a0\u8f7d\u5931\u8d25",
    "report_soon": "\u4e3e\u62a5\u529f\u80fd\u5373\u5c06\u5f00\u653e",
    "pick_ok": "\u5df2\u7cbe\u9009",
    "ignore_ok": "\u5df2\u5ffd\u7565",
    "pick_fail": "\u7cbe\u9009\u5931\u8d25",
    "ignore_fail": "\u5ffd\u7565\u5931\u8d25",
    "acting": "\u5904\u7406\u4e2d\u2026",
    "del_reminder_title": "\u5220\u9664\u63d0\u9192",
    "del_reminder_msg": "\u5220\u9664\u540e\u65e0\u6cd5\u6062\u590d\uff0c\u786e\u8ba4\u5220\u9664\u9009\u4e2d\u7684\u8bc4\u8bba\u5417\uff1f",
    "confirm_ok": "\u786e\u5b9a",
    "cancel": "\u53d6\u6d88",
    "del_done": "\u5df2\u5220\u9664",
    "del_fail": "\u5220\u9664\u5931\u8d25",
    "login_warn": "\u8bf7\u5148\u767b\u5f55",
    "like_fail": "\u70b9\u8d5e\u5931\u8d25",
    "post_ok": "\u53d1\u8868\u6210\u529f",
    "post_fail": "\u53d1\u8868\u5931\u8d25",
    "missing_video_comment": "\u7f3a\u5c11\u89c6\u9891\u6216\u8bc4\u8bba\u4fe1\u606f",
}

THUMB = (
    "M7 10v12M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.67 22H4"
    "a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2a3.13 3.13 0 0 1 3 3.88Z"
)

def p(*parts):
    return "".join(parts)

out = p(
    """<template>
  <CreatorShell>
    <div class="cm-wrap">
      <motion class="cm-panel">
        <div class="cm-head-row">
          <div class="cm-top-tabs">
            <button
              type="button"
              class="cm-top-tab"
              :class="{ on: primaryTab === 'visible' }"
              @click="setPrimaryTab('visible')"
            >""",
    S["tab_visible"],
    """</button>
            <button
              type="button"
              class="cm-top-tab"
              :class="{ on: primaryTab === 'pending' }"
              @click="setPrimaryTab('pending')"
            >""",
    S["tab_pending"],
    """</button>
          </div>
          <div v-if="primaryTab === 'visible'" class="cm-head-search">
            <div class="cm-search">
              <input
                v-model.trim="searchQ"
                type="search"
                class="cm-search-input"
                placeholder=\"""",
    S["search_ph"],
    """\"
                autocomplete="off"
                @keydown.enter.prevent="onSearch"
              />
              <button
                type="button"
                class="cm-search-btn"
                aria-label=\"""",
    S["search_aria"],
    """\"
                @click="onSearch"
              />
            </div>
          </div>
        </div>

        <div class="cm-sub-row">
          <div class="cm-sub-tabs">
            <button
              type="button"
              class="cm-sub-tab"
              :class="{ on: mediaTab === 'video' }"
              @click="setMediaTab('video')"
            >""",
    S["sub_video"],
    """</button>
            <button
              type="button"
              class="cm-sub-tab"
              :class="{ on: mediaTab === 'article' }"
              @click="setMediaTab('article')"
            >""",
    S["sub_article"],
    """</button>
            <button
              v-if="primaryTab === 'pending'"
              type="button"
              class="cm-sub-tab"
              :class="{ on: mediaTab === 'dynamic' }"
              @click="setMediaTab('dynamic')"
            >""",
    S["sub_dynamic"],
    """</button>
          </div>
          <div v-if="primaryTab === 'visible' && mediaTab === 'video'" class="cm-sub-filters">
            <CmVideoPicker
              v-model="videoFilter"
              :options="videoOptions"
              :all-label=""",
    json.dumps(S["all_videos"]),
    """
              @change="onFilterChange"
            />
          </div>
          <div v-else-if="primaryTab === 'pending'" class="cm-sub-filters">
            <select
              v-model="pendingStatus"
              class="cm-filter-select"
              @change="onPendingFilterChange"
            >
              <option value="unprocessed">""",
    S["pending_unprocessed"],
    """</option>
              <option value="ignored">""",
    S["pending_ignored"],
    """</option>
            </select>
            <CmVideoPicker
              v-model="videoFilter"
              :options="videoOptions"
              :all-label=""",
    json.dumps(S["all_videos"]),
    """
              @change="onPendingFilterChange"
            />
            <select
              v-model="pendingScope"
              class="cm-filter-select"
              @change="onPendingFilterChange"
            >
              <option value="all">""",
    S["pending_all"],
    """</option>
              <option value="root">""",
    S["scope_root"],
    """</option>
              <option value="reply">""",
    S["scope_reply"],
    """</option>
            </select>
          </div>
        </motion>

        <div class="cm-action-row">
          <div class="cm-action-left">
            <label class="cm-check-all">
              <input
                v-model="selectAll"
                type="checkbox"
                class="cm-check"
                :disabled="!canSelectRows"
                @change="onSelectAllChange"
              />
              <span>""",
    S["select_all"],
    """</span>
            </label>
            <template v-if="primaryTab === 'visible'">
              <button
                type="button"
                class="cm-tool-btn"
                :disabled="!selectedIds.length"
                @click="onReport"
              >""",
    S["report"],
    """</button>
              <button
                type="button"
                class="cm-tool-btn cm-tool-btn--danger"
                :disabled="!selectedIds.length || deleting"
                @click="onBatchDelete"
              >
                {{ deleting ? \"""",
    S["deleting"],
    """\" : \"""",
    S["delete"],
    """\" }}
              </button>
            </template>
            <template v-else>
              <button
                type="button"
                class="cm-tool-btn"
                :disabled="!selectedIds.length || pendingAction"
                @click="onPendingPick"
              >{{ pendingAction ? \"""",
    S["acting"],
    """\" : \"""",
    S["pick"],
    """\" }}</button>
              <button
                type="button"
                class="cm-tool-btn"
                :disabled="!selectedIds.length || pendingAction"
                @click="onPendingIgnore"
              >{{ pendingAction ? \"""",
    S["acting"],
    """\" : \"""",
    S["ignore"],
    """\" }}</button>
              <button
                type="button"
                class="cm-tool-btn cm-tool-btn--danger"
                :disabled="!selectedIds.length || deleting"
                @click="onBatchDelete"
              >
                {{ deleting ? \"""",
    S["deleting"],
    """\" : \"""",
    S["delete"],
    """\" }}
              </button>
            </template>
          </div>
          <div class="cm-sort-row">
            <button
              v-for="opt in sortOptions"
              :key="opt.value"
              type="button"
              class="cm-sort-link"
              :class="{ on: sortKey === opt.value }"
              @click="setSort(opt.value)"
            >
              {{ opt.label }}
            </button>
          </div>
        </div>

        <p v-if="loadError" class="cm-hint cm-hint--err">{{ loadError }}</p>
        <p v-else-if="loading && showListArea && !rows.length" class="cm-hint">""",
    S["loading"],
    """</p>

        <div v-else-if="showCommentList" class="cm-list">
          <motion v-for="row in rows" :key="row.id" class="cm-row">
            <label class="cm-row-check">
              <input
                v-model="selectedIds"
                type="checkbox"
                class="cm-check"
                :value="row.id"
              />
            </label>
            <router-link
              v-if="userSpaceTo(row)"
              class="cm-row-avatar-link"
              :to="userSpaceTo(row)"
            >
              <img
                class="cm-row-avatar"
                :src="row.avatar_url || defaultAvatar"
                :alt="row.username"
                width="40"
                height="40"
              />
            </router-link>
            <img
              v-else
              class="cm-row-avatar"
              :src="row.avatar_url || defaultAvatar"
              :alt="row.username"
              width="40"
              height="40"
            />
            <div class="cm-row-main">
              <div class="cm-row-body">
              <div class="cm-row-user-line">
                <router-link
                  v-if="userSpaceTo(row)"
                  class="cm-row-user cm-row-user--link"
                  :to="userSpaceTo(row)"
                >
                  {{ row.username || \"""",
    S["user"],
    """\" }}
                </router-link>
                <span v-else class="cm-row-user">{{ row.username || \"""",
    S["user"],
    """\" }}</span>
              </div>
              <a
                v-if="videoHref(row)"
                class="cm-row-content-link"
                :href="videoHref(row)"
                target="_blank"
                rel="noopener noreferrer"
              >
                <p class="cm-row-content">{{ row.content }}</p>
              </a>
              <p v-else class="cm-row-content">{{ row.content }}</p>
              <div class="cm-row-meta">
                <span class="cm-row-time">{{ row.created_at }}</span>
                <VdCommentThumbBtn
                  variant="like"
                  :active="!!row.liked_by_me"
                  :count="row.like_count || 0"
                  @click="toggleRowLike(row)"
                />
                <button type="button" class="cm-row-reply-btn" @click.stop="onToggleReply(row)">
                  """,
    S["reply"],
    """
                </button>
                <button
                  type="button"
                  class="cm-row-del-btn"
                  @click.stop="onRowDelete(row)"
                >
                  """,
    S["delete"],
    """
                </button>
              </div>
              </div>
              <div
                v-if="replyCommentId === row.id"
                class="cm-reply-compose"
                @click.stop
              >
                <MbReplyComposerInner
                  :draft="replyDraft"
                  :avatar-src="selfAvatarSrc"
                  :input-placeholder="replyPlaceholder(row)"
                  :posting="replyPosting"
                  @update:draft="replyDraft = $event"
                  @submit="submitReply(row)"
                />
              </div>
            </div>
            <a
              v-if="videoHref(row)"
              class="cm-row-video"
              :href="videoHref(row)"
              target="_blank"
              rel="noopener noreferrer"
            >
              <img
                class="cm-row-video-cover"
                :src="row.video.cover_url || defaultCover"
                :alt="row.video.title"
              />
              <span class="cm-row-video-title">{{ row.video.title }}</span>
            </a>
          </div>
        </div>

        <div v-else-if="showEmpty" class="cm-empty">
          <div
            v-if="emptyUseImage"
            class="cm-empty-img"
          >
            <img
              :src="articleEmptyImg"
              alt=""
              width="360"
              height="auto"
            />
          </div>
          <p class="cm-empty-txt">{{ emptyText }}</p>
        </div>

        <div v-if="showListArea && total > 0" class="cm-foot">
          <p class="cm-foot-note">""",
    S["foot_note"],
    """</p>
          <div class="cm-pager">
            <button
              type="button"
              class="cm-page-btn"
              :disabled="page <= 1 || loading"
              @click="goPage(page - 1)"
            >""",
    S["prev"],
    """</button>
            <button
              v-for="p in pageNums"
              :key="'p-' + p"
              type="button"
              class="cm-page-num"
              :class="{ on: p === page }"
              @click="goPage(p)"
            >
              {{ p }}
            </button>
            <button
              type="button"
              class="cm-page-btn"
              :disabled="page >= totalPages || loading"
              @click="goPage(page + 1)"
            >""",
    S["next"],
    """</button>
            <span class="cm-page-summary">\u5171{{ totalPages }}\u9875/{{ total }}\u4e2a</span>
          </div>
        </div>
      </div>
    </div>
  </CreatorShell>
</template>

<script>
import { ElMessage, ElMessageBox } from "element-plus";
import { createNamespacedHelpers } from "vuex";
import "@/styles/cm-del-msgbox.scss";
import CreatorShell from "@/components/creator/CreatorShell.vue";
import CmVideoPicker from "@/components/creator/CmVideoPicker.vue";
import VdCommentThumbBtn from "@/components/comment/VdCommentThumbBtn.vue";
import MbReplyComposerInner from "@/pages/minibili/MbReplyComposerInner.vue";
import defaultAvatar from "@/assets/akari.jpg";
import defaultCover from "@/assets/akari.jpg";
import articleEmptyImg from "@/assets/upload_manager/image_text/empty.9e92c422.png";
import {
  mbApproveComment,
  mbDeleteComment,
  mbIgnoreCuratedComment,
  mbListCreatorComments,
  mbListMyVideos,
  mbPostComment,
  mbToggleLike
} from "@/api/minibili";
import { getAccessToken } from "@/utils/authTokens";
import { formatVideoBvid } from "@/utils/videoBvid";
import { minibiliUserSpaceRoute } from "@/utils/minibiliRoutes";

const isMinibiliMode =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

const { mapState } = createNamespacedHelpers("login");

export default {
  name: "CreatorCommentManage",
  components: {
    CreatorShell,
    CmVideoPicker,
    VdCommentThumbBtn,
    MbReplyComposerInner
  },
  data() {
    return {
      defaultAvatar,
      defaultCover,
      articleEmptyImg,
      primaryTab: "visible",
      mediaTab: "video",
      pendingStatus: "unprocessed",
      pendingScope: "all",
      loading: false,
      deleting: false,
      pendingAction: false,
      loadError: "",
      rows: [],
      page: 1,
      pageSize: 10,
      total: 0,
      totalPages: 0,
      sortKey: "recent",
      searchQ: "",
      searchApplied: "",
      videoFilter: "",
      videoOptions: [],
      selectedIds: [],
      selectAll: false,
      visibleSortOptions: [
        { value: "recent", label: \"""",
    S["sort_recent"],
    """\" },
        { value: "likes", label: \"""",
    S["sort_likes"],
    """\" },
        { value: "replies", label: \"""",
    S["sort_replies"],
    """\" }
      ],
      pendingSortOptions: [
        { value: "recent", label: \"""",
    S["sort_recent"],
    """\" },
        { value: "earliest", label: \"""",
    S["sort_earliest"],
    """\" }
      ],
      replyCommentId: null,
      replyDraft: "",
      replyPosting: false
    };
  },
  computed: {
    ...mapState({
      minibiliMe: (s) => s.minibiliMe,
      proInfo: (s) => s.proInfo
    }),
    isMinibiliMode() {
      return isMinibiliMode;
    },
    selfAvatarSrc() {
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
      return this.defaultAvatar;
    },
    sortOptions() {
      return this.primaryTab === "pending"
        ? this.pendingSortOptions
        : this.visibleSortOptions;
    },
    showListArea() {
      return this.mediaTab === "video" && (this.primaryTab === "visible" || this.primaryTab === "pending");
    },
    showCommentList() {
      return this.showListArea && this.rows.length > 0 && !this.loadError;
    },
    canSelectRows() {
      return this.showCommentList;
    },
    showEmpty() {
      if (this.loadError) return false;
      if (this.loading && this.showListArea) return false;
      if (!this.isMinibiliMode) return true;
      if (this.mediaTab !== "video") return true;
      return !this.rows.length;
    },
    emptyUseImage() {
      if (!this.isMinibiliMode) return false;
      return (
        this.mediaTab === "article" ||
        this.mediaTab === "dynamic" ||
        this.mediaTab === "video"
      );
    },
    emptyText() {
      if (!this.isMinibiliMode) return \"""",
    S["empty_hint"],
    """\";
      if (this.primaryTab === "pending") return \"""",
    S["empty_pending"],
    """\";
      if (this.mediaTab === "article" || this.mediaTab === "dynamic") return \"""",
    S["empty_pending"],
    """\";
      return \"""",
    S["empty"],
    """\";
    },
    pageNums() {
      const max = Math.min(this.totalPages, 7);
      if (max <= 0) return [];
      const start = Math.max(1, Math.min(this.page - 2, this.totalPages - max + 1));
      const out = [];
      for (let i = 0; i < max; i += 1) {
        out.push(start + i);
      }
      return out;
    }
  },
  watch: {
    selectedIds() {
      this.selectAll =
        this.canSelectRows &&
        this.selectedIds.length === this.rows.length &&
        this.rows.length > 0;
    },
    primaryTab() {
      if (this.primaryTab === "pending" && (this.sortKey === "likes" || this.sortKey === "replies")) {
        this.sortKey = "recent";
      }
      if (this.primaryTab === "visible" && this.sortKey === "earliest") {
        this.sortKey = "recent";
      }
    }
  },
  mounted() {
    if (this.isMinibiliMode && getAccessToken()) {
      void this.loadVideos();
      void this.fetchList();
    }
  },
  methods: {
    setPrimaryTab(tab) {
      if (this.primaryTab === tab) return;
      this.primaryTab = tab;
      this.selectedIds = [];
      this.selectAll = false;
      this.loadError = "";
      this.page = 1;
      if (tab === "visible") {
        this.pendingStatus = "unprocessed";
      }
      if (this.showListArea) {
        void this.fetchList();
      } else {
        this.rows = [];
        this.total = 0;
        this.totalPages = 0;
      }
    },
    setMediaTab(tab) {
      if (this.mediaTab === tab) return;
      this.mediaTab = tab;
      this.selectedIds = [];
      this.selectAll = false;
      this.page = 1;
      if (this.showListArea) {
        void this.fetchList();
      } else {
        this.rows = [];
        this.total = 0;
        this.totalPages = 0;
      }
    },
    onPendingFilterChange() {
      this.page = 1;
      if (this.showListArea) {
        void this.fetchList();
      }
    },
    async confirmDeleteDialog() {
      await ElMessageBox.confirm(
        \"""",
    S["del_reminder_msg"],
    """\",
        \"""",
    S["del_reminder_title"],
    """\",
        {
          confirmButtonText: \"""",
    S["confirm_ok"],
    """\",
          cancelButtonText: \"""",
    S["cancel"],
    """\",
          customClass: "cm-del-msgbox",
          confirmButtonClass: "cm-del-msgbox__ok",
          cancelButtonClass: "cm-del-msgbox__cancel",
          showClose: true,
          distinguishCancelAndClose: true
        }
      );
    },
    userSpaceTo(row) {
      if (!row || !row.user_id) return null;
      return minibiliUserSpaceRoute(row.user_id);
    },
    videoLink(row) {
      if (!row || !row.video_id) return null;
      return {
        name: "video",
        params: { aid: formatVideoBvid(row.video_id) },
        query: { mb_cid: String(row.id) }
      };
    },
    videoHref(row) {
      const to = this.videoLink(row);
      if (!to) return "";
      return this.$router.resolve(to).href;
    },
    replyPlaceholder(row) {
      const name = String((row && row.username) || "").trim() || \"""",
    S["user"],
    """\";
      return `\u56de\u590d @${name} :`;
    },
    onToggleReply(row) {
      if (!getAccessToken()) {
        ElMessage.warning(\"""",
    S["login_warn"],
    """\");
        return;
      }
      const id = Number(row && row.id) || 0;
      if (!id) return;
      if (this.replyCommentId === id) {
        this.closeReplyComposer();
        return;
      }
      this.replyCommentId = id;
      this.replyDraft = "";
    },
    closeReplyComposer() {
      this.replyCommentId = null;
      this.replyDraft = "";
      this.replyPosting = false;
    },
    patchRowById(commentId, patchFn) {
      const id = Number(commentId) || 0;
      if (!id) return;
      const ix = this.rows.findIndex((x) => Number(x.id) === id);
      if (ix < 0) return;
      this.rows.splice(ix, 1, patchFn({ ...this.rows[ix] }));
    },
    applyLocalLikeState(item, liked) {
      const wasLiked = !!item.liked_by_me;
      let likeCount = Number(item.like_count) || 0;
      if (liked && !wasLiked) likeCount += 1;
      if (!liked && wasLiked) likeCount = Math.max(0, likeCount - 1);
      return {
        ...item,
        liked_by_me: liked,
        like_count: likeCount
      };
    },
    async toggleRowLike(row) {
      if (!getAccessToken()) {
        ElMessage.warning(\"""",
    S["login_warn"],
    """\");
        return;
      }
      const id = Number(row && row.id) || 0;
      if (!id) return;
      try {
        const { liked } = await mbToggleLike(id);
        this.patchRowById(id, (item) => this.applyLocalLikeState(item, !!liked));
      } catch (e) {
        ElMessage.error((e && e.message) || \"""",
    S["like_fail"],
    """\");
      }
    },
    async submitReply(row) {
      const vid = Number(row && row.video_id) || 0;
      const parentId = Number(row && row.id) || 0;
      if (!vid || !parentId) {
        ElMessage.warning(\"""",
    S["missing_video_comment"],
    """\");
        return;
      }
      const content = String(this.replyDraft || "").trim();
      if (!content) return;
      this.replyPosting = true;
      try {
        await mbPostComment(vid, content, parentId);
        ElMessage.success(\"""",
    S["post_ok"],
    """\");
        this.closeReplyComposer();
        await this.fetchList();
      } catch (e) {
        ElMessage.error((e && e.message) || \"""",
    S["post_fail"],
    """\");
      } finally {
        this.replyPosting = false;
      }
    },
    async loadVideos() {
      try {
        const items = [];
        let cursor = "";
        for (let i = 0; i < 20; i += 1) {
          const res = await mbListMyVideos(cursor ? { cursor } : undefined);
          items.push(...(res.items || []));
          if (!res.next_cursor) break;
          cursor = res.next_cursor;
        }
        this.videoOptions = items;
      } catch {
        this.videoOptions = [];
      }
    },
    async fetchList() {
      if (!this.showListArea || !this.isMinibiliMode || !getAccessToken()) return;
      this.loading = true;
      this.loadError = "";
      try {
        const params = {
          page: this.page,
          page_size: this.pageSize,
          sort: this.sortKey
        };
        if (this.searchApplied) params.q = this.searchApplied;
        if (this.videoFilter) params.video_id = Number(this.videoFilter);
        if (this.primaryTab === "pending") {
          params.pending = 1;
          params.pending_status = this.pendingStatus;
          params.scope = this.pendingScope;
        }
        const res = await mbListCreatorComments(params);
        this.rows = res.items || [];
        this.total = Number(res.total) || 0;
        this.totalPages = Number(res.total_pages) || 0;
        this.selectedIds = [];
        this.selectAll = false;
      } catch (e) {
        this.loadError = (e && e.message) || \"""",
    S["load_fail"],
    """\";
        this.rows = [];
      } finally {
        this.loading = false;
      }
    },
    onSearch() {
      this.searchApplied = this.searchQ;
      this.page = 1;
      void this.fetchList();
    },
    onFilterChange() {
      this.page = 1;
      void this.fetchList();
    },
    setSort(key) {
      if (this.sortKey === key) return;
      this.sortKey = key;
      if (this.showListArea) {
        this.page = 1;
        void this.fetchList();
      }
    },
    goPage(p) {
      if (p < 1 || p > this.totalPages || p === this.page) return;
      this.page = p;
      void this.fetchList();
    },
    onSelectAllChange() {
      if (this.selectAll) {
        this.selectedIds = this.rows.map((r) => r.id);
      } else {
        this.selectedIds = [];
      }
    },
    onReport() {
      ElMessage.info(\"""",
    S["report_soon"],
    """\");
    },
    async onPendingPick() {
      if (!this.selectedIds.length || this.pendingAction) return;
      this.pendingAction = true;
      const ids = [...this.selectedIds];
      let ok = 0;
      try {
        for (const id of ids) {
          await mbApproveComment(Number(id));
          ok += 1;
        }
        ElMessage.success(\"""",
    S["pick_ok"],
    """\");
        this.selectedIds = [];
        if (this.rows.length <= ids.length && this.page > 1) {
          this.page -= 1;
        }
        await this.fetchList();
      } catch (e) {
        if (ok > 0) {
          await this.fetchList();
        }
        ElMessage.error((e && e.message) || \"""",
    S["pick_fail"],
    """\");
      } finally {
        this.pendingAction = false;
      }
    },
    async onPendingIgnore() {
      if (!this.selectedIds.length || this.pendingAction) return;
      this.pendingAction = true;
      const ids = [...this.selectedIds];
      let ok = 0;
      try {
        for (const id of ids) {
          await mbIgnoreCuratedComment(Number(id));
          ok += 1;
        }
        ElMessage.success(\"""",
    S["ignore_ok"],
    """\");
        this.selectedIds = [];
        if (this.rows.length <= ids.length && this.page > 1) {
          this.page -= 1;
        }
        await this.fetchList();
      } catch (e) {
        if (ok > 0) {
          await this.fetchList();
        }
        ElMessage.error((e && e.message) || \"""",
    S["ignore_fail"],
    """\");
      } finally {
        this.pendingAction = false;
      }
    },
    async onRowDelete(row) {
      if (!row || !row.id) return;
      try {
        await this.confirmDeleteDialog();
      } catch {
        return;
      }
      this.deleting = true;
      try {
        await mbDeleteComment(row.id);
        ElMessage.success(\"""",
    S["del_done"],
    """\");
        if (this.rows.length <= 1 && this.page > 1) {
          this.page -= 1;
        }
        await this.fetchList();
      } catch (e) {
        ElMessage.error((e && e.message) || \"""",
    S["del_fail"],
    """\");
      } finally {
        this.deleting = false;
      }
    },
    async onBatchDelete() {
      if (!this.selectedIds.length) return;
      try {
        await this.confirmDeleteDialog();
      } catch {
        return;
      }
      this.deleting = true;
      const ids = [...this.selectedIds];
      try {
        for (const id of ids) {
          await mbDeleteComment(id);
        }
        ElMessage.success(\"""",
    S["del_done"],
    """\");
        if (this.rows.length <= ids.length && this.page > 1) {
          this.page -= 1;
        }
        await this.fetchList();
      } catch (e) {
        ElMessage.error((e && e.message) || \"""",
    S["del_fail"],
    """\");
      } finally {
        this.deleting = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "./commentManage.scss";
</style>
""",
)

out = out.replace("<motion ", "<div ").replace("</motion>", "</div>")
TARGET.write_text(out, encoding="utf-8")
print("OK", TARGET)
