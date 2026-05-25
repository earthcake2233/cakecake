<template>
  <div
    v-if="commentsClosed"
    class="mb-space__dyn-cmt-panel mb-dyn-comments--closed"
    @click.stop
  >
    <div class="mb-space__dyn-cmt-head mb-space__dyn-cmt-head--bar">
      <div class="mb-space__dyn-cmt-head__left">
        <h4 class="mb-space__dyn-cmt-title">
          评论<sup class="mb-space__dyn-cmt-count">{{
            formatCount(commentTotal)
          }}</sup>
        </h4>
      </div>
      <div
        v-if="ownerCanCurate"
        class="mb-space__dyn-cmt-head__actions"
      >
        <router-link
          v-if="commentsCurated"
          class="mb-space__dyn-cmt-pending-link"
          :to="pendingCommentsRoute"
          @click.stop
        >
          待精选评论
        </router-link>
        <div
          class="mb-space__dyn-cmt-head-more-wrap vd-cmt-menu-wrap"
          :class="{ 'is-open': ownerMenuOpen }"
          @click.stop
        >
          <button
            type="button"
            class="vd-cmt-menu-trigger mb-space__dyn-cmt-head-menu-trigger"
            aria-haspopup="true"
            :aria-expanded="ownerMenuOpen"
            aria-label="评论管理"
            @click.stop="ownerMenuOpen = !ownerMenuOpen"
          >
            <span class="vd-cmt-menu-dots" aria-hidden="true">
              <span /><span /><span />
            </span>
          </button>
          <div
            v-if="ownerMenuOpen"
            class="vd-cmt-menu-dropdown"
            role="menu"
            @click.stop
          >
            <button
              type="button"
              class="vd-cmt-menu-item"
              role="menuitem"
              @click="onOwnerMenuAction('pick_comment')"
            >
              {{ commentsCurated ? "关闭评论精选" : "开启评论精选" }}
            </button>
            <button
              type="button"
              class="vd-cmt-menu-item"
              role="menuitem"
              @click="onOwnerMenuAction('close_comments')"
            >
              恢复评论
            </button>
          </div>
        </div>
      </div>
    </div>
    <div class="mb-space__dyn-cmt-closed-bar" role="status">UP主已关闭评论</div>
    <p class="mb-space__dyn-cmt-foot">已到达世界的尽头</p>
  </div>
  <div v-else class="vd-comments mb-dyn-comments" @click.stop>
    <div class="vd-cmt-head">
      <h3 class="vd-cmt-title">
        <span class="vd-cmt-count">{{ formatCount(commentTotal) }}</span> 评论
      </h3>
      <div class="vd-cmt-head-row vd-cmt-head-row--toolbar mb-dyn-cmt-toolbar">
          <div class="mb-dyn-cmt-toolbar__main">
            <div class="vd-cmt-sort">
              <span v-if="commentsCurated" class="vd-cmt-curated-label">{{
                MB_COMMENT_CURATED_LABEL
              }}</span>
              <template v-else>
                <button
                  type="button"
                  class="vd-sort-tab"
                  :class="{ on: commentSort === 'hot' }"
                  @click="commentSort = 'hot'"
                >
                  按热度排序
                </button>
                <button
                  type="button"
                  class="vd-sort-tab"
                  :class="{ on: commentSort === 'time' }"
                  @click="commentSort = 'time'"
                >
                  按时间排序
                </button>
              </template>
            </div>
            <div class="mb-dyn-cmt-toolbar__actions">
              <router-link
                v-if="ownerCanCurate && commentsCurated"
                class="mb-dyn-cmt-pending-link"
                :to="pendingCommentsRoute"
                @click.stop
              >
                待精选评论
              </router-link>
              <div
                v-if="ownerCanCurate"
                class="mb-space__dyn-cmt-head-more-wrap vd-cmt-menu-wrap"
                :class="{ 'is-open': ownerMenuOpen }"
                @click.stop
              >
                <button
                  type="button"
                  class="vd-cmt-menu-trigger mb-space__dyn-cmt-head-menu-trigger"
                  aria-haspopup="true"
                  :aria-expanded="ownerMenuOpen"
                  aria-label="评论管理"
                  @click.stop="ownerMenuOpen = !ownerMenuOpen"
                >
                  <span class="vd-cmt-menu-dots" aria-hidden="true">
                    <span /><span /><span />
                  </span>
                </button>
                <div
                  v-if="ownerMenuOpen"
                  class="vd-cmt-menu-dropdown"
                  role="menu"
                  @click.stop
                >
                  <button
                    type="button"
                    class="vd-cmt-menu-item"
                    role="menuitem"
                    @click="onOwnerMenuAction('pick_comment')"
                  >
                    {{
                      commentsCurated ? "关闭评论精选" : "开启评论精选"
                    }}
                  </button>
                  <button
                    type="button"
                    class="vd-cmt-menu-item"
                    role="menuitem"
                    @click="onOwnerMenuAction('close_comments')"
                  >
                    {{ commentsClosed ? "恢复评论" : "关闭评论" }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="vd-cmt-head-row vd-cmt-head-row--pager mb-dyn-cmt-pager">
          <div class="vd-cmt-page-top vd-cmt-page-top--links">
            <span class="vd-page-info">共{{ commentTotalPages }}页</span>
            <template
              v-for="(it, pidx) in commentPagerItems"
              :key="'dyn-cmt-page-top-' + pidx"
            >
              <span
                v-if="it.type === 'ellipsis'"
                class="vd-page-ellipsis"
                >...</span
              >
              <button
                v-else
                type="button"
                class="vd-page-num vd-page-num--link"
                :class="{ on: it.n === commentCurrentPage }"
                @click="setCommentPage(it.n)"
              >
                {{ it.n }}
              </button>
            </template>
            <button
              type="button"
              class="vd-page-next vd-page-next--link"
              :disabled="commentCurrentPage >= commentTotalPages"
              @click="setCommentPage(commentCurrentPage + 1)"
            >
              下一页
            </button>
          </div>
        </div>
    </div>

    <div class="vd-cmt-composer vd-cmt-composer--mb">
        <img
          class="vd-cmt-avatar vd-cmt-avatar--mb"
          :src="composerAvatar"
          width="48"
          height="48"
          alt=""
        />
        <div class="vd-cmt-mb-composer-main">
          <div class="vd-cmt-mb-editor-row">
            <div class="vd-cmt-uni-inbox">
              <template v-if="mbLoggedIn || commentsCurated">
                <textarea
                  v-model="commentDraft"
                  class="vd-cmt-uni-inbox__field"
                  :class="{ 'is-curated-hint': commentsCurated && !mbLoggedIn }"
                  rows="3"
                  maxlength="1000"
                  :readonly="commentsCurated && !mbLoggedIn"
                  :disabled="commentsCurated && !mbLoggedIn"
                  :placeholder="commentComposerPlaceholder"
                />
              </template>
              <div
                v-else
                class="vd-cmt-uni-inbox__guest vd-cmt-login-hint"
              >
                <span class="vd-cmt-login-hint-muted">请先</span>
                <button
                  type="button"
                  class="vd-cmt-login-hint-btn"
                  @click="openLoginModal"
                >
                  登录
                </button>
                <span class="vd-cmt-login-hint-muted"
                  >后发表评论&nbsp;( · ω · )</span
                >
              </div>
              <div class="vd-cmt-uni-inbox__bar">
                <button
                  type="button"
                  class="vd-cmt-uni-emoji"
                  title="表情"
                  :disabled="!mbLoggedIn"
                  @click.prevent
                >
                  <span class="vd-emoji-ico" aria-hidden="true" />
                  表情
                </button>
              </div>
            </div>
            <button
              type="button"
              class="vd-cmt-submit vd-cmt-submit--mb"
              :class="{ 'is-guest': !mbLoggedIn }"
              :disabled="
                mbLoggedIn &&
                (!String(commentDraft || '').trim() || commentPosting)
              "
              @click="submitComment"
            >
              <template v-if="commentPosting">发送中…</template>
              <span v-else class="vd-cmt-submit-lines">发表<br />评论</span>
            </button>
          </div>
        </div>
      </div>
      <div class="vd-cmt-live-mount">
        <MinibiliCommentsLive
          ref="commentsLive"
          embedded
          hide-composer
          :key="
            'dyn-cmt-' +
            (isDynamic ? 'd' : isArticle ? 'a' : 'v') +
            '-' +
            mediaId +
            '-' +
            commentSort
          "
          :video-id="isArticle ? 0 : mediaId"
          :article-id="isArticle ? mediaId : 0"
          :dynamic-id="isDynamic ? mediaId : 0"
          :video-author-id="isArticle || isDynamic ? 0 : videoAuthorId"
          :article-author-id="isArticle ? articleAuthorId : 0"
          :dynamic-author-id="isDynamic ? dynamicAuthorId : 0"
          :comment-sort="commentSort"
          :initial-comments-curated="commentsCurated"
          :initial-comments-closed="commentsClosed"
          @counts="onLiveCounts"
        />
      </div>
      <div class="vd-cmt-page-bottom vd-cmt-page-bottom--mb">
        <div class="vd-cmt-page-bottom-left">
          <template
            v-for="(it, pidx) in commentPagerItems"
            :key="'dyn-cmt-page-mb-' + pidx"
          >
            <span
              v-if="it.type === 'ellipsis'"
              class="vd-page-ellipsis"
              >...</span
            >
            <button
              v-else
              type="button"
              class="vd-page-num vd-page-num--boxed"
              :class="{ on: it.n === commentCurrentPage }"
              @click="setCommentPage(it.n)"
            >
              {{ it.n }}
            </button>
          </template>
          <button
            type="button"
            class="vd-page-next vd-page-next--boxed"
            :disabled="commentCurrentPage >= commentTotalPages"
            @click="setCommentPage(commentCurrentPage + 1)"
          >
            下一页
          </button>
        </div>
        <div class="vd-cmt-page-bottom-right">
          <span class="vd-page-bottom-meta">共{{ commentTotalPages }}页，跳至</span>
          <input
            v-model="commentPageJumpDraft"
            class="vd-page-jump-input"
            type="text"
            @keyup.enter="onCommentPageJump"
          />
          <span class="vd-page-bottom-meta">页</span>
        </div>
      </div>
  </div>
</template>

<script>
import akari from "@/assets/akari.jpg";
import { ElMessage } from "element-plus";
import {
  mbListComments,
  mbListArticleComments,
  mbListDynamicComments,
  mbPostComment,
  mbPostArticleComment,
  mbPostDynamicComment
} from "@/api/minibili";
import { getAccessToken } from "@/utils/authTokens";
import MinibiliCommentsLive from "@/pages/minibili/MinibiliCommentsLive.vue";
import {
  MB_COMMENT_CURATED_LABEL,
  MB_COMMENT_CURATED_PLACEHOLDER,
  MB_COMMENT_PENDING_TOAST
} from "@/constants/minibiliComments";

const COMMENT_PAGE_SIZE = 20;

export default {
  name: "MbDynVideoCommentPanel",
  components: { MinibiliCommentsLive },
  props: {
    video: {
      type: Object,
      default: null
    },
    article: {
      type: Object,
      default: null
    },
    dynamic: {
      type: Object,
      default: null
    },
    videoAuthorId: {
      type: Number,
      default: 0
    },
    articleAuthorId: {
      type: Number,
      default: 0
    },
    dynamicAuthorId: {
      type: Number,
      default: 0
    },
    /** 当前用户是否为该稿件作者（个人空间/自己的动态） */
    ownerCanCurate: {
      type: Boolean,
      default: false
    }
  },
  emits: [
    "counts",
    "patch-video",
    "patch-article",
    "patch-dynamic",
    "curated-dialog",
    "manage-dialog"
  ],
  data() {
    return {
      MB_COMMENT_CURATED_LABEL,
      akari,
      ownerMenuOpen: false,
      commentSort: "hot",
      commentDraft: "",
      commentPosting: false,
      commentTotal: 0,
      commentTotalPages: 1,
      commentCurrentPage: 1,
      commentPageJumpDraft: "1"
    };
  },
  computed: {
    isDynamic() {
      return !!(this.dynamic && Number(this.dynamic.id) > 0);
    },
    isArticle() {
      return !this.isDynamic && !!(this.article && Number(this.article.id) > 0);
    },
    media() {
      if (this.isDynamic) return this.dynamic;
      return this.isArticle ? this.article : this.video;
    },
    mediaId() {
      return Number(this.media && this.media.id) || 0;
    },
    mbLoggedIn() {
      return !!getAccessToken();
    },
    commentsClosed() {
      const m = this.media;
      return !!(m && m.comments_closed);
    },
    commentsCurated() {
      const m = this.media;
      return !!(m && m.comments_curated);
    },
    commentComposerPlaceholder() {
      return this.commentsCurated
        ? MB_COMMENT_CURATED_PLACEHOLDER
        : "评论千万条，等你发一条";
    },
    composerAvatar() {
      const m = this.$store.state.login && this.$store.state.login.minibiliMe;
      const u = m && String(m.avatar_url || "").trim();
      return u || akari;
    },
    pendingCommentsRoute() {
      const id = this.mediaId;
      if (!id) return { name: "creatorComments" };
      if (this.isDynamic) {
        return {
          name: "creatorComments",
          query: {
            tab: "pending",
            media: "dynamic",
            dynamic_id: String(id)
          }
        };
      }
      if (this.isArticle) {
        return {
          name: "creatorComments",
          query: {
            tab: "pending",
            media: "article",
            article_id: String(id)
          }
        };
      }
      return {
        name: "creatorComments",
        query: {
          tab: "pending",
          media: "video",
          video_id: String(id)
        }
      };
    },
    commentPagerItems() {
      const total = Math.max(
        1,
        parseInt(String(this.commentTotalPages), 10) || 1
      );
      const cur = Math.min(
        Math.max(1, parseInt(String(this.commentCurrentPage), 10) || 1),
        total
      );
      if (total <= 7) {
        return Array.from({ length: total }, (_, i) => ({
          type: "page",
          n: i + 1
        }));
      }
      const set = new Set([1, total, cur, cur - 1, cur + 1]);
      if (cur <= 3) {
        for (let i = 1; i <= 4; i++) set.add(i);
      }
      if (cur >= total - 2) {
        for (let i = total - 3; i <= total; i++) set.add(i);
      }
      const sorted = [...set]
        .filter(n => n >= 1 && n <= total)
        .sort((a, b) => a - b);
      const out = [];
      for (let i = 0; i < sorted.length; i++) {
        if (i > 0 && sorted[i] - sorted[i - 1] > 1) {
          out.push({ type: "ellipsis" });
        }
        out.push({ type: "page", n: sorted[i] });
      }
      return out;
    }
  },
  watch: {
    media: {
      immediate: true,
      handler(v) {
        const cc = Number(v && v.comment_count) || 0;
        this.applyCommentTotal(cc);
        this.commentCurrentPage = 1;
        this.commentPageJumpDraft = "1";
      }
    },
    commentSort() {
      this.commentCurrentPage = 1;
      this.commentPageJumpDraft = "1";
    },
    commentTotalPages() {
      const t = Math.max(
        1,
        parseInt(String(this.commentTotalPages), 10) || 1
      );
      if (this.commentCurrentPage > t) this.commentCurrentPage = t;
      if (this.commentCurrentPage < 1) this.commentCurrentPage = 1;
      this.commentPageJumpDraft = String(this.commentCurrentPage);
    }
  },
  mounted() {
    void this.syncCommentsClosedFlag();
    document.addEventListener("click", this.onDocClickCloseOwnerMenu);
  },
  beforeUnmount() {
    document.removeEventListener("click", this.onDocClickCloseOwnerMenu);
  },
  methods: {
    onDocClickCloseOwnerMenu() {
      this.ownerMenuOpen = false;
    },
    onOwnerMenuAction(action) {
      this.ownerMenuOpen = false;
      let kind = action;
      if (action === "pick_comment") {
        kind = this.commentsCurated ? "restore_pick_comment" : "pick_comment";
      } else if (action === "close_comments") {
        kind = this.commentsClosed ? "restore_comments" : "close_comments";
      }
      const payload = {
        kind,
        articleId: this.isArticle ? this.mediaId : 0,
        videoId: this.isArticle || this.isDynamic ? 0 : this.mediaId,
        dynamicId: this.isDynamic ? this.mediaId : 0
      };
      this.$emit("manage-dialog", payload);
      this.$emit("curated-dialog", payload);
    },
    formatCount(n) {
      const v = Number(n) || 0;
      if (v >= 10000) {
        return (v / 10000).toFixed(1).replace(/\.0$/, "") + "万";
      }
      return String(v);
    },
    applyCommentTotal(n) {
      const v = Number(n) || 0;
      this.commentTotal = v;
      this.commentTotalPages = Math.max(1, Math.ceil(v / COMMENT_PAGE_SIZE));
      if (this.commentCurrentPage > this.commentTotalPages) {
        this.commentCurrentPage = this.commentTotalPages;
      }
      this.commentPageJumpDraft = String(this.commentCurrentPage);
    },
    openLoginModal() {
      this.$store.commit("login/SET_LOGIN_TAB", 0);
      this.$store.commit("login/OPEN_LOGIN_MODAL");
    },
    emitPatch(partial) {
      const id = this.mediaId;
      if (!Number.isFinite(id) || id <= 0) {
        return;
      }
      if (this.isDynamic) {
        this.$emit("patch-dynamic", { dynamicId: id, partial });
        return;
      }
      if (this.isArticle) {
        this.$emit("patch-article", { articleId: id, partial });
        return;
      }
      this.$emit("patch-video", { videoId: id, partial });
    },
    async syncCommentsClosedFlag() {
      const id = this.mediaId;
      if (!Number.isFinite(id) || id <= 0) {
        return;
      }
      try {
        if (this.isDynamic) {
          const res = await mbListDynamicComments(id);
          this.emitPatch({
            comments_closed: !!res.comments_closed,
            comments_curated: !!res.comments_curated
          });
        } else if (this.isArticle) {
          const { comments_closed: closedFlag } = await mbListArticleComments(id);
          this.emitPatch({ comments_closed: !!closedFlag });
        } else {
          const { comments_closed: closedFlag } = await mbListComments(id);
          this.emitPatch({ comments_closed: !!closedFlag });
        }
      } catch {
        /* ignore */
      }
    },
    refreshCommentsLive(opts) {
      const ref = this.$refs.commentsLive;
      if (ref && typeof ref.load === "function") {
        return ref.load(opts || { soft: true, preserveExpand: true });
      }
      return Promise.resolve();
    },
    onLiveCounts(n) {
      const count = Number(n) || 0;
      this.applyCommentTotal(count);
      this.emitPatch({ comment_count: count });
      this.$emit("counts", count);
    },
    setCommentPage(n) {
      const t = Math.max(
        1,
        parseInt(String(this.commentTotalPages), 10) || 1
      );
      const raw = parseInt(String(n), 10);
      const p = Number.isFinite(raw)
        ? Math.min(Math.max(1, raw), t)
        : 1;
      this.commentCurrentPage = p;
      this.commentPageJumpDraft = String(p);
    },
    onCommentPageJump() {
      const raw = parseInt(String(this.commentPageJumpDraft).trim(), 10);
      if (!Number.isFinite(raw)) {
        this.commentPageJumpDraft = String(this.commentCurrentPage);
        return;
      }
      this.setCommentPage(raw);
    },
    async submitComment() {
      if (!this.mbLoggedIn) {
        this.openLoginModal();
        return;
      }
      const text = String(this.commentDraft || "").trim();
      if (!text) {
        return;
      }
      const id = this.mediaId;
      this.commentPosting = true;
      try {
        let approved = true;
        if (this.isDynamic) {
          const res = await mbPostDynamicComment(id, text, 0);
          approved = !(res && res.approved === false);
        } else if (this.isArticle) {
          const res = await mbPostArticleComment(id, text, 0);
          approved = !(res && res.approved === false);
        } else {
          const res = await mbPostComment(id, text, 0);
          approved = !(res && res.approved === false);
        }
        this.commentDraft = "";
        if (approved) {
          this.applyCommentTotal(this.commentTotal + 1);
          this.emitPatch({ comment_count: this.commentTotal });
        }
        await this.refreshCommentsLive();
        if (!approved) {
          ElMessage.success(MB_COMMENT_PENDING_TOAST);
        } else {
          ElMessage.success("评论已发表");
        }
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      } finally {
        this.commentPosting = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/styles/vd-comments-chrome.scss";
@import "@/styles/vd-comment-list.scss";
@import "@/pages/minibili/dynCommentPanel.scss";

.vd-cmt-live-mount :deep(.mb-cmt.mb-cmt--embedded) {
  margin-top: 0;
  padding: 0;
  border: none;
  background: transparent;
}
</style>
