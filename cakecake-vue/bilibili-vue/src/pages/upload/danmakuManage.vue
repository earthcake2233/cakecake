<template>
  <CreatorShell>
    <div class="cm-wrap">
      <div class="cm-panel">
        <div class="cm-head-row">
          <div class="cm-top-tabs">
            <button type="button" class="cm-top-tab" :class="{ on: topTab === 'video' }" @click="setTopTab('video')">稿件弹幕</button>
            <button type="button" class="cm-top-tab" :class="{ on: topTab === 'settings' }" @click="setTopTab('settings')">弹幕设置</button>
            <button type="button" class="cm-top-tab" :class="{ on: topTab === 'feedback' }" @click="setTopTab('feedback')">弹幕反馈</button>
          </div>
          <div v-if="topTab === 'video'" class="cm-head-search">
            <div class="cm-search">
              <input
                v-model.trim="searchQ"
                type="search"
                class="cm-search-input"
                placeholder="搜索弹幕关键字"
                autocomplete="off"
                @keydown.enter.prevent="onSearch"
              />
              <button type="button" class="cm-search-btn" :aria-label="\u641c\u7d22" @click="onSearch" />
            </div>
          </div>
        </div>

        <div v-if="topTab === 'video'" class="cm-action-row dm-action-row">
          <div class="cm-action-left">
            <button
              type="button"
              class="cm-tool-btn cm-tool-btn--danger"
              :disabled="!selectedIds.length || deleting"
              @click="onBatchDelete"
            >{{ deleting ? "删除中…" : "删除弹幕" }}</button>
            <div class="dm-seg-groups">
              <div class="dm-seg-group" role="group">
                <button type="button" class="dm-seg-btn" disabled :title="\u5373\u5c06\u5f00\u653e">弹幕保护</button>
                <span class="dm-seg-divider" aria-hidden="true" />
                <button type="button" class="dm-seg-btn" disabled :title="\u5373\u5c06\u5f00\u653e">取消保护</button>
              </div>
              <div class="dm-seg-group" role="group">
                <button type="button" class="dm-seg-btn" disabled :title="\u5373\u5c06\u5f00\u653e">字幕</button>
                <span class="dm-seg-divider" aria-hidden="true" />
                <button
                  type="button"
                  class="dm-seg-btn"
                  :class="{ on: typeFilter === 'scroll' }"
                  @click="toggleTypeFilter('scroll')"
                >普通</button>
              </div>
            </div>
            <button
              type="button"
              class="dm-refresh-btn"
              :disabled="loading"
              :aria-label="\u5237\u65b0"
              @click="fetchList"
            >
              <span class="dm-refresh-ico" aria-hidden="true" />
            </button>
          </div>
          <div class="dm-action-filters">
            <div class="dm-filter">
              <select v-model="typeFilter" class="dm-filter-select" @change="onFilterChange">
                <option value="">全部弹幕</option>
                <option value="scroll">普通弹幕</option>
                <option value="top">顶部弹幕</option>
                <option value="bottom">底部弹幕</option>
              </select>
            </div>
            <CmVideoPicker
              v-model="videoFilter"
              :options="videoOptions"
              :all-label="\u5168\u90e8\u89c6\u9891"
              @change="onFilterChange"
            />
          </div>
        </div>

        <template v-if="topTab === 'video'">
          <p v-if="loadError" class="cm-hint cm-hint--err">{{ loadError }}</p>
          <p v-else-if="loading && showListArea && !rows.length" class="cm-hint">加载中…</p>
          <div v-else-if="showDanmakuList" class="dm-table-wrap">
            <table class="dm-table">
              <thead>
                <tr>
                  <th class="dm-col-check"><input v-model="selectAll" type="checkbox" class="cm-check" :disabled="!canSelectRows" @change="onSelectAllChange" /></th>
                  <th class="dm-col-user">发送者</th>
                  <th class="dm-col-time dm-th-sortable">
                    <span class="dm-th-inner">
                      <span class="dm-th-label">播放时间</span>
                      <span class="dm-sort-arrows">
                        <button
                          type="button"
                          class="dm-sort-btn dm-sort-btn--up"
                          :class="{ on: sortField === 'play_time' && sortDir === 'asc' }"
                          :aria-label="\u64ad\u653e\u65f6\u95f4\u5347\u5e8f"
                          @click="setSort('play_time', 'asc')"
                        />
                        <button
                          type="button"
                          class="dm-sort-btn dm-sort-btn--down"
                          :class="{ on: sortField === 'play_time' && sortDir === 'desc' }"
                          :aria-label="\u64ad\u653e\u65f6\u95f4\u964d\u5e8f"
                          @click="setSort('play_time', 'desc')"
                        />
                      </span>
                    </span>
                  </th>
                  <th class="dm-col-content">弹幕内容</th>
                  <th class="dm-col-like dm-th-sortable">
                    <span class="dm-th-inner dm-th-inner--center">
                      <span class="dm-th-label">点赞</span>
                      <span class="dm-sort-arrows">
                        <button
                          type="button"
                          class="dm-sort-btn dm-sort-btn--up"
                          :class="{ on: sortField === 'like_count' && sortDir === 'asc' }"
                          :aria-label="\u70b9\u8d5e\u5347\u5e8f"
                          @click="setSort('like_count', 'asc')"
                        />
                        <button
                          type="button"
                          class="dm-sort-btn dm-sort-btn--down"
                          :class="{ on: sortField === 'like_count' && sortDir === 'desc' }"
                          :aria-label="\u70b9\u8d5e\u964d\u5e8f"
                          @click="setSort('like_count', 'desc')"
                        />
                      </span>
                    </span>
                  </th>
                  <th class="dm-col-attr">属性</th>
                  <th class="dm-col-sent">发送时间</th>
                  <th class="dm-col-op">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in displayRows" :key="row.id">
                  <td class="dm-col-check"><input v-model="selectedIds" type="checkbox" class="cm-check" :value="row.id" /></td>
                  <td class="dm-col-user"><span class="dm-user">{{ row.username || "用户" }}</span></td>
                  <td class="dm-col-time">{{ row.play_time }}</td>
                  <td class="dm-col-content dm-content-cell">
                    <p class="dm-content">{{ row.content }}</p>
                    <a v-if="videoHref(row)" class="dm-video-link" :href="videoHref(row)" target="_blank" rel="noopener noreferrer">视频：{{ videoTitleShort(row) }}</a>
                  </td>
                  <td class="dm-col-like">
                    <button
                      type="button"
                      class="dm-like-btn"
                      :class="{ 'is-on': row.liked_by_me }"
                      :disabled="likeBusy === row.id"
                      @click="onToggleLike(row)"
                    >
                      <span class="dm-like-ico" aria-hidden="true" />
                      <span class="dm-like-num">{{ row.like_count || 0 }}</span>
                    </button>
                  </td>
                  <td class="dm-col-attr">普通</td>
                  <td class="dm-col-sent">
                    <div class="dm-sent-date">{{ sentDate(row) }}</div>
                    <div class="dm-sent-time">{{ sentClock(row) }}</div>
                  </td>
                  <td class="dm-col-op">
                    <div class="dm-row-menu">
                      <button
                        type="button"
                        class="dm-row-menu-btn"
                        :aria-label="\u64cd\u4f5c"
                        @mouseenter="openRowMenu(row, $event)"
                        @mouseleave="scheduleCloseRowMenu"
                      >···</button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
            <p class="dm-foot-note">弹幕列表显示最新的稿件弹幕，上限1000条</p>
          </div>
          <div v-else-if="showEmpty" class="cm-empty">
            <div class="cm-empty-img"><img :src="emptyImg" alt="" width="360" height="auto" /></div>
            <p class="cm-empty-txt">{{ emptyText }}</p>
          </div>
        </template>
        <div v-else class="cm-empty">
          <div class="cm-empty-img"><img :src="emptyImg" alt="" width="360" height="auto" /></div>
          <p class="cm-empty-txt">功能即将开放</p>
        </div>
      </div>
    </div>
    <Teleport to="body">
      <div
        v-if="rowMenuId && rowMenuPos && rowMenuRow"
        class="dm-row-dropdown dm-row-dropdown--fixed"
        :style="{ top: rowMenuPos.top + 'px', left: rowMenuPos.left + 'px' }"
        @mouseenter="cancelCloseRowMenu"
        @mouseleave="closeRowMenu"
      >
        <button
          type="button"
          class="dm-row-dropdown-item"
          disabled
          :title="\u5373\u5c06\u5f00\u653e"
          @click="onRowProtect(rowMenuRow)"
        >弹幕保护</button>
        <button
          type="button"
          class="dm-row-dropdown-item dm-row-dropdown-item--danger"
          @click="onRowDelete(rowMenuRow)"
        >删除弹幕</button>
      </div>
    </Teleport>
  </CreatorShell>
