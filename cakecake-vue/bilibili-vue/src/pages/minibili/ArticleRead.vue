<template>
  <div class="mb-article-page">
    <div class="mb-article-bg" :style="pageBgStyle" aria-hidden="true" />
    <div class="mb-article-root">
      <p v-if="loading" class="mb-article-loading">加载中…</p>
      <p v-else-if="loadError" class="mb-article-err">{{ loadError }}</p>
      <div
        v-else-if="article"
        class="mb-article-layout"
        :class="{ 'has-toc': toc.length > 0 }"
      >
        <div v-if="toc.length" ref="tocAnchor" class="mb-article-toc-slot">
          <aside
            v-show="tocOpen"
            ref="tocPanel"
            class="mb-article-toc"
            :class="{ 'is-stuck': tocStuck }"
            :style="tocStuck ? tocPanelStyle : null"
            aria-label="目录"
          >
            <div class="mb-article-toc__head">
              <span class="mb-article-toc__label">目录</span>
              <button
                type="button"
                class="mb-article-toc__chev"
                aria-label="收起目录"
                @click="tocOpen = false"
              >
                <svg
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  aria-hidden="true"
                >
                  <path
                    d="M6 14l6-6 6 6"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                  />
                </svg>
              </button>
            </div>
            <p class="mb-article-toc__doc-title" :title="article.title">
              {{ article.title }}
            </p>
            <ul class="mb-article-toc__list">
              <li
                v-for="entry in toc"
                :key="entry.id"
                class="mb-article-toc__item"
              >
                <button
                  type="button"
                  class="mb-article-toc__link"
                  :class="[
                    `is-l${entry.level}`,
                    { 'is-active': activeTocId === entry.id }
                  ]"
                  :title="entry.text"
                  @click="scrollToHeading(entry.id)"
                >
                  <span
                    class="mb-article-toc__link-inner"
                    v-html="tocEntryHtml(entry)"
                  />
                </button>
              </li>
            </ul>
          </aside>
          <button
            v-show="!tocOpen"
            ref="tocPill"
            type="button"
            class="mb-article-toc-pill"
            :class="{ 'is-stuck': tocStuck }"
            :style="tocStuck ? tocPanelStyle : null"
            aria-label="展开目录"
            @click="tocOpen = true"
          >
            <span>目录</span>
            <svg width="12" height="12" viewBox="0 0 24 24" aria-hidden="true">
              <path
                d="M6 10l6 6 6-6"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
              />
            </svg>
          </button>
        </div>

        <article
          class="mb-article-main"
          :class="{ 'mb-article-main--has-cover': coverUrl }"
        >
          <figure v-if="coverUrl" class="mb-article-cover">
            <img :src="coverUrl" :alt="article.title" />
          </figure>
          <div class="mb-article-main__body">
          <h1 class="mb-article-title">{{ article.title }}</h1>
          <header class="mb-article-meta">
            <router-link
              v-if="authorRoute"
              :to="authorRoute"
              class="mb-article-meta__face-link"
            >
              <img
                class="mb-article-meta__face"
                :src="article.author_avatar || defaultAvatar"
                alt=""
              />
            </router-link>
            <img
              v-else
              class="mb-article-meta__face"
              :src="article.author_avatar || defaultAvatar"
              alt=""
            />
            <div>
              <router-link
                v-if="authorRoute"
                :to="authorRoute"
                class="mb-article-meta__name"
              >
                {{ article.author_name || "用户" }}
              </router-link>
              <span v-else class="mb-article-meta__name">{{
                article.author_name || "用户"
              }}</span>
              <p class="mb-article-meta__time">{{ publishedLabel }}</p>
            </div>
          </header>

          <div
            ref="bodyEl"
            class="mb-article-body"
            v-html="article.body_html"
          />

          <footer v-if="!isDynamic" class="mb-article-foot">
            <div
              v-if="article.tags && article.tags.length"
              class="mb-article-tags"
            >
              <span
                v-for="(tag, i) in article.tags"
                :key="'tag-' + i"
                class="mb-article-tag"
                >{{ tag }}</span
              >
            </div>
            <div class="mb-article-foot__legal">
              <span class="mb-article-foot__copy">
                本文为我原创，未经授权禁止转载
              </span>
              <span class="mb-article-foot__cv"
                >cv{{ article.cv_id || article.id }}</span
              >
            </div>
            <div class="mb-article-foot__share">
              <div class="mb-article-foot__share-left">
                <span class="mb-article-foot__share-label">分享至</span>
                <a
                  class="mb-article-share-dot share-feed"
                  href="javascript:;"
                  title="分享到动态"
                  aria-label="分享到动态"
                  @click.prevent="copyShareLink"
                />
                <a
                  class="mb-article-share-dot share-qzone"
                  href="javascript:;"
                  title="分享到QQ空间"
                  aria-label="分享到QQ空间"
                  @click.prevent="copyShareLink"
                />
                <a
                  class="mb-article-share-dot share-qq"
                  href="javascript:;"
                  title="分享到QQ"
                  aria-label="分享到QQ"
                  @click.prevent="copyShareLink"
                />
              </div>
              <button type="button" class="mb-article-foot__report">
                投诉或建议
              </button>
            </div>
          </footer>

          <section id="article-comments" class="mb-article-cmt">
            <h3 class="mb-article-cmt__title">
              评论 {{ article.comment_count }}
            </h3>
            <div class="mb-article-cmt__body">
              <div v-if="!articleCommentsClosed" class="mb-article-cmt__sort">
                <button
                  type="button"
                  class="mb-article-cmt__sort-btn"
                  :class="{ 'is-on': commentSort === 'hot' }"
                  @click="commentSort = 'hot'"
                >
                  最热
                </button>
                <span class="mb-article-cmt__sort-sep" aria-hidden="true"
                  >|</span
                >
                <button
                  type="button"
                  class="mb-article-cmt__sort-btn"
                  :class="{ 'is-on': commentSort === 'time' }"
                  @click="commentSort = 'time'"
                >
                  最新
                </button>
              </div>
              <VdCommentPanelMb
                hide-head
                :article-id="isDynamic ? 0 : articleId"
                :dynamic-id="isDynamic ? contentId : 0"
                :author-id="article.user_id"
                :comment-sort="commentSort"
                :comments-closed="articleCommentsClosed"
                :comments-curated="!!article.comments_curated"
                :initial-total="article.comment_count"
                :highlight-comment-id="mbHighlightCommentId"
                @counts="onCommentCounts"
              />
            </div>
          </section>
          </div>
        </article>

        <aside class="mb-article-rail" aria-label="操作">
          <div class="mb-article-rail__panel">
            <button
              v-if="!isDynamic"
              type="button"
              class="mb-article-rail__btn"
              :class="{ 'is-active': coinDone }"
              title="投币"
              @click="openCoinDialog"
            >
              <span class="mb-article-rail__ico-wrap">
                <img
                  class="mb-article-rail__ico"
                  :src="icoCoin"
                  width="28"
                  height="28"
                  alt=""
                />
              </span>
              <span class="mb-article-rail__num">{{
                formatCount(article.coin_count)
              }}</span>
            </button>
            <button
              type="button"
              class="mb-article-rail__btn"
              :class="{ 'is-active': favorited }"
              :title="isDynamic ? '点赞' : '收藏'"
              @click="toggleFavorite"
            >
              <span class="mb-article-rail__ico-wrap">
                <img
                  v-if="!isDynamic"
                  class="mb-article-rail__ico"
                  :src="icoCollect"
                  width="28"
                  height="28"
                  alt=""
                />
                <svg
                  v-else
                  class="mb-article-rail__ico mb-article-rail__ico--like"
                  viewBox="0 0 24 24"
                  :fill="favorited ? 'currentColor' : 'none'"
                  stroke="currentColor"
                  stroke-width="1.5"
                  aria-hidden="true"
                >
                  <path
                    d="M7 10v12M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.67 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2a3.13 3.13 0 0 1 3 3.88Z"
                  />
                </svg>
              </span>
              <span class="mb-article-rail__num">{{
                formatCount(isDynamic ? article.like_count : article.fav_count)
              }}</span>
            </button>
            <button
              type="button"
              class="mb-article-rail__btn"
              title="转发"
              @click="copyShareLink"
            >
              <span class="mb-article-rail__ico-wrap">
                <img
                  class="mb-article-rail__ico"
                  :src="icoTranspond"
                  width="28"
                  height="28"
                  alt=""
                />
              </span>
              <span class="mb-article-rail__num">{{
                formatCount(article.forward_count)
              }}</span>
            </button>
            <button
              type="button"
              class="mb-article-rail__btn"
              title="评论"
              @click="scrollToComments"
            >
              <span class="mb-article-rail__ico-wrap">
                <img
                  class="mb-article-rail__ico"
                  :src="icoComment"
                  width="28"
                  height="28"
                  alt=""
                />
              </span>
              <span class="mb-article-rail__num">{{
                formatCount(article.comment_count)
              }}</span>
            </button>
          </div>
        </aside>
      </div>
    </div>

    <VideoCoinDialog
      v-model="coinDialogOpen"
      :loading="coinLoading"
      :is-own-video="coinIsOwnArticle"
      :coin-balance="coinBalance"
      :prior-coin-amount="myCoinAmount"
      :daily-coin-exp-progress="coinDailyProgress"
      :daily-coin-exp-max="coinDailyMax"
      @confirm="onCoinConfirm"
      @cancel="coinDialogOpen = false"
    />

    <MbDynamicEditDialog
      v-if="isDynamic && article && article.is_author"
      ref="dynamicEditDialog"
      v-model="dynamicEditOpen"
      :dynamic-id="contentId"
      :initial-title="dynamicEditSeed.title"
      :initial-content="dynamicEditSeed.content"
      :initial-images="dynamicEditSeed.images"
      @published="onDynamicEdited"
      @dirty-change="dynamicEditDirty = $event"
    />
  </div>
