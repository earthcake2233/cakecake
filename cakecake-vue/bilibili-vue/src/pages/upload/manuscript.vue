<template>
  <CreatorShell>
    <div class="mm-wrap">
      <div class="mm-topbar">
        <div class="mm-tabs">
          <button
            type="button"
            class="mm-tab"
            :class="{ on: mainTab === 'video' }"
            @click="mainTab = 'video'"
          >
            视频管理
          </button>
          <button
            type="button"
            class="mm-tab"
            :class="{ on: mainTab === 'article' }"
            @click="mainTab = 'article'"
          >
            图文管理
          </button>
        </div>
        <div class="mm-search">
          <input
            v-model="searchQ"
            type="search"
            class="mm-search-input"
            placeholder="搜索稿件"
            autocomplete="off"
          />
          <span class="mm-search-ico" aria-hidden="true">
            <svg viewBox="0 0 24 24" width="16" height="16">
              <path
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                d="M11 18a7 7 0 100-14 7 7 0 000 14zm8 2l-4-4"
              />
            </svg>
          </span>
        </div>
      </div>

      <template v-if="mainTab === 'video'">
        <div class="mm-subbar">
          <div class="mm-status">
            <button
              type="button"
              class="mm-status-btn"
              :class="{ on: statusFilter === 'all' }"
              @click="setVideoStatusFilter('all')"
            >
              全部稿件
            </button>
            <button
              type="button"
              class="mm-status-btn"
              :class="{ on: statusFilter === 'draft' }"
              @click="setVideoStatusFilter('draft')"
            >
              草稿
            </button>
            <span class="mm-status-split" />
            <button
              type="button"
              class="mm-status-btn mm-status-count"
              :class="{ on: statusFilter === 'processing' }"
              @click="setVideoStatusFilter('processing')"
            >
              进行中 <em>{{ statusCounts.processing }}</em>
            </button>
            <button
              type="button"
              class="mm-status-btn mm-status-count"
              :class="{ on: statusFilter === 'passed' }"
              @click="setVideoStatusFilter('passed')"
            >
              已通过 <em>{{ statusCounts.passed }}</em>
            </button>
            <button
              type="button"
              class="mm-status-btn mm-status-count"
              :class="{ on: statusFilter === 'rejected' }"
              @click="setVideoStatusFilter('rejected')"
            >
              未通过 <em>{{ statusCounts.rejected }}</em>
            </button>
          </div>
          <div ref="sortRoot" class="mm-sort">
            <button
              type="button"
              class="mm-sort-trigger"
              @click="sortOpen = !sortOpen"
            >
              {{ sortLabel }}
              <span class="mm-sort-chevron" :class="{ open: sortOpen }" />
            </button>
            <div v-show="sortOpen" class="mm-sort-menu">
              <button
                v-for="opt in sortOptions"
                :key="opt.value"
                type="button"
                class="mm-sort-item"
                :class="{ active: sortKey === opt.value }"
                @click="pickSort(opt)"
              >
                {{ opt.label }}
                <span v-if="sortKey === opt.value" class="mm-sort-check">✓</span>
              </button>
            </div>
          </div>
        </div>

        <p v-if="isMinibiliMode && videosFetchError" class="mm-video-error">
          {{ videosFetchError }}
        </p>
        <div
          v-if="isMinibiliMode && videosLoading && !videos.length"
          class="mm-video-loading"
        >
          加载中…
        </div>
        <div v-else class="mm-list">
          <div v-for="item in displayVideos" :key="item.id" class="mm-row">
            <router-link
              class="mm-cover-wrap mm-cover-link"
              :to="videoPlayRoute(item)"
              :title="'播放：' + item.title"
            >
              <img class="mm-cover" :src="item.cover" alt="" />
              <span v-if="item.status === 'review'" class="mm-cover-badge">审核中</span>
              <span v-else-if="item.status === 'processing'" class="mm-cover-badge mm-cover-badge--processing">转码中</span>
              <span class="mm-duration">{{ item.duration }}</span>
            </router-link>
            <div class="mm-body">
              <router-link
                class="mm-title-link"
                :to="videoPlayRoute(item)"
              >
                <h3 class="mm-title">{{ item.title }}</h3>
              </router-link>
              <p v-if="item.statusHint" class="mm-status-hint">{{ item.statusHint }}</p>
              <p v-if="item.failReason" class="mm-fail-reason">{{ item.failReason }}</p>
              <p class="mm-time">{{ item.publishedAt }}</p>
              <div class="meta-footer">
                <div class="view-stat" title="播放">
                  <img class="stat-ico" :src="icoPlay" alt="" />
                  <span class="icon-text">{{ formatNum(item.view) }}</span>
                </div>
                <div class="view-stat" title="弹幕">
                  <img class="stat-ico" :src="icoDanmu" alt="" />
                  <span class="icon-text">{{ formatNum(item.danmu) }}</span>
                </div>
                <div class="view-stat" title="评论">
                  <img class="stat-ico" :src="icoComment" alt="" />
                  <span class="icon-text">{{ formatNum(item.reply) }}</span>
                </div>
                <div class="view-stat" title="硬币">
                  <img class="stat-ico" :src="icoCoin" alt="" />
                  <span class="icon-text">{{ formatNum(item.coin) }}</span>
                </div>
                <div class="view-stat" title="收藏">
                  <img class="stat-ico" :src="icoCollect" alt="" />
                  <span class="icon-text">{{ formatNum(item.fav) }}</span>
                </div>
                <div class="view-stat" title="转发">
                  <img class="stat-ico" :src="icoTranspond" alt="" />
                  <span class="icon-text">{{ formatNum(item.share) }}</span>
                </div>
              </div>
            </div>
            <div class="mm-actions">
              <router-link
                class="mm-btn-edit"
                :to="{ name: 'videoEdit', params: { id: item.id } }"
              >
                <img
                  class="mm-btn-edit-img"
                  :src="icoCompile"
                  width="14"
                  height="14"
                  alt=""
                />
                编辑
              </router-link>
              <div
                class="mm-more-wrap"
                @mouseenter="onMoreEnter(item.id)"
                @mouseleave="onMoreLeave"
              >
                <button type="button" class="mm-btn-more" aria-label="更多">
                  <span class="mm-dot-v" />
                </button>
                <div
                  v-show="moreHoverId === item.id"
                  class="mm-more-pop"
                  role="menu"
                  @click.stop
                >
                  <div class="mm-more-grid">
                    <button
                      v-for="act in moreMenuActions"
                      :key="act.key"
                      type="button"
                      class="mm-more-cell"
                      :class="{
                        'mm-more-cell--danger': act.key === 'delete',
                        'is-disabled': act.key === 'fav' && isVideoDraft(item)
                      }"
                      role="menuitem"
                      :disabled="act.key === 'fav' && isVideoDraft(item)"
                      @click.stop="onMoreMenu(act.key, item)"
                    >
                      <span class="mm-more-ico" aria-hidden="true">
                        <img
                          class="mm-more-img"
                          :src="act.icon"
                          width="22"
                          height="22"
                          alt=""
                        />
                      </span>
                      <span class="mm-more-lbl">{{ act.label }}</span>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div
            v-if="!videosLoading && !displayVideos.length"
            class="mm-empty-video"
          >
            <img class="mm-empty-article-img" :src="imgEmpty" alt="" />
            <p class="mm-empty-article-txt">暂无稿件</p>
          </div>
        </div>

        <div v-if="showVideoPager" class="mm-page">
          <span class="mm-page-info">{{ videoPageInfo }}</span>
          <button
            type="button"
            class="mm-page-btn"
            :disabled="videoPage <= 1 || videosLoading"
            @click="goVideoPage(videoPage - 1)"
          >
            上一页
          </button>
          <button
            v-for="p in videoPageNums"
            :key="'vp-' + p"
            type="button"
            class="mm-page-num"
            :class="{ on: p === videoPage }"
            :disabled="videosLoading"
            @click="goVideoPage(p)"
          >
            {{ p }}
          </button>
          <button
            type="button"
            class="mm-page-btn"
            :disabled="videoPage >= videoTotalPages || videosLoading"
            @click="goVideoPage(videoPage + 1)"
          >
            {{ videosLoading ? "加载中…" : "下一页" }}
          </button>
        </div>
      </template>

      <template v-else>
        <div class="mm-article-subtabs">
          <button
            type="button"
            class="mm-article-subtab"
            :class="{ on: articleSubTab === 'article' }"
            @click="articleSubTab = 'article'"
          >
            图文
          </button>
          <button
            type="button"
            class="mm-article-subtab"
            :class="{ on: articleSubTab === 'collection' }"
            @click="articleSubTab = 'collection'"
          >
            文集
          </button>
          <button
            type="button"
            class="mm-article-subtab"
            :class="{ on: articleSubTab === 'draft' }"
            @click="articleSubTab = 'draft'"
          >
            草稿
          </button>
        </div>

        <template v-if="articleSubTab === 'article'">
          <div class="mm-subbar mm-article-toolbar">
            <div class="mm-article-toolbar-main">
              <div class="mm-article-pills">
                <button
                  v-for="p in articleKindPills"
                  :key="p.value"
                  type="button"
                  class="mm-pill"
                  :class="{ on: articleKindFilter === p.value }"
                  @click="setArticleKindFilter(p.value)"
                >
                  {{ p.label }}
                </button>
              </div>
              <div class="mm-status mm-article-status">
                <button
                  type="button"
                  class="mm-status-btn"
                  :class="{ on: articleStatusFilter === 'all' }"
                  @click="setArticleStatusFilter('all')"
                >
                  全部
                </button>
                <button
                  type="button"
                  class="mm-status-btn mm-status-count"
                  :class="{ on: articleStatusFilter === 'processing' }"
                  @click="setArticleStatusFilter('processing')"
                >
                  进行中 <em>{{ articleStatusCounts.processing }}</em>
                </button>
                <button
                  type="button"
                  class="mm-status-btn mm-status-count"
                  :class="{ on: articleStatusFilter === 'passed' }"
                  @click="setArticleStatusFilter('passed')"
                >
                  已通过 <em>{{ articleStatusCounts.passed }}</em>
                </button>
                <button
                  type="button"
                  class="mm-status-btn mm-status-count"
                  :class="{ on: articleStatusFilter === 'rejected' }"
                  @click="setArticleStatusFilter('rejected')"
                >
                  未通过 <em>{{ articleStatusCounts.rejected }}</em>
                </button>
              </div>
            </div>
            <div ref="articleSortRoot" class="mm-sort">
              <button
                type="button"
                class="mm-sort-trigger"
                @click="articleSortOpen = !articleSortOpen"
              >
                {{ articleSortLabel }}
                <span
                  class="mm-sort-chevron"
                  :class="{ open: articleSortOpen }"
                />
              </button>
              <div v-show="articleSortOpen" class="mm-sort-menu">
                <button
                  v-for="opt in articleSortOptions"
                  :key="opt.value"
                  type="button"
                  class="mm-sort-item"
                  :class="{ active: articleSortKey === opt.value }"
                  @click="pickArticleSort(opt)"
                >
                  {{ opt.label }}
                  <span v-if="articleSortKey === opt.value" class="mm-sort-check"
                    >✓</span
                  >
                </button>
              </div>
            </div>
          </div>

          <div
            v-if="isMinibiliMode && articlesLoading && !displayArticles.length"
            class="mm-video-loading"
          >
            加载中…
          </div>
          <div v-else-if="displayArticles.length" class="mm-list">
            <div
              v-for="item in displayArticles"
              :key="item.kind + '-' + item.id"
              class="mm-row mm-row-article"
            >
              <router-link
                v-if="articleDisplayRoute(item)"
                class="mm-cover-wrap mm-cover-link"
                :to="articleDisplayRoute(item)"
                :title="item.status === 'passed' ? '阅读：' + item.title : '编辑：' + item.title"
              >
                <img class="mm-cover" :src="item.cover" alt="" />
                <span class="mm-article-kind-badge">{{ item.kindLabel }}</span>
              </router-link>
              <div v-else class="mm-cover-wrap">
                <img class="mm-cover" :src="item.cover" alt="" />
                <span class="mm-article-kind-badge">{{ item.kindLabel }}</span>
              </div>
              <div class="mm-body">
                <router-link
                  v-if="articleDisplayRoute(item)"
                  class="mm-title-link"
                  :to="articleDisplayRoute(item)"
                >
                  <h3 class="mm-title">{{ item.title }}</h3>
                </router-link>
                <h3 v-else class="mm-title">{{ item.title }}</h3>
                <p class="mm-time">{{ item.publishedAt }}</p>
                <p
                  v-if="item.status === 'rejected' && item.failReason"
                  class="mm-fail-reason"
                >
                  审核未通过：{{ item.failReason }}
                </p>
                <p
                  v-else-if="item.status === 'processing' && item.kind === 'column'"
                  class="mm-status-hint"
                >
                  专栏审核中，马上就能和大家见面啦~
                </p>
                <div v-if="item.tags && item.tags.length" class="mm-article-tags">
                  <span
                    v-for="t in item.tags"
                    :key="t"
                    class="mm-article-tag"
                    >{{ t }}</span
                  >
                </div>
                <div class="meta-footer">
                  <div class="view-stat" title="阅读">
                    <img class="stat-ico" :src="icoPlay" alt="" />
                    <span class="icon-text">{{ formatNum(item.view) }}</span>
                  </div>
                  <div class="view-stat" title="评论">
                    <img class="stat-ico" :src="icoComment" alt="" />
                    <span class="icon-text">{{ formatNum(item.reply) }}</span>
                  </div>
                  <div class="view-stat" title="点赞">
                    <svg
                      class="stat-ico stat-ico-svg"
                      viewBox="0 0 24 24"
                      width="16"
                      height="16"
                      aria-hidden="true"
                    >
                      <path
                        fill="currentColor"
                        d="M1 21h4V9H1v12zm22-11c0-1.1-.9-2-2-2h-6.31l.95-4.57.03-.32c0-.41-.17-.79-.44-1.06L14.17 1 7.59 7.59C7.22 7.95 7 8.45 7 9v10c0 1.1.9 2 2 2h9c.83 0 1.54-.5 1.84-1.22l3.02-7.05c.09-.23.14-.47.14-.73v-2z"
                      />
                    </svg>
                    <span class="icon-text">{{ formatNum(item.like) }}</span>
                  </div>
                  <div class="view-stat" title="收藏">
                    <img class="stat-ico" :src="icoCollect" alt="" />
                    <span class="icon-text">{{ formatNum(item.fav) }}</span>
                  </div>
                </div>
              </div>
              <div class="mm-actions">
                <button type="button" class="mm-btn-edit" @click="onArticleEdit(item)">
                  <img
                    class="mm-btn-edit-img"
                    :src="icoCompile"
                    width="14"
                    height="14"
                    alt=""
                  />
                  编辑
                </button>
                <div
                  class="mm-more-wrap"
                  @mouseenter="onArticleMoreEnter(item.id)"
                  @mouseleave="onArticleMoreLeave"
                >
                  <button type="button" class="mm-btn-more" aria-label="更多">
                    <span class="mm-dot-v" />
                  </button>
                  <div
                    v-show="articleMoreHoverId === item.id"
                    class="mm-more-pop mm-more-pop--article"
                    role="menu"
                    @click.stop
                  >
                    <button
                      type="button"
                      class="mm-more-item-delete"
                      role="menuitem"
                      @click="onArticleMoreMenu('delete', item)"
                    >
                      删除内容
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="!articlesLoading" class="mm-empty-article">
            <img class="mm-empty-article-img" :src="imgEmpty" alt="" />
            <p class="mm-empty-article-txt">
              {{
                articleKindFilter === "moment"
                  ? "当前没有动态"
                  : articleKindFilter === "column"
                    ? "当前没有专栏"
                    : "当前没有图文稿件"
              }}
            </p>
          </div>

          <div v-if="showArticlePager" class="mm-page">
            <span class="mm-page-info">{{ articlePageInfo }}</span>
            <button
              type="button"
              class="mm-page-btn"
              :disabled="articlePage <= 1 || articlesLoading"
              @click="goArticlePage(articlePage - 1)"
            >
              上一页
            </button>
            <button
              v-for="p in articlePageNums"
              :key="'ap-' + p"
              type="button"
              class="mm-page-num"
              :class="{ on: p === articlePage }"
              :disabled="articlesLoading"
              @click="goArticlePage(p)"
            >
              {{ p }}
            </button>
            <button
              type="button"
              class="mm-page-btn"
              :disabled="articlePage >= articleTotalPages || articlesLoading"
              @click="goArticlePage(articlePage + 1)"
            >
              {{ articlesLoading ? "加载中…" : "下一页" }}
            </button>
          </div>
        </template>

        <template v-else-if="articleSubTab === 'collection'">
          <div class="mm-collection-head">
            <span class="mm-collection-label"
              >全部文集 ({{ collections.length }}/{{ collectionMax }})</span
            >
            <button type="button" class="mm-btn-primary" @click="onCreateCollection">
              + 创建文集
            </button>
          </div>
          <div v-if="!collections.length" class="mm-collection-empty">
            <p class="mm-collection-empty-title">快来创建你的文集吧～</p>
            <p class="mm-collection-empty-sub">
              将文章加入文集有助于读者连续阅读哦～
            </p>
            <button
              type="button"
              class="mm-btn-primary mm-btn-primary--lg"
              @click="onCreateCollection"
            >
              创建文集
            </button>
          </div>
          <div v-else class="mm-list">
            <div
              v-for="c in collections"
              :key="c.id"
              class="mm-row mm-row-collection"
            >
              <div class="mm-collection-body">
                <h3 class="mm-title">{{ c.title }}</h3>
                <p class="mm-time">{{ c.articleCount }} 篇文章 · {{ c.updatedAt }}</p>
              </div>
              <div class="mm-actions">
                <button type="button" class="mm-btn-edit" @click="onEditCollection(c)">
                  编辑
                </button>
              </div>
            </div>
          </div>
        </template>

        <template v-else-if="articleSubTab === 'draft'">
          <div
            v-if="isMinibiliMode && articlesLoading && !draftArticles.length"
            class="mm-video-loading"
          >
            加载中…
          </div>
          <div v-else-if="draftArticles.length" class="mm-list">
            <div
              v-for="item in draftArticles"
              :key="'draft-' + item.id"
              class="mm-row mm-row-article"
            >
              <router-link
                v-if="articleDisplayRoute(item)"
                class="mm-cover-wrap mm-cover-link"
                :to="articleDisplayRoute(item)"
                :title="'编辑：' + item.title"
              >
                <img class="mm-cover" :src="item.cover" alt="" />
                <span class="mm-article-kind-badge">{{ item.kindLabel }}</span>
              </router-link>
              <div class="mm-body">
                <router-link
                  v-if="articleDisplayRoute(item)"
                  class="mm-title-link"
                  :to="articleDisplayRoute(item)"
                >
                  <h3 class="mm-title">{{ item.title || "（无标题草稿）" }}</h3>
                </router-link>
                <h3 v-else class="mm-title">{{ item.title || "（无标题草稿）" }}</h3>
                <p class="mm-time">{{ item.publishedAt }}</p>
              </div>
              <div class="mm-actions">
                <button type="button" class="mm-btn-edit" @click="onArticleEdit(item)">
                  <img
                    class="mm-btn-edit-img"
                    :src="icoCompile"
                    width="14"
                    height="14"
                    alt=""
                  />
                  编辑
                </button>
                <button
                  type="button"
                  class="mm-btn-more mm-btn-more--danger"
                  aria-label="删除草稿"
                  @click="openDeleteArticleDialog(item)"
                >
                  删除
                </button>
              </div>
            </div>
          </div>
          <div v-else class="mm-empty-draft">
            <img class="mm-empty-article-img mm-empty-draft-img" :src="imgEmpty" alt="" />
            <p class="mm-empty-article-txt">暂无草稿</p>
          </div>
        </template>
      </template>
    </div>
    <Teleport to="body">
      <div
        v-if="deleteDialogVisible && deleteDialogTarget"
        class="mm-del-overlay"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="deleteDialogTitle ? 'mm-del-title' : undefined"
      >
        <div
          class="mm-del-overlay__backdrop"
          aria-hidden="true"
          @click="closeDeleteDialog"
        />
        <div class="mm-del-modal" @click.stop>
          <button
            type="button"
            class="mm-del-modal__close"
            aria-label="关闭"
            :disabled="deleteDialogSubmitting"
            @click="closeDeleteDialog"
          >
            ×
          </button>
          <h2
            v-if="deleteDialogTitle"
            id="mm-del-title"
            class="mm-del-dialog__title mm-del-dialog__title--compact"
          >
            {{ deleteDialogTitle }}
          </h2>
          <p class="mm-del-dialog__article-msg">{{ deleteDialogConfirmMsg }}</p>
          <p v-if="deleteDialogWarn" class="mm-del-dialog__warn mm-del-dialog__warn--sub">
            {{ deleteDialogWarn }}
          </p>
          <div class="mm-del-dialog__article-foot">
            <button
              type="button"
              class="mm-del-dialog__btn-cancel"
              :disabled="deleteDialogSubmitting"
              @click="closeDeleteDialog"
            >
              取消
            </button>
            <button
              type="button"
              class="mm-del-dialog__btn-confirm"
              :disabled="deleteDialogSubmitting"
              @click="performDeleteTarget"
            >
              {{ deleteDialogSubmitting ? "删除中…" : "确定" }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <VideoFavoriteFolderDialog
      v-model="favDialogOpen"
      :video-id="favDialogVideoId"
      :loading="favDialogLoading"
      @confirm="onFavDialogConfirm"
      @cancel="favDialogOpen = false"
    />
  </CreatorShell>
</template>

<script>
import { ElMessage } from "element-plus";
import CreatorShell from "@/components/creator/CreatorShell.vue";
import { showCreatorVideoReviewNotice } from "@/utils/creatorVideoReviewNotice";
import VideoFavoriteFolderDialog from "@/components/video/VideoFavoriteFolderDialog.vue";
import imgEmpty from "@/assets/empty.png";
import defaultVideoCover from "@/assets/akari.jpg";
import defaultUploadCoverPlaceholder from "@/assets/85251fe9cc54ac2b826a965a90f8dba811edbc7a.gif@920w_518h.webp";
import { getAccessToken } from "@/utils/authTokens";
import {
  minibiliVideoPlayRoute,
  minibiliArticleReadRoute,
  minibiliDynamicReadRoute,
  minibiliUserSpaceDynamicRoute
} from "@/utils/minibiliRoutes";
import {
  mbDeleteMyVideo,
  mbListMyVideos,
  mbListMyArticles,
  mbListMyDynamics,
  mbDeleteMyArticle,
  mbGetMe,
  mbDeleteMyDynamic,
  mbSetVideoFavoriteFolders
} from "@/api/minibili";
import { CREATOR_VIDEO_LIST } from "./creatorVideoMock.js";
import { CREATOR_ARTICLE_LIST } from "./creatorArticleMock.js";
import icoPlay from "@/assets/upload_manager/article/paly.png";
import icoDanmu from "@/assets/upload_manager/article/danmu.png";
import icoComment from "@/assets/upload_manager/article/comment.png";
import icoCoin from "@/assets/upload_manager/article/coin.png";
import icoCollect from "@/assets/upload_manager/article/collect.png";
import icoTranspond from "@/assets/upload_manager/article/transpond.png";
import icoCompile from "@/assets/upload_manager/article/compile.png";
import icoLock from "@/assets/upload_manager/article/lock.png";
import icoShare from "@/assets/upload_manager/article/share.png";
import icoCollect2 from "@/assets/upload_manager/article/collect_2.png";
import icoReferenceRecord from "@/assets/upload_manager/article/reference_record.svg";
import icoDanmuManagement from "@/assets/upload_manager/article/danmumanagement.png";
import icoCommentManagement from "@/assets/upload_manager/article/commentManagement.png";
import icoDeleteManuscript from "@/assets/upload_manager/article/delete_manuscript.svg";

export default {
  name: "ManuscriptPage",
  components: { CreatorShell, VideoFavoriteFolderDialog },
  data() {
    return {
      imgEmpty,
      articleSubTab: "article",
      articleKindFilter: "all",
      articleStatusFilter: "all",
      articleSortOpen: false,
      articleSortKey: "time",
      articleSortOptions: [
        { value: "time", label: "发布时间排序" },
        { value: "view", label: "阅读量排序" },
        { value: "reply", label: "评论数排序" },
        { value: "like", label: "点赞数排序" },
        { value: "fav", label: "收藏数排序" }
      ],
      articleKindPills: [
        { value: "all", label: "全部图文" },
        { value: "column", label: "专栏" },
        { value: "moment", label: "动态" },
        { value: "post", label: "小站帖子" }
      ],
      articles: [],
      articlesLoading: false,
      articlePage: 1,
      articlePageSize: 10,
      articleTotal: 0,
      articleTotalPages: 1,
      articleStatusCountsApi: {
        draft: 0,
        passed: 0,
        processing: 0,
        rejected: 0,
        dynamics: 0
      },
      meUserId: 0,
      collections: [],
      collectionMax: 30,
      articleMoreHoverId: null,
      articleMoreLeaveTimer: null,
      moreHoverId: null,
      moreLeaveTimer: null,
      mainTab: "video",
      searchQ: "",
      statusFilter: "all",
      sortOpen: false,
      sortKey: "time",
      sortOptions: [
        { value: "time", label: "投稿时间排序" },
        { value: "view", label: "播放数排序" },
        { value: "fav", label: "收藏数排序" },
        { value: "danmu", label: "弹幕数排序" },
        { value: "reply", label: "评论数排序" }
      ],
      icoPlay,
      icoDanmu,
      icoComment,
      icoCoin,
      icoCollect,
      icoTranspond,
      icoCompile,
      videos: [],
      videosLoading: false,
      videoPage: 1,
      videoPageSize: 10,
      videoTotal: 0,
      videoTotalPages: 1,
      videoStatusCounts: {
        draft: 0,
        processing: 0,
        passed: 0,
        rejected: 0
      },
      videosFetchError: "",
      searchDebounceTimer: null,
      listRefreshToken: 0,
      deleteDialogVisible: false,
      deleteDialogSubmitting: false,
      deleteDialogTarget: null,
      favDialogOpen: false,
      favDialogVideoId: null,
      favDialogLoading: false,
      reviewNoticeShown: false
    };
  },
  computed: {
    isMinibiliMode() {
      return (
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1"
      );
    },
    videoPageInfo() {
      if (this.isMinibiliMode) {
        return `共 ${this.videoTotal} 条`;
      }
      return `共 ${this.displayVideos.length} 条`;
    },
    showVideoPager() {
      if (this.isMinibiliMode) {
        return this.videoTotal > 0 || this.videos.length > 0;
      }
      return this.displayVideos.length > 0;
    },
    videoPageNums() {
      const max = Math.min(this.videoTotalPages, 7);
      if (max <= 0) return [];
      const start = Math.max(
        1,
        Math.min(this.videoPage - 2, this.videoTotalPages - max + 1)
      );
      const out = [];
      for (let i = 0; i < max; i += 1) {
        out.push(start + i);
      }
      return out;
    },
    displayVideos() {
      return this.isMinibiliMode ? this.videos : this.filteredVideos;
    },
    deleteDialogConfirmMsg() {
      const k = this.deleteDialogTarget?.kind;
      if (k === "article") return "确定要删除此图文内容吗？";
      if (k === "moment") return "确定要删除此动态吗？";
      return "确定要删除此视频吗？";
    },
    articleSortLabel() {
      const o = this.articleSortOptions.find((x) => x.value === this.articleSortKey);
      return o ? o.label : "发布时间排序";
    },
    articlePageInfo() {
      if (this.isMinibiliMode) {
        return `共 ${this.articleTotal} 条`;
      }
      return `共 ${this.displayArticles.length} 条`;
    },
    showArticlePager() {
      if (this.isMinibiliMode) {
        return this.articleTotal > 0 || this.articles.length > 0;
      }
      return this.displayArticles.length > 0;
    },
    articlePageNums() {
      const max = Math.min(this.articleTotalPages, 7);
      if (max <= 0) return [];
      const start = Math.max(
        1,
        Math.min(this.articlePage - 2, this.articleTotalPages - max + 1)
      );
      const out = [];
      for (let i = 0; i < max; i += 1) {
        out.push(start + i);
      }
      return out;
    },
    articleStatusCounts() {
      if (!this.isMinibiliMode) {
        let a = this.articles;
        if (this.articleKindFilter !== "all") {
          a = a.filter((x) => x.kind === this.articleKindFilter);
        }
        return {
          processing: a.filter((x) => x.status === "processing").length,
          passed: a.filter((x) => x.status === "passed").length,
          rejected: a.filter((x) => x.status === "rejected").length
        };
      }
      const c = this.articleStatusCountsApi;
      const dyn = Number(c.dynamics) || 0;
      const processing = Number(c.processing) || 0;
      const passed = Number(c.passed) || 0;
      const rejected = Number(c.rejected) || 0;
      if (this.articleKindFilter === "moment") {
        return { processing: 0, passed: dyn, rejected: 0 };
      }
      if (this.articleKindFilter === "column" || this.articleKindFilter === "post") {
        return { processing, passed, rejected };
      }
      return { processing, passed: passed + dyn, rejected };
    },
    displayArticles() {
      if (this.isMinibiliMode) {
        return this.articles;
      }
      let list = this.articles.filter((a) => a.status !== "draft");
      const q = this.searchQ.trim();
      if (q) {
        list = list.filter((v) => v.title.includes(q));
      }
      if (this.articleKindFilter !== "all") {
        list = list.filter((v) => v.kind === this.articleKindFilter);
      }
      if (this.articleStatusFilter !== "all") {
        list = list.filter((v) => v.status === this.articleStatusFilter);
      }
      const key = this.articleSortKey;
      list.sort((a, b) => {
        if (key === "time") return b.id - a.id;
        return (b[key] || 0) - (a[key] || 0);
      });
      return list;
    },
    draftArticles() {
      if (this.isMinibiliMode) {
        return this.articles;
      }
      return this.articles.filter((a) => a.status === "draft");
    },
    deleteDialogTitle() {
      const k = this.deleteDialogTarget?.kind;
      if (k === "article") return "删除图文";
      if (k === "moment") return "删除动态";
      if (k === "video") return "删除视频";
      return "";
    },
    deleteDialogWarn() {
      const k = this.deleteDialogTarget?.kind;
      if (k === "article") {
        return "图文删除后将无法恢复，数据库与云端封面等资源将一并清除";
      }
      if (k === "moment") {
        return "动态删除后将无法恢复，数据库与云端图片等资源将一并清除";
      }
      if (k === "video") {
        return "视频删除后将无法恢复，数据库与云端视频、封面等资源将一并清除";
      }
      return "";
    },
    moreMenuActions() {
      return [
        { key: "edit", label: "编辑稿件", icon: icoCompile },
        { key: "visibility", label: "可见范围", icon: icoLock },
        { key: "share", label: "分享投稿", icon: icoShare },
        { key: "fav", label: "添加到收藏夹", icon: icoCollect2 },
        { key: "history", label: "编辑记录", icon: icoReferenceRecord },
        { key: "danmu", label: "弹幕管理", icon: icoDanmuManagement },
        { key: "comment", label: "评论管理", icon: icoCommentManagement },
        { key: "delete", label: "删除稿件", icon: icoDeleteManuscript }
      ];
    },
    sortLabel() {
      const o = this.sortOptions.find((x) => x.value === this.sortKey);
      return o ? o.label : "投稿时间排序";
    },
    statusCounts() {
      if (this.isMinibiliMode) {
        const c = this.videoStatusCounts;
        return {
          draft: Number(c.draft) || 0,
          processing: Number(c.processing) || 0,
          passed: Number(c.passed) || 0,
          rejected: Number(c.rejected) || 0
        };
      }
      const v = this.videos;
      return {
        processing: v.filter(
          (x) => x.status === "processing" || x.status === "review"
        ).length,
        passed: v.filter((x) => x.status === "passed").length,
        rejected: v.filter((x) => x.status === "rejected").length
      };
    },
    filteredVideos() {
      let list = [...this.videos];
      const q = this.searchQ.trim();
      if (q) {
        list = list.filter((v) => v.title.includes(q));
      }
      if (this.statusFilter !== "all" && this.statusFilter !== "draft") {
        if (this.statusFilter === "processing") {
          list = list.filter(
            (v) => v.status === "processing" || v.status === "review"
          );
        } else {
          list = list.filter((v) => v.status === this.statusFilter);
        }
      }
      if (this.statusFilter === "draft") {
        list = list.filter((v) => v.status === "draft");
      }
      const key = this.sortKey;
      list.sort((a, b) => {
        if (key === "time") return b.id - a.id;
        return (b[key] || 0) - (a[key] || 0);
      });
      return list;
    }
  },
  watch: {
    "$route.query"() {
      if (String(this.$route.query.reviewNotice || "") !== "1") {
        this.reviewNoticeShown = false;
      }
      this.applyManuscriptRouteQuery();
      if (this.isMinibiliMode && this.mainTab === "video") {
        void this.reloadMyVideos();
      }
    },
    mainTab(val) {
      if (val === "video" && this.isMinibiliMode) {
        void this.reloadMyVideos();
      }
      if (val === "article" && this.isMinibiliMode) {
        void this.reloadMyArticles();
      }
    },
    searchQ() {
      if (!this.isMinibiliMode) return;
      if (this.searchDebounceTimer) clearTimeout(this.searchDebounceTimer);
      this.searchDebounceTimer = setTimeout(() => {
        this.searchDebounceTimer = null;
        if (this.mainTab === "video") {
          this.videoPage = 1;
          void this.reloadMyVideos();
        } else if (this.mainTab === "article" && this.articleSubTab === "article") {
          this.articlePage = 1;
          void this.reloadMyArticles();
        }
      }, 350);
    },
    articleSubTab(val) {
      if (!this.isMinibiliMode || this.mainTab !== "article") return;
      if (val === "article" || val === "draft") {
        this.articlePage = 1;
        void this.reloadMyArticles();
      }
    }
  },
  mounted() {
    this.applyManuscriptRouteQuery();
    if (!this.isMinibiliMode) {
      this.videos = [...CREATOR_VIDEO_LIST];
      this.articles = [...CREATOR_ARTICLE_LIST];
    } else {
      void this.reloadMyVideos();
      void this.refreshArticleStatusCounts();
      if (this.mainTab === "article") {
        void this.reloadMyArticles();
      }
    }
    document.addEventListener("click", this.onDocClick);
  },
  activated() {
    this.applyManuscriptRouteQuery();
    if (!this.isMinibiliMode) return;
    if (this.mainTab === "video") {
      void this.reloadMyVideos();
    } else if (this.mainTab === "article") {
      void this.reloadMyArticles();
    }
  },
  beforeUnmount() {
    if (this.searchDebounceTimer) clearTimeout(this.searchDebounceTimer);
    document.removeEventListener("click", this.onDocClick);
    document.body.classList.remove("mm-del-open");
    if (this.moreLeaveTimer) clearTimeout(this.moreLeaveTimer);
    if (this.articleMoreLeaveTimer) clearTimeout(this.articleMoreLeaveTimer);
  },
  methods: {
    mapArticleFromApi(a) {
      const api = String(a.status || "");
      let st = "processing";
      if (api === "published") st = "passed";
      else if (api === "draft") st = "draft";
      else if (api === "pending_review") st = "processing";
      else if (api === "rejected") st = "rejected";
      return {
        id: a.id,
        title: a.title,
        apiStatus: api,
        failReason: String(a.fail_reason || "").trim(),
        publishedAt: a.published_at || a.created_at || "",
        cover: a.cover_url || defaultVideoCover,
        kind: "column",
        kindLabel: "专栏",
        tags: [],
        view: a.view_count || 0,
        reply: a.comment_count || 0,
        like: a.coin_count || 0,
        fav: a.fav_count || 0,
        status: st
      };
    },
    dynamicSortForApi(sortKey) {
      const k = String(sortKey || "time");
      if (k === "reply" || k === "like") return k;
      return "time";
    },
    sortArticleRows(list, sortKey) {
      const key = String(sortKey || "time");
      const out = [...list];
      out.sort((a, b) => {
        if (key === "time") return b.id - a.id;
        return (Number(b[key]) || 0) - (Number(a[key]) || 0);
      });
      return out;
    },
    setArticleKindFilter(val) {
      if (this.articleKindFilter === val) {
        if (this.isMinibiliMode) void this.reloadMyArticles();
        return;
      }
      this.articleKindFilter = val;
      this.articlePage = 1;
      if (this.isMinibiliMode) void this.reloadMyArticles();
    },
    setArticleStatusFilter(st) {
      if (this.articleStatusFilter === st) {
        if (this.isMinibiliMode) void this.reloadMyArticles();
        return;
      }
      this.articleStatusFilter = st;
      this.articlePage = 1;
      if (this.isMinibiliMode) void this.reloadMyArticles();
    },
    goArticlePage(p) {
      const page = Number(p) || 1;
      if (page < 1 || page > this.articleTotalPages || page === this.articlePage) {
        return;
      }
      this.articlePage = page;
      void this.reloadMyArticles();
    },
    mapDynamicFromApi(d) {
      const imgs = Array.isArray(d.images) ? d.images : [];
      let cover = imgs.length > 0 ? String(imgs[0] || "").trim() : "";
      if (cover && /^https?:\/\//i.test(cover) && this.listRefreshToken > 0) {
        const sep = cover.includes("?") ? "&" : "?";
        cover = `${cover}${sep}v=${this.listRefreshToken}`;
      }
      return {
        id: d.id,
        title: String(d.title || "").trim() || "（无标题动态）",
        apiStatus: "published",
        publishedAt: d.created_at || "",
        cover: cover || defaultVideoCover,
        kind: "moment",
        kindLabel: "动态",
        tags: [],
        view: 0,
        reply: Number(d.comment_count) || 0,
        like: Number(d.like_count) || 0,
        fav: 0,
        status: "passed"
      };
    },
    articleDisplayRoute(item) {
      if (!item || !item.id) return null;
      if (item.kind === "moment") {
        if (item.status === "passed") {
          return minibiliDynamicReadRoute(item.id);
        }
        return minibiliUserSpaceDynamicRoute(this.meUserId);
      }
      if (item.status === "passed") {
        return minibiliArticleReadRoute(item.id);
      }
      if (this.isMinibiliMode) {
        return { name: "articleEdit", params: { id: String(item.id) } };
      }
      return null;
    },
    applyArticleStatusCountsFromApi(counts) {
      const c = counts || {};
      this.articleStatusCountsApi = {
        draft: Number(c.draft) || 0,
        passed: Number(c.passed) || 0,
        processing: Number(c.processing) || 0,
        rejected: Number(c.rejected) || 0,
        dynamics: Number(c.dynamics) || 0
      };
    },
    /** 仅拉取专栏状态统计（切换图文 Tab 前也可展示真实数量） */
    async refreshArticleStatusCounts() {
      if (!this.isMinibiliMode || !getAccessToken()) return;
      try {
        const res = await mbListMyArticles({
          page: 1,
          page_size: 1,
          sort: "time"
        });
        this.applyArticleStatusCountsFromApi(res.counts);
      } catch {
        /* 统计失败不阻断列表 */
      }
    },
    async reloadMyArticles() {
      if (!this.isMinibiliMode || !getAccessToken()) {
        return;
      }
      if (this.articleSubTab === "collection") {
        return;
      }
      this.articlesLoading = true;
      try {
        const me = await mbGetMe();
        this.meUserId = Number(me.user_id) || 0;
        if (!this.meUserId) {
          throw new Error("无法获取当前用户 ID");
        }
        const sort = this.articleSortKey;
        const status =
          this.articleSubTab === "draft"
            ? "draft"
            : this.articleStatusFilter === "all"
              ? undefined
              : this.articleStatusFilter;
        const q = this.searchQ.trim() || undefined;

        if (this.articleSubTab === "draft") {
          const res = await mbListMyArticles({
            page: 1,
            page_size: 50,
            sort: "time",
            status: "draft"
          });
          this.listRefreshToken = Date.now();
          this.articles = (res.items || []).map((a) => this.mapArticleFromApi(a));
          this.applyArticleStatusCountsFromApi(res.counts);
          return;
        }

        if (this.articleKindFilter === "post") {
          this.articles = [];
          this.articleTotal = 0;
          this.articleTotalPages = 1;
          await this.refreshArticleStatusCounts();
          return;
        }

        if (this.articleKindFilter === "column") {
          const res = await mbListMyArticles({
            page: this.articlePage,
            page_size: this.articlePageSize,
            sort,
            status,
            q
          });
          this.listRefreshToken = Date.now();
          this.articles = (res.items || []).map((a) => this.mapArticleFromApi(a));
          this.articlePage = Number(res.page) || this.articlePage;
          this.articleTotal = Number(res.total) || 0;
          this.articleTotalPages = Math.max(1, Number(res.total_pages) || 1);
          this.applyArticleStatusCountsFromApi(res.counts);
          return;
        }

        if (this.articleKindFilter === "moment") {
          if (
            status === "draft" ||
            status === "processing" ||
            status === "rejected"
          ) {
            this.articles = [];
            this.articleTotal = 0;
            this.articleTotalPages = 1;
            const countRes = await mbListMyArticles({
              page: 1,
              page_size: 1,
              sort: "time"
            });
            this.applyArticleStatusCountsFromApi(countRes.counts);
            return;
          }
          const [dynRes, countRes] = await Promise.all([
            mbListMyDynamics({
              page: this.articlePage,
              page_size: this.articlePageSize,
              sort: this.dynamicSortForApi(sort),
              q
            }),
            mbListMyArticles({ page: 1, page_size: 1, sort: "time" })
          ]);
          this.listRefreshToken = Date.now();
          this.articles = (dynRes.items || []).map((d) => this.mapDynamicFromApi(d));
          this.articlePage = Number(dynRes.page) || this.articlePage;
          this.articleTotal = Number(dynRes.total) || 0;
          this.articleTotalPages = Math.max(1, Number(dynRes.total_pages) || 1);
          this.applyArticleStatusCountsFromApi(countRes.counts);
          return;
        }

        const mergePageSize = 50;
        const [artRes, dynRes] = await Promise.all([
          mbListMyArticles({
            page: 1,
            page_size: mergePageSize,
            sort,
            status,
            q
          }),
          mbListMyDynamics({
            page: 1,
            page_size: mergePageSize,
            sort: this.dynamicSortForApi(sort),
            q
          })
        ]);
        this.listRefreshToken = Date.now();
        let merged = [
          ...(artRes.items || []).map((a) => this.mapArticleFromApi(a)),
          ...(dynRes.items || []).map((d) => this.mapDynamicFromApi(d))
        ];
        if (
          status === "processing" ||
          status === "rejected" ||
          status === "draft"
        ) {
          merged = merged.filter((x) => x.status === status);
        }
        merged = this.sortArticleRows(merged, sort);
        this.articleTotal = merged.length;
        this.articleTotalPages = Math.max(
          1,
          Math.ceil(merged.length / this.articlePageSize)
        );
        if (this.articlePage > this.articleTotalPages) {
          this.articlePage = this.articleTotalPages;
        }
        const start = (this.articlePage - 1) * this.articlePageSize;
        this.articles = merged.slice(start, start + this.articlePageSize);
        this.applyArticleStatusCountsFromApi({
          ...artRes.counts,
          dynamics:
            Number(artRes.counts?.dynamics) || Number(dynRes.total) || 0
        });
      } catch (e) {
        ElMessage.error((e && e.message) || "加载图文列表失败");
        this.articles = [];
        this.articleTotal = 0;
        this.articleTotalPages = 1;
      } finally {
        this.articlesLoading = false;
      }
    },
    onArticleEdit(item) {
      if (!item || !item.id) return;
      if (this.isMinibiliMode && item.kind === "moment") {
        const route = minibiliDynamicReadRoute(item.id, { edit: "1" });
        if (route) {
          this.$router.push(route);
        } else {
          ElMessage.warning("无法打开动态编辑页");
        }
        return;
      }
      if (this.isMinibiliMode && item.kind === "column") {
        this.$router.push({ name: "articleEdit", params: { id: String(item.id) } });
        return;
      }
      window.alert(`编辑图文「${item.title}」（演示）`);
    },
    onCreateCollection() {
      ElMessage.info("该功能即将开放");
    },
    onEditCollection(c) {
      window.alert(`编辑文集「${c.title}」（演示）`);
    },
    onArticleMoreEnter(id) {
      if (this.articleMoreLeaveTimer) {
        clearTimeout(this.articleMoreLeaveTimer);
        this.articleMoreLeaveTimer = null;
      }
      this.articleMoreHoverId = id;
    },
    onArticleMoreLeave() {
      if (this.articleMoreLeaveTimer) clearTimeout(this.articleMoreLeaveTimer);
      this.articleMoreLeaveTimer = setTimeout(() => {
        this.articleMoreHoverId = null;
        this.articleMoreLeaveTimer = null;
      }, 220);
    },
    onArticleMoreMenu(key, item) {
      if (this.articleMoreLeaveTimer) {
        clearTimeout(this.articleMoreLeaveTimer);
        this.articleMoreLeaveTimer = null;
      }
      this.articleMoreHoverId = null;
      if (key === "delete") {
        this.openDeleteArticleDialog(item);
      }
    },
    applyManuscriptRouteQuery() {
      const q = this.$route.query || {};
      if (q.tab === "article") {
        this.mainTab = "article";
        const ast = String(q.status || "").trim();
        if (
          ast === "processing" ||
          ast === "passed" ||
          ast === "rejected"
        ) {
          this.articleSubTab = "article";
          this.articleStatusFilter = ast;
        }
      }
      if (q.articleSub === "draft") {
        this.articleSubTab = "draft";
      }
      if (q.tab === "video") {
        this.mainTab = "video";
        const st = String(q.status || "").trim();
        if (
          st === "draft" ||
          st === "processing" ||
          st === "passed" ||
          st === "rejected"
        ) {
          this.statusFilter = st;
        }
      }
      this.tryShowReviewNoticeFromQuery();
    },
    tryShowReviewNoticeFromQuery() {
      if (String(this.$route.query.reviewNotice || "") !== "1") {
        return;
      }
      if (this.reviewNoticeShown) {
        return;
      }
      this.reviewNoticeShown = true;
      const showNotice =
        this.mainTab === "article"
          ? () => import("@/utils/creatorArticleReviewNotice").then(m => m.showCreatorArticleReviewNotice())
          : () => import("@/utils/creatorVideoReviewNotice").then(m => m.showCreatorVideoReviewNotice());
      void showNotice().finally(() => {
        if (String(this.$route.query.reviewNotice || "") !== "1") {
          return;
        }
        const q = { ...this.$route.query };
        delete q.reviewNotice;
        void this.$router.replace({ query: q });
      });
    },
    openDeleteArticleDialog(item) {
      this.deleteDialogTarget = {
        id: item.id,
        title: item.title,
        kind: item.kind === "moment" ? "moment" : "article"
      };
      this.deleteDialogSubmitting = false;
      this.deleteDialogVisible = true;
      document.body.classList.add("mm-del-open");
    },
    dismissDeleteDialog() {
      this.deleteDialogVisible = false;
      this.onDeleteDialogClosed();
    },
    closeDeleteDialog() {
      if (this.deleteDialogSubmitting) return;
      this.dismissDeleteDialog();
    },
    pickArticleSort(opt) {
      if (this.articleSortKey === opt.value) {
        this.articleSortOpen = false;
        return;
      }
      this.articleSortKey = opt.value;
      this.articleSortOpen = false;
      this.articlePage = 1;
      if (this.isMinibiliMode) void this.reloadMyArticles();
    },
    onMoreEnter(id) {
      if (this.moreLeaveTimer) {
        clearTimeout(this.moreLeaveTimer);
        this.moreLeaveTimer = null;
      }
      this.moreHoverId = id;
    },
    onMoreLeave() {
      if (this.moreLeaveTimer) clearTimeout(this.moreLeaveTimer);
      this.moreLeaveTimer = setTimeout(() => {
        this.moreHoverId = null;
        this.moreLeaveTimer = null;
      }, 220);
    },
    onDocClick(e) {
      if (this.$refs.sortRoot && !this.$refs.sortRoot.contains(e.target)) {
        this.sortOpen = false;
      }
      if (
        this.$refs.articleSortRoot &&
        !this.$refs.articleSortRoot.contains(e.target)
      ) {
        this.articleSortOpen = false;
      }
      if (!e.target.closest(".mm-more-wrap")) {
        if (this.moreLeaveTimer) {
          clearTimeout(this.moreLeaveTimer);
          this.moreLeaveTimer = null;
        }
        this.moreHoverId = null;
        if (this.articleMoreLeaveTimer) {
          clearTimeout(this.articleMoreLeaveTimer);
          this.articleMoreLeaveTimer = null;
        }
        this.articleMoreHoverId = null;
      }
    },
    /** 跳转播放页（路由 /video/:aid 为 BV{id}） */
    videoPlayRoute(item) {
      if (item && item.status === "draft") {
        return { name: "videoEdit", params: { id: String(item.id) } };
      }
      return minibiliVideoPlayRoute(item && item.id);
    },
    onMoreMenu(key, item) {
      if (this.moreLeaveTimer) {
        clearTimeout(this.moreLeaveTimer);
        this.moreLeaveTimer = null;
      }
      this.moreHoverId = null;
      if (key === "edit") {
        this.$router.push({ name: "videoEdit", params: { id: String(item.id) } });
        return;
      }
      if (key === "fav") {
        this.openVideoFavDialog(item);
        return;
      }
      if (key === "danmu") {
        this.goCreatorDanmakuManage(item);
        return;
      }
      if (key === "comment") {
        this.goCreatorCommentManage(item);
        return;
      }
      if (key === "delete") {
        this.openDeleteVideoDialog(item);
        return;
      }
      if (key === "visibility" || key === "share" || key === "history") {
        ElMessage.info("该功能即将开放");
        return;
      }
      ElMessage.info("该功能即将开放");
    },
    isVideoDraft(item) {
      return !!(item && item.status === "draft");
    },
    openVideoFavDialog(item) {
      const vid = Number(item && item.id) || 0;
      if (!vid) return;
      if (this.isVideoDraft(item)) {
        ElMessage.warning("未发布的草稿不能添加到收藏夹");
        return;
      }
      if (item.status !== "passed") {
        ElMessage.warning("仅已发布的视频可添加到收藏夹");
        return;
      }
      if (this.isMinibiliMode && !getAccessToken()) {
        ElMessage.warning("请先登录");
        return;
      }
      if (!this.isMinibiliMode) {
        ElMessage.info("该功能即将开放");
        return;
      }
      this.favDialogVideoId = vid;
      this.favDialogLoading = false;
      this.favDialogOpen = true;
    },
    async onFavDialogConfirm(folderIds) {
      const vid = Number(this.favDialogVideoId) || 0;
      if (!vid || this.favDialogLoading) return;
      this.favDialogLoading = true;
      try {
        const res = await mbSetVideoFavoriteFolders(vid, folderIds);
        const ix = this.videos.findIndex((v) => Number(v.id) === vid);
        if (ix >= 0) {
          this.videos.splice(ix, 1, {
            ...this.videos[ix],
            fav: Number(res.fav_count) || this.videos[ix].fav
          });
        }
        this.favDialogOpen = false;
        ElMessage.success(res.favorited ? "已添加到收藏夹" : "已更新收藏夹");
      } catch (e) {
        ElMessage.error((e && e.message) || "收藏操作失败");
      } finally {
        this.favDialogLoading = false;
      }
    },
    goCreatorDanmakuManage(item) {
      const vid = Number(item && item.id) || 0;
      if (!vid) return;
      this.$router.push({
        name: "creatorDanmakus",
        query: { video_id: String(vid) }
      });
    },
    goCreatorCommentManage(item) {
      const vid = Number(item && item.id) || 0;
      if (!vid) return;
      this.$router.push({
        name: "creatorComments",
        query: {
          tab: "visible",
          media: "video",
          video_id: String(vid)
        }
      });
    },
    openDeleteVideoDialog(item) {
      this.deleteDialogTarget = {
        id: Number(item.id) || item.id,
        title: item.title,
        kind: "video"
      };
      this.deleteDialogSubmitting = false;
      this.deleteDialogVisible = true;
      document.body.classList.add("mm-del-open");
    },
    async performDeleteTarget() {
      if (!this.deleteDialogTarget) return;
      const { id, kind } = this.deleteDialogTarget;
      if (this.isMinibiliMode && !getAccessToken()) {
        ElMessage.warning("请先登录");
        return;
      }
      this.deleteDialogSubmitting = true;
      try {
        if (this.isMinibiliMode) {
          if (kind === "article") {
            await mbDeleteMyArticle(id);
            await this.reloadMyArticles();
          } else if (kind === "moment") {
            await mbDeleteMyDynamic(id);
            await this.reloadMyArticles();
          } else {
            await mbDeleteMyVideo(id);
            await this.reloadMyVideos();
          }
        } else if (kind === "article" || kind === "moment") {
          this.articles = this.articles.filter(
            (a) => !(a.id === id && a.kind === kind)
          );
        } else {
          this.videos = this.videos.filter((v) => v.id !== id);
        }
        ElMessage.success("已删除");
        this.dismissDeleteDialog();
      } catch (e) {
        ElMessage.error(e instanceof Error ? e.message : "删除失败");
      } finally {
        this.deleteDialogSubmitting = false;
      }
    },
    onDeleteDialogClosed() {
      document.body.classList.remove("mm-del-open");
      this.deleteDialogTarget = null;
      this.deleteDialogSubmitting = false;
    },
    setVideoStatusFilter(st) {
      if (this.statusFilter === st) {
        if (this.isMinibiliMode) void this.reloadMyVideos();
        return;
      }
      this.statusFilter = st;
      this.videoPage = 1;
      if (this.isMinibiliMode) void this.reloadMyVideos();
    },
    pickSort(opt) {
      if (this.sortKey === opt.value) {
        this.sortOpen = false;
        return;
      }
      this.sortKey = opt.value;
      this.sortOpen = false;
      this.videoPage = 1;
      if (this.isMinibiliMode) void this.reloadMyVideos();
    },
    goVideoPage(p) {
      const page = Number(p) || 1;
      if (page < 1 || page > this.videoTotalPages || page === this.videoPage) {
        return;
      }
      this.videoPage = page;
      void this.reloadMyVideos();
    },
    mapMyVideoRow(raw) {
      const id = Number(raw.id);
      const st = String(raw.status || "");
      let status = "processing";
      if (st === "published") status = "passed";
      else if (st === "draft") status = "draft";
      else if (st === "failed" || st === "rejected") status = "rejected";
      else if (st === "pending_review") status = "review";
      else if (st === "processing") status = "processing";
      const usePendingCover = status === "processing";
      let cover = String(raw.cover_url || "").trim();
      if (
        cover &&
        /^https?:\/\//i.test(cover) &&
        this.listRefreshToken > 0 &&
        !usePendingCover
      ) {
        const sep = cover.includes("?") ? "&" : "?";
        cover = `${cover}${sep}v=${this.listRefreshToken}`;
      }
      return {
        id,
        title: String(raw.title || ""),
        cover: usePendingCover
          ? defaultUploadCoverPlaceholder
          : cover || defaultVideoCover,
        duration: this.formatDurationSec(raw.duration),
        publishedAt: String(raw.created_at || ""),
        view: Number(raw.play_count) || 0,
        danmu: Number(raw.danmaku_count) || 0,
        reply: Number(raw.comment_count) || 0,
        coin: Number(raw.coin_count) || 0,
        fav: Number(raw.fav_count) || 0,
        share: 0,
        status,
        statusHint: this.videoStatusHintForApi(st),
        failReason: this.friendlyFailReasonForDisplay(
          String(raw.fail_reason || "").trim()
        )
      };
    },
    videoStatusHintForApi(st) {
      const s = String(st || "").trim();
      if (s === "pending_review") return "视频审核中，马上就能和大家见面啦~";
      if (s === "processing") return "视频转码处理中，请稍候…";
      return "";
    },
    /** 兼容旧接口把 FFmpeg 整段 stderr 塞进 fail_reason 的情况 */
    friendlyFailReasonForDisplay(s) {
      const t = String(s || "").trim();
      if (!t) return "";
      const lines = t.split(/\n/).length;
      const looksLikeFFmpegLog =
        /ffmpeg\s+version/i.test(t) ||
        (/--enable-/i.test(t) && lines >= 4) ||
        (/configuration:/i.test(t) && /--enable-/i.test(t));
      if (looksLikeFFmpegLog && (t.length > 200 || lines > 6)) {
        return "视频转码失败，请确认文件完整、格式常见（如 MP4），或转为 H.264 视频 + AAC 音频后重新上传。";
      }
      return t;
    },
    formatDurationSec(sec) {
      const s = Math.max(0, Math.floor(Number(sec) || 0));
      const h = Math.floor(s / 3600);
      const m = Math.floor((s % 3600) / 60);
      const r = s % 60;
      if (h > 0) {
        return `${h}:${String(m).padStart(2, "0")}:${String(r).padStart(2, "0")}`;
      }
      return `${String(m).padStart(2, "0")}:${String(r).padStart(2, "0")}`;
    },
    async reloadMyVideos() {
      if (!this.isMinibiliMode) return;
      this.videosFetchError = "";
      if (!getAccessToken()) {
        this.videos = [];
        this.videoTotal = 0;
        this.videoTotalPages = 1;
        this.videosFetchError = "请先登录后查看稿件";
        return;
      }
      this.videosLoading = true;
      try {
        const res = await mbListMyVideos({
          page: this.videoPage,
          page_size: this.videoPageSize,
          sort: this.sortKey,
          status: this.statusFilter === "all" ? undefined : this.statusFilter,
          q: this.searchQ.trim() || undefined
        });
        this.listRefreshToken = Date.now();
        this.videos = (res.items || []).map((raw) => this.mapMyVideoRow(raw));
        this.videoPage = Number(res.page) || this.videoPage;
        this.videoPageSize = Number(res.page_size) || this.videoPageSize;
        this.videoTotal = Number(res.total) || 0;
        this.videoTotalPages = Math.max(1, Number(res.total_pages) || 1);
        const c = res.counts || {};
        this.videoStatusCounts = {
          draft: Number(c.draft) || 0,
          processing: Number(c.processing) || 0,
          passed: Number(c.passed) || 0,
          rejected: Number(c.rejected) || 0
        };
      } catch (e) {
        const msg = e instanceof Error ? e.message : "加载失败";
        this.videosFetchError = msg;
        this.videos = [];
        this.videoTotal = 0;
        this.videoTotalPages = 1;
        ElMessage.error(msg);
      } finally {
        this.videosLoading = false;
      }
    },
    formatNum(n) {
      if (n >= 10000) {
        const w = n / 10000;
        return (w >= 10 ? Math.floor(w) : w.toFixed(1)) + "万";
      }
      return String(n);
    }
  }
};
</script>

<style lang="scss" scoped>
$c-blue: #00a1d6;
$c-text: #18191c;
$c-line: #e3e5e7;
$c-meta: #99a2aa;

.mm-wrap {
  max-width: 1120px;
  margin: 0 auto;
}

.mm-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid $c-line;
  margin-bottom: 16px;
}

