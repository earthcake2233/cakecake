<template>
  <div class="contain">
    <div class="head-contain">
      <div class="search-wrap">
        <div class="">
          <div class="logo-input">
            <router-link
              :to="{ name: 'home' }"
              class="search-logo"
              title="返回主站"
            >
              <span class="search-logo-mark">
                <img
                  class="search-logo-img"
                  src="@/assets/cakelogo.png"
                  alt="cakecake"
                />
              </span>
              <span class="search-logo-split">
                <span class="search-logo-bar" aria-hidden="true">|</span>
                <span class="search-logo-text">搜索</span>
              </span>
            </router-link>
            <div class="search-block">
              <div class="input-wrap">
                <input
                  v-model="searchValue"
                  type="text"
                  @keyup.enter="getAllResult"
                  v-on:input="setSuggest"
                  @focus="setSuggestShow"
                  @blur="setSuggestHide"
                  id="search-keyword"
                  maxlength="100"
                  autocomplete="off"
                />
                <div
                  class="suggest-wrap"
                  v-if="suggestShow && suggestTags.length"
                >
                  <div class="hotword-wrap" style="display: none;">
                    <ul class="horizontal">
                      <li>
                        <a href="javascript:;" class="hz-text">
                          世界杯
                        </a>
                      </li>
                      <li>
                        <a href="javascript:;" class="hz-text">
                          工作细胞
                        </a>
                      </li>
                      <li>
                        <a href="javascript:;" class="hz-text">
                          我不是药神
                        </a>
                      </li>
                      <li>
                        <a href="javascript:;" class="hz-text">
                          2018洲际赛
                        </a>
                      </li>
                      <li>
                        <a href="javascript:;" class="hz-text">
                          人生一串
                        </a>
                      </li>
                      <li>
                        <a href="javascript:;" class="hz-text">
                          第五人格
                        </a>
                      </li>
                      <li>
                        <a href="javascript:;" class="hz-text">
                          我的英雄学院
                        </a>
                      </li>
                      <li>
                        <a href="javascript:;" class="hz-text">
                          防弹少年团
                        </a>
                      </li>
                      <li>
                        <a href="javascript:;" class="hz-text">
                          创造101
                        </a>
                      </li>
                      <li>
                        <a href="javascript:;" class="hz-text">
                          逆水寒
                        </a>
                      </li>
                    </ul>
                  </div>
                  <ul class="keyword-wrap" v-if="suggestShow && suggestTags.length">
                    <li
                      v-for="(item, index) in suggestTags"
                      :key="`search_suggest_${index}`"
                    >
                      <a
                        href="javascript:;"
                        class="vt-text keyword"
                        @mousedown.prevent="pickSuggest(item)"
                        v-html="item.name"
                      ></a>
                    </li>
                  </ul>
                  <ul class="history-wrap" style="display: none;">
                    <li class="title"><span>搜索历史</span></li>
                    <li class="history-li">
                      <a class="vt-text history">我的英雄学院</a
                      ><i class="clear"></i>
                    </li>
                    <li class="history-li">
                      <a class="vt-text history">寄生兽</a><i class="clear"></i>
                    </li>
                    <li class="history-li">
                      <a class="vt-text history">K</a><i class="clear"></i>
                    </li>
                    <li class="history-li">
                      <a class="vt-text history">寄生</a><i class="clear"></i>
                    </li>
                    <li class="history-li">
                      <a class="vt-text history">RNG</a><i class="clear"></i>
                    </li>
                    <li class="clearall"><a>清空搜索历史</a></li>
                  </ul>
                </div>
              </div>
              <div
                class="search-button"
                role="button"
                tabindex="0"
                @click="getAllResult"
                @keyup.enter="getAllResult"
              >
                <i class="icon-search-white"></i>
                <span class="search-text">搜索</span>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="nav-wrap">
        <ul class="wrap clearfix">
          <router-link
            v-for="(item, index) in searchMenu"
            :key="item.type"
            v-slot="{ navigate, isActive }"
            custom
            :to="{
              path: item.path,
              query: {
                keyword: $route.query.keyword
              }
            }"
          >
            <li
              class="sub"
              :class="{ active: isActive }"
              @mouseenter="hoverBarLeft(index)"
              @mouseleave="hoverBarLeave()"
              @click="
                e => {
                  setHoverIndex(index);
                  navigate(e);
                }
              "
            >
              {{ item.title }}
              <span class="num" v-show="index > 0">{{
                topNumChange(item.resultNum)
              }}</span>
            </li>
          </router-link>
        </ul>
        <div class="hover-bar" :style="{ left: hoverBar + 'px' }"></div>
      </div>
    </div>
    <router-view v-slot="{ Component }">
      <component
        :is="Component"
        :allResult="allResult"
        :article-sort="articleSort"
        :video-filters="videoFilters"
        @article-sort-change="onArticleSortChange"
        @video-filters-change="onVideoFiltersChange"
      />
    </router-view>
  </div>
