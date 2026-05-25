<template>
  <div class="mb-cmt" :class="{ 'mb-cmt--embedded': embedded }">
    <div v-if="!embedded" class="mb-cmt__head">
      <h3 class="mb-cmt__title">
        <span class="mb-cmt__count">{{ items.length }}</span> 评论
      </h3>
      <el-button size="small" :loading="loading" @click="load">刷新</el-button>
    </div>
    <div v-if="!hideComposer" class="mb-cmt__composer">
      <el-input
        v-model="draft"
        type="textarea"
        :rows="3"
        maxlength="1000"
        show-word-limit
        placeholder="登录后可发表评论（最多三级回复）"
      />
      <div class="mb-cmt__composer-row">
        <el-button type="primary" :loading="posting" :disabled="!canPost" @click="submit">
          发送
        </el-button>
      </div>
    </div>
    <div v-if="loadError" class="mb-cmt__err">{{ loadError }}</div>
    <p v-else-if="!loading && !items.length" class="mb-cmt__empty">还没有评论~</p>
    <p v-else-if="loading && !items.length" class="mb-cmt__loading">加载中…</p>
    <ul
      v-else
      class="vd-cmt-list"
      :class="{ 'vd-cmt-list--soft-refresh': refreshing }"
    >
      <li
        v-for="(row, ti) in vdThreads"
        :key="row.root.id"
        class="vd-cmt-thread"
        :data-mb-thread="ti"
      >
        <div class="vd-cmt-root-stack">
        <div
          class="vd-cmt-item vd-cmt-item--root"
          :id="commentDomId(row.root.id)"
          :class="mbHighlightRowClass(row.root.id)"
        >
                <MbUserHoverCard
                  v-if="isMinibili && cmtHoverUid(row.root)"
                  :user-id="cmtHoverUid(row.root)"
                >
                  <router-link
                    v-if="userSpaceRoute(row.root.user_id)"
                    class="vd-cmt-face-link"
                    :to="userSpaceRoute(row.root.user_id)"
                  >
                    <img
                      class="vd-cmt-face"
                      :src="commentFaceSrc(row.root)"
                      :alt="displayName(row.root)"
                      width="48"
                      height="48"
                    />
                  </router-link>
                  <img
                    v-else
                    class="vd-cmt-face"
                    :src="commentFaceSrc(row.root)"
                    :alt="displayName(row.root)"
                    width="48"
                    height="48"
                  />
                </MbUserHoverCard>
                <template v-else>
                  <router-link
                    v-if="userSpaceRoute(row.root.user_id)"
                    class="vd-cmt-face-link"
                    :to="userSpaceRoute(row.root.user_id)"
                  >
                    <img
                      class="vd-cmt-face"
                      :src="commentFaceSrc(row.root)"
                      :alt="displayName(row.root)"
                      width="48"
                      height="48"
                    />
                  </router-link>
                  <img
                    v-else
                    class="vd-cmt-face"
                    :src="commentFaceSrc(row.root)"
                    :alt="displayName(row.root)"
                    width="48"
                    height="48"
                  />
                </template>
          <div class="vd-cmt-body">
            <div class="vd-cmt-user-row">
              <MbUserHoverCard
                v-if="isMinibili && cmtHoverUid(row.root)"
                :user-id="cmtHoverUid(row.root)"
              >
                <router-link
                  v-if="userSpaceRoute(row.root.user_id)"
                  class="vd-cmt-name"
                  :class="'tone-' + nameTone(row.root.user_id)"
                  :to="userSpaceRoute(row.root.user_id)"
                  >{{ displayName(row.root) }}</router-link
                >
                <a
                  v-else
                  href="javascript:;"
                  class="vd-cmt-name"
                  :class="'tone-' + nameTone(row.root.user_id)"
                  >{{ displayName(row.root) }}</a
                >
              </MbUserHoverCard>
              <template v-else>
                <router-link
                  v-if="userSpaceRoute(row.root.user_id)"
                  class="vd-cmt-name"
                  :class="'tone-' + nameTone(row.root.user_id)"
                  :to="userSpaceRoute(row.root.user_id)"
                  >{{ displayName(row.root) }}</router-link
                >
                <a
                  v-else
                  href="javascript:;"
                  class="vd-cmt-name"
                  :class="'tone-' + nameTone(row.root.user_id)"
                  >{{ displayName(row.root) }}</a
                >
              </template>
              <img
                class="level-badge"
                :src="levelIconUrl(displayLevel(row.root))"
                width="30"
                height="30"
                alt=""
                :title="'LV' + displayLevel(row.root)"
              />
              <img
                v-if="row.root.is_by_uploader"
                class="vd-cmt-up-badge"
                :src="upImg"
                width="36"
                height="14"
                alt="UP"
              />
            </div>
            <p class="vd-cmt-text">
              <img
                v-if="row.root.pinned"
                class="vd-cmt-pin-ico"
                :src="commentTopImg"
                width="26"
                height="13"
                alt="置顶"
              />{{ row.root.content }}
            </p>
            <div class="vd-cmt-meta-row">
              <div class="vd-cmt-meta-main">
                <span>{{ row.root.created_at }}</span>
                <span class="vd-cmt-ip">{{ ipLabel(row.root) }}</span>
                <VdCommentThumbBtn
                  variant="like"
                  :active="!!row.root.liked_by_me"
                  :count="row.root.like_count"
                  @click="toggleLike(row.root)"
                />
                <VdCommentThumbBtn
                  variant="dislike"
                  :active="!!row.root.disliked_by_me"
                  :show-count="false"
                  @click="toggleDislike(row.root)"
                />
                <button
                  type="button"
                  class="vd-cmt-act"
                  @click.stop="openReplyComposer(row.root, ti)"
                >
                  回复
                </button>
              </div>
              <div
                class="vd-cmt-menu-wrap vd-cmt-menu-wrap--root"
                :class="{ 'is-open': openCommentMenuKey === cmtMenuKey(ti) }"
                @click.stop
              >
                <button
                  type="button"
                  class="vd-cmt-menu-trigger"
                  aria-haspopup="true"
                  :aria-expanded="openCommentMenuKey === cmtMenuKey(ti)"
                  aria-label="更多操作"
                  @click="toggleCommentMenu(cmtMenuKey(ti), $event)"
                >
                  <span class="vd-cmt-menu-dots" aria-hidden="true">
                    <span /><span /><span />
                  </span>
                </button>
                <div
                  v-if="openCommentMenuKey === cmtMenuKey(ti)"
                  class="vd-cmt-menu-dropdown"
                  role="menu"
                >
                  <template v-if="isContentOwner && isOwnComment(row.root)">
                    <button
                      type="button"
                      class="vd-cmt-menu-item"
                      role="menuitem"
                      @click="onMenuPin(row.root)"
                    >
                      {{ row.root.pinned ? "取消置顶" : "置顶" }}
                    </button>
                    <button
                      type="button"
                      class="vd-cmt-menu-item"
                      role="menuitem"
                      @click="onMenuRemove(row.root)"
                    >
                      删除
                    </button>
                  </template>
                  <template v-else-if="isContentOwner && !isOwnComment(row.root)">
                    <button
                      type="button"
                      class="vd-cmt-menu-item"
                      role="menuitem"
                      @click="onMenuPin(row.root)"
                    >
                      {{ row.root.pinned ? "取消置顶" : "设为置顶" }}
                    </button>
                    <button
                      type="button"
                      class="vd-cmt-menu-item"
                      role="menuitem"
                      @click="onMenuStub('加入黑名单')"
                    >
                      加入黑名单
                    </button>
                    <button
                      type="button"
                      class="vd-cmt-menu-item"
                      role="menuitem"
                      @click="onMenuStub('举报')"
                    >
                      举报
                    </button>
                    <button
                      type="button"
                      class="vd-cmt-menu-item"
                      role="menuitem"
                      @click="onMenuRemove(row.root)"
                    >
                      删除
                    </button>
                  </template>
                  <template v-else-if="!isContentOwner && isOwnComment(row.root)">
                    <button
                      type="button"
                      class="vd-cmt-menu-item"
                      role="menuitem"
                      @click="onMenuRemove(row.root)"
                    >
                      删除
                    </button>
                  </template>
                  <template v-else>
                    <button
                      type="button"
                      class="vd-cmt-menu-item"
                      role="menuitem"
                      @click="onMenuStub('加入黑名单')"
                    >
                      加入黑名单
                    </button>
                    <button
                      type="button"
                      class="vd-cmt-menu-item"
                      role="menuitem"
                      @click="onMenuStub('举报')"
                    >
                      举报
                    </button>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </div>

            <ul v-if="row.replyRows.length" class="vd-cmt-replies">
              <li
                v-for="(rr, ri) in row.replyRows"
                :key="rr.c.id"
                :id="commentDomId(rr.c.id)"
                class="vd-cmt-item vd-cmt-item--reply"
                :class="mbHighlightRowClass(rr.c.id)"
              >
                <MbUserHoverCard
                  v-if="isMinibili && cmtHoverUid(rr.c)"
                  :user-id="cmtHoverUid(rr.c)"
                >
                  <router-link
                    v-if="userSpaceRoute(rr.c.user_id)"
                    class="vd-cmt-face-link"
                    :to="userSpaceRoute(rr.c.user_id)"
                  >
                    <img
                      class="vd-cmt-face vd-cmt-face--sm"
                      :src="commentFaceSrc(rr.c)"
                      :alt="displayName(rr.c)"
                      width="32"
                      height="32"
                    />
                  </router-link>
                  <img
                    v-else
                    class="vd-cmt-face vd-cmt-face--sm"
                    :src="commentFaceSrc(rr.c)"
                    :alt="displayName(rr.c)"
                    width="32"
                    height="32"
                  />
                </MbUserHoverCard>
                <template v-else>
                  <router-link
                    v-if="userSpaceRoute(rr.c.user_id)"
                    class="vd-cmt-face-link"
                    :to="userSpaceRoute(rr.c.user_id)"
                  >
                    <img
                      class="vd-cmt-face vd-cmt-face--sm"
                      :src="commentFaceSrc(rr.c)"
                      :alt="displayName(rr.c)"
                      width="32"
                      height="32"
                    />
                  </router-link>
                  <img
                    v-else
                    class="vd-cmt-face vd-cmt-face--sm"
                    :src="commentFaceSrc(rr.c)"
                    :alt="displayName(rr.c)"
                    width="32"
                    height="32"
                  />
                </template>
                <div class="vd-cmt-body">
                  <div class="vd-cmt-user-row">
                    <MbUserHoverCard
                      v-if="isMinibili && cmtHoverUid(rr.c)"
                      :user-id="cmtHoverUid(rr.c)"
                    >
                      <router-link
                        v-if="userSpaceRoute(rr.c.user_id)"
                        class="vd-cmt-name"
                        :class="'tone-' + nameTone(rr.c.user_id)"
                        :to="userSpaceRoute(rr.c.user_id)"
                        >{{ displayName(rr.c) }}</router-link
                      >
                      <a
                        v-else
                        href="javascript:;"
                        class="vd-cmt-name"
                        :class="'tone-' + nameTone(rr.c.user_id)"
                        >{{ displayName(rr.c) }}</a
                      >
                    </MbUserHoverCard>
                    <template v-else>
                      <router-link
                        v-if="userSpaceRoute(rr.c.user_id)"
                        class="vd-cmt-name"
                        :class="'tone-' + nameTone(rr.c.user_id)"
                        :to="userSpaceRoute(rr.c.user_id)"
                        >{{ displayName(rr.c) }}</router-link
                      >
                      <a
                        v-else
                        href="javascript:;"
                        class="vd-cmt-name"
                        :class="'tone-' + nameTone(rr.c.user_id)"
                        >{{ displayName(rr.c) }}</a
                      >
                    </template>
                    <img
                      class="level-badge"
                      :src="levelIconUrl(displayLevel(rr.c))"
                      width="30"
                      height="30"
                      alt=""
                      :title="'LV' + displayLevel(rr.c)"
                    />
                    <img
                      v-if="rr.c.is_by_uploader"
                      class="vd-cmt-up-badge"
                      :src="upImg"
                      width="36"
                      height="14"
                      alt="UP"
                    />
                  </div>
                  <p class="vd-cmt-text">
                    <span v-if="rr.replyTargetName" class="vd-cmt-reply-prefix">
                      <span class="vd-cmt-reply-prefix-muted">回复 </span>
                      <span class="vd-cmt-reply-prefix-atwrap">
                        <span class="vd-cmt-reply-prefix-at">@</span>
                        <span class="vd-cmt-reply-prefix-name">{{
                          rr.replyTargetName
                        }}</span>
                      </span>
                      <span class="vd-cmt-reply-prefix-muted"> :</span>
                    </span>{{ rr.c.content }}
                  </p>
                  <div class="vd-cmt-meta-row">
                    <div class="vd-cmt-meta-main">
                      <span>{{ rr.c.created_at }}</span>
                      <span class="vd-cmt-ip">{{ ipLabel(rr.c) }}</span>
                      <VdCommentThumbBtn
                        variant="like"
                        :active="!!rr.c.liked_by_me"
                        :count="rr.c.like_count"
                        @click="toggleLike(rr.c)"
                      />
                      <VdCommentThumbBtn
                        variant="dislike"
                        :active="!!rr.c.disliked_by_me"
                        :show-count="false"
                        @click="toggleDislike(rr.c)"
                      />
                      <button
                        type="button"
                        class="vd-cmt-act"
                        @click.stop="openReplyComposer(rr.c, ti)"
                      >
                        回复
                      </button>
                    </div>
                    <div
                      class="vd-cmt-menu-wrap vd-cmt-menu-wrap--reply"
                      :class="{
                        'is-open': openCommentMenuKey === cmtMenuKey(ti, ri)
                      }"
                      @click.stop
                    >
                      <button
                        type="button"
                        class="vd-cmt-menu-trigger"
                        aria-haspopup="true"
                        :aria-expanded="
                          openCommentMenuKey === cmtMenuKey(ti, ri)
                        "
                        aria-label="更多操作"
                        @click="toggleCommentMenu(cmtMenuKey(ti, ri), $event)"
                      >
                        <span class="vd-cmt-menu-dots" aria-hidden="true">
                          <span /><span /><span />
                        </span>
                      </button>
                      <div
                        v-if="openCommentMenuKey === cmtMenuKey(ti, ri)"
                        class="vd-cmt-menu-dropdown"
                        role="menu"
                      >
                        <template v-if="isContentOwner && isOwnComment(rr.c)">
                          <button
                            type="button"
                            class="vd-cmt-menu-item"
                            role="menuitem"
                            @click="onMenuRemove(rr.c)"
                          >
                            删除
                          </button>
                        </template>
                        <template v-else-if="isContentOwner && !isOwnComment(rr.c)">
                          <button
                            type="button"
                            class="vd-cmt-menu-item"
                            role="menuitem"
                            @click="onMenuStub('加入黑名单')"
                          >
                            加入黑名单
                          </button>
                          <button
                            type="button"
                            class="vd-cmt-menu-item"
                            role="menuitem"
                            @click="onMenuStub('举报')"
                          >
                            举报
                          </button>
                          <button
                            type="button"
                            class="vd-cmt-menu-item"
                            role="menuitem"
                            @click="onMenuRemove(rr.c)"
                          >
                            删除
                          </button>
                        </template>
                        <template v-else-if="!isContentOwner && isOwnComment(rr.c)">
                          <button
                            type="button"
                            class="vd-cmt-menu-item"
                            role="menuitem"
                            @click="onMenuRemove(rr.c)"
                          >
                            删除
                          </button>
                        </template>
                        <template v-else>
                          <button
                            type="button"
                            class="vd-cmt-menu-item"
                            role="menuitem"
                            @click="onMenuStub('加入黑名单')"
                          >
                            加入黑名单
                          </button>
                          <button
                            type="button"
                            class="vd-cmt-menu-item"
                            role="menuitem"
                            @click="onMenuStub('举报')"
                          >
                            举报
                          </button>
                        </template>
                      </div>
                    </div>
                  </div>
                </div>
              </li>
            </ul>
            <p
              v-if="row.showReplyFold"
              class="vd-cmt-fold vd-cmt-fold--mb-more"
            >
              <span class="vd-cmt-fold-meta"
                >共{{ row.replyTotal }}条回复,</span
              >
              <button
                type="button"
                class="vd-cmt-fold-link"
                @click.prevent.stop="toggleReplyThreadExpand(row.root.id)"
              >
                点击查看
              </button>
            </p>
            <p
              v-if="row.showReplyCollapse"
              class="vd-cmt-fold vd-cmt-fold--mb-more vd-cmt-fold--mb-collapse"
            >
              <button
                type="button"
                class="vd-cmt-fold-link"
                @click.prevent.stop="toggleReplyThreadExpand(row.root.id)"
              >
                收起
              </button>
            </p>
            <div
              v-if="replyThreadIndex === ti"
              class="mb-thread-reply"
              @click.stop
            >
              <MbReplyComposerInner
                v-if="replyParent"
                :input-placeholder="replyComposerPlaceholder"
                :avatar-src="mbSelfAvatarSrc"
                v-model:draft="replyDraftText"
                :posting="replyPosting"
                @submit="submitInlineReply"
              />
            </div>
        </div>
      </li>
    </ul>
  </div>
