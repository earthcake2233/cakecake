<template>
  <div class="vd-comments" id="comment-main">
    <div class="vd-comments-mock">
      <div v-if="!hideHead" class="vd-cmt-head">
        <h3 class="vd-cmt-title">
          <span class="vd-cmt-count">{{ commentTotal }}</span> 评论
        </h3>
        <div class="vd-cmt-head-row vd-cmt-head-row--toolbar">
          <div class="vd-cmt-sort">
            <span v-if="commentsCurated" class="vd-cmt-curated-label">{{
              MB_COMMENT_CURATED_LABEL
            }}</span>
            <template v-else>
              <button
                type="button"
                class="vd-sort-tab"
                :class="{ on: commentSort === 'hot' }"
                @click="$emit('update:commentSort', 'hot')"
              >
                按热度排序
              </button>
              <button
                type="button"
                class="vd-sort-tab"
                :class="{ on: commentSort === 'time' }"
                @click="$emit('update:commentSort', 'time')"
              >
                按时间排序
              </button>
            </template>
          </div>
          <div class="vd-cmt-page-top vd-cmt-page-top--links">
            <span class="vd-page-info">共{{ commentTotalPages }}页</span>
            <template
              v-for="(it, pidx) in commentPagerItems"
              :key="'vd-cmt-page-top-' + pidx"
            >
              <span v-if="it.type === 'ellipsis'" class="vd-page-ellipsis"
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

      <template v-if="commentsClosed">
        <div class="mb-article-cmt-closed-bar" role="status">UP主已关闭评论</div>
        <p class="mb-article-cmt-closed-foot">已到达世界的尽头</p>
      </template>
      <template v-else>
        <div class="vd-cmt-composer vd-cmt-composer--mb">
          <img
            class="vd-cmt-avatar vd-cmt-avatar--mb"
            :src="composerAvatarSrc"
            width="48"
            height="48"
            alt=""
          />
          <div class="vd-cmt-mb-composer-main">
            <div class="vd-cmt-mb-editor-row">
              <div class="vd-cmt-uni-inbox">
                <template v-if="loggedIn || commentsCurated">
                  <textarea
                    v-model="commentDraft"
                    class="vd-cmt-uni-inbox__field"
                    :class="{
                      'is-curated-hint': commentsCurated && !loggedIn
                    }"
                    rows="3"
                    maxlength="1000"
                    :readonly="commentsCurated && !loggedIn"
                    :disabled="commentsCurated && !loggedIn"
                    :placeholder="composerPlaceholder"
                  />
                </template>
                <div v-else class="vd-cmt-uni-inbox__guest vd-cmt-login-hint">
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
                    title="表情（演示）"
                    :disabled="!loggedIn"
                    @click="onEmojiClick"
                  >
                    <span class="vd-emoji-ico" aria-hidden="true" />
                    表情
                  </button>
                </div>
              </div>
              <button
                type="button"
                class="vd-cmt-submit vd-cmt-submit--mb"
                :class="{ 'is-guest': !loggedIn }"
                :disabled="
                  loggedIn && (!commentCanSubmit || commentPosting)
                "
                @click="onComposerSubmit"
              >
                <template v-if="commentPosting">发送中…</template>
                <span v-else class="vd-cmt-submit-lines"
                  >发表<br />评论</span
                >
              </button>
            </div>
          </div>
        </div>
        <div class="vd-cmt-live-mount">
          <MinibiliCommentsLive
            ref="commentsLive"
            embedded
            hide-composer
            :video-id="videoId"
            :article-id="articleId"
            :dynamic-id="dynamicId"
            :video-author-id="videoId ? authorId : null"
            :article-author-id="articleId ? authorId : null"
            :dynamic-author-id="dynamicId ? authorId : null"
            :comment-sort="commentSort"
            :highlight-comment-id="highlightCommentId"
            :initial-comments-curated="commentsCurated"
            :initial-comments-closed="commentsClosed"
            @counts="onCommentCounts"
          />
        </div>
        <div class="vd-cmt-page-bottom vd-cmt-page-bottom--mb">
          <div class="vd-cmt-page-bottom-left">
            <template
              v-for="(it, pidx) in commentPagerItems"
              :key="'vd-cmt-page-mb-' + pidx"
            >
              <span v-if="it.type === 'ellipsis'" class="vd-page-ellipsis"
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
            <span class="vd-page-bottom-meta"
              >共{{ commentTotalPages }}页，跳至</span
            >
            <input
              v-model="commentPageJumpDraft"
              class="vd-page-jump-input"
              type="text"
              @keyup.enter="onCommentPageJump"
            />
            <span class="vd-page-bottom-meta">页</span>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script>
