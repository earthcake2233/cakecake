<template>
  <CreatorShell>
    <div class="cm-wrap">
      <div class="cm-panel">
        <div class="cm-head-row">
          <div class="cm-top-tabs">
            <button
              type="button"
              class="cm-top-tab"
              :class="{ on: primaryTab === 'visible' }"
              @click="setPrimaryTab('visible')"
            >用户可见评论</button>
            <button
              type="button"
              class="cm-top-tab"
              :class="{ on: primaryTab === 'pending' }"
              @click="setPrimaryTab('pending')"
            >待精选评论</button>
          </div>
          <div v-if="primaryTab === 'visible'" class="cm-head-search">
            <div class="cm-search">
              <input
                v-model.trim="searchQ"
                type="search"
                class="cm-search-input"
                :placeholder="searchPlaceholder"
                autocomplete="off"
                @keydown.enter.prevent="onSearch"
              />
              <button
                type="button"
                class="cm-search-btn"
                aria-label="搜索"
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
            >视频评论</button>
            <button
              type="button"
              class="cm-sub-tab"
              :class="{ on: mediaTab === 'article' }"
              @click="setMediaTab('article')"
            >专栏评论</button>
            <button
              v-if="primaryTab === 'pending'"
              type="button"
              class="cm-sub-tab"
              :class="{ on: mediaTab === 'dynamic' }"
              @click="setMediaTab('dynamic')"
            >动态评论</button>
          </div>
          <div v-if="primaryTab === 'visible' && mediaTab === 'video'" class="cm-sub-filters">
            <CmVideoPicker
              v-model="videoFilter"
              :options="videoOptions"
              all-label="全部视频"
              @change="onFilterChange"
            />
          </div>
          <div v-else-if="primaryTab === 'visible' && mediaTab === 'article'" class="cm-sub-filters">
            <CmVideoPicker
              v-model="articleFilter"
              :options="articleOptions"
              all-label="全部专栏"
              search-placeholder="输入专栏搜索关键字"
              @change="onFilterChange"
            />
          </div>
          <div v-else-if="primaryTab === 'pending'" class="cm-sub-filters">
            <select
              v-model="pendingStatus"
              class="cm-filter-select"
              @change="onPendingFilterChange"
            >
              <option value="unprocessed">未处理</option>
              <option value="ignored">已忽略</option>
            </select>
            <CmVideoPicker
              v-if="mediaTab === 'video'"
              v-model="videoFilter"
              :options="videoOptions"
              all-label="全部视频"
              @change="onPendingFilterChange"
            />
            <CmVideoPicker
              v-else-if="mediaTab === 'article'"
              v-model="articleFilter"
              :options="articleOptions"
              all-label="全部专栏"
              search-placeholder="输入专栏搜索关键字"
              @change="onPendingFilterChange"
            />
            <select
              v-model="pendingScope"
              class="cm-filter-select"
              @change="onPendingFilterChange"
            >
              <option value="all">全部</option>
              <option value="root">根评论</option>
              <option value="reply">回复评论</option>
            </select>
          </div>
        </div>

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
              <span>全选</span>
            </label>
            <template v-if="primaryTab === 'visible'">
              <button
                type="button"
                class="cm-tool-btn"
                :disabled="!selectedIds.length"
                @click="onReport"
              >举报</button>
              <button
                type="button"
                class="cm-tool-btn cm-tool-btn--danger"
                :disabled="!selectedIds.length || deleting"
                @click="onBatchDelete"
              >
                {{ deleting ? "删除中…" : "删除" }}
              </button>
            </template>
            <template v-else>
              <button
                type="button"
                class="cm-tool-btn"
                :disabled="pendingPickDisabled"
                @click="onPendingPick"
              >{{ pendingAction ? "处理中…" : "精选" }}</button>
              <button
                type="button"
                class="cm-tool-btn"
                :class="{ 'cm-tool-btn--muted': pendingIgnoreDisabled }"
                :disabled="pendingIgnoreDisabled"
                @click="onPendingIgnore"
              >{{ pendingAction ? "处理中…" : "忽略" }}</button>
              <button
                type="button"
                class="cm-tool-btn cm-tool-btn--danger"
                :disabled="!selectedIds.length || deleting"
                @click="onBatchDelete"
              >
                {{ deleting ? "删除中…" : "删除" }}
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

        <div v-else-if="showCommentList" class="cm-list">
          <div v-for="row in rows" :key="row.id" class="cm-row">
            <label class="cm-row-check">
              <input
                v-model="selectedIds"
                type="checkbox"
                class="cm-check"
                :value="commentRowId(row)"
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
                  {{ row.username || "用户" }}
                </router-link>
                <span v-else class="cm-row-user">{{ row.username || "用户" }}</span>
              </div>
              <a
                v-if="contentHref(row)"
                class="cm-row-content-link"
                :href="contentHref(row)"
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
                  回复
                </button>
                <button
                  type="button"
                  class="cm-row-del-btn"
                  @click.stop="onRowDelete(row)"
                >
                  删除
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
              v-if="contentHref(row) && rowMedia(row)"
              class="cm-row-video"
              :href="contentHref(row)"
              target="_blank"
              rel="noopener noreferrer"
            >
              <img
                class="cm-row-video-cover"
                :src="rowMedia(row).cover_url || defaultCover"
                :alt="rowMedia(row).title"
              />
              <span class="cm-row-video-title">{{ rowMedia(row).title }}</span>
            </a>
          </div>
        </div>

        <p v-else-if="loading && showListArea" class="cm-hint">加载中…</p>

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
          <p class="cm-foot-note">仅展示最近的50000条评论</p>
          <div class="cm-pager">
            <button
              type="button"
              class="cm-page-btn"
              :disabled="page <= 1 || loading"
              @click="goPage(page - 1)"
            >上一页</button>
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
            >下一页</button>
            <span class="cm-page-summary">共{{ totalPages }}页/{{ total }}个</span>
          </div>
        </div>
      </div>
    </div>
  </CreatorShell>