</template>

<script>
import { createNamespacedHelpers } from "vuex";
import { ElMessage, ElMessageBox } from "element-plus";
import MbReplyComposerInner from "./MbReplyComposerInner.vue";
import MbUserHoverCard from "@/components/minibili/MbUserHoverCard.vue";
import VdCommentThumbBtn from "@/components/comment/VdCommentThumbBtn.vue";
import akariCover from "@/assets/akari.jpg";
import commentTopImg from "@/assets/comment_top.png";
import upImg from "@/assets/UP.png";
import {
  mbDeleteComment,
  mbListComments,
  mbPinComment,
  mbPostComment,
  mbToggleLike,
  mbToggleDislike,
  mbListArticleComments,
  mbPostArticleComment,
  mbToggleArticleCommentLike,
  mbToggleArticleCommentDislike,
  mbDeleteArticleComment,
  mbPinArticleComment,
  mbListDynamicComments,
  mbPostDynamicComment,
  mbToggleDynamicCommentLike,
  mbToggleDynamicCommentDislike,
  mbDeleteDynamicComment
} from "@/api/minibili";
import { MB_COMMENT_PENDING_TOAST } from "@/constants/minibiliComments";
import { getAccessToken, getUserId } from "@/utils/authTokens";
import { extractApiErrorMessage } from "@/utils/apiErrorMessage";
import { minibiliUserSpaceRoute } from "@/utils/minibiliRoutes";
import { commentUserLevel, levelIconUrl } from "@/utils/userLevel";

