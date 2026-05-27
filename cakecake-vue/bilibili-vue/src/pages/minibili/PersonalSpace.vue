<template>
  <div class="mb-space">
    <div class="mb-space__col">
      <header class="mb-space__header" aria-label="个人空间头图">
        <div
          class="mb-space__banner-bg"
          :style="{ backgroundImage: 'url(' + bannerUrl + ')' }"
          aria-hidden="true"
        />
        <div class="mb-space__header-shade" aria-hidden="true" />
        <MbSpacePerspective
          v-if="isRealSpaceOwner"
          v-model="spacePerspective"
        />
        <div class="mb-space__header-bar">
        <div class="mb-space__profile">
          <img
            class="mb-space__avatar"
            :src="avatarDisplay"
            width="80"
            height="80"
            alt=""
          />
          <div class="mb-space__profile-text">
            <div class="mb-space__name-row">
              <span class="mb-space__name">{{ displayName }}</span>
              <div class="mb-space__badges">
                <img
                  class="mb-space__level-badge"
                  :src="levelIconUrl(levelDisplay)"
                  width="36"
                  height="36"
                  alt=""
                  :title="'LV' + levelDisplay"
                />
                <span
                  v-if="spaceGenderKey === 'male'"
                  class="mb-space__gender mb-space__gender--ico"
                  role="img"
                  :aria-label="'性别：' + spaceGenderLabel"
                  :title="'性别：' + spaceGenderLabel"
                >
                  <img
                    class="mb-space__gender-img"
                    :src="genderMaleIco"
                    width="18"
                    height="18"
                    alt=""
                  />
                </span>
                <span
                  v-else-if="spaceGenderKey === 'female'"
                  class="mb-space__gender mb-space__gender--ico"
                  role="img"
                  :aria-label="'性别：' + spaceGenderLabel"
                  :title="'性别：' + spaceGenderLabel"
                >
                  <img
                    class="mb-space__gender-img"
                    :src="genderFemaleIco"
                    width="18"
                    height="18"
                    alt=""
                  />
                </span>
              </div>
            </div>
            <p class="mb-space__sign">{{ displaySign }}</p>
          </div>
        </div>
        <div class="mb-space__header-aside">
          <MbSpacePerspectivePicker
            v-if="isRealSpaceOwner && !isPerspectivePreview"
            v-model="spacePerspective"
          />
          <MbSpaceHeaderActions
            :user-id="userIdNum"
            :is-owner="isOwnSpace"
            :followed-by-me="headerActionsFollowedByMe"
            :preview-only="isPerspectivePreview"
            @update:followed-by-me="onHeaderFollowedByMeUpdate"
            @follower-count="onSpaceFollowerCount"
            @login="openMbLoginModalFromDynCmt"
          />
        </div>
        </div>
      </header>

      <nav class="mb-space__navbar" aria-label="空间主导航">
        <div class="mb-space__dock-row">
          <div class="mb-space__tabs">
            <button
              v-for="tab in navTabs"
              :key="tab.key"
              type="button"
              class="mb-space__tab"
              :class="{ 'is-on': activeNav === tab.key }"
              @click="onNavClick(tab.key)"
            >
              <img
                class="mb-space__tab-ico"
                :class="tab.iconClass"
                :src="tab.icon"
                alt=""
              />
              <span>{{ tab.label }}</span>
              <span v-if="tab.badge != null" class="mb-space__tab-badge">{{
                tab.badge
              }}</span>
            </button>
          </div>
          <div class="mb-space__dock-gap" aria-hidden="true" />
          <div class="mb-space__search">
            <input
              v-model.trim="videoSearch"
              type="search"
              class="mb-space__search-input"
                placeholder="搜索视频、动态"
              autocomplete="off"
              @keydown.enter.prevent="onSpaceSearchSubmit"
            />
            <button
              type="button"
              class="mb-space__search-btn"
              aria-label="搜索"
              @click="onSpaceSearchSubmit"
            >
              <span class="mb-space__search-ico" aria-hidden="true" />
            </button>
          </div>
          <div class="mb-space__stats" aria-label="空间数据">
            <button
              type="button"
              class="mb-space__stat mb-space__stat--link"
              @click="openRelations('following')"
            >
              <span class="mb-space__stat-k">关注</span>
              <span class="mb-space__stat-v">{{ statFollowing }}</span>
            </button>
            <button
              type="button"
              class="mb-space__stat mb-space__stat--link"
              @click="openRelations('followers')"
            >
              <span class="mb-space__stat-k">粉丝</span>
              <span class="mb-space__stat-v">{{ statFans }}</span>
            </button>
            <div class="mb-space__stat">
              <span class="mb-space__stat-k">获赞</span>
              <span class="mb-space__stat-v">{{ statLikes }}</span>
            </div>
            <div class="mb-space__stat">
              <span class="mb-space__stat-k">播放</span>
              <span class="mb-space__stat-v">{{ statPlays }}</span>
            </div>
          </div>
        </div>
      </nav>

      <div class="mb-space__body">
        <p v-if="loadError" class="mb-space__err">{{ loadError }}</p>
        <div
          v-else
          class="mb-space__split"
          :class="{
            'mb-space__split--no-aside': activeNav === 'contribute' || activeNav === 'collect'
          }"
        >
          <main class="mb-space__main">
          <template v-if="activeNav === 'home'">
            <header class="mb-space__sec-head">
              <div class="mb-space__sec-left">
                <h2 class="mb-space__sec-title">
                  <span class="mb-space__sec-title-w">视频</span
                  ><span class="mb-space__sec-dot">·</span
                  ><span class="mb-space__sec-count">{{ videoTotalDisplay }}</span>
                </h2>
                <div class="mb-space__subtabs" role="group" aria-label="视频排序">
                  <button
                    type="button"
                    class="mb-space__subtab"
                    :class="{ 'is-on': videoSort === 'new' }"
                    @click="videoSort = 'new'"
                  >
                    最新发布
                  </button>
                  <button
                    type="button"
                    class="mb-space__subtab"
                    :class="{ 'is-on': videoSort === 'play' }"
                    @click="videoSort = 'play'"
                  >
                    最多播放
                  </button>
                  <button
                    type="button"
                    class="mb-space__subtab"
                    :class="{ 'is-on': videoSort === 'fav' }"
                    @click="videoSort = 'fav'"
                  >
                    最多收藏
                  </button>
                </div>
              </div>
              <div class="mb-space__sec-right">
                <button
                  type="button"
                  class="mb-space__play-all"
                  :disabled="!sortedVideos.length"
                  @click="openPlayAll"
                >
                  <svg
                    class="mb-space__play-ico"
                    viewBox="0 0 24 24"
                    aria-hidden="true"
                  >
                    <path
                      fill="currentColor"
                      d="M8 5v14l11-7L8 5zm2 4.2L15.1 12 10 14.8V9.2z"
                    />
                  </svg>
                  播放全部
                </button>
                <button
                  type="button"
                  class="mb-space__sec-more"
                  :disabled="!sortedVideos.length"
                  @click="onSeeMoreVideos"
                >
                  查看更多
                  <span class="mb-space__sec-more-arr" aria-hidden="true">›</span>
                </button>
              </div>
            </header>
          </template>
          <template v-else-if="activeNav === 'contribute'">
            <div class="mb-space__contrib-outer">
              <aside class="mb-space__dyn-sidenav" aria-label="投稿分类">
                <button
                  type="button"
                  class="mb-space__dyn-sub mb-space__dyn-sub--split"
                  :class="{ 'is-on': contribSubtab === 'video' }"
                  @click="contribSubtab = 'video'"
                >
                <span>视频</span>
                  <span class="mb-space__dyn-sub-count">{{
                    videoTotalDisplay
                  }}</span>
                </button>
                <button
                  type="button"
                  class="mb-space__dyn-sub mb-space__dyn-sub--split"
                  :class="{ 'is-on': contribSubtab === 'article' }"
                  @click="contribSubtab = 'article'"
                >
                <span>图文</span>
                  <span class="mb-space__dyn-sub-count">{{
                    spaceArticles.length
                  }}</span>
                </button>
              </aside>
              <template v-if="contribSubtab === 'video'">
                <div class="mb-space__contrib-right-head">
                  <h2 class="mb-space__contrib-h2">我的视频</h2>
                  <div class="mb-space__contrib-actions">
                    <button
                      type="button"
                      class="mb-space__play-all"
                      :disabled="!sortedVideos.length"
                      @click="openPlayAll"
                    >
                      <svg
                        class="mb-space__play-ico"
                        viewBox="0 0 24 24"
                        aria-hidden="true"
                      >
                        <path
                          fill="currentColor"
                          d="M8 5v14l11-7L8 5zm2 4.2L15.1 12 10 14.8V9.2z"
                        />
                      </svg>
                      播放全部
                    </button>
                    <div
                      class="mb-space__contrib-view"
                      role="group"
          aria-label="展示模式"
                    >
                      <button
                        type="button"
                        class="mb-space__contrib-view-btn"
                        :class="{ 'is-on': spaceVideoViewMode === 'grid' }"
          aria-label="搜索"
                                    title="网格视图"
                        @click="spaceVideoViewMode = 'grid'"
                      >
                        <svg
                          class="mb-space__contrib-view-ico"
                          viewBox="0 0 24 24"
                          aria-hidden="true"
                        >
                          <path
                            fill="currentColor"
                            d="M4 4h7v7H4V4zm9 0h7v7h-7V4zM4 13h7v7H4v-7zm9 0h7v7h-7v-7z"
                          />
                        </svg>
                      </button>
                      <button
                        type="button"
                        class="mb-space__contrib-view-btn"
                        :class="{ 'is-on': spaceVideoViewMode === 'list' }"
          aria-label="搜索"
                                    title="列表视图"
                        @click="spaceVideoViewMode = 'list'"
                      >
                        <svg
                          class="mb-space__contrib-view-ico"
                          viewBox="0 0 24 24"
                          aria-hidden="true"
                        >
                          <path
                            fill="currentColor"
                            d="M4 6h16v2H4V6zm0 5h16v2H4v-2zm0 5h16v2H4v-2z"
                          />
                        </svg>
                      </button>
                    </div>
                  </div>
                  <div class="mb-space__contrib-toolbar">
                    <div
                      class="mb-space__subtabs"
                      role="group"
          aria-label="搜索"
                    >
                      <button
                        type="button"
                        class="mb-space__subtab"
                        :class="{ 'is-on': videoSort === 'new' }"
                        @click="videoSort = 'new'"
                      >
                        最新发布
                      </button>
                      <button
                        type="button"
                        class="mb-space__subtab"
                        :class="{ 'is-on': videoSort === 'play' }"
                        @click="videoSort = 'play'"
                      >
                        最多播放
                      </button>
                      <button
                        type="button"
                        class="mb-space__subtab"
                        :class="{ 'is-on': videoSort === 'fav' }"
                        @click="videoSort = 'fav'"
                      >
                        最多收藏
                      </button>
                    </div>
                  </div>
                </div>
                <div class="mb-space__contrib-feed">
                  <p
                    v-if="listLoading && !videos.length"
                    class="mb-space__hint"
                  >
                    加载中…
                  </p>
                  <div
                    v-else-if="!sortedVideos.length"
                    class="mb-space__empty-img"
                    role="img"
          aria-label="暂无视频"
                  >
                    <img :src="dynEmptyImg" alt="" />
                  </div>
                  <ul v-else :class="spaceVideoGridClass">
                    <li
                      v-for="v in sortedVideos"
                      :key="v.id"
                      class="mb-space__vcell"
                    >
                      <router-link
                        class="mb-space__vcell-link"
                        :class="{
                          'mb-space__vcell-link--list':
                            spaceVideoViewMode === 'list'
                        }"
                        :to="minibiliVideoPlayRoute(v.id)"
                      >
                        <div
                          class="mb-space__vthumb-wrap"
                          :class="{
                            'mb-space__vthumb-wrap--list':
                              spaceVideoViewMode === 'list',
                            'mb-space__vthumb-wrap--dyn':
                              spaceVideoViewMode === 'grid'
                          }"
                        >
                          <img
                            class="mb-space__vthumb"
                            :src="v.cover_url || akari"
                            :alt="v.title"
                          />
                          <span
                            v-if="spaceVideoViewMode === 'list'"
                            class="mb-space__vlist-dur"
                            aria-hidden="true"
                            >{{ formatDuration(v.duration) }}</span
                          >
                          <div
                            v-else
                            class="mb-space__vthumb-default"
                            aria-hidden="true"
                          >
                            <div class="mb-space__vthumb-stats-l">
                              <span class="mb-space__vthumb-stat">
                                <img
                                  class="mb-space__vstat-ico"
                                  :src="thumbPlayIco"
                                  alt=""
                                />
                                {{ formatCount(v.play_count) }}
                              </span>
                              <span class="mb-space__vthumb-stat">
                                <img
                                  class="mb-space__vstat-ico"
                                  :src="thumbDanmuIco"
                                  alt=""
                                />
                                {{ formatCount(v.danmaku_count) }}
                              </span>
                            </div>
                            <span class="mb-space__vdur">{{
                              formatDuration(v.duration)
                            }}</span>
                          </div>
                          <button
                            type="button"
                            class="mb-space__vthumb-later"
          aria-label="稍后再看"
                            @click.stop.prevent="onWatchLaterPlaceholder($event, v.id)"
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
                        <div
                          class="mb-space__vtext-col"
                          :class="{
                            'mb-space__vtext-col--list':
                              spaceVideoViewMode === 'list'
                          }"
                        >
                          <template v-if="spaceVideoViewMode === 'list'">
                            <div class="mb-space__vlist-main">
                              <p
                                class="mb-space__vtitle mb-space__vtitle--list"
                                :title="v.title"
                              >
                                {{ v.title }}
                              </p>
                              <p
                                v-if="videoDescriptionText(v) !== '暂无简介'"
                                class="mb-space__vdesc"
                              >
                                {{ videoDescriptionText(v) }}
                              </p>
                            </div>
                          </template>
                          <template v-else>
                            <p class="mb-space__vtitle" :title="v.title">
                              {{ v.title }}
                            </p>
                          </template>
                          <p class="mb-space__vmeta">
                            <template v-if="isVideoSelfOnlyVisible(v)">
                              <svg
                                class="mb-space__vmeta-lock"
                                viewBox="0 0 24 24"
                                aria-hidden="true"
                              >
                                <path
                                  fill="currentColor"
                                  d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"
                                />
                              </svg>
                              <span class="mb-space__vmeta-privacy"
                                >|</span
                              >
                              <span
                                class="mb-space__vmeta-sep"
                                aria-hidden="true"
                                >|</span
                              >
                            </template>
                            <template
                              v-if="spaceVideoViewMode === 'list'"
                            >
                              <span class="mb-space__vmeta-stat">
                                <img
                                  class="mb-space__vmeta-stat-ico"
                                  :src="thumbPlayIco"
                                  alt=""
                                />
                                {{ formatCount(v.play_count) }}
                              </span>
                              <span class="mb-space__vmeta-stat">
                                <img
                                  class="mb-space__vmeta-stat-ico"
                                  :src="thumbDanmuIco"
                                  alt=""
                                />
                                {{ formatCount(v.danmaku_count || 0) }}
                              </span>
                              <span
                                class="mb-space__vmeta-date mb-space__vmeta-date--inline"
                                >{{
                                  formatVideoYMD(v.created_at)
                                }}</span
                              >
                            </template>
                            <span
                              v-else
                              class="mb-space__vmeta-date"
                              >{{ formatVideoYMD(v.created_at) }}</span
                            >
                          </p>
                        </div>
                      </router-link>
                    </li>
                  </ul>
                  <div v-if="nextCursor" class="mb-space__more-wrap">
                    <button
                      type="button"
                      class="mb-space__more"
                      :disabled="listLoading"
                      @click="loadMore"
                    >
                {{ listLoading ? "加载中…" : "加载更多" }}
                    </button>
                  </div>
                </div>
              </template>
              <template v-else>
                <div
                  class="mb-space__contrib-right-head mb-space__contrib-right-head--article"
                >
                  <h2 class="mb-space__contrib-h2">我的专栏</h2>
                </div>
                <div
                  class="mb-space__contrib-feed mb-space__contrib-feed--article"
                  aria-label="图文投稿"
                >
                  <p v-if="spaceArticlesLoading" class="mb-space__hint">
                    加载中…
                  </p>
                  <div
                    v-else-if="!filteredSpaceArticles.length"
                    class="mb-space__empty-img"
                    role="img"
                    aria-label="暂无专栏"
                  >
                    <img :src="dynEmptyImg" alt="" />
                  </div>
                  <ul v-else class="mb-space__article-list">
                    <li
                      v-for="art in filteredSpaceArticles"
                      :key="art.id"
                      class="mb-space__article-item"
                    >
                      <router-link
                        v-if="minibiliArticleReadRoute(art.id)"
                        class="mb-space__article-link"
                        :to="minibiliArticleReadRoute(art.id)"
                      >
                        <div class="mb-space__article-cover-wrap">
                          <img
                            class="mb-space__article-cover"
                            :src="art.cover_url || dynEmptyImg"
                            :alt="art.title"
                          />
                        </div>
                        <div class="mb-space__article-body">
                          <h3 class="mb-space__article-title" :title="art.title">
                            {{ art.title }}
                          </h3>
                          <p class="mb-space__article-meta">
                            <span>{{ art.published_at || art.created_at }}</span>
                            <span class="mb-space__article-meta-dot" aria-hidden="true"
                              >·</span
                            >
                            <span>阅读 {{ art.view_count || 0 }}</span>
                          </p>
                        </div>
                      </router-link>
                    </li>
                  </ul>
                </div>
              </template>
            </div>
          </template>
          <template v-if="activeNav === 'home'">
            <p v-if="listLoading && !videos.length" class="mb-space__hint">
              加载中…
            </p>
            <div
              v-else-if="!sortedVideos.length && !homeDynSearchMatches.length"
              class="mb-space__empty-img"
              role="img"
              :aria-label="spaceSearchQuery ? '无匹配内容' : '暂无视频'"
            >
              <img :src="dynEmptyImg" alt="" />
            </div>
            <ul v-else-if="sortedVideos.length" :class="spaceVideoGridClass">
              <li v-for="v in sortedVideos" :key="v.id" class="mb-space__vcell">
                <router-link
                  class="mb-space__vcell-link"
                  :to="minibiliVideoPlayRoute(v.id)"
                >
                  <div class="mb-space__vthumb-wrap">
                    <img
                      class="mb-space__vthumb"
                      :src="v.cover_url || akari"
                      :alt="v.title"
                    />
                    <div class="mb-space__vthumb-default" aria-hidden="true">
                      <div class="mb-space__vthumb-stats-l">
                        <span class="mb-space__vthumb-stat">
                          <img
                            class="mb-space__vstat-ico"
                            :src="thumbPlayIco"
                            alt=""
                          />
                          {{ formatCount(v.play_count) }}
                        </span>
                        <span class="mb-space__vthumb-stat">
                          <img
                            class="mb-space__vstat-ico"
                            :src="thumbDanmuIco"
                            alt=""
                          />
                          {{ formatCount(v.danmaku_count) }}
                        </span>
                      </div>
                      <span class="mb-space__vdur">{{
                        formatDuration(v.duration)
                      }}</span>
                    </div>
                    <button
                      type="button"
                      class="mb-space__vthumb-later"
          aria-label="搜索"
                      @click.stop.prevent="onWatchLaterPlaceholder($event, v.id)"
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
                  <div class="mb-space__vtext-col">
                    <p class="mb-space__vtitle" :title="v.title">{{ v.title }}</p>
                    <p class="mb-space__vmeta">
                      <template v-if="isVideoSelfOnlyVisible(v)">
                        <svg
                          class="mb-space__vmeta-lock"
                          viewBox="0 0 24 24"
                          aria-hidden="true"
                        >
                          <path
                            fill="currentColor"
                            d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"
                          />
                        </svg>
                        <span class="mb-space__vmeta-privacy">仅自己可见</span>
                        <span class="mb-space__vmeta-sep" aria-hidden="true"
                                >|</span
                        >
                      </template>
                      <span class="mb-space__vmeta-date">{{
                        formatVideoYMD(v.created_at)
                      }}</span>
                    </p>
                  </div>
                </router-link>
              </li>
            </ul>
            <section
              v-if="spaceSearchQuery && homeDynSearchMatches.length"
              class="mb-space__home-block mb-space__home-search-dyn"
              aria-label="相关动态"
            >
              <header class="mb-space__sec-head mb-space__home-sec-head">
                <div class="mb-space__sec-left">
                  <h2 class="mb-space__sec-title">
                    <span class="mb-space__sec-title-w">相关动态</span>
                    <span class="mb-space__sec-dot">·</span>
                    <span class="mb-space__sec-count">{{
                      homeDynSearchMatches.length
                    }}</span>
                  </h2>
                </div>
                <div class="mb-space__sec-right">
                  <button
                    type="button"
                    class="mb-space__sec-more"
                    @click="openDynamicWithSearch"
                  >
                    在动态中查看
                    <span class="mb-space__sec-more-arr" aria-hidden="true">›</span>
                  </button>
                </div>
              </header>
              <div class="mb-space__home-dyn-panel">
                <ul class="mb-space__home-dyn-hits">
                  <li
                    v-for="row in homeDynSearchMatches"
                    :key="'home-dyn-' + row.id"
                    class="mb-space__home-dyn-card"
                  >
                    <component
                      :is="dynRowSearchRoute(row) ? 'router-link' : 'div'"
                      class="mb-space__home-dyn-card-link"
                      :to="dynRowSearchRoute(row) || undefined"
                    >
                      <div class="mb-space__home-dyn-card-thumb">
                        <img
                          class="mb-space__home-dyn-card-cover"
                          :src="dynRowSearchThumb(row) || akari"
                          alt=""
                        />
                        <span class="mb-space__home-dyn-card-kind">{{
                          dynRowSearchKindLabel(row)
                        }}</span>
                        <span
                          v-if="
                            row.kind === 'video' &&
                            row.video &&
                            formatDuration(row.video.duration)
                          "
                          class="mb-space__home-dyn-card-dur"
                          >{{ formatDuration(row.video.duration) }}</span
                        >
                      </div>
                      <div class="mb-space__home-dyn-card-body">
                        <h3 class="mb-space__home-dyn-card-title">
                          {{ dynRowSearchTitle(row) }}
                        </h3>
                        <p
                          v-if="dynRowSearchExcerpt(row)"
                          class="mb-space__home-dyn-card-desc"
                        >
                          {{ dynRowSearchExcerpt(row) }}
                        </p>
                        <p class="mb-space__home-dyn-card-meta">
                          {{ dynRowSearchMeta(row) }}
                        </p>
                      </div>
                    </component>
                  </li>
                </ul>
              </div>
            </section>
            <div v-if="nextCursor && sortedVideos.length" class="mb-space__more-wrap">
              <button
                type="button"
                class="mb-space__more"
                :disabled="listLoading"
                @click="loadMore"
              >
                {{ listLoading ? "加载中…" : "加载更多" }}
              </button>
            </div>
            <section
              v-if="
                showHomeFoldersSection &&
                (homeFoldersTotal > 0 ||
                  homeFoldersLoading ||
                  isRealSpaceOwner)
              "
              class="mb-space__home-block"
              aria-label="收藏夹"
            >
              <header class="mb-space__sec-head mb-space__home-sec-head">
                <div class="mb-space__sec-left">
                  <h2 class="mb-space__sec-title">
                    <span class="mb-space__sec-title-w">收藏夹</span
                    ><span class="mb-space__sec-dot">·</span
                    ><span class="mb-space__sec-count">{{ homeFolderTotalDisplay }}</span>
                  </h2>
                  <span
                    v-if="isSpaceOwner && homeFoldersHidden > 0"
                    class="mb-space__home-privacy"
                  >
                    <svg
                      class="mb-space__home-privacy-lock"
                      viewBox="0 0 24 24"
                      aria-hidden="true"
                    >
                      <path
                        fill="currentColor"
                        d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"
                      />
                    </svg>
                    <span>{{ homeFoldersHidden }}个已隐藏</span>
                  </span>
                </div>
                <div class="mb-space__sec-right">
                  <button
                    type="button"
                    class="mb-space__sec-more"
                    :disabled="!homeFoldersPreview.length && !homeFoldersLoading"
                    @click="onHomeSeeMoreFolders"
                  >
                    查看更多
                    <span class="mb-space__sec-more-arr" aria-hidden="true">›</span>
                  </button>
                </div>
              </header>
              <p v-if="homeFoldersLoading" class="mb-space__hint">加载中…</p>
              <div
                v-else-if="!homeFoldersPreview.length"
                class="mb-space__empty-img"
                role="img"
                aria-label="暂无收藏夹"
              >
                <img :src="dynEmptyImg" alt="" />
              </div>
              <ul v-else class="mb-space__home-folder-grid">
                <li v-for="f in homeFoldersPreview" :key="'hf-' + f.id">
                  <button
                    type="button"
                    class="mb-space__home-folder-card"
                    @click="onHomeOpenCollectFolder(f.id)"
                  >
                    <div class="mb-space__home-folder-cover">
                      <img
                        v-if="f.coverUrl"
                        :src="f.coverUrl"
                        :alt="f.title"
                      />
                      <span v-else class="mb-space__home-folder-cover-ph" />
                      <span class="mb-space__home-folder-count"
                        >{{ f.videoCount }}个视频</span
                      >
                    </div>
                    <p class="mb-space__home-folder-title" :title="f.title">
                      {{ f.title }}
                    </p>
                    <p class="mb-space__home-folder-meta">
                      {{ f.isPublic ? "公开" : "私密" }}
                    </p>
                  </button>
                </li>
              </ul>
            </section>

            <section
              v-if="showHomeRecentCoinsSection"
              class="mb-space__home-block"
              aria-label="最近投币的视频"
            >
              <header class="mb-space__sec-head mb-space__home-sec-head">
                <div class="mb-space__sec-left">
                  <h2 class="mb-space__sec-title">
                    <span class="mb-space__sec-title-w">最近投币的视频</span>
                  </h2>
                  <span
                    v-if="isSpaceOwner && spacePrivacyView && !spacePrivacyView.public_recent_coins"
                    class="mb-space__home-privacy"
                  >
                    <svg
                      class="mb-space__home-privacy-lock"
                      viewBox="0 0 24 24"
                      aria-hidden="true"
                    >
                      <path
                        fill="currentColor"
                        d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"
                      />
                    </svg>
                    <span>仅自己可见</span>
                  </span>
                </div>
              </header>
              <p v-if="homeRecentCoinsLoading" class="mb-space__hint">加载中…</p>
              <div
                v-else-if="!homeRecentCoinsPreview.length"
                class="mb-space__empty-img"
                role="img"
                aria-label="暂无投币记录"
              >
                <img :src="dynEmptyImg" alt="" />
              </div>
              <ul
                v-else
                class="mb-space__video-grid mb-space__video-grid--home-coin"
              >
                <li
                  v-for="v in homeRecentCoinsPreview"
                  :key="'coin-' + v.id"
                  class="mb-space__vcell"
                >
                  <router-link
                    class="mb-space__vcell-link"
                    :to="minibiliVideoPlayRoute(v.id)"
                  >
                    <div class="mb-space__vthumb-wrap">
                      <img
                        class="mb-space__vthumb"
                        :src="v.cover_url || akari"
                        :alt="v.title"
                      />
                      <div class="mb-space__vthumb-default" aria-hidden="true">
                        <div class="mb-space__vthumb-stats-l">
                          <span class="mb-space__vthumb-stat">
                            <img
                              class="mb-space__vstat-ico"
                              :src="thumbPlayIco"
                              alt=""
                            />
                            {{ formatCount(v.play_count) }}
                          </span>
                          <span class="mb-space__vthumb-stat">
                            <img
                              class="mb-space__vstat-ico"
                              :src="thumbDanmuIco"
                              alt=""
                            />
                            {{ formatCount(v.comment_count ?? v.danmaku_count) }}
                          </span>
                        </div>
                        <span class="mb-space__vdur">{{
                          formatDuration(v.duration)
                        }}</span>
                      </div>
                    </div>
                    <div class="mb-space__vtext-col">
                      <p class="mb-space__vtitle" :title="v.title">{{ v.title }}</p>
                      <p class="mb-space__vmeta mb-space__vmeta--home-coin">
                        <span class="mb-space__vmeta-up">
                          <img
                            class="mb-space__collect-up-ico"
                            :src="collectUpIco"
                            alt=""
                          />
                          {{ v.uploader }}
                        </span>
                      </p>
                    </div>
                  </router-link>
                </li>
              </ul>
            </section>

          </template>
          <div v-else-if="activeNav === 'dynamic'" class="mb-space__dynamic">
            <div class="mb-space__dyn-layout">
              <aside class="mb-space__dyn-sidenav" aria-label="视频排序">
                <button
                  type="button"
                  class="mb-space__dyn-sub"
                  :class="{ 'is-on': dynamicSubtab === 'all' }"
                  @click="dynamicSubtab = 'all'"
                >全部</button>
                <button
                  type="button"
                  class="mb-space__dyn-sub"
                  :class="{ 'is-on': dynamicSubtab === 'video' }"
                  @click="dynamicSubtab = 'video'"
                >视频</button>
              </aside>
              <div class="mb-space__dyn-feed">
                <template v-if="dynVisibleList.length">
                  <article
                    v-for="row in dynVisibleList"
                    :key="row.id"
                    class="mb-space__dyn-card"
                  >
                    <div
                      v-if="isDynRowPinned(row)"
                      class="mb-space__dyn-pin-badge"
                      role="status"
          aria-label="搜索"
                    >
                      <img
                        class="mb-space__dyn-pin-badge-ico"
                        :src="dynPinTopIco"
                        width="16"
                        height="16"
                        alt=""
                      />
                      <span class="mb-space__dyn-pin-badge-txt">置顶</span>
                    </div>
                    <template v-if="row.kind === 'image'">
                      <header class="mb-space__dyn-head">
                        <img
                          class="mb-space__dyn-avatar"
                          :src="avatarDisplay"
                          width="48"
                          height="48"
                          alt=""
                        />
                        <div class="mb-space__dyn-head-main">
                          <div class="mb-space__dyn-name">{{ displayName }}</div>
                          <div class="mb-space__dyn-subline">
                            <span class="mb-space__dyn-date">{{
                              formatDynDateCN(row.ts)
                            }}</span>
                          </div>
                        </div>
                        <div v-if="isOwnSpace" class="mb-space__dyn-head-tools">
                          <div class="mb-space__dyn-more-wrap">
                            <button
                              type="button"
                              class="mb-space__dyn-more"
                              aria-haspopup="true"
                              aria-expanded="false"
          aria-label="更多"
                            >
                              ⋮
                            </button>
                            <div class="mb-space__dyn-more-menu" role="menu">
                              <button
                                type="button"
                                class="mb-space__dyn-more-item"
                                role="menuitem"
                                @click.stop="onDynMenuPin(row)"
                              >
                                {{
                                  isDynRowPinned(row) ? "取消置顶" : "置顶"
                                }}
                              </button>
                              <button
                                type="button"
                                class="mb-space__dyn-more-item mb-space__dyn-more-item--del"
                                role="menuitem"
                                @click.stop="openDynDeleteDialog(row)"
                              >删除</button>
                            </div>
                          </div>
                        </div>
                      </header>
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
                          <p
                            v-for="(line, idx) in dynImageLegacyLines(
                              row.post
                            )"
                            :key="'l' + idx"
                            class="mb-space__dyn-textline"
                          >
                            {{ line }}
                          </p>
                          <div
                            v-if="row.post.images && row.post.images.length"
                            class="mb-space__dyn-img-row"
                          >
                            <div
                              v-for="(im, ix) in row.post.images"
                              :key="'im' + ix"
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
                          <p
                            v-for="(line, idx) in dynImageLegacyLines(
                              row.post
                            )"
                            :key="'lf' + idx"
                            class="mb-space__dyn-textline"
                          >
                            {{ line }}
                          </p>
                          <div
                            v-if="row.post.images && row.post.images.length"
                            class="mb-space__dyn-img-row"
                          >
                            <div
                              v-for="(im, ix) in row.post.images"
                              :key="'imf' + ix"
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
                              dynCommentDynamicId === Number(row.post.id)
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
                        :dynamic-author-id="userIdNum"
                        :owner-can-curate="isDynCommentVideoOwner"
                        @patch-dynamic="onDynCommentPatchDynamic"
                        @counts="onDynDynamicCommentsLiveCounts"
                        @manage-dialog="onDynPanelManageDialog"
                      />
                    </template>
                    <template v-else-if="row.kind === 'article' && row.article">
                      <header class="mb-space__dyn-head">
                        <img
                          class="mb-space__dyn-avatar"
                          :src="avatarDisplay"
                          width="48"
                          height="48"
                          alt=""
                        />
                        <div class="mb-space__dyn-head-main">
                          <div class="mb-space__dyn-name">{{ displayName }}</div>
                          <div class="mb-space__dyn-subline">
                            <span class="mb-space__dyn-date">{{
                              formatDynDateCN(row.ts)
                            }}</span>
                          </div>
                        </div>
                        <div v-if="isOwnSpace" class="mb-space__dyn-head-tools">
                          <div class="mb-space__dyn-more-wrap">
                            <button
                              type="button"
                              class="mb-space__dyn-more"
                              aria-haspopup="true"
                              aria-expanded="false"
          aria-label="更多"
                            >
                              ⋮
                            </button>
                            <div class="mb-space__dyn-more-menu" role="menu">
                              <button
                                type="button"
                                class="mb-space__dyn-more-item"
                                role="menuitem"
                                @click.stop="onDynMenuPin(row)"
                              >
                                {{
                                  isDynRowPinned(row) ? "取消置顶" : "置顶"
                                }}
                              </button>
                              <button
                                type="button"
                                class="mb-space__dyn-more-item mb-space__dyn-more-item--del"
                                role="menuitem"
                                @click.stop="openDynDeleteDialog(row)"
                              >删除</button>
                            </div>
                          </div>
                        </div>
                      </header>
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
                          @click.stop="
                            toggleDynCommentPanel(row.article.id, 'article')
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
                              formatCount(row.article.comment_count || 0)
                            }}</span></span
                          >
                        </button>
                        <button
                          type="button"
                          class="mb-space__dyn-act-bar__btn"
                          :class="{
                            'mb-space__dyn-ico-act--liked':
                              row.article.favorited_by_me
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
                        :article-author-id="userIdNum"
                        :owner-can-curate="isDynCommentVideoOwner"
                        @patch-article="onDynCommentPatchArticle"
                        @counts="onDynArticleCommentsLiveCounts"
                        @manage-dialog="onDynPanelManageDialog"
                      />
                    </template>
                    <template v-else-if="row.kind === 'video' && row.video">
                      <header class="mb-space__dyn-head">
                        <img
                          class="mb-space__dyn-avatar"
                          :src="avatarDisplay"
                          width="48"
                          height="48"
                          alt=""
                        />
                        <div class="mb-space__dyn-head-main">
                          <div class="mb-space__dyn-name">{{ displayName }}</div>
                          <div class="mb-space__dyn-subline">
                            <span class="mb-space__dyn-date">{{
                              formatDynDateCN(row.ts)
                            }}</span>
                          </div>
                        </div>
                        <div v-if="isOwnSpace" class="mb-space__dyn-head-tools">
                          <div class="mb-space__dyn-more-wrap">
                            <button
                              type="button"
                              class="mb-space__dyn-more"
                              aria-haspopup="true"
                              aria-expanded="false"
          aria-label="更多"
                            >
                              ⋮
                            </button>
                            <div class="mb-space__dyn-more-menu" role="menu">
                              <button
                                type="button"
                                class="mb-space__dyn-more-item"
                                role="menuitem"
                                @click.stop="onDynMenuPin(row)"
                              >
                                {{
                                  isDynRowPinned(row) ? "取消置顶" : "置顶"
                                }}
                              </button>
                              <button
                                type="button"
                                class="mb-space__dyn-more-item mb-space__dyn-more-item--del"
                                role="menuitem"
                                @click.stop="openDynDeleteDialog(row)"
                              >删除</button>
                            </div>
                          </div>
                        </div>
                      </header>
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
          aria-label="搜索"
                              @click.stop.prevent="
                                onWatchLaterPlaceholder($event, row.video.id)
                              "
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
                                    title="开启评论精选"
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
                            'is-on': dynCommentVideoId === row.video.id
                          }"
                                    title="开启评论精选"
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
                      <div
                        v-if="dynCommentVideoId === row.video.id"
                        class="mb-space__dyn-cmt-panel"
                        @click.stop
                      >
                        <div class="mb-space__dyn-cmt-head mb-space__dyn-cmt-head--bar">
                          <div class="mb-space__dyn-cmt-head__left">
                            <h4 class="mb-space__dyn-cmt-title">
                              评论<sup class="mb-space__dyn-cmt-count">{{
                                formatCount(row.video.comment_count || 0)
                              }}</sup>
                            </h4>
                            <div
                              v-if="!dynVideoCommentsClosed(row.video)"
                              class="mb-space__dyn-cmt-sort-row"
                              role="group"
                              :aria-label="
                                dynVideoCommentsCurated(row.video)
                                  ? '精选评论'
                                  : '评论排序'
                              "
                            >
                              <span
                                v-if="dynVideoCommentsCurated(row.video)"
                                class="mb-space__dyn-cmt-curated-label"
                              >{{ MB_COMMENT_CURATED_LABEL }}</span>
                              <template v-else>
                                <button
                                  type="button"
                                  class="mb-space__dyn-cmt-tab"
                                  :class="{ 'is-on': dynCommentSort === 'hot' }"
                                  @click="dynCommentSort = 'hot'"
                                >热门
                                </button>
                                <span
                                  class="mb-space__dyn-cmt-sep"
                                  aria-hidden="true"
                                  >|</span
                                >
                                <button
                                  type="button"
                                  class="mb-space__dyn-cmt-tab"
                                  :class="{ 'is-on': dynCommentSort === 'time' }"
                                  @click="dynCommentSort = 'time'"
                                >最新
                                </button>
                              </template>
                            </div>
                          </div>
                          <div
                            v-if="isDynCommentVideoOwner"
                            class="mb-space__dyn-cmt-head__actions"
                          >
                            <router-link
                              v-if="dynVideoCommentsCurated(row.video)"
                              class="mb-space__dyn-cmt-pending-link"
                              :to="
                                dynCreatorPendingCommentsRoute(
                                  'video',
                                  row.video.id
                                )
                              "
                              @click.stop
                            >
                              待精选评论
                            </router-link>
                            <div
                              class="mb-space__dyn-cmt-head-more-wrap vd-cmt-menu-wrap"
                              :class="{ 'is-open': dynCmtHeadMenuOpen }"
                              @click.stop
                            >
                            <button
                              type="button"
                              class="vd-cmt-menu-trigger mb-space__dyn-cmt-head-menu-trigger"
                              aria-haspopup="true"
                              :aria-expanded="dynCmtHeadMenuOpen"
                              aria-label="评论管理"
                              @click="toggleDynCmtHeadMenu($event)"
                            >
                              <span class="vd-cmt-menu-dots" aria-hidden="true">
                                <span /><span /><span />
                              </span>
                            </button>
                            <div
                              v-if="dynCmtHeadMenuOpen"
                              class="vd-cmt-menu-dropdown"
                              role="menu"
                              @click.stop
                            >
                              <button
                                type="button"
                                class="vd-cmt-menu-item"
                                role="menuitem"
                                @click="
                                  openDynMbStationDialog(
                                    dynVideoCommentsCurated(row.video)
                                      ? 'restore_pick_comment'
                                      : 'pick_comment',
                                    row.video.id
                                  )
                                "
                              >
                                {{
                                  dynVideoCommentsCurated(row.video)
                                    ? "关闭评论精选"
                                    : "开启评论精选"
                                }}
                              </button>
                              <button
                                type="button"
                                class="vd-cmt-menu-item"
                                role="menuitem"
                                @click="
                                  openDynMbStationDialog(
                                    row.video.comments_closed
                                      ? 'restore_comments'
                                      : 'close_comments',
                                    row.video.id
                                  )
                                "
                              >
                                {{
                                  row.video.comments_closed ? "恢复评论" : "关闭评论"
                                }}
                              </button>
                              <button
                                type="button"
                                class="vd-cmt-menu-item"
                                role="menuitem"
                                @click="
                                  openDynMbStationDialog(
                                    row.video.danmaku_closed
                                      ? 'restore_danmaku'
                                      : 'close_danmaku',
                                    row.video.id
                                  )
                                "
                              >
                                {{
                                  row.video.danmaku_closed ? "恢复弹幕" : "关闭弹幕"
                                }}
                              </button>
                            </div>
                            </div>
                          </div>
                        </div>
                        <template v-if="dynVideoCommentsClosed(row.video)">
                          <div class="mb-space__dyn-cmt-closed-bar" role="status">
                            UP主已关闭评论
                          </div>
                          <p class="mb-space__dyn-cmt-foot">已到达世界的尽头</p>
                        </template>
                        <template v-else>
                        <div class="vd-cmt-composer vd-cmt-composer--mb">
                          <img
                            class="vd-cmt-avatar vd-cmt-avatar--mb"
                            :src="dynComposerAvatar"
                            width="48"
                            height="48"
                            alt=""
                          />
                          <div class="vd-cmt-mb-composer-main">
                            <div class="vd-cmt-mb-editor-row">
                              <div class="vd-cmt-uni-inbox">
                                <template
                                  v-if="
                                    mbLoggedIn ||
                                      dynVideoCommentsCurated(row.video)
                                  "
                                >
                                  <textarea
                                    v-model="dynCommentDraft"
                                    class="vd-cmt-uni-inbox__field"
                                    :class="{
                                      'is-curated-hint':
                                        dynVideoCommentsCurated(row.video) &&
                                        !mbLoggedIn
                                    }"
                                    rows="3"
                                    maxlength="1000"
                                    :readonly="
                                      dynVideoCommentsCurated(row.video) &&
                                        !mbLoggedIn
                                    "
                                    :disabled="
                                      dynVideoCommentsCurated(row.video) &&
                                        !mbLoggedIn
                                    "
                                    :placeholder="
                                      dynCommentComposerPlaceholder(row.video)
                                    "
                                  />
                                </template>
                                <div
                                  v-else
                                  class="vd-cmt-uni-inbox__guest vd-cmt-login-hint"
                                >
                                  <span class="vd-cmt-login-hint-muted">请先</span>
                                  <a
                                    href="#"
                                    class="vd-cmt-login-hint-btn"
                                    @click.prevent="openMbLoginModalFromDynCmt"
                                    >登录</a
                                  >
                                  <span class="vd-cmt-login-hint-muted">后参与评论</span
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
                                    (!String(dynCommentDraft || '').trim() ||
                                      dynCommentPosting)
                                "
                                @click="submitDynComment(row.video.id)"
                              >
                                <template v-if="dynCommentPosting">发送中</template>
                                <span v-else class="vd-cmt-submit-lines"
                                  >发<br />表</span
                                >
                              </button>
                            </div>
                          </div>
                        </div>
                        <div class="mb-space__dyn-cmt-live vd-cmt-live-mount">
                          <MinibiliCommentsLive
                            ref="dynCommentsLive"
                            embedded
                            hide-composer
                            :key="'dyn-cmt-' + row.video.id"
                            :video-id="Number(row.video.id)"
                            :video-author-id="userIdNum"
                            :comment-sort="dynCommentSort"
                            :initial-comments-curated="dynVideoCommentsCurated(row.video)"
                            @counts="onDynCommentsLiveCounts"
                          />
                        </div>
                        </template>
                      </div>
                    </template>
                  </article>
                </template>
                <div
                  v-else-if="spaceSearchQuery"
                  class="mb-space__dyn-search-empty"
                >
                  <p class="mb-space__dyn-search-empty-t">
                    未找到与「<em>{{ videoSearch.trim() }}</em>」相关的动态
                  </p>
                  <button
                    type="button"
                    class="mb-space__dyn-search-empty-btn"
                    @click="videoSearch = ''"
                  >
                    清除搜索
                  </button>
                </div>
                <div
                  v-else
                  class="mb-space__empty-img"
                  role="img"
                  aria-label="暂无动态"
                >
                  <img :src="dynEmptyImg" alt="" />
                </div>
                <p v-if="dynVisibleList.length" class="mb-space__dyn-end">
                  暂没有更多动态了～
                </p>
              </div>
            </div>
          </div>
          <div
            v-else-if="activeNav === 'collect'"
            class="mb-space__collect-outer"
          >
            <div
              v-if="!canViewCollectTab"
              class="mb-space__collect-privacy-block"
            >
              <p class="mb-space__hint">对方已设置不公开收藏内容</p>
            </div>
            <template v-else>
            <aside class="mb-space__collect-sidenav" aria-label="收藏分类">
              <div class="mb-space__collect-nav-block">
                <button
                  type="button"
                  class="mb-space__collect-nav-head"
                  :class="{ 'is-open': collectFoldersOpen }"
                  :aria-expanded="collectFoldersOpen"
                  @click="collectFoldersOpen = !collectFoldersOpen"
                >
                  <span>我创建的收藏夹</span>
                  <svg
                    class="mb-space__collect-nav-chev"
                    viewBox="0 0 24 24"
                    aria-hidden="true"
                  >
                    <path fill="currentColor" d="M7 10l5 5 5-5H7z" />
                  </svg>
                </button>
                <div
                  class="mb-space__collect-folder-collapse"
                  :class="{ 'is-open': collectFoldersOpen }"
                >
                <ul class="mb-space__collect-folder-list">
                  <li v-if="isOwnSpace && mbLoggedIn" class="mb-space__collect-folder-new-li">
                    <button
                      type="button"
                      class="mb-space__collect-folder-new"
                      @click="openCollectFolderCreate"
                    >
                      <img
                        class="mb-space__collect-folder-ico"
                        :src="collectNewFolderIco"
                        width="18"
                        height="18"
                        alt=""
                      />
                      <span>新建收藏夹</span>
                    </button>
                  </li>
                  <li
                    v-for="folder in collectFolders"
                    :key="folder.id"
                    class="mb-space__collect-folder-item"
                    @mouseenter="onCollectFolderItemMouseEnter(folder.id)"
                    @mouseleave="onCollectFolderItemMouseLeave(folder.id)"
                  >
                    <button
                      type="button"
                      class="mb-space__collect-folder-btn"
                      :class="{
                        'is-on':
                          collectSideNav === 'folders' &&
                          collectFolderId === folder.id,
                      }"
                      @click="selectCollectFolder(folder.id)"
                    >
                      <img
                        class="mb-space__collect-folder-ico"
                        :src="collectFolderIco"
                        width="18"
                        height="18"
                        alt=""
                      />
                      <span class="mb-space__collect-folder-btn-title">{{
                        folder.title
                      }}</span>
                      <span
                        v-if="isOwnSpace && mbLoggedIn"
                        class="mb-space__collect-folder-trail"
                      >
                        <span
                          class="mb-space__collect-folder-count"
                          :class="{
                            'is-hidden':
                              collectFolderHoverId === folder.id ||
                              collectFolderMenuId === folder.id
                          }"
                        >{{ folder.videoCount }}</span>
                        <span
                          class="mb-space__collect-folder-more"
                          :class="{
                            'is-active':
                              collectFolderHoverId === folder.id ||
                              collectFolderMenuId === folder.id
                          }"
                          @mouseenter.stop="onCollectFolderMoreEnter(folder.id)"
                          @mouseleave.stop="onCollectFolderMoreLeave(folder.id)"
                          @click.stop
                        >
                          <span
                            class="mb-space__collect-folder-more-dots"
                            aria-hidden="true"
                          >
                            <i /><i /><i />
                          </span>
                          <ul
                            v-show="collectFolderMenuId === folder.id"
                            class="mb-space__collect-folder-menu"
                            role="menu"
                            @mouseenter.stop="onCollectFolderMoreEnter(folder.id)"
                            @mouseleave.stop="onCollectFolderMoreLeave(folder.id)"
                            @click.stop
                          >
                            <li role="none">
                              <button
                                type="button"
                                role="menuitem"
                                @click.stop="openCollectFolderEdit(folder)"
                              >
                                编辑信息
                              </button>
                            </li>
                            <li v-if="!folder.isDefault" role="none">
                              <button
                                type="button"
                                role="menuitem"
                                @click.stop="openCollectFolderDelete(folder)"
                              >
                                删除
                              </button>
                            </li>
                          </ul>
                        </span>
                      </span>
                      <span v-else class="mb-space__collect-folder-count">{{
                        folder.videoCount
                      }}</span>
                    </button>
                  </li>
                </ul>
                </div>
              </div>
              <div class="mb-space__collect-nav-block mb-space__collect-nav-block--other">
                <button
                  type="button"
                  class="mb-space__collect-nav-head is-open"
                  disabled
                >
                  <span>其他收藏</span>
                </button>
                <button
                  type="button"
                  class="mb-space__collect-folder-btn"
                  :class="{ 'is-on': collectSideNav === 'articleFav' }"
                  @click="selectCollectArticleFav"
                >
                  <img
                    class="mb-space__collect-folder-ico"
                    :src="collectFolderIco"
                    width="18"
                    height="18"
                    alt=""
                  />
                  <span class="mb-space__collect-folder-btn-title">图文收藏夹</span>
                  <span class="mb-space__collect-folder-count">{{
                    articleFavTotal
                  }}</span>
                </button>
              </div>
              <button
                type="button"
                class="mb-space__collect-later-btn"
                :class="{ 'is-on': collectSideNav === 'later' }"
                @click="selectCollectLater"
              >
                稍后再看
              </button>
            </aside>

            <div class="mb-space__collect-main">
              <template v-if="collectSideNav === 'folders'">
                <header class="mb-space__collect-head">
                  <div class="mb-space__collect-head-body">
                    <div class="mb-space__collect-folder-cover-wrap">
                      <div class="mb-space__collect-folder-cover">
                        <img
                          v-if="activeCollectFolder.coverUrl"
                          :src="activeCollectFolder.coverUrl"
                          alt=""
                        />
                        <span v-else class="mb-space__collect-folder-cover-ph" />
                      </div>
                    </div>
                    <div class="mb-space__collect-folder-info">
                      <h2 class="mb-space__collect-folder-title">
                        {{ activeCollectFolder.title }}
                      </h2>
                      <p class="mb-space__collect-folder-meta">
                        <span>{{
                          activeCollectFolder.isPublic ? "公开" : "私密"
                        }}</span>
                        <span class="mb-space__collect-folder-meta-sep">·</span>
                        <span>视频数: {{ activeCollectFolder.videoCount }}</span>
                      </p>
                      <button
                        type="button"
                        class="mb-space__play-all mb-space__play-all--collect"
                        :disabled="!collectDisplayVideos.length"
                        @click="onCollectPlayAll"
                      >
                        <svg
                          class="mb-space__play-ico"
                          viewBox="0 0 24 24"
                          aria-hidden="true"
                        >
                          <path
                            fill="currentColor"
                            d="M8 5v14l11-7L8 5zm2 4.2L15.1 12 10 14.8V9.2z"
                          />
                        </svg>
                        播放全部
                      </button>
                    </div>
                  </div>
                  <button
                    v-if="isOwnSpace && mbLoggedIn && !collectBatchMode"
                    type="button"
                    class="mb-space__collect-batch"
                    :disabled="collectFolderId == null"
                    @click="enterCollectBatchMode"
                  >
                    批量操作
                  </button>
                  <button
                    v-else-if="collectBatchMode"
                    type="button"
                    class="mb-space__collect-back"
                    @click="exitCollectBatchMode"
                  >
                    返回
                  </button>
                </header>

                <div
                  v-if="collectBatchMode"
                  class="mb-space__collect-batch-bar"
                >
                  <div class="mb-space__collect-batch-left">
                    <label class="mb-space__collect-batch-check">
                      <input
                        type="checkbox"
                        :checked="collectBatchAllSelected"
                        @change="toggleCollectBatchSelectAll"
                      />
                      <span>全选</span>
                    </label>
                    <span class="mb-space__collect-batch-count"
                      >已选择 {{ collectBatchSelectedCount }} 个视频</span
                    >
                  </div>
                  <div class="mb-space__collect-batch-actions">
                    <button
                      type="button"
                      class="mb-space__collect-batch-act"
                      :disabled="collectBatchActionSaving"
                      @click="openCollectClearInvalid"
                    >
                      清除失效内容
                    </button>
                    <button
                      type="button"
                      class="mb-space__collect-batch-act"
                      :disabled="
                        !collectBatchSelectedCount || collectBatchActionSaving
                      "
                      @click="onCollectBatchUnfavorite"
                    >
                      取消收藏
                    </button>
                    <button
                      type="button"
                      class="mb-space__collect-batch-act"
                      :disabled="
                        !collectBatchSelectedCount || collectBatchActionSaving
                      "
                      @click="onCollectBatchCopyTo"
                    >
                      <svg viewBox="0 0 24 24" aria-hidden="true">
                        <path
                          fill="currentColor"
                          d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"
                        />
                      </svg>
                      复制至
                    </button>
                    <button
                      type="button"
                      class="mb-space__collect-batch-act"
                      :disabled="
                        !collectBatchSelectedCount || collectBatchActionSaving
                      "
                      @click="onCollectBatchMoveTo"
                    >
                      <svg viewBox="0 0 24 24" aria-hidden="true">
                        <path
                          fill="currentColor"
                          d="M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"
                        />
                      </svg>
                      移动至
                    </button>
                  </div>
                </div>

                <div v-else class="mb-space__collect-toolbar">
                  <div
                    class="mb-space__subtabs mb-space__collect-subtabs"
                    role="group"
                    aria-label="收藏排序"
                  >
                    <button
                      type="button"
                      class="mb-space__subtab"
                      :class="{ 'is-on': collectSort === 'recent' }"
                      @click="collectSort = 'recent'"
                    >
                      最近收藏
                    </button>
                    <button
                      type="button"
                      class="mb-space__subtab"
                      :class="{ 'is-on': collectSort === 'play' }"
                      @click="collectSort = 'play'"
                    >
                      最多播放
                    </button>
                    <button
                      type="button"
                      class="mb-space__subtab"
                      :class="{ 'is-on': collectSort === 'submit' }"
                      @click="collectSort = 'submit'"
                    >
                      最近投稿
                    </button>
                  </div>
                  <div class="mb-space__collect-search" role="search">
                    <div class="mb-space__collect-search-group">
                      <label class="mb-space__collect-search-scope">
                        <select
                          v-model="collectSearchScope"
                          class="mb-space__collect-search-select"
                        >
                          <option value="current">当前收藏夹</option>
                          <option value="all">全部收藏夹</option>
                        </select>
                        <svg
                          class="mb-space__collect-search-chev"
                          viewBox="0 0 24 24"
                          aria-hidden="true"
                        >
                          <path fill="currentColor" d="M7 10l5 5 5-5H7z" />
                        </svg>
                      </label>
                      <input
                        v-model.trim="collectSearchKeyword"
                        type="search"
                        class="mb-space__collect-search-input"
                        placeholder="请输入关键词"
                        autocomplete="off"
                      />
                      <button
                        type="button"
                        class="mb-space__collect-search-btn"
                        aria-label="搜索"
                        disabled
                      >
                        <svg viewBox="0 0 24 24" aria-hidden="true">
                          <path
                            fill="currentColor"
                            d="M15.5 14h-.79l-.28-.27A6.471 6.471 0 0016 9.5 6.5 6.5 0 109.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"
                          />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>

                <div class="mb-space__collect-feed" aria-label="收藏视频列表">
                  <p v-if="collectLoading" class="mb-space__hint">加载中…</p>
                  <ul
                    v-else-if="collectDisplayVideos.length"
                    :class="[
                      'mb-space__video-grid',
                      'mb-space__collect-grid',
                      { 'is-batch': collectBatchMode }
                    ]"
                  >
                    <li
                      v-for="v in collectDisplayVideos"
                      :key="'fav-' + v.id"
                      class="mb-space__vcell"
                      :class="{
                        'is-collect-menu-open': collectVideoMenuId === v.id,
                        'is-batch-selected':
                          collectBatchMode && isCollectBatchSelected(v.id)
                      }"
                    >
                      <div class="mb-space__collect-vthumb-shell">
                        <label
                          v-if="collectBatchMode"
                          class="mb-space__collect-batch-cell-check"
                          @click.stop.prevent="toggleCollectBatchSelect(v.id)"
                        >
                          <input
                            type="checkbox"
                            class="mb-space__collect-batch-cell-check-input"
                            :checked="isCollectBatchSelected(v.id)"
                            tabindex="-1"
                            @click.stop
                          />
                          <span
                            class="mb-space__collect-batch-cell-check-box"
                            aria-hidden="true"
                          />
                        </label>
                        <router-link
                          v-if="!collectBatchMode"
                          class="mb-space__vcell-link mb-space__vcell-link--collect-thumb"
                          :to="minibiliVideoPlayRoute(v.id)"
                        >
                          <div class="mb-space__vthumb-wrap">
                            <img
                            class="mb-space__vthumb"
                            :src="v.cover_url || akari"
                            :alt="v.title"
                          />
                          <div class="mb-space__vthumb-default" aria-hidden="true">
                            <div class="mb-space__vthumb-stats-l">
                              <span class="mb-space__vthumb-stat">
                                <img
                                  class="mb-space__vstat-ico"
                                  :src="thumbPlayIco"
                                  alt=""
                                />
                                {{ formatCount(v.play_count) }}
                              </span>
                              <span class="mb-space__vthumb-stat">
                                <img
                                  class="mb-space__vstat-ico"
                                  :src="thumbDanmuIco"
                                  alt=""
                                />
                                {{ formatCount(v.danmaku_count) }}
                              </span>
                            </div>
                            <span class="mb-space__vdur">{{
                              formatDuration(v.duration)
                            }}</span>
                          </div>
                          <button
                            type="button"
                            class="mb-space__vthumb-later"
                            aria-label="稍后再看"
                            @click.stop.prevent="onWatchLaterPlaceholder($event, v.id)"
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
                      </router-link>
                        <div
                          v-else
                          class="mb-space__vcell-link mb-space__vcell-link--collect-thumb mb-space__vcell-link--batch"
                          role="button"
                          tabindex="0"
                          @click="toggleCollectBatchSelect(v.id)"
                          @keydown.enter.prevent="toggleCollectBatchSelect(v.id)"
                        >
                          <div class="mb-space__vthumb-wrap">
                            <img
                              class="mb-space__vthumb"
                              :src="v.cover_url || akari"
                              :alt="v.title"
                            />
                            <div class="mb-space__vthumb-default" aria-hidden="true">
                              <div class="mb-space__vthumb-stats-l">
                                <span class="mb-space__vthumb-stat">
                                  <img
                                    class="mb-space__vstat-ico"
                                    :src="thumbPlayIco"
                                    alt=""
                                  />
                                  {{ formatCount(v.play_count) }}
                                </span>
                                <span class="mb-space__vthumb-stat">
                                  <img
                                    class="mb-space__vstat-ico"
                                    :src="thumbDanmuIco"
                                    alt=""
                                  />
                                  {{ formatCount(v.danmaku_count) }}
                                </span>
                              </div>
                              <span class="mb-space__vdur">{{
                                formatDuration(v.duration)
                              }}</span>
                            </div>
                          </div>
                        </div>
                      </div>
                      <div class="mb-space__vtext-col">
                        <div class="mb-space__vtitle-row">
                          <router-link
                            v-if="!collectBatchMode"
                            class="mb-space__vtitle mb-space__vtitle--collect-link"
                            :to="minibiliVideoPlayRoute(v.id)"
                            :title="v.title"
                          >
                            {{ v.title }}
                          </router-link>
                          <span
                            v-else
                            class="mb-space__vtitle mb-space__vtitle--collect-batch"
                            :title="v.title"
                            role="button"
                            tabindex="0"
                            @click="toggleCollectBatchSelect(v.id)"
                            @keydown.enter.prevent="toggleCollectBatchSelect(v.id)"
                          >{{ v.title }}</span>
                            <div
                              v-if="isOwnSpace && mbLoggedIn && !collectBatchMode"
                              class="mb-space__vmore-wrap"
                            >
                              <button
                                type="button"
                                class="mb-space__vmore-btn"
                                aria-label="更多操作"
                                @click.stop.prevent="toggleCollectVideoMenu(v.id)"
                              >
                                <span
                                  class="mb-space__vmore-ico"
                                  aria-hidden="true"
                                >&#8942;</span>
                              </button>
                              <ul
                                v-if="collectVideoMenuId === v.id"
                                class="mb-space__vmore-menu"
                                role="menu"
                                @click.stop
                              >
                                <li role="none">
                                  <button
                                    type="button"
                                    role="menuitem"
                                    @click.stop.prevent="onCollectUnfavorite(v)"
                                  >
                                    取消收藏
                                  </button>
                                </li>
                                <li role="none">
                                  <button
                                    type="button"
                                    role="menuitem"
                                    @click.stop.prevent="onCollectCopyTo(v)"
                                  >
                                    复制至
                                  </button>
                                </li>
                                <li role="none">
                                  <button
                                    type="button"
                                    role="menuitem"
                                    @click.stop.prevent="onCollectMoveTo(v)"
                                  >
                                    移动至
                                  </button>
                                </li>
                              </ul>
                            </div>
                          </div>
                          <p class="mb-space__vmeta mb-space__vmeta--collect">
                            <span class="mb-space__vmeta-up">
                              <img
                                class="mb-space__collect-up-ico"
                                :src="collectUpIco"
                                width="28"
                                height="14"
                                alt=""
                              />
                              {{ v.uploader }}
                            </span>
                            <span class="mb-space__vmeta-sep" aria-hidden="true">·</span>
                            <span
                              v-if="formatFavoritedMD(v.favorited_at)"
                              class="mb-space__collect-fav-at"
                            >收藏于{{ formatFavoritedMD(v.favorited_at) }}</span>
                          </p>
                        </div>
                    </li>
                  </ul>
                  <div
                    v-else
                    class="mb-space__empty-img"
                    role="img"
                    aria-label="暂无收藏视频"
                  >
                    <img :src="dynEmptyImg" alt="" />
                  </div>
                </div>
              </template>

              <template v-else-if="collectSideNav === 'articleFav'">
                <header class="mb-space__collect-head mb-space__collect-head--later">
                  <h2 class="mb-space__collect-later-title">图文收藏夹</h2>
                </header>
                <ul
                  v-if="articleFavItems.length"
                  class="mb-space__video-grid mb-space__collect-grid mb-space__article-fav-grid"
                  aria-label="图文收藏列表"
                >
                  <li
                    v-for="art in articleFavItems"
                    :key="'af-' + art.id"
                    class="mb-space__vcell mb-space__vcell--article-fav"
                  >
                    <router-link
                      v-if="!art.unavailable && minibiliArticleReadRoute(art.id)"
                      class="mb-space__vcell-link"
                      :to="minibiliArticleReadRoute(art.id)"
                    >
                      <div class="mb-space__vthumb-wrap mb-space__vthumb-wrap--article-fav">
                        <img
                          class="mb-space__vthumb"
                          :src="art.cover_url || akari"
                          :alt="art.title"
                        />
                      </div>
                      <div class="mb-space__vtext-col">
                        <p class="mb-space__vtitle" :title="art.title">
                          {{ art.title }}
                        </p>
                        <p class="mb-space__vmeta mb-space__vmeta--article-fav">
                          <span class="mb-space__vmeta-up">{{ art.author_name || "专栏" }}</span>
                          <template v-if="formatArticlePubDate(art.published_at || art.created_at)">
                            <span class="mb-space__vmeta-sep" aria-hidden="true">·</span>
                            <span>{{
                              formatArticlePubDate(art.published_at || art.created_at)
                            }}</span>
                          </template>
                        </p>
                      </div>
                    </router-link>
                    <div
                      v-else
                      class="mb-space__vcell-link mb-space__vcell-link--unavailable"
                    >
                      <div class="mb-space__vthumb-wrap">
                        <img
                          class="mb-space__vthumb"
                          :src="dynEmptyImg"
                          :alt="art.title"
                        />
                      </div>
                      <div class="mb-space__vtext-col">
                        <p class="mb-space__vtitle" :title="art.title">
                          {{ art.title }}
                        </p>
                        <p class="mb-space__vmeta mb-space__vmeta--article-fav">
                          专栏已不可用
                        </p>
                      </div>
                    </div>
                  </li>
                </ul>
                <p
                  v-else-if="articleFavLoading"
                  class="mb-space__hint"
                >
                  加载中…
                </p>
                <p
                  v-else-if="articleFavTotal > 0"
                  class="mb-space__hint"
                >
                  收藏的专栏暂无法展示（可能已删除或未发布）
                </p>
                <div v-else class="mb-space__empty-img mb-space__collect-later-empty" role="img">
                  <img :src="dynEmptyImg" alt="" />
                  <p class="mb-space__collect-later-hint">还没有收藏的专栏</p>
                </div>
              </template>

              <template v-else>
                <header
                  class="mb-space__collect-head mb-space__collect-head--later"
                >
                  <h2 class="mb-space__collect-later-title">稍后再看</h2>
                  <button
                    type="button"
                    class="mb-space__play-all mb-space__play-all--collect"
                    disabled
                  >
                    <svg
                      class="mb-space__play-ico"
                      viewBox="0 0 24 24"
                      aria-hidden="true"
                    >
                      <path
                        fill="currentColor"
                        d="M8 5v14l11-7L8 5zm2 4.2L15.1 12 10 14.8V9.2z"
                      />
                    </svg>
                    播放全部
                  </button>
                </header>
                <p v-if="watchLaterLoading" class="mb-space__hint">加载中…</p>
                <ul
                  v-else-if="watchLaterVideos.length"
                  class="mb-space__video-grid mb-space__collect-grid"
                  aria-label="稍后再看视频列表"
                >
                  <li
                    v-for="v in watchLaterVideos"
                    :key="'wl-' + v.id"
                    class="mb-space__vcell"
                  >
                    <router-link
                      class="mb-space__vcell-link"
                      :to="minibiliVideoPlayRoute(v.id)"
                    >
                      <div class="mb-space__vthumb-wrap">
                        <img
                          class="mb-space__vthumb"
                          :src="v.cover_url || akari"
                          :alt="v.title"
                        />
                        <div class="mb-space__vthumb-default" aria-hidden="true">
                          <div class="mb-space__vthumb-stats-l">
                            <span class="mb-space__vthumb-stat">
                              <img
                                class="mb-space__vstat-ico"
                                :src="thumbPlayIco"
                                alt=""
                              />
                              {{ formatCount(v.play_count) }}
                            </span>
                            <span class="mb-space__vthumb-stat">
                              <img
                                class="mb-space__vstat-ico"
                                :src="thumbDanmuIco"
                                alt=""
                              />
                              {{ formatCount(v.danmaku_count) }}
                            </span>
                          </div>
                          <span class="mb-space__vdur">{{
                            formatDuration(v.duration)
                          }}</span>
                        </div>
                      </div>
                      <div class="mb-space__vtext-col">
                        <p class="mb-space__vtitle" :title="v.title">
                          {{ v.title }}
                        </p>
                        <p class="mb-space__vmeta">{{ v.uploader }}</p>
                      </div>
                    </router-link>
                  </li>
                </ul>
                <div
                  v-else
                  class="mb-space__empty-img mb-space__collect-later-empty"
                  role="img"
                  aria-label="稍后再看暂无内容"
                >
                  <img :src="dynEmptyImg" alt="" />
                  <p class="mb-space__collect-later-hint">
                    {{ isOwnSpace ? "还没有稍后再看的视频" : "仅可在自己的空间查看稍后再看" }}
                  </p>
                </div>
              </template>
            </div>
            </template>
          </div>

          <div
            v-else-if="activeNav === 'settings'"
            class="mb-space__settings-panel"
          >
            <section
              v-if="isOwnSpace && mbLoggedIn"
              class="mb-space-privacy"
              aria-label="隐私设置"
            >
              <h2 class="mb-space-privacy__title">隐私设置</h2>
              <p v-if="spacePrivacyLoading" class="mb-space-privacy__hint">
                加载中…
              </p>
              <div v-else class="mb-space-privacy__grid">
                <div
                  v-for="(col, colIdx) in privacySettingColumns"
                  :key="'privacy-col-' + colIdx"
                  class="mb-space-privacy__col"
                >
                  <div
                    v-for="item in col"
                    :key="item.key"
                    class="mb-space-privacy__row"
                  >
                    <span class="mb-space-privacy__label">{{ item.label }}</span>
                    <button
                      type="button"
                      class="mb-space-privacy__switch"
                      :class="{ 'is-on': spacePrivacy[item.key] }"
                      :disabled="spacePrivacySaving"
                      :aria-label="item.label"
                      :aria-pressed="spacePrivacy[item.key] ? 'true' : 'false'"
                      @click="onToggleSpacePrivacy(item.key)"
                    />
                  </div>
                </div>
              </div>
            </section>
            <div
              v-else
              class="mb-space__empty-img"
              role="img"
              aria-label="暂无权限"
            >
              <img :src="dynEmptyImg" alt="" />
            </div>
          </div>
        </main>

        <aside
          v-if="activeNav !== 'contribute' && activeNav !== 'collect'"
          class="mb-space__aside"
          aria-label="搜索"
        >
          <section class="mb-space__card mb-space__card--creator">
            <router-link
              class="mb-space__creator-top"
              :to="{ name: 'upload' }"
            >
              <span class="mb-space__creator-top-inner">
                <span class="mb-space__creator-bulb-wrap" aria-hidden="true">
                  <img
                    class="mb-space__creator-bulb-img"
                    :src="bulbUrl"
                    alt=""
                  />
                </span>
                <span class="mb-space__creator-top-t">创作中心</span>
                <span class="mb-space__creator-chev" aria-hidden="true">›</span>
              </span>
            </router-link>
            <div class="mb-space__creator-split-row">
              <router-link
                v-if="isOwnSpace"
                class="mb-space__creator-split-cell"
                :to="{ name: 'videoPublish' }"
              >
                <img
                  class="mb-space__creator-cell-ico mb-space__creator-cell-ico--invert"
                  :src="creatorSbUpload"
                  alt=""
                />
                <span>视频投稿</span>
              </router-link>
              <span v-else class="mb-space__creator-split-cell is-muted">
                <img
                  class="mb-space__creator-cell-ico mb-space__creator-cell-ico--invert"
                  :src="creatorSbUpload"
                  alt=""
                />
                <span>视频投稿</span>
              </span>
              <span class="mb-space__creator-split-v" aria-hidden="true" />
              <router-link
                v-if="isOwnSpace"
                class="mb-space__creator-split-cell"
                :to="{ name: 'manuscript' }"
              >
                <img
                  class="mb-space__creator-cell-ico mb-space__creator-cell-ico--invert"
                  :src="creatorSbContent"
                  alt=""
                />
                <span>投稿管理</span>
              </router-link>
              <span v-else class="mb-space__creator-split-cell is-muted">
                <img
                  class="mb-space__creator-cell-ico mb-space__creator-cell-ico--invert"
                  :src="creatorSbContent"
                  alt=""
                />
                <span>投稿管理</span>
              </span>
            </div>
          </section>

          <section class="mb-space__card mb-space__card--notice">
              <h3 class="mb-space__card-title">公告</h3>
            <div v-if="isOwnSpace" class="mb-space__notice-shell" :class="{ 'is-notice-focused': noticeFocused }">
              <textarea
                v-model="noticeDraft"
                class="mb-space__notice-textarea"
                rows="7"
                placeholder="编辑我的公告"
                spellcheck="false"
                @input="onNoticeInput"
                @focus="onNoticeFocus"
                @blur="onNoticeBlur"
              />
              <span v-show="noticeFocused" class="mb-space__notice-count">{{ noticeCharCount }}/150</span>
            </div>
            <template v-else>
              <p
                v-if="visitorNoticeText"
                class="mb-space__notice-display"
              >
                {{ visitorNoticeText }}
              </p>
              <div
                v-else
                class="mb-space__empty-img mb-space__empty-img--notice"
                role="img"
          aria-label="暂无公告"
              >
                <img :src="dynEmptyImg" alt="" />
              </div>
            </template>
          </section>

          <section class="mb-space__card mb-space__card--profile">
            <div class="mb-space__card-title-row">
              <h3 class="mb-space__card-title">个人资料</h3>
              <router-link
                v-if="isOwnSpace"
                class="mb-space__card-edit"
                :to="{ path: '/minibili/account', query: { tab: 'info' } }"
              >
                编辑
                <span class="mb-space__creator-chev" aria-hidden="true">›</span>
              </router-link>
            </div>
            <div class="mb-space__profile-row">
              <img
                class="mb-space__profile-ico mb-space__profile-ico--invert"
                :src="profileUidIco"
                alt=""
              />
              <span class="mb-space__profile-val">{{ asideUidText }}</span>
            </div>
            <div class="mb-space__profile-row">
              <img
                class="mb-space__profile-ico"
                :src="profileCakeIco"
                alt=""
              />
              <span class="mb-space__profile-val">{{ asideBirthText }}</span>
            </div>
          </section>
        </aside>
        </div>
      </div>
    </div>
  </div>
  <Teleport to="body">
    <div
      v-if="noticeSuccessVisible"
      class="mb-space__notice-toast-overlay"
      aria-live="polite"
    >
      <div class="mb-space__notice-toast-box">公告保存成功</div>
    </div>
  </Teleport>
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
          <p class="mb-space__dyn-del-sub">
            删除后不可恢复，是否继续？
          </p>
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

  <MbCollectFolderCreateDialog
    v-model="collectFolderCreateOpen"
    :mode="collectFolderDialogMode"
    :initial="collectFolderEditInitial"
    :loading="collectFolderCreateSaving"
    @submit="onCollectFolderCreateSubmit"
  />

  <MbStationDialog
    v-model="collectFolderDeleteOpen"
    title="确认提示"
    message="确定删除这个收藏夹吗？"
    :loading="collectFolderDeleteSaving"
    @confirm="onCollectFolderDeleteConfirm"
    @cancel="collectFolderDeleteOpen = false"
  />

  <MbCollectVideoFolderTransferDialog
    v-model="collectTransferOpen"
    :mode="collectTransferMode"
    :from-folder-id="collectFolderId"
    :video-count="collectTransferVideoCount"
    :loading="collectTransferSaving"
    @confirm="onCollectTransferConfirm"
    @cancel="onCollectTransferClose"
    @folder-created="onCollectTransferFolderCreated"
  />

  <MbStationDialog
    v-model="collectClearInvalidOpen"
    title="清除失效内容"
    message="是否一键清除当前文件夹所有失效内容？"
    :loading="collectClearInvalidSaving"
    @confirm="onCollectClearInvalidConfirm"
    @cancel="collectClearInvalidOpen = false"
  />

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
import { createNamespacedHelpers } from "vuex";
import bannerUrl from "@/assets/personal_space/1sz3p8w2Sk.png@3840w_400h_1c_100q.avif";
import bulbUrl from "@/assets/personal_space/bulb.png";
import creatorSbUpload from "@/assets/update.png";
import creatorSbContent from "@/assets/contentManagement.png";
import iconHome from "@/assets/home.png";
import iconUpdate from "@/assets/personal_space/update.png";
import iconContribute from "@/assets/personal_space/contribute.png";
import iconCollect from "@/assets/personal_space/collect.png";
import iconSet from "@/assets/personal_space/set.png";
import profileUidIco from "@/assets/personal_space/UID.png";
import profileCakeIco from "@/assets/personal_space/cake.png";
import thumbPlayIco from "@/assets/personal_space/play.png";
import thumbDanmuIco from "@/assets/personal_space/danmu.png";
import thumbLaterIco from "@/assets/personal_space/latertowatch.png";
import genderMaleIco from "@/assets/personal_space/male.png";
import genderFemaleIco from "@/assets/personal_space/female.png";
import dynPinTopIco from "@/assets/personal_space/top.png";
import shareIco from "@/assets/personal_space/share.png";
import dynCollectIco from "@/assets/text/collect.png";
import collectUpIco from "@/assets/personal_space/UP.png";
import collectFolderIco from "@/assets/personal_space/file.png";
import collectNewFolderIco from "@/assets/personal_space/new_file.png";
import akari from "@/assets/akari.jpg";
import dynEmptyImg from "@/assets/empty_2.png";
import MbStationDialog from "@/components/minibili/MbStationDialog.vue";
import MbSpaceHeaderActions from "@/components/minibili/MbSpaceHeaderActions.vue";
import MbSpacePerspective from "@/components/minibili/MbSpacePerspective.vue";
import MbSpacePerspectivePicker from "@/components/minibili/MbSpacePerspectivePicker.vue";
import MbCollectVideoFolderTransferDialog from "@/components/minibili/MbCollectVideoFolderTransferDialog.vue";
import MbCollectFolderCreateDialog from "@/components/minibili/MbCollectFolderCreateDialog.vue";
import MinibiliCommentsLive from "./MinibiliCommentsLive.vue";
import MbDynVideoCommentPanel from "@/components/minibili/MbDynVideoCommentPanel.vue";
import {
  MB_COMMENT_CURATED_LABEL,
  MB_COMMENT_CURATED_PLACEHOLDER
} from "@/constants/minibiliComments";
import {
  mbGetUserPublic,
  mbListUserPublishedVideos,
  mbListUserPublishedArticles,
  mbListMyArticleFavorites,
  mbListUserArticleFavorites,
  mbListComments,
  mbPostComment,
  mbPutMeAnnouncement,
  mbToggleVideoLike,
  mbToggleWatchLater,
  mbListMyWatchLater,
  mbListMyFavorites,
  mbListUserFavorites,
  mbListMyFavoriteFolders,
  mbListUserFavoriteFolders,
  mbListUserRecentCoinVideos,
  mbGetMeSpacePrivacy,
  mbPutMeSpacePrivacy,
  mbCreateFavoriteFolder,
  mbUpdateFavoriteFolder,
  mbDeleteFavoriteFolder,
  mbClearInvalidFavoritesInFolder,
  mbBatchRemoveVideosFromFavoriteFolder,
  mbRemoveVideoFromFavoriteFolder,
  mbCopyVideoToFavoriteFolder,
  mbMoveVideoFavoriteFolder,
  mbDeleteMyVideo,
  mbDeleteMyArticle,
  mbDeleteMyDynamic,
  mbListUserPublishedDynamics,
  mbToggleDynamicLike,
  mbToggleArticleFavorite,
  mbPatchVideoPlayback,
  mbPatchArticlePlayback,
  mbPatchDynamicPlayback
} from "@/api/minibili";
import { getAccessToken, getUserId } from "@/utils/authTokens";
import {
  minibiliUserSpaceRelationsRoute,
  minibiliVideoPlayRoute,
  minibiliArticleReadRoute,
  minibiliDynamicReadRoute
} from "@/utils/minibiliRoutes";
import { ElMessage } from "element-plus";
import { showMbDarkToast } from "@/utils/mbToast";
import { extractApiErrorMessage } from "@/utils/apiErrorMessage";
import { levelIconUrl } from "@/utils/userLevel";
import { personalSpaceZhCN } from "@/i18n/personalSpace.zh-CN";
import {
  buildSpaceViewerProfile,
  isSpacePerspectivePreviewMode,
  writeStoredSpacePerspective
} from "@/utils/spacePerspective";