</template>

<script>
import { h } from "vue";
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
  mbApproveArticleComment,
  mbApproveDynamicComment,
  mbDeleteArticleComment,
  mbDeleteComment,
  mbDeleteDynamicComment,
  mbIgnoreCuratedComment,
  mbIgnoreCuratedArticleComment,
  mbIgnoreCuratedDynamicComment,
  mbListCreatorComments,
  mbListMyArticles,
  mbListMyVideos,
  mbPostArticleComment,
  mbPostComment,
  mbToggleArticleCommentLike,
  mbToggleDynamicCommentLike,
  mbToggleLike
} from "@/api/minibili";
import { getAccessToken } from "@/utils/authTokens";
import { clearStuckPageOverlays } from "@/utils/clearPageOverlays";
import { formatVideoBvid } from "@/utils/videoBvid";
import {
  minibiliArticleReadRoute,
  minibiliDynamicReadRoute,
  minibiliUserSpaceRoute
} from "@/utils/minibiliRoutes";

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
      articleFilter: "",
      articleOptions: [],
      dynamicFilter: "",
      dynamicOptions: [],
      selectedIds: [],
      selectAll: false,
      visibleSortOptions: [
        { value: "recent", label: "最近发布" },
        { value: "likes", label: "点赞最多" },
        { value: "replies", label: "回复最多" }
      ],
      pendingSortOptions: [
        { value: "recent", label: "最近发布" },
        { value: "earliest", label: "最早发布" }
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
    searchPlaceholder() {
      if (this.mediaTab === "article") return "搜索专栏评论";
      if (this.mediaTab === "dynamic") return "搜索动态评论";
      return "搜索视频评论";
    },
    showListArea() {
      if (this.mediaTab === "dynamic") {
        return this.primaryTab === "pending";
      }
      if (this.mediaTab === "article" || this.mediaTab === "video") {
        return this.primaryTab === "visible" || this.primaryTab === "pending";
      }
      return false;
    },
    showCommentList() {
      return this.showListArea && this.rows.length > 0 && !this.loadError;
    },
    canSelectRows() {
      return this.showCommentList;
    },
    showEmpty() {
      if (this.loadError) return false;
      if (this.loading) return false;
      if (this.rows.length > 0) return false;
      if (!this.isMinibiliMode) return true;
      if (!this.showListArea) return true;
      return true;
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
      if (!this.isMinibiliMode) return "请登录后查看";
      if (this.searchApplied) {
        return this.mediaTab === "article" ? "没有匹配的专栏评论" : "没有匹配的评论";
      }
      if (this.primaryTab === "pending") {
        if (this.pendingStatus === "ignored") return "暂无已忽略评论";
        return "还没有待精选评论哦~";
      }
      if (this.mediaTab === "article" || this.mediaTab === "dynamic") return "还没有评论哦~";
      return "暂无评论";
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
    },
    selectedPendingRows() {
      const ids = new Set(
        (this.selectedIds || []).map((id) => Number(id)).filter((n) => n > 0)
      );
      if (!ids.size) return [];
      return this.rows.filter((r) => ids.has(this.commentRowId(r)));
    },
    pendingPickDisabled() {
      if (this.pendingAction || this.deleting) return true;
      return this.selectedPendingRows.length === 0;
    },
    pendingIgnoreDisabled() {
      if (this.pendingAction || this.deleting) return true;
      if (this.pendingStatus === "ignored") return true;
      if (this.selectedPendingRows.length === 0) return true;
      return this.selectedPendingRows.every((r) => !!r.curated_ignored);
    }
  },
  watch: {
    selectedIds() {
      this.selectAll =
        this.canSelectRows &&
        this.selectedIds.length === this.rows.length &&
        this.rows.length > 0;
    },
    searchQ(val) {
      const q = String(val || "").trim();
      if (!q && this.searchApplied) {
        this.searchApplied = "";
        if (this.showListArea && this.isMinibiliMode && getAccessToken()) {
          this.page = 1;
          void this.fetchList();
        }
      }
    },
    primaryTab(tab) {
      if (tab === "pending" && (this.sortKey === "likes" || this.sortKey === "replies")) {
        this.sortKey = "recent";
      }
      if (tab === "visible" && this.sortKey === "earliest") {
        this.sortKey = "recent";
      }
      if (tab === "visible" && this.mediaTab === "dynamic") {
        this.mediaTab = "video";
        this.rows = [];
        this.total = 0;
        this.totalPages = 0;
      }
    }
  },
  mounted() {
    clearStuckPageOverlays();
    this.loading = false;
    void this.$nextTick(() => this.onPageEnter());
  },
  beforeUnmount() {
    clearStuckPageOverlays();
  },
  methods: {
    onPageEnter() {
      clearStuckPageOverlays();
      this.applyRouteQuery();
      if (!this.isMinibiliMode || !getAccessToken()) {
        this.loading = false;
        this.loadError = "";
        return;
      }
      void this.loadVideos();
      void this.loadArticles();
      void this.fetchList();
    },
    applyRouteQuery() {
      const q = this.$route && this.$route.query;
      if (!q) return;
      if (String(q.tab || "") === "pending" || String(q.pending || "") === "1") {
        this.primaryTab = "pending";
      }
      const media = String(q.media || "").trim();
      if (media === "article" || media === "video" || media === "dynamic") {
        this.mediaTab = media;
      }
      const vid = String(q.video_id || "").trim();
      if (vid) this.videoFilter = vid;
      const aid = String(q.article_id || "").trim();
      if (aid) this.articleFilter = aid;
      const did = String(q.dynamic_id || "").trim();
      if (did) this.dynamicFilter = did;
    },
    syncSearchFromInput() {
      this.searchApplied = String(this.searchQ || "").trim();
    },
    setPrimaryTab(tab) {
      if (this.primaryTab === tab) {
        if (tab === "visible" && this.showListArea && this.isMinibiliMode && getAccessToken()) {
          this.syncSearchFromInput();
          void this.fetchList();
        }
        return;
      }
      this.primaryTab = tab;
      this.selectedIds = [];
      this.selectAll = false;
      this.loadError = "";
      this.page = 1;
      if (tab === "visible") {
        this.pendingStatus = "unprocessed";
      }
      if (this.showListArea) {
        if (tab === "visible") this.syncSearchFromInput();
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
      this.searchApplied = "";
      this.searchQ = "";
      if (tab === "article" && (this.sortKey === "earliest")) {
        this.sortKey = "recent";
      }
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
        h("div", { class: "cm-del-msgbox-body" }, [
          h("p", { class: "cm-del-msgbox-msg" }, "删除后无法恢复，确认删除选中的评论吗？")
        ]),
        "删除提醒",
        {
          confirmButtonText: "确定",
          cancelButtonText: "取消",
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
    rowMedia(row) {
      if (!row) return null;
      if (this.mediaTab === "article") return row.article || null;
      if (this.mediaTab === "dynamic") return row.dynamic || null;
      return row.video || null;
    },
    videoLink(row) {
      if (!row || !row.video_id) return null;
      return {
        name: "video",
        params: { aid: formatVideoBvid(row.video_id) },
        query: { mb_cid: String(row.id) }
      };
    },
    articleLink(row) {
      if (!row || !row.article_id) return null;
      return minibiliArticleReadRoute(row.article_id);
    },
    dynamicLink(row) {
      if (!row || !row.dynamic_id) return null;
      return minibiliDynamicReadRoute(row.dynamic_id);
    },
    contentLink(row) {
      if (this.mediaTab === "article") return this.articleLink(row);
      if (this.mediaTab === "dynamic") return this.dynamicLink(row);
      return this.videoLink(row);
    },
    contentHref(row) {
      const to = this.contentLink(row);
      if (!to) return "";
      return this.$router.resolve(to).href;
    },
    replyPlaceholder(row) {
      const name = String((row && row.username) || "").trim() || "用户";
      return `回复 @${name} :`;
    },
    onToggleReply(row) {
      if (!getAccessToken()) {
        ElMessage.warning("请先登录");
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
        ElMessage.warning("请先登录");
        return;
      }
      const id = Number(row && row.id) || 0;
      if (!id) return;
      try {
        const { liked } =
          this.mediaTab === "article"
            ? await mbToggleArticleCommentLike(id)
            : this.mediaTab === "dynamic"
              ? await mbToggleDynamicCommentLike(id)
              : await mbToggleLike(id);
        this.patchRowById(id, (item) => this.applyLocalLikeState(item, !!liked));
      } catch (e) {
        ElMessage.error((e && e.message) || "点赞失败");
      }
    },
    async submitReply(row) {
      const parentId = Number(row && row.id) || 0;
      const content = String(this.replyDraft || "").trim();
      if (!parentId || !content) return;
      this.replyPosting = true;
      try {
        if (this.mediaTab === "article") {
          const aid = Number(row && row.article_id) || 0;
          if (!aid) {
            ElMessage.warning("缺少专栏或评论信息");
            return;
          }
          await mbPostArticleComment(aid, content, parentId);
        } else {
          const vid = Number(row && row.video_id) || 0;
          if (!vid) {
            ElMessage.warning("缺少视频或评论信息");
            return;
          }
          await mbPostComment(vid, content, parentId);
        }
        ElMessage.success("发表成功");
        this.closeReplyComposer();
        await this.fetchList();
      } catch (e) {
        ElMessage.error((e && e.message) || "发表失败");
      } finally {
        this.replyPosting = false;
      }
    },
    async loadVideos() {
      try {
        const items = [];
        for (let page = 1; page <= 20; page += 1) {
          const res = await mbListMyVideos({ page, page_size: 50, sort: "time" });
          items.push(...(res.items || []));
          if (page >= (res.total_pages || 1)) break;
        }
        this.videoOptions = items;
      } catch {
        this.videoOptions = [];
      }
    },
    async loadArticles() {
      try {
        const items = [];
        for (let page = 1; page <= 20; page += 1) {
          const res = await mbListMyArticles({
            page,
            page_size: 50,
            sort: "time",
            status: "passed"
          });
          items.push(...(res.items || []));
          if (page >= (res.total_pages || 1)) break;
        }
        this.articleOptions = items;
      } catch {
        this.articleOptions = [];
      }
    },
    async fetchList() {
      if (!this.showListArea || !this.isMinibiliMode || !getAccessToken()) {
        this.loading = false;
        return;
      }
      this.loading = true;
      this.loadError = "";
      try {
        const params = {
          page: this.page,
          page_size: this.pageSize,
          sort: this.sortKey
        };
        if (this.searchApplied) params.q = this.searchApplied;
        if (this.mediaTab === "article") {
          params.media = "article";
          if (this.articleFilter) params.article_id = Number(this.articleFilter);
        } else if (this.mediaTab === "dynamic") {
          params.media = "dynamic";
          if (this.dynamicFilter) params.dynamic_id = Number(this.dynamicFilter);
        } else if (this.videoFilter) {
          params.video_id = Number(this.videoFilter);
        }
        if (this.primaryTab === "pending") {
          params.pending = "1";
          params.pending_status = this.pendingStatus;
          params.scope = this.pendingScope;
        }
        const raw = await mbListCreatorComments(params);
        const payload =
          raw && Array.isArray(raw.items)
            ? raw
            : raw && raw.data && typeof raw.data === "object"
              ? raw.data
              : raw;
        const items = Array.isArray(payload?.items) ? payload.items : [];
        this.rows = items;
        this.total = Number(payload?.total) || 0;
        this.totalPages = Number(payload?.total_pages) || 0;
        this.selectedIds = [];
        this.selectAll = false;
      } catch (e) {
        this.loadError = (e && e.message) || "加载失败";
        this.rows = [];
        this.total = 0;
        this.totalPages = 0;
      } finally {
        this.loading = false;
      }
    },
    onSearch() {
      this.syncSearchFromInput();
      this.page = 1;
      void this.fetchList();
    },
    onFilterChange() {
      this.page = 1;
      void this.fetchList();
    },
    setSort(key) {
      const changed = this.sortKey !== key;
      this.sortKey = key;
      if (this.showListArea) {
        if (changed) this.page = 1;
        void this.fetchList();
      }
    },
    goPage(p) {
      if (p < 1 || p > this.totalPages || p === this.page) return;
      this.page = p;
      void this.fetchList();
    },
    commentRowId(row) {
      const n = Number(row && row.id);
      return Number.isFinite(n) && n > 0 ? n : 0;
    },
    onSelectAllChange() {
      if (this.selectAll) {
        this.selectedIds = this.rows
          .map((r) => this.commentRowId(r))
          .filter((n) => n > 0);
      } else {
        this.selectedIds = [];
      }
    },
    onReport() {
      ElMessage.info("举报功能即将开放");
    },
    async onPendingPick() {
      const targets = this.selectedPendingRows.filter((r) => !r.approved);
      if (!targets.length || this.pendingAction) return;
      this.pendingAction = true;
      let ok = 0;
      const approve =
        this.mediaTab === "article"
          ? mbApproveArticleComment
          : this.mediaTab === "dynamic"
            ? mbApproveDynamicComment
            : mbApproveComment;
      try {
        for (const row of targets) {
          await approve(this.commentRowId(row));
          ok += 1;
        }
        ElMessage.success("已精选");
        this.selectedIds = [];
        if (this.rows.length <= targets.length && this.page > 1) {
          this.page -= 1;
        }
        await this.fetchList();
      } catch (e) {
        if (ok > 0) {
          await this.fetchList();
        }
        ElMessage.error((e && e.message) || "精选失败");
      } finally {
        this.pendingAction = false;
      }
    },
    async onPendingIgnore() {
      if (this.pendingIgnoreDisabled || this.pendingAction) return;
      const targets = this.selectedPendingRows.filter((r) => !r.curated_ignored);
      if (!targets.length) return;
      this.pendingAction = true;
      let ok = 0;
      const ignore =
        this.mediaTab === "article"
          ? mbIgnoreCuratedArticleComment
          : this.mediaTab === "dynamic"
            ? mbIgnoreCuratedDynamicComment
            : mbIgnoreCuratedComment;
      try {
        for (const row of targets) {
          await ignore(this.commentRowId(row));
          ok += 1;
        }
        ElMessage.success("已忽略");
        this.selectedIds = [];
        if (this.rows.length <= targets.length && this.page > 1) {
          this.page -= 1;
        }
        await this.fetchList();
      } catch (e) {
        if (ok > 0) {
          await this.fetchList();
        }
        ElMessage.error((e && e.message) || "忽略失败");
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
        if (this.mediaTab === "article") {
          await mbDeleteArticleComment(row.id);
        } else if (this.mediaTab === "dynamic") {
          await mbDeleteDynamicComment(row.id);
        } else {
          await mbDeleteComment(row.id);
        }
        ElMessage.success("已删除");
        if (this.rows.length <= 1 && this.page > 1) {
          this.page -= 1;
        }
        await this.fetchList();
      } catch (e) {
        ElMessage.error((e && e.message) || "删除失败");
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
        const del =
          this.mediaTab === "article"
            ? mbDeleteArticleComment
            : this.mediaTab === "dynamic"
              ? mbDeleteDynamicComment
              : mbDeleteComment;
        for (const id of ids) {
          await del(id);
        }
        ElMessage.success("已删除");
        if (this.rows.length <= ids.length && this.page > 1) {
          this.page -= 1;
        }
        await this.fetchList();
      } catch (e) {
        ElMessage.error((e && e.message) || "删除失败");
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
