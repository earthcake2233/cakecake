<template>
  <div class="mb-space">
    <div class="mb-space__col">
      <MbSpaceChrome
        :user-id="userIdNum"
        :profile="profile"
        :perspective="spacePerspectiveQuery"
        active-nav=""
        :relations-tab="activeTab"
        :stat-following="followingTotal"
        :stat-fans="followerTotal"
        :followed-by-me="spaceFollowedByMe"
        @update:followed-by-me="spaceFollowedByMe = $event"
        @follower-count="followerTotal = $event"
        @nav="onChromeNav"
        @relations="onChromeRelations"
        @update:perspective="onRelationsPerspectiveChange"
      />

      <div class="mb-space__body">
        <p v-if="loadError" class="mb-rel__denied">{{ loadError }}</p>

        <div v-else class="mb-rel__layout">
          <aside class="mb-rel__side" aria-label="关注与粉丝导航">
            <div class="mb-rel__side-block">
              <h2 class="mb-rel__side-title">{{ sideFollowingTitle }}</h2>
              <button
                v-if="canManageFollowGroups"
                type="button"
                class="mb-rel__side-new"
                @click="openFollowGroupCreate"
              >
                + 新建分组
              </button>
              <button
                type="button"
                class="mb-rel__side-item"
                :class="{ 'is-on': activeTab === 'following' && !activeGroupId }"
                @click="selectFollowingAll"
              >
                <span>{{ rel.sideAllFollowing }}</span>
                <span class="mb-rel__side-count">{{ followingTotal }}</span>
              </button>
              <button
                v-for="g in followGroups"
                :key="'fg-' + g.id"
                type="button"
                class="mb-rel__side-item mb-rel__side-item--group"
                :class="{ 'is-on': activeTab === 'following' && activeGroupId === g.id }"
                @click="selectFollowGroup(g.id)"
                @mouseenter="followGroupHoverId = g.id"
                @mouseleave="onFollowGroupItemLeave(g.id)"
              >
                <span class="mb-rel__side-label">{{ g.name }}</span>
                <span class="mb-rel__side-trail">
                  <span
                    class="mb-rel__side-count"
                    :class="{
                      'is-hidden':
                        canManageFollowGroups &&
                        (followGroupHoverId === g.id || followGroupMenuId === g.id)
                    }"
                  >{{ g.member_count || 0 }}</span>
                  <span
                    v-if="canManageFollowGroups"
                    class="mb-rel__side-more"
                    :class="{
                      'is-active':
                        followGroupHoverId === g.id || followGroupMenuId === g.id
                    }"
                    @mouseenter.stop="onFollowGroupMoreEnter(g.id)"
                    @mouseleave.stop="onFollowGroupMoreLeave(g.id)"
                    @click.stop
                  >
                    <span class="mb-rel__side-more-dots" aria-hidden="true">
                      <i /><i /><i />
                    </span>
                    <ul
                      v-show="followGroupMenuId === g.id"
                      class="mb-rel__side-menu"
                      role="menu"
                      @mouseenter.stop="onFollowGroupMoreEnter(g.id)"
                      @mouseleave.stop="onFollowGroupMoreLeave(g.id)"
                      @click.stop
                    >
                      <li role="none">
                        <button
                          type="button"
                          role="menuitem"
                          @click.stop="openFollowGroupEdit(g)"
                        >
                          修改名称
                        </button>
                      </li>
                      <li role="none">
                        <button
                          type="button"
                          role="menuitem"
                          @click.stop="openFollowGroupDelete(g)"
                        >
                          删除分组
                        </button>
                      </li>
                    </ul>
                  </span>
                </span>
              </button>
              <button type="button" class="mb-rel__side-item" disabled>
                <span>特别关注</span>
                <span class="mb-rel__side-count">0</span>
              </button>
              <button type="button" class="mb-rel__side-item" disabled>
                <span>悄悄关注</span>
                <span class="mb-rel__side-count">0</span>
              </button>
              <button type="button" class="mb-rel__side-item" disabled>
                <span>默认分组</span>
                <span class="mb-rel__side-count">0</span>
              </button>
            </div>
            <div class="mb-rel__side-divider" aria-hidden="true" />
            <div class="mb-rel__side-block">
              <h2 class="mb-rel__side-title">{{ sideFansTitle }}</h2>
              <button
                type="button"
                class="mb-rel__side-item"
                :class="{ 'is-on': activeTab === 'followers' }"
                @click="switchTab('followers')"
              >
                <span>{{ sideFansLabel }}</span>
                <span class="mb-rel__side-count">{{ followerTotal }}</span>
              </button>
            </div>
            <div class="mb-rel__side-divider" aria-hidden="true" />
          </aside>

          <main class="mb-rel__main">
            <header class="mb-rel__head">
              <div class="mb-rel__head-left">
                <h1 class="mb-rel__title">{{ mainTitle }}</h1>
                <div v-if="activeTab === 'following'" class="mb-rel__subtabs">
                  <button type="button" class="mb-rel__subtab is-on" disabled>
                    最近关注
                  </button>
                  <button type="button" class="mb-rel__subtab" disabled>
                    最常访问
                  </button>
                </div>
              </div>
              <div class="mb-rel__head-right">
                <button type="button" class="mb-rel__batch" disabled>批量操作</button>
                <div class="mb-rel__search">
                  <input
                    v-model.trim="searchKeyword"
                    type="search"
                    class="mb-rel__search-input"
                    placeholder="输入关键词"
                    autocomplete="off"
                  />
                  <button type="button" class="mb-rel__search-btn" aria-label="搜索">
                    <svg viewBox="0 0 24 24" aria-hidden="true">
                      <path
                        fill="currentColor"
                        d="M15.5 14h-.79l-.28-.27A6.471 6.471 0 0016 9.5 6.5 6.5 0 109.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"
                      />
                    </svg>
                  </button>
                </div>
              </div>
            </header>

            <p v-if="listPrivacyDenied" class="mb-rel__denied">
              {{ listPrivacyDeniedMessage }}
            </p>
            <p v-else-if="listLoading" class="mb-rel__loading">加载中...</p>
            <div
              v-else-if="!filteredItems.length"
              class="mb-rel__empty"
              role="img"
              :aria-label="emptyHint"
            >
              <img :src="emptyImg" alt="" />
            </div>
            <ul v-else class="mb-rel__grid">
              <li v-for="u in filteredItems" :key="'rel-' + u.user_id">
                <article class="mb-rel__card">
                  <router-link
                    class="mb-rel__avatar-link"
                    :to="userSpaceRoute(u.user_id)"
                  >
                    <img
                      class="mb-rel__avatar"
                      :src="u.avatar_url || akari"
                      :alt="u.nickname"
                    />
                  </router-link>
                  <div class="mb-rel__card-body">
                    <div class="mb-rel__name-row">
                      <router-link
                        class="mb-rel__name"
                        :to="userSpaceRoute(u.user_id)"
                        :title="u.nickname"
                      >
                        {{ u.nickname }}
                      </router-link>
                    </div>
                    <p class="mb-rel__sign" :title="displayUserSign(u)">
                      {{ displayUserSign(u) }}
                    </p>
                    <MbRelFollowTag
                      :user-id="u.user_id"
                      :nickname="u.nickname"
                      :label="relationTag(u)"
                      :interactive="canRelFollowActions"
                      :follow-groups="followGroups"
                      @unfollow="onRelUnfollow"
                      @group-change="onRelGroupChange"
                      @groups-updated="loadFollowGroups"
                    />
                  </div>
                </article>
              </li>
            </ul>
          </main>
        </div>
      </div>
    </div>
    <MbFollowGroupCreateDialog
      v-model="followGroupDialogOpen"
      :mode="followGroupDialogMode"
      :initial="followGroupEditInitial"
      :loading="followGroupDialogSaving"
      @submit="onFollowGroupDialogSubmit"
    />
    <MbStationDialog
      v-model="followGroupDeleteOpen"
      title="删除分组"
      message="删除后，该分组下的用户依旧保留？"
      :loading="followGroupDeleteSaving"
      @confirm="onFollowGroupDeleteConfirm"
      @cancel="followGroupDeleteOpen = false"
    />
  </div>