import { createNamespacedHelpers } from "vuex";
import akariCover from "@/assets/akari.jpg";
import MinibiliCommentsLive from "@/pages/minibili/MinibiliCommentsLive.vue";
import {
  MB_COMMENT_CURATED_LABEL,
  MB_COMMENT_CURATED_PLACEHOLDER
} from "@/constants/minibiliComments";
import { getAccessToken } from "@/utils/authTokens";

const { mapState } = createNamespacedHelpers("login");

export default {
  name: "VdCommentPanelMb",
  components: { MinibiliCommentsLive },
  props: {
    videoId: { type: Number, default: 0 },
    articleId: { type: Number, default: 0 },
    dynamicId: { type: Number, default: 0 },
    authorId: { type: Number, default: null },
    commentSort: { type: String, default: "hot" },
    commentsClosed: { type: Boolean, default: false },
    commentsCurated: { type: Boolean, default: false },
    hideHead: { type: Boolean, default: false },
    initialTotal: { type: Number, default: 0 },
    highlightCommentId: { type: [Number, String], default: null }
  },
  emits: ["counts", "update:commentSort"],
  data() {
    return {
      MB_COMMENT_CURATED_LABEL,
      commentTotal: 0,
      commentTotalPages: 1,
      commentCurrentPage: 1,
      commentPageJumpDraft: "1",
      commentDraft: "",
      commentPosting: false
    };
  },
  computed: {
    ...mapState({
      proInfo: state => state.proInfo,
      minibiliMe: state => state.minibiliMe
    }),
    loggedIn() {
      void this.$store.state.login.signIn;
      return !!getAccessToken();
    },
    composerAvatarSrc() {
      void this.minibiliMe;
      void this.proInfo;
      const m = this.minibiliMe;
      if (m) {
        const u = String(m.avatar_url || "").trim();
        if (u) return u;
      }
      const p = this.proInfo;
      if (p && typeof p === "object" && !Array.isArray(p) && p.face) {
        return p.face;
      }
      return akariCover;
    },
    composerPlaceholder() {
      return this.commentsCurated
        ? MB_COMMENT_CURATED_PLACEHOLDER
        : "评论千万条，等你发一条";
    },
    commentCanSubmit() {
      return (
        this.loggedIn &&
        this.commentDraft.trim().length > 0 &&
        !this.commentPosting
      );
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
    initialTotal: {
      immediate: true,
      handler(n) {
        this.applyCommentTotal(n);
      }
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
  methods: {
    applyCommentTotal(n) {
      const v = Number(n);
      if (!Number.isFinite(v) || v < 0) return;
      this.commentTotal = v;
      this.commentTotalPages = Math.max(1, Math.ceil(v / 20));
      if (this.commentCurrentPage > this.commentTotalPages) {
        this.commentCurrentPage = this.commentTotalPages;
      }
      this.commentPageJumpDraft = String(this.commentCurrentPage);
    },
    onCommentCounts(n) {
      this.applyCommentTotal(n);
      this.$emit("counts", n);
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
    openLoginModal() {
      this.$store.commit("login/SET_LOGIN_TAB", 0);
      this.$store.commit("login/OPEN_LOGIN_MODAL");
    },
    onComposerSubmit() {
      if (!this.loggedIn) {
        this.openLoginModal();
        return;
      }
      void this.submitComment();
    },
    onEmojiClick() {
      if (!this.loggedIn) this.openLoginModal();
    },
    async submitComment() {
      if (!getAccessToken() || !this.commentDraft.trim()) return;
      const ref = this.$refs.commentsLive;
      if (!ref) return;
      this.commentPosting = true;
      try {
        const ok = await ref.postCommentExtern(this.commentDraft, 0);
        if (ok) this.commentDraft = "";
      } finally {
        this.commentPosting = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
.vd-cmt-live-mount {
  overflow: visible;
}

.vd-cmt-live-mount :deep(.mb-cmt.mb-cmt--embedded) {
  margin-top: 0;
  padding: 12px 0 16px;
  border: none;
  border-radius: 0;
  background: transparent;
  overflow: visible;
}
</style>

<style lang="scss">
@import "@/styles/vd-comment-list.scss";
@import "@/styles/vd-comment-panel.scss";
</style>