const { mapState, mapActions } = createNamespacedHelpers("login");

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
  close_danmaku: {
    title: "关闭评论",
    message:
      "关闭评论将会禁止任何在此评论区发表内容，且已有评论在关闭期间将不可见"
  },
  restore_danmaku: {
    title: "关闭评论",
    message:
      "恢复评论后，用户可正常发表评论、参与评论互动，已有的评论也恢复正常展示"
  },
  restore_pick_comment: {
    title: "关闭评论精选",
    message: "关闭后，新评论将直接对所有人可见，无需再经过精选确认。"
  }
};

const DEFAULT_SIGN = "这个家伙很懒，什么都没有写";

export default {
  name: "MinibiliPersonalSpace",
  components: {
    MbStationDialog,
    MbSpaceHeaderActions,
    MbSpacePerspective,
    MbSpacePerspectivePicker,
    MinibiliCommentsLive,
    MbDynVideoCommentPanel,
    MbCollectFolderCreateDialog,
    MbCollectVideoFolderTransferDialog
  },
  data() {
    return {
      MB_COMMENT_CURATED_LABEL,
      bannerUrl,
      bulbUrl,
      creatorSbUpload,
      creatorSbContent,
      profileUidIco,
      profileCakeIco,
      thumbPlayIco,
      thumbDanmuIco,
      thumbLaterIco,
      genderMaleIco,
      genderFemaleIco,
      dynPinTopIco,
      shareIco,
      dynCollectIco,
      collectUpIco,
      akari,
      dynEmptyImg,
      profile: null,
      videos: [],
      nextCursor: "",
      listLoading: false,
      profileLoading: false,
      loadError: "",
      activeNav: "home",
      videoSort: "new",
      videoSearch: "",
      statFollowing: 0,
      statFans: 0,
      spaceFollowedByMe: false,
      statLikes: 0,
      noticeDraft: "",
      noticeSaved: "",
      noticeSaving: false,
      noticeSuccessVisible: false,
      noticeFocused: false,
      _noticeSuccessTid: null,
      _spaceSearchNavTimer: null,
      dynamicSubtab: "all",
      imageDynamics: [],
      /** 动态评论面板当前视频 id，null 为关闭 */
      dynCommentVideoId: null,
      dynCommentArticleId: null,
      dynCommentDynamicId: null,
      dynCommentSort: "hot",
      dynCommentDraft: "",
      dynCommentPosting: false,
      dynCmtHeadMenuOpen: false,
      dynMbStationOpen: false,
      dynMbStationTitle: "",
      dynMbStationMessage: "",
      dynMbStationLoading: false,
      dynMbStationKind: null,
      dynMbStationVideoId: null,
      dynMbStationArticleId: null,
      dynMbStationDynamicId: null,
      /** 动态列表 feed 行 id，如 dyn-v-1 / dyn-i-abc */
      dynPinnedRowIds: [],
      dynDeleteTarget: null,
      dynDeleteSubmitting: false,
      /** 图文投稿占位数量（后续接专栏） */
      spaceArticles: [],
      spaceArticlesLoading: false,
      /** 投稿子页 | 图文 */
      contribSubtab: "video",
      /** 空间视频展示 | 列表 */
      spaceVideoViewMode: "grid",
      collectFoldersOpen: true,
      collectSideNav: "folders",
      collectFolderId: null,
      collectSort: "recent",
      collectSearchScope: "current",
      collectSearchKeyword: "",
      collectFolders: [],
      collectVideos: [],
      collectLoading: false,
      collectFoldersLoading: false,
      collectFolderCreateOpen: false,
      collectFolderCreateSaving: false,
      collectFolderDialogMode: "create",
      collectFolderEditTarget: null,
      collectFolderHoverId: null,
      collectFolderMenuId: null,
      _collectFolderMenuCloseTimer: null,
      collectFolderDeleteOpen: false,
      collectFolderDeleteTarget: null,
      collectFolderDeleteSaving: false,
      collectVideoMenuId: null,
      collectTransferOpen: false,
      collectTransferMode: "copy",
      collectTransferVideoId: null,
      collectTransferSaving: false,
      collectBatchMode: false,
      collectBatchSelectedIds: [],
      collectBatchActionSaving: false,
      collectClearInvalidOpen: false,
      collectClearInvalidSaving: false,
      collectFolderIco,
      collectNewFolderIco,
      watchLaterVideos: [],
      watchLaterLoading: false,
      articleFavItems: [],
      articleFavTotal: 0,
      articleFavLoading: false,
      homeFolders: [],
      homeFoldersTotal: 0,
      homeFoldersHidden: 0,
      homeFoldersLoading: false,
      homeRecentCoins: [],
      homeRecentCoinsTotal: 0,
      homeRecentCoinsLoading: false,
      spacePrivacy: {
        public_favorites: false,
        public_recent_coins: false,
        public_following: false,
        public_fans: false,
        public_birthday: true
      },
      spacePrivacyLoading: false,
      spacePrivacySaving: false,
      spacePerspective: "self",
      _profileOwnerSnapshot: null
    };
  },
  computed: {
    ...mapState({
      proInfo: (s) => s.proInfo,
      minibiliMe: (s) => s.minibiliMe
    }),
    userIdNum() {
      const raw = this.$route.params.userId;
      const n = parseInt(String(raw || "").trim(), 10);
      return Number.isFinite(n) && n > 0 ? n : 0;
    },
    navProfileRecord() {
      const p = this.proInfo;
      return p && typeof p === "object" && !Array.isArray(p) ? p : null;
    },
    isOwnSpace() {
      return this.isSpaceOwner;
    },
    isPerspectivePreview() {
      return (
        this.isRealSpaceOwner &&
        isSpacePerspectivePreviewMode(this.spacePerspective)
      );
    },
    /** 真实登录用户是否为该空间 UP 主 */
    isRealSpaceOwner() {
      const uid = this.userIdNum;
      if (!uid) {
        return false;
      }
      const me = getUserId();
      if (me != null && Number(me) === uid) {
        return true;
      }
      const p = this.profile;
      return !!(
        p &&
        Number(p.user_id) === uid &&
        p.is_owner === true
      );
    },
    /** 当前页面是否按 UP 主权限展示（预览模式下视为访客） */
    isSpaceOwner() {
      if (this.isPerspectivePreview) {
        return false;
      }
      return this.isRealSpaceOwner;
    },
    headerActionsFollowedByMe() {
      if (this.spacePerspective === "fan" && this.isRealSpaceOwner) {
        return true;
      }
      if (this.spacePerspective === "visitor" && this.isRealSpaceOwner) {
        return false;
      }
      return this.spaceFollowedByMe;
    },
    canViewCollectTab() {
      if (this.isRealSpaceOwner && !this.isPerspectivePreview) {
        return true;
      }
      const priv = this.spacePrivacyView;
      return !!(priv && priv.public_favorites);
    },
    /**
     * 侧栏展示用：UP 主 UID、生日等
     * 供 MinibiliCommentsLive 的 videoAuthorId 使用
     */
    isDynCommentVideoOwner() {
      if (this.isPerspectivePreview) {
        return false;
      }
      const me = getUserId();
      if (me == null || !this.userIdNum) {
        return false;
      }
      return Number(this.userIdNum) === Number(me);
    },
    displayName() {
      if (this.profile && String(this.profile.nickname || "").trim()) {
        return String(this.profile.nickname).trim();
      }
      return this.userIdNum ? "UID " + this.userIdNum : "?";
    },
    displaySign() {
      if (this.profile && String(this.profile.sign || "").trim()) {
        return String(this.profile.sign).trim();
      }
      return DEFAULT_SIGN;
    },
    /** 优先用 profile，否则回退 Vuex minibiliMe */
    spaceGenderKey() {
      const p =
        this.profile && typeof this.profile === "object" && !Array.isArray(this.profile)
          ? this.profile
          : null;
      let g = p && p.gender != null ? String(p.gender).trim() : "";
      if (
        g !== "male" &&
        g !== "female" &&
        g !== "secret" &&
        this.isOwnSpace &&
        this.minibiliMe &&
        this.minibiliMe.gender != null
      ) {
        g = String(this.minibiliMe.gender).trim();
      }
      if (g === "male" || g === "female" || g === "secret") {
        return g;
      }
      return "secret";
    },
    spaceGenderLabel() {
      const k = this.spaceGenderKey;
      if (k === "male") {
        return "";
      }
      if (k === "female") {
        return "";
      }
        return "";
    },
    avatarDisplay() {
      const u = this.profile && String(this.profile.avatar_url || "").trim();
      if (u) {
        return u;
      }
      return akari;
    },
    mbLoggedIn() {
      if (this.isRealSpaceOwner && this.spacePerspective === "visitor") {
        return false;
      }
      return !!getAccessToken();
    },
    /** 仅 nav/side 变化时触发 applySpaceRouteQuery，避免 deep $route 导致收藏列表反复 loading */
    spaceRouteSig() {
      const q = this.$route.query || {};
      return [
        this.$route.name,
        this.$route.params.userId,
        q.nav,
        q.side
      ].join("|");
    },
    /** 收藏夹列表（自己的空间） */
    dynComposerAvatar() {
      const p = this.navProfileRecord;
      const face =
        p && typeof p.face === "string" ? String(p.face).trim() : "";
      if (face) {
        return face;
      }
      return akari;
    },
    levelDisplay() {
      const fromProfile =
        this.profile &&
        this.profile.level_info &&
        this.profile.level_info.current_level;
      if (fromProfile != null) {
        const n = Number(fromProfile);
        if (Number.isFinite(n) && n >= 1) {
          return Math.min(6, Math.max(1, Math.floor(n)));
        }
      }
      if (
        this.isRealSpaceOwner &&
        !this.isPerspectivePreview &&
        this.navProfileRecord &&
        this.navProfileRecord.level_info
      ) {
        const lv = this.navProfileRecord.level_info.current_level;
        if (lv != null) {
          const n = Number(lv);
          if (Number.isFinite(n) && n >= 1) {
            return Math.min(6, Math.max(1, Math.floor(n)));
          }
        }
      }
      return 1;
    },
    videoTotalDisplay() {
      return this.videos.length;
    },
    homeFolderTotalDisplay() {
      const n = Number(this.homeFoldersTotal);
      return Number.isFinite(n) && n > 0 ? n : this.homeFolders.length;
    },
    homeFoldersPreview() {
      return (this.homeFolders || []).slice(0, 5);
    },
    homeRecentCoinsPreview() {
      return (this.homeRecentCoins || []).slice(0, 5);
    },
    spacePrivacyView() {
      const p = this.profile;
      if (p && p.privacy && typeof p.privacy === "object") {
        return p.privacy;
      }
      if (!p) {
        return null;
      }
      return this.spacePrivacy;
    },
    showHomeFoldersSection() {
      if (this.isRealSpaceOwner && !this.isPerspectivePreview) {
        return true;
      }
      const priv = this.spacePrivacyView;
      return !!(priv && priv.public_favorites);
    },
    showHomeRecentCoinsSection() {
      if (this.isRealSpaceOwner && !this.isPerspectivePreview) {
        return true;
      }
      const priv = this.spacePrivacyView;
      return !!(priv && priv.public_recent_coins);
    },
    showPublicBirthdayAside() {
      if (this.isRealSpaceOwner && !this.isPerspectivePreview) {
        return true;
      }
      const priv = this.spacePrivacyView;
      return priv ? priv.public_birthday !== false : true;
    },
    canOpenFollowingList() {
      if (this.isRealSpaceOwner && !this.isPerspectivePreview) {
        return true;
      }
      const priv = this.spacePrivacyView;
      if (!priv) {
        return false;
      }
      return !!priv.public_following;
    },
    canOpenFollowersList() {
      if (this.isRealSpaceOwner && !this.isPerspectivePreview) {
        return true;
      }
      const priv = this.spacePrivacyView;
      if (!priv) {
        return false;
      }
      return !!priv.public_fans;
    },
    privacySettingColumns() {
      return [
        [
          { key: "public_favorites", label: "公开我的收藏" },
          { key: "public_recent_coins", label: "公开最近投币的视频" }
        ],
        [
          { key: "public_following", label: "公开我的关注列表" },
          { key: "public_fans", label: "公开我的粉丝列表" }
        ],
        [{ key: "public_birthday", label: "公开我的生日" }]
      ];
    },
    spaceVideoGridClass() {
      const cls = ["mb-space__video-grid"];
      if (this.activeNav === "contribute") {
        cls.push("mb-space__video-grid--contrib");
        if (this.spaceVideoViewMode === "list") {
          cls.push("mb-space__video-grid--list");
        }
      }
      return cls;
    },
    activeCollectFolder() {
      const list = this.collectFolders || [];
      const hit = list.find((f) => f.id === this.collectFolderId);
      return (
        hit ||
        list[0] || {
          id: null,
          title: "默认收藏夹",
          isPublic: true,
          videoCount: 0,
          coverUrl: null,
          description: ""
        }
      );
    },
    collectFolderEditInitial() {
      const f = this.collectFolderEditTarget;
      if (!f) {
        return null;
      }
      return {
        title: f.title,
        description: f.description || "",
        is_public: f.isPublic,
        cover_url: f.coverUrl || null
      };
    },
    collectDisplayVideos() {
      if (this.collectSideNav !== "folders") return [];
      let list = Array.isArray(this.collectVideos)
        ? this.collectVideos.slice()
        : [];
      if (this.collectSearchScope === "all") {
        list = this.dedupeCollectVideosById(list);
      }
      const q = String(this.collectSearchKeyword || "")
        .trim()
        .toLowerCase();
      if (q) {
        list = list.filter((v) =>
          String(v.title || "")
            .toLowerCase()
            .includes(q)
        );
      }
      if (this.collectSort === "play") {
        list.sort(
          (a, b) =>
            (Number(b.play_count) || 0) - (Number(a.play_count) || 0)
        );
      } else if (this.collectSort === "submit") {
        list.sort((a, b) =>
          String(b.created_at || "").localeCompare(
            String(a.created_at || "")
          )
        );
      }
      return list;
    },
    collectBatchSelectedCount() {
      return this.collectBatchSelectedIds.length;
    },
    collectBatchAllSelected() {
      const list = this.collectDisplayVideos;
      if (!list.length) return false;
      return list.every((v) =>
        this.collectBatchSelectedIds.includes(Number(v.id))
      );
    },
    collectTransferVideoCount() {
      if (this.collectBatchMode && this.collectBatchSelectedIds.length) {
        return this.collectBatchSelectedIds.length;
      }
      return this.collectTransferVideoId ? 1 : 0;
    },
    navTabs() {
      const n =
        this.videos.length + (this.spaceArticles.length || 0);
      const contribBadge = n > 0 ? n : null;
      return [
        {
          key: "home",
          label: "主页",
          icon: iconHome,
          iconClass: "mb-space__tab-ico--home",
          badge: null
        },
        {
          key: "dynamic",
          label: "动态",
          icon: iconUpdate,
          iconClass: "",
          badge: null
        },
        {
          key: "contribute",
          label: "投稿",
          icon: iconContribute,
          iconClass: "",
          badge: contribBadge
        },
        {
          key: "collect",
          label: "收藏",
          icon: iconCollect,
          iconClass: "mb-space__tab-ico--collect",
          badge: null
        },
        {
          key: "settings",
          label: "设置",
          icon: iconSet,
          iconClass: "",
          badge: null
        }
      ].filter((tab) => {
        if (tab.key === "settings") {
          return this.isSpaceOwner && this.mbLoggedIn;
        }
        if (tab.key === "collect") {
          return this.canViewCollectTab;
        }
        return true;
      });
    },
    spaceSearchQuery() {
      return this.videoSearch.trim().toLowerCase();
    },
    filteredVideos() {
      const q = this.spaceSearchQuery;
      if (!q) {
        return this.videos;
      }
      return this.videos.filter((v) =>
        String(v.title || "")
          .toLowerCase()
          .includes(q)
      );
    },
    filteredSpaceArticles() {
      const q = this.spaceSearchQuery;
      if (!q) {
        return this.spaceArticles || [];
      }
      return (this.spaceArticles || []).filter((a) =>
        String(a.title || "")
          .toLowerCase()
          .includes(q)
      );
    },
    sortedVideos() {
      const list = [...this.filteredVideos];
      if (this.videoSort === "play") {
        list.sort((a, b) => Number(b.play_count) - Number(a.play_count));
        return list;
      }
      if (this.videoSort === "fav") {
        list.sort((a, b) => {
          const likeDiff =
            Number(b.like_count ?? 0) - Number(a.like_count ?? 0);
          if (likeDiff !== 0) {
            return likeDiff;
          }
          return Number(b.danmaku_count) - Number(a.danmaku_count);
        });
        return list;
      }
      list.sort((a, b) => Number(b.id) - Number(a.id));
      return list;
    },
    /** 收藏展示列表（含搜索过滤） */
    videosChronoForDynamic() {
      const list = [...this.filteredVideos];
      list.sort((a, b) => Number(b.id) - Number(a.id));
      return list;
    },
    dynVideoEntries() {
      return this.videosChronoForDynamic.map((v) => ({
        kind: "video",
        id: `dyn-v-${v.id}`,
        ts: this.parseVideoTs(v.created_at),
        video: v
      }));
    },
    dynImageEntries() {
      return (this.imageDynamics || []).map((p) => ({
        kind: "image",
        id: `dyn-i-${p.id}`,
        ts: this.parseVideoTs(p.createdAt || p.created_at),
        post: p,
        liked_by_me: !!p.liked_by_me,
        stats: p.stats || { forward: 0, comment: 0, like: 0 }
      }));
    },
    dynArticleEntries() {
      return (this.spaceArticles || []).map((a) => ({
        kind: "article",
        id: `dyn-a-${a.id}`,
        ts: this.parseVideoTs(a.published_at || a.created_at),
        article: {
          id: a.id,
          title: a.title,
          cover_url: a.cover_url,
          view_count: a.view_count,
          comment_count: a.comment_count,
          fav_count: a.fav_count,
          forward_count: a.forward_count,
          favorited_by_me: !!a.favorited_by_me,
          comments_closed: !!a.comments_closed,
          comments_curated: !!a.comments_curated
        }
      }));
    },
    dynMergedSorted() {
      const merged = [
        ...this.dynVideoEntries,
        ...this.dynImageEntries,
        ...this.dynArticleEntries
      ];
      merged.sort((a, b) => b.ts - a.ts);
      if (!this.isOwnSpace) {
        return merged;
      }
      const pinSet = new Set(this.dynPinnedRowIds || []);
      merged.sort((a, b) => {
        const ap = pinSet.has(a.id);
        const bp = pinSet.has(b.id);
        if (ap && bp) {
          return b.ts - a.ts;
        }
        if (ap) {
          return -1;
        }
        if (bp) {
          return 1;
        }
        return b.ts - a.ts;
      });
      return merged;
    },
    dynSearchFiltered() {
      const q = this.spaceSearchQuery;
      if (!q) {
        return this.dynMergedSorted;
      }
      return this.dynMergedSorted.filter((row) =>
        this.dynRowMatchesSearch(row, q)
      );
    },
    /** 主页搜索时展示的动态命中（最多 8 条） */
    homeDynSearchMatches() {
      if (!this.spaceSearchQuery) {
        return [];
      }
      return this.dynSearchFiltered.slice(0, 8);
    },
    dynVisibleList() {
      if (this.dynamicSubtab === "video") {
        return this.dynSearchFiltered.filter((x) => x.kind === "video");
      }
      return this.dynSearchFiltered;
    },
    statPlays() {
      let t = 0;
      for (const v of this.videos) {
        t += Number(v.play_count) || 0;
      }
      return t;
    },
    /** 格式化 UID 展示 */
    asideUidText() {
      return this.userIdNum ? String(this.userIdNum) : "?";
    },
    noticeCharCount() {
      return [...String(this.noticeDraft || "")].length;
    },
    noticeDirty() {
      return (
        String(this.noticeDraft || "").trim() !==
        String(this.noticeSaved || "").trim()
      );
    },
    visitorNoticeText() {
      if (this.isOwnSpace) {
        return "";
      }
      const p = this.profile;
      if (!p || typeof p !== "object") {
        return "";
      }
      const a =
        p.announcement != null ? String(p.announcement).trim() : "";
      return a;
    },
    /**
     * 生日优先 GET /space/:id 的 birthday，否则 Vuex minibiliMe
     * 与账号资料页 profile 一致
     */
    asideBirthText() {
      if (!this.isSpaceOwner && !this.showPublicBirthdayAside) {
        return "未公开";
      }
      const pickRaw = (o) => {
        if (!o || typeof o !== "object") {
          return "";
        }
        const raw =
          o.birthday ?? o.birth_date ?? o.birth ?? "";
        return String(raw).trim();
      };
      const format = (s) => {
        if (!s) {
          return "";
        }
        const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(s);
        if (m) {
          return `${m[2]}-${m[3]}`;
        }
        if (/^\d{2}-\d{2}$/.test(s)) {
          return s;
        }
        return s.length > 12 ? s.slice(0, 12) : s;
      };
      if (
        this.isRealSpaceOwner &&
        !this.isPerspectivePreview &&
        this.minibiliMe
      ) {
        const s = format(pickRaw(this.minibiliMe));
        if (s) {
          return s;
        }
      }
      const s = format(pickRaw(this.profile));
      return s || "未填写";
    }
  },
  watch: {
    videoSearch(val) {
      const q = String(val || "").trim();
      if (!q) {
        this.clearSpaceSearchNavTimer();
        return;
      }
      void this.loadImageDynamics();
      if (!this.spaceArticles.length && !this.spaceArticlesLoading) {
        void this.loadSpaceArticles();
      }
      if (this.shouldAutoRedirectSearchToDynamic()) {
        this.scheduleSpaceSearchNavToDynamic();
      }
    },
    contribSubtab(val) {
      if (val === "article") {
        void this.loadSpaceArticles();
      }
    },
    activeNav(val) {
      if (
        val === "dynamic" &&
        this.userIdNum &&
        !this.spaceArticles.length &&
        !this.spaceArticlesLoading
      ) {
        void this.loadSpaceArticles();
      }
    },
    userIdNum() {
      this.resetAndLoad();
    },
    spacePerspective(mode) {
      if (this.isRealSpaceOwner && this.userIdNum) {
        writeStoredSpacePerspective(this.userIdNum, mode);
      }
      this.applyPerspectiveView();
    },
    isOwnSpace(val) {
      if (val && this.profile) {
        this.syncNoticeFromProfile();
      }
    },
    spaceRouteSig: {
      handler() {
        this.applySpaceRouteQuery(this.$route);
      },
    },
    collectSearchScope() {
      if (this.activeNav === "collect" && this.collectSideNav === "folders") {
        void this.loadCollectFavorites();
      }
    },
    activeNav(val) {
      if (val === "home") {
        void this.loadHomeSections();
      }
      if (val === "settings" && this.isSpaceOwner && this.mbLoggedIn) {
        void this.loadSpacePrivacy();
      }
      if (val === "collect") {
        if (!this.canViewCollectTab) {
          this.activeNav = "home";
          return;
        }
        void this.loadArticleFavCount();
      }
    },
    "profile.privacy": {
      handler() {
        if (this.activeNav === "home") {
          void this.loadHomeSections();
        }
      },
      deep: true
    }
  },
  beforeRouteUpdate(to, _from, next) {
    this.applySpaceRouteQuery(to);
    next();
  },
  mounted() {
    document.addEventListener("click", this.onDynCommentDocClick);
    this.resetAndLoad();
  },
  beforeUnmount() {
    document.removeEventListener("click", this.onDynCommentDocClick);
    this.articleFavLoading = false;
    this.clearCollectFolderMenuCloseTimer();
    if (this._noticeSuccessTid) {
      clearTimeout(this._noticeSuccessTid);
      this._noticeSuccessTid = null;
    }
    this.clearSpaceSearchNavTimer();
  },
  methods: {
    minibiliVideoPlayRoute,
    minibiliArticleReadRoute,
    minibiliDynamicReadRoute,
    ...mapActions(["refreshMinibiliMe"]),
    onCollectPlayAll() {
      const list = this.collectDisplayVideos;
      if (!list.length) return;
      const first = list[0];
      this.$router
        .push(minibiliVideoPlayRoute(first.id))
        .catch(() => {});
    },

    selectCollectFolder(folderId) {
      this.collectSideNav = "folders";
      this.collectFolderId = folderId;
      void this.syncCollectSideRoute(null);
      if (this.collectBatchMode) {
        this.exitCollectBatchMode();
      }
      void this.loadCollectFolders().then(() => this.loadCollectFavorites());
    },
    enterCollectBatchMode() {
      if (!this.isOwnSpace || !this.mbLoggedIn || this.collectFolderId == null) {
        return;
      }
      this.collectSearchScope = "current";
      this.collectBatchMode = true;
      this.collectBatchSelectedIds = [];
    },
    exitCollectBatchMode() {
      this.collectBatchMode = false;
      this.collectBatchSelectedIds = [];
      this.collectBatchActionSaving = false;
    },
    isCollectBatchSelected(videoId) {
      return this.collectBatchSelectedIds.includes(Number(videoId));
    },
    toggleCollectBatchSelect(videoId) {
      const id = Number(videoId);
      if (!id) return;
      const set = new Set(this.collectBatchSelectedIds);
      if (set.has(id)) {
        set.delete(id);
      } else {
        set.add(id);
      }
      this.collectBatchSelectedIds = [...set];
    },
    toggleCollectBatchSelectAll() {
      const list = this.collectDisplayVideos;
      if (this.collectBatchAllSelected) {
        this.collectBatchSelectedIds = [];
        return;
      }
      this.collectBatchSelectedIds = list.map((v) => Number(v.id)).filter(Boolean);
    },
    openCollectClearInvalid() {
      if (this.collectFolderId == null) return;
      this.collectClearInvalidOpen = true;
    },
    async onCollectClearInvalidConfirm() {
      if (this.collectClearInvalidSaving || this.collectFolderId == null) return;
      this.collectClearInvalidSaving = true;
      try {
        const res = await mbClearInvalidFavoritesInFolder(this.collectFolderId);
        const n = Number(res.cleared) || 0;
        this.collectClearInvalidOpen = false;
        await this.loadCollectFolders();
        await this.loadCollectFavorites();
        showMbDarkToast(n > 0 ? `已清除 ${n} 条失效内容` : "暂无失效内容");
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败";
        ElMessage.error(String(msg));
      } finally {
        this.collectClearInvalidSaving = false;
      }
    },
    async onCollectBatchUnfavorite() {
      const ids = this.collectBatchSelectedIds.slice();
      if (!ids.length || this.collectFolderId == null || this.collectBatchActionSaving) {
        return;
      }
      this.collectBatchActionSaving = true;
      try {
        await mbBatchRemoveVideosFromFavoriteFolder(this.collectFolderId, ids);
        const idSet = new Set(ids.map(Number));
        this.collectVideos = this.collectVideos.filter(
          (row) => !idSet.has(Number(row.id))
        );
        this.collectBatchSelectedIds = [];
        await this.loadCollectFolders();
        showMbDarkToast("取消收藏成功");
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败";
        ElMessage.error(String(msg));
      } finally {
        this.collectBatchActionSaving = false;
      }
    },
    onCollectBatchCopyTo() {
      if (!this.collectBatchSelectedIds.length) return;
      this.collectTransferVideoId = null;
      this.collectTransferMode = "copy";
      this.collectTransferOpen = true;
    },
    onCollectBatchMoveTo() {
      if (!this.collectBatchSelectedIds.length) return;
      this.collectTransferVideoId = null;
      this.collectTransferMode = "move";
      this.collectTransferOpen = true;
    },
    onCollectTransferClose() {
      this.collectTransferOpen = false;
      if (!this.collectBatchMode) {
        this.collectTransferVideoId = null;
      }
    },
    async selectCollectArticleFav() {
      this.activeNav = "collect";
      this.collectSideNav = "articleFav";
      void this.syncCollectSideRoute("articleFav");
      await this.loadArticleFavorites(true);
      void this.loadArticleFavCount();
    },
    normalizeArticleFavPayload(raw) {
      const o = raw && typeof raw === "object" ? raw : null;
      if (!o) {
        return { items: [], total: 0 };
      }
      if (typeof o.code === "number" && o.data && typeof o.data === "object") {
        const d = o.data;
        return {
          items: Array.isArray(d.items) ? d.items : [],
          total: Number(d.total) || 0
        };
      }
      return {
        items: Array.isArray(o.items) ? o.items : [],
        total: Number(o.total) || 0
      };
    },
    useMyArticleFavoritesApi() {
      return this.isRealSpaceOwner && !!getAccessToken();
    },
    syncCollectSideRoute(side) {
      const uid = this.userIdNum;
      if (!uid || this.$route.name !== "minibiliUserSpace") return;
      const query = { ...(this.$route.query || {}), nav: "collect" };
      if (side) {
        query.side = side;
      } else {
        delete query.side;
      }
      void this.$router.replace({
        name: "minibiliUserSpace",
        params: { userId: String(uid) },
        query
      });
    },
    async loadArticleFavCount() {
      if (!this.userIdNum) {
        this.articleFavTotal = 0;
        return;
      }
      try {
        let res;
        if (this.useMyArticleFavoritesApi()) {
          res = await mbListMyArticleFavorites({ limit: 1 });
        } else {
          res = await mbListUserArticleFavorites(this.userIdNum, { limit: 1 });
        }
        this.articleFavTotal = Number(res.total) || 0;
      } catch {
        this.articleFavTotal = 0;
      }
    },
    async loadArticleFavorites(force = false) {
      if (!this.userIdNum || this.collectSideNav !== "articleFav") {
        return;
      }
      if (!force && this.articleFavItems.length > 0) {
        return;
      }
      this.articleFavLoading = true;
      try {
        let raw;
        if (this.useMyArticleFavoritesApi()) {
          raw = await mbListMyArticleFavorites({ limit: 50 });
        } else {
          raw = await mbListUserArticleFavorites(this.userIdNum, { limit: 50 });
        }
        const res = this.normalizeArticleFavPayload(raw);
        const list = Array.isArray(res.items) ? res.items : [];
        this.articleFavItems = list
          .map((row) => ({
            ...row,
            id: Number(row.id)
          }))
          .filter((row) => Number.isFinite(row.id) && row.id > 0);
        this.articleFavTotal =
          Number(res.total) || this.articleFavItems.length;
      } catch {
        this.articleFavItems = [];
        this.articleFavTotal = 0;
      } finally {
        this.articleFavLoading = false;
      }
    },
    selectCollectLater() {
      if (this.mbLoggedIn) {
        this.$router.push({ name: "minibiliWatchLater" }).catch(() => {});
        return;
      }
      this.collectSideNav = "later";
    },
    applySpaceRouteQuery(route) {
      const r = route || this.$route;
      const q = r.query || {};
      if (q.nav === "collect" && q.side === "later") {
        this.$router.replace({ name: "minibiliWatchLater" }).catch(() => {});
        return;
      }
      if (r.name !== "minibiliUserSpace") {
        return;
      }
      if (q.nav === "dynamic") {
        if (this.activeNav !== "dynamic") {
          this.activeNav = "dynamic";
        }
        void this.loadImageDynamics();
        return;
      }
      if (q.nav === "collect") {
        if (!this.canViewCollectTab) {
          this.activeNav = "home";
          void this.syncSpaceRouteForNav("home");
          return;
        }
        if (this.activeNav !== "collect") {
          this.activeNav = "collect";
          void this.loadArticleFavCount();
          void this.loadCollectFolders().then(() => this.loadCollectFavorites());
        }
        void this.loadArticleFavCount();
        if (q.side === "articleFav") {
          this.activeNav = "collect";
          this.collectSideNav = "articleFav";
          void this.loadArticleFavorites();
        }
        return;
      }
      if (q.nav === "contribute") {
        if (this.activeNav !== "contribute") {
          this.activeNav = "contribute";
        }
        if (q.side === "article") {
          this.contribSubtab = "article";
          void this.loadSpaceArticles();
        } else {
          this.contribSubtab = "video";
        }
        return;
      }
      if (q.nav === "settings") {
        if (!this.isSpaceOwner || !this.mbLoggedIn) {
          this.activeNav = "home";
          void this.syncSpaceRouteForNav("home");
          return;
        }
        if (this.activeNav !== "settings") {
          this.activeNav = "settings";
          void this.loadSpacePrivacy();
        }
        return;
      }
      if (this.activeNav !== "home") {
        this.activeNav = "home";
        void this.loadHomeSections();
      }
    },
    syncSpaceRouteForNav(navKey, { usePush = false, side = null } = {}) {
      const uid = this.userIdNum;
      if (!uid || this.$route.name !== "minibiliUserSpace") {
        return Promise.resolve();
      }
      const uidStr = String(uid);
      const query = { ...(this.$route.query || {}) };
      delete query.side;
      if (navKey === "collect") {
        query.nav = "collect";
      } else if (navKey === "dynamic") {
        query.nav = "dynamic";
      } else if (navKey === "contribute") {
        query.nav = "contribute";
      } else if (navKey === "settings") {
        query.nav = "settings";
      } else {
        delete query.nav;
      }
      if (
        side &&
        (navKey === "collect" || navKey === "contribute")
      ) {
        query.side = side;
      }
      const loc = {
        name: "minibiliUserSpace",
        params: { userId: uidStr },
        query
      };
      return usePush
        ? this.$router.push(loc).catch(() => {})
        : this.$router.replace(loc).catch(() => {});
    },
    mapCollectFolderRow(row) {
      return {
        id: Number(row.id),
        title: String(row.title || ""),
        description: String(row.description || ""),
        isPublic: !!row.is_public,
        isDefault: !!row.is_default,
        videoCount: Number(row.video_count) || 0,
        coverUrl: row.cover_url ? String(row.cover_url) : null
      };
    },
    async loadCollectFolders() {
      if (this.collectSideNav !== "folders") return;
      this.collectFoldersLoading = true;
      try {
        let items = [];
        if (this.isOwnSpace && this.mbLoggedIn) {
          const res = await mbListMyFavoriteFolders();
          items = Array.isArray(res.items) ? res.items : [];
        } else if (this.userIdNum > 0) {
          const res = await mbListUserFavoriteFolders(this.userIdNum);
          items = Array.isArray(res.items) ? res.items : [];
        }
        this.collectFolders = items.map((row) => this.mapCollectFolderRow(row));
        if (!this.collectFolders.length) {
          this.collectFolderId = null;
          return;
        }
        const cur = this.collectFolders.find(
          (f) => f.id === this.collectFolderId
        );
        if (!cur) {
          const def =
            this.collectFolders.find((f) => f.isDefault) ||
            this.collectFolders[0];
          this.collectFolderId = def ? def.id : null;
        }
      } catch {
        this.collectFolders = [];
        this.collectFolderId = null;
      } finally {
        this.collectFoldersLoading = false;
      }
    },
    openCollectFolderCreate() {
      if (!this.isOwnSpace || !this.mbLoggedIn) return;
      this.collectFolderDialogMode = "create";
      this.collectFolderEditTarget = null;
      this.collectFolderCreateOpen = true;
    },
    openCollectFolderEdit(folder) {
      if (!this.isOwnSpace || !this.mbLoggedIn) return;
      this.clearCollectFolderMenuCloseTimer();
      this.collectFolderMenuId = null;
      this.collectFolderDialogMode = "edit";
      this.collectFolderEditTarget = folder;
      this.collectFolderCreateOpen = true;
    },
    openCollectFolderDelete(folder) {
      if (!this.isOwnSpace || !this.mbLoggedIn || !folder || folder.isDefault) {
        return;
      }
      this.clearCollectFolderMenuCloseTimer();
      this.collectFolderMenuId = null;
      this.collectFolderDeleteTarget = folder;
      this.collectFolderDeleteOpen = true;
    },
    clearCollectFolderMenuCloseTimer() {
      if (this._collectFolderMenuCloseTimer) {
        clearTimeout(this._collectFolderMenuCloseTimer);
        this._collectFolderMenuCloseTimer = null;
      }
    },
    scheduleCollectFolderMenuClose(folderId) {
      const id = Number(folderId);
      this.clearCollectFolderMenuCloseTimer();
      this._collectFolderMenuCloseTimer = setTimeout(() => {
        if (this.collectFolderMenuId === id) {
          this.collectFolderMenuId = null;
        }
        if (this.collectFolderHoverId === id) {
          this.collectFolderHoverId = null;
        }
        this._collectFolderMenuCloseTimer = null;
      }, 320);
    },
    onCollectFolderItemMouseEnter(folderId) {
      this.clearCollectFolderMenuCloseTimer();
      this.collectFolderHoverId = Number(folderId);
    },
    onCollectFolderItemMouseLeave(folderId) {
      this.scheduleCollectFolderMenuClose(folderId);
    },
    onCollectFolderMoreEnter(folderId) {
      const id = Number(folderId);
      this.clearCollectFolderMenuCloseTimer();
      this.collectFolderHoverId = id;
      this.collectFolderMenuId = id;
    },
    onCollectFolderMoreLeave(folderId) {
      this.scheduleCollectFolderMenuClose(folderId);
    },
    async onCollectFolderCreateSubmit(payload) {
      if (this.collectFolderCreateSaving) return;
      this.collectFolderCreateSaving = true;
      try {
        const body = {
          title: payload.title,
          description: payload.description || "",
          is_public: payload.is_public,
          cover: payload.cover || null
        };
        if (this.collectFolderDialogMode === "edit") {
          const editId = this.collectFolderEditTarget && this.collectFolderEditTarget.id;
          if (!editId) return;
          const row = await mbUpdateFavoriteFolder(editId, body);
          const folder = this.mapCollectFolderRow(row);
          this.collectFolders = this.collectFolders.map((f) =>
            f.id === folder.id ? folder : f
          );
          this.collectFolderCreateOpen = false;
          showMbDarkToast("已保存");
        } else {
          const row = await mbCreateFavoriteFolder(body);
          const folder = this.mapCollectFolderRow(row);
          this.collectFolders = [...this.collectFolders, folder];
          this.collectFolderId = folder.id;
          this.collectFolderCreateOpen = false;
          showMbDarkToast("收藏夹创建成功");
          await this.loadCollectFavorites();
        }
      } catch (e) {
        await this.loadCollectFolders();
        ElMessage.error(extractApiErrorMessage(e, "操作失败，请稍后重试"));
      } finally {
        this.collectFolderCreateSaving = false;
      }
    },
    async onCollectFolderDeleteConfirm() {
      const f = this.collectFolderDeleteTarget;
      if (!f || f.isDefault || this.collectFolderDeleteSaving) return;
      this.collectFolderDeleteSaving = true;
      try {
        await mbDeleteFavoriteFolder(f.id);
        const wasCurrent = this.collectFolderId === f.id;
        await this.loadCollectFolders();
        if (wasCurrent) {
          const def =
            this.collectFolders.find((row) => row.isDefault) ||
            this.collectFolders[0];
          if (def) {
            this.collectFolderId = def.id;
          }
          await this.loadCollectFavorites();
        }
        this.collectFolderDeleteOpen = false;
        this.collectFolderDeleteTarget = null;
        showMbDarkToast("已删除");
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "删除失败";
        ElMessage.error(String(msg));
      } finally {
        this.collectFolderDeleteSaving = false;
      }
    },
    toggleCollectVideoMenu(videoId) {
      const id = Number(videoId);
      this.collectVideoMenuId =
        this.collectVideoMenuId === id ? null : id;
    },
    closeCollectVideoMenu() {
      this.collectVideoMenuId = null;
    },
    async onCollectUnfavorite(v) {
      this.closeCollectVideoMenu();
      if (this.collectFolderId == null) return;
      const videoId = Number(v && v.id);
      if (!videoId) return;
      try {
        await mbRemoveVideoFromFavoriteFolder(
          videoId,
          this.collectFolderId
        );
        this.collectVideos = this.collectVideos.filter(
          (row) => Number(row.id) !== videoId
        );
        await this.loadCollectFolders();
        showMbDarkToast("取消收藏成功");
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败";
        ElMessage.error(String(msg));
      }
    },
    onCollectCopyTo(v) {
      this.closeCollectVideoMenu();
      this.collectTransferVideoId = Number(v && v.id) || null;
      this.collectTransferMode = "copy";
      this.collectTransferOpen = true;
    },
    onCollectMoveTo(v) {
      this.closeCollectVideoMenu();
      this.collectTransferVideoId = Number(v && v.id) || null;
      this.collectTransferMode = "move";
      this.collectTransferOpen = true;
    },
    onCollectTransferFolderCreated() {
      void this.loadCollectFolders();
    },
    async onCollectTransferConfirm(targetFolderId) {
      const folderId = Number(targetFolderId);
      if (!folderId || this.collectTransferSaving) return;
      const batchIds =
        this.collectBatchMode && this.collectBatchSelectedIds.length
          ? this.collectBatchSelectedIds.slice()
          : [];
      const videoId = batchIds.length ? null : this.collectTransferVideoId;
      if (!batchIds.length && !videoId) return;
      if (
        this.collectTransferMode === "move" &&
        this.collectFolderId == null
      ) {
        return;
      }
      this.collectTransferSaving = true;
      try {
        const ids = batchIds.length ? batchIds : [Number(videoId)];
        if (this.collectTransferMode === "move") {
          for (const vid of ids) {
            await mbMoveVideoFavoriteFolder(
              vid,
              this.collectFolderId,
              folderId
            );
          }
          const idSet = new Set(ids.map(Number));
          this.collectVideos = this.collectVideos.filter(
            (row) => !idSet.has(Number(row.id))
          );
          showMbDarkToast("已移动到其他收藏夹");
        } else {
          for (const vid of ids) {
            await mbCopyVideoToFavoriteFolder(vid, folderId);
          }
          showMbDarkToast("已复制到收藏夹");
        }
        this.collectTransferOpen = false;
        if (this.collectBatchMode) {
          this.collectBatchSelectedIds = [];
        } else {
          this.collectTransferVideoId = null;
        }
        await this.loadCollectFolders();
        if (
          this.collectTransferMode === "copy" &&
          folderId === this.collectFolderId
        ) {
          await this.loadCollectFavorites();
        }
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败";
        ElMessage.error(String(msg));
      } finally {
        this.collectTransferSaving = false;
      }
    },
    dedupeCollectVideosById(list) {
      const rows = Array.isArray(list) ? list : [];
      const seen = new Set();
      const out = [];
      for (const v of rows) {
        const id = Number(v && v.id);
        if (!Number.isFinite(id) || id <= 0 || seen.has(id)) {
          continue;
        }
        seen.add(id);
        out.push(v);
      }
      return out;
    },
    async loadCollectFavorites() {
      if (this.collectSideNav !== "folders") return;
      this.collectLoading = true;
      try {
        const params = { limit: 200 };
        if (
          this.collectSearchScope === "current" &&
          this.collectFolderId != null
        ) {
          params.folder_id = this.collectFolderId;
        }
        let res;
        if (this.isOwnSpace && this.mbLoggedIn) {
          res = await mbListMyFavorites(params);
        } else if (this.userIdNum > 0) {
          res = await mbListUserFavorites(this.userIdNum, params);
        } else {
          this.collectVideos = [];
          return;
        }
        const rawItems = Array.isArray(res.items) ? res.items : [];
        this.collectVideos =
          this.collectSearchScope === "all"
            ? this.dedupeCollectVideosById(rawItems)
            : rawItems;
        if (this.collectFolders.length && this.collectFolderId != null) {
          const idx = this.collectFolders.findIndex(
            (f) => f.id === this.collectFolderId
          );
          if (idx >= 0) {
            const f = { ...this.collectFolders[idx] };
            f.videoCount = this.collectVideos.length;
            const first = this.collectVideos[0];
            f.coverUrl =
              first && first.cover_url
                ? String(first.cover_url)
                : f.coverUrl;
            const next = this.collectFolders.slice();
            next[idx] = f;
            this.collectFolders = next;
          }
        }
      } catch {
        this.collectVideos = [];
      } finally {
        this.collectLoading = false;
      }
    },
    async loadWatchLater() {
      if (!this.isOwnSpace || !this.mbLoggedIn) {
        this.watchLaterVideos = [];
        return;
      }
      this.watchLaterLoading = true;
      try {
        const res = await mbListMyWatchLater({ limit: 100 });
        this.watchLaterVideos = Array.isArray(res.items) ? res.items : [];
      } catch {
        this.watchLaterVideos = [];
      } finally {
        this.watchLaterLoading = false;
      }
    },
    async onWatchLaterToggle(videoId) {
      if (!this.mbLoggedIn) {
        ElMessage.warning("请先登录后再点赞");
        return;
      }
      const id = Number(videoId);
      if (!Number.isFinite(id) || id <= 0) {
        return;
      }
      try {
        const res = await mbToggleWatchLater(id);
        if (!res.in_watch_later) {
          this.watchLaterVideos = this.watchLaterVideos.filter(
            (v) => Number(v.id) !== id
          );
        }
        ElMessage.success(
          res.in_watch_later ? "已加入稍后再看" : "已移出稍后再看"
        );
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      }
    },
    onWatchLaterPlaceholder(e, videoId) {
      if (e) {
        e.stopPropagation();
        e.preventDefault();
      }
      void this.onWatchLaterToggle(videoId);
    },
    levelIconUrl,
    commentCount(v) {
      if (!v || typeof v !== "object") {
        return 0;
      }
      const c = v.comment_count;
      if (c == null) {
        return 0;
      }
      const n = Number(c);
      return Number.isFinite(n) ? n : 0;
    },
    /** 与 B 站一致：visibility / is_private 等表示仅自己可见 */
    isVideoSelfOnlyVisible(v) {
      if (this.isPerspectivePreview) {
        return false;
      }
      if (!v || typeof v !== "object") {
        return false;
      }
      const o = v;
      if (o.visibility === "private" || o.is_private === true) {
        return true;
      }
      return false;
    },
    onNavClick(key) {
      if (key === "settings") {
        if (!this.isOwnSpace || !this.mbLoggedIn) {
          return;
        }
        this.activeNav = "settings";
        void this.syncSpaceRouteForNav("settings");
        void this.loadSpacePrivacy();
        return;
      }
      this.closeDynCommentMenus();
      this.dynCommentVideoId = null;
      this.dynCommentArticleId = null;
      this.dynCommentDraft = "";
      this.dynDeleteTarget = null;
      if (key === "collect" && !this.canViewCollectTab) {
        return;
      }
      if (key === "dynamic") {
        this.activeNav = "dynamic";
        void this.syncSpaceRouteForNav("dynamic");
        void this.loadImageDynamics();
        return;
      }
      if (key === "home" || key === "collect") {
        this.activeNav = key;
        void this.syncSpaceRouteForNav(
          key === "collect" ? "collect" : "home"
        ).then(() => {
          this.applySpaceRouteQuery();
        });
        if (key === "home") {
          void this.loadHomeSections();
        } else if (
          this.collectSideNav === "later" &&
          this.isOwnSpace &&
          this.mbLoggedIn
        ) {
          void this.loadWatchLater();
        } else if (this.collectSideNav === "articleFav") {
          void this.loadArticleFavCount();
          void this.loadArticleFavorites();
        } else {
          void this.loadArticleFavCount();
          void this.loadCollectFolders().then(() => this.loadCollectFavorites());
        }
        return;
      }
      if (key === "contribute") {
        this.activeNav = "contribute";
        const side =
          this.contribSubtab === "article" ? "article" : "video";
        void this.syncSpaceRouteForNav("contribute", { side });
        return;
      }
      this.activeNav = key;
    },
    resetAndLoad() {
      this.spacePerspective = "self";
      if (this.userIdNum) {
        writeStoredSpacePerspective(this.userIdNum, "self");
      }
      this._profileOwnerSnapshot = null;
      this.closeDynCommentMenus();
      this.profile = null;
      this.videos = [];
      this.nextCursor = "";
      this.loadError = "";
      this.activeNav = "home";
      this.contribSubtab = "video";
      this.spaceVideoViewMode = "grid";
      this.collectFoldersOpen = true;
      this.collectSideNav = "folders";
      this.collectFolderId = null;
      this.collectFolderHoverId = null;
      this.collectFolderMenuId = null;
      this.clearCollectFolderMenuCloseTimer();
      this.exitCollectBatchMode();
      this.collectClearInvalidOpen = false;
      this.collectFolderDialogMode = "create";
      this.collectFolderEditTarget = null;
      this.collectFolderDeleteOpen = false;
      this.collectFolderDeleteTarget = null;
      this.collectSort = "recent";
      this.collectSearchScope = "current";
      this.collectSearchKeyword = "";
      this.collectVideos = [];
      this.collectLoading = false;
      this.articleFavItems = [];
      this.articleFavTotal = 0;
      this.articleFavLoading = false;
      this.watchLaterVideos = [];
      this.watchLaterLoading = false;
      this.homeFolders = [];
      this.homeFoldersTotal = 0;
      this.homeFoldersHidden = 0;
      this.homeFoldersLoading = false;
      this.homeRecentCoins = [];
      this.homeRecentCoinsTotal = 0;
      this.homeRecentCoinsLoading = false;
      this.spacePrivacy = {
        public_favorites: false,
        public_recent_coins: false,
        public_following: false,
        public_fans: false,
        public_birthday: true
      };
      this.dynamicSubtab = "all";
      this.imageDynamics = [];
      this.dynCommentVideoId = null;
      this.dynCommentArticleId = null;
      this.dynCommentDraft = "";
      this.dynDeleteTarget = null;
      this.dynPinnedRowIds = [];
      this.spaceArticles = [];
      this.spaceArticlesLoading = false;
      this.noticeDraft = "";
      this.noticeSaved = "";
      this.noticeFocused = false;
      void this.loadProfile();
      void this.loadSpaceArticles();
      void this.loadVideos(true);
      void this.loadHomeSections();
      this.applySpaceRouteQuery();
    },
    syncNoticeFromProfile() {
      const raw =
        this.profile && this.profile.announcement != null
          ? String(this.profile.announcement)
          : "";
      this.noticeDraft = raw;
      this.noticeSaved = raw;
    },
    onNoticeInput() {
      const r = [...String(this.noticeDraft || "")];
      if (r.length > 150) {
        this.noticeDraft = r.slice(0, 150).join("");
      }
    },
    onNoticeFocus() {
      this.noticeFocused = true;
    },
    async onNoticeBlur() {
      try {
        if (this.isOwnSpace && !this.noticeSaving && this.noticeDirty) {
          await this.saveNotice();
        }
      } finally {
        this.noticeFocused = false;
      }
    },
    flashNoticeSuccess() {
      this.noticeSuccessVisible = true;
      if (this._noticeSuccessTid) {
        clearTimeout(this._noticeSuccessTid);
      }
      this._noticeSuccessTid = setTimeout(() => {
        this.noticeSuccessVisible = false;
        this._noticeSuccessTid = null;
      }, 2200);
    },
    async saveNotice() {
      if (!this.isOwnSpace || this.noticeSaving) {
        return;
      }
      const trimmed = String(this.noticeDraft || "").trim();
      if ([...trimmed].length > 150) {
        return;
      }
      this.noticeSaving = true;
      try {
        const res = await mbPutMeAnnouncement(trimmed);
        const saved = String(res.announcement || "");
        this.noticeSaved = saved;
        this.noticeDraft = saved;
        if (this.profile && typeof this.profile === "object") {
          this.profile = { ...this.profile, announcement: saved };
        }
        this.flashNoticeSuccess();
      } catch {
        /* 忽略非 http 封面 */
      } finally {
        this.noticeSaving = false;
      }
    },
    async loadProfile() {
      if (!this.userIdNum) {
        this.loadError = "无效的用户";
        return;
      }
      const requestUid = this.userIdNum;
      this.profileLoading = true;
      try {
        const profile = await mbGetUserPublic(requestUid, {
          skipGlobalErrorToast: true
        });
        if (this.userIdNum !== requestUid) {
          return;
        }
        this._profileOwnerSnapshot = profile;
        this.applyPerspectiveView();
      } catch (e) {
          this.loadError = (e && e.message) || "无法加载用户资料";
        this.profile = null;
        this._profileOwnerSnapshot = null;
      } finally {
        this.profileLoading = false;
        if (
          this.profile &&
          this.isRealSpaceOwner &&
          !this.isPerspectivePreview
        ) {
          this.syncNoticeFromProfile();
        }
        if (this.isRealSpaceOwner && this.spacePerspective === "self") {
          void this.refreshMinibiliMe();
        }
        this.loadImageDynamics();
        this.loadDynPins();
      }
    },
    async loadVideos(reset) {
      if (!this.userIdNum) {
        return;
      }
      if (this.listLoading) {
        return;
      }
      this.listLoading = true;
      try {
        const payload = await mbListUserPublishedVideos(this.userIdNum, {
          limit: 30,
          cursor: reset ? undefined : this.nextCursor || undefined
        });
        const items = Array.isArray(payload.items) ? payload.items : [];
        if (reset) {
          this.videos = items;
        } else {
          const seen = new Set(this.videos.map((x) => x.id));
          for (const it of items) {
            if (!seen.has(it.id)) {
              seen.add(it.id);
              this.videos.push(it);
            }
          }
        }
        this.nextCursor = String(payload.next_cursor || "").trim();
      } catch (e) {
        if (!this.profileLoading && !this.profile) {
          this.loadError = (e && e.message) || "无法加载用户资料";
        }
      } finally {
        this.listLoading = false;
      }
    },
    loadMore() {
      if (!this.nextCursor) {
        return;
      }
      void this.loadVideos(false);
    },
    async loadSpaceArticles() {
      if (!this.userIdNum) {
        this.spaceArticles = [];
        return;
      }
      this.spaceArticlesLoading = true;
      try {
        const res = await mbListUserPublishedArticles(this.userIdNum, {
          limit: 40
        });
        this.spaceArticles = res.items || [];
      } catch {
        this.spaceArticles = [];
      } finally {
        this.spaceArticlesLoading = false;
      }
    },
    formatDuration(sec) {
      const n = Number(sec) || 0;
      const m = Math.floor(n / 60);
      const s = Math.floor(n % 60);
      return `${m}:${String(s).padStart(2, "0")}`;
    },
    /** 收藏时间，输入如 YYYY-MM-DD HH:mm:ss */
    formatArticlePubDate(raw) {
      const s = String(raw || "").trim();
      if (!s) return "";
      const m = /^(\d{4})-(\d{1,2})-(\d{1,2})/.exec(s);
      if (m) {
        const y = Number(m[1]);
        const mo = Number(m[2]);
        const d = Number(m[3]);
        const now = new Date();
        if (now.getFullYear() === y) {
          return `${mo}-${d}`;
        }
        return `${y}-${mo}-${d}`;
      }
      return s.length > 10 ? s.slice(0, 10) : s;
    },
    formatFavoritedMD(raw) {
      const s = String(raw || "").trim();
      if (!s) {
        return "";
      }
      const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(s);
      if (m) {
        return `${m[2]}-${m[3]}`;
      }
      return s.length >= 10 ? s.slice(5, 10) : "";
    },

    formatVideoYMD(raw) {
      const s = String(raw || "").trim();
      if (!s) {
        return "";
      }
      const m = /^(\d{4}-\d{2}-\d{2})/.exec(s);
      if (m) {
        return m[1];
      }
      return s.length >= 10 ? s.slice(0, 10) : s;
    },
    parseVideoTs(s) {
      const raw = String(s || "").trim();
      if (!raw) {
        return 0;
      }
      const t = Date.parse(raw.replace(" ", "T"));
      return Number.isFinite(t) ? t : 0;
    },
    /** 图文动态可检索文案（title / content / 旧版 lines） */
    dynImageSearchText(post) {
      if (!post || typeof post !== "object") {
        return "";
      }
      const parts = [
        post.title,
        post.content,
        ...this.dynImageLegacyLines(post),
        ...(Array.isArray(post.lines) ? post.lines : [])
      ];
      return parts
        .map((s) => String(s || "").trim())
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
    },
    dynRowMatchesSearch(row, q) {
      const needle = q || this.spaceSearchQuery;
      if (!needle) {
        return true;
      }
      if (row.kind === "video") {
        return String(row.video.title || "")
          .toLowerCase()
          .includes(needle);
      }
      if (row.kind === "article") {
        return String(row.article.title || "")
          .toLowerCase()
          .includes(needle);
      }
      if (row.kind === "image") {
        return this.dynImageSearchText(row.post).includes(needle);
      }
      return false;
    },
    dynRowSearchTitle(row) {
      if (!row) {
        return "";
      }
      if (row.kind === "video") {
        return String(row.video.title || "").trim() || "视频";
      }
      if (row.kind === "article") {
        return String(row.article.title || "").trim() || "专栏";
      }
      if (row.kind === "image" && row.post) {
        const title = String(row.post.title || "").trim();
        if (title) {
          return title;
        }
        const content = String(row.post.content || "").trim();
        if (content) {
          return content.length > 40 ? `${content.slice(0, 40)}…` : content;
        }
        const legacy = this.dynImageLegacyLines(row.post)[0];
        return legacy || "图文动态";
      }
      return "动态";
    },
    dynRowSearchRoute(row) {
      if (!row) {
        return null;
      }
      if (row.kind === "video") {
        return this.minibiliVideoPlayRoute(row.video.id);
      }
      if (row.kind === "article") {
        return this.minibiliArticleReadRoute(row.article.id);
      }
      if (row.kind === "image") {
        return this.minibiliDynamicReadRoute(row.post.id);
      }
      return null;
    },
    dynRowSearchThumb(row) {
      if (!row) {
        return "";
      }
      if (row.kind === "video") {
        return String(row.video?.cover_url || "").trim();
      }
      if (row.kind === "article") {
        return String(row.article?.cover_url || "").trim();
      }
      if (row.kind === "image" && row.post) {
        const imgs = Array.isArray(row.post.images) ? row.post.images : [];
        if (imgs.length) {
          return String(imgs[0] || "").trim();
        }
        return String(row.post.cover_url || "").trim();
      }
      return "";
    },
    dynRowSearchKindLabel(row) {
      if (!row) {
        return "动态";
      }
      if (row.kind === "video") {
        return "视频";
      }
      if (row.kind === "article") {
        return "专栏";
      }
      return "图文";
    },
    dynRowSearchExcerpt(row, maxLen = 56) {
      if (!row) {
        return "";
      }
      const clip = (s) => {
        const t = String(s || "").trim();
        if (!t) {
          return "";
        }
        return t.length > maxLen ? `${t.slice(0, maxLen)}…` : t;
      };
      if (row.kind === "video") {
        const d = this.videoDescriptionText(row.video);
        return d === "暂无简介" ? "" : clip(d);
      }
      if (row.kind === "article") {
        return clip(this.articleDescriptionText(row.article));
      }
      if (row.kind === "image" && row.post) {
        const content = String(row.post.content || "").trim();
        if (content) {
          return clip(content);
        }
        const legacy = this.dynImageLegacyLines(row.post).join(" ");
        return clip(legacy);
      }
      return "";
    },
    dynRowSearchMeta(row) {
      if (!row) {
        return "";
      }
      const parts = [];
      const date = this.formatDynDateCN(row.ts);
      if (date) {
        parts.push(date);
      }
      if (row.kind === "video" && row.video) {
        parts.push(`${this.formatCount(row.video.play_count)}播放`);
      } else if (row.kind === "article" && row.article) {
        parts.push(`${this.formatCount(row.article.view_count)}阅读`);
      } else if (row.kind === "image" && row.post) {
        const n = Array.isArray(row.post.images) ? row.post.images.length : 0;
        if (n > 0) {
          parts.push(`${n}张图片`);
        }
      }
      return parts.join(" · ");
    },
    shouldAutoRedirectSearchToDynamic() {
      if (!this.spaceSearchQuery || this.activeNav === "dynamic") {
        return false;
      }
      return (
        this.activeNav === "collect" ||
        this.activeNav === "settings" ||
        this.activeNav === "contribute"
      );
    },
    clearSpaceSearchNavTimer() {
      if (this._spaceSearchNavTimer) {
        clearTimeout(this._spaceSearchNavTimer);
        this._spaceSearchNavTimer = null;
      }
    },
    scheduleSpaceSearchNavToDynamic() {
      this.clearSpaceSearchNavTimer();
      this._spaceSearchNavTimer = setTimeout(() => {
        this._spaceSearchNavTimer = null;
        if (!this.shouldAutoRedirectSearchToDynamic()) {
          return;
        }
        this.openDynamicWithSearch();
      }, 350);
    },
    ensureSpaceSearchData() {
      void this.loadImageDynamics();
      if (!this.spaceArticles.length && !this.spaceArticlesLoading) {
        void this.loadSpaceArticles();
      }
    },
    openDynamicWithSearch() {
      this.clearSpaceSearchNavTimer();
      this.closeDynCommentMenus();
      this.dynamicSubtab = "all";
      this.activeNav = "dynamic";
      void this.syncSpaceRouteForNav("dynamic");
      this.ensureSpaceSearchData();
      this.$nextTick(() => {
        const el = this.$el?.querySelector?.(".mb-space__dyn-feed");
        el?.scrollIntoView?.({ behavior: "smooth", block: "start" });
      });
    },
    onSpaceSearchSubmit() {
      const q = this.videoSearch.trim();
      if (!q) {
        return;
      }
      this.ensureSpaceSearchData();
      if (this.activeNav !== "dynamic") {
        this.openDynamicWithSearch();
      }
    },
    /** 仅旧版 localStorage 图文（无 title/content 字段）走 lines 展示 */
    dynImageLegacyLines(post) {
      if (!post || typeof post !== "object") {
        return [];
      }
      const title = String(post.title || "").trim();
      const content = String(post.content || "").trim();
      if (title || content) {
        return [];
      }
      const lines = Array.isArray(post.lines) ? post.lines : [];
      return lines.map((l) => String(l || "").trim()).filter(Boolean);
    },
    formatDynDateCN(ts) {
      const d = new Date(ts);
      if (!Number.isFinite(d.getTime())) {
        return "";
      }
      return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
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
    patchArticleMeta(articleId, partial) {
      const id = Number(articleId);
      if (!Number.isFinite(id) || id <= 0) {
        return;
      }
      this.spaceArticles = this.spaceArticles.map((a) =>
        Number(a.id) === id ? { ...a, ...partial } : a
      );
    },
    patchVideoMeta(videoId, partial) {
      const id = Number(videoId);
      if (!Number.isFinite(id) || id <= 0) {
        return;
      }
      this.videos = this.videos.map((v) =>
        Number(v.id) === id ? { ...v, ...partial } : v
      );
    },
    dynVideoCommentsClosed(v) {
      return !!(v && v.comments_closed);
    },
    dynVideoCommentsCurated(v) {
      return !!(v && v.comments_curated);
    },
    dynArticleCommentsCurated(a) {
      return !!(a && a.comments_curated);
    },
    dynCreatorPendingCommentsRoute(media, id) {
      const mid = Number(id) || 0;
      if (media === "article") {
        return {
          name: "creatorComments",
          query: {
            tab: "pending",
            media: "article",
            article_id: String(mid)
          }
        };
      }
      return {
        name: "creatorComments",
        query: {
          tab: "pending",
          media: "video",
          video_id: String(mid)
        }
      };
    },
    dynCommentComposerPlaceholder(v) {
      return this.dynVideoCommentsCurated(v)
        ? MB_COMMENT_CURATED_PLACEHOLDER
        : "说点什么…";
    },
    openDynMbStationDialog(kind, videoId, articleId = 0, dynamicId = 0) {
      this.closeDynCommentMenus();
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
      } else if (kind === "close_danmaku") {
        body = { danmaku_closed: true };
      } else if (kind === "restore_danmaku") {
        body = { danmaku_closed: false };
      } else {
        this.dynMbStationOpen = false;
        return;
      }
      if (isArticle && (kind === "close_danmaku" || kind === "restore_danmaku")) {
        this.dynMbStationOpen = false;
        return;
      }
      this.dynMbStationLoading = true;
      try {
        if (isArticle) {
          const r = await mbPatchArticlePlayback(aid, body);
          this.patchArticleMeta(aid, {
            comments_closed: r.comments_closed,
            comments_curated: r.comments_curated
          });
          if (Number(this.dynCommentArticleId) === aid) {
            await this.refreshDynArticleCommentsLive();
          }
        } else if (isVideo) {
          const r = await mbPatchVideoPlayback(vid, body);
          this.patchVideoMeta(vid, {
            comments_closed: r.comments_closed,
            comments_curated: r.comments_curated,
            danmaku_closed: r.danmaku_closed
          });
          if (Number(this.dynCommentVideoId) === vid) {
            await this.refreshDynCommentsLive();
          }
        } else {
          const r = await mbPatchDynamicPlayback(did, body);
          this.patchImageDynamicMeta(did, {
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
      if (!this.mbLoggedIn) {
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
        this.patchVideoMeta(vid, {
          liked_by_me: liked,
          like_count: Math.max(0, base + delta)
        });
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      }
    },
    toggleDynCommentPanel(mediaId, kind = "video") {
      const id = Number(mediaId);
      if (!Number.isFinite(id) || id <= 0) {
        return;
      }
      if (kind === "dynamic" || kind === "image") {
        if (this.dynCommentDynamicId === id) {
          this.closeDynCommentMenus();
          this.dynCommentDynamicId = null;
          return;
        }
        this.closeDynCommentMenus();
        this.dynCommentDynamicId = id;
        this.dynCommentVideoId = null;
        this.dynCommentArticleId = null;
        this.dynCommentDraft = "";
        return;
      }
      if (kind === "article") {
        if (this.dynCommentArticleId === id) {
          this.closeDynCommentMenus();
          this.dynCommentArticleId = null;
          return;
        }
        this.closeDynCommentMenus();
        this.dynCommentArticleId = id;
        this.dynCommentVideoId = null;
        this.dynCommentDynamicId = null;
        this.dynCommentDraft = "";
        return;
      }
      if (this.dynCommentVideoId === id) {
        this.closeDynCommentMenus();
        this.dynCommentVideoId = null;
        this.dynCommentDraft = "";
        return;
      }
      this.closeDynCommentMenus();
      this.dynCommentVideoId = id;
      this.dynCommentArticleId = null;
      this.dynCommentDynamicId = null;
      this.dynCommentSort = "hot";
      this.dynCommentDraft = "";
      void this.syncDynCommentsClosedFlag(id);
    },
    onDynCommentPatchDynamic({ dynamicId, partial }) {
      this.patchImageDynamicMeta(dynamicId, partial);
    },
    onDynDynamicCommentsLiveCounts(n) {
      const did = Number(this.dynCommentDynamicId);
      if (Number.isFinite(did) && did > 0) {
        this.patchImageDynamicMeta(did, { comment_count: Number(n) || 0 });
      }
    },
    patchImageDynamicMeta(dynamicId, partial) {
      const id = Number(dynamicId);
      if (!Number.isFinite(id) || id <= 0) {
        return;
      }
      for (const p of this.imageDynamics || []) {
        if (Number(p.id) !== id) {
          continue;
        }
        if (!p.stats) {
          p.stats = { forward: 0, comment: 0, like: 0 };
        }
        if (partial.comment_count != null) {
          p.stats.comment = Number(partial.comment_count) || 0;
        }
        if (partial.like_count != null) {
          p.stats.like = Number(partial.like_count) || 0;
        }
        if (partial.liked_by_me != null) {
          p.liked_by_me = !!partial.liked_by_me;
        }
        if (partial.comments_closed != null) {
          p.comments_closed = !!partial.comments_closed;
        }
        if (partial.comments_curated != null) {
          p.comments_curated = !!partial.comments_curated;
        }
      }
    },
    async onDynImageLike(row) {
      if (!this.mbLoggedIn) {
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
        this.patchImageDynamicMeta(did, {
          liked_by_me: liked,
          like_count: Math.max(0, base + delta)
        });
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      }
    },
    onDynCommentPatchArticle({ articleId, partial }) {
      this.patchArticleMeta(articleId, partial);
    },
    onDynArticleCommentsLiveCounts(n) {
      const aid = Number(this.dynCommentArticleId);
      if (Number.isFinite(aid) && aid > 0) {
        this.patchArticleMeta(aid, { comment_count: Number(n) || 0 });
      }
    },
    async onDynArticleFavorite(row) {
      if (!this.mbLoggedIn) {
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
        this.patchArticleMeta(aid, {
          favorited_by_me: favorited,
          fav_count:
            fav_count != null ? Number(fav_count) : Math.max(0, base + delta)
        });
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      }
    },
    async syncDynCommentsClosedFlag(videoId) {
      const vid = Number(videoId);
      if (!Number.isFinite(vid) || vid <= 0) {
        return;
      }
      try {
        const { comments_closed: closedFlag } = await mbListComments(vid);
        this.patchVideoMeta(vid, { comments_closed: !!closedFlag });
      } catch {
        /* ignore */
      }
    },
    refreshDynCommentsLive(opts) {
      const ref = this.$refs.dynCommentsLive;
      if (ref && typeof ref.load === "function") {
        return ref.load(opts || { soft: true, preserveExpand: true });
      }
      return Promise.resolve();
    },
    refreshDynArticleCommentsLive(opts) {
      const ref = this.$refs.dynArticleCommentPanel;
      if (ref && typeof ref.refreshCommentsLive === "function") {
        return ref.refreshCommentsLive(opts || { soft: true, preserveExpand: true });
      }
      return Promise.resolve();
    },
    onDynCommentsLiveCounts(n) {
      const vid = Number(this.dynCommentVideoId);
      if (Number.isFinite(vid) && vid > 0) {
        this.patchVideoMeta(vid, { comment_count: Number(n) || 0 });
      }
    },
    async submitDynComment(videoId) {
      if (!this.mbLoggedIn) {
        this.openMbLoginModalFromDynCmt();
        return;
      }
      const text = String(this.dynCommentDraft || "").trim();
      if (!text) {
        return;
      }
      const vid = Number(videoId);
      this.dynCommentPosting = true;
      try {
        const res = await mbPostComment(vid, text, 0);
        this.dynCommentDraft = "";
        if (res && res.approved !== false) {
          const vrow = this.videos.find((x) => Number(x.id) === vid);
          const cc = Number(vrow && vrow.comment_count) || 0;
          this.patchVideoMeta(vid, { comment_count: cc + 1 });
        }
        await this.refreshDynCommentsLive();
        if (res && res.approved === false) {
          ElMessage.success("评论已提交，待UP主精选后展示");
        } else {
          ElMessage.success("发表成功");
        }
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      } finally {
        this.dynCommentPosting = false;
      }
    },
    onDynCommentDocClick() {
      this.dynCmtHeadMenuOpen = false;
      this.collectVideoMenuId = null;
    },
    closeDynCommentMenus() {
      this.dynCmtHeadMenuOpen = false;
    },
    toggleDynCmtHeadMenu(e) {
      if (e) e.stopPropagation();
      this.dynCmtHeadMenuOpen = !this.dynCmtHeadMenuOpen;
    },
    openMbLoginModalFromDynCmt() {
      this.$store.commit("login/SET_LOGIN_TAB", 0);
      this.$store.commit("login/OPEN_LOGIN_MODAL");
    },
    dynPinsStorageKey() {
      return `minibili_space_dyn_pins_${this.userIdNum}`;
    },
    loadDynPins() {
      if (!this.userIdNum || this.isPerspectivePreview) {
        this.dynPinnedRowIds = [];
        return;
      }
      try {
        const raw = localStorage.getItem(this.dynPinsStorageKey());
        if (!raw) {
          this.dynPinnedRowIds = [];
          return;
        }
        const arr = JSON.parse(raw);
        if (!Array.isArray(arr)) {
          this.dynPinnedRowIds = [];
          return;
        }
        this.dynPinnedRowIds = arr
          .map((x) => String(x || "").trim())
          .filter(Boolean);
      } catch {
        this.dynPinnedRowIds = [];
      }
    },
    saveDynPins() {
      if (!this.userIdNum) {
        return;
      }
      try {
        localStorage.setItem(
          this.dynPinsStorageKey(),
          JSON.stringify(this.dynPinnedRowIds || [])
        );
      } catch {
        /* ignore */
      }
    },
    isDynRowPinned(row) {
      return !!(row && (this.dynPinnedRowIds || []).includes(row.id));
    },
    onDynMenuPin(row) {
      if (!row || !row.id) {
        return;
      }
      const cur = [...(this.dynPinnedRowIds || [])];
      const ix = cur.indexOf(row.id);
      if (ix >= 0) {
        cur.splice(ix, 1);
        this.dynPinnedRowIds = cur;
        this.saveDynPins();
        ElMessage.success("已置顶");
      } else {
        this.dynPinnedRowIds = [row.id, ...cur.filter((id) => id !== row.id)];
        this.saveDynPins();
        ElMessage.success("已置顶");
      }
      this.$nextTick(() => {
        const root = this.$el;
        if (!root || typeof root.querySelector !== "function") {
          return;
        }
        const feed = root.querySelector(".mb-space__dyn-feed");
        if (feed && typeof feed.scrollIntoView === "function") {
          feed.scrollIntoView({ behavior: "smooth", block: "start" });
        }
      });
    },
    removeDynPinRowId(rowId) {
      if (!rowId) {
        return;
      }
      this.dynPinnedRowIds = (this.dynPinnedRowIds || []).filter(
        (id) => id !== rowId
      );
      this.saveDynPins();
    },
    openDynDeleteDialog(row) {
      if (!row || !this.isOwnSpace) {
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
    persistImageDynamics() {
      if (!this.userIdNum) {
        return;
      }
      const key = `minibili_space_dyn_img_${this.userIdNum}`;
      try {
        localStorage.setItem(key, JSON.stringify(this.imageDynamics || []));
      } catch {
        /* ignore */
      }
    },
    async confirmDynDelete() {
      const t = this.dynDeleteTarget;
      if (!t || this.dynDeleteSubmitting) {
        return;
      }
      if (
        (t.kind === "video" || t.kind === "article" || t.kind === "image") &&
        !this.mbLoggedIn
      ) {
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
          this.videos = this.videos.filter((x) => Number(x.id) !== vid);
          if (Number(this.dynCommentVideoId) === vid) {
            this.closeDynCommentMenus();
            this.dynCommentVideoId = null;
            this.dynCommentDraft = "";
          }
        } else if (t.kind === "article") {
          const aid = Number(t.article && t.article.id);
          if (!Number.isFinite(aid) || aid <= 0) {
            throw new Error("无效的专栏");
          }
          await mbDeleteMyArticle(aid);
          this.spaceArticles = this.spaceArticles.filter(
            (x) => Number(x.id) !== aid
          );
          if (Number(this.dynCommentArticleId) === aid) {
            this.dynCommentArticleId = null;
          }
        } else {
          const did = Number(t.post && t.post.id);
          if (Number.isFinite(did) && did > 0) {
            await mbDeleteMyDynamic(did);
            this.imageDynamics = (this.imageDynamics || []).filter(
              (x) => Number(x.id) !== did
            );
            if (Number(this.dynCommentDynamicId) === did) {
              this.dynCommentDynamicId = null;
            }
          } else {
            const pid = t.post && String(t.post.id || "").trim();
            this.imageDynamics = (this.imageDynamics || []).filter(
              (x) => String(x.id) !== pid
            );
            this.persistImageDynamics();
          }
        }
        this.removeDynPinRowId(t.id);
        this.dynDeleteTarget = null;
        ElMessage({
          message: "删除成功",
          duration: 2200,
          customClass: "mb-space-dyn-delete-toast"
        });
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      } finally {
        this.dynDeleteSubmitting = false;
      }
    },
    normalizeImageDynamics(arr) {
      return (Array.isArray(arr) ? arr : [])
        .filter((x) => x && typeof x === "object")
        .map((x) => ({
          id: String(x.id || "").trim(),
          type: "image",
          createdAt: String(x.createdAt || "").trim(),
          lines: Array.isArray(x.lines) ? x.lines.map((l) => String(l)) : [],
          images: Array.isArray(x.images) ? x.images.map((u) => String(u)) : [],
          stats: {
            forward: Number(x.stats && x.stats.forward) || 0,
            comment: Number(x.stats && x.stats.comment) || 0,
            like: Number(x.stats && x.stats.like) || 0
          }
        }))
        .filter((x) => x.id);
    },
    async loadImageDynamics() {
      if (!this.userIdNum || this.isPerspectivePreview) {
        this.imageDynamics = [];
        return;
      }
      try {
        const { items } = await mbListUserPublishedDynamics(this.userIdNum, {
          limit: 50
        });
        this.imageDynamics = (items || []).map((item) => ({
          id: item.id,
          type: "image",
          createdAt: item.created_at,
          created_at: item.created_at,
          title: item.title || "",
          content: item.content || "",
          images: Array.isArray(item.images) ? item.images : [],
          stats: {
            forward: 0,
            comment: Number(item.comment_count) || 0,
            like: Number(item.like_count) || 0
          },
          liked_by_me: !!item.liked_by_me,
          comments_closed: !!item.comments_closed,
          comments_curated: !!item.comments_curated
        }));
      } catch {
        this.imageDynamics = [];
      }
    },
    formatCount(n) {
      const v = Number(n) || 0;
      if (v >= 10000) {
        return (v / 10000).toFixed(1).replace(/\.0$/, "") + "万";
      }
      return String(v);
    },
    openPlayAll() {
      const list = this.sortedVideos;
      if (!list.length) {
        return;
      }
      const first = list[0];
      this.$router
        .push(minibiliVideoPlayRoute(first.id))
        .catch(() => {});
    },
    async loadHomeFolders() {
      if (!this.userIdNum) {
        this.homeFolders = [];
        this.homeFoldersTotal = 0;
        this.homeFoldersHidden = 0;
        return;
      }
      this.homeFoldersLoading = true;
      try {
        const res = await mbListUserFavoriteFolders(this.userIdNum);
        const items = Array.isArray(res.items) ? res.items : [];
        this.homeFolders = items.map((row) => this.mapCollectFolderRow(row));
        const total = Number(res.total);
        this.homeFoldersTotal = Number.isFinite(total)
          ? total
          : this.homeFolders.length;
        const hidden = Number(res.hidden_count);
        this.homeFoldersHidden =
          Number.isFinite(hidden) && hidden > 0 ? hidden : 0;
      } catch {
        this.homeFolders = [];
        this.homeFoldersTotal = 0;
        this.homeFoldersHidden = 0;
      } finally {
        this.homeFoldersLoading = false;
      }
    },
    async loadHomeRecentCoins() {
      if (!this.userIdNum) {
        this.homeRecentCoins = [];
        this.homeRecentCoinsTotal = 0;
        return;
      }
      if (!this.showHomeRecentCoinsSection) {
        this.homeRecentCoins = [];
        this.homeRecentCoinsTotal = 0;
        return;
      }
      this.homeRecentCoinsLoading = true;
      try {
        const res = await mbListUserRecentCoinVideos(this.userIdNum, {
          limit: 20
        });
        this.homeRecentCoins = Array.isArray(res.items) ? res.items : [];
        this.homeRecentCoinsTotal = Number(res.total) || 0;
      } catch {
        this.homeRecentCoins = [];
        this.homeRecentCoinsTotal = 0;
      } finally {
        this.homeRecentCoinsLoading = false;
      }
    },
    async loadHomeSections() {
      if (this.activeNav !== "home") {
        return;
      }
      await Promise.all([
        this.loadHomeFolders(),
        this.loadHomeRecentCoins()
      ]);
    },
    onHomeSeeMoreFolders() {
      this.onHomeOpenCollectFolder(null);
    },
    async onHomeOpenCollectFolder(folderId) {
      this.collectSideNav = "folders";
      if (folderId != null) {
        this.collectFolderId = Number(folderId);
      }
      if (this.$route.query.nav !== "collect") {
        await this.syncSpaceRouteForNav("collect", { usePush: true });
      }
      this.activeNav = "collect";
      void this.loadCollectFolders().then(() => this.loadCollectFavorites());
    },
    openRelations(tab) {
      const t = tab === "followers" ? "followers" : "following";
      const extra =
        this.isPerspectivePreview && this.spacePerspective !== "self"
          ? { perspective: this.spacePerspective }
          : {};
      const r = minibiliUserSpaceRelationsRoute(this.userIdNum, t, extra);
      if (!r) {
        return;
      }
      if (this.profileLoading || !this.profile) {
        this.$router.push(r).catch(() => {});
        return;
      }
      if (t === "followers" && !this.canOpenFollowersList) {
        showMbDarkToast(personalSpaceZhCN.relations.followersHiddenToast);
        return;
      }
      if (t === "following" && !this.canOpenFollowingList) {
        showMbDarkToast(personalSpaceZhCN.relations.followingHiddenToast);
        return;
      }
      this.$router.push(r).catch(() => {});
    },
    onSpaceFollowerCount(n) {
      const fans = Number(n);
      if (Number.isFinite(fans) && fans >= 0) {
        this.statFans = fans;
      }
    },
    applyPerspectiveView() {
      const snap = this._profileOwnerSnapshot;
      if (!snap || typeof snap !== "object") {
        return;
      }
      if (this.spacePerspective === "self") {
        this.profile = { ...snap };
      } else if (isSpacePerspectivePreviewMode(this.spacePerspective)) {
        this.profile = buildSpaceViewerProfile(snap, this.spacePerspective);
      } else {
        this.profile = { ...snap };
      }
      if (this.profile) {
        this.applySpacePrivacyFromProfile(this.profile);
        this.spaceFollowedByMe = !!this.profile.followed_by_me;
      }
      if (this.isPerspectivePreview) {
        this.closeDynCommentMenus();
        this.dynCommentVideoId = null;
        this.dynCommentArticleId = null;
        this.dynCommentDynamicId = null;
        this.dynCommentDraft = "";
        this.imageDynamics = [];
        this.dynPinnedRowIds = [];
        if (this.activeNav === "settings") {
          this.activeNav = "home";
          void this.syncSpaceRouteForNav("home");
        }
        if (this.activeNav === "collect" && !this.canViewCollectTab) {
          this.activeNav = "home";
          this.collectFolders = [];
          this.collectVideos = [];
          this.collectFolderId = null;
          void this.syncSpaceRouteForNav("home");
        }
        if (this.activeNav === "collect" && this.collectSideNav === "later") {
          this.collectSideNav = "folders";
          this.watchLaterVideos = [];
        }
      }
      if (this.activeNav === "home") {
        void this.loadHomeSections();
      }
      if (this.activeNav === "collect" && this.canViewCollectTab) {
        void this.loadCollectFolders().then(() => this.loadCollectFavorites());
      }
      if (this.isRealSpaceOwner && this.userIdNum) {
        writeStoredSpacePerspective(this.userIdNum, this.spacePerspective);
      }
    },
    onHeaderFollowedByMeUpdate(v) {
      if (this.isPerspectivePreview) {
        return;
      }
      this.spaceFollowedByMe = !!v;
    },
    applySpacePrivacyFromProfile(profile) {
      if (!profile || typeof profile !== "object") {
        return;
      }
      const following = Number(profile.following_count);
      const followers = Number(profile.follower_count);
      if (Number.isFinite(following)) {
        this.statFollowing = following;
      }
      if (Number.isFinite(followers)) {
        this.statFans = followers;
      }
      this.spaceFollowedByMe = !!profile.followed_by_me;
      const p = profile.privacy;
      if (!p || typeof p !== "object") {
        return;
      }
      this.spacePrivacy = {
        public_favorites: !!p.public_favorites,
        public_recent_coins: !!p.public_recent_coins,
        public_following: !!p.public_following,
        public_fans: !!p.public_fans,
        public_birthday: p.public_birthday !== false
      };
      if (this.activeNav === "home") {
        void this.loadHomeSections();
      }
    },
    async loadSpacePrivacy() {
      if (!this.isOwnSpace || !this.mbLoggedIn) {
        return;
      }
      this.spacePrivacyLoading = true;
      try {
        const res = await mbGetMeSpacePrivacy();
        this.spacePrivacy = {
          public_favorites: !!res.public_favorites,
          public_recent_coins: !!res.public_recent_coins,
          public_following: !!res.public_following,
          public_fans: !!res.public_fans,
          public_birthday: res.public_birthday !== false
        };
        if (this.profile && typeof this.profile === "object") {
          this.profile = {
            ...this.profile,
            privacy: { ...this.spacePrivacy }
          };
        }
      } catch {
        /* ignore */
      } finally {
        this.spacePrivacyLoading = false;
      }
    },
    async onToggleSpacePrivacy(key) {
      if (this.spacePrivacySaving || !key) {
        return;
      }
      const k = String(key);
      if (!(k in this.spacePrivacy)) {
        return;
      }
      const next = !this.spacePrivacy[k];
      const prev = { ...this.spacePrivacy };
      this.spacePrivacy = { ...this.spacePrivacy, [k]: next };
      this.spacePrivacySaving = true;
      try {
        const res = await mbPutMeSpacePrivacy({ [k]: next });
        this.spacePrivacy = {
          public_favorites: !!res.public_favorites,
          public_recent_coins: !!res.public_recent_coins,
          public_following: !!res.public_following,
          public_fans: !!res.public_fans,
          public_birthday: res.public_birthday !== false
        };
        if (this.profile && typeof this.profile === "object") {
          this.profile = {
            ...this.profile,
            privacy: { ...this.spacePrivacy }
          };
        }
        if (this.activeNav === "home") {
          void this.loadHomeSections();
        }
        if (this.activeNav === "collect") {
          void this.loadCollectFolders().then(() => this.loadCollectFavorites());
        }
      } catch {
        this.spacePrivacy = prev;
        if (this.profile && typeof this.profile === "object") {
          this.profile = {
            ...this.profile,
            privacy: { ...prev }
          };
        }
        ElMessage.error("保存失败，请稍后重试");
      } finally {
        this.spacePrivacySaving = false;
      }
    },
    /** 主页视频区「查看更多」→ 投稿 · 视频（push 历史，浏览器后退回主页） */
    async onSeeMoreVideos() {
      this.closeDynCommentMenus();
      this.dynCommentVideoId = null;
      this.dynCommentArticleId = null;
      this.contribSubtab = "video";
      const q = this.$route.query || {};
      if (q.nav !== "contribute" || q.side !== "video") {
        await this.syncSpaceRouteForNav("contribute", {
          usePush: true,
          side: "video"
        });
      }
      this.activeNav = "contribute";
    }
  }
};
</script>

<style lang="scss" scoped>
/* 个人空间：顶栏 + 主体分栏 */
$mb-space-nav-pad-l: 62px;
$mb-space-nav-pad-r: 20px;
.mb-space {
  --text1: #18191c;
  min-height: 100vh;
  padding-bottom: 0;
  background: #fff;
  box-sizing: border-box;
  padding-left: env(safe-area-inset-left, 0px);
  padding-right: env(safe-area-inset-right, 0px);
}

/* 主栏与 space-main 对齐 */
.mb-space__col {
  width: 100%;
  max-width: 2260px;
  min-height: 100vh;
  margin-left: auto;
  margin-right: auto;
  box-sizing: border-box;
  background: #fff;
  display: flex;
  flex-direction: column;
}

.mb-space__split {
  display: flex;
  flex-wrap: nowrap;
  align-items: stretch;
  flex: 1 1 auto;
  width: 100%;
  box-sizing: border-box;
  min-width: 0;
  min-height: 0;
  background: #fff;
  padding: 16px 0 28px $mb-space-nav-pad-l;
  gap: 0;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.mb-space__split--no-aside {
  padding-right: $mb-space-nav-pad-r;
  flex: 1 1 auto;
  min-height: 0;
}

.mb-space__aside {
  width: 292px;
  flex: 0 0 292px;
  box-sizing: border-box;
  /* 侧栏卡片 stack */
  background: transparent;
  border: none;
  border-radius: 0;
  box-shadow: none;
  color: #18191c;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 10px $mb-space-nav-pad-r 18px 10px;
  position: sticky;
  top: 0;
}

.mb-space__card {
  background: #f6f7f8;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 2px rgba(15, 20, 30, 0.05);
  flex-shrink: 0;
  border: 1px solid rgba(0, 0, 0, 0.04);
}

.mb-space__creator-top {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px 14px 15px;
  background: #f6f7f8;
  border-bottom: 1px solid #e8eaed;
  color: #00aeec;
  font-size: 15px;
  font-weight: 600;
  text-decoration: none;
  &:hover {
    color: #0099d4;
    .mb-space__creator-bulb-wrap {
      background: #dff2fc;
    }
  }
}

.mb-space__creator-top-inner {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

.mb-space__creator-bulb-wrap {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  border-radius: 50%;
  background: #e8f4fc;
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
}

.mb-space__creator-bulb-img {
  display: block;
  width: 20px;
  height: 20px;
  object-fit: contain;
}

.mb-space__creator-top-t {
  flex-shrink: 0;
  line-height: 1.25;
}

.mb-space__creator-chev {
  flex-shrink: 0;
  opacity: 0.9;
  font-size: 18px;
  line-height: 1;
  font-weight: 400;
}

.mb-space__creator-split-row {
  display: flex;
  align-items: stretch;
  margin: 10px 10px 12px;
  border: 1px solid #e3e5e7;
  border-radius: 8px;
  overflow: hidden;
  background: #eceef1;
}

.mb-space__creator-split-v {
  width: 1px;
  flex-shrink: 0;
  background: #dde0e5;
}

.mb-space__creator-split-cell {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 6px;
  font-size: 13px;
  color: #18191c;
  text-decoration: none;
  background: #fff;
  &:hover:not(.is-muted) {
    background: #f5fbff;
    color: #00a1d6;
  }
  &.is-muted {
    color: #99a2aa;
    cursor: default;
  }
}

.mb-space__creator-cell-ico {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  object-fit: contain;
  display: block;
}

/* 顶栏搜索与统计 */
.mb-space__creator-cell-ico--invert {
  filter: invert(1);
}

.mb-space__creator-split-cell.is-muted .mb-space__creator-cell-ico--invert {
  opacity: 0.42;
}

.mb-space__card--notice {
  padding: 16px 18px 20px;
  min-height: 220px;
  display: flex;
  flex-direction: column;
  background: #f5f6f7;
  border-radius: 10px;
  box-shadow: none;
  border: 1px solid #ebecee;

  > .mb-space__card-title {
    margin: 0 0 8px;
    font-size: 15px;
    font-weight: 700;
    color: #18191c;
  }
}

.mb-space__notice-shell {
  position: relative;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.mb-space__notice-textarea {
  flex: 1;
  width: 100%;
  min-height: 148px;
  box-sizing: border-box;
  margin: 0;
  padding: 2px 4px 10px 2px;
  border: none;
  border-radius: 0;
  font-size: 14px;
  line-height: 1.55;
  color: #18191c;
  background: transparent;
  resize: none;
  outline: none;
  font-family: inherit;
  &::placeholder {
    color: #9499a0;
  }
}

.mb-space__notice-shell.is-notice-focused .mb-space__notice-textarea {
  padding-right: 56px;
  padding-bottom: 22px;
}

.mb-space__notice-count {
  position: absolute;
  right: 2px;
  bottom: 2px;
  font-size: 12px;
  color: #9499a0;
  line-height: 1.2;
  user-select: none;
  pointer-events: none;
}

.mb-space__notice-display {
  margin: 0;
  font-size: 13px;
  line-height: 1.55;
  color: #18191c;
  white-space: pre-wrap;
  word-break: break-word;
}

.mb-space__notice-toast-overlay {
  position: fixed;
  inset: 0;
  z-index: 10050;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.mb-space__notice-toast-box {
  pointer-events: none;
  padding: 14px 32px;
  border-radius: 8px;
  background: #4a4a4a;
  color: #fff;
  font-size: 15px;
  font-weight: 500;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.18);
}

.mb-space__card-title {
  margin: 0 0 10px;
  font-size: 14px;
  font-weight: 600;
  color: #18191c;
}

.mb-space__notice-link {
  font-size: 13px;
  color: #99a2aa;
  text-decoration: none;
  &:hover {
    color: #00a1d6;
  }
}

.mb-space__card--profile {
  padding: 16px 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mb-space__card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin: 0;
}

.mb-space__card-title-row .mb-space__card-title {
  margin: 0;
}

.mb-space__card-edit {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 0;
  border: none;
  background: transparent;
  font-size: 12px;
  color: #99a2aa;
  text-decoration: none;
  &:hover {
    color: #00a1d6;
  }
}

.mb-space__profile-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 13px;
  color: #18191c;
}

.mb-space__profile-ico {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  object-fit: contain;
  display: block;
}

/* UID 行 */
.mb-space__profile-ico--invert {
  filter: invert(1);
}

.mb-space__profile-val {
  min-width: 0;
  word-break: break-all;
}

.mb-space__header {
  position: relative;
  z-index: 2;
  height: 160px;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  overflow: visible;
  background: #222;
}

.mb-space__banner-bg {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center center;
  background-repeat: no-repeat;
  background-color: #222;
}

.mb-space__header-shade {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 80px;
  pointer-events: none;
  background: linear-gradient(
    to top,
    rgba(0, 0, 0, 0.45) 0%,
    rgba(0, 0, 0, 0) 100%
  );
}

.mb-space__header-bar {
  position: relative;
  z-index: 3;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  padding: 0 20px 14px 62px;
  box-sizing: border-box;
  overflow: visible;
}

.mb-space__header-aside {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-shrink: 0;
  margin-left: auto;
  min-height: 34px;
}

.mb-space__profile {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
  min-width: 0;
  flex-wrap: wrap;
  box-sizing: border-box;
}

.mb-space__avatar {
  display: block;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  border: 2px solid #fff;
  object-fit: cover;
  background: #222;
  flex-shrink: 0;
  box-shadow: 0 1px 6px rgba(0, 0, 0, 0.25);
}

.mb-space__profile-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: flex-start;
  gap: 5px;
  min-height: 80px;
}

.mb-space__name-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 8px;
}

.mb-space__name {
  font-size: 21px;
  font-weight: 600;
  line-height: 1.25;
  color: #fff;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.35);
}

.mb-space__level-badge {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  display: block;
  object-fit: contain;
}

.mb-space__badges {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  gap: 6px;
}

.mb-space__gender {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  padding: 0 7px;
  box-sizing: border-box;
  font-size: 14px;
  font-weight: 600;
  line-height: 1;
  letter-spacing: 0.02em;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.35);
}

.mb-space__gender--ico {
  padding: 0;
  min-width: 0;
  height: auto;
  background: transparent;
  text-shadow: none;
}

.mb-space__gender-img {
  display: block;
  width: 18px;
  height: 18px;
  object-fit: contain;
  flex-shrink: 0;
}

.mb-space__sign {
  margin: 0;
  max-width: 760px;
  font-size: 12px;
  line-height: 1.5;
  color: rgba(255, 255, 255, 0.88);
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.35);
}

.mb-space__navbar {
  position: relative;
  z-index: 1;
  box-sizing: border-box;
  background: #fff;
  border-bottom: 1px solid #e5e9ef;
  padding: 0 $mb-space-nav-pad-r 0 $mb-space-nav-pad-l;
}

.mb-space__dock-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px 16px;
  min-height: 50px;
  padding: 2px 0 4px;
}

.mb-space__tabs {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px;
  flex-shrink: 0;
}

.mb-space__dock-gap {
  flex: 1;
  min-width: 12px;
  height: 1px;
}

.mb-space__tab {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 48px;
  padding: 0 12px;
  border: none;
  background: transparent;
  font-size: 13px;
  color: #18191c;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -4px;
  &:hover {
    color: #00a1d6;
  }
  &.is-on {
    color: #00a1d6;
    border-bottom-color: #00a1d6;
    font-weight: 600;
  }
}

.mb-space__tab-ico {
  width: 18px;
  height: 18px;
  object-fit: contain;
  flex-shrink: 0;
}

.mb-space__tab-ico--home {
  filter: invert(52%) sepia(90%) saturate(420%) hue-rotate(99deg)
    brightness(96%) contrast(92%);
}

.mb-space__tab-ico--collect {
  width: 24px;
  height: 24px;
}

.mb-space__tab-badge {
  margin-left: 2px;
  font-size: 12px;
  color: #999;
  font-weight: 400;
}

.mb-space__search {
  display: flex;
  align-items: center;
  border: 1px solid #ccd0d7;
  border-radius: 6px;
  overflow: hidden;
  background: #fff;
  flex-shrink: 0;
}

.mb-space__search-input {
  width: 148px;
  border: none;
  background: transparent;
  padding: 6px 10px;
  font-size: 12px;
  color: #9499a0;
  outline: none;
  &::placeholder {
    color: #99a2aa;
  }
}

.mb-space__search-btn {
  width: 34px;
  height: 30px;
  border: none;
  border-left: 1px solid #e5e9ef;
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.mb-space__search-ico {
  width: 13px;
  height: 13px;
  border: 2px solid #99a2aa;
  border-radius: 50%;
  box-sizing: border-box;
  position: relative;
  &::after {
    content: "";
    position: absolute;
    right: -4px;
    bottom: -4px;
    width: 5px;
    height: 2px;
    background: #99a2aa;
    transform: rotate(45deg);
    border-radius: 1px;
  }
}

.mb-space__stats {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 18px 26px;
  flex-shrink: 0;
}

.mb-space__stat {
  text-align: center;
  min-width: 44px;
}

.mb-space__stat--link {
  margin: 0;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  font: inherit;
  color: inherit;

  &:hover:not(:disabled) .mb-space__stat-v {
    color: #00a1d6;
  }

  &:disabled {
    cursor: default;
    opacity: 1;
  }
}

.mb-space__stat-k {
  display: block;
  font-size: 12px;
  line-height: 1.2;
  color: #99a2aa;
}

.mb-space__stat-v {
  display: block;
  margin-top: 2px;
  font-size: 14px;
  font-weight: 500;
  color: #222;
}

.mb-space__body {
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  padding: 0;
  min-width: 0;
  min-height: 0;
  background: #fff;
}

.mb-space__main {
  flex: 1;
  min-width: 0;
  box-sizing: border-box;
  background: #fff;
  border-radius: 0;
  padding: 0;
  box-shadow: none;
}

.mb-space__sec-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px 16px;
  margin-bottom: 0;
  border-bottom: 1px solid #e5e9ef;
  padding-bottom: 12px;
}

.mb-space__sec-left {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 16px;
  flex: 1;
  min-width: 0;
}

.mb-space__sec-title {
  margin: 0;
  padding: 0;
  border: 0;
  display: inline-flex;
  align-items: baseline;
  flex-shrink: 0;
  column-gap: 9px;
  row-gap: 4px;
}

.mb-space__sec-title-w {
  font-size: 18px;
  font-weight: 600;
  color: #18191c;
  line-height: 1.25;
}

.mb-space__sec-dot,
.mb-space__sec-count {
  font-size: 13px;
  font-weight: 400;
  color: #9499a0;
}

.mb-space__subtabs {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.mb-space__subtab {
  margin: 0;
  padding: 8px 16px;
  min-height: 34px;
  border: none;
  border-radius: 8px;
  background: #f1f2f3;
  color: #18191c;
  font-size: 13px;
  font-weight: 400;
  line-height: 1.3;
  cursor: pointer;
  box-sizing: border-box;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s ease, color 0.15s ease;
  &:hover {
    background: #e8e9ec;
  }
  &.is-on {
    background: #00a1d6;
    color: #fff;
    font-weight: 500;
  }
}

.mb-space__sec-right {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-shrink: 0;
}

.mb-space__play-all,
.mb-space__sec-more {
  margin: 0;
  height: 32px;
  padding: 0 14px;
  border-radius: 6px;
  border: 1px solid #ccd0d7;
  background: #fff;
  color: #18191c;
  font-size: 13px;
  font-weight: 400;
  line-height: 1;
  cursor: pointer;
  box-sizing: border-box;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  &:hover:not(:disabled) {
    border-color: #00aeec;
    color: #00aeec;
  }
  &:disabled {
    opacity: 0.45;
    cursor: default;
  }
}

.mb-space__play-ico {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.mb-space__sec-more-arr {
  font-size: 15px;
  line-height: 1;
  opacity: 0.65;
  margin-left: 1px;
}

/* 主页 Tab：标题 + 排序/播放 */
.mb-space__contrib-outer {
  display: grid;
  grid-template-columns: 136px minmax(0, 1fr);
  grid-template-rows: auto auto;
  column-gap: 24px;
  row-gap: 26px;
  width: 100%;
  box-sizing: border-box;
}

.mb-space__contrib-outer > .mb-space__dyn-sidenav {
  grid-column: 1;
  grid-row: 1 / span 2;
  align-self: start;
}

.mb-space__contrib-right-head {
  grid-column: 2;
  grid-row: 1;
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  grid-template-rows: auto auto;
  column-gap: 16px;
  row-gap: 8px;
  align-items: end;
}

.mb-space__contrib-right-head > .mb-space__contrib-h2 {
  grid-column: 1;
  grid-row: 1;
  align-self: center;
}

.mb-space__contrib-actions {
  grid-column: 2;
  grid-row: 1 / span 2;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  justify-content: flex-end;
  gap: 10px;
  margin-right: 28px;
  margin-bottom: 2px;
  padding-top: 6px;
}

.mb-space__contrib-right-head > .mb-space__contrib-toolbar {
  grid-column: 1;
  grid-row: 2;
}

.mb-space__contrib-feed {
  grid-column: 2;
  grid-row: 2;
  min-width: 0;
}

.mb-space__contrib-right-head--article {
  grid-column: 2;
  grid-row: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  padding: 0;
}

.mb-space__contrib-feed--article {
  grid-column: 2;
  grid-row: 2;
}

.mb-space__contrib-h2 {
  margin: 0;
  color: var(--text1);
  font-weight: 600;
  font-size: 24px;
  line-height: 1.25;
}

.mb-space__contrib-toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  flex-wrap: wrap;
  gap: 12px 16px;
  margin-top: 0;
  padding-top: 0;
  border-top: none;
}

.mb-space__contrib-view {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  background: #f1f2f3;
  border-radius: 8px;
  flex-shrink: 0;
}

.mb-space__contrib-view-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0;
  padding: 6px 9px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #9499a0;
  cursor: pointer;
  line-height: 0;
  transition:
    background 0.12s ease,
    color 0.12s ease;
  &:hover {
    color: #61666d;
  }
  &.is-on {
    background: #fff;
    color: #00a1d6;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
  }
}

.mb-space__contrib-view-ico {
  width: 18px;
  height: 18px;
  display: block;
}

.mb-space__contrib-article-inner {
  text-align: center;
  max-width: 280px;
}

.mb-space__contrib-article-t {
  margin: 0 0 8px;
  font-size: 16px;
  font-weight: 600;
  color: #18191c;
}

.mb-space__article-fav-grid {
  margin-top: 14px;
}

.mb-space__vcell--article-fav .mb-space__vthumb-wrap {
  aspect-ratio: 16 / 10;
}

.mb-space__vcell--article-fav .mb-space__vtitle {
  margin: 8px 0 0;
  font-size: 14px;
  line-height: 1.4;
  color: #18191c;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.mb-space__vmeta--article-fav {
  margin: 4px 0 0;
  font-size: 12px;
  color: #9499a0;
  line-height: 1.35;
}

.mb-space__article-list {
  list-style: none;
  margin: 0;
  padding: 0;
  width: 100%;
  box-sizing: border-box;
}

.mb-space__article-item {
  margin: 0;
  border-bottom: 1px solid #f1f2f3;

  &:last-child {
    border-bottom: none;
  }
}

.mb-space__article-link {
  display: flex;
  align-items: flex-start;
  gap: 20px;
  padding: 14px 4px 14px 0;
  text-decoration: none;
  color: inherit;
  border-radius: 6px;
  transition: background 0.15s ease;

  &:hover {
    background: #f6f7f8;

    .mb-space__article-title {
      color: #00aeec;
    }
  }
}

.mb-space__article-cover-wrap {
  flex-shrink: 0;
  width: 173px;
  height: 98px;
  border-radius: 4px;
  overflow: hidden;
  background: #f1f2f3;
}

.mb-space__article-cover {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.mb-space__article-body {
  flex: 1;
  min-width: 0;
  padding-top: 2px;
}

.mb-space__article-title {
  margin: 0 0 10px;
  font-size: 15px;
  font-weight: 500;
  line-height: 1.4;
  color: #18191c;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
  transition: color 0.15s ease;
}

.mb-space__article-meta {
  margin: 0;
  font-size: 12px;
  line-height: 1.35;
  color: #9499a0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.mb-space__article-meta-dot {
  color: #c9ccd0;
}

.mb-space__contrib-article-sub {
  margin: 0;
  font-size: 13px;
  color: #9499a0;
  line-height: 1.5;
}

.mb-space__video-grid {
  list-style: none;
  margin: 14px 0 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(196px, 1fr));
  gap: 16px 14px;
  width: 100%;
  box-sizing: border-box;
}

.mb-space__video-grid--contrib:not(.mb-space__video-grid--list) {
  grid-template-columns: repeat(auto-fill, 148px);
  gap: 14px;
  justify-content: start;
}

.mb-space__video-grid--list {
  grid-template-columns: 1fr;
  gap: 14px;
}

.mb-space__vdesc {
  margin: 0;
  font-size: 13px;
  line-height: 1.45;
  color: #9499a0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

.mb-space__vmeta-stat {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: #9499a0;
}

.mb-space__vmeta-stat-ico {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  object-fit: contain;
  opacity: 0.85;
}

.mb-space__contrib-feed .mb-space__video-grid {
  margin-top: 0;
}

/* 投稿网格缩略 148×104 */
.mb-space__vthumb-wrap--dyn {
  width: 148px;
  max-width: 100%;
  aspect-ratio: unset;
  height: 104px;
}

.mb-space__video-grid--contrib:not(.mb-space__video-grid--list)
  .mb-space__vtext-col {
  max-width: 148px;
}

.mb-space__video-grid--contrib.mb-space__video-grid--list
  .mb-space__vtext-col {
  max-width: none;
}

.mb-space__vcell {
  margin: 0;
  padding: 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-self: stretch;
  height: 100%;
}

.mb-space__vcell-link {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  max-width: 100%;
  text-decoration: none;
  color: inherit;
  &:hover .mb-space__vtitle {
    color: #00a1d6;
  }
}

.mb-space__vthumb-wrap {
  position: relative;
  border-radius: 8px;
  overflow: hidden;
  background: #eee;
  width: 100%;
  aspect-ratio: 16 / 9;
}

.mb-space__vthumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

/* 稍后再看 hover */
.mb-space__vthumb-default {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 10px;
  padding: 26px 8px 6px;
  box-sizing: border-box;
  font-size: 12px;
  line-height: 1.2;
  color: #fff;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.45);
  background: linear-gradient(
    to top,
    rgba(0, 0, 0, 0.68) 0%,
    rgba(0, 0, 0, 0) 100%
  );
  pointer-events: none;
  transition: opacity 0.18s ease, visibility 0.18s ease;
}

.mb-space__vthumb-stats-l {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: center;
  gap: 8px 10px;
  min-width: 0;
}

.mb-space__vthumb-stat {
  display: inline-flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
  flex-shrink: 0;
}

/* 动态卡片 */
.mb-space__vthumb-wrap--dyn .mb-space__vthumb-default {
  padding: 22px 6px 5px;
  font-size: 11px;
}

.mb-space__vthumb-wrap--dyn .mb-space__vstat-ico {
  width: 15px;
  height: 15px;
}

.mb-space__vthumb-wrap--dyn .mb-space__vdur {
  font-size: 10px;
  line-height: 16px;
  padding: 0;
}

.mb-space__vstat-ico {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  object-fit: contain;
  display: block;
  opacity: 0.95;
  mix-blend-mode: screen;
}

.mb-space__vdur {
  flex-shrink: 0;
  padding: 0;
  font-size: 11px;
  line-height: 18px;
  color: #fff;
  background: transparent;
  border-radius: 0;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.9);
}

/* 视频格 hover */
.mb-space__vthumb-later {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 3;
  margin: 0;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  opacity: 0;
  visibility: hidden;
  transition: opacity 0.18s ease, visibility 0.18s ease;
  display: block;
  width: fit-content;
  max-width: calc(100% - 16px);
  overflow: visible;
}

.mb-space__vthumb-later-inner {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: fit-content;
}

/* 收藏 Tab */
.mb-space__vlater-ico-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 26px;
  height: 26px;
  padding: 0;
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.58);
  box-sizing: border-box;
}