</template>

<script>
import { ElMessage } from "element-plus";
import akari from "@/assets/akari.jpg";
import emptyImg from "@/assets/empty_2.png";
import MbSpaceChrome from "@/components/minibili/MbSpaceChrome.vue";
import MbFollowGroupCreateDialog from "@/components/minibili/MbFollowGroupCreateDialog.vue";
import MbRelFollowTag from "@/components/minibili/MbRelFollowTag.vue";
import MbStationDialog from "@/components/minibili/MbStationDialog.vue";
import {
  mbCreateFollowGroup,
  mbDeleteFollowGroup,
  mbGetUserPublic,
  mbListMyFollowGroups,
  mbListUserFollowing,
  mbListUserFollowers,
  mbUpdateFollowGroup
} from "@/api/minibili";
import { personalSpaceZhCN } from "@/i18n/personalSpace.zh-CN";
import { showMbDarkToast } from "@/utils/mbToast";
import { getUserId } from "@/utils/authTokens";
import {
  buildSpaceViewerProfile,
  isSpacePerspectivePreviewMode,
  resolveSpacePerspective,
  writeStoredSpacePerspective
} from "@/utils/spacePerspective";
import {
  minibiliUserSpaceRoute,
  minibiliUserSpaceRelationsRoute
} from "@/utils/minibiliRoutes";