.mm-tabs {
  display: flex;
  align-items: center;
  gap: 28px;
  flex-shrink: 0;
}

.mm-tab {
  padding: 0 2px 10px;
  margin-bottom: -17px;
  border: none;
  background: none;
  font-size: 15px;
  color: $c-text;
  cursor: pointer;
  border-bottom: 3px solid transparent;
}
.mm-tab.on {
  color: $c-blue;
  font-weight: 600;
  border-bottom-color: $c-blue;
}
.mm-tab:hover:not(.on) {
  color: $c-blue;
}

.mm-search {
  position: relative;
  flex: 1;
  max-width: 320px;
  min-width: 160px;
}

.mm-search-input {
  width: 100%;
  height: 34px;
  padding: 0 36px 0 12px;
  border: 1px solid #ccd0d7;
  border-radius: 4px;
  font-size: 13px;
  color: $c-text;
  outline: none;
  box-sizing: border-box;
}
.mm-search-input::placeholder {
  color: #c0c4cc;
}
.mm-search-input:focus {
  border-color: rgba(0, 161, 214, 0.55);
}

.mm-search-ico {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: #9499a0;
  pointer-events: none;
  display: flex;
}

.mm-subbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 12px;
}

.mm-status {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 8px;
  font-size: 13px;
}