.mb-space__vlater-ico {
  width: 16px;
  height: 16px;
  object-fit: contain;
  display: block;
  mix-blend-mode: screen;
  opacity: 0.95;
}

/* 收藏搜索组 */
.mb-space__vlater-txt {
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(calc(-50% - 8px));
  margin-top: 5px;
  font-size: 12px;
  line-height: 1.2;
  color: #fff;
  font-weight: 500;
  white-space: nowrap;
  text-shadow: none;
  padding: 4px 8px;
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.58);
  box-sizing: border-box;
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
  transition: opacity 0.16s ease, visibility 0.16s ease;
}

.mb-space__vthumb-later:hover .mb-space__vlater-txt,
.mb-space__vthumb-later:focus-visible .mb-space__vlater-txt {
  opacity: 1;
  visibility: visible;
  pointer-events: auto;
}

.mb-space__vthumb-wrap:hover .mb-space__vthumb-default {
  opacity: 0;
  visibility: hidden;
}

.mb-space__vthumb-wrap:hover .mb-space__vthumb-later {
  opacity: 1;
  visibility: visible;
}

/* 收藏格 hover */
.mb-space__dyn-vdur,
.mb-space__dyn-vplay {
  transition:
    opacity 0.18s ease,
    visibility 0.18s ease;
}