const { mapState } = createNamespacedHelpers("login");

export default {
  name: "MinibiliCommentsLive",
  components: {
    MbReplyComposerInner,
    MbUserHoverCard,
    VdCommentThumbBtn
  },
  props: {
    videoId: { type: Number, default: 0 },
    articleId: { type: Number, default: 0 },
    dynamicId: { type: Number, default: 0 },
    /** 稿件作者 ID，与当前登录用户比较以展示 UP 菜单与 UP 标 */
    videoAuthorId: { type: Number, default: null },
    articleAuthorId: { type: Number, default: null },
    dynamicAuthorId: { type: Number, default: null },
    embedded: { type: Boolean, default: false },
    hideComposer: { type: Boolean, default: false },
    /** 根评论排序：hot | time */
    commentSort: { type: String, default: "hot" },
    /** 路由 ?mb_cid=：定位并高亮该条评论 */
    highlightCommentId: { type: [Number, String], default: null },
    /** 父级已知精选状态时可传入，避免首屏闪烁 */
    initialCommentsCurated: { type: Boolean, default: null },
    /** 父级已知关闭评论时可传入，避免请求评论列表报 403 */
    initialCommentsClosed: { type: Boolean, default: null }
  },
  emits: ["counts"],
  data() {
    return {
      items: [],
      draft: "",
      loading: false,
      posting: false,
      loadError: "",
      openCommentMenuKey: "",
      defaultAvatar: akariCover,
      commentTopImg,
      upImg,
      /** 根评论 id → 是否展开全部楼中楼（默认每楼最多展示 3 条） */
      expandedReplyThreads: {},
      /** 根评论展示顺序（点赞/点踩后保持，避免热评重排导致评论跳走） */
      rootDisplayOrder: {},
      refreshing: false,
      /** 正在回复的楼所在列表下标（回复框固定在该一级评论底部） */
      replyThreadIndex: null,
      replyParent: null,
      replyDraftText: "",
      replyPosting: false,
      /** 路由 mb_cid 对应的临时高亮 id，超时后清除蓝色框 */
      activeHighlightCommentId: null,
      _highlightClearTimer: null,
      /** 高亮退场：先加 leaving 类做背景过渡，再移除 active */
      highlightLeaving: false,
      _highlightFadeEndTimer: null,
      commentsCuratedLocal: false,
      commentsClosedLocal: false
    };
  },
  computed: {
    ...mapState({
      proInfo: state => state.proInfo,
      minibiliMe: state => state.minibiliMe
    }),
    replyComposerPlaceholder() {
      if (!this.replyParent) return "回复 @用户 :";
      return `回复 @${this.displayName(this.replyParent)} :`;
    },
    isDynamic() {
      return Number(this.dynamicId) > 0;
    },
    isArticle() {
      return !this.isDynamic && Number(this.articleId) > 0;
    },
    contentAuthorId() {
      if (this.isDynamic) return this.dynamicAuthorId;
      if (this.isArticle) return this.articleAuthorId;
      return this.videoAuthorId;
    },
    isContentOwner() {
      const me = getUserId();
      const aid = this.contentAuthorId;
      if (me == null || aid == null) return false;
      return Number(aid) === Number(me);
    },
    /** 与 isContentOwner 相同，保留旧名供模板渐进替换 */
    isVideoOwner() {
      return this.isContentOwner;
    },
    /** 回复框：当前登录用户头像（与 Vuex 同步） */
    commentsCurated() {
      if (
        this.initialCommentsCurated !== null &&
        this.initialCommentsCurated !== undefined
      ) {
        return !!this.initialCommentsCurated;
      }
      return this.commentsCuratedLocal;
    },
    commentsClosed() {
      if (
        this.initialCommentsClosed !== null &&
        this.initialCommentsClosed !== undefined
      ) {
        return !!this.initialCommentsClosed;
      }
      return this.commentsClosedLocal;
    },
    mbSelfAvatarSrc() {
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
      return this.defaultAvatar;
    },
    canPost() {
      void this.$store.state.login.signIn;
      return !!getAccessToken() && this.draft.trim().length > 0;
    },
    isMinibili() {
      return (
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1"
      );
    },
    vdThreads() {
      void this.commentSort;
      const flat = this.items || [];
      const byId = {};
      flat.forEach(c => {
        byId[c.id] = { ...c, _replies: [] };
      });
      const roots = [];
      flat.forEach(c => {
        const pid = Number(c.parent_id) || 0;
        const node = byId[c.id];
        if (pid && byId[pid]) {
          byId[pid]._replies.push(node);
        } else {
          roots.push(node);
        }
      });
      const sortMode = this.commentSort === "time" ? "time" : "hot";
      const order = this.rootDisplayOrder || {};
      const hasOrder = Object.keys(order).length > 0;
      roots.sort((a, b) => {
        const ap = a.pinned ? 1 : 0;
        const bp = b.pinned ? 1 : 0;
        if (bp !== ap) return bp - ap;
        if (hasOrder) {
          const oa = order[String(a.id)];
          const ob = order[String(b.id)];
          if (oa != null && ob != null) return oa - ob;
          if (oa != null) return -1;
          if (ob != null) return 1;
        }
        return this.compareRootComments(a, b, sortMode);
      });
      const replyTargetName = (rootId, ch) => {
        const pid = Number(ch.parent_id) || 0;
        const rid = Number(rootId) || 0;
        if (!pid || pid === rid) return "";
        const p = flat.find(x => Number(x.id) === pid);
        if (!p) return "";
        return p.username || "用户" + p.user_id;
      };
      const collectReplyRows = (rootId, node, depth, out) => {
        const reps = (node._replies || []).slice().sort((a, b) => a.id - b.id);
        for (const ch of reps) {
          out.push({
            c: ch,
            depth,
            replyTargetName: replyTargetName(rootId, ch)
          });
          collectReplyRows(rootId, ch, depth + 1, out);
        }
      };
      const exp = this.expandedReplyThreads || {};
      return roots.map(r => {
        const replyRowsAll = [];
        collectReplyRows(r.id, r, 1, replyRowsAll);
        const total = replyRowsAll.length;
        const rid = String(r.id);
        const expanded = !!exp[rid];
        const replyRows =
          expanded || total <= 3 ? replyRowsAll : replyRowsAll.slice(0, 3);
        return {
          root: r,
          replyRows,
          replyTotal: total,
          showReplyFold: total > 3 && !expanded,
          showReplyCollapse: total > 3 && expanded
        };
      });
    }
  },
  watch: {
    videoId() {
      this.onTargetChange();
    },
    articleId() {
      this.onTargetChange();
    },
    dynamicId() {
      this.onTargetChange();
    },
    "$store.state.login.signIn"(v) {
      if (String(v) === "1" && getAccessToken()) {
        this.load({ soft: true, preserveExpand: true });
      }
    },
    commentSort() {
      this.recomputeRootDisplayOrder();
    },
    highlightCommentId: {
      handler() {
        this.clearHighlightDismissTimer();
        if (this.highlightCommentId == null || this.highlightCommentId === "") {
          this.activeHighlightCommentId = null;
          return;
        }
        this.syncRouteHighlightToActive();
        if (this.activeHighlightCommentId == null) return;
        this.$nextTick(() => this.scrollToHighlightComment());
      },
      immediate: true
    },
    items() {
      if (this.highlightCommentId == null || this.highlightCommentId === "") {
        return;
      }
      this.syncRouteHighlightToActive();
      if (this.activeHighlightCommentId == null) return;
      this.$nextTick(() => this.scrollToHighlightComment());
    }
  },
  mounted() {
    if (!this.commentsClosed) {
      this.load();
    }
    document.addEventListener("click", this.onDocClick);
  },
  beforeUnmount() {
    document.removeEventListener("click", this.onDocClick);
    this.clearHighlightDismissTimer();
  },
  methods: {
    onTargetChange() {
      this.clearHighlightDismissTimer();
      this.activeHighlightCommentId = null;
      this.closeReplyComposer();
      if (this.commentsClosed) {
        this.items = [];
        this.loadError = "";
        return;
      }
      this.load({ soft: false, preserveExpand: false });
      this.$nextTick(() => {
        this.syncRouteHighlightToActive();
        if (this.activeHighlightCommentId != null) {
          this.$nextTick(() => this.scrollToHighlightComment());
        }
      });
    },
    userSpaceRoute(userId) {
      return minibiliUserSpaceRoute(userId);
    },
    ipLabel(c) {
      const loc =
        c && typeof c.ip_location === "string" ? c.ip_location.trim() : "";
      return loc ? `IP属地：${loc}` : "IP属地：—";
    },
    /** 把路由里的 mb_cid（prop）同步到用于渲染蓝框的 activeHighlightCommentId */
    syncRouteHighlightToActive() {
      const v = this.highlightCommentId;
      if (v == null || v === "") {
        this.activeHighlightCommentId = null;
        this.highlightLeaving = false;
        return;
      }
      const idn = parseInt(String(v), 10);
      if (!Number.isFinite(idn) || idn <= 0) {
        this.activeHighlightCommentId = null;
        this.highlightLeaving = false;
        return;
      }
      this.highlightLeaving = false;
      this.activeHighlightCommentId = idn;
    },
    mbHighlightRowClass(id) {
      if (!this.isHighlightTarget(id)) return {};
      const o = { "vd-cmt-item--mb-highlight": true };
      if (this.highlightLeaving) {
        o["vd-cmt-item--mb-highlight-leaving"] = true;
      }
      return o;
    },
    commentDomId(id) {
      const n = Number(id);
      return Number.isFinite(n) && n > 0 ? `mb-cmt-${n}` : undefined;
    },
    isHighlightTarget(id) {
      const t = this.activeHighlightCommentId;
      if (t == null || t === "") return false;
      return Number(t) === Number(id);
    },
    clearHighlightDismissTimer() {
      if (this._highlightClearTimer != null) {
        clearTimeout(this._highlightClearTimer);
        this._highlightClearTimer = null;
      }
      if (this._highlightFadeEndTimer != null) {
        clearTimeout(this._highlightFadeEndTimer);
        this._highlightFadeEndTimer = null;
      }
      this.highlightLeaving = false;
    },
    restartHighlightDismissTimer() {
      this.clearHighlightDismissTimer();
      this._highlightClearTimer = setTimeout(() => {
        this._highlightClearTimer = null;
        if (this.activeHighlightCommentId == null) return;
        this.highlightLeaving = true;
        this._highlightFadeEndTimer = setTimeout(() => {
          this.activeHighlightCommentId = null;
          this.highlightLeaving = false;
          this._highlightFadeEndTimer = null;
        }, 420);
      }, 5000);
    },
    scrollToHighlightComment() {
      const raw = this.highlightCommentId;
      if (raw == null || raw === "") return;
      const idn = parseInt(String(raw), 10);
      if (!Number.isFinite(idn) || idn <= 0) return;
      const flat = this.items || [];
      if (!flat.length) return;
      const c = flat.find(x => Number(x.id) === idn);
      if (!c) {
        if (flat.length > 0) {
          this.activeHighlightCommentId = null;
          this.highlightLeaving = false;
          this.clearHighlightDismissTimer();
        }
        return;
      }
      let node = c;
      let p = Number(node.parent_id) || 0;
      while (p > 0) {
        const parent = flat.find(x => Number(x.id) === p);
        if (!parent) break;
        node = parent;
        p = Number(parent.parent_id) || 0;
      }
      const rootId = String(node.id);
      if (Number(c.parent_id) !== 0) {
        this.expandedReplyThreads = {
          ...this.expandedReplyThreads,
          [rootId]: true
        };
      }
      this.$nextTick(() => {
        const el = document.getElementById(`mb-cmt-${idn}`);
        if (el && typeof el.scrollIntoView === "function") {
          el.scrollIntoView({ block: "center", behavior: "auto" });
          this.restartHighlightDismissTimer();
          return;
        }
        this.$nextTick(() => {
          const el2 = document.getElementById(`mb-cmt-${idn}`);
          if (el2 && typeof el2.scrollIntoView === "function") {
            el2.scrollIntoView({ block: "center", behavior: "auto" });
            this.restartHighlightDismissTimer();
            return;
          }
          window.setTimeout(() => {
            const el3 = document.getElementById(`mb-cmt-${idn}`);
            if (el3 && typeof el3.scrollIntoView === "function") {
              el3.scrollIntoView({ block: "center", behavior: "auto" });
              this.restartHighlightDismissTimer();
            }
          }, 120);
        });
      });
    },
    onDocClick(ev) {
      this.openCommentMenuKey = "";
      const t = ev && ev.target;
      if (!t || !(t instanceof Element)) {
        this.closeReplyComposer();
        return;
      }
      if (
        t.closest(".mb-r-compose") ||
        t.closest("button.vd-cmt-act") ||
        t.closest(".vd-cmt-menu-wrap") ||
        t.closest(".vd-cmt-menu-dropdown") ||
        t.closest(".vd-cmt-fold-link")
      ) {
        return;
      }
      this.closeReplyComposer();
    },
    openLoginModal() {
      this.$store.commit("login/SET_LOGIN_TAB", 0);
      this.$store.commit("login/OPEN_LOGIN_MODAL");
    },
    toggleReplyThreadExpand(rootId) {
      const k = String(rootId);
      const cur = !!this.expandedReplyThreads[k];
      this.expandedReplyThreads = {
        ...this.expandedReplyThreads,
        [k]: !cur
      };
    },
    cmtMenuKey(ti, ri = null) {
      return ri == null ? `mb-t-${ti}` : `mb-t-${ti}-r-${ri}`;
    },
    toggleCommentMenu(key, e) {
      if (e) e.stopPropagation();
      this.openCommentMenuKey =
        this.openCommentMenuKey === key ? "" : key;
    },
    closeCommentMenu() {
      this.openCommentMenuKey = "";
    },
    displayName(c) {
      return c.username || "用户" + c.user_id;
    },
    cmtHoverUid(c) {
      const n = Number(c && c.user_id) || 0;
      return n > 0 ? n : 0;
    },
    commentFaceSrc(c) {
      if (!c || typeof c !== "object") {
        return this.defaultAvatar;
      }
      const u = String(c.avatar_url || "").trim();
      return u || this.defaultAvatar;
    },
    nameTone(userId) {
      const tones = ["blue", "pink", "black"];
      const u = Number(userId) || 0;
      return tones[Math.abs(u) % 3];
    },
    displayLevel(c) {
      return commentUserLevel(c);
    },
    levelIconUrl,
    isOwnComment(c) {
      const me = getUserId();
      return me != null && Number(c.user_id) === Number(me);
    },
    isRootComment(c) {
      return (Number(c.parent_id) || 0) === 0;
    },
    commentHasReplies(cid) {
      const id = Number(cid);
      return (this.items || []).some(x => Number(x.parent_id) === id);
    },
    deleteConfirmLines(c) {
      const own = this.isOwnComment(c);
      const root = this.isRootComment(c);
      const has = this.commentHasReplies(c.id);
      if (!own) {
        return has
          ? "删除评论后，评论下所有回复都会被删除。\n是否继续？"
          : "删除评论后将不可恢复\n是否继续？";
      }
      if (root) {
        return has
          ? "删除评论后，评论下所有回复都会被删除。\n是否继续？"
          : "删除评论后将不可恢复\n是否继续？";
      }
      return "确定删除这条评论？";
    },
    async load(opts = {}) {
      if (this.commentsClosed) {
        this.items = [];
        this.loadError = "";
        this.loading = false;
        this.refreshing = false;
        return;
      }
      const hadItems = (this.items || []).length > 0;
      const soft =
        opts.soft !== undefined ? !!opts.soft : hadItems;
      this.loadError = "";
      if (soft) {
        this.refreshing = true;
      } else {
        this.loading = true;
      }
      try {
        let res;
        if (this.isDynamic) {
          res = await mbListDynamicComments(this.dynamicId);
          this.commentsClosedLocal = !!res.comments_closed;
          if (
            this.initialCommentsCurated === null ||
            this.initialCommentsCurated === undefined
          ) {
            this.commentsCuratedLocal = !!res.comments_curated;
          }
        } else if (this.isArticle) {
          res = await mbListArticleComments(this.articleId);
          this.commentsClosedLocal = !!res.comments_closed;
          if (
            this.initialCommentsCurated === null ||
            this.initialCommentsCurated === undefined
          ) {
            this.commentsCuratedLocal = !!res.comments_curated;
          }
        } else {
          res = await mbListComments(this.videoId);
          this.commentsClosedLocal = !!res.comments_closed;
          if (
            this.initialCommentsCurated === null ||
            this.initialCommentsCurated === undefined
          ) {
            this.commentsCuratedLocal = !!res.comments_curated;
          }
        }
        this.items = res.items || [];
        if (!soft && !opts.preserveExpand) {
          this.expandedReplyThreads = {};
        }
        this.recomputeRootDisplayOrder();
        this.$emit("counts", this.items.length);
      } catch (e) {
        const apiCode =
          e && typeof e.minibiliApiCode === "number" ? e.minibiliApiCode : 0;
        if (apiCode === 40303) {
          this.commentsClosedLocal = true;
          this.items = [];
          this.loadError = "";
          this.$emit("counts", 0);
          return;
        }
        this.loadError = (e && e.message) || "加载评论失败";
        ElMessage.error(this.loadError);
      } finally {
        this.loading = false;
        this.refreshing = false;
      }
    },
    async submit() {
      const content = this.draft.trim();
      if (!content) return;
      const ok = await this.postCommentExtern(content, 0);
      if (ok) this.draft = "";
    },
    async postCommentExtern(content, parentId) {
      if (this.posting) return false;
      if (this.commentsClosed) {
        ElMessage.warning("UP主已关闭评论");
        return false;
      }
      if (!getAccessToken()) {
        this.openLoginModal();
        return false;
      }
      const c = String(content || "").trim();
      if (!c) {
        return false;
      }
      let pid = 0;
      if (parentId !== undefined && parentId !== null && parentId !== "") {
        const n = parseInt(String(parentId), 10);
        pid = Number.isNaN(n) || n < 0 ? 0 : n;
      }
      this.posting = true;
      try {
        if (this.isDynamic) {
          const res = await mbPostDynamicComment(this.dynamicId, c, pid);
          if (res && res.approved === false) {
            ElMessage.success(MB_COMMENT_PENDING_TOAST);
          } else {
            ElMessage.success("发送成功");
          }
        } else if (this.isArticle) {
          const res = await mbPostArticleComment(this.articleId, c, pid);
          if (res && res.approved === false) {
            ElMessage.success(MB_COMMENT_PENDING_TOAST);
          } else {
            ElMessage.success("发送成功");
          }
        } else {
          const res = await mbPostComment(this.videoId, c, pid);
          if (res && res.approved === false) {
            ElMessage.success(MB_COMMENT_PENDING_TOAST);
          } else {
            ElMessage.success("发送成功");
          }
        }
        await this.load({ soft: true, preserveExpand: true });
        return true;
      } catch (e) {
        ElMessage.error(extractApiErrorMessage(e, "发送失败"));
        return false;
      } finally {
        this.posting = false;
      }
    },
    openReplyComposer(parent, ti) {
      if (!getAccessToken()) {
        this.openLoginModal();
        return;
      }
      if (
        this.replyThreadIndex === ti &&
        this.replyParent &&
        Number(this.replyParent.id) === Number(parent.id)
      ) {
        this.closeReplyComposer();
        return;
      }
      this.closeCommentMenu();
      this.replyParent = parent;
      this.replyThreadIndex = ti;
      this.replyDraftText = "";
      this.$nextTick(() => {
        const ta = this.$el?.querySelector?.(
          `[data-mb-thread="${ti}"] .mb-inbox__field`
        );
        if (ta) ta.focus();
      });
    },
    closeReplyComposer() {
      this.replyThreadIndex = null;
      this.replyParent = null;
      this.replyDraftText = "";
      this.replyPosting = false;
    },
    async submitInlineReply() {
      if (!this.replyParent) return;
      const text = String(this.replyDraftText || "").trim();
      if (!text) return;
      this.replyPosting = true;
      try {
        if (this.isDynamic) {
          const res = await mbPostDynamicComment(
            this.dynamicId,
            text,
            this.replyParent.id
          );
          if (res && res.approved === false) {
            ElMessage.success(MB_COMMENT_PENDING_TOAST);
          } else {
            ElMessage.success("发送成功");
          }
        } else if (this.isArticle) {
          const res = await mbPostArticleComment(
            this.articleId,
            text,
            this.replyParent.id
          );
          if (res && res.approved === false) {
            ElMessage.success(MB_COMMENT_PENDING_TOAST);
          } else {
            ElMessage.success("发送成功");
          }
        } else {
          const res = await mbPostComment(this.videoId, text, this.replyParent.id);
          if (res && res.approved === false) {
            ElMessage.success(MB_COMMENT_PENDING_TOAST);
          } else {
            ElMessage.success("发送成功");
          }
        }
        this.closeReplyComposer();
        await this.load({ soft: true, preserveExpand: true });
      } catch (e) {
        ElMessage.error(extractApiErrorMessage(e, "发送失败"));
      } finally {
        this.replyPosting = false;
      }
    },
    onMenuStub(label) {
      this.closeCommentMenu();
      ElMessage.info(`${label}功能即将开放`);
    },
    async onMenuPin(c) {
      if (!getAccessToken()) {
        this.openLoginModal();
        return;
      }
      this.closeCommentMenu();
      try {
        if (this.isDynamic) {
          ElMessage.info("图文动态暂不支持置顶评论");
          return;
        }
        if (this.isArticle) {
          await mbPinArticleComment(c.id);
        } else {
          await mbPinComment(c.id);
        }
        await this.load({ soft: true, preserveExpand: true });
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      }
    },
    async onMenuRemove(c) {
      this.closeCommentMenu();
      const msg = this.deleteConfirmLines(c);
      try {
        await ElMessageBox.confirm(
          msg.includes("\n") ? msg.replace(/\n/g, "<br/>") : msg,
          "提示",
          {
            confirmButtonText: "确认",
            cancelButtonText: "取消",
            center: true,
            showClose: false,
            customClass: "vd-cmb-del-msgbox",
            confirmButtonClass: "vd-cmb-del-msgbox__ok",
            cancelButtonClass: "vd-cmb-del-msgbox__cancel",
            dangerouslyUseHTMLString: msg.includes("\n")
          }
        );
      } catch {
        return;
      }
      try {
        if (this.isDynamic) {
          await mbDeleteDynamicComment(c.id);
        } else if (this.isArticle) {
          await mbDeleteArticleComment(c.id);
        } else {
          await mbDeleteComment(c.id);
        }
        await this.load({ soft: true, preserveExpand: true });
      } catch (e) {
        ElMessage.error((e && e.message) || "删除失败");
      }
    },
    compareRootComments(a, b, sortMode) {
      if (sortMode === "hot") {
        return (
          (Number(b.like_count) || 0) - (Number(a.like_count) || 0) ||
          (Number(b.id) || 0) - (Number(a.id) || 0)
        );
      }
      return (Number(b.id) || 0) - (Number(a.id) || 0);
    },
    buildCommentRoots(flat) {
      const byId = {};
      (flat || []).forEach(c => {
        byId[c.id] = { ...c, _replies: [] };
      });
      const roots = [];
      (flat || []).forEach(c => {
        const pid = Number(c.parent_id) || 0;
        const node = byId[c.id];
        if (pid && byId[pid]) {
          byId[pid]._replies.push(node);
        } else {
          roots.push(node);
        }
      });
      return roots;
    },
    recomputeRootDisplayOrder() {
      const sortMode = this.commentSort === "time" ? "time" : "hot";
      const roots = this.buildCommentRoots(this.items || []);
      roots.sort((a, b) => {
        const ap = a.pinned ? 1 : 0;
        const bp = b.pinned ? 1 : 0;
        if (bp !== ap) return bp - ap;
        return this.compareRootComments(a, b, sortMode);
      });
      const order = {};
      roots.forEach((r, i) => {
        order[String(r.id)] = i;
      });
      this.rootDisplayOrder = order;
    },
    patchCommentById(commentId, patchFn) {
      const id = Number(commentId) || 0;
      if (!id) return;
      const ix = (this.items || []).findIndex(x => Number(x.id) === id);
      if (ix < 0) return;
      const next = patchFn({ ...this.items[ix] });
      this.items.splice(ix, 1, next);
    },
    applyLocalLikeState(item, liked) {
      const wasLiked = !!item.liked_by_me;
      let likeCount = Number(item.like_count) || 0;
      if (liked && !wasLiked) likeCount += 1;
      if (!liked && wasLiked) likeCount = Math.max(0, likeCount - 1);
      return {
        ...item,
        liked_by_me: liked,
        disliked_by_me: liked ? false : item.disliked_by_me,
        like_count: likeCount
      };
    },
    applyLocalDislikeState(item, disliked) {
      const wasLiked = !!item.liked_by_me;
      let likeCount = Number(item.like_count) || 0;
      if (disliked && wasLiked) likeCount = Math.max(0, likeCount - 1);
      return {
        ...item,
        disliked_by_me: disliked,
        liked_by_me: disliked ? false : item.liked_by_me,
        like_count: likeCount
      };
    },
    async toggleLike(c) {
      if (!getAccessToken()) {
        this.openLoginModal();
        return;
      }
      try {
        const { liked } = this.isDynamic
          ? await mbToggleDynamicCommentLike(c.id)
          : this.isArticle
            ? await mbToggleArticleCommentLike(c.id)
            : await mbToggleLike(c.id);
        this.patchCommentById(c.id, item =>
          this.applyLocalLikeState(item, !!liked)
        );
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      }
    },
    async toggleDislike(c) {
      if (!getAccessToken()) {
        this.openLoginModal();
        return;
      }
      try {
        const { disliked } = this.isDynamic
          ? await mbToggleDynamicCommentDislike(c.id)
          : this.isArticle
            ? await mbToggleArticleCommentDislike(c.id)
            : await mbToggleDislike(c.id);
        this.patchCommentById(c.id, item =>
          this.applyLocalDislikeState(item, !!disliked)
        );
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      }
    }
  }
};
</script>

