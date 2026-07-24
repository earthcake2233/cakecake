<template>
  <div class="video-page" :class="{ 'video-page--wide': playerWide }">
    <div class="video-main bili-wrapper">
      <!-- 顶栏：左稿件信息、右 UP（与主站 PC 稿一致） -->
      <header class="video-info-strip">
        <div class="strip-video">
          <h1 class="video-title" :title="pageTitle">{{ pageTitle }}</h1>
          <div class="video-sub-row">
            <span v-if="videoZoneCrumbs.length" class="crumbs">
              <template
                v-for="(crumb, idx) in videoZoneCrumbs"
                :key="crumb.key"
              >
                <span v-if="idx > 0" class="slash">&gt;</span>
                <router-link
                  v-if="crumb.to"
                  class="crumb-link"
                  :to="crumb.to"
                  >{{ crumb.label }}</router-link
                >
                <span v-else class="last">{{ crumb.label }}</span>
              </template>
            </span>
            <span class="pub-time">{{ pubTimeDisplay }}</span>
            <a href="javascript:;" class="link-report">稿件投诉</a>
          </div>
          <div class="video-stats-row">
            <span class="v-stat v-stat--play">
              <i class="v-ico" />
              <span class="v-num">{{ stats.play }}</span>
            </span>
            <span class="v-stat v-stat--dm">
              <i class="v-ico" />
              <span class="v-num">{{ stats.dm }}</span>
            </span>
            <span class="v-stat v-stat--coin">
              <i class="v-ico" />
              <span class="v-label">硬币</span>
              <span class="v-num">{{ stats.coin }}</span>
            </span>
            <span class="v-stat v-stat--fav">
              <i class="v-ico" />
              <span class="v-label">收藏</span>
              <span class="v-num">{{ stats.fav }}</span>
            </span>
          </div>
        </div>
        <div class="strip-up up-card">
          <MbUserHoverCard
            v-if="isMb && mbVideoAuthorId"
            :user-id="mbVideoAuthorId"
            @follow-change="onUpHoverFollowChange"
          >
            <router-link
              v-if="mbVideoAuthorRoute"
              class="up-face"
              :to="mbVideoAuthorRoute"
            >
              <img
                class="up-face-img"
                :src="upFaceSrc"
                :alt="upMeta.name"
                width="68"
                height="68"
              />
            </router-link>
          </MbUserHoverCard>
          <router-link
            v-else-if="isMb && mbVideoAuthorRoute"
            class="up-face"
            :to="mbVideoAuthorRoute"
          >
            <img
              class="up-face-img"
              :src="upFaceSrc"
              :alt="upMeta.name"
              width="68"
              height="68"
            />
          </router-link>
          <a v-else class="up-face" href="javascript:;">
            <img
              class="up-face-img"
              :src="upFaceSrc"
              :alt="upMeta.name"
              width="68"
              height="68"
            />
          </a>
          <div class="up-core">
            <div class="up-row up-row--head">
              <MbUserHoverCard
                v-if="isMb && mbVideoAuthorId"
                :user-id="mbVideoAuthorId"
                @follow-change="onUpHoverFollowChange"
              >
                <router-link
                  v-if="mbVideoAuthorRoute"
                  class="up-name"
                  :to="mbVideoAuthorRoute"
                  >{{ upMeta.name }}</router-link
                >
              </MbUserHoverCard>
              <router-link
                v-else-if="isMb && mbVideoAuthorRoute"
                class="up-name"
                :to="mbVideoAuthorRoute"
                >{{ upMeta.name }}</router-link
              >
              <a v-else class="up-name" href="javascript:;">{{ upMeta.name }}</a>
              <a
                v-if="isMb && mbVideoAuthorId"
                href="javascript:;"
                class="up-msg"
                @click.prevent="onUpMessageClick"
              >
                <svg
                  class="up-msg-svg"
                  viewBox="0 0 24 24"
                  aria-hidden="true"
                  focusable="false"
                >
                  <path
                    fill="currentColor"
                    d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 2v.01L12 13 4 6.01V6h16zm0 12H4V8.99l8 6.99 8-6.99V18z"
                  />
                </svg>
                <span>发消息</span>
              </a>
              <a v-else href="javascript:;" class="up-msg">
                <svg
                  class="up-msg-svg"
                  viewBox="0 0 24 24"
                  aria-hidden="true"
                  focusable="false"
                >
                  <path
                    fill="currentColor"
                    d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 2v.01L12 13 4 6.01V6h16zm0 12H4V8.99l8 6.99 8-6.99V18z"
                  />
                </svg>
                <span>发消息</span>
              </a>
            </div>
            <p class="up-bio">{{ upMeta.bio }}</p>
            <div class="up-stats">
              <span>投稿：{{ upMeta.archive }}</span>
              <span>粉丝：{{ upMeta.fans }}</span>
            </div>
            <div class="up-btns">
              <div class="btn-follow-wrap">
                <span
                  v-show="follow.hint"
                  class="btn-follow-hint"
                  role="status"
                >{{ follow.hint }}</span>
                <button
                  v-if="isMb"
                  type="button"
                  class="btn-follow-main"
                  :class="{ 'is-followed': followed }"
                  :disabled="follow.pending"
                  @mouseenter="onFollowBtnEnter"
                  @mouseleave="onFollowBtnLeave"
                  @click="onFollowClick"
                >
                  {{ followButtonLabel }}
                </button>
              </div>
              <button type="button" class="btn-charge-main">充电</button>
            </div>
          </div>
        </div>
      </header>

      <!-- 左侧栈 position:relative 定高；侧栏 absolute + height:100% 与播放器+工具栏齐平，列表仅在 .side-scroll 内滚动（flex 同行会被侧栏内容撑高整行） -->
      <div class="video-body-row">
        <div class="video-body-stack">
          <div class="player-box-wrap notranslate" translate="no">
            <div v-if="mbDetailLoadError" class="video-load-error" role="alert">
              <p class="video-load-error__text">{{ mbDetailLoadError }}</p>
              <router-link class="video-load-error__link" :to="{ name: 'home' }">
                返回首页
              </router-link>
            </div>
            <VideoPlayerBox
              ref="playerBox"
              v-else
              v-model:wide-mode="playerWide"
              :hot-title="pageTitle"
              :media-src="mbPlayerMediaSrc"
              :seek-to="_seekTime"
              :minibili-video-id="mbNumericId != null ? mbNumericId : 0"
              :minibili-danmaku-closed="!!(apiDetail && apiDetail.danmaku_closed)"
              :danmaku-catalog="mbDanmakuCatalog"
              @mb-danmaku-committed="onMbDanmakuCommitted"
            />
          </div>

          <div class="video-toolbar-wrap">
            <div class="video-toolbar">
              <div class="toolbar-share">
                <span class="share-label">分享</span>
                <span class="share-num">{{ stats.share }}</span>
                <a class="share-dot share-feed" href="javascript:;" title="分享到动态">
                  <i />
                </a>
                <a class="share-dot share-qzone" href="javascript:;" title="分享到QQ空间">
                  <i />
                </a>
                <a class="share-dot share-qq" href="javascript:;" title="分享到QQ">
                  <i />
                </a>
              </div>

              <div class="toolbar-ops">
                <button
                  type="button"
                  class="toolbar-op collect"
                  :class="{ 'is-active': fav.done }"
                  @click="onFavClick"
                  @mouseenter="onFavEnter"
                  @mouseleave="onFavLeave"
                >
                  <span class="op-icon-wrap">
                    <span
                      class="op-icon fav-sprite"
                      :class="{
                        animating: fav.animating && !fav.done,
                        'is-done': fav.done
                      }"
                    />
                  </span>
                  <span class="op-lines">
                    <span class="op-title">{{
                      fav.done ? "已收藏" : "收藏"
                    }}</span>
                    <span class="op-num">{{ stats.fav }}</span>
                  </span>
                </button>

                <button
                  type="button"
                  class="toolbar-op coin-op"
                  :class="{ 'is-active': coin.done }"
                  @click="onCoinClick"
                  @mouseenter="onCoinEnter"
                  @mouseleave="onCoinLeave"
                >
                  <span class="op-icon-wrap is-coin">
                    <span
                      class="op-icon coin-sprite"
                      :class="{
                        animating: coin.animating && !coin.done,
                        'is-done': coin.done
                      }"
                    />
                  </span>
                  <span class="op-lines">
                    <span class="op-title">{{
                      coin.done ? "已投币" : "硬币"
                    }}</span>
                    <span class="op-num">{{ stats.coin }}</span>
                  </span>
                </button>

                <button
                  type="button"
                  class="toolbar-op watch-later-op"
                  :class="{ 'is-active': wait.done }"
                  @click="onWaitClick"
                  @mouseenter="onWaitEnter"
                  @mouseleave="onWaitLeave"
                >
                  <span class="op-icon-wrap is-wait">
                    <span
                      class="op-icon wait-sprite"
                      :class="{
                        animating: wait.animating && !wait.done,
                        'is-done': wait.done
                      }"
                    />
                  </span>
                  <span class="op-lines">
                    <span class="op-title">稍后看</span>
                    <span class="op-sub">马克一下~</span>
                  </span>
                </button>
              </div>
            </div>
          </div>

          <div class="video-side-dock">
            <aside class="video-side video-side--tall side-panel-card">
              <div class="side-head">
                <span class="side-head-text"
                  >{{ sideWatchingDisplay }}人正在看，{{ stats.dm }} 条弹幕</span
                >
                <button type="button" class="side-settings" title="设置" />
              </div>
              <div class="side-tabs">
                <template v-if="!isMb">
                  <span class="tab on">推荐视频</span>
                  <span class="tab">弹幕列表</span>
                  <span class="tab">屏蔽设定</span>
                </template>
                <template v-else>
                  <span
                    class="tab"
                    :class="{ on: sideTab === 'related' }"
                    role="button"
                    tabindex="0"
                    @click="sideTab = 'related'"
                    @keyup.enter="sideTab = 'related'"
                    >推荐视频</span
                  >
                  <span
                    class="tab"
                    :class="{ on: sideTab === 'dm' }"
                    role="button"
                    tabindex="0"
                    @click="sideTab = 'dm'"
                    @keyup.enter="sideTab = 'dm'"
                    >弹幕列表</span
                  >
                  <span
                    class="tab"
                    :class="{ on: sideTab === 'block' }"
                    role="button"
                    tabindex="0"
                    @click="sideTab = 'block'"
                    @keyup.enter="sideTab = 'block'"
                    >屏蔽设定</span
                  >
                </template>
              </div>
              <div class="side-scroll">
                <div
                  v-if="isMb && mbNumericId && sideTab === 'dm'"
                  class="side-scroll-dm-wrap"
                >
                  <MinibiliDanmakuFeed
                    :video-id="mbNumericId"
                    :catalog="mbDanmakuCatalog"
                    :ws-hint="mbDanmakuWsHint"
                  />
                </div>
                <div
                  v-if="isMb && mbNumericId && sideTab === 'block'"
                  class="side-block-placeholder"
                >
                  <p class="side-block-placeholder__t">屏蔽设定敬请期待</p>
                </div>
                <ul v-show="!isMb || sideTab === 'related'" class="related-list">
                  <li
                    v-for="(item, idx) in relatedVideos"
                    :key="item.id != null ? `rel-${item.id}` : `rel-${idx}`"
                    class="related-item"
                  >
                    <router-link
                      v-if="item.playRoute"
                      :to="item.playRoute"
                      class="related-link"
                    >
                      <div class="related-thumb-wrap">
                        <img
                          class="related-thumb"
                          :src="item.cover"
                          :alt="item.title"
                        />
                        <button
                          v-if="isMb && item.id"
                          type="button"
                          class="related-watch-later"
                          :class="{
                            'related-watch-later--on': item.inWatchLater
                          }"
                          :disabled="!!relatedWatchLaterPending[item.id]"
                          aria-label="稍后再看"
                            @click.stop.prevent="onRelatedWatchLater(item)"
                        >
                          <span class="related-watch-later-inner">
                            <span class="related-watch-later-ico-wrap">
                              <img
                                class="related-watch-later-ico"
                                :src="thumbLaterIco"
                                alt=""
                              />
                            </span>
                            <span class="related-watch-later-txt"
                              >稍后再看</span
                            >
                          </span>
                        </button>
                        <span v-else class="related-duration">{{
                          item.duration
                        }}</span>
                      </div>
                      <div class="related-info">
                        <p class="related-title">{{ item.title }}</p>
                        <p class="related-meta">
                          <span class="r-stat">
                            <i class="r-ico r-ico--play" />{{ item.playShort }}
                          </span>
                          <span class="r-stat">
                            <i class="r-ico r-ico--dm" />{{ item.dm }}
                          </span>
                        </p>
                      </div>
                    </router-link>
                    <a v-else href="javascript:;" class="related-link">
                      <div class="related-thumb-wrap">
                        <img
                          class="related-thumb"
                          :src="item.cover"
                          :alt="item.title"
                        />
                        <span class="related-duration">{{ item.duration }}</span>
                      </div>
                      <div class="related-info">
                        <p class="related-title">{{ item.title }}</p>
                        <p class="related-meta">
                          <span class="r-stat">
                            <i class="r-ico r-ico--play" />{{ item.playShort }}
                          </span>
                          <span class="r-stat">
                            <i class="r-ico r-ico--dm" />{{ item.dm }}
                          </span>
                        </p>
                      </div>
                    </a>
                  </li>
                </ul>
              </div>
            </aside>
          </div>
        </div>
      </div>

      <!-- 工具栏下方：标签 / 简介 / 猜你喜欢 / 评论 -->
      <section class="video-below-deck" aria-label="稿件扩展信息">
        <div class="below-main">
          <div class="vd-tags-block">
            <div class="vd-tags-row">
              <router-link
                v-for="(t, i) in videoTags"
                :key="`vd-tag-${i}-${t}`"
                class="vd-tag"
                :to="{ name: 'searchAll', query: { keyword: t } }"
                >{{ t }}</router-link
              >
              <button type="button" class="vd-tag vd-tag--add" title="添加标签">
                +
              </button>
            </div>
          </div>

          <p class="vd-copywarn">
            <span class="vd-copywarn-ico" aria-hidden="true">🚫</span>
            未经作者授权 禁止转载
          </p>
          <p class="vd-desc">{{ videoDescText }}</p>

          <div v-if="alsoLikedVideos.length" class="vd-also">
            <h3 class="vd-also-title">看过该视频的还喜欢</h3>
            <div class="vd-also-carousel">
              <div class="vd-also-stage">
                <button
                  v-show="alsoCanPrev"
                  type="button"
                  class="vd-also-nav vd-also-nav--prev"
                  aria-label="向左滑动"
                  @click="alsoScrollPrev"
                />
                <div ref="alsoViewport" class="vd-also-viewport">
                  <ul class="vd-also-list" :style="alsoTrackStyle">
                    <li
                      v-for="(v, i) in alsoLikedVideos"
                      :key="v.id != null ? `also-${v.id}` : `also-${i}`"
                      class="vd-also-item"
                    >
                      <router-link
                        v-if="v.playRoute"
                        :to="v.playRoute"
                        class="vd-also-link"
                      >
                        <div class="vd-also-thumb-wrap">
                          <img
                            class="vd-also-thumb"
                            :src="v.cover"
                            :alt="v.title"
                          />
                          <span
                            v-if="v.duration"
                            class="vd-also-dur"
                            aria-hidden="true"
                            >{{ v.duration }}</span
                          >
                          <button
                            v-if="isMb && v.id"
                            type="button"
                            class="vd-also-watch-later"
                            :class="{
                              'vd-also-watch-later--on': v.inWatchLater
                            }"
                            :disabled="!!relatedWatchLaterPending[v.id]"
                            aria-label="稍后再看"
                            @click.stop.prevent="onAlsoWatchLater(v)"
                          >
                            <span class="vd-also-watch-later-inner">
                              <span class="vd-also-watch-later-ico-wrap">
                                <img
                                  class="vd-also-watch-later-ico"
                                  :src="thumbLaterIco"
                                  alt=""
                                />
                              </span>
                              <span class="vd-also-watch-later-txt"
                                >稍后再看</span
                              >
                            </span>
                          </button>
                          <span
                            v-else-if="!isMb && v.badge"
                            class="vd-also-badge"
                            >{{ v.badge }}</span
                          >
                        </div>
                        <p class="vd-also-t">{{ v.title }}</p>
                      </router-link>
                      <a v-else href="javascript:;" class="vd-also-link">
                        <div class="vd-also-thumb-wrap">
                          <img
                            class="vd-also-thumb"
                            :src="v.cover"
                            :alt="v.title"
                          />
                          <span
                            v-if="v.duration"
                            class="vd-also-dur"
                            aria-hidden="true"
                            >{{ v.duration }}</span
                          >
                          <span v-if="v.badge" class="vd-also-badge">{{
                            v.badge
                          }}</span>
                        </div>
                        <p class="vd-also-t">{{ v.title }}</p>
                      </a>
                    </li>
                  </ul>
                </div>
                <button
                  v-show="alsoCanNext"
                  type="button"
                  class="vd-also-nav vd-also-nav--next"
                  aria-label="向右滑动"
                  @click="alsoScrollNext"
                />
              </div>
            </div>
          </div>

          <div class="vd-comments" id="comment-main">
            <div class="vd-comments-mock">
            <div class="vd-cmt-head">
              <h3 class="vd-cmt-title">
                <span class="vd-cmt-count">{{ commentTotal }}</span> 评论
              </h3>
              <div class="vd-cmt-head-row vd-cmt-head-row--toolbar">
                <div class="vd-cmt-sort">
                  <span
                    v-if="mbCommentsCurated"
                    class="vd-cmt-curated-label"
                  >{{ MB_COMMENT_CURATED_LABEL }}</span>
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
                <div class="vd-cmt-page-top vd-cmt-page-top--links">
                  <span class="vd-page-info">共{{ commentTotalPages }}页</span>
                  <template
                    v-for="(it, pidx) in commentPagerItems"
                    :key="'vd-cmt-page-top-' + pidx"
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

            <template v-if="isMb && mbNumericId">
              <template v-if="apiDetail && apiDetail.comments_closed">
                <div class="vd-cmt-mb-closed-bar" role="status">
                  UP主已关闭评论区
                </div>
                <p class="vd-cmt-mb-closed-foot">没有更多评论</p>
              </template>
              <template v-else>
                <div class="vd-cmt-composer vd-cmt-composer--mb">
                  <img
                    class="vd-cmt-avatar vd-cmt-avatar--mb"
                    :src="mbComposerAvatarSrc"
                    width="48"
                    height="48"
                    alt=""
                  />
                  <div class="vd-cmt-mb-composer-main">
                    <div class="vd-cmt-mb-editor-row">
                      <div class="vd-cmt-uni-inbox">
                        <template v-if="mbLoggedIn || mbCommentsCurated">
                          <textarea
                            v-model="mbCommentDraft"
                            class="vd-cmt-uni-inbox__field"
                            :class="{ 'is-curated-hint': mbCommentsCurated && !mbLoggedIn }"
                            rows="3"
                            maxlength="1000"
                            :readonly="mbCommentsCurated && !mbLoggedIn"
                            :disabled="mbCommentsCurated && !mbLoggedIn"
                            :placeholder="mbCommentComposerPlaceholder"
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
                            @click="openMbLoginModal"
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
                            :disabled="!mbLoggedIn"
                            @click="onMbEmojiClick"
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
                          (!mbCommentCanSubmit || mbCommentPosting)
                        "
                        @click="onMbComposerSubmit"
                      >
                        <template v-if="mbCommentPosting">发送中…</template>
                        <span v-else class="vd-cmt-submit-lines"
                          >发表<br />评论</span
                        >
                      </button>
                    </div>
                  </div>
                </div>
                <div class="vd-cmt-live-mount">
                  <MinibiliCommentsLive
                    ref="mbCommentsLive"
                    embedded
                    hide-composer
                    :video-id="mbNumericId"
                    :video-author-id="mbVideoAuthorId"
                    :highlight-comment-id="mbHighlightCommentId"
                    :initial-comments-curated="!!(apiDetail && apiDetail.comments_curated)"
                    @counts="onMbCommentCounts"
                  />
                </div>
                <div class="vd-cmt-page-bottom vd-cmt-page-bottom--mb">
                  <div class="vd-cmt-page-bottom-left">
                    <template
                      v-for="(it, pidx) in commentPagerItems"
                      :key="'vd-cmt-page-mb-' + pidx"
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
            </template>
            <template v-else>
            <div class="vd-cmt-composer">
              <img
                class="vd-cmt-avatar"
                :src="mbComposerAvatarSrc"
                width="48"
                height="48"
                alt=""
              />
              <div class="vd-cmt-input-column">
                <div class="vd-cmt-input-shell">
                  <textarea
                    class="vd-cmt-textarea"
                    rows="3"
                    :placeholder="commentPlaceholder"
                    readonly
                  />
                  <div class="vd-cmt-input-foot">
                    <button type="button" class="vd-emoji-btn">
                      <span class="vd-emoji-ico" aria-hidden="true" />
                      表情
                    </button>
                  </div>
                </div>
              </div>
              <button type="button" class="vd-cmt-submit">发表评论</button>
            </div>

            <ul class="vd-cmt-list">
              <li
                v-for="(thread, ti) in commentThreads"
                :key="ti"
                class="vd-cmt-thread"
              >
                <div class="vd-cmt-item vd-cmt-item--root">
                  <img
                    class="vd-cmt-face"
                    :src="thread.avatar"
                    width="48"
                    height="48"
                    alt=""
                  />
                  <div class="vd-cmt-body">
                    <div class="vd-cmt-user-row">
                      <a
                        href="javascript:;"
                        class="vd-cmt-name"
                        :class="'tone-' + thread.nameTone"
                        >{{ thread.user }}</a
                      >
                      <img
                        class="level-badge"
                        :src="levelIconUrl(thread.level)"
                        width="30"
                        height="30"
                        alt=""
                        :title="'LV' + thread.level"
                      />
                    </div>
                    <p class="vd-cmt-text">{{ thread.content }}</p>
                    <div class="vd-cmt-meta-row">
                      <div class="vd-cmt-meta-main">
                        <span>{{ thread.time }}</span>
                        <span class="vd-cmt-ip">{{ thread.ip }}</span>
                        <button type="button" class="vd-cmt-act">
                          <span class="vd-ico-like" aria-hidden="true" />{{
                            thread.likes
                          }}
                        </button>
                        <button type="button" class="vd-cmt-act vd-cmt-act--quiet">
                          <span class="vd-ico-unlike" aria-hidden="true" />
                        </button>
                        <button type="button" class="vd-cmt-act">回复</button>
                      </div>
                      <div
                        class="vd-cmt-menu-wrap vd-cmt-menu-wrap--root"
                        :class="{
                          'is-open':
                            openCommentMenuKey === cmtMenuKey(ti)
                        }"
                        @click.stop
                      >
                        <button
                          type="button"
                          class="vd-cmt-menu-trigger"
                          aria-haspopup="true"
                          :aria-expanded="
                            openCommentMenuKey === cmtMenuKey(ti)
                          "
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
                          <button
                            type="button"
                            class="vd-cmt-menu-item"
                            role="menuitem"
                            @click="closeCommentMenu"
                          >
                            加入黑名单
                          </button>
                          <button
                            type="button"
                            class="vd-cmt-menu-item"
                            role="menuitem"
                            @click="closeCommentMenu"
                          >
                            举报
                          </button>
                        </div>
                      </div>
                    </div>

                    <ul
                      v-if="thread.replies && thread.replies.length"
                      class="vd-cmt-replies"
                    >
                      <li
                        v-for="(rp, ri) in thread.replies"
                        :key="ri"
                        class="vd-cmt-item vd-cmt-item--reply"
                      >
                        <img
                          class="vd-cmt-face vd-cmt-face--sm"
                          :src="rp.avatar"
                          width="32"
                          height="32"
                          alt=""
                        />
                        <div class="vd-cmt-body">
                          <div class="vd-cmt-user-row">
                            <a
                              href="javascript:;"
                              class="vd-cmt-name"
                              :class="'tone-' + rp.nameTone"
                              >{{ rp.user }}</a
                            >
                            <img
                              class="level-badge"
                              :src="levelIconUrl(rp.level)"
                              width="30"
                              height="30"
                              alt=""
                              :title="'LV' + rp.level"
                            />
                          </div>
                          <p class="vd-cmt-text">{{ rp.content }}</p>
                          <div class="vd-cmt-meta-row">
                            <div class="vd-cmt-meta-main">
                              <span>{{ rp.time }}</span>
                              <span class="vd-cmt-ip">{{ rp.ip }}</span>
                              <button type="button" class="vd-cmt-act">
                                <span class="vd-ico-like" aria-hidden="true" />{{
                                  rp.likes
                                }}
                              </button>
                              <button
                                type="button"
                                class="vd-cmt-act vd-cmt-act--quiet"
                              >
                                <span class="vd-ico-unlike" aria-hidden="true" />
                              </button>
                              <button type="button" class="vd-cmt-act">
                                回复
                              </button>
                              <a
                                v-if="rp.viewConv"
                                href="javascript:;"
                                class="vd-cmt-link"
                                >查看对话</a
                              >
                            </div>
                            <div
                              class="vd-cmt-menu-wrap vd-cmt-menu-wrap--reply"
                              :class="{
                                'is-open':
                                  openCommentMenuKey === cmtMenuKey(ti, ri)
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
                                @click="
                                  toggleCommentMenu(cmtMenuKey(ti, ri), $event)
                                "
                              >
                                <span class="vd-cmt-menu-dots" aria-hidden="true">
                                  <span /><span /><span />
                                </span>
                              </button>
                              <div
                                v-if="
                                  openCommentMenuKey === cmtMenuKey(ti, ri)
                                "
                                class="vd-cmt-menu-dropdown"
                                role="menu"
                              >
                                <button
                                  type="button"
                                  class="vd-cmt-menu-item"
                                  role="menuitem"
                                  @click="closeCommentMenu"
                                >
                                  加入黑名单
                                </button>
                                <button
                                  type="button"
                                  class="vd-cmt-menu-item"
                                  role="menuitem"
                                  @click="closeCommentMenu"
                                >
                                  举报
                                </button>
                              </div>
                            </div>
                          </div>
                        </div>
                      </li>
                    </ul>
                    <p v-if="thread.replyFold" class="vd-cmt-fold">
                      {{ thread.replyFold }}
                      <a href="javascript:;">点击查看</a>
                    </p>
                  </div>
                </div>
              </li>
            </ul>

            <div class="vd-cmt-page-bottom vd-cmt-page-bottom--mock">
              <div class="vd-cmt-page-bottom-left">
                <template
                  v-for="(it, pidx) in commentPagerItems"
                  :key="'vd-cmt-page-bot-' + pidx"
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
        </div>
      </section>
    </div>

    <VideoCoinDialog
      v-model="coinDialogOpen"
      :loading="coin.dialogLoading"
      :is-own-video="mbCoinIsOwnVideo"
      :coin-balance="mbCoinBalance"
      :prior-coin-amount="coin.myAmount"
      :daily-coin-exp-progress="coin.dailyExpProgress"
      :daily-coin-exp-max="coin.dailyExpMax"
      @confirm="onCoinDialogConfirm"
      @cancel="coinDialogOpen = false"
    />

    <VideoFavoriteFolderDialog
      v-model="favDialogOpen"
      :video-id="mbNumericId"
      :loading="fav.dialogLoading"
      @confirm="onFavDialogConfirm"
      @cancel="favDialogOpen = false"
    />
  </div>
