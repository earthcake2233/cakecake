<template>
  <div class="app-header">
    <nav-menu
      :leftNav="leftNav"
      :headBanner="headBanner"
      :menuShow="showHomeChrome"
    ></nav-menu>
    <template v-if="showHomeChrome">
      <div
        v-if="showGlobalHeadBanner"
        class="head-banner"
        :style="{ 'background-image': 'url(' + headBanner.pic + ')' }"
      >
        <div class="bili-wrapper head-content">
          <div class="search">
            <div class="searchform">
              <input
                v-model="searchValue"
                type="text"
                :placeholder="searchPlaceholder"
                @keyup.enter="searchALL()"
                @input="onSearchInput"
                @focus="onSearchFocus"
                @blur="onSearchBlur"
                class="search-keyword"
              />
              <button
                type="submit"
                class="search-submit"
                @click="searchALL()"
              ></button>
            </div>
            <ul v-if="suggestShow" class="bilibili-suggest">
              <li class="kw">
                <div class="b-line">
                  <p><span>关键词</span></p>
                </div>
              </li>
              <li
                class="suggest-item"
                v-for="(item, index) in suggestTagList"
                :key="`suggest_item_${index}`"
              >
                <a
                  href="javascript:;"
                  @mousedown.prevent="searchByHistory(item.value)"
                  v-html="item.name"
                ></a>
              </li>
            </ul>
            <div
              v-else-if="historyPanelShow"
              class="bilibili-suggest search-history-panel"
            >
              <div class="search-history-head">
                <div class="b-line">
                  <p><span>历史搜索</span></p>
                </div>
              </div>
              <ul class="search-history-list">
                <li
                  v-for="(kw, index) in searchHistory"
                  :key="`search_hist_${index}_${kw}`"
                  class="search-history-item"
                >
                  <a
                    href="javascript:;"
                    class="search-history-link"
                    @mousedown.prevent="searchByHistory(kw)"
                    >{{ kw }}</a
                  >
                  <button
                    type="button"
                    class="search-history-del"
                    aria-label="删除"
                    @mousedown.prevent="removeHistoryItem(index)"
                  >
                    ×
                  </button>
                </li>
              </ul>
            </div>
            <router-link class="link-ranking" :to="{ name: 'Ranking' }">
              <span>排行榜</span>
            </router-link>
          </div>
          <a
            class="head-logo"
            :style="{ background: 'url(' + headBanner.litpic + ')' }"
          ></a>
        </div>
        <a href="" target="_blank" class="banner-link"></a>
      </div>
      <div v-if="showGlobalPrimaryMenu" class="bili-wrapper">
        <div class="primary-menu">
          <ul class="nav-menu">
            <li
              v-for="(item, index) in menuLeft"
              :key="`menuLeft_item_${index}`"
              :class="item.class"
            >
              <a :href="item.href">
                <div class="num-wrap" v-if="item.num">
                  <!-- eslint-disable-next-line -->
                  <span>{{ item.num < 1000 ? item.num : 999 + "+" }}</span>
                </div>
                <div class="nav-name">
                  {{ item.name }}
                </div>
              </a>
              <ul class="sub-nav" v-if="item.items">
                <li
                  v-for="(navitem, ind) in item.items"
                  :key="`sub_navs_item_${ind}`"
                >
                  <a :href="navitem.href">
                    <span>{{ navitem.name }}</span>
                  </a>
                </li>
              </ul>
            </li>
            <li
              class="side-nav nav-square"
              v-for="(item, index) in menuRight"
              :key="`menuRight_item_${index}`"
            >
              <a :href="item.href" class="side-link" :class="item.class">
                <i :class="item.icon"></i>
                <span>{{ item.name }}</span>
              </a>
              <div
                class="sub-nav"
                v-if="item.fieldClass != ''"
                :class="item.fieldClass"
              >
                <ul>
                  <li
                    v-for="(itemnav, index) in item.fields"
                    :key="`item_fields_${index}`"
                  >
                    <a :href="itemnav.href">
                      <i
                        class="icon-prim"
                        :class="itemnav.icon"
                        v-if="itemnav.icon"
                      ></i>
                      <span>{{ itemnav.name }}</span>
                    </a>
                  </li>
                </ul>
                <div :class="item.fieldImgClass">
                  <a
                    v-for="(itemnavImg, index) in item.fieldImg"
                    :key="`fieldImg_item_${index}`"
                    :href="itemnavImg.href"
                    target="_blank"
                    :title="itemnavImg.title"
                    :class="itemnavImg.imgclass"
                  >
                    <img :alt="itemnavImg.title" :src="itemnavImg.src" />
                  </a>
                </div>
              </div>
            </li>
          </ul>
          <div class="gif-menu nav-gif" v-if="menuIcon.links">
            <a
              :href="menuIcon.links[0]"
              target="_blank"
              :title="menuIcon.title"
              class="random-p"
            >
              <img :src="menuIcon.icon" alt="" />
            </a>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script>
import NavMenu from "../../components/navMenu/navMenu";
import { mapState, mapMutations, mapActions } from "vuex";
import {
  shouldShowMinibiliCompactHeader,
  shouldShowHomeHeaderChrome
} from "@/utils/minibiliRoutes";
import {
  addSearchHistory,
  loadSearchHistoryAsync,
  removeSearchHistoryAt
} from "@/utils/searchHistory";