.mm-status-btn {
  border: none;
  background: none;
  padding: 4px 8px;
  color: #505050;
  cursor: pointer;
  border-radius: 4px;
}
.mm-status-btn:hover {
  color: $c-blue;
}
.mm-status-btn.on {
  color: $c-blue;
  font-weight: 600;
}
.mm-status-count em {
  font-style: normal;
  font-weight: 600;
  margin-left: 2px;
}
.mm-status-split {
  width: 1px;
  height: 14px;
  background: #e5e9ef;
  margin: 0 6px;
}

.mm-sort {
  position: relative;
}

.mm-sort-trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 10px;
  border: 1px solid #ccd0d7;
  border-radius: 4px;
  background: #fff;
  font-size: 13px;
  color: #505050;
  cursor: pointer;
}
.mm-sort-trigger:hover {
  border-color: #aeb6bf;
}

.mm-sort-chevron {
  width: 0;
  height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 5px solid #717171;
  transition: transform 0.15s;
}
.mm-sort-chevron.open {
  transform: rotate(180deg);
}

.mm-sort-menu {
  position: absolute;
  right: 0;
  top: calc(100% + 4px);
  min-width: 148px;
  padding: 4px 0;
  background: #fff;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
  z-index: 50;
}

.mm-sort-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 8px 12px;
  border: none;
  background: none;
  font-size: 13px;
  color: $c-text;
  cursor: pointer;
  text-align: left;
}
.mm-sort-item:hover {
  background: #f4f5f7;
}
.mm-sort-item.active {
  color: $c-blue;
}
.mm-sort-check {
  color: $c-blue;
  font-size: 12px;
}