</template>

<script>
import { mapState, mapMutations, mapActions } from "vuex";
import { addSearchHistory } from "@/utils/searchHistory";
import { DEFAULT_VIDEO_FILTERS } from "@/utils/searchFilters";

export default {
  created() {
    const keyword = this.$route.query.keyword;
    this.setSearchValue(keyword);
    this.fetchForRoute(keyword);
  },
  watch: {
    "$route.fullPath"(to, from) {
      const keyword = this.$route.query.keyword;
      const prevKw = from && from.query ? from.query.keyword : undefined;
      const kw = Array.isArray(keyword) ? keyword[0] : keyword;
      const prev = Array.isArray(prevKw) ? prevKw[0] : prevKw;
      if (String(kw || "") !== String(prev || "")) {
        this.videoFilters = { ...DEFAULT_VIDEO_FILTERS };
        this.articleSort = "default";
      }
      this.setSearchValue(keyword);
      this.fetchForRoute(keyword);
    }
  },
  computed: {
    searchValue: {
      get() {
        return this.$store.state.search.searchWord;
      },
      set(value) {
        this.setSearchValue(value);
      }
    },
    ...mapState("search", [
      //命名空间获取state
      "searchWord",
      "searchMenu",
      "hoverBar",
      "hoverIndex",
      "allResult",
      "suggest"
    ]),
    suggestTags() {
      const s = this.suggest;
      if (s && Array.isArray(s.tag)) {
        return s.tag;
      }
      if (Array.isArray(s)) {
        return s;
      }
      return [];
    }
  },
  components: {},
  data() {
    return {
      suggestShow: false,
      articleSort: "default",
      videoFilters: { ...DEFAULT_VIDEO_FILTERS }
    };
  },
  methods: {
    fetchForRoute(keyword) {
      if (keyword == null || keyword === "") {
        return;
      }
      const type = this.searchRouteType();
      this.setAllResult({
        highlight: 1,
        keyword,
        type,
        sort: type === "article" ? this.articleSort : "",
        videoFilters: this.videoFilters
      });
    },
    onVideoFiltersChange(filters) {
      this.videoFilters = { ...DEFAULT_VIDEO_FILTERS, ...filters };
      const keyword = this.routeKeyword();
      if (keyword) {
        this.fetchForRoute(keyword);
      }
    },
    searchRouteType() {
      const name = this.$route.name;
      if (name === "searchArticle") {
        return "article";
      }
      if (name === "searchVideo") {
        return "video";
      }
      if (name === "upuser") {
        return "user";
      }
      return "all";
    },
    onArticleSortChange(sort) {
      this.articleSort = sort;
      const keyword = this.$route.query.keyword;
      if (keyword) {
        this.setAllResult({
          highlight: 1,
          keyword,
          type: "article",
          sort
        });
      }
    },
    ...mapMutations("search", {
      setSearchValue: "SET_SEARCH_VALUE",
      setHoverBar: "SET_HOVER_BAR",
      setHoverIndex: "SET_HOVER_INDEX"
    }),
    ...mapActions("search", ["setAllResult", "setSuggest", "setSeason"]),
    setSuggestShow() {
      const hasInput = String(this.searchValue || "").length > 0;
      this.suggestShow = hasInput;
      if (hasInput) {
        this.setSuggest();
      }
    },
    setSuggestHide() {
      this.suggestShow = false;
    },
    hoverBarLeft(index) {
      this.setHoverBar(index * 114);
    },
    hoverBarLeave() {
      this.setHoverBar(this.hoverIndex * 114);
    },
    topNumChange(num) {
      if (num > 99) {
        return "99+";
      } else {
        return num;
      }
    },
    routeKeyword() {
      const q = this.$route.query.keyword;
      const raw = Array.isArray(q) ? q[0] : q;
      return raw != null ? String(raw).trim() : "";
    },
    getAllResult() {
      const kw = String(this.searchValue || "").trim();
      if (!kw) {
        return;
      }
      this.setSearchValue(kw);
      addSearchHistory(kw);
      if (this.routeKeyword() !== kw) {
        this.$router.push({
          path: this.$route.path,
          query: { ...this.$route.query, keyword: kw }
        });
        return;
      }
      this.fetchForRoute(kw);
    },
    pickSuggest(item) {
      const kw = String((item && item.value) || "").trim();
      if (!kw) {
        return;
      }
      this.setSearchValue(kw);
      this.suggestShow = false;
      addSearchHistory(kw);
      const path = this.$route.path.startsWith("/search")
        ? this.$route.path
        : "/search/all";
      if (this.routeKeyword() === kw && path === this.$route.path) {
        this.fetchForRoute(kw);
        return;
      }
      this.$router.push({ path, query: { ...this.$route.query, keyword: kw } });
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style lang="scss">
@import "../../style/mixin";

a {
  outline: 0;
  color: $blue;
  text-decoration: none;
  cursor: pointer;
}
.contain {
  width: 980px;
  margin: 0 auto;
}
.head-contain {
  padding-top: 40px;
}
.search-wrap {
  @include wh(587px, 44px);
  margin: 0 auto;
  position: relative;
  .logo-input {
    display: flex;
    align-items: center;
    width: 587px;
    height: 44px;
  }
  /* 138px 品牌区 + 19px 间距 + 430px 搜索框 = 587px */
  .search-logo {
    flex: 0 0 138px;
    width: 138px;
    height: 38px;
    display: flex;
    align-items: center;
    text-decoration: none;
    flex-shrink: 0;
    overflow: hidden;
  }
  .search-logo-mark {
    flex: 1 1 auto;
    min-width: 0;
    height: 38px;
    display: flex;
    align-items: center;
    overflow: hidden;
  }
  .search-logo-img {
    display: block;
    height: 38px;
    width: auto;
    max-width: 100%;
    object-fit: contain;
    object-position: left center;
  }
  .search-logo-split {
    flex: 0 0 auto;
    display: inline-flex;
    align-items: center;
    height: 38px;
    margin-left: 3px;
    white-space: nowrap;
  }
  .search-logo-bar {
    margin: 0 6px 0 3px;
    color: $blue;
    font-size: 16px;
    line-height: 38px;
    font-weight: 300;
    user-select: none;
  }
  /* 像素感标题字：纯 CSS，不依赖雪碧图 */
  .search-logo-text {
    color: $blue;
    font-family: SimHei, "Microsoft YaHei", "PingFang SC", sans-serif;
    font-size: 17px;
    font-weight: 800;
    line-height: 38px;
    letter-spacing: 0.5px;
    -webkit-font-smoothing: none;
    -moz-osx-font-smoothing: grayscale;
    text-rendering: geometricPrecision;
  }
  .search-block {
    flex: 0 0 430px;
    width: 430px;
    margin-left: 19px;
    display: flex;
    align-items: center;
    .input-wrap {
      position: relative;
      background: $white;
      @include borderRadius(4px);
      flex: 0 0 330px;
      width: 330px;
      margin-right: 10px;
      input {
        @include wh(296px, 18px);
        -webkit-box-shadow: none;
        box-shadow: none;
        padding: 11px 15px;
        background: transparent;
        border: 2px solid $border_color;
        @include borderRadius(4px);
        color: $black;
      }
    }
    .search-button {
      cursor: pointer;
      flex: 0 0 90px;
      background: $blue;
      @include sc(16px, $white);
      letter-spacing: 2px;
      line-height: 42px;
      text-align: center;
      width: 90px;
      @include borderRadius(4px);
      &:hover {
        background: #00b5e5;
      }
      .icon-search-white {
        background-image: url(../../assets/search/sprite.png);
        background-position: -148px -327px;
        @include wh(18px, 19px);
        vertical-align: middle;
        margin-top: -2px;
        display: inline-block;
      }
    }
    .search-text {
      margin-left: 7px;
    }
  }
}
.suggest-wrap {
  border: 1px solid #e5e9ef;
  position: absolute;
  width: 327px;
  @include borderRadius(4px);
  text-align: center;
  padding: 10px 0;
  color: $black;
  background: $white;
  z-index: 100;
  overflow: hidden;
  margin-top: 5px;
  -webkit-box-shadow: rgba(0, 0, 0, 0.16) 0 2px 4px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.16);
  .horizontal {
    padding: 0 20px;
    max-height: 84px;
    overflow: hidden;
    .hz-text {
      @include borderRadius(4px);
      margin-right: 15px;
      margin-bottom: 10px;
      border: 1px solid #e5e9ef;
      @include sc(14px, $black);
      height: 18px;
      padding: 7px 8px;
      float: left;
      text-align: center;
      &:hover {
        border-color: $blue;
        color: $blue;
      }
    }
  }
  .history-wrap {
    margin-top: 20px;
    position: relative;
    padding-bottom: 20px;
    .history-li {
      position: relative;
      margin: 0;
      .clear {
        position: absolute;
        right: 20px;
        top: 10px;
        background-image: url(../../assets/search/sprite.png);
        background-position: -485px -41px;
        @include wh(12px, 12px);
        cursor: pointer;
      }
    }
    .clearall {
      position: absolute;
      bottom: 0;
      right: 20px;
      @include sc(12px, $blue);
    }
  }
  .title {
    border-top: 1px solid #e5e9ef;
    height: 10px;
    line-height: 10px;
    margin: 0 20px;
    span {
      display: inline-block;
      @include sc(12px, $grau);
      padding: 0 10px;
      text-align: center;
      background: $white;
      position: relative;
      top: -6px;
    }
  }
  .vt-text {
    height: 32px;
    display: block;
    line-height: 32px;
    @include sc(14px, $black);
    position: relative;
    text-align: left;
    white-space: nowrap;
    -o-text-overflow: ellipsis;
    text-overflow: ellipsis;
    overflow: hidden;
    cursor: pointer;
    padding: 0 20px;
    margin: 0 0 4px;
    &:hover {
      background-color: #e5e9ef;
    }
  }
  .keyword-wrap {
    .keyword {
      padding: 0 20px;
      color: $black;
      .suggest_high_light {
        color: $pink;
      }
    }
  }
}
.nav-wrap {
  clear: both;
  width: 100%;
  border-bottom: 1px solid $border_color;
  height: 54px;
  padding: 0 0 1px;
  margin: 18px auto 0;
  position: relative;
  .wrap {
    & > .sub {
      float: left;
      line-height: 54px;
      text-align: center;
      cursor: pointer;
      width: 39px;
      padding: 0;
      font-size: 16px;
      padding-right: 75px;
      &:last-child {
        padding-right: 0;
      }
      & > span {
        position: absolute;
        margin-left: 6px;
        @include sc(12px, #6d757a);
      }
    }
  }
  .hover-bar {
    position: absolute;
    @include wh(39px, 2px);
    background-color: $blue;
    bottom: -1px;
    -webkit-transition: left 0.2s;
    -o-transition: left 0.2s;
    transition: left 0.2s;
  }
}
.nav-wrap .wrap,
.nav-wrap .wrap > .sub {
  height: 100%;
  position: relative;
}
.nav-wrap .wrap > .sub.active,
.nav-wrap .wrap > .sub:hover {
  color: $blue;
}
</style>