.mb-space__dyn-vbox-l:hover .mb-space__dyn-vdur,
.mb-space__dyn-vbox-l:hover .mb-space__dyn-vplay {
  opacity: 0;
  visibility: hidden;
}

.mb-space__dyn-vbox-l:hover .mb-space__vthumb-later {
  opacity: 1;
  visibility: visible;
}

.mb-space__vtext-col {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-width: 0;
  margin-top: 2px;
}

.mb-space__vtitle {
  flex: 1 1 auto;
  margin: 6px 0 0;
  min-height: calc(1.35em * 2);
  font-size: 13px;
  line-height: 1.35;
  color: #222;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}


.mb-space__vmeta {
  flex-shrink: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 6px;
  margin: 0;
  margin-top: auto;
  padding-top: 4px;
  font-size: 12px;
  line-height: 1.35;
  color: #9499a0;
}

.mb-space__vmeta-lock {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
  color: #c0c4cc;
}

.mb-space__vmeta-privacy {
  color: #9499a0;
}

.mb-space__vmeta-sep {
  color: #c9ccd0;
  user-select: none;
}

.mb-space__vmeta-date {
  color: #9499a0;
  white-space: nowrap;
}

/* 列表模式 column 布局 */
.mb-space__video-grid--contrib.mb-space__video-grid--list .mb-space__vcell {
  padding: 14px 0;
  border-bottom: 1px solid #f1f2f3;
}

.mb-space__video-grid--contrib.mb-space__video-grid--list
  .mb-space__vcell:first-child {
  padding-top: 4px;
}

.mb-space__vcell-link.mb-space__vcell-link--list {
  flex-direction: row;
  align-items: stretch;
  gap: 12px;
}

.mb-space__vthumb-wrap.mb-space__vthumb-wrap--list {
  width: 193px;
  max-width: none;
  flex: 0 0 193px;
  height: 109px;
  aspect-ratio: unset;
}

.mb-space__vlist-dur {
  position: absolute;
  right: 6px;
  bottom: 5px;
  z-index: 2;
  padding: 0;
  border-radius: 0;
  background: transparent;
  color: #fff;
  font-size: 12px;
  line-height: 1.3;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.9);
  pointer-events: none;
  transition:
    opacity 0.18s ease,
    visibility 0.18s ease;
}