.mm-video-error {
  margin: 0 0 12px;
  font-size: 13px;
  color: #e53935;
}

.mm-video-loading {
  padding: 48px 0;
  text-align: center;
  font-size: 14px;
  color: $c-meta;
}

.mm-fail-reason {
  margin: 0 0 4px;
  font-size: 12px;
  line-height: 1.4;
  color: #e53935;
}

.mm-status-hint {
  margin: 0 0 4px;
  font-size: 12px;
  line-height: 1.5;
  color: #00a1d6;
}

.mm-cover-badge {
  position: absolute;
  left: 6px;
  top: 6px;
  z-index: 2;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 11px;
  line-height: 16px;
  color: #fff;
  background: rgba(255, 102, 0, 0.92);
  pointer-events: none;
}

.mm-cover-badge--processing {
  background: rgba(0, 0, 0, 0.65);
}

.mm-empty-video {
  min-height: 280px;
  padding: 48px 16px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  text-align: center;
}

.mm-list {
  border-top: 1px solid $c-line;
}

.mm-row {
  display: flex;
  align-items: stretch;
  gap: 16px;
  padding: 20px 0;
  border-bottom: 1px solid $c-line;
}

.mm-cover-wrap {
  position: relative;
  flex-shrink: 0;
  width: 160px;
  height: 100px;
  border-radius: 4px;
  overflow: hidden;
  background: #f1f2f3;
}