<style scoped lang="scss">
@import "../../styles/vd-comment-list.scss";

.mb-cmt {
  margin-top: 16px;
  padding: 16px;
  border: 1px solid #e3e5e7;
  border-radius: 2px;
  background: #fff;
}
.mb-cmt__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.mb-cmt__title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #18191c;
}
.mb-cmt__count {
  color: #00aeec;
}
.mb-cmt__composer-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-top: 10px;
}
.mb-cmt__err {
  color: #f56c6c;
  font-size: 14px;
}
.mb-cmt__empty,
.mb-cmt__loading {
  margin: 12px 0;
  font-size: 14px;
  color: #9499a0;
}
.mb-cmt--embedded {
  margin-top: 0;
}

.vd-cmt-root-stack {
  display: flex;
  flex-direction: column;
  width: 100%;
}

.vd-cmt-list--soft-refresh {
  opacity: 0.9;
  transition: opacity 0.22s ease;
  pointer-events: none;
}

.mb-thread-reply {
  margin-top: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e5e9ef;
}

.vd-cmt-item.vd-cmt-item--mb-highlight {
  background-color: #edf6fd;
  border-radius: 4px;
  transition: background-color 0.4s ease-out;
}

.vd-cmt-item.vd-cmt-item--mb-highlight.vd-cmt-item--mb-highlight-leaving {
  background-color: transparent;
}

.vd-cmt-item.vd-cmt-item--mb-highlight:not(:hover)
  .vd-cmt-menu-wrap:not(.is-open)
  .vd-cmt-menu-trigger {
  visibility: hidden;
  opacity: 0;
  pointer-events: none;
}

.vd-cmt-face-link {
  display: block;
  flex-shrink: 0;
  line-height: 0;
  border-radius: 50%;
  overflow: hidden;
  align-self: flex-start;
}

a.vd-cmt-name {
  text-decoration: none;
}
</style>

<style lang="scss">
@import "../../styles/vd-cmb-del-msgbox.scss";
</style>
