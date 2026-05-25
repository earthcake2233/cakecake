<template>
  <div class="mb-dyn-page">
    <div class="mb-dyn-bg" :style="dynBgLayerStyle" aria-hidden="true" />
    <div class="mb-dyn-root">
      <div v-if="!token" class="bili-wrapper mb-dyn-wrap">
        <div class="mb-dyn-card mb-dyn-login">
          <p>
            请先
            <a href="#" class="mb-dyn-login__link" @click.prevent="openLoginModal">登录</a>
            后查看关注动态。
          </p>
        </div>
      </div>

      <div v-else class="bili-wrapper mb-dyn-wrap">
        <div class="mb-dyn-grid">
          <!-- 左：个人卡片 -->
          <aside class="mb-dyn-left" aria-label="我的信息">
            <section class="mb-dyn-card mb-dyn-profile">
              <div class="mb-dyn-profile__head">
                <router-link
                  v-if="mySpaceRoute"
                  class="mb-dyn-profile__avatar-link"
                  :to="mySpaceRoute"
                >
                  <img
                    class="mb-dyn-profile__avatar"
                    :src="profileAvatar"
                    width="48"
                    height="48"
                    alt=""
                  />
                </router-link>
                <img
                  v-else
                  class="mb-dyn-profile__avatar"
                  :src="profileAvatar"
                  width="48"
                  height="48"
                  alt=""
                />
                <div class="mb-dyn-profile__info">
                  <router-link
                    v-if="mySpaceRoute"
                    class="mb-dyn-profile__name"
                    :to="mySpaceRoute"
                  >
                    {{ profileName }}
                  </router-link>
                  <span v-else class="mb-dyn-profile__name">{{ profileName }}</span>
                  <img
                    class="mb-dyn-profile__level"
                    :src="levelIconUrl(levelDisplay)"
                    width="28"
                    height="28"
                    alt=""
                    :title="'LV' + levelDisplay"
                  />
                </div>
              </div>
              <div class="mb-dyn-profile__stats">
                <router-link
                  v-if="relationsFollowingRoute"
                  class="mb-dyn-profile__stat"
                  :to="relationsFollowingRoute"
                >
                  <b>{{ statFollowing }}</b>
                  <span>关注</span>
                </router-link>
                <span v-else class="mb-dyn-profile__stat">
                  <b>{{ statFollowing }}</b>
                  <span>关注</span>
                </span>
                <router-link
                  v-if="relationsFollowersRoute"
                  class="mb-dyn-profile__stat"
                  :to="relationsFollowersRoute"
                >
                  <b>{{ statFollowers }}</b>
                  <span>粉丝</span>
                </router-link>
                <span v-else class="mb-dyn-profile__stat">
                  <b>{{ statFollowers }}</b>
                  <span>粉丝</span>
                </span>
                <router-link
                  v-if="mySpaceDynamicRoute"
                  class="mb-dyn-profile__stat"
                  :to="mySpaceDynamicRoute"
                >
                  <b>{{ statDynamics }}</b>
                  <span>动态</span>
                </router-link>
                <span v-else class="mb-dyn-profile__stat">
                  <b>{{ statDynamics }}</b>
                  <span>动态</span>
                </span>
              </div>
            </section>
          </aside>

          <!-- 中：主 feed -->
          <main class="mb-dyn-center">
            <section class="mb-dyn-card mb-dyn-editor" aria-label="发布动态">
              <div class="mb-dyn-editor__compose">
                <input
                  v-model="draftTitle"
                  type="text"
                  class="mb-dyn-editor__title"
                  maxlength="20"
                  placeholder="好的标题更容易获得支持，选填20字"
                />
                <textarea
                  v-model="draftContent"
                  class="mb-dyn-editor__content"
                  rows="3"
                  maxlength="233"
                  placeholder="有什么想和大家分享的？"
                />
              </div>
              <div v-if="draftImageMode" class="mb-dyn-editor__media">
                <div class="mb-dyn-editor__media-grid">
                  <div
                    v-for="(item, ix) in draftImagePreviews"
                    :key="'draft-img-' + ix"
                    class="mb-dyn-editor__media-item"
                  >
                    <img :src="item.url" alt="" />
                    <button
                      type="button"
                      class="mb-dyn-editor__media-remove"
                      aria-label="移除图片"
                      @click="removeDraftImage(ix)"
                    >
                      ×
                    </button>
                  </div>
                  <button
                    v-if="draftImageFiles.length < maxDraftImages"
                    type="button"
                    class="mb-dyn-editor__media-add"
                    aria-label="添加图片"
                    @click="openDraftImagePicker"
                  >
                    <span class="mb-dyn-editor__media-add-plus">+</span>
                  </button>
                </div>
              </div>
              <input
                ref="draftImageInput"
                type="file"
                accept="image/jpeg,image/png,image/webp,image/gif"
                multiple
                class="mb-dyn-editor__file-input"
                @change="onDraftImagesSelected"
              />
              <div class="mb-dyn-editor__bar">
                <div class="mb-dyn-editor__tools" aria-label="插入内容">
                  <button type="button" class="mb-dyn-editor__tool" title="表情" aria-label="表情">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="1.5" />
                      <path d="M9 10h.01M15 10h.01M8.5 14.5c1.2 1.5 6.3 1.5 7.5 0" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
                    </svg>
                  </button>
                  <button
                    type="button"
                    class="mb-dyn-editor__tool"
                    :class="{ 'is-on': draftImageMode }"
                    title="图片"
                    aria-label="图片"
                    @click="onDraftImageToolClick"
                  >
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <rect x="4" y="5" width="16" height="14" rx="2" stroke="currentColor" stroke-width="1.5" />
                      <circle cx="9" cy="10" r="1.5" fill="currentColor" />
                      <path d="M6 17l4.5-4 3 2.5L18 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                    </svg>
                  </button>
                </div>
                <div class="mb-dyn-editor__right">
                  <span class="mb-dyn-editor__count">{{ draftCharCount }}/233</span>
                  <button
                    type="button"
                    class="mb-dyn-editor__submit"
                    :disabled="!canPublishDynamic || publishSubmitting"
                    @click="onPublishDynamic"
                  >
                    {{ publishSubmitting ? "发布中…" : "发布" }}
                  </button>
                </div>
              </div>
            </section>

            <section
              v-if="followStrip.length"
              class="mb-dyn-card mb-dyn-follow-strip"
              aria-label="关注的人"
            >
              <div class="mb-dyn-follow-scroll">
                <button
                  v-for="item in followStrip"
                  :key="'fs-' + item.id"
                  type="button"
                  class="mb-dyn-follow-item"
                  :class="{ 'is-on': activeFollowId === item.id }"
                  @click="activeFollowId = item.id"
                >
                  <div
                    class="mb-dyn-follow-avatar-wrap"
                    :class="{ 'is-on': activeFollowId === item.id }"
                  >
                    <span
                      v-if="item.isAll"
                      class="mb-dyn-follow-avatar mb-dyn-follow-avatar--all"
                    >
                      <img
                        class="mb-dyn-follow-windmill"
                        :src="windmillIcon"
                        width="22"
                        height="22"
                        alt=""
                      />
                    </span>
                    <img
                      v-else
                      class="mb-dyn-follow-avatar"
                      :src="item.avatar || akari"
                      :alt="item.label"
                    />
                  </div>
                  <span class="mb-dyn-follow-name">{{ item.label }}</span>
                </button>
              </div>
            </section>

            <nav class="mb-dyn-tabs" aria-label="动态分类">
              <button
                v-for="tab in feedTabs"
                :key="tab.key"
                type="button"
                class="mb-dyn-tab"
                :class="{ 'is-on': activeTab === tab.key }"
                @click="activeTab = tab.key"
              >
                {{ tab.label }}
              </button>
            </nav>

            <div v-if="loading" class="mb-dyn-card mb-dyn-loading" role="status">
              加载中…
            </div>
            <div v-else-if="!visibleFeed.length" class="mb-dyn-card mb-dyn-empty">
              暂无动态，关注更多 UP 主或发布稿件后会出现在这里。
            </div>
            <div v-else class="mb-dyn-feed" aria-label="动态列表">
              <article
                v-for="row in visibleFeed"
                :key="row.id"
                class="mb-space__dyn-card"
              >
                <header class="mb-space__dyn-head">
                  <router-link
                    v-if="userSpaceDynamicRoute(row.userId)"
                    class="mb-dyn-post__avatar-link"
                    :to="userSpaceDynamicRoute(row.userId)"
                  >
                    <img
                      class="mb-space__dyn-avatar"
                      :src="row.avatar || akari"
                      width="48"
                      height="48"
                      alt=""
                    />
                  </router-link>
                  <img
                    v-else
                    class="mb-space__dyn-avatar"
                    :src="row.avatar || akari"
                    width="48"
                    height="48"
                    alt=""
                  />
                  <div class="mb-space__dyn-head-main">
                    <router-link
                      v-if="userSpaceDynamicRoute(row.userId)"
                      class="mb-dyn-post__name-link"
                      :to="userSpaceDynamicRoute(row.userId)"
                    >
                      <div class="mb-space__dyn-name">{{ row.nickname }}</div>
                    </router-link>
                    <div v-else class="mb-space__dyn-name">{{ row.nickname }}</div>
                    <div class="mb-space__dyn-subline">
                      <span class="mb-space__dyn-date">{{
                        formatDynDateCN(row.ts)
                      }}</span>
                      <template v-if="row.verb">
                        <span class="mb-space__dyn-dot" aria-hidden="true">·</span>
                        <span class="mb-space__dyn-verb">{{ row.verb }}</span>
                      </template>
                    </div>
                  </div>
                  <div v-if="isDynRowOwner(row)" class="mb-space__dyn-head-tools">
                    <div class="mb-space__dyn-more-wrap">
                      <button
                        type="button"
                        class="mb-space__dyn-more"
                        aria-haspopup="true"
                        aria-label="更多"
                      >
                        ⋮
                      </button>
                      <div class="mb-space__dyn-more-menu" role="menu">
                        <button
                          type="button"
                          class="mb-space__dyn-more-item mb-space__dyn-more-item--del"
                          role="menuitem"
                          @click.stop="openDynDeleteDialog(row)"
                        >
                          删除
                        </button>
                      </div>
                    </div>
                  </div>
                </header>


                <template v-if="row.kind === 'image' && row.post">
                  <router-link
                    v-if="minibiliDynamicReadRoute(row.post.id)"
                    :to="minibiliDynamicReadRoute(row.post.id)"
                    class="mb-space__dyn-body mb-space__dyn-body--link"
                  >
                    <div class="mb-space__dyn-img-block">
                      <p
                        v-if="row.post.title"
                        class="mb-space__dyn-textline mb-space__dyn-textline--title"
                      >
                        {{ row.post.title }}
                      </p>
                      <p
                        v-if="row.post.content"
                        class="mb-space__dyn-textline"
                      >
                        {{ row.post.content }}
                      </p>
                      <div
                        v-if="row.post.images && row.post.images.length"
                        class="mb-space__dyn-img-row"
                      >
                        <div
                          v-for="(im, ix) in row.post.images"
                          :key="'dyn-im-' + ix"
                          class="mb-space__dyn-img-cell"
                        >
                          <img class="mb-space__dyn-img" :src="im" alt="" />
                        </div>
                      </div>
                    </div>
                  </router-link>
                  <div v-else class="mb-space__dyn-body">
                    <div class="mb-space__dyn-img-block">
                      <p
                        v-if="row.post.title"
                        class="mb-space__dyn-textline mb-space__dyn-textline--title"
                      >
                        {{ row.post.title }}
                      </p>
                      <p
                        v-if="row.post.content"
                        class="mb-space__dyn-textline"
                      >
                        {{ row.post.content }}
                      </p>
                      <div
                        v-if="row.post.images && row.post.images.length"
                        class="mb-space__dyn-img-row"
                      >
                        <div
                          v-for="(im, ix) in row.post.images"
                          :key="'dyn-im-fb-' + ix"
                          class="mb-space__dyn-img-cell"
                        >
                          <img class="mb-space__dyn-img" :src="im" alt="" />
                        </div>
                      </div>
                    </div>
                  </div>
                  <div class="mb-space__dyn-act-bar" aria-label="动态操作">
                    <button
                      type="button"
                      class="mb-space__dyn-act-bar__btn"
                      title="转发"
                      @click.stop
                    >
                      <img
                        class="mb-space__dyn-ico-act__share"
                        :src="shareIco"
                        alt=""
                      />
                      <span class="mb-space__dyn-act-bar__txt">转发</span>
                    </button>
                    <button
                      type="button"
                      class="mb-space__dyn-act-bar__btn"
                      :class="{
                        'is-on':
                          dynCommentDynamicId === Number(row.post && row.post.id)
                      }"
                      title="评论"
                      @click.stop="
                        toggleDynCommentPanel(row.post.id, 'dynamic')
                      "
                    >
                      <svg
                        class="mb-space__dyn-ico-act__svg"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="1.5"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        aria-hidden="true"
                      >
                        <path
                          d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"
                        />
                      </svg>
                      <span class="mb-space__dyn-act-bar__txt"
                        >评论
                        <span class="mb-space__dyn-act-bar__num">{{
                          formatCount(row.stats.comment || 0)
                        }}</span></span
                      >
                    </button>
                    <button
                      type="button"
                      class="mb-space__dyn-act-bar__btn"
                      :class="{
                        'mb-space__dyn-ico-act--liked': row.liked_by_me
                      }"
                      title="点赞"
                      @click.stop="onDynImageLike(row)"
                    >
                      <svg
                        v-if="!row.liked_by_me"
                        class="mb-space__dyn-ico-act__svg mb-space__dyn-ico-act__svg--like-outline"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="1.5"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        aria-hidden="true"
                      >
                        <path
                          d="M7 10v12M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.67 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2a3.13 3.13 0 0 1 3 3.88Z"
                        />
                      </svg>
                      <svg
                        v-else
                        class="mb-space__dyn-ico-act__svg mb-space__dyn-ico-act__svg--like-solid"
                        viewBox="0 0 24 24"
                        fill="currentColor"
                        aria-hidden="true"
                      >
                        <path
                          d="M1 21h4V9H1v12zm22-11c0-1.1-.9-2-2-2h-6.31l.95-4.57.03-.32c0-.41-.17-.79-.44-1.06L14.17 1 7.59 7.59C7.22 7.95 7 8.45 7 9v10c0 1.1.9 2 2 2h9c.83 0 1.54-.5 1.84-1.22l3.02-7.05c.09-.23.14-.47.14-.73v-2z"
                        />
                      </svg>
                      <span class="mb-space__dyn-act-bar__txt"
                        >点赞
                        <span class="mb-space__dyn-act-bar__num">{{
                          formatCount(row.stats.like || 0)
                        }}</span></span
                      >
                    </button>
                  </div>
                  <MbDynVideoCommentPanel
                    v-if="
                      row.post &&
                      dynCommentDynamicId === Number(row.post.id)
                    "
                    ref="dynDynamicCommentPanel"
                    :dynamic="dynImageCommentPayload(row)"
                    :dynamic-author-id="Number(row.userId)"
                    :owner-can-curate="isDynRowOwner(row)"
                    @patch-dynamic="onDynCommentPatchDynamic"
                    @counts="onDynDynamicCommentsLiveCounts"
                    @manage-dialog="onDynPanelManageDialog"
                  />
                </template>

                <template v-else-if="row.kind === 'text'">
                  <p
                    v-for="(line, ix) in row.lines"
                    :key="'ln-' + ix"
                    class="mb-space__dyn-textline"
                  >
                    {{ line }}
                  </p>
                  <footer class="mb-space__dyn-foot">
                    <span class="mb-space__dyn-foot-i"
                      ><i class="mb-space__dyn-ico mb-space__dyn-ico--fwd" />转发
                      {{ row.stats.forward }}</span
                    >
                    <span class="mb-space__dyn-foot-i"
                      ><i class="mb-space__dyn-ico mb-space__dyn-ico--cmt" />评论
                      {{ row.stats.comment }}</span
                    >
                    <span class="mb-space__dyn-foot-i"
                      ><i class="mb-space__dyn-ico mb-space__dyn-ico--like" />点赞
                      {{ row.stats.like }}</span
                    >
                  </footer>
                </template>

                <template v-else-if="row.kind === 'video' && row.video">
                  <div class="mb-space__dyn-video-info">
                    <router-link
                      class="mb-space__dyn-vbox"
                      :to="minibiliVideoPlayRoute(row.video.id)"
                    >
                      <div class="mb-space__dyn-vbox-l">
                        <img
                          class="mb-space__dyn-vcover"
                          :src="row.video.cover_url || akari"
                          :alt="row.video.title"
                        />
                        <span class="mb-space__dyn-vdur">{{
                          formatDuration(row.video.duration)
                        }}</span>
                        <span class="mb-space__dyn-vplay" aria-hidden="true" />
                        <button
                          type="button"
                          class="mb-space__vthumb-later"
                          aria-label="稍后再看"
                          @click.stop.prevent="onWatchLaterStub($event)"
                        >
                          <span class="mb-space__vthumb-later-inner">
                            <span class="mb-space__vlater-ico-wrap">
                              <img
                                class="mb-space__vlater-ico"
                                :src="thumbLaterIco"
                                alt=""
                              />
                            </span>
                            <span class="mb-space__vlater-txt">稍后再看</span>
                          </span>
                        </button>
                      </div>
                      <div class="mb-space__dyn-vbox-r">
                        <div class="mb-space__dyn-vbox-title">
                          {{ row.video.title }}
                        </div>
                        <div class="mb-space__dyn-vbox-desc">
                          {{ videoDescriptionText(row.video) }}
                        </div>
                        <div class="mb-space__dyn-vstats">
                          <span class="mb-space__dyn-vstat"
                            ><img
                              class="mb-space__dyn-vstat-ico"
                              :src="thumbPlayIco"
                              alt=""
                            />
                            {{ formatCount(row.video.play_count) }}</span
                          >
                          <span class="mb-space__dyn-vstat"
                            ><img
                              class="mb-space__dyn-vstat-ico"
                              :src="thumbDanmuIco"
                              alt=""
                            />
                            {{
                              formatCount(row.video.danmaku_count || 0)
                            }}</span
                          >
                        </div>
                      </div>
                    </router-link>
                  </div>
                  <div class="mb-space__dyn-act-bar" aria-label="动态操作">
                    <button
                      type="button"
                      class="mb-space__dyn-act-bar__btn"
                      title="转发"
                      @click.stop
                    >
                      <img
                        class="mb-space__dyn-ico-act__share"
                        :src="shareIco"
                        alt=""
                      />
                      <span class="mb-space__dyn-act-bar__txt">转发</span>
                    </button>
                    <button
                      type="button"
                      class="mb-space__dyn-act-bar__btn"
                      :class="{ 'is-on': dynCommentVideoId === row.video.id }"
                      title="评论"
                      @click.stop="toggleDynCommentPanel(row.video.id)"
                    >
                      <svg
                        class="mb-space__dyn-ico-act__svg"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="1.5"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        aria-hidden="true"
                      >
                        <path
                          d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"
                        />
                      </svg>
                      <span class="mb-space__dyn-act-bar__txt"
                        >评论
                        <span class="mb-space__dyn-act-bar__num">{{
                          formatCount(row.video.comment_count || 0)
                        }}</span></span
                      >
                    </button>
                    <button
                      type="button"
                      class="mb-space__dyn-act-bar__btn"
                      :class="{
                        'mb-space__dyn-ico-act--liked': row.video.liked_by_me
                      }"
                      title="点赞"
                      @click.stop="onDynVideoLike(row)"
                    >
                      <svg
                        v-if="!row.video.liked_by_me"
                        class="mb-space__dyn-ico-act__svg mb-space__dyn-ico-act__svg--like-outline"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="1.5"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        aria-hidden="true"
                      >
                        <path
                          d="M7 10v12M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.67 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2a3.13 3.13 0 0 1 3 3.88Z"
                        />
                      </svg>
                      <svg
                        v-else
                        class="mb-space__dyn-ico-act__svg mb-space__dyn-ico-act__svg--like-solid"
                        viewBox="0 0 24 24"
                        fill="currentColor"
                        aria-hidden="true"
                      >
                        <path
                          d="M1 21h4V9H1v12zm22-11c0-1.1-.9-2-2-2h-6.31l.95-4.57.03-.32c0-.41-.17-.79-.44-1.06L14.17 1 7.59 7.59C7.22 7.95 7 8.45 7 9v10c0 1.1.9 2 2 2h9c.83 0 1.54-.5 1.84-1.22l3.02-7.05c.09-.23.14-.47.14-.73v-2z"
                        />
                      </svg>
                      <span class="mb-space__dyn-act-bar__txt"
                        >点赞
                        <span class="mb-space__dyn-act-bar__num">{{
                          formatCount(row.video.like_count || 0)
                        }}</span></span
                      >
                    </button>
                  </div>
                  <MbDynVideoCommentPanel
                    v-if="dynCommentVideoId === row.video.id"
                    :video="row.video"
                    :video-author-id="Number(row.userId)"
                    :owner-can-curate="isDynRowOwner(row)"
                    @patch-video="onDynCommentPatchVideo"
                    @counts="onDynCommentsLiveCounts"
                    @manage-dialog="onDynPanelManageDialog"
                  />
                </template>

                <template v-else-if="row.kind === 'article' && row.article">
                  <div class="mb-space__dyn-video-info">
                    <router-link
                      v-if="minibiliArticleReadRoute(row.article.id)"
                      class="mb-space__dyn-vbox mb-space__dyn-vbox--article"
                      :to="minibiliArticleReadRoute(row.article.id)"
                    >
                      <div class="mb-space__dyn-vbox-l">
                        <img
                          class="mb-space__dyn-vcover"
                          :src="row.article.cover_url || akari"
                          :alt="row.article.title"
                        />
                      </div>
                      <div class="mb-space__dyn-vbox-r">
                        <div class="mb-space__dyn-vbox-title">
                          {{ row.article.title }}
                        </div>
                        <div class="mb-space__dyn-vbox-desc">
                          {{ articleDescriptionText(row.article) }}
                        </div>
                        <div class="mb-space__dyn-vstats">
                          <span class="mb-space__dyn-vstat"
                            ><img
                              class="mb-space__dyn-vstat-ico"
                              :src="thumbPlayIco"
                              alt=""
                            />
                            {{ formatCount(row.article.view_count) }}</span
                          >
                          <span class="mb-space__dyn-vstat"
                            ><img
                              class="mb-space__dyn-vstat-ico"
                              :src="thumbDanmuIco"
                              alt=""
                            />
                            {{
                              formatCount(row.article.comment_count || 0)
                            }}</span
                          >
                        </div>
                      </div>
                    </router-link>
                    <a
                      v-else
                      class="mb-space__dyn-vbox mb-space__dyn-vbox--article"
                      href="#"
                      @click.prevent
                    >
                      <div class="mb-space__dyn-vbox-l">
                        <img
                          class="mb-space__dyn-vcover"
                          :src="row.article.cover_url || akari"
                          :alt="row.article.title"
                        />
                      </div>
                      <div class="mb-space__dyn-vbox-r">
                        <div class="mb-space__dyn-vbox-title">
                          {{ row.article.title }}
                        </div>
                        <div class="mb-space__dyn-vbox-desc">
                          {{ articleDescriptionText(row.article) }}
                        </div>
                        <div class="mb-space__dyn-vstats">
                          <span class="mb-space__dyn-vstat"
                            ><img
                              class="mb-space__dyn-vstat-ico"
                              :src="thumbPlayIco"
                              alt=""
                            />
                            {{ formatCount(row.article.view_count) }}</span
                          >
                          <span class="mb-space__dyn-vstat"
                            ><img
                              class="mb-space__dyn-vstat-ico"
                              :src="thumbDanmuIco"
                              alt=""
                            />
                            {{
                              formatCount(row.article.comment_count || 0)
                            }}</span
                          >
                        </div>
                      </div>
                    </a>
                  </div>
                  <div class="mb-space__dyn-act-bar" aria-label="动态操作">
                    <button
                      type="button"
                      class="mb-space__dyn-act-bar__btn"
                      title="转发"
                      @click.stop
                    >
                      <img
                        class="mb-space__dyn-ico-act__share"
                        :src="shareIco"
                        alt=""
                      />
                      <span class="mb-space__dyn-act-bar__txt">转发</span>
                    </button>
                    <button
                      type="button"
                      class="mb-space__dyn-act-bar__btn"
                      :class="{
                        'is-on': dynCommentArticleId === row.article.id
                      }"
                      title="评论"
                      @click.stop="toggleDynCommentPanel(row.article.id, 'article')"
                    >
                      <svg
                        class="mb-space__dyn-ico-act__svg"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="1.5"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        aria-hidden="true"
                      >
                        <path
                          d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"
                        />
                      </svg>
                      <span class="mb-space__dyn-act-bar__txt"
                        >评论
                        <span class="mb-space__dyn-act-bar__num">{{
                          formatCount(row.article.comment_count || 0)
                        }}</span></span
                      >
                    </button>
                    <button
                      type="button"
                      class="mb-space__dyn-act-bar__btn"
                      :class="{
                        'mb-space__dyn-ico-act--liked': row.article.favorited_by_me
                      }"
                      title="收藏"
                      @click.stop="onDynArticleFavorite(row)"
                    >
                      <span
                        class="mb-space__dyn-collect-ico-wrap"
                        :class="{ 'is-on': row.article.favorited_by_me }"
                      >
                        <img
                          class="mb-space__dyn-ico-act__collect"
                          :src="dynCollectIco"
                          alt=""
                        />
                      </span>
                      <span class="mb-space__dyn-act-bar__txt"
                        >收藏
                        <span class="mb-space__dyn-act-bar__num">{{
                          formatCount(row.article.fav_count || 0)
                        }}</span></span
                      >
                    </button>
                  </div>
                  <MbDynVideoCommentPanel
                    v-if="dynCommentArticleId === row.article.id"
                    ref="dynArticleCommentPanel"
                    :article="row.article"
                    :article-author-id="Number(row.userId)"
                    :owner-can-curate="isDynRowOwner(row)"
                    @patch-article="onDynCommentPatchArticle"
                    @counts="onDynArticleCommentsLiveCounts"
                    @manage-dialog="onDynPanelManageDialog"
                  />
                </template>
              </article>
              <p class="mb-dyn-end">没有更多了</p>
            </div>
          </main>

          <!-- 右：热搜 -->
          <aside class="mb-dyn-right" aria-label="热搜">
            <section class="mb-dyn-card mb-dyn-hot">
              <h2 class="mb-dyn-hot__title">cakecake 热搜</h2>
              <ol class="mb-dyn-hot__list">
                <li
                  v-for="item in hotSearch"
                  :key="'hot-' + item.rank"
                  class="mb-dyn-hot__row"
                  @click="goHotSearch(item)"
                >
                  <span
                    class="mb-dyn-hot__rank"
                    :class="hotRankClass(item.rank)"
                  >{{ item.rank }}</span>
                  <span class="mb-dyn-hot__topic">{{ item.title }}</span>
                  <span
                    v-if="item.badge === '热'"
                    class="mb-dyn-hot__badge mb-dyn-hot__badge--hot"
                  >热</span>
                  <span
                    v-else-if="item.badge === '新'"
                    class="mb-dyn-hot__badge mb-dyn-hot__badge--new"
                  >新</span>
                  <span
                    v-else-if="item.badge === '荐'"
                    class="mb-dyn-hot__badge mb-dyn-hot__badge--rec"
                  >荐</span>
                </li>
              </ol>
            </section>
          </aside>
        </div>
      </div>
    </div>
  </div>
  <Teleport to="body">
    <div
      v-if="dynDeleteTarget"
      class="mb-space__dyn-del-mask"
      role="presentation"
      @click.self="closeDynDeleteDialog"
    >
      <div class="mb-space__dyn-del-box" role="dialog" aria-modal="true">
        <button
          type="button"
          class="mb-space__dyn-del-close"
          aria-label="关闭"
          :disabled="dynDeleteSubmitting"
          @click="closeDynDeleteDialog"
        >
          ×
        </button>
        <div class="mb-space__dyn-del-copy">
          <h3 class="mb-space__dyn-del-title">
            {{
              dynDeleteTarget.kind === "article"
                ? "删除专栏？"
                : dynDeleteTarget.kind === "video"
                  ? "删除视频？"
                  : "删除动态？"
            }}
          </h3>
          <p class="mb-space__dyn-del-sub">删除后不可恢复，是否继续？</p>
        </div>
        <div class="mb-space__dyn-del-actions">
          <button
            type="button"
            class="mb-space__dyn-del-cancel"
            :disabled="dynDeleteSubmitting"
            @click="closeDynDeleteDialog"
          >
            取消
          </button>
          <button
            type="button"
            class="mb-space__dyn-del-ok"
            :disabled="dynDeleteSubmitting"
            @click="confirmDynDelete"
          >
            删除
          </button>
        </div>
      </div>
    </div>
  </Teleport>
  <MbStationDialog
    v-model="dynMbStationOpen"
    :title="dynMbStationTitle"
    :message="dynMbStationMessage"
    :loading="dynMbStationLoading"
    @confirm="confirmDynMbStationDialog"
    @cancel="onDynMbStationDialogCancel"
  />