.mm-cover-link {
  text-decoration: none;
  color: inherit;
  box-sizing: border-box;
  cursor: pointer;
}

.mm-cover-link:hover .mm-cover {
  opacity: 0.94;
}

.mm-cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.mm-duration {
  position: absolute;
  right: 6px;
  bottom: 4px;
  padding: 0 4px;
  font-size: 12px;
  line-height: 18px;
  color: #fff;
  background: rgba(0, 0, 0, 0.65);
  border-radius: 2px;
}

.mm-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
}

.mm-title-link {
  text-decoration: none;
  color: inherit;
  display: block;
  width: fit-content;
  max-width: 100%;
}

.mm-title-link:hover .mm-title {
  color: $c-blue;
}

.mm-title {
  margin: 0 0 8px;
  font-size: 15px;
  font-weight: 600;
  color: $c-text;
  line-height: 1.45;
}

.mm-time {
  margin: 0 0 10px;
  font-size: 12px;
  color: $c-meta;
}

.meta-footer {
  font-size: 12px;
  color: $c-meta;
  user-select: none;
}

.view-stat {
  display: inline-block;
  margin-right: 20px;
  vertical-align: middle;
  pointer-events: none;
}

.stat-ico {
  width: 16px;
  height: 16px;
  vertical-align: middle;
  margin-right: 4px;
  object-fit: contain;
}