</template>

<script>
import { createNamespacedHelpers } from "vuex";
import akariCover from "@/assets/akari.jpg";
import thumbLaterIco from "@/assets/personal_space/latertowatch.png";
import http from "@/utils/http";
import { getAccessToken, getUserId } from "@/utils/authTokens";
import {
  mbWsDanmakuUrl,
  mbSetVideoFavoriteFolders,
  mbPostVideoCoin,
  mbToggleWatchLater,
  mbMarkWatchLaterWatched,
  mbToggleUserFollow,
  mbPostViewHistory,
  mbListVideos,
  mbListUserPublishedVideos
} from "@/api/minibili";
import {
  filterRecommendPool,
  interleaveRecommendPools,
  mapToAlsoLikedRow,
  mapToRelatedVideoRow,
  zoneParentFromDetail
} from "@/utils/videoRecommendFeeds";
import { ElMessage } from "element-plus";
import { minibiliUserSpaceRoute } from "@/utils/minibiliRoutes";
import {
  formatVideoBvid,
  parseVideoIdFromRoute
} from "@/utils/videoBvid";
import { buildDocumentTitle } from "@/constants/siteTitle";
import { videoZoneCrumbs as buildVideoZoneCrumbs } from "@/utils/videoZone.js";
import VideoPlayerBox from "@/components/video/VideoPlayerBox.vue";
import VideoCoinDialog from "@/components/video/VideoCoinDialog.vue";
import VideoFavoriteFolderDialog from "@/components/video/VideoFavoriteFolderDialog.vue";
import MinibiliCommentsLive from "@/pages/minibili/MinibiliCommentsLive.vue";
import {
  MB_COMMENT_CURATED_LABEL,
  MB_COMMENT_CURATED_PLACEHOLDER
} from "@/constants/minibiliComments";
import MinibiliDanmakuFeed from "@/pages/minibili/MinibiliDanmakuFeed.vue";
import MbUserHoverCard, {
  invalidateUserHoverProfileCache
} from "@/components/minibili/MbUserHoverCard.vue";