export default {
  name: "MinibiliSpaceRelations",
  components: {
    MbSpaceChrome,
    MbFollowGroupCreateDialog,
    MbRelFollowTag,
    MbStationDialog
  },
  data() {
    return {
      akari,
      emptyImg,
      profile: null,
      items: [],
      followGroups: [],
      followingTotal: 0,
      followerTotal: 0,
      listLoading: false,
      loadError: "",
      listPrivacyDenied: null,
      searchKeyword: "",
      sortMode: "recent",
      followGroupHoverId: null,
      followGroupMenuId: null,
      _followGroupMenuCloseTimer: null,
      followGroupDialogOpen: false,
      followGroupDialogMode: "create",
      followGroupEditTarget: null,
      followGroupDialogSaving: false,
      followGroupDeleteOpen: false,
      followGroupDeleteTarget: null,
      followGroupDeleteSaving: false,
      spaceFollowedByMe: false,
      _profileOwnerSnapshot: null,
      rel: personalSpaceZhCN.relations
    };
  },
  computed: {
    userIdNum() {
      const raw = this.$route.params.userId;
      const n = parseInt(String(raw || "").trim(), 10);
      return Number.isFinite(n) && n > 0 ? n : 0;
    },
    activeTab() {
      return this.$route.query.tab === "followers" ? "followers" : "following";
    },
    spacePerspectiveQuery() {
      return resolveSpacePerspective(
        this.userIdNum,
        this.$route.query.perspective,
        this.isRealSpaceOwner
      );
    },
    isPerspectivePreview() {
      return (
        this.isRealSpaceOwner &&
        isSpacePerspectivePreviewMode(this.spacePerspectiveQuery)
      );
    },
    relViewAsOwner() {
      return this.isRealSpaceOwner && !this.isPerspectivePreview;
    },
    sideFollowingTitle() {
      return this.relViewAsOwner
        ? personalSpaceZhCN.relations.sideMyFollowing
        : personalSpaceZhCN.relations.sideTheirFollowing;
    },
    sideFansTitle() {
      return this.relViewAsOwner
        ? personalSpaceZhCN.relations.sideMyFans
        : personalSpaceZhCN.relations.sideTheirFans;
    },
    sideFansLabel() {
      return this.sideFansTitle;
    },
    activeGroupId() {
      if (this.activeTab !== "following") {
        return 0;
      }
      const raw = this.$route.query.groupId;
      const n = parseInt(String(raw || "").trim(), 10);
      return Number.isFinite(n) && n > 0 ? n : 0;
    },
    canManageFollowGroups() {
      return this.isSpaceOwner && getUserId() != null;
    },
    canRelFollowActions() {
      return (
        this.canManageFollowGroups &&
        this.relViewAsOwner &&
        this.activeTab === "following"
      );
    },
    selectedFollowGroup() {
      if (!this.activeGroupId) {
        return null;
      }
      return this.followGroups.find((g) => g.id === this.activeGroupId) || null;
    },
    spaceRoute() {
      const r = minibiliUserSpaceRoute(this.userIdNum);
      return r || { name: "home" };
    },
    isRealSpaceOwner() {
      const uid = this.userIdNum;
      if (!uid) return false;
      const me = getUserId();
      return me != null && Number(me) === uid;
    },
    isSpaceOwner() {
      if (this.isPerspectivePreview) {
        return false;
      }
      return this.isRealSpaceOwner;
    },
    canViewFollowing() {
      if (this.relViewAsOwner) {
        return true;
      }
      return !!(
        this.profile &&
        this.profile.privacy &&
        this.profile.privacy.public_following
      );
    },
    canViewFollowers() {
      if (this.relViewAsOwner) {
        return true;
      }
      return !!(
        this.profile &&
        this.profile.privacy &&
        this.profile.privacy.public_fans
      );
    },
    mainTitle() {
      if (this.activeTab === "followers") {
        return this.sideFansTitle;
      }
      if (this.activeGroupId && this.selectedFollowGroup) {
        return this.selectedFollowGroup.name;
      }
      return personalSpaceZhCN.relations.titleAllFollowing;
    },
    listPrivacyDeniedMessage() {
      if (this.listPrivacyDenied === "followers") {
        return personalSpaceZhCN.relations.followersHiddenInline;
      }
      if (this.listPrivacyDenied === "following") {
        return personalSpaceZhCN.relations.followingHiddenInline;
      }
      return "";
    },
    emptyHint() {
      if (this.activeTab === "followers") {
        return "还没有粉丝";
      }
      if (this.activeGroupId) {
        return "该分组还没有关注";
      }
      return "还没有关注任何人";
    },
    followGroupEditInitial() {
      const g = this.followGroupEditTarget;
      if (!g) return null;
      return { name: g.name };
    },
    filteredItems() {
      const q = this.searchKeyword.trim().toLowerCase();
      let list = [...this.items];
      if (this.sortMode === "recent") {
        list.sort((a, b) =>
          String(b.followed_at || "").localeCompare(String(a.followed_at || ""))
        );
      }
      if (!q) return list;
      return list.filter((u) => {
        const nick = String(u.nickname || "").toLowerCase();
        const sign = String(u.sign || "").toLowerCase();
        return nick.includes(q) || sign.includes(q);
      });
    }
  },
  watch: {
    userIdNum() {
      void this.bootstrap();
    },
    "$route.query.tab"() {
      void this.loadList();
    },
    "$route.query.groupId"() {
      void this.loadList();
    },
    "$route.query.perspective"() {
      this.applyRelationsPerspectiveProfile();
      void this.loadList();
    },
    spacePerspectiveQuery() {
      this.applyRelationsPerspectiveProfile();
    }
  },
  mounted() {
    void this.bootstrap();
  },
  beforeUnmount() {
    this.clearFollowGroupMenuTimer();
  },
  methods: {
    userSpaceRoute(userId) {
      const r = minibiliUserSpaceRoute(userId);
      return r || this.spaceRoute;
    },
    displayUserSign(u) {
      const s = String((u && u.sign) || "").trim();
      return s || "这个人很懒，什么都没有写";
    },
    relationTag(u) {
      if (this.activeTab === "followers") {
        return this.rel.tagFans;
      }
      return u.mutual ? this.rel.tagMutual : this.rel.tagFollowing;
    },
    onRelUnfollow(payload) {
      const uid = Number(payload && payload.userId) || 0;
      if (!uid) return;
      this.items = this.items.filter((u) => Number(u.user_id) !== uid);
      this.followingTotal = Math.max(0, (Number(this.followingTotal) || 0) - 1);
      void this.loadFollowGroups();
    },
    onRelGroupChange(payload) {
      const uid = Number(payload && payload.userId) || 0;
      const gid = Number(payload && payload.groupId) || 0;
      const inGroup = !!(payload && payload.inGroup);
      if (
        this.activeGroupId > 0 &&
        gid === this.activeGroupId &&
        !inGroup &&
        uid > 0
      ) {
        this.items = this.items.filter((u) => Number(u.user_id) !== uid);
      }
      void this.loadFollowGroups();
    },
    onChromeNav(key) {
      const base = minibiliUserSpaceRoute(this.userIdNum);
      if (!base) return;
      if (key === "collect") {
        this.$router.push({ ...base, query: { nav: "collect" } }).catch(() => {});
        return;
      }
      if (key === "settings") {
        this.$router.push({ ...base, query: { nav: "settings" } }).catch(() => {});
        return;
      }
      this.$router.push(base).catch(() => {});
    },
    onChromeRelations(tab) {
      this.switchTab(tab === "followers" ? "followers" : "following");
    },
    onRelationsPerspectiveChange(mode) {
      const m =
        mode === "fan" || mode === "visitor" ? mode : "self";
      writeStoredSpacePerspective(this.userIdNum, m);
      const query = { ...this.$route.query };
      if (m === "self") {
        delete query.perspective;
      } else {
        query.perspective = m;
      }
      if (m !== "self" && query.tab === "followers" && !this.canViewFollowers) {
        query.tab = "following";
        delete query.groupId;
      }
      void this.$router
        .replace({ name: this.$route.name, params: this.$route.params, query })
        .then(() => {
          this.applyRelationsPerspectiveProfile();
          if (this.activeTab === "followers" && !this.canViewFollowers) {
            showMbDarkToast(personalSpaceZhCN.relations.followersHiddenToast);
            void this.replaceRelationsTab("following").then(() =>
              this.loadList()
            );
            return;
          }
          void this.loadList();
        });
    },
    relationsExtraQuery() {
      const q = {};
      if (this.spacePerspectiveQuery !== "self") {
        q.perspective = this.spacePerspectiveQuery;
      }
      return q;
    },
    replaceRelationsTab(tab) {
      const t = tab === "followers" ? "followers" : "following";
      const r = minibiliUserSpaceRelationsRoute(
        this.userIdNum,
        t,
        this.relationsExtraQuery()
      );
      if (!r) {
        return Promise.resolve();
      }
      const query = { ...r.query };
      if (t === "following" && this.$route.query.groupId) {
        query.groupId = String(this.$route.query.groupId);
      } else {
        delete query.groupId;
      }
      return this.$router.replace({
        name: r.name,
        params: r.params,
        query
      });
    },
    switchTab(tab) {
      const t = tab === "followers" ? "followers" : "following";
      if (t === "followers" && !this.canViewFollowers) {
        showMbDarkToast(personalSpaceZhCN.relations.followersHiddenToast);
        return;
      }
      if (t === "following" && !this.canViewFollowing) {
        showMbDarkToast(personalSpaceZhCN.relations.followingHiddenToast);
        return;
      }
      if (this.$route.query.tab === t && !this.$route.query.groupId) {
        return;
      }
      void this.replaceRelationsTab(t);
    },
    selectFollowingAll() {
      if (this.activeTab !== "following") {
        this.switchTab("following");
        return;
      }
      this.navigateFollowingGroup(0);
    },
    selectFollowGroup(groupId) {
      const id = Number(groupId) || 0;
      if (!id) return;
      if (this.activeTab !== "following") {
        if (!this.canViewFollowing) {
          showMbDarkToast(personalSpaceZhCN.relations.followingHiddenToast);
          return;
        }
        const r = minibiliUserSpaceRelationsRoute(
          this.userIdNum,
          "following",
          this.relationsExtraQuery()
        );
        if (!r) return;
        this.$router
          .replace({
            ...r,
            query: { ...r.query, groupId: String(id) }
          })
          .catch(() => {});
        return;
      }
      this.navigateFollowingGroup(id);
    },
    navigateFollowingGroup(groupId) {
      const r = minibiliUserSpaceRelationsRoute(
        this.userIdNum,
        "following",
        this.relationsExtraQuery()
      );
      if (!r) return;
      const query = { ...r.query };
      if (groupId > 0) {
        query.groupId = String(groupId);
      }
      const curGid = String(this.$route.query.groupId || "");
      const nextGid = groupId > 0 ? String(groupId) : "";
      if (this.$route.query.tab === "following" && curGid === nextGid) {
        return;
      }
      this.$router.replace({ ...r, query }).catch(() => {});
    },
    clearFollowGroupMenuTimer() {
      if (this._followGroupMenuCloseTimer) {
        clearTimeout(this._followGroupMenuCloseTimer);
        this._followGroupMenuCloseTimer = null;
      }
    },
    onFollowGroupItemLeave(id) {
      if (this.followGroupMenuId === id) {
        return;
      }
      if (this.followGroupHoverId === id) {
        this.followGroupHoverId = null;
      }
    },
    onFollowGroupMoreEnter(id) {
      this.clearFollowGroupMenuTimer();
      this.followGroupHoverId = id;
      this.followGroupMenuId = id;
    },
    onFollowGroupMoreLeave(id) {
      this.clearFollowGroupMenuTimer();
      this._followGroupMenuCloseTimer = setTimeout(() => {
        if (this.followGroupMenuId === id) {
          this.followGroupMenuId = null;
        }
        if (this.followGroupHoverId === id) {
          this.followGroupHoverId = null;
        }
        this._followGroupMenuCloseTimer = null;
      }, 120);
    },
    openFollowGroupCreate() {
      this.followGroupDialogMode = "create";
      this.followGroupEditTarget = null;
      this.followGroupDialogOpen = true;
    },
    openFollowGroupEdit(group) {
      this.clearFollowGroupMenuTimer();
      this.followGroupMenuId = null;
      this.followGroupHoverId = null;
      this.followGroupDialogMode = "edit";
      this.followGroupEditTarget = group;
      this.followGroupDialogOpen = true;
    },
    openFollowGroupDelete(group) {
      this.clearFollowGroupMenuTimer();
      this.followGroupMenuId = null;
      this.followGroupHoverId = null;
      this.followGroupDeleteTarget = group;
      this.followGroupDeleteOpen = true;
    },
    async onFollowGroupDialogSubmit(payload) {
      if (this.followGroupDialogSaving) return;
      this.followGroupDialogSaving = true;
      try {
        if (this.followGroupDialogMode === "edit") {
          const target = this.followGroupEditTarget;
          if (!target || !target.id) return;
          const row = await mbUpdateFollowGroup(target.id, { name: payload.name });
          this.followGroups = this.followGroups.map((g) =>
            g.id === row.id ? row : g
          );
          this.followGroupDialogOpen = false;
          showMbDarkToast("已保存");
        } else {
          const row = await mbCreateFollowGroup({ name: payload.name });
          this.followGroups = [...this.followGroups, row];
          this.followGroupDialogOpen = false;
          showMbDarkToast("分组创建成功");
          this.selectFollowGroup(row.id);
        }
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败，请稍后重试";
        ElMessage.error(String(msg));
      } finally {
        this.followGroupDialogSaving = false;
      }
    },
    async onFollowGroupDeleteConfirm() {
      const target = this.followGroupDeleteTarget;
      if (!target || !target.id || this.followGroupDeleteSaving) return;
      this.followGroupDeleteSaving = true;
      try {
        await mbDeleteFollowGroup(target.id);
        this.followGroups = this.followGroups.filter((g) => g.id !== target.id);
        this.followGroupDeleteOpen = false;
        this.followGroupDeleteTarget = null;
        showMbDarkToast("分组已删除");
        if (this.activeGroupId === target.id) {
          this.navigateFollowingGroup(0);
        }
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "删除失败，请稍后重试";
        ElMessage.error(String(msg));
      } finally {
        this.followGroupDeleteSaving = false;
      }
    },
    async loadFollowGroups() {
      if (!this.canManageFollowGroups) {
        this.followGroups = [];
        return;
      }
      try {
        const res = await mbListMyFollowGroups();
        this.followGroups = Array.isArray(res.items) ? res.items : [];
      } catch {
        this.followGroups = [];
      }
      if (this.activeGroupId > 0) {
        const exists = this.followGroups.some((g) => g.id === this.activeGroupId);
        if (!exists) {
          this.navigateFollowingGroup(0);
        }
      }
    },
    async bootstrap() {
      this.loadError = "";
      this.listPrivacyDenied = null;
      this.items = [];
      if (!this.userIdNum) {
        this.loadError = "无效的用户";
        return;
      }
      const requestUid = this.userIdNum;
      this.profile = null;
      try {
        const profile = await mbGetUserPublic(requestUid, {
          skipGlobalErrorToast: true
        });
        if (this.userIdNum !== requestUid) {
          return;
        }
        this._profileOwnerSnapshot = profile;
        this.applyRelationsPerspectiveProfile();
        this.followingTotal = Number(this.profile.following_count) || 0;
        this.followerTotal = Number(this.profile.follower_count) || 0;
        this.spaceFollowedByMe = !!this.profile.followed_by_me;
      } catch (e) {
        if (this.userIdNum !== requestUid) {
          return;
        }
        this.loadError = (e && e.message) || "无法加载用户资料";
        this.profile = null;
        return;
      }
      await this.loadFollowGroups();
      if (
        isSpacePerspectivePreviewMode(this.spacePerspectiveQuery) &&
        !this.$route.query.perspective
      ) {
        await this.$router
          .replace({
            name: this.$route.name,
            params: this.$route.params,
            query: {
              ...this.$route.query,
              perspective: this.spacePerspectiveQuery
            }
          })
          .catch(() => {});
      }
      await this.loadList();
    },
    applyRelationsPerspectiveProfile() {
      const snap = this._profileOwnerSnapshot;
      if (!snap || typeof snap !== "object") {
        return;
      }
      if (this.spacePerspectiveQuery === "self") {
        this.profile = { ...snap };
      } else if (isSpacePerspectivePreviewMode(this.spacePerspectiveQuery)) {
        this.profile = buildSpaceViewerProfile(
          snap,
          this.spacePerspectiveQuery
        );
      } else {
        this.profile = { ...snap };
      }
      if (this.profile) {
        this.spaceFollowedByMe = !!this.profile.followed_by_me;
      }
    },
    async loadList() {
      if (!this.userIdNum || !this.profile) return;
      if (Number(this.profile.user_id) !== this.userIdNum) return;
      const requestUid = this.userIdNum;
      const tab = this.activeTab;
      const allowed =
        tab === "followers" ? this.canViewFollowers : this.canViewFollowing;
      if (!allowed) {
        this.loadError = "";
        this.listPrivacyDenied = tab;
        this.items = [];
        this.listLoading = false;
        if (tab === "followers" && this.canViewFollowing) {
          void this.replaceRelationsTab("following").then(() => this.loadList());
        }
        return;
      }
      this.listPrivacyDenied = null;
      this.loadError = "";
      this.listLoading = true;
      try {
        if (tab === "followers") {
          const res = await mbListUserFollowers(requestUid, { limit: 200 });
          if (this.userIdNum !== requestUid) {
            return;
          }
          this.items = Array.isArray(res.items) ? res.items : [];
          this.followerTotal = Number(res.total) || this.items.length;
        } else {
          const params = { limit: 200 };
          if (this.activeGroupId > 0) {
            params.groupId = this.activeGroupId;
          }
          const res = await mbListUserFollowing(requestUid, params);
          if (this.userIdNum !== requestUid) {
            return;
          }
          this.items = Array.isArray(res.items) ? res.items : [];
          if (this.activeGroupId > 0) {
            const cnt = Number(res.total) || this.items.length;
            this.followGroups = this.followGroups.map((g) =>
              g.id === this.activeGroupId ? { ...g, member_count: cnt } : g
            );
          } else {
            this.followingTotal = Number(res.total) || this.items.length;
          }
        }
      } catch (e) {
        if (this.userIdNum !== requestUid) {
          return;
        }
        const msg = (e && e.message) || "";
        const code = e && e.minibiliApiCode;
        if (
          code === 40300 ||
          msg.includes("无权限") ||
          msg.includes("403") ||
          msg.includes("Forbidden")
        ) {
          this.loadError =
            tab === "followers"
              ? "对方已设置不公开粉丝列表"
              : "对方已设置不公开关注列表";
        } else {
          this.loadError = msg || "加载失败";
        }
        this.items = [];
      } finally {
        if (this.userIdNum === requestUid) {
          this.listLoading = false;
        }
      }
    }
  }
};
</script>

<style lang="scss" scoped>
.mb-space {
  min-height: 100vh;
  background: #fff;
  box-sizing: border-box;
}

.mb-space__col {
  width: 100%;
  max-width: 2260px;
  min-height: 100vh;
  margin-left: auto;
  margin-right: auto;
  display: flex;
  flex-direction: column;
  background: #fff;
}

.mb-space__body {
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: #fff;
}

@import "../../styles/mb-space-relations.scss";
</style>