.icon-text {
  vertical-align: middle;
}

.mm-actions {
  flex-shrink: 0;
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  gap: 12px;
  padding-top: 2px;
  position: relative;
}

.mm-btn-edit {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 12px;
  border: 1px solid #ccd0d7;
  border-radius: 4px;
  background: #fff;
  font-size: 13px;
  color: #505050;
  cursor: pointer;
  text-decoration: none;
  box-sizing: border-box;
}
.mm-btn-edit:hover {
  border-color: $c-blue;
  color: $c-blue;
}
.mm-btn-edit-img {
  flex-shrink: 0;
  width: 14px;
  height: 14px;
  object-fit: contain;
  display: block;
}

.mm-btn-more {
  width: 30px;
  height: 30px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  color: #9499a0;
}
.mm-btn-more:hover {
  background: #f4f5f7;
  color: #505050;
}

.mm-more-wrap {
  position: relative;
  /* 扩大热区，避免从三点移到面板途中离开 wrap */
  padding-bottom: 10px;
  margin-bottom: -10px;
}

.mm-more-pop {
  position: absolute;
  right: 0;
  top: 100%;
  min-width: 320px;
  padding: 14px 14px 12px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.12), 0 0 1px rgba(0, 0, 0, 0.08);
  z-index: 120;
}

