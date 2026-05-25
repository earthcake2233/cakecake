<template>
  <div class="body-contain">
    <div class="all-contain">
      <search-filter
        v-if="showVideoFilters"
        :model-value="videoFilters"
        :view-mode="videoViewMode"
        @change="onVideoFiltersChange"
        @view-change="videoViewMode = $event"
      />
      <mb-user-search-list
        v-if="showUserBlock"
        :items="userItems"
        :num-results="userNumResults"
        :is-user-tab="isUserTab"
      />
      <mb-search-empty
        v-else-if="userTabEmpty"
        mode="empty-user"
      />
      <mb-article-search-list
        v-else-if="isArticleTab && allResult.result"
        :items="allResult.result.article || []"
        :num-results="articleNumResults"
        :sort="articleSort"
        @sort-change="onArticleSortChange"
      />
      <mb-search-coming-soon v-else-if="isComingSoonTab" />
      <mb-search-empty
        v-else-if="searchEmptyMode"
        :mode="searchEmptyMode"
      />
      <div v-else-if="showResultWrap" class="result-wrap clearfix">
        <ul
          class="bangumi-list all-class"
          v-if="allResult.result && allResult.result.media_bangumi && allResult.result.media_bangumi.length"
        >
          <li
            class="synthetical"
            v-for="(item, index) in allResult.result.media_bangumi"
            :key="`allResult_result_media_bangumi_${index}`"
          >
            <div class="cardBangumibox" v-if="item.season">
              <div class="left-img">
                <a
                  :href="
                    'https://www.bilibili.com/bangumi/play/ss' +
                      item.season_id +
                      '/'
                  "
                  target="_blank"
                  :title="item.season.title"
                >
                  <div class="modal-box">
                    <div class="bangumi-tag vip" v-if="item.season.payment">
                      会员专享
                    </div>
                    <div class="lazy-img">
                      <img alt="" v-lazy="item.cover" />
                    </div>
                  </div>
                </a>
              </div>
              <div class="right-info">
                <div class="headline">
                  <span class="bangumi-label">番剧</span>
                  <a
                    :href="
                      'https://www.bilibili.com/bangumi/media/md' +
                        item.media_id +
                        '/'
                    "
                    target="_blank"
                    :title="item.season.title"
                    class="title"
                    v-html="item.title"
                  >
                  </a>
                </div>
                <div class="info-items">
                  <div class="top-info">
                    <div class="des lf">
                      风格：
                      <span class="type-s">{{ item.styles }}</span>
                    </div>
                    <div class="des">
                      地区：
                      <span class="type-s">{{ item.areas }}</span>
                    </div>
                    <div class="des lf">
                      开播时间：
                      <span class="type-s">{{ timeChange(item.pubtime) }}</span>
                    </div>
                    <div class="des cv">
                      声优：
                      <span :title="item.cv" class="type-s">
                        {{ item.cv }}
                      </span>
                    </div>
                  </div>
                  <div :title="item.desc" class="des info">
                    简介：{{ item.desc }}
                  </div>
                </div>
                <div class="nav">
                  <div class="main-container">
                    <div class="nav-container">
                      <div class="ep-card">
                        <div class="single-box">
                          <ul class="ep-box clearfix close" v-if="item.season">
                            <li
                              class="ep-sub"
                              v-for="(item, index) in item.season.episodes"
                              :key="`item_season_episodes_${index}`"
                            >
                              <a
                                :href="
                                  'https://www.bilibili.com/bangumi/play/ep' +
                                    item.ep_id +
                                    '/'
                                "
                                target="_blank"
                              >
                                <div
                                  :title="item.index + ' ' + item.index_title"
                                  class="ep-item"
                                >
                                  {{ item.index }}
                                </div>
                              </a>
                            </li>
                          </ul>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
                <div class="score">
                  <div class="score-num">
                    {{ item.media_score.score }}
                    <span class="fen">分</span>
                  </div>
                  <div class="user-count">
                    {{ userCount(item.media_score.user_count) + "人点评" }}
                  </div>
                </div>
              </div>
            </div>
          </li>
          <li
            v-if="allResult.top_tlist && allResult.top_tlist.media_bangumi > 0"
            class="card-more"
          >
            共找到{{ allResult.top_tlist.media_bangumi }}部相关番剧，
            <router-link :to="{ path: '/search/bangumi' }" class=""
              >点击查看</router-link
            >
            &gt;
          </li>
        </ul>
        <ul
          v-if="allResult.result"
          class="video-contain clearfix"
          :class="{ 'video-contain--list': isVideoListView }"
        >
          <li
            v-for="(item, index) in allResult.result.video"
            :key="`allResult_result_video_${index}`"
            :class="['video', isVideoListView ? 'list' : 'matrix']"
          >
            <div class="img video-thumb-hover search-video-thumb">
              <router-link
                :to="videoPlayRoute(item)"
                class="search-video-thumb__link"
              >
                <div class="lazy-img">
                  <img alt="" v-lazy="item.pic" />
                </div>
                <span class="so-imgTag_rb">
                  {{ item.duration }}
                </span>
              </router-link>
              <WatchLaterBtn
                :video-id="item.aid"
                :in-watch-later="!!item.in_watch_later"
              />
            </div>
            <div class="info">
              <div class="headline clearfix">
                <span
                  v-if="isVideoListView && videoTypeName(item)"
                  class="type zone-tag"
                  >{{ videoTypeName(item) }}</span
                >
                <router-link
                  :to="videoPlayRoute(item)"
                  title=""
                  class="title"
                  custom
                  v-slot="{ href, navigate }"
                >
                  <a :href="href" class="title" @click.prevent="navigate">
                    <span v-html="item.title"></span>
                  </a>
                </router-link>
              </div>
              <div v-if="isVideoListView && item.description" class="des">
                {{ plainDescription(item.description) }}
              </div>
              <div class="tags">
                <span title="观看" class="so-icon watch-num">
                  <i class="icon-playtime"></i>
                  {{ userCount(item.play) }}
                </span>
                <span
                  v-if="isVideoListView"
                  title="弹幕"
                  class="so-icon danmaku-num"
                >
                  <i class="icon-subtitle"></i>
                  {{ userCount(item.video_review) }}
                </span>
                <span title="上传时间" class="so-icon time">
                  <i class="icon-date"></i>
                  {{ timeChange(item.pubdate) }}
                </span>
                <span title="up主" class="so-icon uploader">
                  <i class="icon-uper"></i>
                  <router-link
                    v-if="item.mid"
                    :to="{
                      name: 'minibiliUserSpace',
                      params: { userId: item.mid }
                    }"
                    class="up-name"
                    >{{ item.author }}</router-link
                  >
                  <span v-else class="up-name">{{ item.author }}</span>
                </span>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script>