</template>

<script>
import akari from "@/assets/akari.jpg";
import windmillIcon from "@/assets/dynamics/windmill.png";
import thumbPlayIco from "@/assets/personal_space/play.png";
import thumbDanmuIco from "@/assets/personal_space/danmu.png";
import thumbLaterIco from "@/assets/personal_space/latertowatch.png";
import shareIco from "@/assets/personal_space/share.png";
import dynCollectIco from "@/assets/text/collect.png";
import MbDynVideoCommentPanel from "@/components/minibili/MbDynVideoCommentPanel.vue";
import MbStationDialog from "@/components/minibili/MbStationDialog.vue";
import { ElMessage } from "element-plus";
import { getAccessToken } from "@/utils/authTokens";
import {
  mbGetMe,
  mbGetUserPublic,
  mbListUserFollowing,
  mbListUserPublishedVideos,
  mbListUserPublishedArticles,
  mbToggleVideoLike,
  mbToggleArticleFavorite,
  mbPatchArticlePlayback,
  mbPatchVideoPlayback,
  mbPatchDynamicPlayback,
  mbListUserPublishedDynamics,
  mbPostUserDynamic,
  mbToggleDynamicLike,
  mbDeleteMyDynamic,
  mbDeleteMyVideo,
  mbDeleteMyArticle,
  mbGetHotSearch
} from "@/api/minibili";
import { addSearchHistory } from "@/utils/searchHistory";
import {
  minibiliUserSpaceRoute,
  minibiliUserSpaceRelationsRoute,
  minibiliUserSpaceDynamicRoute,
  minibiliVideoPlayRoute,
  minibiliArticleReadRoute,
  minibiliDynamicReadRoute
} from "@/utils/minibiliRoutes";
import { levelIconUrl } from "@/utils/userLevel";