/* 向上的透明命中条：盖住面板与按钮之间的斜线路径 */
.mm-more-pop::before {
  content: "";
  position: absolute;
  left: -16px;
  right: -12px;
  height: 18px;
  bottom: 100%;
  pointer-events: auto;
}

.mm-more-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(72px, 1fr));
  gap: 14px 10px;
}

.mm-more-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 4px 2px;
  border: none;
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  color: #505050;
  font-size: 12px;
  line-height: 1.25;
  text-align: center;
}
.mm-more-cell:hover {
  color: $c-blue;
  background: #f6f7f8;
}

.mm-more-ico {
  display: flex;
  align-items: center;
  justify-content: center;
}

.mm-more-img {
  width: 22px;
  height: 22px;
  object-fit: contain;
  display: block;
}

.mm-more-lbl {
  display: block;
  max-width: 76px;
}

.mm-more-cell--danger:hover {
  color: #e53935;
  background: #fff5f5;
}
.mm-more-cell.is-disabled,
.mm-more-cell:disabled {
  opacity: 0.45;
  cursor: not-allowed;
  pointer-events: none;
}

.mm-del-overlay {
  position: fixed;
  inset: 0;
  z-index: 10080;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
  box-sizing: border-box;
}
.mm-del-overlay__backdrop {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
}
.mm-del-modal {
  position: relative;
  z-index: 1;
  width: min(400px, calc(100vw - 48px));
  box-sizing: border-box;
  padding: 28px 24px 22px;
  border-radius: 12px;
  background: #fff;
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.12),
    0 2px 8px rgba(0, 0, 0, 0.06);
  text-align: center;
}
.mm-del-modal__close {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 32px;
  height: 32px;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 8px;
  background: transparent;
  font-size: 22px;
  line-height: 1;
  color: #9499a0;
  cursor: pointer;
  &:hover:not(:disabled) {
    color: #61666d;
    background: #f1f2f3;
  }
  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}
