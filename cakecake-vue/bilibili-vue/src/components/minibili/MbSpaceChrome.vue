<template>
  <div class="mb-space-chrome">
    <header class="mb-space-chrome__header" aria-label="个人空间头图">
      <div
        class="mb-space-chrome__banner-bg"
        :style="{ backgroundImage: 'url(' + bannerUrl + ')' }"
        aria-hidden="true"
      />
      <div class="mb-space-chrome__header-shade" aria-hidden="true" />
      <MbSpacePerspective
        v-if="isPerspectivePreview"
        :model-value="perspectiveMode"
        @update:model-value="onPerspectiveUpdate"
      />
      <div class="mb-space-chrome__header-bar">
      <div class="mb-space-chrome__profile">
        <img
          class="mb-space-chrome__avatar"
          :src="avatarDisplay"
          width="80"
          height="80"
          alt=""
        />
        <div class="mb-space-chrome__profile-text">
          <div class="mb-space-chrome__name-row">
            <span class="mb-space-chrome__name">{{ displayName }}</span>
            <div class="mb-space-chrome__badges">
              <img
                class="mb-space-chrome__level-badge"
                :src="levelIconUrl(levelDisplay)"
                width="36"
                height="36"
                alt=""
                :title="'LV' + levelDisplay"
              />
              <span
                v-if="spaceGenderKey === 'male'"
                class="mb-space-chrome__gender--ico"
                role="img"
                aria-label="男"
              >
                <img
                  class="mb-space-chrome__gender-img"
                  :src="genderMaleIco"
                  width="18"
                  height="18"
                  alt=""
                />
              </span>
              <span
                v-else-if="spaceGenderKey === 'female'"
                class="mb-space-chrome__gender--ico"
                role="img"
                aria-label="女"
              >
                <img
                  class="mb-space-chrome__gender-img"
                  :src="genderFemaleIco"
                  width="18"
                  height="18"
                  alt=""
                />
              </span>
            </div>
          </div>
          <p class="mb-space-chrome__sign">{{ displaySign }}</p>
        </div>
      </div>
      <MbSpaceHeaderActions
        :user-id="userId"
        :is-owner="isSpaceOwner"
        :followed-by-me="followedByMe"
        :preview-only="isPerspectivePreview"
        @update:followed-by-me="$emit('update:followedByMe', $event)"
        @follower-count="$emit('follower-count', $event)"
        @login="openLoginModal"
      />
      </div>
    </header>

    <nav class="mb-space-chrome__navbar" aria-label="空间主导航">
      <div class="mb-space-chrome__dock-row">
        <div class="mb-space-chrome__tabs">
          <button
            v-for="tab in navTabs"
            :key="tab.key"
            type="button"
            class="mb-space-chrome__tab"
            :class="{ 'is-on': activeNav === tab.key }"
            @click="$emit('nav', tab.key)"
          >
            <img
              class="mb-space-chrome__tab-ico"
              :class="tab.iconClass"
              :src="tab.icon"
              alt=""
            />
            <span>{{ tab.label }}</span>
            <span v-if="tab.badge != null" class="mb-space-chrome__tab-badge">{{
              tab.badge
            }}</span>
          </button>
        </div>
        <div class="mb-space-chrome__dock-gap" aria-hidden="true" />
        <div class="mb-space-chrome__search">
          <input
            :value="searchKeyword"
            type="search"
            class="mb-space-chrome__search-input"
            placeholder="搜索视频、动态"
            autocomplete="off"
            @input="onSearchInput"
            @keydown.enter.prevent
          />
          <button
            type="button"
            class="mb-space-chrome__search-btn"
            aria-label="搜索"
          >
            <span class="mb-space-chrome__search-ico" aria-hidden="true" />
          </button>
        </div>
        <div class="mb-space-chrome__stats" aria-label="空间数据">
          <button
            type="button"
            class="mb-space-chrome__stat mb-space-chrome__stat--link"
            :class="{ 'is-on': relationsTab === 'following' }"
            @click="onRelationsStatClick('following')"
          >
            <span class="mb-space-chrome__stat-k">关注</span>
            <span class="mb-space-chrome__stat-v">{{ statFollowing }}</span>
          </button>
          <button
            type="button"
            class="mb-space-chrome__stat mb-space-chrome__stat--link"
            :class="{ 'is-on': relationsTab === 'followers' }"
            @click="onRelationsStatClick('followers')"
          >
            <span class="mb-space-chrome__stat-k">粉丝</span>
            <span class="mb-space-chrome__stat-v">{{ statFans }}</span>
          </button>
          <div class="mb-space-chrome__stat">
            <span class="mb-space-chrome__stat-k">获赞</span>
            <span class="mb-space-chrome__stat-v">{{ statLikes }}</span>
          </div>
          <div class="mb-space-chrome__stat">
            <span class="mb-space-chrome__stat-k">播放</span>
            <span class="mb-space-chrome__stat-v">{{ statPlays }}</span>
          </div>
        </div>
      </div>
    </nav>
  </div>
</template>

<script>
import { createNamespacedHelpers } from "vuex";
import bannerUrl from "@/assets/personal_space/1sz3p8w2Sk.png@3840w_400h_1c_100q.avif";
import iconHome from "@/assets/home.png";
import iconUpdate from "@/assets/personal_space/update.png";
import iconContribute from "@/assets/personal_space/contribute.png";
import iconCollect from "@/assets/personal_space/collect.png";
import iconSet from "@/assets/personal_space/set.png";
import genderMaleIco from "@/assets/personal_space/male.png";
import genderFemaleIco from "@/assets/personal_space/female.png";
import akari from "@/assets/akari.jpg";
import MbSpaceHeaderActions from "@/components/minibili/MbSpaceHeaderActions.vue";
import MbSpacePerspective from "@/components/minibili/MbSpacePerspective.vue";
import { personalSpaceZhCN } from "@/i18n/personalSpace.zh-CN";
import { getAccessToken, getUserId } from "@/utils/authTokens";
import { isSpacePerspectivePreviewMode } from "@/utils/spacePerspective";
import { showMbDarkToast } from "@/utils/mbToast";
import { levelIconUrl } from "@/utils/userLevel";

