# -*- coding: utf-8 -*-
import json
from pathlib import Path

TARGET = Path(__file__).resolve().parents[1] / "cakecake-vue/bilibili-vue/src/pages/upload/danmakuManage.vue"

S = {
    "tab_video": "\u7a3f\u4ef6\u5f39\u5e55",
    "tab_settings": "\u5f39\u5e55\u8bbe\u7f6e",
    "tab_feedback": "\u5f39\u5e55\u53cd\u9988",
    "search_ph": "\u641c\u7d22\u5f39\u5e55\u5173\u952e\u5b57",
    "search_aria": "\u641c\u7d22",
    "del_btn": "\u5220\u9664\u5f39\u5e55",
    "deleting": "\u5220\u9664\u4e2d\u2026",
    "protect": "\u5f39\u5e55\u4fdd\u62a4",
    "unprotect": "\u53d6\u6d88\u4fdd\u62a4",
    "subtitle": "\u5b57\u5e55",
    "normal": "\u666e\u901a",
    "refresh": "\u5237\u65b0",
    "all_dm": "\u5168\u90e8\u5f39\u5e55",
    "scroll_dm": "\u666e\u901a\u5f39\u5e55",
    "top_dm": "\u9876\u90e8\u5f39\u5e55",
    "bottom_dm": "\u5e95\u90e8\u5f39\u5e55",
    "all_video": "\u5168\u90e8\u89c6\u9891",
    "loading": "\u52a0\u8f7d\u4e2d\u2026",
    "sender": "\u53d1\u9001\u8005",
    "play_time": "\u64ad\u653e\u65f6\u95f4",
    "content": "\u5f39\u5e55\u5185\u5bb9",
    "like": "\u70b9\u8d5e",
    "attr": "\u5c5e\u6027",
    "sent": "\u53d1\u9001\u65f6\u95f4",
    "op": "\u64cd\u4f5c",
    "foot": "\u5f39\u5e55\u5217\u8868\u663e\u793a\u6700\u65b0\u7684\u7a3f\u4ef6\u5f39\u5e55\uff0c\u4e0a\u96501000\u6761",
    "soon": "\u529f\u80fd\u5373\u5c06\u5f00\u653e",
    "empty_hint": "\u8bf7\u5f00\u542f Mini-Bili \u6a21\u5f0f\u5e76\u767b\u5f55\u540e\u67e5\u770b",
    "empty_search": "\u6ca1\u6709\u5339\u914d\u7684\u5f39\u5e55",
    "empty": "\u8fd8\u6ca1\u6709\u5f39\u5e55\u54e6~",
    "user": "\u7528\u6237",
    "untitled": "\u672a\u547d\u540d\u89c6\u9891",
    "load_fail": "\u52a0\u8f7d\u5931\u8d25",
    "del_title": "\u5220\u9664\u63d0\u9192",
    "del_msg": "\u5220\u9664\u540e\u65e0\u6cd5\u6062\u590d\uff0c\u786e\u8ba4\u5220\u9664\u9009\u4e2d\u7684\u5f39\u5e55\u5417\uff1f",
    "ok": "\u786e\u5b9a",
    "cancel": "\u53d6\u6d88",
    "del_done": "\u5df2\u5220\u9664",
    "del_fail": "\u5220\u9664\u5931\u8d25",
    "blacklist_sender": "\u62c9\u9ed1\u5f39\u5e55\u53d1\u9001\u8005",
    "blacklist_done": "\u5df2\u62c9\u9ed1\u53d1\u9001\u8005",
    "soon_title": "\u5373\u5c06\u5f00\u653e",
    "sort_play_asc": "\u64ad\u653e\u65f6\u95f4\u5347\u5e8f",
    "sort_play_desc": "\u64ad\u653e\u65f6\u95f4\u964d\u5e8f",
    "sort_like_asc": "\u70b9\u8d5e\u5347\u5e8f",
    "sort_like_desc": "\u70b9\u8d5e\u964d\u5e8f",
}