.mm-del-dialog__title {
  margin: 0 0 10px;
  padding: 0;
  border: none;
  font-size: 17px;
  font-weight: 600;
  color: $c-text;
  line-height: 1.45;
  text-align: center;
}
.mm-del-dialog__warn {
  display: block;
  width: 100%;
  margin: 0;
  padding: 0;
  box-sizing: border-box;
  font-size: 13px;
  color: #9499a0;
  line-height: 1.55;
  text-align: center;
}

.mm-dot-v {
  display: block;
  width: 3px;
  height: 3px;
  margin: 0 auto;
  background: currentColor;
  border-radius: 50%;
  box-shadow: 0 -6px 0 currentColor, 0 6px 0 currentColor;
}

.mm-page {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 24px;
  font-size: 13px;
  color: $c-meta;
}

.mm-page-info {
  margin-right: auto;
}

.mm-page-btn {
  min-width: 64px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid #ccd0d7;
  border-radius: 4px;
  background: #fff;
  font-size: 12px;
  color: #505050;
  cursor: pointer;
}
.mm-page-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.mm-page-current {
  min-width: 28px;
  height: 28px;
  line-height: 28px;
  text-align: center;
  background: $c-blue;
  color: #fff;
  border-radius: 4px;
  font-size: 13px;
}
.mm-page-num {
  min-width: 28px;
  height: 28px;
  padding: 0 6px;
  border: 1px solid #ccd0d7;
  border-radius: 4px;
  background: #fff;
  font-size: 12px;
  color: #505050;
  cursor: pointer;
  &:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
  &.on {
    background: $c-blue;
    border-color: $c-blue;
    color: #fff;
  }
}

.mm-article-subtabs {
  display: flex;
  align-items: center;
  gap: 28px;
  padding-bottom: 12px;
  margin-bottom: 16px;
  border-bottom: 1px solid $c-line;
}

.mm-article-subtab {
  padding: 0 2px 8px;
  margin-bottom: -13px;
  border: none;
  background: none;
  font-size: 14px;
  color: $c-text;
  cursor: pointer;
  border-bottom: 2px solid transparent;
}
.mm-article-subtab.on {
  color: $c-text;
  font-weight: 600;
  border-bottom-color: $c-blue;
}
.mm-article-subtab:hover:not(.on) {
  color: $c-blue;
}

.mm-article-toolbar {
  align-items: flex-start;
}

.mm-article-toolbar-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mm-article-pills {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.mm-pill {
  padding: 6px 14px;
  border: 1px solid #e5e9ef;
  border-radius: 16px;
  background: #fff;
  font-size: 13px;
  color: #505050;
  cursor: pointer;
  line-height: 1.2;
}
.mm-pill:hover {
  border-color: rgba(0, 161, 214, 0.45);
  color: $c-blue;
}
.mm-pill.on {
  background: $c-blue;
  border-color: $c-blue;
  color: #fff;
  font-weight: 600;
}

.mm-article-status {
  flex-wrap: wrap;
  gap: 4px 8px;
}

.mm-article-kind-badge {
  position: absolute;
  left: 6px;
  bottom: 6px;
  padding: 2px 6px;
  font-size: 11px;
  line-height: 16px;
  color: #fff;
  background: rgba(0, 0, 0, 0.55);
  border-radius: 2px;
}

.mm-article-tags {
  margin: 0 0 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.mm-article-tag {
  display: inline-block;
  padding: 2px 8px;
  font-size: 12px;
  color: $c-meta;
  background: #f4f5f7;
  border-radius: 4px;
}

.stat-ico-svg {
  vertical-align: middle;
  margin-right: 4px;
  color: #9499a0;
}

.mm-more-pop--article {
  min-width: 108px;
  padding: 6px 0;
}

.mm-more-item-delete {
  display: block;
  width: 100%;
  margin: 0;
  padding: 8px 16px;
  border: none;
  background: transparent;
  font-size: 14px;
  line-height: 1.4;
  color: #18191c;
  text-align: center;
  cursor: pointer;
  box-sizing: border-box;
  &:hover {
    color: #e53935;
    background: #fff5f5;
  }
}

.mm-del-dialog__title--compact {
  margin: 0 0 12px;
  font-size: 17px;
  font-weight: 600;
  text-align: center;
}
.mm-del-dialog__warn--sub {
  margin: 0 0 16px;
  font-size: 12px;
  line-height: 1.5;
  text-align: center;
}
.mm-del-dialog__article-msg {
  margin: 0 0 28px;
  padding: 0 8px;
  font-size: 16px;
  font-weight: 400;
  color: #61666d;
  line-height: 1.5;
}

.mm-del-dialog__article-foot {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.mm-del-dialog__btn-cancel,
.mm-del-dialog__btn-confirm {
  flex: 1;
  max-width: 140px;
  height: 40px;
  margin: 0;
  padding: 0 20px;
  border-radius: 20px;
  font-size: 15px;
  line-height: 1;
  cursor: pointer;
  box-sizing: border-box;
  transition:
    opacity 0.15s ease,
    background 0.15s ease,
    border-color 0.15s ease;
  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}

.mm-del-dialog__btn-cancel {
  border: 1px solid #e3e5e7;
  background: #fff;
  color: #18191c;
  &:hover:not(:disabled) {
    border-color: #c9ccd0;
    background: #f6f7f8;
  }
}

.mm-del-dialog__btn-confirm {
  border: none;
  background: #f9788a;
  color: #fff;
  &:hover:not(:disabled) {
    background: #f06278;
  }
}

.mm-collection-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}

.mm-collection-label {
  font-size: 14px;
  font-weight: 600;
  color: $c-text;
}

.mm-btn-primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 32px;
  padding: 0 16px;
  border: none;
  border-radius: 4px;
  background: $c-blue;
  font-size: 13px;
  color: #fff;
  cursor: pointer;
}
.mm-btn-primary:hover {
  filter: brightness(1.05);
}

.mm-btn-primary--lg {
  height: 40px;
  padding: 0 28px;
  font-size: 14px;
}

.mm-collection-empty {
  min-height: 320px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 40px 16px;
  box-sizing: border-box;
}

.mm-collection-empty-title {
  margin: 0 0 8px;
  font-size: 16px;
  font-weight: 600;
  color: $c-text;
}

.mm-collection-empty-sub {
  margin: 0 0 24px;
  font-size: 13px;
  color: $c-meta;
  line-height: 1.6;
  max-width: 360px;
}

.mm-draft-hint {
  margin-bottom: 8px;
}

.mm-draft-title {
  margin: 0 0 6px;
  font-size: 15px;
  font-weight: 600;
  color: $c-text;
}

.mm-draft-sub {
  margin: 0;
  font-size: 13px;
  color: $c-meta;
}

.mm-empty-draft {
  min-height: 280px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 16px;
}

.mm-empty-draft-img {
  width: min(300px, 82vw);
}

.mm-row-collection {
  align-items: center;
}

.mm-collection-body {
  flex: 1;
  min-width: 0;
}

.mm-empty-article {
  min-height: 360px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 24px 16px;
  box-sizing: border-box;
}

.mm-empty-article-img {
  width: min(380px, 88vw);
  height: auto;
  display: block;
  margin-left: auto;
  margin-right: auto;
  object-fit: contain;
}

.mm-empty-article-txt {
  margin: 0;
  color: $c-meta;
  font-size: 14px;
}
</style>