</template>

<script>
import { h } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import "@/styles/cm-del-msgbox.scss";
import CreatorShell from "@/components/creator/CreatorShell.vue";
import CmVideoPicker from "@/components/creator/CmVideoPicker.vue";
import emptyImg from "@/assets/upload_manager/image_text/empty.9e92c422.png";
import {
  mbBlockUser,
  mbDeleteDanmaku,
  mbListCreatorDanmakus,
  mbListMyVideos,
  mbToggleDanmakuLike
} from "@/api/minibili";
import { getAccessToken } from "@/utils/authTokens";
import { formatVideoBvid } from "@/utils/videoBvid";

const isMinibiliMode =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

export default {
  name: "CreatorDanmakuManage",
  components: { CreatorShell, CmVideoPicker },
  data() {
    return {
      emptyImg,
      topTab: "video",
      loading: false,
      deleting: false,
      loadError: "",
      rows: [],
      searchQ: "",
      searchApplied: "",
      videoFilter: "",
      videoOptions: [],
      typeFilter: "",
      sortField: "",
      sortDir: "desc",
      rowMenuId: null,
      rowMenuPos: null,
      rowMenuCloseTimer: null,
      likeBusy: null,
      selectedIds: [],
      selectAll: false
    };
  },
  computed: {
    isMinibiliMode() { return isMinibiliMode; },
    displayRows() {
      const list = [...(this.rows || [])];
      if (!this.sortField) return list;
      const dir = this.sortDir === "asc" ? 1 : -1;
      if (this.sortField === "play_time") {
        list.sort((a, b) => (Number(a.video_time) - Number(b.video_time)) * dir);
      } else if (this.sortField === "like_count") {
        list.sort(
          (a, b) => (Number(a.like_count || 0) - Number(b.like_count || 0)) * dir
        );
      }
      return list;
    },
    rowMenuRow() {
      if (!this.rowMenuId) return null;
      return this.displayRows.find((r) => r.id === this.rowMenuId) || null;
    },
    showListArea() {
      return this.topTab === "video";
    },
    showDanmakuList() {
      return this.showListArea && this.rows.length > 0 && !this.loadError;
    },
    canSelectRows() {
      return this.showDanmakuList;
    },
    showEmpty() {
      if (this.loadError) return false;
      if (this.loading && this.showListArea) return false;
      if (!this.isMinibiliMode) return true;
      if (this.topTab !== "video") return true;
      return !this.rows.length;
    },
    emptyText() {
      if (!this.isMinibiliMode) return "请登录后查看";
      if (this.searchApplied) return "没有匹配的弹幕";
      return "还没有弹幕哦~";
    }
  },
  watch: {
    selectedIds() {
      const list = this.displayRows;
      this.selectAll =
        this.canSelectRows &&
        this.selectedIds.length === list.length &&
        list.length > 0;
    },
    searchQ(val) {
      const q = String(val || "").trim();
      if (!q && this.searchApplied) {
        this.searchApplied = "";
        if (this.topTab === "video" && this.isMinibiliMode && getAccessToken()) {
          void this.fetchList();
        }
      }
    },
    topTab(tab) {
      if (tab === "video" && this.isMinibiliMode && getAccessToken()) {
        this.syncSearchFromInput();
        void this.fetchList();
      }
    }
  },
  mounted() {
    this.applyRouteQuery();
    if (this.isMinibiliMode && getAccessToken()) {
      void this.loadVideos();
      void this.fetchList();
    }
  },
  beforeUnmount() {
    this.cancelCloseRowMenu();
  },
  methods: {
    applyRouteQuery() {
      const q = this.$route && this.$route.query;
      if (!q) return;
      const vid = String(q.video_id || "").trim();
      if (vid) this.videoFilter = vid;
    },
    syncSearchFromInput() {
      this.searchApplied = String(this.searchQ || "").trim();
    },
    setTopTab(tab) {
      if (this.topTab === tab) {
        if (tab === "video" && this.isMinibiliMode && getAccessToken()) {
          this.syncSearchFromInput();
          void this.fetchList();
        }
        return;
      }
      this.topTab = tab;
    },
    videoLink(row) {
      if (!row || !row.video_id) return null;
      return { name: "video", params: { aid: formatVideoBvid(row.video_id) } };
    },
    videoHref(row) {
      const to = this.videoLink(row);
      return to ? this.$router.resolve(to).href : "";
    },
    videoTitleShort(row) {
      const t = String((row && row.video && row.video.title) || "").trim();
      if (t.length <= 18) return t || "未命名视频";
      return t.slice(0, 18) + "…";
    },
    toggleTypeFilter(type) {
      this.typeFilter = this.typeFilter === type ? "" : type;
      void this.fetchList();
    },
    async loadVideos() {
      try {
        const items = [];
        for (let page = 1; page <= 20; page += 1) {
          const res = await mbListMyVideos({ page, page_size: 50, sort: "time" });
          items.push(...(res.items || []));
          if (page >= (res.total_pages || 1)) break;
        }
        this.videoOptions = items;
      } catch { this.videoOptions = []; }
    },
    async fetchList() {
      if (!this.isMinibiliMode || !getAccessToken() || this.topTab !== "video") return;
      this.loading = true;
      this.loadError = "";
      try {
        const params = { limit: 1000 };
        if (this.searchApplied) params.q = this.searchApplied;
        if (this.videoFilter) params.video_id = Number(this.videoFilter);
        if (this.typeFilter) params.type = this.typeFilter;
        const res = await mbListCreatorDanmakus(params);
        this.rows = res.items || [];
        this.selectedIds = [];
        this.selectAll = false;
      } catch (e) {
        this.loadError = (e && e.message) || "加载失败";
        this.rows = [];
      } finally { this.loading = false; }
    },
    onSearch() {
      this.syncSearchFromInput();
      void this.fetchList();
    },
    onFilterChange() {
      if (this.showListArea) void this.fetchList();
    },
    setSort(field, dir) {
      if (this.sortField === field && this.sortDir === dir) {
        this.sortField = "";
        this.sortDir = "desc";
        return;
      }
      this.sortField = field;
      this.sortDir = dir;
    },
    sentDate(row) {
      const s = String((row && row.created_at) || "").trim();
      if (!s) return "—";
      const i = s.indexOf(" ");
      return i >= 0 ? s.slice(0, i) : s;
    },
    sentClock(row) {
      const s = String((row && row.created_at) || "").trim();
      if (!s) return "";
      const i = s.indexOf(" ");
      return i >= 0 ? s.slice(i + 1) : "";
    },
    onRowProtect() {
      ElMessage.info("即将开放");
    },
    openRowMenu(row, evt) {
      this.cancelCloseRowMenu();
      if (!row || !evt || !evt.currentTarget) return;
      const el = evt.currentTarget;
      const rect = el.getBoundingClientRect();
      const menuW = 112;
      const menuH = 84;
      let top = rect.bottom + 4;
      let left = rect.right - menuW;
      if (top + menuH > window.innerHeight - 8) {
        top = rect.top - menuH - 4;
      }
      if (top < 8) top = 8;
      if (left < 8) left = 8;
      if (left + menuW > window.innerWidth - 8) {
        left = window.innerWidth - menuW - 8;
      }
      this.rowMenuId = row.id;
      this.rowMenuPos = { top, left };
    },
    scheduleCloseRowMenu() {
      this.cancelCloseRowMenu();
      this.rowMenuCloseTimer = setTimeout(() => this.closeRowMenu(), 150);
    },
    cancelCloseRowMenu() {
      if (this.rowMenuCloseTimer) {
        clearTimeout(this.rowMenuCloseTimer);
        this.rowMenuCloseTimer = null;
      }
    },
    closeRowMenu() {
      this.cancelCloseRowMenu();
      this.rowMenuId = null;
      this.rowMenuPos = null;
    },
    async onToggleLike(row) {
      if (!row || !row.id || !this.isMinibiliMode || !getAccessToken()) return;
      this.likeBusy = row.id;
      try {
        const res = await mbToggleDanmakuLike(Number(row.id));
        row.liked_by_me = !!res.liked;
        row.like_count = Number(res.like_count) || 0;
      } catch (e) {
        ElMessage.error((e && e.message) || "加载失败");
      } finally {
        this.likeBusy = null;
      }
    },
    onSelectAllChange() {
      const list = this.displayRows;
      this.selectedIds = this.selectAll ? list.map((r) => r.id) : [];
    },
    async confirmDeleteDialog() {
      let blacklistSender = false;
      await ElMessageBox({
        title: "删除提醒",
        message: () =>
          h("div", { class: "cm-del-msgbox-body" }, [
            h("p", { class: "cm-del-msgbox-msg" }, "删除后无法恢复，确认删除选中的弹幕吗？"),
            h("label", { class: "cm-del-msgbox-check" }, [
              h("input", {
                type: "checkbox",
                class: "cm-del-msgbox-check-input",
                onChange: (e) => {
                  blacklistSender = !!(e && e.target && e.target.checked);
                }
              }),
              h("span", { class: "cm-del-msgbox-check-label" }, "拉黑弹幕发送者")
            ])
          ]),
        showCancelButton: true,
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        customClass: "cm-del-msgbox",
        confirmButtonClass: "cm-del-msgbox__ok",
        cancelButtonClass: "cm-del-msgbox__cancel",
        showClose: true,
        distinguishCancelAndClose: true
      });
      return { blacklistSender };
    },
    async blacklistDanmakuSenders(rows) {
      const uids = new Set();
      for (const row of rows || []) {
        const uid = Number(row && row.user_id);
        if (Number.isFinite(uid) && uid > 0) uids.add(uid);
      }
      for (const uid of uids) {
        try {
          await mbBlockUser(uid);
        } catch {
          /* ignore single block failure */
        }
      }
      if (uids.size > 0) {
        ElMessage.success("已拉黑发送者");
      }
    },
    async onRowDelete(row) {
      if (!row || !row.id) return;
      this.closeRowMenu();
      let blacklistSender = false;
      try {
        ({ blacklistSender } = await this.confirmDeleteDialog());
      } catch {
        return;
      }
      this.deleting = true;
      try {
        await mbDeleteDanmaku(row.id);
        if (blacklistSender && row.user_id) {
          await this.blacklistDanmakuSenders([row]);
        }
        ElMessage.success("已删除");
        await this.fetchList();
      } catch (e) {
        ElMessage.error((e && e.message) || "删除失败");
      } finally { this.deleting = false; }
    },
    async onBatchDelete() {
      if (!this.selectedIds.length) return;
      let blacklistSender = false;
      try {
        ({ blacklistSender } = await this.confirmDeleteDialog());
      } catch {
        return;
      }
      const targets = this.displayRows.filter((r) =>
        this.selectedIds.includes(r.id)
      );
      this.deleting = true;
      const ids = [...this.selectedIds];
      let ok = 0;
      try {
        for (const id of ids) {
          await mbDeleteDanmaku(Number(id));
          ok += 1;
        }
        if (blacklistSender && targets.length) {
          await this.blacklistDanmakuSenders(targets);
        }
        ElMessage.success("已删除");
        await this.fetchList();
      } catch (e) {
        if (ok > 0) await this.fetchList();
        ElMessage.error((e && e.message) || "删除失败");
      } finally { this.deleting = false; }
    }
  }
};
</script>

<style lang="scss">
@import "./commentManage.scss";
@import "./danmakuManage.scss";
</style>