.mb-space__vthumb-wrap:hover .mb-space__vlist-dur {
  opacity: 0;
  visibility: hidden;
}

.mb-space__vtext-col.mb-space__vtext-col--list {
  flex: 1 1 auto;
  align-self: stretch;
  margin-top: 0;
  min-height: 109px;
  padding: 4px 0 2px;
  box-sizing: border-box;
}

.mb-space__vlist-main {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.mb-space__vtext-col.mb-space__vtext-col--list .mb-space__vtitle--list {
  flex: 0 0 auto;
  margin: 0;
  min-height: 0;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.35;
  -webkit-line-clamp: 1;
  line-clamp: 1;
}

.mb-space__vtext-col.mb-space__vtext-col--list .mb-space__vmeta {
  margin-top: auto;
  padding-top: 0;
  flex-wrap: nowrap;
  align-items: center;
  gap: 14px;
}

.mb-space__vtext-col.mb-space__vtext-col--list .mb-space__vmeta-stat-ico {
  width: 18px;
  height: 18px;
  mix-blend-mode: normal;
  opacity: 1;
  /* 图标 PNG 着色 #9499a0 */
  filter: brightness(0) saturate(100%) invert(68%) sepia(6%) saturate(462%)
    hue-rotate(169deg) brightness(93%) contrast(88%);
}

/* 收藏 Tab 布局 */
.mb-space__collect-outer {
  display: grid;
  grid-template-columns: 160px minmax(0, 1fr);
  gap: 24px;
  width: 100%;
  box-sizing: border-box;
  padding-bottom: 24px;
  overflow: visible;
}

.mb-space__collect-sidenav {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 4px 8px 0 0;
  overflow: visible;
}

.mb-space__collect-nav-head {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #18191c;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  text-align: left;
  &:hover {
    background: rgba(0, 0, 0, 0.04);
  }
}

.mb-space__collect-nav-chev {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  color: #9499a0;
  transition: transform 0.28s ease;
}

.mb-space__collect-nav-head.is-open .mb-space__collect-nav-chev {
  transform: rotate(180deg);
}

.mb-space__collect-nav-block {
  overflow: visible;
}

.mb-space__collect-folder-collapse {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 0.28s ease;
  overflow: hidden;

  &.is-open {
    grid-template-rows: 1fr;
    overflow: visible;
  }

  .mb-space__collect-folder-list {
    min-height: 0;
    overflow: hidden;
  }

  &.is-open .mb-space__collect-folder-list {
    overflow: visible;
  }
}

.mb-space__collect-folder-list {
  margin: 0;
  padding: 0 0 4px 10px;
  list-style: none;
  overflow: visible;
}

.mb-space__collect-folder-new-li {
  margin-bottom: 4px;
}

.mb-space__collect-folder-new {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 10px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #61666d;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
  line-height: 1.3;
  &:hover {
    background: rgba(0, 0, 0, 0.04);
    color: #18191c;
  }
}

.mb-space__collect-folder-btn {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mb-space__collect-folder-ico {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  object-fit: contain;
  opacity: 0.72;
}

.mb-space__collect-folder-btn.is-on .mb-space__collect-folder-ico {
  opacity: 1;
}

.mb-space__collect-folder-btn.is-on .mb-space__collect-folder-more-dots i {
  background: #00aeec;
}

.mb-space__collect-folder-btn-title {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mb-space__collect-folder-count {
  flex-shrink: 0;
  font-size: 13px;
  line-height: 20px;
  color: #9499a0;
  margin-left: auto;
  font-variant-numeric: tabular-nums;
}

.mb-space__collect-folder-trail {
  position: relative;
  flex-shrink: 0;
  width: 32px;
  height: 20px;
  margin-left: auto;
}

.mb-space__collect-folder-trail .mb-space__collect-folder-count {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  width: 100%;
  height: 100%;
  margin-left: 0;
  text-align: right;
}

.mb-space__collect-folder-count.is-hidden {
  visibility: hidden;
}

.mb-space__collect-folder-btn.is-on .mb-space__collect-folder-count {
  color: #00aeec;
}

.mb-space__collect-folder-item {
  position: relative;
  list-style: none;
  overflow: visible;
}

.mb-space__collect-folder-more {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  pointer-events: none;

  &.is-active {
    opacity: 1;
    pointer-events: auto;
  }
}

.mb-space__collect-folder-more-dots {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  width: 14px;
  height: 18px;

  i {
    display: block;
    width: 3px;
    height: 3px;
    border-radius: 50%;
    background: #9499a0;
  }
}

.mb-space__collect-folder-btn:hover .mb-space__collect-folder-more-dots i,
.mb-space__collect-folder-more:hover .mb-space__collect-folder-more-dots i {
  background: #61666d;
}

.mb-space__collect-folder-menu {
  position: absolute;
  top: 50%;
  left: calc(100% - 6px);
  transform: translateY(-50%);
  z-index: 100;
  min-width: 108px;
  margin: 0;
  padding: 6px 0;
  list-style: none;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  box-sizing: border-box;

  /* 与三点之间的透明桥，避免移入菜单时 hover 断开 */
  &::before {
    content: "";
    position: absolute;
    top: -8px;
    bottom: -8px;
    right: 100%;
    width: 14px;
  }

  button {
    display: block;
    width: 100%;
    padding: 9px 16px;
    border: none;
    background: transparent;
    color: #18191c;
    font-size: 14px;
    line-height: 1.3;
    text-align: center;
    cursor: pointer;
    white-space: nowrap;

    &:hover {
      background: #f6f7f8;
    }
  }
}

.mb-space__collect-folder-btn {
  width: 100%;
  padding: 9px 10px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #61666d;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
  line-height: 1.3;
  overflow: visible;
  transition:
    background 0.15s ease,
    color 0.15s ease;
  &:hover {
    background: rgba(0, 0, 0, 0.04);
    color: #18191c;
  }
  &.is-on {
    background: rgba(0, 174, 236, 0.1);
    color: #00aeec;
    font-weight: 500;
  }
}

.mb-space__collect-later-btn {
  width: 100%;
  padding: 10px 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #18191c;
  font-size: 14px;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
  &:hover {
    background: rgba(0, 0, 0, 0.04);
  }
  &.is-on {
    background: rgba(0, 174, 236, 0.1);
    color: #00aeec;
  }
}

.mb-space__collect-main {
  min-width: 0;
}

.mb-space__collect-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 18px;
  border-bottom: 1px solid #f1f2f3;
}

.mb-space__collect-head--later {
  border-bottom: none;
  padding-bottom: 12px;
}

.mb-space__collect-folder-intro {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.mb-space__collect-folder-cover {
  flex: 0 0 72px;
  width: 72px;
  height: 72px;
  border-radius: 6px;
  overflow: hidden;
  background: #f1f2f3;
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
}

.mb-space__collect-folder-cover-ph {
  display: block;
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, #e3e5e7 0%, #f6f7f8 100%);
}

.mb-space__collect-folder-title {
  margin: 0 0 6px;
  font-size: 18px;
  font-weight: 600;
  color: #18191c;
  line-height: 1.3;
}

.mb-space__collect-folder-meta {
  margin: 0;
  font-size: 13px;
  color: #9499a0;
  line-height: 1.4;
}

.mb-space__collect-folder-meta-sep {
  margin: 0 6px;
}

.mb-space__collect-head-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.mb-space__play-all--collect {
  min-width: 108px;
  height: 36px;
  padding: 0 18px;
  border: none;
  border-radius: 6px;
  background: #00aeec;
  color: #fff;
  font-size: 14px;
  &:hover:not(:disabled) {
    background: #00b5e7;
    border-color: transparent;
    color: #fff;
  }
}

.mb-space__collect-batch {
  height: 36px;
  padding: 0 14px;
  border: 1px solid #ccd0d7;
  border-radius: 6px;
  background: #fff;
  color: #61666d;
  font-size: 14px;
  cursor: pointer;
  &:disabled {
    opacity: 0.45;
    cursor: default;
  }
}

.mb-space__collect-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  padding: 16px 0 14px;
}

.mb-space__collect-search {
  display: flex;
  align-items: center;
  gap: 0;
  flex-shrink: 0;
}

.mb-space__collect-search-scope {
  position: relative;
  display: flex;
  align-items: center;
}

.mb-space__collect-search-select {
  appearance: none;
  height: 32px;
  padding: 0 28px 0 12px;
  border: 1px solid #e3e5e7;
  border-right: none;
  border-radius: 6px 0 0 6px;
  background: #fff;
  color: #61666d;
  font-size: 13px;
  cursor: pointer;
}

.mb-space__collect-search-chev {
  position: absolute;
  right: 8px;
  width: 14px;
  height: 14px;
  pointer-events: none;
  color: #9499a0;
}

.mb-space__collect-search-input {
  width: 180px;
  height: 32px;
  padding: 0 12px;
  border: 1px solid #e3e5e7;
  border-left: none;
  border-right: none;
  background: #fff;
  color: #18191c;
  font-size: 13px;
  outline: none;
  &::placeholder {
    color: #c9ccd0;
  }
  &:focus {
    border-color: #00aeec;
  }
}

.mb-space__collect-search-btn {
  width: 36px;
  height: 32px;
  padding: 0;
  border: 1px solid #e3e5e7;
  border-radius: 0 6px 6px 0;
  background: #fff;
  color: #9499a0;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  svg {
    width: 18px;
    height: 18px;
  }
  &:disabled {
    opacity: 0.5;
    cursor: default;
  }
}

.mb-space__collect-grid {
  grid-template-columns: repeat(auto-fill, minmax(196px, 1fr));
  gap: 16px 14px;
}

.mb-space__collect-sk {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.mb-space__collect-sk-thumb {
  width: 100%;
  aspect-ratio: 16 / 9;
  border-radius: 8px;
  background: linear-gradient(90deg, #f1f2f3 25%, #e3e5e7 50%, #f1f2f3 75%);
  background-size: 200% 100%;
  animation: mb-space-collect-shimmer 1.2s ease-in-out infinite;
}

.mb-space__collect-sk-line {
  height: 12px;
  border-radius: 4px;
  background: #f1f2f3;
  &--t {
    width: 92%;
  }
  &--s {
    width: 48%;
  }
}

.mb-space__collect-sk-foot {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mb-space__collect-sk-avatar {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  flex-shrink: 0;
  background: #f1f2f3;
}

@keyframes mb-space-collect-shimmer {
  0% {
    background-position: 100% 0;
  }
  100% {
    background-position: -100% 0;
  }
}

.mb-space__collect-later-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #18191c;
}

.mb-space__collect-later-empty {
  flex-direction: column;
  align-items: center;
  padding-top: 24px;
  width: 100%;
}

.mb-space__collect-later-hint {
  margin: 12px 0 0;
  text-align: center;
  font-size: 14px;
  color: #9499a0;
  width: 100%;
}



.mb-space__dynamic {
  padding: 0 0 24px;
}

.mb-space__dyn-layout {
  display: flex;
  align-items: flex-start;
  gap: 0 24px;
}

.mb-space__dyn-sidenav {
  flex: 0 0 136px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 4px 10px 0 0;
  box-sizing: border-box;
}

.mb-space__dyn-sub {
  width: 100%;
  padding: 12px 14px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: #18191c;
  font-size: 15px;
  font-weight: 500;
  letter-spacing: 0.02em;
  cursor: pointer;
  text-align: left;
  line-height: 1.25;
  transition:
    background 0.15s ease,
    color 0.15s ease;
  &:hover {
    background: rgba(0, 0, 0, 0.04);
  }
  &.is-on {
    background: #00aeec;
    color: #fff;
    font-weight: 600;
    box-shadow: 0 1px 3px rgba(0, 174, 236, 0.35);
  }
}

.mb-space__dyn-sub--split {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.mb-space__dyn-sub-count {
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 500;
  color: #9499a0;
}

.mb-space__dyn-sub.is-on .mb-space__dyn-sub-count {
  color: rgba(255, 255, 255, 0.9);
}

.mb-space__dyn-feed {
  flex: 1 1 0;
  min-width: 0;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.mb-space__dyn-card {
  position: relative;
  border: 1px solid #e5e9ef;
  border-radius: 10px;
  background: #fff;
  padding: 16px 18px 14px;
  box-sizing: border-box;
}

.mb-space__dyn-pin-badge {
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 6;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px 2px 6px;
  border-radius: 4px;
  background: rgba(0, 174, 236, 0.1);
  border: 1px solid rgba(0, 174, 236, 0.35);
  box-sizing: border-box;
  pointer-events: none;
}

.mb-space__dyn-pin-badge-ico {
  display: block;
  width: 16px;
  height: 16px;
  object-fit: contain;
  flex-shrink: 0;
}

.mb-space__dyn-pin-badge-txt {
  font-size: 12px;
  font-weight: 600;
  color: #00aeec;
  line-height: 1.2;
}

.mb-space__dyn-card:has(.mb-space__dyn-head-tools) .mb-space__dyn-pin-badge {
  right: 36px;
}

$dyn-head-avatar: 48px;
$dyn-head-gap: 12px;
$dyn-body-indent: $dyn-head-avatar + $dyn-head-gap;

.mb-space__dyn-body {
  margin-left: $dyn-body-indent;
  min-width: 0;
}

.mb-space__dyn-body--link {
  display: block;
  color: inherit;
  text-decoration: none;
  cursor: pointer;

  &:hover .mb-space__dyn-textline--title {
    color: #fb7299;
  }
}

.mb-space__dyn-head {
  display: flex;
  align-items: flex-start;
  gap: $dyn-head-gap;
  margin-bottom: 12px;
}

.mb-space__dyn-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  border: 1px solid #f0f0f0;
}

.mb-space__dyn-head-main {
  flex: 1;
  min-width: 0;
}

.mb-space__dyn-name {
  font-size: 15px;
  font-weight: 600;
  color: #fb7299;
  line-height: 1.3;
}

.mb-space__dyn-subline {
  margin-top: 4px;
  font-size: 12px;
  color: #9499a0;
  line-height: 1.4;
}

.mb-space__dyn-date {
  margin-right: 8px;
}

.mb-space__dyn-verb {
  color: #9499a0;
}

.mb-space__dyn-head-tools {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.mb-space__dyn-more {
  border: none;
  background: transparent;
  color: #99a2aa;
  font-size: 18px;
  line-height: 1;
  padding: 0 2px;
  cursor: pointer;
  &:hover {
    color: #00aeec;
  }
}

.mb-space__dyn-more-wrap {
  position: relative;
  line-height: 1;
}

.mb-space__dyn-more-menu {
  display: none;
  position: absolute;
  right: 0;
  top: calc(100% + 4px);
  z-index: 30;
  min-width: 112px;
  padding: 6px 0;
  margin: 0;
  list-style: none;
  background: #fff;
  border-radius: 8px;
  box-shadow: 2px 4px 16px rgba(0, 0, 0, 0.12);
  box-sizing: border-box;
  &::before {
    content: "";
    position: absolute;
    left: 0;
    right: 0;
    top: -10px;
    height: 10px;
  }
}

.mb-space__dyn-more-wrap:hover .mb-space__dyn-more-menu,
.mb-space__dyn-more-menu:hover {
  display: block;
}

.mb-space__dyn-more-item {
  display: block;
  width: 100%;
  margin: 0;
  padding: 10px 18px;
  border: none;
  background: transparent;
  text-align: left;
  font-size: 14px;
  color: #61666d;
  cursor: pointer;
  line-height: 1.35;
  box-sizing: border-box;
  &:hover {
    background: rgba(0, 0, 0, 0.04);
    color: #18191c;
  }
}

.mb-space__dyn-more-item--del {
  color: #61666d;
}

.mb-space__dyn-textline {
  margin: 0 0 8px;
  font-size: 14px;
  line-height: 1.65;
  color: #18191c;
  word-break: break-word;
}

.mb-space__dyn-textline--title {
  font-size: 15px;
  font-weight: 600;
  line-height: 1.5;
  margin-bottom: 4px;
}

.mb-space__dyn-img-block {
  margin-bottom: 0;
}

.mb-space__dyn-img-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.mb-space__dyn-img-cell {
  flex: 0 0 auto;
  width: 120px;
  height: 120px;
  border-radius: 8px;
  overflow: hidden;
  background: #f4f4f4;
}

.mb-space__dyn-img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

/* 动态评论 + 精选弹窗 */
.mb-space__dyn-video-info {
  margin-top: 0;
  border: 1px solid #e5e9ef;
  border-radius: 8px;
  background: #fff;
  overflow: hidden;
  box-sizing: border-box;
}

.mb-space__dyn-act-bar {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  width: 100%;
  margin-top: 8px;
  padding: 2px 0 10px;
  box-sizing: border-box;
  border-bottom: 1px solid #f0f1f2;
}

.mb-space__dyn-act-bar__btn {
  display: inline-flex;
  flex-direction: row;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  padding: 8px 4px;
  margin: 0;
  border: none;
  background: none;
  color: #9499a0;
  cursor: pointer;
  transition: color 0.12s ease;
  &:hover {
    color: #00aeec;
    .mb-space__dyn-ico-act__share {
      filter: brightness(0) saturate(100%) invert(55%) sepia(93%)
        saturate(2456%) hue-rotate(163deg) brightness(98%) contrast(101%);
      opacity: 1;
    }
    .mb-space__dyn-collect-ico-wrap:not(.is-on) .mb-space__dyn-ico-act__collect {
      filter: brightness(0) saturate(100%) invert(55%) sepia(93%)
        saturate(2456%) hue-rotate(163deg) brightness(98%) contrast(101%);
      opacity: 1;
    }
  }
  &.is-on {
    color: #00aeec;
    font-weight: 600;
  }
}

.mb-space__dyn-act-bar__txt {
  font-size: 13px;
  line-height: 1.2;
}

.mb-space__dyn-act-bar__num {
  margin-left: 2px;
  font-weight: 500;
  color: #99a2aa;
  font-size: 12px;
}

.mb-space__dyn-ico-act--liked {
  color: #00aeec;
  .mb-space__dyn-act-bar__num {
    color: #00aeec;
  }
  &:hover {
    color: #0099cc;
    .mb-space__dyn-ico-act__svg--like-solid {
      opacity: 1;
    }
  }
}

.mb-space__dyn-collect-ico-wrap {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  box-sizing: border-box;
  transition:
    background 0.12s ease,
    border-color 0.12s ease;

  &.is-on {
    background: #00aeec;
    .mb-space__dyn-ico-act__collect {
      filter: brightness(0) invert(1);
      opacity: 1;
    }
  }
}

.mb-space__dyn-ico-act__collect {
  display: block;
  width: 16px;
  height: 16px;
  object-fit: contain;
  flex-shrink: 0;
  opacity: 0.88;
  filter: brightness(0) saturate(100%) invert(58%) sepia(5%) saturate(464%)
    hue-rotate(171deg) brightness(92%) contrast(86%);
  transition:
    filter 0.12s ease,
    opacity 0.12s ease;
}

.mb-space__dyn-ico-act__share {
  display: block;
  width: 16px;
  height: 16px;
  object-fit: contain;
  opacity: 0.88;
  filter: brightness(0) saturate(100%) invert(58%) sepia(5%) saturate(464%)
    hue-rotate(171deg) brightness(92%) contrast(86%);
  transition:
    filter 0.12s ease,
    opacity 0.12s ease;
}

.mb-space__dyn-ico-act__svg {
  display: block;
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  opacity: 0.92;
}

.mb-space__dyn-ico-act__svg--like-solid {
  opacity: 1;
}

.mb-space__dyn-cmt-panel {
  margin-top: 12px;
  padding: 0 2px 4px;
  box-sizing: border-box;
}

.mb-space__dyn-cmt-head--bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: nowrap;
  min-width: 0;
}

.mb-space__dyn-cmt-head__left {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
  gap: 18px;
  min-width: 0;
  flex: 1;
}

.mb-space__dyn-cmt-title {
  margin: 0;
  flex: 0 0 auto;
  font-size: 18px;
  font-weight: 600;
  color: #18191c;
  line-height: 1.2;
  display: flex;
  align-items: baseline;
}

.mb-space__dyn-cmt-count {
  margin-left: 4px;
  font-size: 13px;
  font-weight: 600;
  color: #9499a0;
  line-height: 1;
}

.mb-space__dyn-cmt-curated-label {
  font-size: 13px;
  font-weight: 600;
  color: #18191c;
  line-height: 1.2;
}

.mb-space__dyn-cmt-sort-row {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  line-height: 1.2;
  padding-top: 1px;
}

.mb-space__dyn-cmt-tab {
  border: none;
  background: none;
  padding: 0;
  font-size: 13px;
  font-weight: 400;
  line-height: 1.2;
  color: #9499a0;
  cursor: pointer;
  &.is-on {
    color: #18191c;
    font-weight: 600;
  }
  &:hover:not(.is-on) {
    color: #61666d;
  }
}

$vd-cmt-blue-cp: #00a1d6;
$vd-cmt-blue-hover-cp: #0090c2;

.mb-space__dyn-cmt-head__actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
}

.mb-space__dyn-cmt-pending-link {
  flex-shrink: 0;
  padding: 0 4px;
  font-size: 13px;
  font-weight: 500;
  line-height: 1.2;
  color: $vd-cmt-blue-cp;
  text-decoration: none;
  white-space: nowrap;
  &:hover {
    color: $vd-cmt-blue-hover-cp;
  }
}

.mb-space__dyn-cmt-head-more-wrap {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  align-self: center;
}

.mb-space__dyn-cmt-head-menu-trigger {
  padding: 4px 6px;
}

.mb-space__dyn-cmt-sep {
  color: #e3e5e7;
  user-select: none;
}

$icons-cmt-w: 1000px;
$icons-cmt-h: 1000px;

.mb-space__dyn-cmt-panel .vd-cmt-composer--mb {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 14px;
}

.mb-space__dyn-cmt-panel .vd-cmt-avatar {
  border-radius: 50%;
  flex-shrink: 0;
  object-fit: cover;
}

.mb-space__dyn-cmt-panel .vd-cmt-avatar--mb {
  align-self: center;
}

.mb-space__dyn-cmt-panel .vd-cmt-mb-composer-main {
  flex: 1;
  min-width: 0;
}

.mb-space__dyn-cmt-panel .vd-cmt-mb-editor-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 76px;
  column-gap: 10px;
  align-items: stretch;
}

.mb-space__dyn-cmt-panel .vd-cmt-uni-inbox {
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

.mb-space__dyn-cmt-panel .vd-cmt-uni-inbox:focus-within {
  border-color: $vd-cmt-blue-cp;
  box-shadow: 0 0 0 3px rgba(0, 161, 214, 0.14);
}

.mb-space__dyn-cmt-panel .vd-cmt-uni-inbox__field {
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

.mb-space__dyn-cmt-panel .vd-cmt-uni-inbox__guest.vd-cmt-login-hint {
  min-height: 72px;
  background: transparent;
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

.mb-space__dyn-cmt-panel .vd-cmt-login-hint-muted {
  color: #9499a0;
}

.mb-space__dyn-cmt-panel .vd-cmt-login-hint-btn {
  display: inline-block;
  flex-shrink: 0;
  margin: 0 4px;
  padding: 2px 12px;
  border: none;
  border-radius: 4px;
  background: $vd-cmt-blue-cp;
  color: #fff !important;
  font-size: 13px;
  font-weight: 500;
  font-family: inherit;
  line-height: 20px;
  cursor: pointer;
  vertical-align: baseline;
  text-decoration: none;
  &:hover {
    background: $vd-cmt-blue-hover-cp;
    color: #fff !important;
  }
}

.mb-space__dyn-cmt-panel .vd-cmt-uni-inbox__bar {
  display: flex;
  align-items: center;
  padding: 6px 10px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  background: #f8f9fb;
}

.mb-space__dyn-cmt-panel .vd-cmt-uni-emoji {
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

.mb-space__dyn-cmt-panel .vd-cmt-uni-emoji:hover:not(:disabled) {
  color: $vd-cmt-blue-cp;
  background: rgba(0, 161, 214, 0.08);
}

.mb-space__dyn-cmt-panel .vd-cmt-uni-emoji:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.mb-space__dyn-cmt-panel .vd-emoji-ico {
  display: inline-block;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  vertical-align: middle;
  background-image: url("@/assets/icons-comment.2f36fc5.png");
  background-repeat: no-repeat;
  background-size: $icons-cmt-w $icons-cmt-h;
  background-position: -408px -24px;
}

.mb-space__dyn-cmt-panel .vd-cmt-mb-editor-row .vd-cmt-submit--mb {
  grid-column: 2;
  grid-row: 1;
  align-self: stretch;
  justify-self: stretch;
  width: auto;
  min-width: 0;
  max-width: none;
  min-height: 72px;
  padding: 0 8px;
  border: none;
  border-radius: 10px;
  background: $vd-cmt-blue-cp;
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  line-height: 1.35;
  white-space: normal;
  cursor: pointer;
  &:hover:not(:disabled):not(.is-guest) {
    background: $vd-cmt-blue-hover-cp;
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

.mb-space__dyn-cmt-panel .vd-cmt-submit-lines {
  display: inline-block;
  text-align: center;
  line-height: 1.3;
}

.mb-space__dyn-cmt-live {
  margin-top: 4px;
}

.mb-space__dyn-cmt-fold-open {
  display: block;
  margin: 8px 0 0 60px;
  padding: 0;
  border: none;
  background: transparent;
  font-size: 12px;
  color: #9499a0;
  text-align: left;
  cursor: pointer;
  line-height: 1.45;
  &:hover {
    color: $vd-cmt-blue-cp;
  }
}

.vd-cmt-root-stack {
  display: flex;
  flex-direction: column;
  width: 100%;
}
.mb-space__dyn-cmt-view-more-wrap {
  padding: 14px 8px 6px;
  text-align: center;
}

.mb-space__dyn-cmt-view-more {
  padding: 0;
  border: none;
  background: transparent;
  font-size: 13px;
  color: #9499a0;
  cursor: pointer;
  &:hover {
    color: #00aeec;
  }
}

.mb-space__dyn-cmt-foot {
  margin: 0;
  padding: 14px 8px 8px;
  text-align: center;
  font-size: 13px;
  color: #99a2aa;
}

.mb-space__dyn-cmt-closed-bar {
  margin: 10px 0 0;
  padding: 12px 14px;
  border-radius: 8px;
  background: #f6f7f8;
  text-align: center;
  font-size: 14px;
  line-height: 1.5;
  color: #9499a0;
}

.mb-space__dyn-vbox {
  display: flex;
  flex-direction: row;
  align-items: stretch;
  gap: 0;
  padding: 0;
  border: none;
  border-radius: 0;
  background: #fff;
  text-decoration: none;
  color: inherit;
  box-sizing: border-box;
  min-height: 0;
  transition: background 0.12s ease;
  &:hover {
    background: #fafafa;
  }
}

.mb-space__dyn-vbox--article {
  .mb-space__dyn-vdur,
  .mb-space__dyn-vplay,
  .mb-space__vthumb-later {
    display: none;
  }
}

.mb-space__dyn-vbox-l {
  position: relative;
  flex: 0 0 148px;
  width: 148px;
  min-height: 104px;
  align-self: stretch;
  overflow: hidden;
  background: #eee;
}

.mb-space__dyn-vcover {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.mb-space__dyn-vdur {
  position: absolute;
  right: 6px;
  bottom: 5px;
  z-index: 1;
  padding: 0;
  border-radius: 0;
  background: transparent;
  color: #fff;
  font-size: 12px;
  line-height: 1.3;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.9);
}

.mb-space__dyn-vplay {
  position: absolute;
  left: 50%;
  top: 50%;
  z-index: 1;
  width: 40px;
  height: 40px;
  margin: -20px 0 0 -20px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.45);
  &::after {
    content: "";
    position: absolute;
    left: 15px;
    top: 12px;
    border-style: solid;
    border-width: 8px 0 8px 12px;
    border-color: transparent transparent transparent #fff;
  }
}

.mb-space__dyn-vbox-r {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  gap: 6px;
  padding: 12px 14px 12px 12px;
  box-sizing: border-box;
}

.mb-space__dyn-vbox-title {
  font-size: 15px;
  font-weight: 700;
  color: #18191c;
  line-height: 1.35;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.mb-space__dyn-vbox-desc {
  font-size: 12px;
  color: #9499a0;
  word-break: break-word;
  line-height: 1.45;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.mb-space__dyn-vstats {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 14px 18px;
  margin-top: 4px;
  font-size: 12px;
  color: #99a2aa;
}

.mb-space__dyn-vstat {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.mb-space__dyn-vstat-ico {
  width: 18px;
  height: 18px;
  object-fit: contain;
  flex-shrink: 0;
  display: block;
  /* 响应式 */
  filter: brightness(0) invert(0.58);
  opacity: 0.92;
}

.mb-space__dyn-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px solid #f1f2f4;
  font-size: 13px;
  color: #6d7380;
}

.mb-space__dyn-foot-i {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  flex: 1;
  justify-content: center;
  &:first-child {
    justify-content: flex-start;
  }
  &:last-child {
    justify-content: flex-end;
  }
}

.mb-space__dyn-ico {
  display: inline-block;
  width: 16px;
  height: 16px;
  opacity: 0.72;
  vertical-align: middle;
}

.mb-space__dyn-ico--fwd {
  border: 1px solid currentColor;
  border-radius: 50%;
  border-right-color: transparent;
  transform: rotate(-45deg);
}

.mb-space__dyn-ico--cmt {
  border: 1px solid currentColor;
  border-radius: 3px;
  position: relative;
  &::after {
    content: "";
    position: absolute;
    left: 3px;
    bottom: -3px;
    width: 5px;
    height: 5px;
    border-left: 1px solid currentColor;
    border-bottom: 1px solid currentColor;
    transform: skewX(-12deg);
  }
}

.mb-space__dyn-ico--like {
  border: 1px solid currentColor;
  border-radius: 2px 2px 0 0;
  position: relative;
  &::after {
    content: "";
    position: absolute;
    left: 2px;
    bottom: -3px;
    width: 10px;
    height: 4px;
    border: 1px solid currentColor;
    border-top: none;
    border-radius: 0 0 3px 3px;
  }
}

.mb-space__empty-img {
  margin: 0;
  padding: 0;
  background: none;
  border: none;
  display: flex;
  justify-content: center;
  align-items: center;
  img {
    display: block;
    max-width: 100%;
    height: auto;
    margin: 0;
    padding: 0;
    border: none;
    background: none;
  }
}

.mb-space__empty-img--notice {
  padding: 4px 0 8px;
  img {
    max-height: 148px;
    width: auto;
    max-width: 100%;
  }
}

.mb-space__settings-panel {
  padding: 8px 4px 28px;
  text-align: left;
  box-sizing: border-box;
  width: 100%;
}

.mb-space__dyn-search-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 16px 32px;
  text-align: center;
}

.mb-space__dyn-search-empty-t {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: #61666d;
  em {
    font-style: normal;
    color: #18191c;
    font-weight: 500;
  }
}

.mb-space__dyn-search-empty-btn {
  padding: 6px 16px;
  border: 1px solid #e3e5e7;
  border-radius: 6px;
  background: #fff;
  color: #61666d;
  font-size: 13px;
  cursor: pointer;
  transition:
    border-color 0.15s ease,
    color 0.15s ease;
  &:hover {
    border-color: #00aeec;
    color: #00aeec;
  }
}

.mb-space__dyn-end {
  margin: 8px 0 0;
  padding: 20px 8px 8px;
  text-align: center;
  font-size: 13px;
  color: #99a2aa;
}

@media (max-width: 720px) {
  .mb-space__dyn-layout {
    flex-direction: column;
  }
  .mb-space__dyn-sidenav {
    flex-direction: row;
    flex: 0 0 auto;
    width: 100%;
    max-width: none;
    padding: 0 0 8px;
  }
  .mb-space__dyn-sub {
    flex: 1;
    padding: 10px 12px;
    font-size: 14px;
  }
}

.mb-space__hint {
  margin: 0;
  padding: 28px 12px;
  text-align: center;
  color: #888;
  font-size: 13px;
}

.mb-space__link {
  color: #00a1d6;
}

.mb-space__err {
  padding: 24px;
  text-align: center;
  color: #c00;
}

.mb-space__more-wrap {
  margin-top: 18px;
  text-align: center;
}

.mb-space__more {
  padding: 7px 24px;
  border-radius: 4px;
  border: 1px solid #ccd0d7;
  background: #fff;
  cursor: pointer;
  font-size: 13px;
  &:hover:not(:disabled) {
    border-color: #00a1d6;
    color: #00a1d6;
  }
  &:disabled {
    opacity: 0.6;
    cursor: default;
  }
}

@media (max-width: 960px) {
  .mb-space__video-grid {
    grid-template-columns: repeat(auto-fill, minmax(168px, 1fr));
  }
}

@media (max-width: 640px) {
  .mb-space__contrib-outer {
    grid-template-columns: 1fr;
    grid-template-rows: auto auto auto;
    row-gap: 14px;
  }

  .mb-space__contrib-outer .mb-space__dyn-sidenav {
    grid-column: 1;
    grid-row: 1;
    flex-direction: row;
    flex-wrap: wrap;
    flex: none;
    width: 100%;
    padding: 4px 0 0;
  }

  .mb-space__contrib-outer .mb-space__dyn-sub--split {
    flex: 1 1 calc(50% - 5px);
    min-width: 0;
  }

  .mb-space__contrib-right-head,
  .mb-space__contrib-right-head--article {
    grid-column: 1;
    grid-row: 2;
  }

  .mb-space__contrib-feed,
  .mb-space__contrib-feed--article {
    grid-column: 1;
    grid-row: 3;
  }

  .mb-space__article-cover-wrap {
    width: 128px;
    height: 72px;
  }

  .mb-space__article-link {
    gap: 12px;
    padding: 12px 0;
  }
}

@media (max-width: 520px) {
  .mb-space__video-grid {
    grid-template-columns: repeat(auto-fill, minmax(148px, 1fr));
  }

  .mb-space__navbar {
    padding: 0 12px 0 42px;
  }

  .mb-space__split {
    padding: 12px 0 20px 42px;
    gap: 0;
  }

  .mb-space__main {
    padding: 0;
  }

  .mb-space__header-bar {
    padding: 0 12px 12px 42px;
  }

  .mb-space__header {
    height: 140px;
  }

  .mb-space__header-shade {
    height: 72px;
  }

  .mb-space__aside {
    padding: 12px 12px 16px 10px;
  }
}

@import "../../styles/mb-space-home.scss";
@import "../../styles/mb-space-privacy.scss";
@import "../../styles/mb-space-collect.scss";
@import "../../styles/vd-comment-list.scss";
</style>

<style lang="scss">
@import "../../styles/vd-cmb-del-msgbox.scss";
@import "./dynDeleteDialog.scss";
</style>