const { mapState, mapActions } = createNamespacedHelpers("login");

export default {
  components: {
    VideoPlayerBox,
    VideoCoinDialog,
    VideoFavoriteFolderDialog,
    MinibiliCommentsLive,
    MinibiliDanmakuFeed,
    MbUserHoverCard
  },
  data() {
    const demoCover = akariCover;
    const mbOn =
      import.meta.env.VITE_MINIBILI_API === "true" ||
      import.meta.env.VITE_MINIBILI_API === "1";
    return {
      MB_COMMENT_CURATED_LABEL,
      thumbLaterIco,
      apiDetail: null,
      _mbRecGen: 0,
      relatedWatchLaterPending: {},
      alsoCarouselIndex: 0,
      alsoScrollStep: 0,
      alsoVisibleCount: 5,
      _alsoResizeBound: null,
      mbVideoUrl: "",
      /** 稿件加载失败时在播放区展示，不跳转 404（避免污染浏览器历史） */
      mbDetailLoadError: "",
      /** 侧栏 Tab：related | dm | block */
      sideTab: "related",
      /** WebSocket 同步的弹幕（供列表 + 播放器 Canvas） */
      mbDanmakuCatalog: [],
      mbDanmakuWsHint: "",
      _mbDmWs: null,
      _seekTime: 0,
      playerWide: false,
      /** 侧栏「正在看」：Mini-Bili 下为真实人数（详情 + WS）；非 MB 见 sideWatchingDisplay */
      watching: 0,
      followed: false,
      follow: { pending: false, hover: false, hint: "" },
      _followHintTimer: null,
      upMeta: {
        name: "",
        bio: "",
        archive: "0",
        fans: "0"
      },
      stats: {
        play: "29.4万",
        dm: "211",
        coin: "154",
        fav: "1212",
        share: "2"
      },
      fav: {
        animating: false,
        hover: false,
        done: false,
        pending: false,
        dialogLoading: false
      },
      favDialogOpen: false,
      coin: {
        animating: false,
        hover: false,
        done: false,
        myAmount: 0,
        dailyExpProgress: 0,
        dailyExpMax: 50,
        pending: false,
        dialogLoading: false
      },
      coinDialogOpen: false,
      wait: { animating: false, hover: false, done: false, pending: false },
      commentSort: "hot",
      openCommentMenuKey: "",
      videoDescText:
        "据美国国防部消息，美方近期公布了部分与不明飞行物（UFO）相关的调查文件，相关内容在网络上引发热烈讨论。本文为本地静态演示页文案占位，用于还原播放页下方信息区布局。",
      alsoLikedVideos: mbOn ? [] : [
        {
          title: "每天一杯奶茶，血液竟会变成乳白色？",
          duration: "04:49",
          badge: "食话实说",
          cover: demoCover
        },
        {
          title: "熊孩子玩火点燃杨絮火势瞬间蔓延，村民提桶端盆扑救",
          duration: "12:08",
          badge: "CCTV",
          cover: demoCover
        },
        {
          title:
            "B站网友让我“夜闯”德特里克堡，我在路上遇见了一个“圈内…”",
          duration: "12:08",
          badge: "12:08",
          cover: demoCover
        },
        {
          title: "gogogo和我一起出发珀斯！（来自天依的肯定(づ｡◕‿‿◕｡)づ",
          duration: "03:42",
          badge: "03:42",
          cover: demoCover
        },
        {
          title: "机器下班AI还在“复盘”？探访会自我进化的汽车工厂",
          duration: "18:56",
          badge: "纪录片",
          cover: demoCover
        },
        {
          title: "【演示】横向滑动样本 ⑥",
          duration: "07:33",
          cover: demoCover
        },
        {
          title: "【演示】横向滑动样本 ⑦",
          duration: "09:41",
          cover: demoCover
        }
      ],
      commentTotal: 2753,
      commentTotalPages: 138,
      /** 评论区页码（当前列表未接后端分页时仅用于顶栏/底栏展示与跳转） */
      commentCurrentPage: 1,
      commentPageJumpDraft: "1",
      commentPlaceholder:
        "只是一直在等你而已，才不是想被评论呢~",
      mbCommentDraft: "",
      mbCommentPosting: false,
      commentThreads: [
        {
          user: "一条会飞的隆",
          nameTone: "blue",
          level: 4,
          avatar: demoCover,
          content:
            "太厉害了，不到全球7%的领土却贡献了全球50%的UFO目击[doge]，还打下来了好厉害，这个外星飞船千里迢迢飞来地球搞了半天连个导弹都没办法[doge]",
          time: "2026-05-08 23:20",
          ip: "IP属地：广东",
          likes: 310,
          more: true,
          replyFold: "共27条回复，",
          replies: [
            {
              user: "云崖くん",
              nameTone: "pink",
              level: 1,
              avatar: demoCover,
              content: "笑死我了哈哈哈哈😂😂",
              time: "2026-05-08 23:27",
              ip: "IP属地：安徽",
              likes: 5,
              viewConv: false
            },
            {
              user: "宁静致远202",
              nameTone: "black",
              level: 2,
              avatar: demoCover,
              content: "毕竟是受到神仙眷顾的土地[doge]",
              time: "2026-05-08 23:48",
              ip: "IP属地：北京",
              likes: 2,
              viewConv: false
            },
            {
              user: "小野塚小町",
              nameTone: "black",
              level: 3,
              avatar: demoCover,
              content:
                "😂那我们就不得不祭出孟照国了，人家和外星人的交流深入多了。",
              time: "6小时前",
              ip: "IP属地：重庆",
              likes: 1,
              viewConv: false
            },
            {
              user: "北风其凉",
              nameTone: "black",
              level: 5,
              avatar: demoCover,
              content: "演示：本条为 LV5。",
              time: "5小时前",
              ip: "IP属地：江苏",
              likes: 0,
              viewConv: false
            },
            {
              user: "城南旧事",
              nameTone: "black",
              level: 6,
              avatar: demoCover,
              content: "演示：本条为 LV6。",
              time: "5小时前",
              ip: "IP属地：浙江",
              likes: 0,
              viewConv: false
            }
          ]
        },
        {
          user: "荣耀之剑又空大",
          nameTone: "pink",
          level: 6,
          avatar: demoCover,
          content:
            "外星人只能是人型！\n外星人只能说英语！\n外星人只能接触美国！\n一句话！文明！太特么文明了😅😅",
          time: "16小时前",
          ip: "IP属地：广东",
          likes: 83,
          more: false,
          replyFold: "",
          replies: [
            {
              user: "非中非非",
              nameTone: "black",
              level: 5,
              avatar: demoCover,
              content:
                "外星人一定要在美国，外星人几十亿年不来，转挑你活着的几十年来😅",
              time: "7小时前",
              ip: "IP属地：天津",
              likes: 1,
              viewConv: false
            },
            {
              user: "二象性的猫",
              nameTone: "black",
              level: 6,
              avatar: demoCover,
              content:
                "回复 @爱喝可乐好喝多喝 ：非洲大区这么乱，政府也都是不管事的，怎么没见他们说外星人？拜托你是相信非洲部落国家能铁板一块保密？",
              time: "4小时前",
              ip: "IP属地：河南",
              likes: 1,
              viewConv: true
            },
            {
              user: "爱喝可乐好喝多喝",
              nameTone: "black",
              level: 4,
              avatar: demoCover,
              content:
                "你这种思维就局限了不是. 有没有可能外星人全世界跑.但就美国开源了.其他的国家是保密呢?",
              time: "4小时前",
              ip: "IP属地：上海",
              likes: 0,
              viewConv: false
            }
          ]
        },
        {
          user: "深山老林客",
          nameTone: "black",
          level: 6,
          avatar: demoCover,
          content: "基本无奈",
          time: "5小时前",
          ip: "IP属地：山东",
          likes: 5,
          more: false,
          replyFold: "共12条回复，",
          replies: [
            {
              user: "冥月渝",
              nameTone: "pink",
              level: 5,
              avatar: demoCover,
              content: "回复 @我本将心__ ：还有地平说支持者",
              time: "4小时前",
              ip: "IP属地：广东",
              likes: 2,
              viewConv: true
            }
          ]
        }
      ],
      relatedVideos: mbOn ? [] : [
        {
          title: "【演示】本地静态相关视频样式占位 ①",
          duration: "12:08",
          playShort: "32.1万",
          dm: "856",
          cover: demoCover
        },
        {
          title: "【演示】本地静态相关视频样式占位 ②",
          duration: "05:42",
          playShort: "8.4万",
          dm: "120",
          cover: demoCover
        },
        {
          title: "【演示】本地静态相关视频样式占位 ③",
          duration: "18:56",
          playShort: "120万",
          dm: "3401",
          cover: demoCover
        },
        {
          title: "【演示】本地静态相关视频样式占位 ④",
          duration: "03:15",
          playShort: "6400",
          dm: "12",
          cover: demoCover
        },
        {
          title: "【演示】本地静态相关视频样式占位 ⑤",
          duration: "42:00",
          playShort: "56万",
          dm: "892",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑥ · 稍长标题用于观察两行省略与列表滚动区域",
          duration: "07:33",
          playShort: "12.8万",
          dm: "445",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑦",
          duration: "01:02",
          playShort: "2.1万",
          dm: "88",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑧ · 科普向合集节选（本地占位）",
          duration: "25:17",
          playShort: "89万",
          dm: "5021",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑨",
          duration: "09:41",
          playShort: "15.6万",
          dm: "1203",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑩",
          duration: "14:09",
          playShort: "42万",
          dm: "2890",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑪ · 竖屏直播切片占位文案",
          duration: "04:55",
          playShort: "9800",
          dm: "56",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑫",
          duration: "36:22",
          playShort: "210万",
          dm: "1.2万",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑬",
          duration: "02:48",
          playShort: "6.7万",
          dm: "334",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑭ · 音乐现场节选（演示）",
          duration: "11:11",
          playShort: "33万",
          dm: "678",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑮",
          duration: "08:20",
          playShort: "4.2万",
          dm: "201",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑯",
          duration: "19:45",
          playShort: "72万",
          dm: "4102",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑰ · 游戏实况章节占位",
          duration: "52:03",
          playShort: "156万",
          dm: "9800",
          cover: demoCover
        },
        {
          title: "【滚动测试】推荐列表样本 ⑱ · 列表末尾用于确认滚到底部边界",
          duration: "06:06",
          playShort: "1.3万",
          dm: "99",
          cover: demoCover
        }
      ]
    };
  },
  computed: {
    ...mapState({
      proInfo: state => state.proInfo,
      minibiliMe: state => state.minibiliMe
    }),
    aidParam() {
      return this.$route.params.aid || "";
    },
    pageTitle() {
      if (this.apiDetail && this.apiDetail.title) {
        return this.apiDetail.title;
      }
      const a = this.aidParam;
      return a ? `${a} 示例标题（本地页面）` : "番剧";
    },
    isMb() {
      return (
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1"
      );
    },
    /** 评论框左侧头像：与 Vuex 个人资料同步（不写死 akari） */
    mbComposerAvatarSrc() {
      void this.minibiliMe;
      void this.proInfo;
      if (this.isMb && this.minibiliMe) {
        const u = String(this.minibiliMe.avatar_url || "").trim();
        if (u) {
          return u;
        }
      }
      const p = this.proInfo;
      if (p && typeof p === "object" && !Array.isArray(p) && p.face) {
        return p.face;
      }
      return akariCover;
    },
    /** 侧栏「正在看」展示：MB 为后端/WS 人数；本地克隆页为占位 */
    sideWatchingDisplay() {
      if (!this.isMb) return 145;
      return this.watching;
    },
    mbNumericId() {
      return parseVideoIdFromRoute(this.aidParam);
    },
    videoBvidDisplay() {
      const id = this.mbNumericId;
      return id != null ? formatVideoBvid(id) : "";
    },
    /** 消息中心等跳转：?mb_cid=评论ID，评论区对应楼层标蓝并滚入视野 */
    mbHighlightCommentId() {
      const q = this.$route.query && this.$route.query.mb_cid;
      if (q == null || q === "") return null;
      const raw = Array.isArray(q) ? q[0] : q;
      const n = parseInt(String(raw), 10);
      return Number.isFinite(n) && n > 0 ? n : null;
    },
    /** 播放页标签：Mini-Bili 用详情接口 tags；本地演示保留占位 */
    videoTags() {
      if (!this.isMb) {
        return ["UFO", "美国", "外星人", "特朗普", "文件"];
      }
      const d = this.apiDetail;
      if (!d || !Array.isArray(d.tags)) {
        return [];
      }
      return d.tags.map(t => String(t).trim()).filter(Boolean);
    },
    /** 稿件 UP 主头像（Mini-Bili 详情 uploader_avatar_url） */
    upFaceSrc() {
      const d = this.apiDetail;
      if (this.isMb && d && String(d.uploader_avatar_url || "").trim()) {
        return String(d.uploader_avatar_url).trim();
      }
      return akariCover;
    },
    pubTimeDisplay() {
      if (this.apiDetail && this.apiDetail.created_at) {
        return this.apiDetail.created_at;
      }
      return "2026-05-08 23:17:50";
    },
    /** 播放页分区面包屑（Mini-Bili 详情 zone；无分区则不展示） */
    videoZoneCrumbs() {
      if (!this.isMb || !this.apiDetail) {
        return [];
      }
      const zone =
        this.apiDetail.zone ||
        (this.apiDetail.zone_parent && this.apiDetail.zone_child
          ? `${this.apiDetail.zone_parent}-${this.apiDetail.zone_child}`
          : this.apiDetail.zone_parent || "");
      return buildVideoZoneCrumbs(zone);
    },
    /** 交给 VideoPlayerBox 内层 <video>：Mini-Bili 已发布稿件用 OSS MP4，否则走组件内演示片 */
    mbPlayerMediaSrc() {
      if (!this.isMb) return "";
      return (this.mbVideoUrl && String(this.mbVideoUrl).trim()) || "";
    },
    mbVideoAuthorId() {
      const d = this.apiDetail;
      if (!d || d.user_id == null) return null;
      const n = Number(d.user_id);
      return Number.isFinite(n) ? n : null;
    },
    mbVideoAuthorRoute() {
      if (!this.isMb || this.mbVideoAuthorId == null) {
        return null;
      }
      return minibiliUserSpaceRoute(this.mbVideoAuthorId);
    },
    mbLoggedIn() {
      void this.$store.state.login.signIn;
      void this.$store.state.login.minibiliMe;
      void this.$route.fullPath;
      return !!getAccessToken();
    },
    mbCommentsCurated() {
      return !!(this.apiDetail && this.apiDetail.comments_curated);
    },
    mbCommentComposerPlaceholder() {
      return this.mbCommentsCurated
        ? MB_COMMENT_CURATED_PLACEHOLDER
        : "评论千万条，等你发一条";
    },
    mbCoinIsOwnVideo() {
      if (!this.isMb || !this.mbLoggedIn) return false;
      const me = getUserId();
      const author = this.mbVideoAuthorId;
      if (me == null || author == null) return false;
      return Number(me) === Number(author);
    },
    mbCoinBalance() {
      const me = this.minibiliMe;
      if (me && typeof me.coin_balance === "number") {
        return me.coin_balance;
      }
      const p = this.proInfo;
      if (p && typeof p === "object" && !Array.isArray(p) && typeof p.money === "number") {
        return p.money;
      }
      return 0;
    },
    mbFollowIsOwnVideo() {
      return this.mbCoinIsOwnVideo;
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
    },
    mbCommentCanSubmit() {
      return (
        this.mbLoggedIn &&
        this.mbCommentDraft.trim().length > 0 &&
        !this.mbCommentPosting
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
    },
    alsoMaxIndex() {
      return Math.max(
        0,
        this.alsoLikedVideos.length - this.alsoVisibleCount
      );
    },
    alsoCanPrev() {
      return this.alsoCarouselIndex > 0;
    },
    alsoCanNext() {
      return this.alsoCarouselIndex < this.alsoMaxIndex;
    },
    alsoTrackStyle() {
      const step = this.alsoScrollStep;
      if (!step) return {};
      return {
        transform: `translateX(-${this.alsoCarouselIndex * step}px)`
      };
    }
  },
  watch: {
    aidParam() {
      this.mbDetailLoadError = "";
      this.mbCommentDraft = "";
      this.commentCurrentPage = 1;
      this.commentPageJumpDraft = "1";
      this.watching = 0;
      this.resetEngagementUi();
      this.coinDialogOpen = false;
      if (this.isMb) {
        this._mbRecGen += 1;
        this.relatedVideos = [];
        this.alsoLikedVideos = [];
        this.alsoCarouselIndex = 0;
      }
      this._seekTime = 0;
      var tq = Number(this.$route.query.t);
      if (Number.isFinite(tq) && tq > 0) this._seekTime = tq;
      this.syncTitle();
      this.syncMinibiliDetail();
      this.syncMbDanmakuWs();
    },
    "$route.query.t": {
      handler(t) {
        var tq = Number(t);
        if (Number.isFinite(tq) && tq > 0) this._seekTime = tq;
      }
    },
    mbLoggedIn(v, oldV) {
      if (!this.isMb) {
        return;
      }
      if (v && oldV !== v) {
        void this.refreshMinibiliMe().catch(() => {});
      }
      this.syncMbDanmakuWs();
    },
    alsoLikedVideos: {
      handler() {
        this.alsoCarouselIndex = 0;
        this.$nextTick(() => this.syncAlsoCarouselMetrics());
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
  mounted() {
    this.syncTitle();
    this.syncMinibiliDetail();
    var tq = Number(this.$route.query.t);
    if (Number.isFinite(tq) && tq > 0) { this._seekTime = tq; } else { this._seekTime = 0; }
    if (this.isMb) {
      void this.refreshMinibiliMe().catch(() => {});
    }
    this.syncMbDanmakuWs();
    document.addEventListener("click", this.closeCommentMenu);
    this._alsoResizeBound = () => this.syncAlsoCarouselMetrics();
    window.addEventListener("resize", this._alsoResizeBound);
    this.$nextTick(() => this.syncAlsoCarouselMetrics());
  },
  beforeUnmount() {
    this.clearFollowHintTimer();
    this.teardownMbDanmakuWs();
    document.removeEventListener("click", this.closeCommentMenu);
    if (this._alsoResizeBound) {
      window.removeEventListener("resize", this._alsoResizeBound);
      this._alsoResizeBound = null;
    }
  },
  methods: {
    ...mapActions(["refreshMinibiliMe"]),
    normalizeDanmakuRow(it) {
      const rawFs = String((it && it.font_size) || "").trim().toLowerCase();
      let font_size = "md";
      if (rawFs === "sm" || rawFs === "small") font_size = "sm";
      else if (rawFs === "lg" || rawFs === "large") font_size = "lg";
      return {
        id: Number(it.id) || 0,
        content: String(it.content || ""),
        color: String(it.color || "#ffffff"),
        type: String(it.type || "scroll"),
        font_size,
        video_time: Number(it.video_time) || 0,
        user: String(it.user || ""),
        created_at: String(it.created_at || "")
      };
    },
    mergeMbHistory(items) {
      const rows = (items || []).map(it => this.normalizeDanmakuRow(it));
      rows.sort((a, b) => b.id - a.id);
      this.mbDanmakuCatalog = rows.slice(0, 400);
    },
    onMbDanmakuCommitted(row) {
      if (!row || row.id == null) return;
      this.pushMbLive(row);
    },
    pushMbLive(d) {
      const row = this.normalizeDanmakuRow(d);
      if (!row.id) return;
      const ix = this.mbDanmakuCatalog.findIndex(x => x.id === row.id);
      if (ix >= 0) this.mbDanmakuCatalog.splice(ix, 1, row);
      else this.mbDanmakuCatalog.unshift(row);
      if (this.mbDanmakuCatalog.length > 400) {
        this.mbDanmakuCatalog.length = 400;
      }
    },
    teardownMbDanmakuWs() {
      const w = this._mbDmWs;
      this._mbDmWs = null;
      if (!w) return;
      w.onopen = null;
      w.onclose = null;
      w.onerror = null;
      w.onmessage = null;
      try {
        w.close();
      } catch {
        /* noop */
      }
    },
    syncMbDanmakuWs() {
      this.teardownMbDanmakuWs();
      this.mbDanmakuCatalog = [];
      this.mbDanmakuWsHint = "";
      if (!this.isMb || this.mbNumericId == null) {
        return;
      }
      const id = this.mbNumericId;
      const token = getAccessToken();
      const url = mbWsDanmakuUrl(id, token || undefined);
      const ws = new WebSocket(url);
      this._mbDmWs = ws;
      ws.onopen = () => {
        this.mbDanmakuWsHint = "";
      };
      ws.onclose = () => {
        if (this._mbDmWs === ws) this._mbDmWs = null;
      };
      ws.onerror = () => {
        this.mbDanmakuWsHint = "弹幕通道连接异常";
      };
      ws.onmessage = ev => {
        let msg;
        try {
          msg = JSON.parse(ev.data);
        } catch {
          return;
        }
        if (msg.type === "auth_failed") {
          this.mbDanmakuWsHint = msg.msg || "鉴权失败";
          return;
        }
        if (msg.type === "history" && Array.isArray(msg.items)) {
          this.mergeMbHistory(msg.items);
          if (this.isMb) {
            const w = Number(msg.watching_count);
            if (Number.isFinite(w) && w >= 0) this.watching = Math.floor(w);
          }
          return;
        }
        if (msg.type === "watching" && this.isMb) {
          const w = Number(msg.count);
          if (Number.isFinite(w) && w >= 0) this.watching = Math.floor(w);
          return;
        }
        if (msg.type === "danmaku" && msg.data) {
          this.pushMbLive(msg.data);
        }
      };
    },
    /** 连接 Mini-Bili 后端时拉取稿件元数据，并通过 VideoPlayerBox 的 mediaSrc 播放 OSS */
    syncMinibiliDetail() {
      const on =
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1";
      if (!on) {
        this.mbVideoUrl = "";
        this.mbDetailLoadError = "";
        return;
      }
      this.mbDetailLoadError = "";
      const idNum = this.mbNumericId;
      if (idNum == null || idNum <= 0) {
        this.mbDetailLoadError = "视频不存在或链接无效";
        this.apiDetail = null;
        this.mbVideoUrl = "";
        return;
      }
      const id = String(idNum);
      this.upMeta.bio = "";
      http
        .get(`/api/v1/videos/${id}`)
        .then(body => {
          if (!body || body.code !== 0 || !body.data) {
            this.mbDetailLoadError = "视频不存在或暂不可播放";
            this.apiDetail = null;
            this.mbVideoUrl = "";
            return;
          }
          const d = body.data;
          this.apiDetail = d;
          const url = (d.video_url && String(d.video_url).trim()) || "";
          this.mbVideoUrl = url;
          this.stats.play = String(d.play_count ?? 0);
          this.stats.dm = String(d.danmaku_count ?? 0);
          this.stats.fav = String(d.fav_count ?? 0);
          this.stats.coin = String(d.coin_count ?? 0);
          this.fav.done = !!d.favorited_by_me;
          this.applyCoinStateFromDetail(d);
          this.wait.done = !!d.in_watch_later;
          if (d.in_watch_later && getAccessToken()) {
            void mbMarkWatchLaterWatched(id).catch(() => {});
          }
          if (getAccessToken()) {
            const dur = Number(d.duration_sec ?? d.duration ?? 0);
            void mbPostViewHistory(idNum, {
              progress_sec: 0,
              duration_sec: Number.isFinite(dur) && dur > 0 ? dur : 0,
              device: "web"
            }).catch(() => {});
          }
          {
            const w = Number(d.watching_count);
            this.watching =
              Number.isFinite(w) && w >= 0 ? Math.floor(w) : 0;
          }
          const cc = Number(d.comment_count);
          if (!Number.isNaN(cc)) {
            this.commentTotal = cc;
            this.commentTotalPages = Math.max(1, Math.ceil(cc / 20));
            if (this.commentCurrentPage > this.commentTotalPages) {
              this.commentCurrentPage = this.commentTotalPages;
            }
            this.commentPageJumpDraft = String(this.commentCurrentPage);
          }
          if (d.uploader) {
            this.upMeta.name = d.uploader;
          }
          this.followed = !!d.followed_by_me;
          const fans = Number(d.uploader_follower_count);
          if (Number.isFinite(fans) && fans >= 0) {
            this.upMeta.fans = String(fans);
          }
          const archives = Number(d.uploader_published_count);
          if (Number.isFinite(archives) && archives >= 0) {
            this.upMeta.archive = String(archives);
          }
          const sign = String(d.uploader_sign ?? "").trim();
          this.upMeta.bio =
            sign || "这个家伙很懒，什么都没有写";
          if (typeof d.description === "string") {
            this.videoDescText = d.description;
          }
          this.syncTitle();
          void this.loadMbRecommendFeeds(d);
        })
        .catch(() => {
          this.mbDetailLoadError = "视频不存在或暂不可播放";
          this.apiDetail = null;
          this.mbVideoUrl = "";
        });
    },
    /** 推荐视频（全站热榜）与同分区/同 UP 稿件交替合并，分别填入侧栏与下方还喜欢 */
    async loadMbRecommendFeeds(detail) {
      if (!this.isMb || this.mbNumericId == null) return;
      const gen = ++this._mbRecGen;
      const excludeId = this.mbNumericId;
      const fallbackCover = akariCover;
      const zoneParent = zoneParentFromDetail(detail);
      const authorId = Number(detail && detail.user_id);
      try {
        const hotP = mbListVideos({ limit: 50, sort: "hot" });
        const zoneP = zoneParent
          ? mbListVideos({ limit: 50, sort: "hot", zone_parent: zoneParent })
          : Promise.resolve({ items: [], next_cursor: "" });
        const upP =
          Number.isFinite(authorId) && authorId > 0
            ? mbListUserPublishedVideos(authorId, { limit: 40 })
            : Promise.resolve({ items: [], next_cursor: "" });
        const [hotRes, zoneRes, upRes] = await Promise.all([hotP, zoneP, upP]);
        if (gen !== this._mbRecGen) return;

        const recommendedPool = filterRecommendPool(hotRes.items, excludeId);
        let alsoPool = filterRecommendPool(zoneRes.items, excludeId);
        if (alsoPool.length < 8) {
          const upItems = filterRecommendPool(upRes.items, excludeId);
          alsoPool = interleaveRecommendPools(alsoPool, upItems);
        }
        if (alsoPool.length < 8) {
          const hotTail = filterRecommendPool(hotRes.items, excludeId).slice(
            12
          );
          alsoPool = interleaveRecommendPools(alsoPool, hotTail);
        }

        const merged = interleaveRecommendPools(recommendedPool, alsoPool);
        if (!merged.length) return;

        this.relatedVideos = merged.map(it =>
          mapToRelatedVideoRow(it, fallbackCover)
        );
        this.alsoLikedVideos = merged.map(it =>
          mapToAlsoLikedRow(it, fallbackCover)
        );
        this.alsoCarouselIndex = 0;
        this.$nextTick(() => this.syncAlsoCarouselMetrics());
      } catch {
        /* 保留空列表或上一批数据 */
      }
    },
    onMbCommentCounts(n) {
      const v = Number(n);
      if (Number.isNaN(v)) return;
      this.commentTotal = v;
      this.commentTotalPages = Math.max(1, Math.ceil(v / 20));
      if (this.commentCurrentPage > this.commentTotalPages) {
        this.commentCurrentPage = this.commentTotalPages;
      }
      this.commentPageJumpDraft = String(this.commentCurrentPage);
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
    openMbLoginModal() {
      this.$store.commit("login/SET_LOGIN_TAB", 0);
      this.$store.commit("login/OPEN_LOGIN_MODAL");
    },
    onMbComposerSubmit() {
      if (!this.mbLoggedIn) {
        this.openMbLoginModal();
        return;
      }
      this.submitMbComment();
    },
    onMbEmojiClick() {
      if (!this.mbLoggedIn) {
        this.openMbLoginModal();
      }
    },
    async submitMbComment() {
      if (this.mbCommentPosting) return;
      if (!getAccessToken() || !this.mbCommentDraft.trim()) return;
      const ref = this.$refs.mbCommentsLive;
      if (!ref) return;
      this.mbCommentPosting = true;
      try {
        const ok = await ref.postCommentExtern(this.mbCommentDraft, 0);
        if (ok) this.mbCommentDraft = "";
      } finally {
        this.mbCommentPosting = false;
      }
    },
    cmtMenuKey(ti, ri = null) {
      return ri == null ? `t-${ti}` : `t-${ti}-r-${ri}`;
    },
    toggleCommentMenu(key, e) {
      if (e) e.stopPropagation();
      this.openCommentMenuKey =
        this.openCommentMenuKey === key ? "" : key;
    },
    closeCommentMenu() {
      this.openCommentMenuKey = "";
    },
    /** 本地 `public/user-profile/level_0～level_6.svg`（与主站同源）；等级 >6 仍用 level_6 */
    levelIconUrl(lv) {
      const n = Math.min(Math.max(parseInt(lv, 10) || 0, 0), 6);
      const base = import.meta.env.BASE_URL;
      return `${base}user-profile/level_${n}.svg`;
    },
    syncTitle() {
      const label =
        this.videoBvidDisplay ||
        (this.aidParam ? String(this.aidParam) : "");
      document.title = buildDocumentTitle(label);
    },
    resetEngagementUi() {
      this.fav.done = false;
      this.fav.animating = false;
      this.fav.pending = false;
      this.coin.done = false;
      this.coin.myAmount = 0;
      this.coin.dailyExpProgress = 0;
      this.coin.animating = false;
      this.coin.pending = false;
      this.wait.done = false;
      this.wait.animating = false;
      this.wait.pending = false;
    },
    onFavEnter() {
      if (this.fav.done) return;
      this.fav.hover = true;
      this.fav.animating = false;
      this.$nextTick(() => {
        if (!this.fav.hover || this.fav.done) return;
        this.fav.animating = true;
      });
    },
    onFavLeave() {
      this.fav.hover = false;
      this.fav.animating = false;
    },
    onCoinEnter() {
      if (this.coin.done) return;
      this.coin.hover = true;
      this.coin.animating = false;
      this.$nextTick(() => {
        if (!this.coin.hover || this.coin.done) return;
        this.coin.animating = true;
      });
    },
    onCoinLeave() {
      this.coin.hover = false;
      this.coin.animating = false;
    },
    onWaitEnter() {
      if (this.wait.done) return;
      this.wait.hover = true;
      this.wait.animating = false;
      this.$nextTick(() => {
        if (!this.wait.hover || this.wait.done) return;
        this.wait.animating = true;
      });
    },
    onWaitLeave() {
      this.wait.hover = false;
      this.wait.animating = false;
    },
    onFavClick() {
      if (!this.isMb || this.mbNumericId == null) return;
      if (!this.mbLoggedIn) {
        this.openMbLoginModal();
        return;
      }
      this.fav.animating = false;
      this.favDialogOpen = true;
    },
    async onFavDialogConfirm(folderIds) {
      if (!this.isMb || this.mbNumericId == null) return;
      if (this.fav.dialogLoading) return;
      this.fav.dialogLoading = true;
      try {
        const res = await mbSetVideoFavoriteFolders(
          this.mbNumericId,
          folderIds
        );
        this.fav.done = !!res.favorited;
        this.stats.fav = String(res.fav_count ?? 0);
        this.favDialogOpen = false;
        if (this.apiDetail) {
          this.apiDetail = {
            ...this.apiDetail,
            favorited_by_me: this.fav.done,
            fav_count: res.fav_count
          };
        }
        ElMessage.success(this.fav.done ? "已添加到收藏夹" : "已取消收藏");
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "收藏操作失败";
        ElMessage.error(String(msg));
      } finally {
        this.fav.dialogLoading = false;
      }
    },
    applyCoinStateFromDetail(d) {
      if (!d || typeof d !== "object") {
        return;
      }
      let amt = Number(d.my_coin_amount);
      if (!Number.isFinite(amt) || amt < 0) {
        amt = d.coined_by_me ? 1 : 0;
      }
      amt = Math.min(2, Math.max(0, Math.floor(amt)));
      this.coin.myAmount = amt;
      this.coin.done = amt >= 2;
      const prog = Number(d.daily_coin_exp_progress);
      this.coin.dailyExpProgress = Number.isFinite(prog) ? Math.max(0, prog) : 0;
      const max = Number(d.daily_coin_exp_max);
      this.coin.dailyExpMax = Number.isFinite(max) && max > 0 ? max : 50;
    },
    onCoinClick() {
      if (!this.isMb || this.mbNumericId == null) return;
      if (!this.mbLoggedIn) {
        this.openMbLoginModal();
        return;
      }
      if (this.coin.myAmount >= 2) {
        ElMessage.info("已为该视频投满 2 枚硬币");
        return;
      }
      this.coinDialogOpen = true;
    },
    async onCoinDialogConfirm(amount) {
      if (!this.isMb || this.mbNumericId == null) return;
      const n = this.coin.myAmount >= 1 ? 1 : amount === 2 ? 2 : 1;
      this.coin.dialogLoading = true;
      try {
        const res = await mbPostVideoCoin(this.mbNumericId, n);
        const myAmt = Number(res.my_coin_amount);
        this.coin.myAmount =
          Number.isFinite(myAmt) && myAmt > 0
            ? Math.min(2, Math.floor(myAmt))
            : Math.min(2, this.coin.myAmount + n);
        this.coin.done = this.coin.myAmount >= 2;
        if (typeof res.daily_coin_exp_progress === "number") {
          this.coin.dailyExpProgress = res.daily_coin_exp_progress;
        }
        this.stats.coin = String(res.coin_count ?? 0);
        if (this.apiDetail) {
          this.apiDetail = {
            ...this.apiDetail,
            coined_by_me: true,
            my_coin_amount: this.coin.myAmount,
            coin_count: res.coin_count,
            daily_coin_exp_progress: this.coin.dailyExpProgress
          };
        }
        this.coinDialogOpen = false;
        ElMessage.success(`已成功投出 ${n} 枚硬币`);
        await this.refreshMinibiliMe();
      } catch (e) {
        ElMessage.error((e && e.message) || "投币失败");
      } finally {
        this.coin.dialogLoading = false;
      }
    },
    clearFollowHintTimer() {
      if (this._followHintTimer) {
        clearTimeout(this._followHintTimer);
        this._followHintTimer = null;
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
    onFollowBtnEnter() {
      if (this.followed && !this.follow.pending) {
        this.follow.hover = true;
      }
    },
    onFollowBtnLeave() {
      this.follow.hover = false;
    },
    onUpHoverFollowChange(payload) {
      if (!payload || !this.isMb) return;
      this.followed = !!payload.followed;
      const fans = Number(payload.follower_count);
      if (Number.isFinite(fans) && fans >= 0) {
        this.upMeta.fans = String(fans);
      }
      if (this.apiDetail) {
        this.apiDetail = {
          ...this.apiDetail,
          followed_by_me: this.followed,
          uploader_follower_count: fans
        };
      }
      invalidateUserHoverProfileCache(this.mbVideoAuthorId);
    },
    onUpMessageClick() {
      if (!this.isMb || this.mbVideoAuthorId == null) return;
      if (!this.mbLoggedIn) {
        this.openMbLoginModal();
        return;
      }
      if (this.mbFollowIsOwnVideo) {
        ElMessage.warning("不能给自己发消息哦~");
        return;
      }
      void this.$router
        .push({
          path: "/minibili/messages",
          query: {
            cat: "my_message",
            peer_id: String(this.mbVideoAuthorId)
          }
        })
        .catch(() => {});
    },
    async onFollowClick() {
      if (!this.isMb) return;
      const authorId = this.mbVideoAuthorId;
      if (authorId == null) return;
      if (this.mbFollowIsOwnVideo) {
        this.showFollowHint("不能关注自己");
        return;
      }
      if (!this.mbLoggedIn) {
        this.openMbLoginModal();
        return;
      }
      if (this.follow.pending) return;
      this.follow.pending = true;
      try {
        const res = await mbToggleUserFollow(authorId);
        this.followed = !!res.followed;
        const fans = Number(res.follower_count);
        if (Number.isFinite(fans) && fans >= 0) {
          this.upMeta.fans = String(fans);
        }
        if (this.apiDetail) {
          this.apiDetail = {
            ...this.apiDetail,
            followed_by_me: this.followed,
            uploader_follower_count: res.follower_count
          };
        }
        invalidateUserHoverProfileCache(authorId);
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
    },
    syncAlsoCarouselMetrics() {
      const el = this.$refs.alsoViewport;
      if (!el) return;
      const gap = 12;
      const n = this.alsoVisibleCount;
      const w = el.clientWidth;
      if (!Number.isFinite(w) || w <= 0) return;
      const itemW = (w - gap * (n - 1)) / n;
      const stage = el.closest(".vd-also-stage");
      const host = stage || el;
      host.style.setProperty("--vd-also-item-w", `${itemW}px`);
      host.style.setProperty("--vd-also-gap", `${gap}px`);
      this.alsoScrollStep = itemW + gap;
      const max = Math.max(0, this.alsoLikedVideos.length - n);
      if (this.alsoCarouselIndex > max) {
        this.alsoCarouselIndex = max;
      }
    },
    alsoScrollPrev() {
      this.alsoCarouselIndex = Math.max(
        0,
        this.alsoCarouselIndex - this.alsoVisibleCount
      );
    },
    alsoScrollNext() {
      this.alsoCarouselIndex = Math.min(
        this.alsoMaxIndex,
        this.alsoCarouselIndex + this.alsoVisibleCount
      );
    },
    patchMbWatchLaterInLists(id, on) {
      const patchList = list => {
        for (let i = 0; i < list.length; i++) {
          if (Number(list[i].id) === id) {
            list.splice(i, 1, { ...list[i], inWatchLater: on });
          }
        }
      };
      patchList(this.relatedVideos);
      patchList(this.alsoLikedVideos);
      if (this.mbNumericId === id) {
        this.wait.done = on;
        if (this.apiDetail) {
          this.apiDetail = { ...this.apiDetail, in_watch_later: on };
        }
      }
    },
    async toggleMbWatchLater(videoId) {
      if (!this.isMb) return null;
      if (!this.mbLoggedIn) {
        this.openMbLoginModal();
        return null;
      }
      const id = Number(videoId);
      if (!Number.isFinite(id) || id <= 0) return null;
      if (this.relatedWatchLaterPending[id]) return null;
      this.relatedWatchLaterPending = {
        ...this.relatedWatchLaterPending,
        [id]: true
      };
      try {
        const res = await mbToggleWatchLater(id);
        const on = !!res.in_watch_later;
        this.patchMbWatchLaterInLists(id, on);
        ElMessage.success(on ? "已加入稍后再看" : "已移出稍后再看");
        return on;
      } catch (e) {
        ElMessage.error((e && e.message) || "稍后再看操作失败");
        return null;
      } finally {
        const next = { ...this.relatedWatchLaterPending };
        delete next[id];
        this.relatedWatchLaterPending = next;
      }
    },
    onRelatedWatchLater(item) {
      void this.toggleMbWatchLater(item && item.id);
    },
    onAlsoWatchLater(item) {
      void this.toggleMbWatchLater(item && item.id);
    },
    async onWaitClick() {
      if (!this.isMb || this.mbNumericId == null) return;
      if (!this.mbLoggedIn) {
        this.openMbLoginModal();
        return;
      }
      if (this.wait.pending) return;
      this.wait.pending = true;
      this.wait.animating = false;
      try {
        const res = await mbToggleWatchLater(this.mbNumericId);
        this.wait.done = !!res.in_watch_later;
        if (this.apiDetail) {
          this.apiDetail = {
            ...this.apiDetail,
            in_watch_later: this.wait.done
          };
        }
        ElMessage.success(
          this.wait.done ? "已加入稍后再看" : "已从稍后再看移除"
        );
      } catch (e) {
        ElMessage.error((e && e.message) || "稍后再看操作失败");
      } finally {
        this.wait.pending = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@use "sass:math";
@use "sass:color";
@import "../../style/mixin";

/* 主站 PC 常见色（与现版稿区一致，略不同于全局 mixin 旧色） */
$bili-blue: #00aeec;
$bili-pink: #fb7299;
$text1: #18191c;
$text2: #9499a0;
$line: #e3e5e7;

/* 雪碧图原始尺寸（与资源文件一致，用于 background-size 缩放） */
$icons-w: 1600px;
$icons-h: 2444px;

/* 评论区 icons-comment.2f36fc5.png（IHDR 1000×1000，坐标与主站一致） */
$icons-comment-w: 1000px;
$icons-comment-h: 1000px;

/* 动画条：尺寸来自 PNG 头解析 */
$anim-fav-w: 1200px;
$anim-fav-h: 80px;
$anim-fav-frames: 20;
$anim-fav-fw: math.div($anim-fav-w, $anim-fav-frames);

$anim-coin-w: 720px;
$anim-coin-h: 92px;
$anim-coin-frames: 9;
$anim-coin-fw: math.div($anim-coin-w, $anim-coin-frames);

$anim-wait-w: 1040px;
$anim-wait-h: 80px;
$anim-wait-frames: 16;
$anim-wait-fw: math.div($anim-wait-w, $anim-wait-frames);

/* 收藏 / 硬币 / 稍后看：动画时长（偏慢更易看清） */
$dur-fav: 2.3s;
$dur-coin: 1s;
$dur-wait: 1.3s;

/* 分享圆标：雪碧图为原始尺寸，坐标与主站一致；圆形窗口仅裁切显示（可调大可放大图标） */
$share-icon-view: 40px;

@mixin share-sprite($x, $y) {
  background-image: url("@/assets/icons.png");
  background-repeat: no-repeat;
  background-size: $icons-w $icons-h;
  background-position: $x $y;
}

.video-page {
  padding-top: 16px;
  padding-bottom: 40px;
  background: #fff;
}

/* 顶栏：左稿件区、右 UP 区（主站布局） */
.video-info-strip {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 32px;
  padding: 12px 0 16px;
  margin-bottom: 12px;
  border-bottom: 1px solid $line;
}

.strip-video {
  flex: 1;
  min-width: 0;
  padding-right: 16px;
}

/* UP 卡片（与主站稿：昵称 + 发消息 / 签名 / 投稿粉丝 / 关注 + 充电） */
.strip-up.up-card {
  display: flex;
  align-items: flex-start;
  flex-shrink: 0;
  width: 318px;
  gap: 16px;
}

.up-face {
  flex-shrink: 0;
}

.up-face-img {
  display: block;
  width: 68px;
  height: 68px;
  border-radius: 50%;
  object-fit: cover;
}

.up-core {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 0;
  min-width: 0;
  flex: 1;
}

.up-row--head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 6px;
}

.up-card .up-name {
  font-size: 15px;
  font-weight: 500;
  color: #00a1d6;
  line-height: 22px;
  text-decoration: none;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  &:hover {
    color: color.adjust(#00a1d6, $lightness: -6%);
  }
}

.up-msg {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  font-size: 12px;
  line-height: 18px;
  color: #999;
  text-decoration: none;
  &:hover {
    color: #666;
  }
}

.up-msg-svg {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.up-bio {
  margin: 0 0 8px;
  font-size: 12px;
  line-height: 18px;
  color: #212121;
}

.up-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 28px;
  margin-bottom: 12px;
  font-size: 12px;
  line-height: 18px;
  color: #999;
}

.up-btns {
  display: flex;
  align-items: stretch;
  gap: 8px;
}

.btn-follow-wrap {
  position: relative;
  flex: 1;
  min-width: 0;
}

.btn-follow-hint {
  position: absolute;
  left: 0;
  bottom: calc(100% + 8px);
  z-index: 5;
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

.btn-follow-main {
  display: block;
  width: 100%;
  min-width: 0;
  height: 32px;
  padding: 0 12px;
  border: none;
  border-radius: 6px;
  background: #00a1d6;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
  &:hover {
    background: color.adjust(#00a1d6, $lightness: -6%);
  }
  &.is-followed {
    background: #e7e7e7;
    color: #61666d;
    &:hover {
      background: #ddd;
    }
  }
  &:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }
}

.btn-charge-main {
  flex: 0 0 auto;
  min-width: 72px;
  height: 32px;
  padding: 0 14px;
  border: 1px solid #fb7299;
  border-radius: 6px;
  background: #fff;
  color: #fb7299;
  font-size: 13px;
  cursor: pointer;
  &:hover {
    background: color.adjust(#fb7299, $lightness: 38%);
  }
}

.video-title {
  font-size: 20px;
  font-weight: 600;
  color: $text1;
  line-height: 28px;
  margin: 0 0 12px;
}

.video-sub-row {
  font-size: 13px;
  color: $text2;
  line-height: 20px;
  margin-bottom: 12px;
  .crumbs a,
  .crumbs .crumb-link {
    color: $bili-blue;
    text-decoration: none;
    &:hover {
      color: color.adjust($bili-blue, $lightness: -4%);
    }
  }
  .slash {
    margin: 0 6px;
    color: #c9ccd0;
  }
  .pub-time {
    margin-left: 16px;
  }
}

.link-report {
  margin-left: 16px;
  color: $bili-blue;
  text-decoration: none;
  &:hover {
    color: color.adjust($bili-blue, $lightness: -4%);
  }
}

.video-stats-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0 24px;
  row-gap: 8px;
  font-size: 13px;
  line-height: 20px;
}

.v-stat {
  display: inline-flex;
  align-items: center;
  white-space: nowrap;
  .v-ico {
    display: inline-block;
    width: 14px;
    height: 14px;
    margin-right: 4px;
    flex-shrink: 0;
    background: url("@/assets/icons.png") no-repeat;
    background-size: $icons-w $icons-h;
    vertical-align: -2px;
  }
  .v-num {
    color: $text2;
  }
  .v-label {
    margin-right: 4px;
  }
}

.v-stat--play .v-ico {
  background-position: -282px -90px;
}
.v-stat--dm .v-ico {
  background-position: -282px -218px;
}
.v-stat--coin {
  color: $bili-blue;
  .v-ico {
    background-position: -282px -410px;
  }
  .v-num {
    color: $bili-blue;
  }
}
.v-stat--fav {
  color: #ff7e29;
  .v-ico {
    background-position: -282px -346px;
  }
  .v-num {
    color: #ff7e29;
  }
}

/* 左侧播放列宽度（播放器 / 工具栏 / 下方稿件扩展区与之对齐，避免整块撑满 1160） */
$video-col-width: 819px;

/* 侧栏不参与文档流高度：绝对定位于左侧栈，高度强制等于播放器+工具栏 */
.video-body-row {
  width: 100%;
}

.video-body-stack {
  position: relative;
  width: $video-col-width;
  max-width: 100%;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.video-side-dock {
  position: absolute;
  left: calc(100% + 16px);
  top: 0;
  width: 325px;
  height: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.video-side-dock > .video-side {
  flex: 1 1 auto;
  min-height: 0;
  width: 100%;
  max-height: 100%;
}

/* 自定义播放器：顶栏 + 16:9 画面 + 弹幕条，高度由内容撑开 */
.player-box-wrap {
  width: 100%;
  border-radius: 2px;
  overflow: hidden;
  border: 1px solid #e2e2e2;
  background: #000;
}

.video-load-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  min-height: 360px;
  padding: 32px 24px;
  color: #fff;
  text-align: center;
  background: #111;
}

.video-load-error__text {
  margin: 0;
  font-size: 16px;
  line-height: 1.6;
}

.video-load-error__link {
  color: #00a1d6;
  text-decoration: none;
  font-size: 14px;
}

.video-load-error__link:hover {
  text-decoration: underline;
}

/* Mini-Bili 评论嵌在原有 vd- 评论区顶栏下，收紧外框避免双重视觉边框 */
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

/* 宽屏：隐藏右侧推荐栏，播放列占满主内容宽度 */
.video-page--wide .video-body-stack {
  width: 100%;
  max-width: 100%;
}

.video-page--wide .video-side-dock {
  display: none;
}

.video-toolbar-wrap {
  width: 100%;
  max-width: 100%;
  margin-top: 0;
  flex-shrink: 0;
}

/* —— 底部操作栏：与播放区等宽，分享左对齐、互动按钮右对齐 —— */
.video-toolbar {
  margin-top: 0;
  width: 100%;
  min-height: 84px;
  padding: 10px 16px 12px;
  box-sizing: border-box;
  border: 1px solid #e5e9ef;
  border-radius: 2px;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: nowrap;
}

.toolbar-share {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
  gap: 8px;
  flex-shrink: 0;
  min-width: 0;
}

.toolbar-ops {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
  gap: 4px;
  flex-shrink: 0;
}

.share-label {
  font-size: 14px;
  color: #222;
}

.share-num {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: #f4f4f4;
  color: $grau;
  font-size: 12px;
  margin-right: 4px;
}

.share-dot {
  display: inline-block;
  width: $share-icon-view;
  height: $share-icon-view;
  border-radius: 50%;
  overflow: hidden;
  cursor: pointer;
  flex-shrink: 0;
  i {
    display: block;
    width: $share-icon-view;
    height: $share-icon-view;
  }
  &.share-feed i {
    @include share-sprite(-1357px, -972px);
  }
  &.share-qzone i {
    @include share-sprite(-1357px, -726px);
  }
  &.share-qq i {
    @include share-sprite(-1357px, -796px);
  }
}

.toolbar-op {
  border: 0;
  background: transparent;
  cursor: pointer;
  padding: 6px 10px;
  text-align: left;
  font-size: 12px;
  color: #505050;
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  min-width: 0;
  flex-shrink: 0;
  box-sizing: border-box;
  &:hover .op-title {
    color: $bili-blue;
  }
}

/* 图标左、文案右，与主站播放页工具栏一致 */
.op-icon-wrap {
  position: relative;
  width: 56px;
  height: 56px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.op-icon-wrap.is-coin {
  height: 56px;
}

.op-icon-wrap.is-wait {
  height: 56px;
}

.op-lines {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 2px;
  min-width: 0;
  min-height: 34px;
}

.op-icon {
  display: block;
  background-repeat: no-repeat;
}

/* 收藏：默认显示 anim-fav 最左侧第一帧（与硬币一致） */
.fav-sprite {
  position: absolute;
  left: 50%;
  top: 50%;
  width: $anim-fav-fw;
  height: $anim-fav-h;
  transform: translate(-50%, -50%) scale(0.72);
  transform-origin: center center;
  background-image: url("@/assets/anim-fav.png");
  background-size: $anim-fav-w $anim-fav-h;
  background-position: 0 0;
}
.fav-sprite.is-done {
  background-position: (-($anim-fav-frames - 1) * $anim-fav-fw) 0;
  transform: translate(-50%, -50%) scale(0.72);
}
.fav-sprite.animating {
  animation: fav-strip $dur-fav steps($anim-fav-frames - 1) infinite;
}

@keyframes fav-strip {
  from {
    background-position: 0 0;
  }
  to {
    background-position: (-($anim-fav-frames - 1) * $anim-fav-fw) 0;
  }
}

/* 硬币：单帧 80×92，缩放纳入视口 */
.coin-sprite {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 80px;
  height: $anim-coin-h;
  transform: translate(-50%, -50%) scale(0.68);
  transform-origin: center center;
  background-image: url("@/assets/anim-coin.png");
  background-size: $anim-coin-w $anim-coin-h;
  background-position: 0 0;
}
.coin-sprite.is-done {
  background-position: (-($anim-coin-frames - 1) * $anim-coin-fw) 0;
  transform: translate(-50%, -50%) scale(0.68);
}
.coin-sprite.animating {
  animation: coin-strip $dur-coin steps($anim-coin-frames - 1) infinite;
}

@keyframes coin-strip {
  from {
    background-position: 0 0;
  }
  to {
    background-position: (-($anim-coin-frames - 1) * $anim-coin-fw) 0;
  }
}

/* 稍后看：默认 anim-wait 最左侧第一帧 */
.wait-sprite {
  position: absolute;
  left: 50%;
  top: 50%;
  width: $anim-wait-fw;
  height: $anim-wait-h;
  transform: translate(-50%, -50%) scale(0.72);
  transform-origin: center center;
  background-image: url("@/assets/anim-wait.png");
  background-size: $anim-wait-w $anim-wait-h;
  background-position: 0 0;
}
.wait-sprite.is-done {
  background-position: (-($anim-wait-frames - 1) * $anim-wait-fw) 0;
  transform: translate(-50%, -50%) scale(0.72);
}
.wait-sprite.animating {
  animation: wait-strip $dur-wait steps($anim-wait-frames - 1) infinite;
}

@keyframes wait-strip {
  from {
    background-position: 0 0;
  }
  to {
    background-position: (-($anim-wait-frames - 1) * $anim-wait-fw) 0;
  }
}

.toolbar-op.is-active .op-title {
  color: $text1;
  font-weight: 600;
}
.toolbar-op .op-title {
  line-height: 16px;
  font-size: 12px;
  color: #505050;
}
.toolbar-op .op-num {
  color: $grau;
  line-height: 16px;
  font-size: 12px;
}
.toolbar-op .op-sub {
  font-size: 12px;
  color: $grau;
  line-height: 16px;
}

.video-side {
  width: 100%;
  min-width: 0;
  min-height: 0;
  padding-left: 0;
  box-sizing: border-box;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.video-side--tall {
  min-height: 0;
}

/* 与下方工具栏卡片视觉统一；推荐列表仅在 .side-scroll 内滚动 */
.side-panel-card {
  border: 1px solid #e5e9ef;
  border-radius: 2px;
  background: #fff;
  box-sizing: border-box;
  padding: 0 10px 8px;
}

.side-scroll {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
  margin-right: -4px;
  padding-right: 4px;
}

/* 弹幕列表 tab：占满侧栏剩余高度，列表区固定撑开（少弹幕时留白在表体内） */
.side-scroll-dm-wrap {
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.side-head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0 10px;
  border-bottom: 1px solid $line;
  font-size: 12px;
  color: $text2;
  line-height: 18px;
}

.side-head-text {
  flex: 1;
  min-width: 0;
}

.side-settings {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  margin: 0;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 4px;
  color: $text2;
  font-size: 16px;
  line-height: 1;
  &:hover {
    background: #f1f2f3;
    color: $text1;
  }
  &::before {
    content: "\2699";
    display: block;
  }
}

.side-tabs {
  flex-shrink: 0;
  display: flex;
  align-items: flex-end;
  width: 100%;
  gap: 0;
  padding: 10px 0 0;
  margin-bottom: 0;
  border-bottom: 1px solid $line;
  font-size: 13px;
}

.side-tabs .tab {
  flex: 1;
  padding: 0 4px 8px;
  margin: 0;
  text-align: center;
  color: $text2;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  box-sizing: border-box;
  &.on {
    color: $bili-blue;
    font-weight: 600;
    border-bottom-color: $bili-blue;
  }
}

.side-block-placeholder {
  padding: 24px 12px;
  font-size: 13px;
  color: $text2;
  text-align: center;
}
.side-block-placeholder__t {
  margin: 0;
}

.related-list {
  list-style: none;
  margin: 0;
  padding: 10px 0 0;
}

.related-item + .related-item {
  margin-top: 12px;
}

.related-link {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  text-decoration: none;
  color: inherit;
  &:hover .related-title {
    color: $bili-blue;
  }
}

.related-thumb-wrap {
  position: relative;
  flex: 0 0 88px;
  width: 88px;
  aspect-ratio: 16 / 10;
  border-radius: 2px;
  overflow: hidden;
  background: #222;
}

.related-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.related-duration {
  position: absolute;
  right: 4px;
  bottom: 4px;
  padding: 1px 5px;
  border-radius: 2px;
  background: rgba(0, 0, 0, 0.65);
  color: #fff;
  font-size: 11px;
  line-height: 16px;
}

.related-watch-later {
  position: absolute;
  right: 4px;
  bottom: 4px;
  z-index: 2;
  margin: 0;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  display: block;
  width: fit-content;
  max-width: calc(100% - 8px);
  overflow: visible;
  opacity: 0;
  visibility: hidden;
  transition: opacity 0.18s ease, visibility 0.18s ease;

  &:disabled {
    cursor: wait;
  }
}

.related-thumb-wrap:hover .related-watch-later,
.related-watch-later:focus-visible {
  opacity: 1;
  visibility: visible;

  &:disabled {
    opacity: 0.75;
  }
}

.related-watch-later-inner {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: fit-content;
}

.related-watch-later-ico-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.58);
  box-sizing: border-box;
}

.related-watch-later--on .related-watch-later-ico-wrap {
  background: rgba(0, 161, 214, 0.88);
}

.related-watch-later-ico {
  width: 14px;
  height: 14px;
  object-fit: contain;
  display: block;
  mix-blend-mode: screen;
  opacity: 0.95;
}

.related-watch-later-txt {
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  margin-top: 4px;
  font-size: 11px;
  line-height: 1.2;
  color: #fff;
  font-weight: 500;
  white-space: nowrap;
  padding: 3px 6px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.58);
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
  transition: opacity 0.16s ease, visibility 0.16s ease;
}

.related-watch-later:hover .related-watch-later-txt,
.related-watch-later:focus-visible .related-watch-later-txt {
  opacity: 1;
  visibility: visible;
}

.related-info {
  flex: 1;
  min-width: 0;
}

.related-title {
  margin: 0 0 6px;
  font-size: 13px;
  color: #222;
  line-height: 18px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.related-meta {
  margin: 0;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 12px;
  color: $text2;
  line-height: 17px;
}

.r-stat {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.r-ico {
  display: inline-block;
  width: 12px;
  height: 12px;
  flex-shrink: 0;
  background: url("@/assets/icons.png") no-repeat;
  background-size: $icons-w $icons-h;
}
.r-ico--play {
  background-position: -282px -90px;
}
.r-ico--dm {
  background-position: -282px -218px;
}

/* ========= 工具栏下方：标签 / 简介 / 猜你喜欢 / 评论 ========= */
.video-below-deck {
  margin-top: 28px;
  width: 100%;
}

.below-main {
  width: 100%;
  max-width: $video-col-width;
  min-width: 0;
}

.vd-tags-block {
  margin-bottom: 16px;
}

.vd-tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.vd-tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 12px;
  height: 26px;
  border-radius: 13px;
  border: 1px solid #e7e7e7;
  background: #fff;
  font-size: 12px;
  color: #61666d;
  text-decoration: none;
  &:hover {
    border-color: $bili-blue;
    color: $bili-blue;
  }
}

.vd-tag--add {
  cursor: pointer;
  width: 26px;
  padding: 0;
  font-size: 16px;
  line-height: 1;
}

.vd-copywarn {
  margin: 0 0 12px;
  font-size: 14px;
  color: #ff6699;
  display: flex;
  align-items: center;
  gap: 6px;
}

.vd-copywarn-ico {
  font-size: 14px;
}

.vd-desc {
  margin: 0 0 22px;
  font-size: 14px;
  line-height: 22px;
  color: #212121;
}

.vd-also {
  margin-bottom: 28px;
  padding-bottom: 22px;
  border-bottom: 1px solid #e3e5e7;
}

.vd-also-title {
  margin: 0 0 14px;
  font-size: 16px;
  font-weight: 600;
  color: #18191c;
}

.vd-also-carousel {
  position: relative;
}

.vd-also-stage {
  position: relative;
}

.vd-also-viewport {
  overflow: hidden;
  width: 100%;
}

.vd-also-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  gap: var(--vd-also-gap, 12px);
  transition: transform 0.28s ease;
  will-change: transform;
}

.vd-also-item {
  flex: 0 0 var(--vd-also-item-w, 154px);
  min-width: 0;
}

.vd-also-nav {
  position: absolute;
  top: 0;
  z-index: 5;
  width: 28px;
  height: calc(var(--vd-also-item-w, 154px) * 10 / 16);
  padding: 0;
  border: none;
  cursor: pointer;
  background: rgba(0, 0, 0, 0.45);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.18s ease;

  &::after {
    content: "";
    display: block;
    width: 8px;
    height: 8px;
    border-top: 2px solid #fff;
    border-right: 2px solid #fff;
  }

  &:hover {
    background: rgba(0, 0, 0, 0.62);
  }
}

.vd-also-nav--prev {
  left: 0;
  border-radius: 4px 0 0 4px;

  &::after {
    transform: rotate(-135deg);
    margin-left: 3px;
  }
}

.vd-also-nav--next {
  right: 0;
  border-radius: 0 4px 4px 0;

  &::after {
    transform: rotate(45deg);
    margin-right: 3px;
  }
}

.vd-also-link {
  text-decoration: none;
  color: inherit;
  display: block;
  &:hover .vd-also-t {
    color: $bili-blue;
  }
}

.vd-also-thumb-wrap {
  position: relative;
  border-radius: 4px;
  overflow: hidden;
  aspect-ratio: 16 / 10;
  background: #222;
  margin-bottom: 8px;
}

.vd-also-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.vd-also-dur {
  position: absolute;
  left: 6px;
  bottom: 6px;
  padding: 2px 6px;
  border-radius: 2px;
  background: rgba(0, 0, 0, 0.65);
  color: #fff;
  font-size: 11px;
  line-height: 14px;
  opacity: 0;
  visibility: hidden;
  transition: opacity 0.18s ease, visibility 0.18s ease;
  pointer-events: none;
}

.vd-also-watch-later {
  position: absolute;
  right: 6px;
  bottom: 6px;
  z-index: 2;
  margin: 0;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  display: block;
  width: fit-content;
  max-width: calc(100% - 12px);
  overflow: visible;
  opacity: 0;
  visibility: hidden;
  transition: opacity 0.18s ease, visibility 0.18s ease;

  &:disabled {
    cursor: wait;
  }
}

.vd-also-watch-later-inner {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: fit-content;
}

.vd-also-watch-later-ico-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.58);
  box-sizing: border-box;
}

.vd-also-watch-later--on .vd-also-watch-later-ico-wrap {
  background: rgba(0, 161, 214, 0.88);
}

.vd-also-watch-later-ico {
  width: 14px;
  height: 14px;
  object-fit: contain;
  display: block;
  mix-blend-mode: screen;
  opacity: 0.95;
}

.vd-also-watch-later-txt {
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  margin-top: 4px;
  font-size: 11px;
  line-height: 1.2;
  color: #fff;
  font-weight: 500;
  white-space: nowrap;
  padding: 3px 6px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.58);
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
  transition: opacity 0.16s ease, visibility 0.16s ease;
}

.vd-also-watch-later:hover .vd-also-watch-later-txt,
.vd-also-watch-later:focus-visible .vd-also-watch-later-txt {
  opacity: 1;
  visibility: visible;
}

.vd-also-badge {
  position: absolute;
  left: 6px;
  bottom: 6px;
  padding: 2px 6px;
  border-radius: 2px;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 11px;
  line-height: 14px;
  opacity: 0;
  visibility: hidden;
  transition: opacity 0.18s ease, visibility 0.18s ease;
  pointer-events: none;
}

.vd-also-thumb-wrap:hover .vd-also-dur,
.vd-also-thumb-wrap:hover .vd-also-watch-later,
.vd-also-watch-later:focus-visible {
  opacity: 1;
  visibility: visible;

  &:disabled {
    opacity: 0.75;
  }
}

.vd-also-thumb-wrap:hover .vd-also-badge {
  opacity: 1;
  visibility: visible;
}

.vd-also-t {
  margin: 0;
  font-size: 13px;
  line-height: 18px;
  color: #18191c;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 评论区（主站 PC：#00a1d6、#f4f5f7 输入底、顶栏仅当前页蓝） */
$vd-cmt-blue: #00a1d6;
$vd-cmt-blue-hover: #0090c2;
$vd-cmt-input-bg: #f4f5f7;
$vd-cmt-line: #e5e9ef;

.vd-comments {
  padding-top: 4px;
}

.vd-cmt-head {
  margin-bottom: 0;
}

.vd-cmt-title {
  margin: 0 0 10px;
  padding-left: 2px;
  font-size: 18px;
  font-weight: 600;
  color: #18191c;
}

.vd-cmt-count {
  font-weight: 600;
  color: #18191c;
}

.vd-cmt-head-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px 20px;
  flex-wrap: wrap;
}

.vd-cmt-head-row--toolbar {
  padding-bottom: 10px;
  border-bottom: 1px solid $vd-cmt-line;
  margin-bottom: 16px;
  flex-wrap: nowrap;
  gap: 12px;
  min-width: 0;
}

.vd-cmt-sort {
  display: flex;
  gap: 22px;
  flex-shrink: 0;
  min-width: 0;
}

.vd-sort-tab {
  position: relative;
  border: none;
  background: none;
  padding: 0 2px 8px;
  font-size: 13px;
  color: #18191c;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  &.on {
    color: $vd-cmt-blue;
    font-weight: 600;
    border-bottom-color: $vd-cmt-blue;
  }
  &.on::after {
    content: "";
    position: absolute;
    left: 50%;
    bottom: -9px;
    margin-left: -4px;
    border-left: 4px solid transparent;
    border-right: 4px solid transparent;
    border-bottom: 4px solid $vd-cmt-blue;
    transform: translateY(-1px);
  }
  &:hover:not(.on) {
    color: #61666d;
  }
}

.vd-cmt-page-top {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
  gap: 2px 4px;
  font-size: 12px;
  color: #9499a0;
  flex-shrink: 0;
  margin-left: auto;
  justify-content: flex-end;
  min-width: 0;
}

/* 顶栏分页：纯文字链式；勿复用 .vd-page-num 盒式背景（类名叠加会盖住数字） */
.vd-cmt-page-top--links {
  .vd-page-info {
    display: inline;
    margin-right: 8px;
    padding: 0;
    border: none;
    background: transparent;
    color: #9499a0;
    font-size: 12px;
    font-weight: 400;
  }
  button.vd-page-num.vd-page-num--link {
    appearance: none;
    -webkit-appearance: none;
    margin: 0;
    min-width: 0;
    height: auto;
    padding: 0 5px;
    border: none;
    border-radius: 0;
    box-shadow: none;
    background: transparent;
    background-image: none;
    font-size: 12px;
    font-weight: 400;
    line-height: 1.5;
    color: #18191c;
    cursor: pointer;
    display: inline;
    align-items: unset;
    justify-content: unset;
    &:hover:not(:disabled):not(.on) {
      color: $vd-cmt-blue;
    }
    &.on {
      background: transparent;
      background-image: none;
      border: none;
      box-shadow: none;
      color: $vd-cmt-blue;
      font-weight: 600;
    }
  }
  button.vd-page-next.vd-page-next--link {
    appearance: none;
    -webkit-appearance: none;
    margin: 0;
    height: auto;
    padding: 0 0 0 6px;
    border: none;
    border-radius: 0;
    box-shadow: none;
    background: transparent;
    background-image: none;
    font-size: 12px;
    font-weight: 400;
    line-height: 1.5;
    color: #18191c;
    cursor: pointer;
    display: inline;
    align-items: unset;
    justify-content: unset;
    &:hover:not(:disabled) {
      color: $vd-cmt-blue;
    }
    &:disabled {
      background: transparent;
      color: #c0c4cc;
      cursor: not-allowed;
    }
  }
  .vd-page-ellipsis {
    color: #9499a0;
    padding: 0 4px;
  }
}

.vd-page-info {
  margin-right: 0;
}

.vd-page-num {
  box-sizing: border-box;
  appearance: none;
  -webkit-appearance: none;
  margin: 0;
  min-width: 28px;
  height: 26px;
  padding: 0 6px;
  border: 1px solid $vd-cmt-line;
  border-radius: 4px;
  background: #fff;
  background-image: none;
  font-size: 12px;
  font-weight: 400;
  line-height: 1;
  color: #18191c;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  &.on {
    background: $vd-cmt-blue;
    background-image: none;
    border-color: $vd-cmt-blue;
    color: #fff;
  }
  &:hover:not(.on) {
    border-color: #c9ccd0;
    color: #18191c;
  }
}

.vd-page-num--boxed {
  min-width: 28px;
  height: 26px;
}

.vd-page-ellipsis {
  padding: 0 4px;
  color: #9499a0;
}

.vd-page-next {
  box-sizing: border-box;
  appearance: none;
  -webkit-appearance: none;
  margin: 0;
  height: 26px;
  padding: 0 10px;
  border: 1px solid $vd-cmt-line;
  border-radius: 4px;
  background: #fff;
  background-image: none;
  font-size: 12px;
  font-weight: 400;
  line-height: 1;
  color: #18191c;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  &:hover:not(:disabled) {
    border-color: #c9ccd0;
    color: #18191c;
  }
  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.vd-page-next--boxed {
  height: 26px;
}

.vd-cmt-page-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px 24px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid $vd-cmt-line;
  font-size: 12px;
}

.vd-cmt-page-bottom-left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.vd-cmt-page-bottom-right {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.vd-page-bottom-meta {
  color: #9499a0;
}

.vd-page-jump-input {
  width: 42px;
  height: 24px;
  padding: 0 6px;
  border: 1px solid $vd-cmt-line;
  border-radius: 2px;
  font-size: 12px;
  text-align: center;
  box-sizing: border-box;
  background: #fff;
  color: #18191c;
}

.vd-cmt-page-bottom--mb {
  margin-top: 12px;
  flex-wrap: nowrap;
  gap: 12px 16px;
  min-width: 0;
}

.vd-cmt-page-bottom--mb .vd-cmt-page-bottom-left {
  flex: 1 1 auto;
  min-width: 0;
}

.vd-cmt-page-bottom--mb .vd-cmt-page-bottom-right {
  flex: 0 0 auto;
  flex-shrink: 0;
}

.vd-cmt-composer {
  display: flex;
  gap: 12px;
  align-items: stretch;
  margin-bottom: 18px;
  padding-bottom: 18px;
  border-bottom: 1px solid $vd-cmt-line;
}

.vd-cmt-composer--mb {
  gap: 10px;
  align-items: center;
}

.vd-cmt-mb-closed-bar {
  margin: 0 0 8px;
  padding: 12px 14px;
  border-radius: 8px;
  background: #f6f7f8;
  text-align: center;
  font-size: 14px;
  line-height: 1.5;
  color: #9499a0;
}

.vd-cmt-mb-closed-foot {
  margin: 0;
  padding: 8px 0 4px;
  text-align: center;
  font-size: 13px;
  color: #99a2aa;
}

.vd-cmt-avatar {
  border-radius: 50%;
  flex-shrink: 0;
  object-fit: cover;
  align-self: flex-start;
}

.vd-cmt-avatar--mb {
  align-self: center;
}

.vd-cmt-mb-composer-main {
  flex: 1;
  min-width: 0;
}

.vd-cmt-mb-editor-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 76px;
  column-gap: 10px;
  align-items: stretch;
}

.vd-cmt-uni-inbox {
  grid-column: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid rgba(22, 24, 35, 0.1);
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.vd-cmt-uni-inbox:focus-within {
  border-color: $vd-cmt-blue;
  box-shadow: 0 0 0 3px rgba(0, 161, 214, 0.14);
}

.vd-cmt-uni-inbox__field {
  display: block;
  width: 100%;
  box-sizing: border-box;
  margin: 0;
  border: none;
  resize: none;
  padding: 12px 14px;
  min-height: 72px;
  font-size: 15px;
  line-height: 1.55;
  color: #18191c;
  background: transparent;
  outline: none;
  &::placeholder {
    color: #9499a0;
  }
}

.vd-cmt-uni-inbox__guest.vd-cmt-login-hint {
  min-height: 72px;
  background: transparent;
}

.vd-cmt-uni-inbox__bar {
  display: flex;
  align-items: center;
  padding: 6px 10px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  background: #f8f9fb;
}

.vd-cmt-uni-emoji {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 4px 8px;
  border: none;
  border-radius: 8px;
  background: transparent;
  font-size: 13px;
  color: #64748b;
  cursor: pointer;
  transition:
    color 0.15s ease,
    background 0.15s ease;
}

.vd-cmt-uni-emoji:hover:not(:disabled) {
  color: $vd-cmt-blue;
  background: rgba(0, 161, 214, 0.08);
}

.vd-cmt-uni-emoji:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.vd-cmt-mb-editor-row .vd-cmt-submit--mb {
  grid-column: 2;
  grid-row: 1;
  align-self: stretch;
  justify-self: stretch;
  width: auto;
  min-width: 0;
  max-width: none;
}

.vd-cmt-input-column {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.vd-cmt-input-shell {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 88px;
  border: 1px solid $vd-cmt-line;
  border-radius: 4px;
  overflow: hidden;
  background: #fff;
  box-sizing: border-box;
}

.vd-cmt-textarea {
  display: block;
  width: 100%;
  flex: 1;
  min-height: 56px;
  border: none;
  resize: none;
  padding: 10px 12px;
  font-size: 14px;
  line-height: 22px;
  color: #18191c;
  background: #fff;
  box-sizing: border-box;
  &::placeholder {
    color: #9499a0;
  }
}

.vd-cmt-login-hint {
  flex: 1;
  min-height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 0 2px;
  padding: 10px 12px;
  font-size: 14px;
  line-height: 22px;
  box-sizing: border-box;
  text-align: center;
}

.vd-cmt-login-hint-muted {
  color: #9499a0;
}

.vd-cmt-login-hint-btn {
  display: inline-block;
  flex-shrink: 0;
  margin: 0 4px;
  padding: 2px 12px;
  border: none;
  border-radius: 4px;
  background: $vd-cmt-blue;
  color: #fff !important;
  font-size: 13px;
  font-weight: 500;
  font-family: inherit;
  line-height: 20px;
  cursor: pointer;
  vertical-align: baseline;
  &:hover {
    background: $vd-cmt-blue-hover;
    color: #fff !important;
  }
}

.vd-cmt-input-foot {
  flex-shrink: 0;
  padding: 6px 12px 10px;
  background: #fff;
}

.vd-emoji-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  margin: 0;
  border: none;
  border-radius: 0;
  background: none;
  font-size: 12px;
  color: #9499a0;
  cursor: pointer;
  &:hover:not(:disabled) {
    color: $vd-cmt-blue;
  }
  &:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
}

.vd-emoji-ico {
  display: inline-block;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  vertical-align: middle;
  background-image: url("@/assets/icons-comment.2f36fc5.png");
  background-repeat: no-repeat;
  background-size: $icons-comment-w $icons-comment-h;
  background-position: -408px -24px;
}

.vd-cmt-submit {
  flex-shrink: 0;
  align-self: stretch;
  min-width: 96px;
  padding: 0 16px;
  border: none;
  border-radius: 8px;
  background: $bili-blue;
  color: #fff;
  font-size: 14px;
  line-height: 1.35;
  cursor: pointer;
  &:hover:not(:disabled) {
    background: color.adjust($bili-blue, $lightness: -5%);
  }
  &:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }
}

.vd-cmt-submit--mb {
  min-width: 72px;
  max-width: 76px;
  padding: 8px 10px;
  border-radius: 10px;
  background: $vd-cmt-blue;
  font-size: 13px;
  font-weight: 500;
  line-height: 1.35;
  white-space: normal;
  &:hover:not(:disabled):not(.is-guest) {
    background: $vd-cmt-blue-hover;
  }
  &:disabled {
    background: #e3e5e7;
    color: #fff;
    opacity: 1;
    cursor: not-allowed;
  }
  &.is-guest {
    background: #e3e5e7;
    color: #fff;
    cursor: pointer;
    &:hover {
      background: #d7d9dc;
    }
  }
}

.vd-cmt-submit-lines {
  display: inline-block;
  text-align: center;
  line-height: 1.3;
}

@import "../../styles/vd-comment-list.scss";
</style>