def p(*parts):
    return "".join(parts)


out = p(
    """<template>
  <CreatorShell>
    <motion class="cm-wrap">
      <motion class="cm-panel">
        <motion class="cm-head-row">
          <motion class="cm-top-tabs">
            <button type="button" class="cm-top-tab" :class="{ on: topTab === 'video' }" @click="setTopTab('video')">""",
    S["tab_video"],
    """</button>
            <button type="button" class="cm-top-tab" :class="{ on: topTab === 'settings' }" @click="setTopTab('settings')">""",
    S["tab_settings"],
    """</button>
            <button type="button" class="cm-top-tab" :class="{ on: topTab === 'feedback' }" @click="setTopTab('feedback')">""",
    S["tab_feedback"],
    """</button>
          </motion>
          <motion v-if="topTab === 'video'" class="cm-head-search">
            <motion class="cm-search">
              <input
                v-model.trim="searchQ"
                type="search"
                class="cm-search-input"
                placeholder=\"""",
    S["search_ph"],
    """\"
                autocomplete="off"
                @keydown.enter.prevent="onSearch"
              />
              <button type="button" class="cm-search-btn" :aria-label=""",
    json.dumps(S["search_aria"]),
    """ @click="onSearch" />
            </motion>
          </motion>
        </motion>

        <motion v-if="topTab === 'video'" class="cm-action-row dm-action-row">
          <motion class="cm-action-left">
            <button
              type="button"
              class="cm-tool-btn cm-tool-btn--danger"
              :disabled="!selectedIds.length || deleting"
              @click="onBatchDelete"
            >{{ deleting ? \"""",
    S["deleting"],
    """\" : \"""",
    S["del_btn"],
    """\" }}</button>
            <motion class="dm-seg-groups">
              <motion class="dm-seg-group" role="group">
                <button type="button" class="dm-seg-btn" disabled :title=""",
    json.dumps(S["soon_title"]),
    """>""",
    S["protect"],
    """</button>
                <span class="dm-seg-divider" aria-hidden="true" />
                <button type="button" class="dm-seg-btn" disabled :title=""",
    json.dumps(S["soon_title"]),
    """>""",
    S["unprotect"],
    """</button>
              </motion>
              <motion class="dm-seg-group" role="group">
                <button type="button" class="dm-seg-btn" disabled :title=""",
    json.dumps(S["soon_title"]),
    """>""",
    S["subtitle"],
    """</button>
                <span class="dm-seg-divider" aria-hidden="true" />
                <button
                  type="button"
                  class="dm-seg-btn"
                  :class="{ on: typeFilter === 'scroll' }"
                  @click="toggleTypeFilter('scroll')"
                >""",
    S["normal"],
    """</button>
              </motion>
            </motion>
            <button
              type="button"
              class="dm-refresh-btn"
              :disabled="loading"
              :aria-label=""",
    json.dumps(S["refresh"]),
    """
              @click="fetchList"
            >
              <span class="dm-refresh-ico" aria-hidden="true" />
            </button>
          </motion>
          <motion class="dm-action-filters">
            <motion class="dm-filter">
              <select v-model="typeFilter" class="dm-filter-select" @change="onFilterChange">
                <option value="">""",
    S["all_dm"],
    """</option>
                <option value="scroll">""",
    S["scroll_dm"],
    """</option>
                <option value="top">""",
    S["top_dm"],
    """</option>
                <option value="bottom">""",
    S["bottom_dm"],
    """</option>
              </select>
            </motion>
            <CmVideoPicker
              v-model="videoFilter"
              :options="videoOptions"
              :all-label=""",
    json.dumps(S["all_video"]),
    """
              @change="onFilterChange"
            />
          </motion>
        </motion>

        <template v-if="topTab === 'video'">
          <p v-if="loadError" class="cm-hint cm-hint--err">{{ loadError }}</p>
          <p v-else-if="loading && showListArea && !rows.length" class="cm-hint">""",
    S["loading"],
    """</p>
          <motion v-else-if="showDanmakuList" class="dm-table-wrap">
            <table class="dm-table">
              <thead>
                <tr>
                  <th class="dm-col-check"><input v-model="selectAll" type="checkbox" class="cm-check" :disabled="!canSelectRows" @change="onSelectAllChange" /></th>
                  <th class="dm-col-user">""",
    S["sender"],
    """</th>
                  <th class="dm-col-time dm-th-sortable">
                    <span class="dm-th-inner">
                      <span class="dm-th-label">""",
    S["play_time"],
    """</span>
                      <span class="dm-sort-arrows">
                        <button
                          type="button"
                          class="dm-sort-btn dm-sort-btn--up"
                          :class="{ on: sortField === 'play_time' && sortDir === 'asc' }"
                          :aria-label=""",
    json.dumps(S["sort_play_asc"]),
    """
                          @click="setSort('play_time', 'asc')"
                        />
                        <button
                          type="button"
                          class="dm-sort-btn dm-sort-btn--down"
                          :class="{ on: sortField === 'play_time' && sortDir === 'desc' }"
                          :aria-label=""",
    json.dumps(S["sort_play_desc"]),
    """
                          @click="setSort('play_time', 'desc')"
                        />
                      </span>
                    </span>
                  </th>
                  <th class="dm-col-content">""",
    S["content"],
    """</th>
                  <th class="dm-col-like dm-th-sortable">
                    <span class="dm-th-inner dm-th-inner--center">
                      <span class="dm-th-label">""",
    S["like"],
    """</span>
                      <span class="dm-sort-arrows">
                        <button
                          type="button"
                          class="dm-sort-btn dm-sort-btn--up"
                          :class="{ on: sortField === 'like_count' && sortDir === 'asc' }"
                          :aria-label=""",
    json.dumps(S["sort_like_asc"]),
    """
                          @click="setSort('like_count', 'asc')"
                        />
                        <button
                          type="button"
                          class="dm-sort-btn dm-sort-btn--down"
                          :class="{ on: sortField === 'like_count' && sortDir === 'desc' }"
                          :aria-label=""",
    json.dumps(S["sort_like_desc"]),
    """
                          @click="setSort('like_count', 'desc')"
                        />
                      </span>
                    </span>
                  </th>
                  <th class="dm-col-attr">""",
    S["attr"],
    """</th>
                  <th class="dm-col-sent">""",
    S["sent"],
    """</th>
                  <th class="dm-col-op">""",
    S["op"],
    """</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in displayRows" :key="row.id">
                  <td class="dm-col-check"><input v-model="selectedIds" type="checkbox" class="cm-check" :value="row.id" /></td>
                  <td class="dm-col-user"><span class="dm-user">{{ row.username || \"""",
    S["user"],
    """\" }}</span></td>
                  <td class="dm-col-time">{{ row.play_time }}</td>
                  <td class="dm-col-content dm-content-cell">
                    <p class="dm-content">{{ row.content }}</p>
                    <a v-if="videoHref(row)" class="dm-video-link" :href="videoHref(row)" target="_blank" rel="noopener noreferrer">\u89c6\u9891\uff1a{{ videoTitleShort(row) }}</a>
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
                  <td class="dm-col-attr">""",
    S["normal"],
    """</td>
                  <td class="dm-col-sent">
                    <div class="dm-sent-date">{{ sentDate(row) }}</div>
                    <div class="dm-sent-time">{{ sentClock(row) }}</div>
                  </td>
                  <td class="dm-col-op">
                    <div class="dm-row-menu">
                      <button
                        type="button"
                        class="dm-row-menu-btn"
                        :aria-label=""",
    json.dumps(S["op"]),
    """
                        @mouseenter="openRowMenu(row, $event)"
                        @mouseleave="scheduleCloseRowMenu"
                      >\u00b7\u00b7\u00b7</button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
            <p class="dm-foot-note">""",
    S["foot"],
    """</p>
          </motion>
          <motion v-else-if="showEmpty" class="cm-empty">
            <motion class="cm-empty-img"><img :src="emptyImg" alt="" width="360" height="auto" /></motion>
            <p class="cm-empty-txt">{{ emptyText }}</p>
          </motion>
        </template>
        <motion v-else class="cm-empty">
          <motion class="cm-empty-img"><img :src="emptyImg" alt="" width="360" height="auto" /></motion>
          <p class="cm-empty-txt">""",
    S["soon"],
    """</p>
        </motion>
      </motion>
    </motion>
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
          :title=""",
    json.dumps(S["soon_title"]),
    """
          @click="onRowProtect(rowMenuRow)"
        >""",
    S["protect"],
    """</button>
        <button
          type="button"
          class="dm-row-dropdown-item dm-row-dropdown-item--danger"
          @click="onRowDelete(rowMenuRow)"
        >""",
    S["del_btn"],
    """</button>
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
      if (!this.isMinibiliMode) return \"""",
    S["empty_hint"],
    """\";
      if (this.searchApplied) return \"""",
    S["empty_search"],
    """\";
      return \"""",
    S["empty"],
    """\";
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
    if (this.isMinibiliMode && getAccessToken()) {
      void this.loadVideos();
      void this.fetchList();
    }
  },
  beforeUnmount() {
    this.cancelCloseRowMenu();
  },
  methods: {
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
      if (t.length <= 18) return t || \"""",
    S["untitled"],
    """\";
      return t.slice(0, 18) + "\u2026";
    },
    toggleTypeFilter(type) {
      this.typeFilter = this.typeFilter === type ? "" : type;
      void this.fetchList();
    },
    async loadVideos() {
      try {
        const items = [];
        let cursor = "";
        for (let i = 0; i < 20; i++) {
          const res = await mbListMyVideos(cursor ? { cursor } : undefined);
          items.push(...(res.items || []));
          if (!res.next_cursor) break;
          cursor = res.next_cursor;
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
        this.loadError = (e && e.message) || \"""",
    S["load_fail"],
    """\";
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
      if (!s) return "\u2014";
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
      ElMessage.info(\"""",
    S["soon_title"],
    """\");
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
        ElMessage.error((e && e.message) || \"""",
    S["load_fail"],
    """\");
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
        title: \"""",
    S["del_title"],
    """\",
        message: () =>
          h("div", { class: "cm-del-msgbox-body" }, [
            h("p", { class: "cm-del-msgbox-msg" }, \"""",
    S["del_msg"],
    """\"),
            h("label", { class: "cm-del-msgbox-check" }, [
              h("input", {
                type: "checkbox",
                class: "cm-del-msgbox-check-input",
                onChange: (e) => {
                  blacklistSender = !!(e && e.target && e.target.checked);
                }
              }),
              h("span", { class: "cm-del-msgbox-check-label" }, \"""",
    S["blacklist_sender"],
    """\")
            ])
          ]),
        showCancelButton: true,
        confirmButtonText: \"""",
    S["ok"],
    """\",
        cancelButtonText: \"""",
    S["cancel"],
    """\",
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
        ElMessage.success(\"""",
    S["blacklist_done"],
    """\");
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
        ElMessage.success(\"""",
    S["del_done"],
    """\");
        await this.fetchList();
      } catch (e) {
        ElMessage.error((e && e.message) || \"""",
    S["del_fail"],
    """\");
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
        ElMessage.success(\"""",
    S["del_done"],
    """\");
        await this.fetchList();
      } catch (e) {
        if (ok > 0) await this.fetchList();
        ElMessage.error((e && e.message) || \"""",
    S["del_fail"],
    """\");
      } finally { this.deleting = false; }
    }
  }
};
</script>

<style lang="scss">
@import "./commentManage.scss";
@import "./danmakuManage.scss";
</style>
""",
)

out = out.replace('<motion ', '<div ').replace('</motion>', '</div>')
TARGET.write_text(out, encoding="utf-8")
print("OK", TARGET)