</template>

<script>
import { ElMessage } from "element-plus";
import DOMPurify from "dompurify";
import akari from "@/assets/akari.jpg";
import icoCoin from "@/assets/text/coin.png";
import icoCollect from "@/assets/text/collect.png";
import icoTranspond from "@/assets/text/transpond.png";
import icoComment from "@/assets/text/comment .png";
import VideoCoinDialog from "@/components/video/VideoCoinDialog.vue";
import VdCommentPanelMb from "@/components/comment/VdCommentPanelMb.vue";
import MbDynamicEditDialog from "@/components/minibili/MbDynamicEditDialog.vue";
import {
  mbGetArticle,
  mbGetUserDynamic,
  mbPostArticleView,
  mbToggleArticleFavorite,
  mbToggleDynamicLike,
  mbPostArticleCoin,
  mbGetMe
} from "@/api/minibili";
import { getAccessToken, getUserId } from "@/utils/authTokens";
import { minibiliUserSpaceRoute } from "@/utils/minibiliRoutes";

const PAGE_TITLE_ARTICLE = "专栏 - cakecake";
const PAGE_TITLE_DYNAMIC = "动态 - cakecake";

export default {
  name: "ArticleReadPage",
  components: { VdCommentPanelMb, VideoCoinDialog, MbDynamicEditDialog },
  data() {
    return {
      defaultAvatar: akari,
      icoCoin,
      icoCollect,
      icoTranspond,
      icoComment,
      pageBg: new URL("../../assets/dynamics/bg.png@1c.avif", import.meta.url)
        .href,
      loading: true,
      loadError: "",
      article: null,
      favorited: false,
      tocOpen: true,
      commentSort: "hot",
      tocPanelStyle: null,
      tocStuck: false,
      activeTocId: "",
      _tocObserver: null,
      coinDialogOpen: false,
      coinLoading: false,
      coinBalance: 0,
      myCoinAmount: 0,
      coinDailyProgress: 0,
      coinDailyMax: 50,
      dynamicEditOpen: false,
      dynamicEditDirty: false,
      dynamicEditSeed: { title: "", content: "", images: [] },
      rawDynamicDetail: null
    };
  },
  computed: {
    pageBgStyle() {
      return { backgroundImage: `url(${this.pageBg})` };
    },
    isDynamic() {
      return this.$route.name === "minibiliDynamicRead";
    },
    contentId() {
      const n = parseInt(String(this.$route.params.id || ""), 10);
      return Number.isFinite(n) && n > 0 ? n : 0;
    },
    articleId() {
      return this.isDynamic ? 0 : this.contentId;
    },
    pageTitle() {
      return this.isDynamic ? PAGE_TITLE_DYNAMIC : PAGE_TITLE_ARTICLE;
    },
    toc() {
      return (this.article && this.article.toc) || [];
    },
    authorRoute() {
      if (!this.article) return null;
      return minibiliUserSpaceRoute(this.article.user_id);
    },
    publishedLabel() {
      const t = this.article && this.article.published_at;
      return t || "";
    },
    coinDone() {
      return (Number(this.myCoinAmount) || 0) > 0;
    },
    /** 与播放页一致：UP 主给自己专栏投币时弹窗内提示「up主不能自己投币」 */
    coinIsOwnArticle() {
      if (!getAccessToken() || !this.article) return false;
      if (this.article.is_author) return true;
      const me = getUserId();
      const author = this.article.user_id;
      if (me == null || author == null) return false;
      return Number(me) === Number(author);
    },
    coverUrl() {
      const u = this.article && this.article.cover_url;
      return u && String(u).trim() ? String(u).trim() : "";
    },
    /** 消息中心等跳转：?mb_cid=评论ID，评论区对应楼层标蓝并滚入视野 */
    mbHighlightCommentId() {
      const q = this.$route.query && this.$route.query.mb_cid;
      if (q == null || q === "") return null;
      const raw = Array.isArray(q) ? q[0] : q;
      const n = parseInt(String(raw), 10);
      return Number.isFinite(n) && n > 0 ? n : null;
    },
    articleCommentsClosed() {
      return !!(this.article && this.article.comments_closed);
    },
    wantDynamicEditFromQuery() {
      const q = this.$route.query && this.$route.query.edit;
      const raw = Array.isArray(q) ? q[0] : q;
      return raw === "1" || raw === 1 || raw === true;
    }
  },
  watch: {
    "$route.params.id"() {
      void this.loadContent();
    },
    "$route.name"() {
      void this.loadContent();
    },
    "$route.query.edit"() {
      this.syncDynamicEditFromRoute();
    },
    dynamicEditOpen(open) {
      if (!open) {
        this.clearDynamicEditQuery();
      }
    },
    mbHighlightCommentId(val) {
      if (val) {
        this.$nextTick(() => this.scrollToComments());
      }
    },
    toc: {
      handler(list) {
        if (this.loading) return;
        this.scheduleTocLayoutSync();
      }
    },
    tocOpen() {
      this.scheduleTocLayoutSync();
    },
    loading(isLoading) {
      if (!isLoading && this.article && this.toc.length) {
        this.scheduleTocLayoutSync();
      }
    }
  },
  mounted() {
    document.documentElement.classList.add("mb-article-read-page");
    void this.loadContent();
  },
  beforeRouteLeave(to, from, next) {
    void this.guardDynamicEditLeave().then(ok => {
      if (ok) next();
      else next(false);
    });
  },
  deactivated() {
    this.dynamicEditOpen = false;
    this.dynamicEditDirty = false;
  },
  beforeUnmount() {
    document.documentElement.classList.remove("mb-article-read-page");
    this.unbindTocLayoutSync();
    this.teardownTocObserver();
    this.dynamicEditOpen = false;
    this.dynamicEditDirty = false;
  },
  methods: {
    tocEntryHtml(entry) {
      if (!entry) return "";
      const raw =
        (entry.text_html && String(entry.text_html).trim()) ||
        (entry.text && String(entry.text).trim()) ||
        "";
      if (!raw) return "";
      return DOMPurify.sanitize(raw, {
        ALLOWED_TAGS: [
          "font",
          "b",
          "strong",
          "i",
          "em",
          "u",
          "s",
          "del",
          "span",
          "br"
        ],
        ALLOWED_ATTR: ["color", "class"]
      });
    },
    async guardDynamicEditLeave() {
      if (!this.dynamicEditOpen || !this.dynamicEditDirty) {
        return true;
      }
      const dlg = this.$refs.dynamicEditDialog;
      if (dlg && typeof dlg.confirmLeaveIfDirty === "function") {
        return dlg.confirmLeaveIfDirty();
      }
      return true;
    },
    syncDynamicEditFromRoute() {
      if (
        !this.isDynamic ||
        !this.article ||
        !this.article.is_author ||
        !this.wantDynamicEditFromQuery
      ) {
        return;
      }
      this.seedDynamicEditor();
      this.dynamicEditOpen = true;
    },
    seedDynamicEditor() {
      const d = this.rawDynamicDetail;
      this.dynamicEditSeed = {
        title: (d && d.title) || (this.article && this.article.title) || "",
        content: (d && d.content) || "",
        images:
          d && Array.isArray(d.images)
            ? [...d.images]
            : []
      };
    },
    clearDynamicEditQuery() {
      if (!this.wantDynamicEditFromQuery) return;
      const q = { ...this.$route.query };
      delete q.edit;
      void this.$router.replace({ ...this.$route, query: q });
    },
    async onDynamicEdited(item) {
      this.dynamicEditDirty = false;
      this.clearDynamicEditQuery();
      if (item && typeof item === "object") {
        this.rawDynamicDetail = {
          ...this.rawDynamicDetail,
          ...item,
          images: Array.isArray(item.images) ? item.images : []
        };
        this.article = this.mapDynamicToArticleView({
          ...(this.rawDynamicDetail || {}),
          user_id: this.article.user_id,
          author_name: this.article.author_name,
          author_avatar: this.article.author_avatar,
          is_author: true
        });
        this.favorited = !!item.liked_by_me;
      } else {
        await this.loadDynamic();
      }
    },
    formatCount(n) {
      const v = Number(n);
      if (!Number.isFinite(v) || v < 0) return "0";
      if (v >= 10000) {
        const w = v / 10000;
        return (w >= 10 ? Math.floor(w) : w.toFixed(1).replace(/\.0$/, "")) + "万";
      }
      return String(v);
    },
    escapeHtml(text) {
      return String(text || "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
    },
    buildDynamicBodyHtml(content, images) {
      const parts = [];
      const c = String(content || "").trim();
      if (c) {
        parts.push(`<p>${this.escapeHtml(c).replace(/\n/g, "<br>")}</p>`);
      }
      const imgs = Array.isArray(images) ? images : [];
      if (imgs.length) {
        parts.push('<div class="mb-article-dyn-imgs">');
        for (const u of imgs) {
          const src = this.escapeHtml(String(u || "").trim());
          if (src) {
            parts.push(`<figure class="mb-article-dyn-imgs__cell"><img src="${src}" alt="" loading="lazy" /></figure>`);
          }
        }
        parts.push("</div>");
      }
      return parts.join("");
    },
    mapDynamicToArticleView(d) {
      return {
        id: d.id,
        user_id: d.user_id,
        title: d.title || "动态",
        cover_url: "",
        body_html: this.buildDynamicBodyHtml(d.content, d.images),
        toc: [],
        tags: [],
        view_count: 0,
        comment_count: Number(d.comment_count) || 0,
        coin_count: 0,
        fav_count: 0,
        like_count: Number(d.like_count) || 0,
        forward_count: 0,
        published_at: d.created_at || "",
        author_name: d.author_name || "",
        author_avatar: d.author_avatar || "",
        favorited_by_me: false,
        liked_by_me: !!d.liked_by_me,
        is_author: !!d.is_author,
        comments_closed: !!d.comments_closed,
        comments_curated: !!d.comments_curated
      };
    },
    async loadContent() {
      if (this.isDynamic) {
        await this.loadDynamic();
      } else {
        await this.loadArticle();
      }
    },
    async loadDynamic() {
      if (!this.contentId) {
        this.loadError = "无效的动态 ID";
        this.loading = false;
        return;
      }
      this.loading = true;
      this.loadError = "";
      try {
        const dyn = await mbGetUserDynamic(this.contentId);
        this.rawDynamicDetail = dyn;
        const art = this.mapDynamicToArticleView(dyn);
        this.article = art;
        this.favorited = !!dyn.liked_by_me;
        document.title = `${art.title} - ${this.pageTitle}`;
        this.$nextTick(() => {
          this.syncDynamicEditFromRoute();
          if (this.mbHighlightCommentId) {
            this.scrollToComments();
          }
        });
      } catch (e) {
        this.article = null;
        this.loadError = (e && e.message) || "动态不存在";
        document.title = this.pageTitle;
      } finally {
        this.loading = false;
      }
    },
    async loadArticle() {
      if (!this.contentId) {
        this.loadError = "无效的专栏 ID";
        this.loading = false;
        return;
      }
      this.loading = true;
      this.loadError = "";
      try {
        const art = await mbGetArticle(this.contentId);
        this.article = art;
        this.favorited = !!art.favorited_by_me;
        this.myCoinAmount = Number(art.my_coin_amount) || 0;
        document.title = `${art.title} - ${this.pageTitle}`;
        if (getAccessToken()) {
          try {
            await mbPostArticleView(this.contentId);
          } catch {
            /* ignore */
          }
        }
        this.$nextTick(() => {
          this.setupTocObserver();
          this.scheduleTocLayoutSync();
          if (this.mbHighlightCommentId) {
            this.scrollToComments();
          }
        });
      } catch (e) {
        this.article = null;
        this.loadError =
          (e && e.message) || "专栏不存在或尚未发布";
        document.title = this.pageTitle;
      } finally {
        this.loading = false;
        if (this.article && this.toc.length) {
          this.scheduleTocLayoutSync();
        }
      }
    },
    setupTocObserver() {
      this.teardownTocObserver();
      const root = this.$refs.bodyEl;
      if (!root || !this.toc.length) return;
      const headings = root.querySelectorAll("h1, h2, h3");
      if (!headings.length) return;
      const byId = {};
      headings.forEach(el => {
        if (el.id) byId[el.id] = el;
      });
      this._tocObserver = new IntersectionObserver(
        entries => {
          const visible = entries
            .filter(e => e.isIntersecting)
            .sort((a, b) => b.intersectionRatio - a.intersectionRatio);
          if (visible.length && visible[0].target.id) {
            this.activeTocId = visible[0].target.id;
          }
        },
        { rootMargin: "-64px 0px -60% 0px", threshold: [0, 0.25, 1] }
      );
      Object.values(byId).forEach(el => this._tocObserver.observe(el));
      if (!this.activeTocId && this.toc[0]) {
        this.activeTocId = this.toc[0].id;
      }
    },
    teardownTocObserver() {
      if (this._tocObserver) {
        this._tocObserver.disconnect();
        this._tocObserver = null;
      }
    },
    scheduleTocLayoutSync() {
      if (!this.toc.length) {
        this.unbindTocLayoutSync();
        this.tocPanelStyle = null;
        this.tocStuck = false;
        return;
      }
      if (this._tocLayoutSyncPending) return;
      this._tocLayoutSyncPending = true;
      this.$nextTick(() => {
        requestAnimationFrame(() => {
          requestAnimationFrame(() => {
            this._tocLayoutSyncPending = false;
            if (!this.toc.length || this.loading) return;
            this.bindTocLayoutSync();
            this.syncTocFixedPosition();
          });
        });
      });
    },
    bindTocLayoutSync() {
      this.unbindTocLayoutSync();
      if (!this.toc.length || this.loading) return;
      const anchor = this.$refs.tocAnchor;
      if (!anchor) return;
      this._onTocLayoutSync = () => this.syncTocFixedPosition();
      window.addEventListener("resize", this._onTocLayoutSync, { passive: true });
      document.addEventListener("scroll", this._onTocLayoutSync, {
        passive: true,
        capture: true
      });
      if (typeof ResizeObserver !== "undefined") {
        this._tocLayoutObserver = new ResizeObserver(() =>
          this.syncTocFixedPosition()
        );
        this._tocLayoutObserver.observe(anchor);
        const panel = this.$refs.tocPanel;
        const pill = this.$refs.tocPill;
        if (panel) this._tocLayoutObserver.observe(panel);
        if (pill) this._tocLayoutObserver.observe(pill);
      }
    },
    unbindTocLayoutSync() {
      if (this._tocLayoutObserver) {
        this._tocLayoutObserver.disconnect();
        this._tocLayoutObserver = null;
      }
      if (this._onTocLayoutSync) {
        window.removeEventListener("resize", this._onTocLayoutSync);
        document.removeEventListener("scroll", this._onTocLayoutSync, true);
        this._onTocLayoutSync = null;
      }
      const anchor = this.$refs.tocAnchor;
      if (anchor && anchor.style) {
        anchor.style.minHeight = "";
      }
    },
    syncTocFixedPosition() {
      if (!this.toc.length || this.loading) {
        this.tocPanelStyle = null;
        this.tocStuck = false;
        return;
      }
      const anchor = this.$refs.tocAnchor;
      if (!anchor || typeof anchor.getBoundingClientRect !== "function") {
        return;
      }
      const floater = this.tocOpen ? this.$refs.tocPanel : this.$refs.tocPill;
      if (floater) {
        const panelHeight = Math.ceil(floater.offsetHeight || 0);
        if (panelHeight > 0) {
          anchor.style.minHeight = `${panelHeight}px`;
        }
      }

      const rect = anchor.getBoundingClientRect();
      if (rect.width < 1) {
        this.tocPanelStyle = null;
        this.tocStuck = false;
        return;
      }
      const stickyTop = 64;
      this.tocStuck = rect.top <= stickyTop;
      if (this.tocStuck) {
        const style = {
          left: `${Math.max(0, Math.round(rect.left))}px`
        };
        if (this.tocOpen) {
          style.width = `${Math.round(rect.width)}px`;
        }
        this.tocPanelStyle = style;
      } else {
        this.tocPanelStyle = null;
      }
    },
    scrollToHeading(id) {
      const root = this.$refs.bodyEl;
      if (!root || !id) return;
      const el = root.querySelector(`#${CSS.escape(id)}`);
      if (!el) return;
      const navOffset = 64;
      const top =
        el.getBoundingClientRect().top + window.scrollY - navOffset - 8;
      window.scrollTo({ top: Math.max(0, top), behavior: "smooth" });
      this.activeTocId = id;
    },
    scrollToComments() {
      document
        .getElementById("article-comments")
        ?.scrollIntoView({ behavior: "smooth" });
    },
    onCommentCounts(n) {
      if (this.article && typeof n === "number") {
        this.article.comment_count = n;
      }
    },
    async toggleFavorite() {
      if (!getAccessToken()) {
        this.$store.commit("login/OPEN_LOGIN_MODAL");
        return;
      }
      try {
        if (this.isDynamic) {
          const res = await mbToggleDynamicLike(this.contentId);
          this.favorited = !!res.liked;
          if (this.article) {
            const base = Number(this.article.like_count) || 0;
            const delta = Number(res.like_count_delta) || 0;
            this.article.like_count = Math.max(0, base + delta);
          }
          return;
        }
        const res = await mbToggleArticleFavorite(this.contentId);
        this.favorited = !!res.favorited;
        if (this.article) this.article.fav_count = res.fav_count;
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      }
    },
    async openCoinDialog() {
      if (!getAccessToken()) {
        this.$store.commit("login/OPEN_LOGIN_MODAL");
        return;
      }
      if (this.myCoinAmount >= 2) {
        ElMessage.info("已为该专栏投满 2 枚硬币");
        return;
      }
      try {
        const me = await mbGetMe();
        this.coinBalance =
          typeof me.coin_balance === "number" ? me.coin_balance : 0;
        const prog = Number(me.daily_coin_exp_progress);
        this.coinDailyProgress = Number.isFinite(prog) ? Math.max(0, prog) : 0;
        const max = Number(me.daily_coin_exp_max);
        this.coinDailyMax = Number.isFinite(max) && max > 0 ? max : 50;
      } catch {
        this.coinBalance = 0;
      }
      this.coinDialogOpen = true;
    },
    async onCoinConfirm(amount) {
      this.coinLoading = true;
      try {
        const res = await mbPostArticleCoin(this.contentId, amount);
        if (this.article) {
          this.article.coin_count = res.coin_count;
        }
        this.myCoinAmount = res.my_coin_amount;
        this.coinBalance = res.coin_balance;
        this.coinDailyProgress = res.daily_coin_exp_progress;
        this.coinDailyMax = res.daily_coin_exp_max;
        ElMessage.success("投币成功");
        this.coinDialogOpen = false;
      } catch (e) {
        ElMessage.error((e && e.message) || "投币失败");
      } finally {
        this.coinLoading = false;
      }
    },
    copyShareLink() {
      const url = window.location.href;
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(url).then(
          () => ElMessage.success("链接已复制"),
          () => ElMessage.info(url)
        );
      } else {
        ElMessage.info(url);
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "./articleRead.scss";
</style>

<style lang="scss">
/* 不能放在 scoped 内：.app-body 在组件外，否则选择器无法命中 */
html.mb-article-read-page .app-body {
  overflow: visible;
}
</style>