const PAGE_TITLE = "动态 - cakecake";

const HOT_SEARCH_BADGES = new Set(["热", "新", "荐"]);

function normalizeHotSearchBadge(badge) {
  const b = String(badge || "").trim();
  return HOT_SEARCH_BADGES.has(b) ? b : "";
}

/** 与个人空间动态 Tab 评论管理弹窗文案一致 */
const DYN_MB_STATION = {
  pick_comment: {
    title: "开启评论精选",
    message:
      "开启精选评论后，所有评论都需经过我的确认后再向所有用户展示。可前往PC端创作中心"
  },
  close_comments: {
    title: "关闭评论",
    message:
      "关闭评论将会禁止任何在此评论区发表内容，且已有评论在关闭期间将不可见"
  },
  restore_comments: {
    title: "关闭评论",
    message:
      "恢复评论后，用户可正常发表评论、参与评论互动，已有的评论也恢复正常展示"
  },
  restore_pick_comment: {
    title: "关闭评论精选",
    message: "关闭后，新评论将直接对所有人可见，无需再经过精选确认。"
  }
};

export default {
  name: "MinibiliDynamics",
  components: { MbDynVideoCommentPanel, MbStationDialog },
  data() {
    return {
      akari,
      windmillIcon,
      thumbPlayIco,
      thumbDanmuIco,
      thumbLaterIco,
      shareIco,
      dynCollectIco,
      dynPageBg: new URL("../../assets/dynamics/bg.png@1c.avif", import.meta.url).href,
      draftTitle: "",
      draftContent: "",
      draftImageMode: false,
      draftImageFiles: [],
      draftImagePreviews: [],
      maxDraftImages: 9,
      publishSubmitting: false,
      loading: false,
      activeTab: "all",
      activeFollowId: "all",
      me: null,
      spaceProfile: null,
      followStrip: [],
      feedAll: [],
      dynCommentVideoId: null,
      dynCommentArticleId: null,
      dynCommentDynamicId: null,
      dynDeleteTarget: null,
      dynDeleteSubmitting: false,
      dynMbStationOpen: false,
      dynMbStationTitle: "",
      dynMbStationMessage: "",
      dynMbStationLoading: false,
      dynMbStationKind: null,
      dynMbStationVideoId: null,
      dynMbStationArticleId: null,
      dynMbStationDynamicId: null,
      hotSearch: [],
      feedTabs: [
        { key: "all", label: "全部" },
        { key: "image", label: "图文" },
        { key: "video", label: "视频投稿" },
        { key: "article", label: "专栏" }
      ]
    };
  },
  computed: {
    token() {
      return getAccessToken();
    },
    dynBgLayerStyle() {
      return { backgroundImage: `url(${this.dynPageBg})` };
    },
    profileName() {
      if (!this.me) return "—";
      return this.me.nickname || this.me.username || "用户";
    },
    profileAvatar() {
      const u = this.me && String(this.me.avatar_url || "").trim();
      return u || akari;
    },
    levelDisplay() {
      const fromMe =
        this.me &&
        this.me.level_info &&
        this.me.level_info.current_level;
      if (fromMe != null) {
        return Number(fromMe) || 1;
      }
      const fromProfile =
        this.spaceProfile &&
        this.spaceProfile.level_info &&
        this.spaceProfile.level_info.current_level;
      return fromProfile != null ? Number(fromProfile) || 1 : 1;
    },
    statFollowing() {
      const n = this.spaceProfile && this.spaceProfile.following_count;
      return n != null ? n : "—";
    },
    statFollowers() {
      const n = this.spaceProfile && this.spaceProfile.follower_count;
      return n != null ? n : "—";
    },
    statDynamics() {
      const n = this.spaceProfile && this.spaceProfile.published_count;
      if (n != null) return n;
      if (!this.me) return "—";
      const uid = Number(this.me.user_id);
      return this.feedAll.filter((r) => Number(r.userId) === uid).length;
    },
    mySpaceRoute() {
      if (!this.me) return null;
      return minibiliUserSpaceRoute(this.me.user_id);
    },
    mySpaceDynamicRoute() {
      const base = this.mySpaceRoute;
      if (!base) return null;
      return { ...base, query: { nav: "dynamic" } };
    },
    relationsFollowingRoute() {
      if (!this.me) return null;
      return minibiliUserSpaceRelationsRoute(this.me.user_id, "following");
    },
    relationsFollowersRoute() {
      if (!this.me) return null;
      return minibiliUserSpaceRelationsRoute(this.me.user_id, "followers");
    },
    visibleFeed() {
      let list = this.feedAll;
      if (this.activeFollowId !== "all") {
        const uid = Number(this.activeFollowId);
        list = list.filter(row => row.userId === uid);
      }
      if (this.activeTab === "video") {
        list = list.filter(row => row.kind === "video");
      } else if (this.activeTab === "article") {
        list = list.filter(row => row.kind === "article");
      } else if (this.activeTab === "image") {
        list = list.filter(row => row.kind === "image");
      }
      return list;
    },
    draftCharCount() {
      return (
        String(this.draftTitle || "").length +
        String(this.draftContent || "").length
      );
    },
    canPublishDynamic() {
      return (
        !!String(this.draftTitle || "").trim() ||
        !!String(this.draftContent || "").trim() ||
        this.draftImageFiles.length > 0
      );
    }
  },
  mounted() {
    this.onPageEnter();
  },
  activated() {
    this.onPageEnter();
  },
  beforeUnmount() {
    this.clearDraftImages();
  },
  methods: {
    levelIconUrl,
    minibiliVideoPlayRoute,
    minibiliArticleReadRoute,
    minibiliDynamicReadRoute,
    onPageEnter() {
      document.title = PAGE_TITLE;
      void this.loadHotSearch();
      if (this.token) {
        void this.loadFeed();
      }
    },
    async loadHotSearch() {
      try {
        const res = await mbGetHotSearch(10);
        const items = Array.isArray(res.items) ? res.items : [];
        this.hotSearch = items
          .map((it, i) => ({
            rank: Number(it.rank) > 0 ? Number(it.rank) : i + 1,
            title: String(it.title || "").trim(),
            badge: normalizeHotSearchBadge(it.badge),
            video_id: it.video_id
          }))
          .filter(it => it.title);
      } catch {
        this.hotSearch = [];
      }
    },
    goHotSearch(item) {
      const kw = String(item && item.title).trim();
      if (!kw) {
        return;
      }
      addSearchHistory(kw);
      this.$router.push({ path: "/search/all", query: { keyword: kw } });
    },
    openLoginModal() {
      this.$store.commit("login/SET_LOGIN_TAB", 0);
      this.$store.commit("login/OPEN_LOGIN_MODAL");
    },
    onDraftImageToolClick() {
      this.draftImageMode = !this.draftImageMode;
    },
    openDraftImagePicker() {
      const el = this.$refs.draftImageInput;
      if (el) {
        el.value = "";
        el.click();
      }
    },
    onDraftImagesSelected(ev) {
      const input = ev && ev.target;
      const picked = input && input.files ? Array.from(input.files) : [];
      if (!picked.length) return;
      const remain = this.maxDraftImages - this.draftImageFiles.length;
      if (remain <= 0) {
        ElMessage.warning(`最多上传 ${this.maxDraftImages} 张图片`);
        return;
      }
      const slice = picked.slice(0, remain);
      for (const file of slice) {
        if (!file.type.startsWith("image/")) continue;
        const url = URL.createObjectURL(file);
        this.draftImageFiles.push(file);
        this.draftImagePreviews.push({ url, file });
      }
      this.draftImageMode = true;
      if (input) input.value = "";
    },
    removeDraftImage(ix) {
      const i = Number(ix);
      if (!Number.isFinite(i) || i < 0) return;
      const prev = this.draftImagePreviews[i];
      if (prev && prev.url) {
        try {
          URL.revokeObjectURL(prev.url);
        } catch {
          /* ignore */
        }
      }
      this.draftImagePreviews.splice(i, 1);
      this.draftImageFiles.splice(i, 1);
    },
    clearDraftImages() {
      for (const p of this.draftImagePreviews) {
        if (p && p.url) {
          try {
            URL.revokeObjectURL(p.url);
          } catch {
            /* ignore */
          }
        }
      }
      this.draftImagePreviews = [];
      this.draftImageFiles = [];
      this.draftImageMode = false;
    },
    async onPublishDynamic() {
      if (!this.canPublishDynamic || this.publishSubmitting) return;
      if (!this.token) {
        this.openLoginModal();
        return;
      }
      this.publishSubmitting = true;
      try {
        const item = await mbPostUserDynamic({
          title: this.draftTitle,
          content: this.draftContent,
          images: this.draftImageFiles
        });
        const me = this.me;
        if (me) {
          const row = this.imageRowFromItem(
            {
              userId: me.user_id,
              nickname: me.nickname || me.username,
              avatar: me.avatar_url || ""
            },
            item
          );
          this.feedAll = [row, ...this.feedAll].sort(
            (a, b) => (b.ts || 0) - (a.ts || 0)
          );
          this.bumpPublishedCount(1);
        }
        this.draftTitle = "";
        this.draftContent = "";
        this.clearDraftImages();
        ElMessage.success("动态已发布");
      } catch (e) {
        ElMessage.error((e && e.message) || "发布失败");
      } finally {
        this.publishSubmitting = false;
      }
    },
    hotRankClass(rank) {
      if (rank === 1) return "mb-dyn-hot__rank--top1";
      if (rank === 2) return "mb-dyn-hot__rank--top2";
      if (rank === 3) return "mb-dyn-hot__rank--top3";
      return "";
    },
    formatCount(n) {
      const v = Number(n) || 0;
      if (v >= 10000) {
        return (v / 10000).toFixed(1).replace(/\.0$/, "") + "万";
      }
      return String(v);
    },
    formatDuration(sec) {
      const s = Math.max(0, Math.floor(Number(sec) || 0));
      const h = Math.floor(s / 3600);
      const m = Math.floor((s % 3600) / 60);
      const ss = s % 60;
      const pad = n => String(n).padStart(2, "0");
      if (h > 0) return `${h}:${pad(m)}:${pad(ss)}`;
      return `${m}:${pad(ss)}`;
    },
    formatFeedTime(ts) {
      const t = Number(ts) || 0;
      if (!t) return "";
      const diff = Date.now() - t;
      const min = Math.floor(diff / 60000);
      if (min < 1) return "刚刚";
      if (min < 60) return `${min}分钟前`;
      const hr = Math.floor(min / 60);
      if (hr < 24) return `${hr}小时前`;
      const d = new Date(t);
      const pad = n => String(n).padStart(2, "0");
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
    },
    videoDescriptionText(v) {
      const d =
        v && typeof v.description === "string" ? v.description.trim() : "";
      return d || "暂无简介";
    },
    articleDescriptionText(a) {
      const t = a && typeof a.title === "string" ? a.title.trim() : "";
      return t ? "专栏 · 点击查看全文" : "专栏文章";
    },
    userSpaceDynamicRoute(userId) {
      return minibiliUserSpaceDynamicRoute(userId);
    },
    formatDynDateCN(ts) {
      const d = new Date(ts);
      if (!Number.isFinite(d.getTime())) {
        return "";
      }
      return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
    },
    onWatchLaterStub(e) {
      if (e) {
        e.stopPropagation();
        e.preventDefault();
      }
    },
    toggleDynCommentPanel(mediaId, kind = "video") {
      const id = Number(mediaId);
      if (!Number.isFinite(id) || id <= 0) {
        return;
      }
      if (kind === "dynamic" || kind === "image") {
        if (this.dynCommentDynamicId === id) {
          this.dynCommentDynamicId = null;
          return;
        }
        this.dynCommentDynamicId = id;
        this.dynCommentVideoId = null;
        this.dynCommentArticleId = null;
        return;
      }
      if (kind === "article") {
        if (this.dynCommentArticleId === id) {
          this.dynCommentArticleId = null;
          return;
        }
        this.dynCommentArticleId = id;
        this.dynCommentVideoId = null;
        this.dynCommentDynamicId = null;
        return;
      }
      if (this.dynCommentVideoId === id) {
        this.dynCommentVideoId = null;
        return;
      }
      this.dynCommentVideoId = id;
      this.dynCommentArticleId = null;
      this.dynCommentDynamicId = null;
    },
    dynImageCommentPayload(row) {
      const post = row.post || {};
      const pid = Number(post.id);
      return {
        id: pid,
        comment_count: Number(row.stats && row.stats.comment) || 0,
        comments_closed: !!post.comments_closed,
        comments_curated: !!post.comments_curated
      };
    },
    onDynCommentPatchDynamic({ dynamicId, partial }) {
      this.patchDynImageMeta(dynamicId, partial);
    },
    onDynDynamicCommentsLiveCounts(n) {
      const did = Number(this.dynCommentDynamicId);
      if (Number.isFinite(did) && did > 0) {
        this.patchDynImageMeta(did, { comment_count: Number(n) || 0 });
      }
    },
    patchDynImageMeta(dynamicId, partial) {
      const id = Number(dynamicId);
      if (!Number.isFinite(id) || id <= 0) {
        return;
      }
      for (const row of this.feedAll) {
        if (row.kind !== "image" || !row.post || Number(row.post.id) !== id) {
          continue;
        }
        if (partial.comment_count != null && row.stats) {
          row.stats.comment = Number(partial.comment_count) || 0;
        }
        if (partial.like_count != null && row.stats) {
          row.stats.like = Number(partial.like_count) || 0;
        }
        if (partial.liked_by_me != null) {
          row.liked_by_me = !!partial.liked_by_me;
        }
        if (partial.comments_closed != null) {
          row.post.comments_closed = !!partial.comments_closed;
        }
        if (partial.comments_curated != null) {
          row.post.comments_curated = !!partial.comments_curated;
        }
      }
    },
    async onDynImageLike(row) {
      if (!this.token) {
        ElMessage.warning("请先登录后再点赞");
        return;
      }
      const did = Number(row.post && row.post.id);
      if (!Number.isFinite(did) || did <= 0) {
        return;
      }
      const was = !!row.liked_by_me;
      try {
        const { liked } = await mbToggleDynamicLike(did);
        let delta = 0;
        if (liked && !was) {
          delta = 1;
        } else if (!liked && was) {
          delta = -1;
        }
        const base = Number(row.stats && row.stats.like) || 0;
        this.patchDynImageMeta(did, {
          liked_by_me: liked,
          like_count: Math.max(0, base + delta)
        });
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      }
    },
    openDynDeleteDialog(row) {
      if (!row || !this.isDynRowOwner(row)) {
        return;
      }
      const payload = { kind: row.kind, id: row.id };
      if (row.kind === "video") {
        payload.video = row.video;
      } else if (row.kind === "article") {
        payload.article = row.article;
      } else {
        payload.post = row.post;
      }
      this.dynDeleteTarget = payload;
    },
    closeDynDeleteDialog() {
      if (this.dynDeleteSubmitting) {
        return;
      }
      this.dynDeleteTarget = null;
    },
    async confirmDynDelete() {
      const t = this.dynDeleteTarget;
      if (!t || this.dynDeleteSubmitting) {
        return;
      }
      if (!this.token) {
        ElMessage.warning("请先登录");
        return;
      }
      this.dynDeleteSubmitting = true;
      try {
        if (t.kind === "video") {
          const vid = Number(t.video && t.video.id);
          if (!Number.isFinite(vid) || vid <= 0) {
            throw new Error("无效的视频");
          }
          await mbDeleteMyVideo(vid);
          if (this.dynCommentVideoId === vid) {
            this.dynCommentVideoId = null;
          }
        } else if (t.kind === "article") {
          const aid = Number(t.article && t.article.id);
          if (!Number.isFinite(aid) || aid <= 0) {
            throw new Error("无效的专栏");
          }
          await mbDeleteMyArticle(aid);
          if (this.dynCommentArticleId === aid) {
            this.dynCommentArticleId = null;
          }
        } else if (t.kind === "image") {
          const did = Number(t.post && t.post.id);
          if (!Number.isFinite(did) || did <= 0) {
            throw new Error("无效的动态");
          }
          await mbDeleteMyDynamic(did);
          if (this.dynCommentDynamicId === did) {
            this.dynCommentDynamicId = null;
          }
        } else {
          return;
        }
        this.feedAll = this.feedAll.filter((x) => x.id !== t.id);
        this.bumpPublishedCount(-1);
        this.dynDeleteTarget = null;
        ElMessage({
          message: "删除成功",
          duration: 2200,
          customClass: "mb-space-dyn-delete-toast"
        });
      } catch (e) {
        ElMessage.error((e && e.message) || "删除失败");
      } finally {
        this.dynDeleteSubmitting = false;
      }
    },
    onDynCommentPatchVideo({ videoId, partial }) {
      this.patchDynVideoMeta(videoId, partial);
    },
    onDynCommentsLiveCounts(n) {
      const vid = Number(this.dynCommentVideoId);
      if (Number.isFinite(vid) && vid > 0) {
        this.patchDynVideoMeta(vid, { comment_count: Number(n) || 0 });
      }
    },
    onDynCommentPatchArticle({ articleId, partial }) {
      this.patchDynArticleMeta(articleId, partial);
    },
    onDynArticleCommentsLiveCounts(n) {
      const aid = Number(this.dynCommentArticleId);
      if (Number.isFinite(aid) && aid > 0) {
        this.patchDynArticleMeta(aid, { comment_count: Number(n) || 0 });
      }
    },
    patchDynVideoMeta(videoId, partial) {
      const id = Number(videoId);
      if (!Number.isFinite(id) || id <= 0) {
        return;
      }
      for (const row of this.feedAll) {
        if (row.kind === "video" && row.video && Number(row.video.id) === id) {
          Object.assign(row.video, partial);
        }
      }
    },
    patchDynArticleMeta(articleId, partial) {
      const id = Number(articleId);
      if (!Number.isFinite(id) || id <= 0) {
        return;
      }
      for (const row of this.feedAll) {
        if (
          row.kind === "article" &&
          row.article &&
          Number(row.article.id) === id
        ) {
          Object.assign(row.article, partial);
        }
      }
    },
    isDynRowOwner(row) {
      const me = this.me;
      if (!me || !row) return false;
      return Number(row.userId) === Number(me.user_id);
    },
    bumpPublishedCount(delta) {
      const d = Number(delta) || 0;
      if (!d || !this.spaceProfile) return;
      const base = Number(this.spaceProfile.published_count);
      this.spaceProfile.published_count =
        Number.isFinite(base) && base >= 0 ? Math.max(0, base + d) : d > 0 ? d : 0;
    },
    refreshDynArticleCommentsLive(opts) {
      const ref = this.$refs.dynArticleCommentPanel;
      if (ref && typeof ref.refreshCommentsLive === "function") {
        return ref.refreshCommentsLive(opts || { soft: true, preserveExpand: true });
      }
      return Promise.resolve();
    },
    openDynMbStationDialog(kind, videoId, articleId = 0, dynamicId = 0) {
      const cfg = DYN_MB_STATION[kind];
      if (!cfg) {
        return;
      }
      this.dynMbStationKind = kind;
      this.dynMbStationVideoId = Number(videoId) || null;
      this.dynMbStationArticleId = Number(articleId) || null;
      this.dynMbStationDynamicId = Number(dynamicId) || null;
      this.dynMbStationTitle = cfg.title;
      this.dynMbStationMessage = cfg.message;
      this.dynMbStationOpen = true;
    },
    onDynPanelManageDialog({ kind, articleId, videoId, dynamicId }) {
      const aid = Number(articleId) || 0;
      const vid = Number(videoId) || 0;
      const did = Number(dynamicId) || 0;
      if (aid > 0) {
        this.openDynMbStationDialog(kind, 0, aid, 0);
      } else if (vid > 0) {
        this.openDynMbStationDialog(kind, vid, 0, 0);
      } else if (did > 0) {
        this.openDynMbStationDialog(kind, 0, 0, did);
      }
    },
    onDynMbStationDialogCancel() {
      if (this.dynMbStationLoading) {
        return;
      }
      this.dynMbStationOpen = false;
    },
    async confirmDynMbStationDialog() {
      const kind = this.dynMbStationKind;
      const vid = Number(this.dynMbStationVideoId);
      const aid = Number(this.dynMbStationArticleId);
      const did = Number(this.dynMbStationDynamicId);
      const isArticle = Number.isFinite(aid) && aid > 0;
      const isVideo = Number.isFinite(vid) && vid > 0;
      const isDynamic = Number.isFinite(did) && did > 0;
      if (!isArticle && !isVideo && !isDynamic) {
        this.dynMbStationOpen = false;
        return;
      }
      let body = {};
      if (kind === "pick_comment") {
        body = { comments_curated: true };
      } else if (kind === "restore_pick_comment") {
        body = { comments_curated: false };
      } else if (kind === "close_comments") {
        body = { comments_closed: true };
      } else if (kind === "restore_comments") {
        body = { comments_closed: false };
      } else {
        this.dynMbStationOpen = false;
        return;
      }
      this.dynMbStationLoading = true;
      try {
        if (isArticle) {
          const r = await mbPatchArticlePlayback(aid, body);
          this.patchDynArticleMeta(aid, {
            comments_closed: r.comments_closed,
            comments_curated: r.comments_curated
          });
          if (Number(this.dynCommentArticleId) === aid) {
            await this.refreshDynArticleCommentsLive();
          }
        } else if (isVideo) {
          const r = await mbPatchVideoPlayback(vid, body);
          this.patchDynVideoMeta(vid, {
            comments_closed: r.comments_closed,
            comments_curated: r.comments_curated,
            danmaku_closed: r.danmaku_closed
          });
        } else {
          const r = await mbPatchDynamicPlayback(did, body);
          this.patchDynImageMeta(did, {
            comments_closed: r.comments_closed,
            comments_curated: r.comments_curated
          });
          if (Number(this.dynCommentDynamicId) === did) {
            const ref = this.$refs.dynDynamicCommentPanel;
            const panel = Array.isArray(ref) ? ref[0] : ref;
            if (panel && typeof panel.refreshCommentsLive === "function") {
              await panel.refreshCommentsLive();
            }
          }
        }
        const okMsg =
          kind === "pick_comment"
            ? "已开启评论精选"
            : kind === "restore_pick_comment"
              ? "已关闭评论精选"
              : kind === "close_comments"
                ? "已关闭评论"
                : kind === "restore_comments"
                  ? "已恢复评论"
                  : "操作成功";
        ElMessage.success(okMsg);
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      } finally {
        this.dynMbStationLoading = false;
        this.dynMbStationOpen = false;
      }
    },
    async onDynVideoLike(row) {
      if (!this.token) {
        ElMessage.warning("请先登录后再点赞");
        return;
      }
      const vid = Number(row.video.id);
      const v = row.video;
      const was = !!v.liked_by_me;
      try {
        const { liked } = await mbToggleVideoLike(vid);
        let delta = 0;
        if (liked && !was) {
          delta = 1;
        } else if (!liked && was) {
          delta = -1;
        }
        const base = Number(v.like_count) || 0;
        this.patchDynVideoMeta(vid, {
          liked_by_me: liked,
          like_count: Math.max(0, base + delta)
        });
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      }
    },
    async onDynArticleFavorite(row) {
      if (!this.token) {
        ElMessage.warning("请先登录后再收藏");
        return;
      }
      const aid = Number(row.article.id);
      const a = row.article;
      const was = !!a.favorited_by_me;
      try {
        const { favorited, fav_count } = await mbToggleArticleFavorite(aid);
        let delta = 0;
        if (favorited && !was) {
          delta = 1;
        } else if (!favorited && was) {
          delta = -1;
        }
        const base = Number(a.fav_count) || 0;
        this.patchDynArticleMeta(aid, {
          favorited_by_me: favorited,
          fav_count:
            fav_count != null ? Number(fav_count) : Math.max(0, base + delta)
        });
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      }
    },
    imageRowFromItem(author, item) {
      const tsRaw = item.created_at;
      return {
        id: `dyn-i-${author.userId}-${item.id}`,
        kind: "image",
        userId: author.userId,
        nickname: author.nickname,
        avatar: author.avatar,
        ts: tsRaw ? Date.parse(tsRaw.replace(/-/g, "/")) || 0 : 0,
        verb: "",
        post: {
          id: item.id,
          title: item.title || "",
          content: item.content || "",
          images: Array.isArray(item.images) ? item.images : [],
          comments_closed: !!item.comments_closed,
          comments_curated: !!item.comments_curated
        },
        liked_by_me: !!item.liked_by_me,
        stats: {
          forward: 0,
          comment: Number(item.comment_count) || 0,
          like: Number(item.like_count) || 0
        }
      };
    },
    articleRowFromItem(author, item) {
      const tsRaw = item.published_at || item.created_at;
      return {
        id: `dyn-a-${author.userId}-${item.id}`,
        kind: "article",
        userId: author.userId,
        nickname: author.nickname,
        avatar: author.avatar,
        ts: tsRaw ? Date.parse(tsRaw) || 0 : 0,
        verb: "投稿了文章",
        article: {
          id: item.id,
          title: item.title,
          cover_url: item.cover_url,
          view_count: item.view_count,
          comment_count: item.comment_count,
          fav_count: item.fav_count,
          forward_count: item.forward_count,
          favorited_by_me: !!item.favorited_by_me,
          comments_closed: !!item.comments_closed,
          comments_curated: !!item.comments_curated
        },
        stats: {
          forward: item.forward_count || 0,
          comment: item.comment_count || 0,
          like: item.fav_count || 0
        }
      };
    },
    videoRowFromItem(author, item) {
      return {
        id: `dyn-v-${author.userId}-${item.id}`,
        kind: "video",
        userId: author.userId,
        nickname: author.nickname,
        avatar: author.avatar,
        ts: item.created_at ? Date.parse(item.created_at) || 0 : 0,
        verb: "投稿了视频",
        video: {
          id: item.id,
          title: item.title,
          description: item.description || "",
          cover_url: item.cover_url,
          duration: item.duration,
          play_count: item.play_count,
          danmaku_count: item.danmaku_count,
          comment_count: item.comment_count,
          comments_closed: !!item.comments_closed,
          danmaku_closed: !!item.danmaku_closed,
          like_count: item.like_count,
          liked_by_me: !!item.liked_by_me
        },
        stats: {
          forward: 0,
          comment: item.comment_count || 0,
          like: item.like_count || 0
        }
      };
    },
    async loadFeed() {
      this.loading = true;
      try {
        const me = await mbGetMe();
        this.me = me;

        try {
          this.spaceProfile = await mbGetUserPublic(me.user_id);
        } catch {
          this.spaceProfile = null;
        }

        const authorMe = {
          userId: me.user_id,
          nickname: me.nickname || me.username,
          avatar: me.avatar_url || ""
        };

        let following = [];
        try {
          const fol = await mbListUserFollowing(me.user_id, { limit: 24 });
          following = fol.items || [];
        } catch {
          following = [];
        }

        this.followStrip = [
          { id: "all", label: "全部动态", isAll: true },
          ...following.map(u => ({
            id: String(u.user_id),
            label: u.nickname || `用户${u.user_id}`,
            avatar: u.avatar_url || ""
          }))
        ];

        const authors = [
          authorMe,
          ...following.map(u => ({
            userId: u.user_id,
            nickname: u.nickname || `用户${u.user_id}`,
            avatar: u.avatar_url || ""
          }))
        ];

        const feedRows = [];
        await Promise.all(
          authors.slice(0, 8).map(async author => {
            try {
              const [videos, articles, dynamics] = await Promise.all([
                mbListUserPublishedVideos(author.userId, { limit: 8 }),
                mbListUserPublishedArticles(author.userId, { limit: 8 }),
                mbListUserPublishedDynamics(author.userId, { limit: 8 })
              ]);
              for (const item of videos.items || []) {
                feedRows.push(this.videoRowFromItem(author, item));
              }
              for (const item of articles.items || []) {
                feedRows.push(this.articleRowFromItem(author, item));
              }
              for (const item of dynamics.items || []) {
                feedRows.push(this.imageRowFromItem(author, item));
              }
            } catch {
              /* 单个 UP 失败不影响整页 */
            }
          })
        );

        this.feedAll = feedRows.sort((a, b) => (b.ts || 0) - (a.ts || 0));
      } catch {
        this.feedAll = [];
        this.followStrip = [];
        this.me = null;
        this.spaceProfile = null;
      } finally {
        this.loading = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "./dynamics.scss";
</style>

<style lang="scss">
@import "./dynDeleteDialog.scss";
</style>