import { count2, timeChange } from "../../utils/utils";
import searchFilter from "../searchFilter/searchFilter";
import MbArticleSearchList from "./MbArticleSearchList.vue";
import MbUserSearchList from "./MbUserSearchList.vue";
import MbSearchComingSoon from "./MbSearchComingSoon.vue";
import MbSearchEmpty from "./MbSearchEmpty.vue";
import WatchLaterBtn from "../common/WatchLaterBtn.vue";
import { DEFAULT_SEARCH_VIDEO_VIEW } from "@/utils/searchFilters";
import { videoPlayRouteAid } from "@/utils/videoBvid";

const SEARCH_COMING_SOON_ROUTES = new Set([
  "searchBangumi",
  "searchPgc",
  "searchLive",
  "searchTopic",
  "photo"
]);

export default {
  props: {
    allResult: {
      type: [Object, Array],
      default: () => []
    },
    articleSort: {
      type: String,
      default: "default"
    },
    videoFilters: {
      type: Object,
      default: () => ({})
    }
  },
  emits: ["article-sort-change", "video-filters-change"],
  data() {
    return {
      videoViewMode: DEFAULT_SEARCH_VIDEO_VIEW
    };
  },
  computed: {
    isVideoListView() {
      return this.videoViewMode === "list";
    },
    isArticleTab() {
      return this.$route.name === "searchArticle";
    },
    isUserTab() {
      return this.$route.name === "upuser";
    },
    isAllTab() {
      return this.$route.name === "searchAll";
    },
    isComingSoonTab() {
      return SEARCH_COMING_SOON_ROUTES.has(this.$route.name);
    },
    showVideoFilters() {
      return !this.isArticleTab && !this.isUserTab && !this.isComingSoonTab;
    },
    showResultWrap() {
      return (
        !this.isArticleTab &&
        !this.isUserTab &&
        !this.isComingSoonTab &&
        !this.searchEmptyMode
      );
    },
    userTabEmpty() {
      return (
        this.isUserTab &&
        this.allResult &&
        this.allResult.result &&
        !this.userItems.length
      );
    },
    searchEmptyMode() {
      if (this.isUserTab || this.isArticleTab) {
        const st = this.allResult && this.allResult.search_status;
        if (st === "unavailable") {
          return "unavailable";
        }
        return "";
      }
      const st = this.allResult && this.allResult.search_status;
      if (st === "unavailable" || st === "empty") {
        return st;
      }
      if (!this.isAllTab && this.$route.name === "searchVideo") {
        const v =
          this.allResult &&
          this.allResult.result &&
          this.allResult.result.video;
        if (!v || !v.length) {
          return "empty";
        }
      }
      if (this.isAllTab && this.allResult && this.allResult.result) {
        const r = this.allResult.result;
        const has =
          (r.video && r.video.length) ||
          (r.bili_user && r.bili_user.length) ||
          (r.article && r.article.length);
        if (!has && st !== "ok") {
          return "empty";
        }
      }
      return "";
    },
    userItems() {
      return (
        (this.allResult &&
          this.allResult.result &&
          this.allResult.result.bili_user) ||
        []
      );
    },
    showUserBlock() {
      if (this.isUserTab) {
        return this.userItems.length > 0;
      }
      return (
        this.isAllTab &&
        this.allResult &&
        this.allResult.result &&
        this.userItems.length > 0
      );
    },
    userNumResults() {
      const n = this.allResult && this.allResult.numResults;
      if (typeof n === "number" && n >= 0 && this.isUserTab) {
        return n;
      }
      const t =
        this.allResult &&
        this.allResult.top_tlist &&
        this.allResult.top_tlist.bili_user;
      return typeof t === "number" ? t : 0;
    },
    articleNumResults() {
      const n = this.allResult && this.allResult.numResults;
      if (typeof n === "number" && n >= 0) {
        return n;
      }
      const t =
        this.allResult &&
        this.allResult.top_tlist &&
        this.allResult.top_tlist.article;
      return typeof t === "number" ? t : 0;
    }
  },
  components: {
    searchFilter,
    MbArticleSearchList,
    MbUserSearchList,
    MbSearchComingSoon,
    MbSearchEmpty,
    WatchLaterBtn
  },
  methods: {
    onArticleSortChange(sort) {
      this.$emit("article-sort-change", sort);
    },
    onVideoFiltersChange(val) {
      this.$emit("video-filters-change", val);
    },
    videoPlayRoute(item) {
      const aid = videoPlayRouteAid(item && item.aid);
      return { name: "video", params: { aid: aid || "BV0" } };
    },
    videoTypeName(item) {
      const t = item && (item.type_name || item.typeName);
      return t ? String(t).trim() : "";
    },
    plainDescription(html) {
      return String(html || "")
        .replace(/<[^>]+>/g, "")
        .trim();
    },
    timeChange(time) {
      return timeChange(time);
    },
    userCount(num) {
      return count2(num);
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style lang="scss">
@import "../../style/mixin";

//筛选
.filter-wrap {
  padding: 20px 0 10px;
  position: relative;
  border-bottom: 1px solid #e5e9ef;
  display: table;
  @include wh(100%, auto);
  .filter-type {
    display: block;
    overflow: hidden;
    margin-bottom: 10px;
    &.tids_1 {
      position: relative;
      & > .filter-item {
        margin-right: 9px;
      }
    }
  }
  .filter-item {
    float: left;
    padding-left: 8px;
    padding-right: 8px;
    @include borderRadius(4px);
    margin-right: 15px;
    a {
      @include sc(12px, $black);
      line-height: 20px;
    }
    &:not(.active):hover {
      a {
        color: $blue;
      }
    }
    &.active {
      background-color: $blue;
      a {
        color: $white;
      }
    }
  }
  .fold {
    position: absolute;
    @include borderRadius(4px);
    line-height: 24px;
    @include sc(12px, #6d757a);
    right: 0;
    display: inline-block;
    width: 74px;
    text-align: center;
    &.up {
      bottom: 18px;
    }
    &:hover {
      background-color: #e5e9ef;
      color: $blue;
    }
    &.down {
      left: 416px;
      top: 18px;
    }
  }
  &.filter-wrap--folded {
    padding-bottom: 16px;
    .filter-type.order {
      margin-bottom: 0;
    }
    .fold.down {
      top: 18px;
      bottom: auto;
    }
  }
}
.arrow-down,
.arrow-up {
  display: inline-block;
  @include wh(12px, 6px);
  vertical-align: middle;
  margin-top: -2px;
  margin-left: 3px;
}
.arrow-up {
  background-image: url(../../assets/search/sprite.png);
  background-position: -442px -403px;
}
.arrow-down {
  background-image: url(../../assets/search/sprite.png);
  background-position: -442px -439px;
}
.switch-wrap {
  position: absolute;
  top: 10px;
  right: 0;
  z-index: 999;
  .type {
    position: absolute;
    top: 0;
    cursor: pointer;
  }
  .aver {
    right: 26px;
  }
  .switch-item {
    @include wh(16px, 16px);
  }
  .imgleft {
    right: 0;
  }
}
.all-contain {
  .switch-wrap {
    top: 20px;
  }
}
.result-wrap {
  padding-bottom: 10px;
}
.all-class {
  border-bottom: 1px solid #e5e9ef;
}
.synthetical {
  margin-right: -10px;
  height: 120px;
  -webkit-transition: all 0.2s linear;
  -o-transition: all 0.2s linear;
  transition: all 0.2s linear;
  width: 878px;
  padding: 15px 0 15px 102px;
  position: relative;
  border-bottom: 1px solid #e5e9ef;
  .left-img {
    position: absolute;
    left: 0;
    top: 15px;
    @include wh(90px, 120px);
  }
  .headline {
    height: 24px;
  }
  .title {
    display: inline-block;
    height: 24px;
    max-width: 600px;
    overflow: hidden;
    -o-text-overflow: ellipsis;
    text-overflow: ellipsis;
    white-space: nowrap;
    vertical-align: middle;
    margin-right: 6px;
    font-weight: 700;
    line-height: 24px;
    @include sc(16px, $black);
    .keyword {
      font-weight: 700;
    }
  }
  .des {
    margin-top: 14px;
    margin-bottom: 7px;
    width: 762px;
    height: 36px;
    @include sc(12px, $grau);
    line-height: 18px;
  }
}
.all-class .movie-item,
.all-class .synthetical {
  border-bottom: none;
}
.bangumi-list {
  .synthetical {
    height: auto;
    .cardBangumibox {
      min-height: 168px;
      .modal-box {
        top: 0;
        position: absolute;
        @include wh(100%, 100%);
        &:hover {
          background: rgba(0, 0, 0, 0.5);
          background-image: url(../../assets/search/play.png);
          background-repeat: no-repeat;
          background-size: 38%;
          background-position: 40px 60px;
          @include borderRadius(4px);
        }
        .bangumi-tag {
          border-radius: 0 4px 0 4px;
          @include wh(56px, 21px);
          line-height: 21px;
          text-align: center;
          position: absolute;
          right: 0;
          background: #ffa726;
          color: $white;
          &.vip {
            background: $pink;
          }
        }
        .lazy-img {
          position: relative;
          z-index: -1;
          img {
            display: block;
            width: 100%;
            min-height: 100%;
            @include borderRadius(4px);
          }
        }
      }
    }
    .left-img {
      @include wh(126px, 168px);
    }
    .headline {
      margin-bottom: 15px;
      margin-top: 3px;
      font-size: medium;
      .bangumi-label {
        display: inline-block;
        height: 22px;
        padding: 0 10px;
        border: 1px solid #979797;
        text-align: center;
        line-height: 24px;
        margin-right: 12px;
        @include sc(12px, #979797);
        @include borderRadius(100px);
      }
    }
    .des {
      overflow: hidden;
      display: inline-block;
      @include wh(35%, auto);
      margin-top: 0;
      line-height: 17px;
      margin-bottom: 0;
      color: #b0b7bd;
      .type-s {
        color: $black;
      }
      &.info {
        display: block;
        @include wh(100%, 34px);
        -o-text-overflow: ellipsis;
        text-overflow: ellipsis;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
      }
      &.cv {
        height: 17px;
        line-height: 17px;
        -o-text-overflow: ellipsis;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }
    .score {
      position: absolute;
      right: 0;
      top: 20px;
      @include wh(100px, auto);
      .score-num {
        @include sc(20px, #ffa726);
        position: absolute;
        text-align: right;
        top: 8px;
        right: 0;
        .fen {
          font-size: 14px;
        }
      }
      .user-count {
        position: absolute;
        top: 34px;
        text-align: right;
        font-size: 14px;
        right: 0;
      }
    }
  }
  .right-info {
    margin-left: 40px;
    .info-items {
      height: 80px;
      .top-info {
        width: 90%;
      }
    }
  }
}
.keyword {
  color: $pink;
}
.main-container {
  position: relative;
}
.nav-container {
  .ep-card {
    overflow: hidden;
  }
  .single-box {
    .ep-box {
      margin-left: -7px;
      padding-top: 10px;
      z-index: -1;
    }
    .ep-sub {
      float: left;
      cursor: pointer;
    }
    .ep-item {
      margin-top: 1px;
      display: inline-block;
      @include wh(40px, 32px);
      @include borderRadius(4px);
      border: 1px solid #e5e9ef;
      text-align: center;
      line-height: 32px;
      padding: 0 5px;
      overflow: hidden;
      margin-left: 7px;
      color: #000;
      &:hover {
        background: $blue;
        color: $white;
      }
    }
  }
}
.card-more {
  padding-bottom: 15px;
  padding-right: 8px;
  line-height: 12px;
  color: $grau;
  text-align: right;
}
.video-contain {
  display: block;
  position: relative;
  overflow: hidden;
  &--list {
    margin-top: 4px;
  }
}
.video {
  &.list {
    display: flex;
    align-items: stretch;
    width: 100%;
    float: none;
    margin: 0;
    padding: 20px 0;
    border-bottom: 1px solid #e5e9ef;
    overflow: hidden;
    &:last-child {
      border-bottom: none;
    }
    .img,
    .search-video-thumb {
      flex-shrink: 0;
      float: none;
      width: 169px;
      height: 106px;
    }
    .search-video-thumb,
    .search-video-thumb__link {
      height: 106px;
    }
    .search-video-thumb {
      border-radius: 4px;
    }
    .lazy-img img {
      width: 169px;
      height: 106px;
      object-fit: cover;
      border-radius: 4px;
    }
    .info {
      flex: 1;
      display: flex;
      flex-direction: column;
      margin-left: 15px;
      padding: 0;
      min-height: 106px;
      float: none;
    }
    .headline {
      flex-shrink: 0;
      height: auto;
      max-height: none;
      margin-bottom: 6px;
      line-height: 24px;
      .zone-tag {
        display: inline-block;
        height: 20px;
        line-height: 20px;
        padding: 0 10px;
        margin-right: 8px;
        border: 1px solid #e5e9ef;
        border-radius: 12px;
        @include sc(12px, #999);
        vertical-align: middle;
      }
      .title {
        @include sc(16px, $black);
        line-height: 24px;
        font-weight: 700;
        display: inline;
      }
    }
    .des {
      flex: 1 1 auto;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
      margin-bottom: 0;
      max-height: 36px;
      @include sc(12px, #999);
      line-height: 18px;
    }
    .tags {
      flex-shrink: 0;
      margin-top: auto;
      padding-top: 6px;
      line-height: 16px;
      .so-icon {
        margin-right: 16px;
        margin-bottom: 0;
        float: none;
        display: inline-block;
        vertical-align: middle;
        height: 16px;
        line-height: 16px;
        padding-left: 18px;
      }
      .so-icon i {
        top: 2px;
      }
      .uploader .up-name {
        max-width: 200px;
        color: $grau;
        &:hover {
          color: $blue;
        }
      }
    }
  }
  &.matrix {
    @include wh(168px, 208px);
    border: 1px solid #e5e9ef;
    @include borderRadius(4px);
    float: left;
    margin-right: 32px;
    margin-top: 20px;
    &:nth-child(5n) {
      margin-right: 0;
    }
    .img {
      height: 100px;
      @include borderRadius(4px);
      position: relative;
      overflow: hidden;
    }
    .info {
      padding: 8px 10px 0;
      .so-icon {
        margin-right: 8px;
        margin-bottom: 12px;
        float: left;
        &.watch-num {
          width: 41px;
          overflow: inherit;
        }
      }
      .time {
        margin-right: 0;
        width: 67px;
      }
    }
    .headline {
      margin-bottom: 12px;
      height: 40px;
      overflow: hidden;
    }
    .type {
      &.avid {
        display: none;
      }
    }
    .hide {
      display: none;
    }
    .title {
      @include sc(12px, $black);
      line-height: 20px;
    }
  }
  .so-imgTag_rb {
    position: absolute;
    right: 0;
    bottom: 0;
    line-height: 18px;
    padding: 0 5px;
    color: $white;
    background-color: #333;
    background-color: rgba(0, 0, 0, 0.5);
    border-top-left-radius: 4px;
  }
  .search-video-thumb {
    position: relative;
    display: block;
    height: 100px;
    overflow: hidden;
    border-radius: 4px;
  }
  .search-video-thumb__link {
    display: block;
    height: 100px;
  }
  .img {
    &:hover {
      .so-imgTag_rb {
        display: none;
      }
    }
  }
  .so-icon {
    &.watch-num {
      white-space: nowrap;
    }
    & > a {
      color: $grau;
      cursor: pointer;
    }
    .up-name {
      display: inline-block;
      max-width: 132px;
      height: 16px;
      overflow: hidden;
      -o-text-overflow: ellipsis;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
}
.watch-later-trigger {
  display: none;
  @include wh(22px, 22px);
  position: absolute;
  right: 6px;
  bottom: 4px;
  cursor: pointer;
  background-image: url(../../assets/play.png);
}
.so-icon {
  display: inline-block;
  @include sc(12px, $grau);
  height: 12px;
  vertical-align: text-top;
  line-height: 12px;
  padding-left: 16px;
  position: relative;
  i {
    position: absolute;
    left: 0;
    top: 0;
    background: url(../../assets/search/sprite.png) no-repeat;
    @include wh(12px, 12px);
    &.icon-playtime {
      background-position: -485px -543px;
    }
    &.icon-subtitle {
      background-position: -442px -124px;
    }
    &.icon-date {
      background-position: -442px -165px;
    }
    &.icon-uper {
      background-position: -442px -83px;
    }
  }
}
.video.list .so-icon i {
  @include wh(14px, 14px);
  &.icon-playtime {
    background-position: -485px -543px;
  }
  &.icon-subtitle {
    background-position: -442px -124px;
  }
  &.icon-date {
    background-position: -442px -165px;
  }
  &.icon-uper {
    background-position: -442px -83px;
  }
}
</style>