export default {
  async created() {
    this.setHeadBanner({
      pf: 0,
      id: 142
    });
    this.setSearchDefaultWords();
    this.setMenuIcon();
    this.searchHistory = await loadSearchHistoryAsync();
  },
  beforeUnmount() {
    this.clearHideSearchPanelTimer();
    this.stopHotPlaceholderRotate();
  },
  components: {
    NavMenu
  },
  data() {
    return {
      suggestShow: false,
      historyPanelVisible: false,
      searchHistory: [],
      _hideSearchPanelTimer: null,
      hotPlaceholderIndex: 0,
      _hotPlaceholderTimer: null
    };
  },
  computed: {
    // 使用对象展开运算符将此对象混入到外部对象中
    ...mapState("header", [
      //命名空间获取state
      "leftNav",
      "headBanner", //登录状态获取
      "searchWord",
      "suggest",
      "menuLeft",
      "menuRight",
      "menuIcon"
    ]),
    searchValue: {
      get() {
        return this.$store.state.header.searchValue;
      },
      set(value) {
        this.setSearchValue(value);
      }
    },
    /** 个人中心 / 消息 / 个人空间等：仅顶栏 nav-menu，样式同消息中心 */
    isCompactHeaderRoute() {
      return shouldShowMinibiliCompactHeader(this.$route);
    },
    /** 首页头图与分区导航；搜索页与个人中心等为 false */
    showHomeChrome() {
      return shouldShowHomeHeaderChrome(this.$route);
    },
    showGlobalHeadBanner() {
      return !this.isCompactHeaderRoute;
    },
    showGlobalPrimaryMenu() {
      return !this.isCompactHeaderRoute;
    },
    historyPanelShow() {
      return (
        this.historyPanelVisible &&
        !this.suggestShow &&
        this.searchHistory.length > 0
      );
    },
    hotPlaceholderList() {
      const list = this.searchWord && this.searchWord.hot_list;
      if (Array.isArray(list) && list.length) {
        return list.map(s => String(s).trim()).filter(Boolean);
      }
      const one = String(
        (this.searchWord && this.searchWord.show_name) || ""
      ).trim();
      return one ? [one] : [];
    },
    searchPlaceholder() {
      const list = this.hotPlaceholderList;
      if (!list.length) {
        return "搜索";
      }
      return list[this.hotPlaceholderIndex % list.length];
    },
    defaultSearchKeyword() {
      return (
        this.searchPlaceholder ||
        String((this.searchWord && this.searchWord.word) || "").trim()
      );
    },
    suggestTagList() {
      const tags = this.suggest && this.suggest.tag;
      return Array.isArray(tags) ? tags : [];
    }
  },
  watch: {
    hotPlaceholderList: {
      immediate: true,
      handler() {
        this.hotPlaceholderIndex = 0;
        this.stopHotPlaceholderRotate();
        this.startHotPlaceholderRotate();
      }
    }
  },
  methods: {
    ...mapMutations("header", {
      setSearchValue: "SET_SEARCH_WORD"
    }),
    ...mapActions("header", [
      "setHeadBanner", // 将 `this.setHeadBanner(amount)` 映射为 `this.$store.dispatch('headBanner', amount)`
      "setSearchDefaultWords",
      "setSuggest",
      "setMenuIcon"
    ]),
    clearHideSearchPanelTimer() {
      if (this._hideSearchPanelTimer) {
        clearTimeout(this._hideSearchPanelTimer);
        this._hideSearchPanelTimer = null;
      }
    },
    startHotPlaceholderRotate() {
      const list = this.hotPlaceholderList;
      if (list.length <= 1) {
        return;
      }
      this._hotPlaceholderTimer = setInterval(() => {
        this.hotPlaceholderIndex =
          (this.hotPlaceholderIndex + 1) % list.length;
      }, 3500);
    },
    stopHotPlaceholderRotate() {
      if (this._hotPlaceholderTimer) {
        clearInterval(this._hotPlaceholderTimer);
        this._hotPlaceholderTimer = null;
      }
    },
    syncSearchPanels() {
      const hasInput = String(this.searchValue || "").length > 0;
      if (hasInput) {
        this.suggestShow = true;
        this.historyPanelVisible = false;
      } else {
        this.suggestShow = false;
        this.historyPanelVisible = this.searchHistory.length > 0;
      }
    },
    onSearchFocus() {
      this.clearHideSearchPanelTimer();
      void loadSearchHistoryAsync().then(list => {
        this.searchHistory = list;
        this.syncSearchPanels();
      });
    },
    onSearchBlur() {
      this.clearHideSearchPanelTimer();
      this._hideSearchPanelTimer = setTimeout(() => {
        this.suggestShow = false;
        this.historyPanelVisible = false;
        this._hideSearchPanelTimer = null;
      }, 180);
    },
    onSearchInput() {
      this.setSuggest();
      this.syncSearchPanels();
    },
    removeHistoryItem(index) {
      this.searchHistory = removeSearchHistoryAt(index);
      if (!this.searchHistory.length) {
        this.historyPanelVisible = false;
      }
    },
    searchByHistory(keyword) {
      const kw = String(keyword || "").trim();
      if (!kw) {
        return;
      }
      this.setSearchValue(kw);
      this.searchHistory = addSearchHistory(kw);
      this.suggestShow = false;
      this.historyPanelVisible = false;
      this.$router.push({ path: "/search/all", query: { keyword: kw } });
    },
    searchALL() {
      const raw = String(this.searchValue || "").trim();
      const kw = raw || String(this.defaultSearchKeyword || "").trim();
      if (kw) {
        this.searchHistory = addSearchHistory(kw);
        if (!raw) {
          this.setSearchValue(kw);
        }
      }
      this.suggestShow = false;
      this.historyPanelVisible = false;
      this.$router.push({ path: "/search/all", query: { keyword: kw } });
    },
    routerPath() {
      return this.$route.matched[0].path === "search";
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style></style>