const { mapState } = createNamespacedHelpers("login");
const DEFAULT_SIGN = "这个家伙很懒，什么都没有写";

export default {
  name: "MbSpaceChrome",
  components: { MbSpaceHeaderActions, MbSpacePerspective },
  props: {
    userId: { type: Number, default: 0 },
    profile: { type: Object, default: null },
    activeNav: { type: String, default: "" },
    relationsTab: { type: String, default: null },
    contribBadge: { type: Number, default: null },
    statFollowing: { type: Number, default: 0 },
    statFans: { type: Number, default: 0 },
    statLikes: { type: Number, default: 0 },
    statPlays: { type: Number, default: 0 },
    searchKeyword: { type: String, default: "" },
    followedByMe: { type: Boolean, default: false },
    /** self | fan | visitor — 个人空间视角预览时传入 */
    perspective: { type: String, default: "self" }
  },
  emits: [
    "nav",
    "relations",
    "update:searchKeyword",
    "update:followedByMe",
    "follower-count",
    "update:perspective"
  ],
  data() {
    return { bannerUrl, genderMaleIco, genderFemaleIco, akari };
  },
  computed: {
    ...mapState({
      proInfo: (s) => s.proInfo,
      minibiliMe: (s) => s.minibiliMe
    }),
    navProfileRecord() {
      const p = this.proInfo;
      return p && typeof p === "object" && !Array.isArray(p) ? p : null;
    },
    perspectiveMode() {
      const p = String(this.perspective || "self");
      return isSpacePerspectivePreviewMode(p) ? p : "self";
    },
    isRealSpaceOwner() {
      const uid = this.userId;
      if (!uid) return false;
      const me = getUserId();
      return me != null && Number(me) === uid;
    },
    isPerspectivePreview() {
      return (
        this.isRealSpaceOwner && isSpacePerspectivePreviewMode(this.perspectiveMode)
      );
    },
    isSpaceOwner() {
      if (this.isPerspectivePreview) {
        return false;
      }
      const uid = this.userId;
      if (!uid) return false;
      const me = getUserId();
      if (me != null && Number(me) === uid) {
        return true;
      }
      const p = this.profile;
      return !!(p && Number(p.user_id) === uid && p.is_owner === true);
    },
    mbLoggedIn() {
      return !!getAccessToken();
    },
    spacePrivacyView() {
      const p = this.profile;
      if (p && p.privacy && typeof p.privacy === "object") {
        return {
          public_favorites: !!p.privacy.public_favorites,
          public_following: !!p.privacy.public_following,
          public_fans: !!p.privacy.public_fans
        };
      }
      return null;
    },
    canViewCollectTab() {
      if (this.isSpaceOwner) {
        return true;
      }
      const priv = this.spacePrivacyView;
      return !!(priv && priv.public_favorites);
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
    displayName() {
      if (this.profile && String(this.profile.nickname || "").trim()) {
        return String(this.profile.nickname).trim();
      }
      return this.userId ? "UID " + this.userId : "?";
    },
    displaySign() {
      if (this.profile && String(this.profile.sign || "").trim()) {
        return String(this.profile.sign).trim();
      }
      return DEFAULT_SIGN;
    },
    avatarDisplay() {
      const u = this.profile && String(this.profile.avatar_url || "").trim();
      return u || this.akari;
    },
    spaceGenderKey() {
      const p =
        this.profile && typeof this.profile === "object" ? this.profile : null;
      let g = p && p.gender != null ? String(p.gender).trim() : "";
      if (
        g !== "male" &&
        g !== "female" &&
        g !== "secret" &&
        this.isSpaceOwner &&
        this.minibiliMe &&
        this.minibiliMe.gender != null
      ) {
        g = String(this.minibiliMe.gender).trim();
      }
      if (g === "male" || g === "female") return g;
      return "secret";
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
        this.isSpaceOwner &&
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
    navTabs() {
      const badge =
        this.contribBadge != null && this.contribBadge > 0
          ? this.contribBadge
          : null;
      return [
        {
          key: "home",
          label: "主页",
          icon: iconHome,
          iconClass: "mb-space-chrome__tab-ico--home",
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
          badge
        },
        {
          key: "collect",
          label: "收藏",
          icon: iconCollect,
          iconClass: "mb-space-chrome__tab-ico--collect",
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
    }
  },
  methods: {
    onPerspectiveUpdate(mode) {
      this.$emit("update:perspective", mode);
    },
    onRelationsStatClick(tab) {
      const t = tab === "followers" ? "followers" : "following";
      if (!this.profile) {
        this.$emit("relations", t);
        return;
      }
      if (t === "following" && !this.canOpenFollowingList) {
        showMbDarkToast(personalSpaceZhCN.relations.followingHiddenToast);
        return;
      }
      if (t === "followers" && !this.canOpenFollowersList) {
        showMbDarkToast(personalSpaceZhCN.relations.followersHiddenToast);
        return;
      }
      this.$emit("relations", t);
    },
    levelIconUrl,
    onSearchInput(e) {
      const v = e && e.target ? e.target.value : "";
      this.$emit("update:searchKeyword", v);
    },
    openLoginModal() {
      this.$store.commit("login/SET_LOGIN_TAB", 0);
      this.$store.commit("login/OPEN_LOGIN_MODAL");
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../../styles/mb-space-chrome.scss";
</style>
