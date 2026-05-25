# -*- coding: utf-8 -*-
"""Regenerate commentManage.vue with UTF-8 Chinese (unicode escapes only in source)."""
from pathlib import Path

TARGET = Path(__file__).resolve().parents[1] / "cakecake-vue/bilibili-vue/src/pages/upload/commentManage.vue"

S = {
    "tab_visible": "\u7528\u6237\u53ef\u89c1\u8bc4\u8bba",
    "tab_pending": "\u5f85\u7cbe\u9009\u8bc4\u8bba",
    "sub_video": "\u89c6\u9891\u8bc4\u8bba",
    "sub_article": "\u4e13\u680f\u8bc4\u8bba",
    "sub_dynamic": "\u52a8\u6001\u8bc4\u8bba",
    "select_all": "\u5168\u9009",
    "report": "\u4e3e\u62a5",
    "pick": "\u7cbe\u9009",
    "ignore": "\u5ffd\u7565",
    "deleting": "\u5220\u9664\u4e2d\u2026",
    "delete": "\u5220\u9664",
    "search_ph": "\u641c\u7d22\u89c6\u9891\u8bc4\u8bba",
    "search_aria": "\u641c\u7d22",
    "all_videos": "\u5168\u90e8\u89c6\u9891",
    "pending_unprocessed": "\u672a\u5904\u7406",
    "pending_all": "\u5168\u90e8",
    "loading": "\u52a0\u8f7d\u4e2d\u2026",
    "user": "\u7528\u6237",
    "reply": "\u56de\u590d",
    "empty": "\u6682\u65e0\u8bc4\u8bba",
    "empty_pending": "\u8fd8\u6ca1\u6709\u8bc4\u8bba\u54e6~",
    "empty_hint": "\u8bf7\u5f00\u542f Mini-Bili \u6a21\u5f0f\u5e76\u767b\u5f55\u540e\u67e5\u770b",
    "foot_note": "\u4ec5\u5c55\u793a\u6700\u8fd1\u768450000\u6761\u8bc4\u8bba",
    "prev": "\u4e0a\u4e00\u9875",
    "next": "\u4e0b\u4e00\u9875",
    "sort_recent": "\u6700\u8fd1\u53d1\u5e03",
    "sort_earliest": "\u6700\u65e9\u53d1\u5e03",
    "sort_likes": "\u70b9\u8d5e\u6700\u591a",
    "sort_replies": "\u56de\u590d\u6700\u591a",
    "draft": "\u7a3f\u4ef6",
    "load_fail": "\u52a0\u8f7d\u5931\u8d25",
    "report_soon": "\u4e3e\u62a5\u529f\u80fd\u5373\u5c06\u5f00\u653e",
    "pick_soon": "\u7cbe\u9009\u529f\u80fd\u5373\u5c06\u5f00\u653e",
    "ignore_soon": "\u5ffd\u7565\u529f\u80fd\u5373\u5c06\u5f00\u653e",
    "del_reminder_title": "\u5220\u9664\u63d0\u9192",
    "del_reminder_msg": "\u5220\u9664\u540e\u65e0\u6cd5\u6062\u590d\uff0c\u786e\u8ba4\u5220\u9664\u9009\u4e2d\u7684\u8bc4\u8bba\u5417\uff1f",
    "confirm_ok": "\u786e\u5b9a",
    "cancel": "\u53d6\u6d88",
    "del_done": "\u5df2\u5220\u9664",
    "del_fail": "\u5220\u9664\u5931\u8d25",
    "ellipsis": "\u2026",
}

THUMB = (
    "M7 10v12M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.67 22H4"
    "a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2a3.13 3.13 0 0 1 3 3.88Z"
)

def vue(s: str) -> str:
    return s.replace("{", "{{").replace("}", "}}")

# fmt: off
content = vue(r"""
<template>
  <CreatorShell>
    <motion class="cm-wrap">
      <div class="cm-panel">
        <motion class="cm-head-row">
          <div class="cm-top-tabs">
            <button
              type="button"
              class="cm-top-tab"
              :class="{ on: primaryTab === 'visible' }"
              @click="setPrimaryTab('visible')"
            >""") + S["tab_visible"] + vue(r"""</button>
            <button
              type="button"
              class="cm-top-tab"
              :class="{ on: primaryTab === 'pending' }"
              @click="setPrimaryTab('pending')"
            >""") + S["tab_pending"] + vue(r"""</button>
          </div>
          <div v-if="primaryTab === 'visible'" class="cm-head-search">
            <div class="cm-search">
              <input
                v-model.trim="searchQ"
                type="search"
                class="cm-search-input"
                placeholder=""") + f'"{S["search_ph"]}"' + vue(r"""
                autocomplete="off"
                @keydown.enter.prevent="onSearch"
              />
              <button
                type="button"
                class="cm-search-btn"
                :aria-label="""") + f'"{S["search_aria"]}"' + vue(r"""
                @click="onSearch"
              />
            </div>
          </div>
        </motion>

        <motion class="cm-sub-row">
          <div class="cm-sub-tabs">
            <button
              type="button"
              class="cm-sub-tab"
              :class="{ on: mediaTab === 'video' }"
              @click="setMediaTab('video')"
            >""") + S["sub_video"] + vue(r"""</button>
            <button
              type="button"
              class="cm-sub-tab"
              :class="{ on: mediaTab === 'article' }"
              @click="setMediaTab('article')"
            >""") + S["sub_article"] + vue(r"""</button>
            <button
              v-if="primaryTab === 'pending'"
              type="button"
              class="cm-sub-tab"
              :class="{ on: mediaTab === 'dynamic' }"
              @click="setMediaTab('dynamic')"
            >""") + S["sub_dynamic"] + vue(r"""</button>
          </div>
          <div v-if="primaryTab === 'visible' && mediaTab === 'video'" class="cm-sub-filters">
            <CmVideoPicker
              v-model="videoFilter"
              :options="videoOptions"
              :all-label="""") + f'"{S["all_videos"]}"' + vue(r"""
              @change="onFilterChange"
            />
          </div>
          <motion v-else-if="primaryTab === 'pending'" class="cm-sub-filters">
            <select v-model="pendingStatus" class="cm-filter-select">
              <option value="unprocessed">""") + S["pending_unprocessed"] + vue(r"""</option>
              <option value="all">""") + S["pending_all"] + vue(r"""</option>
            </select>
            <CmVideoPicker
              v-model="videoFilter"
              :options="videoOptions"
              :all-label="""") + f'"{S["all_videos"]}"' + vue(r"""
            />
            <select v-model="pendingScope" class="cm-filter-select">
              <option value="all">""") + S["pending_all"] + vue(r"""</option>
            </select>
          </motion>
        </motion>

        <motion class="cm-action-row">
          <div class="cm-action-left">
            <label class="cm-check-all">
              <input
                v-model="selectAll"
                type="checkbox"
                class="cm-check"
                :disabled="!canSelectRows"
                @change="onSelectAllChange"
              />
              <span>""") + S["select_all"] + vue(r"""</span>
            </label>
            <template v-if="primaryTab === 'visible'">
              <button
                type="button"
                class="cm-tool-btn"
                :disabled="!selectedIds.length"
                @click="onReport"
              >""") + S["report"] + vue(r"""</button>
              <button
                type="button"
                class="cm-tool-btn cm-tool-btn--danger"
                :disabled="!selectedIds.length || deleting"
                @click="onBatchDelete"
              >
                {{ deleting ? """) + f'"{S["deleting"]}"' + vue(r""" : """) + f'"{S["delete"]}"' + vue(r""" }}
              </button>
            </template>
            <template v-else>
              <button
                type="button"
                class="cm-tool-btn"
                :disabled="!selectedIds.length"
                @click="onPendingPick"
              >""") + S["pick"] + vue(r"""</button>
              <button
                type="button"
                class="cm-tool-btn"
                :disabled="!selectedIds.length"
                @click="onPendingIgnore"
              >""") + S["ignore"] + vue(r"""</button>
              <button
                type="button"
                class="cm-tool-btn cm-tool-btn--danger"
                :disabled="!selectedIds.length || deleting"
                @click="onBatchDelete"
              >
                {{ deleting ? """) + f'"{S["deleting"]}"' + vue(r""" : """) + f'"{S["delete"]}"' + vue(r""" }}
              </button>
            </template>
          </div>
          <div class="cm-sort-row">
            <button
              v-for="opt in sortOptions"
              :key="opt.value"
              type="button"
              class="cm-sort-link"
              :class="{ on: sortKey === opt.value }"
              @click="setSort(opt.value)"
            >
              {{ opt.label }}
            </button>
          </motion>
        </motion>

        <p v-if="loadError" class="cm-hint cm-hint--err">{{ loadError }}</p>
        <p v-else-if="loading && showListArea && !rows.length" class="cm-hint">""") + S["loading"] + vue(r"""</p>

        <div v-else-if="showCommentList" class="cm-list">
          <div v-for="row in rows" :key="row.id" class="cm-row">
            <label class="cm-row-check">
              <input
                v-model="selectedIds"
                type="checkbox"
                class="cm-check"
                :value="row.id"
              />
            </label>
            <router-link
              v-if="userSpaceTo(row)"
              class="cm-row-avatar-link"
              :to="userSpaceTo(row)"
            >
              <img
                class="cm-row-avatar"
                :src="row.avatar_url || defaultAvatar"
                :alt="row.username"
                width="40"
                height="40"
              />
            </router-link>
            <img
              v-else
              class="cm-row-avatar"
              :src="row.avatar_url || defaultAvatar"
              :alt="row.username"
              width="40"
              height="40"
            />
            <div class="cm-row-body">
              <div class="cm-row-user-line">
                <router-link
                  v-if="userSpaceTo(row)"
                  class="cm-row-user cm-row-user--link"
                  :to="userSpaceTo(row)"
                >
                  {{ row.username || """) + f'"{S["user"]}"' + vue(r""" }}
                </router-link>
                <span v-else class="cm-row-user">{{ row.username || """) + f'"{S["user"]}"' + vue(r""" }}</span>
              </div>
              <a
                v-if="videoHref(row)"
                class="cm-row-content-link"
                :href="videoHref(row)"
                target="_blank"
                rel="noopener noreferrer"
              >
                <p class="cm-row-content">{{ row.content }}</p>
              </a>
              <p v-else class="cm-row-content">{{ row.content }}</p>
              <div class="cm-row-meta">
                <span class="cm-row-time">{{ row.created_at }}</span>
                <span class="cm-row-like" aria-hidden="true">
                  <svg
                    class="cm-row-like-ico"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <path d="""") + f'"{THUMB}"' + vue(r""" />
                  </svg>
                </span>
                <button type="button" class="cm-row-reply-btn" @click="openVideoTab(row)">
                  """) + S["reply"] + vue(r"""
                </button>
                <button
                  type="button"
                  class="cm-row-del-btn"
                  @click.stop="onRowDelete(row)"
                >
                  """) + S["delete"] + vue(r"""
                </button>
              </div>
            </div>
            <a
              v-if="videoHref(row)"
              class="cm-row-video"
              :href="videoHref(row)"
              target="_blank"
              rel="noopener noreferrer"
            >
              <img
                class="cm-row-video-cover"
                :src="row.video.cover_url || defaultCover"
                :alt="row.video.title"
              />
              <span class="cm-row-video-title">{{ row.video.title }}</span>
            </a>
          </div>
        </div>

        <div v-else-if="showEmpty" class="cm-empty">
          <div v-if="emptyUseImage" class="cm-empty-img">
            <img :src="articleEmptyImg" alt="" width="220" height="auto" />
          </motion>
          <p class="cm-empty-txt">{{ emptyText }}</p>
        </div>

        <div v-if="primaryTab === 'visible' && total > 0" class="cm-foot">
          <p class="cm-foot-note">""") + S["foot_note"] + vue(r"""</p>
          <div class="cm-pager">
            <button
              type="button"
              class="cm-page-btn"
              :disabled="page <= 1 || loading"
              @click="goPage(page - 1)"
            >""") + S["prev"] + vue(r"""</button>
            <button
              v-for="p in pageNums"
              :key="'p-' + p"
              type="button"
              class="cm-page-num"
              :class="{ on: p === page }"
              @click="goPage(p)"
            >
              {{ p }}
            </button>
            <button
              type="button"
              class="cm-page-btn"
              :disabled="page >= totalPages || loading"
              @click="goPage(page + 1)"
            >""") + S["next"] + vue(r"""</button>
            <span class="cm-page-summary">\u5171{{ totalPages }}\u9875/{{ total }}\u4e2a</span>
          </div>
        </div>
      </div>
    </motion>
  </CreatorShell>
</template>

<script>
import { ElMessage, ElMessageBox } from "element-plus";
import "@/styles/cm-del-msgbox.scss";
import CreatorShell from "@/components/creator/CreatorShell.vue";
import CmVideoPicker from "@/components/creator/CmVideoPicker.vue";
import defaultAvatar from "@/assets/akari.jpg";
import defaultCover from "@/assets/akari.jpg";
import articleEmptyImg from "@/assets/upload_manager/image_text/empty.9e92c422.png";
import {
  mbDeleteComment,
  mbListCreatorComments,
  mbListMyVideos
} from "@/api/minibili";
import { getAccessToken } from "@/utils/authTokens";
import { formatVideoBvid } from "@/utils/videoBvid";
import { minibiliUserSpaceRoute } from "@/utils/minibiliRoutes";

const isMinibiliMode =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

export default {
  name: "CreatorCommentManage",
  components: { CreatorShell, CmVideoPicker },
  data() {
    return {
      defaultAvatar,
      defaultCover,
      articleEmptyImg,
      primaryTab: "visible",
      mediaTab: "video",
      pendingStatus: "unprocessed",
      pendingScope: "all",
      loading: false,
      deleting: false,
      loadError: "",
      rows: [],
      page: 1,
      pageSize: 10,
      total: 0,
      totalPages: 0,
      sortKey: "recent",
      searchQ: "",
      searchApplied: "",
      videoFilter: "",
      videoOptions: [],
      selectedIds: [],
      selectAll: false,
      visibleSortOptions: [
        { value: "recent", label: """) + f'"{S["sort_recent"]}"' + vue(r""" },
        { value: "likes", label: """) + f'"{S["sort_likes"]}"' + vue(r""" },
        { value: "replies", label: """) + f'"{S["sort_replies"]}"' + vue(r""" }
      ],
      pendingSortOptions: [
        { value: "recent", label: """) + f'"{S["sort_recent"]}"' + vue(r""" },
        { value: "earliest", label: """) + f'"{S["sort_earliest"]}"' + vue(r""" }
      ]
    };
  },
  computed: {
    isMinibiliMode() {
      return isMinibiliMode;
    },
    sortOptions() {
      return this.primaryTab === "pending"
        ? this.pendingSortOptions
        : this.visibleSortOptions;
    },
    showListArea() {
      return this.primaryTab === "visible" && this.mediaTab === "video";
    },
    showCommentList() {
      return this.showListArea && this.rows.length > 0 && !this.loadError;
    },
    canSelectRows() {
      return this.showCommentList;
    },
    showEmpty() {
      if (this.loadError) return false;
      if (this.loading && this.showListArea) return false;
      if (!this.isMinibiliMode) return true;
      if (this.primaryTab === "pending") return true;
      if (this.mediaTab !== "video") return true;
      return !this.rows.length;
    },
    emptyUseImage() {
      if (!this.isMinibiliMode) return false;
      if (this.primaryTab === "pending") return this.mediaTab === "video";
      return this.mediaTab === "article";
    },
    emptyText() {
      if (!this.isMinibiliMode) return """) + f'"{S["empty_hint"]}"' + vue(r""";
      if (this.primaryTab === "pending") return """) + f'"{S["empty_pending"]}"' + vue(r""";
      if (this.mediaTab === "article") return """) + f'"{S["empty_pending"]}"' + vue(r""";
      if (this.mediaTab === "dynamic") return """) + f'"{S["empty_pending"]}"' + vue(r""";
      return """) + f'"{S["empty"]}"' + vue(r""";
    },
    pageNums() {
      const max = Math.min(this.totalPages, 7);
      if (max <= 0) return [];
      const start = Math.max(1, Math.min(this.page - 2, this.totalPages - max + 1));
      const out = [];
      for (let i = 0; i < max; i += 1) {
        out.push(start + i);
      }
      return out;
    }
  },
  watch: {
    selectedIds() {
      this.selectAll =
        this.canSelectRows &&
        this.selectedIds.length === this.rows.length &&
        this.rows.length > 0;
    },
    primaryTab() {
      if (this.primaryTab === "pending" && this.sortKey === "likes") {
        this.sortKey = "recent";
      }
      if (this.primaryTab === "pending" && this.sortKey === "replies") {
        this.sortKey = "recent";
      }
      if (this.primaryTab === "visible" && this.sortKey === "earliest") {
        this.sortKey = "recent";
      }
    }
  },
  mounted() {
    if (this.isMinibiliMode && getAccessToken()) {
      void this.loadVideos();
      void this.fetchList();
    }
  },
  methods: {
    setPrimaryTab(tab) {
      if (this.primaryTab === tab) return;
      this.primaryTab = tab;
      this.selectedIds = [];
      this.selectAll = false;
      this.loadError = "";
      if (tab === "visible" && this.mediaTab === "video") {
        void this.fetchList();
      } else {
        this.rows = [];
        this.total = 0;
        this.totalPages = 0;
      }
    },
    setMediaTab(tab) {
      if (this.mediaTab === tab) return;
      this.mediaTab = tab;
      this.selectedIds = [];
      this.selectAll = false;
      this.page = 1;
      if (this.primaryTab === "visible" && tab === "video") {
        void this.fetchList();
      } else {
        this.rows = [];
        this.total = 0;
        this.totalPages = 0;
      }
    },
    async confirmDeleteDialog() {
      await ElMessageBox.confirm(
        """) + f'"{S["del_reminder_msg"]}"' + vue(r""",
        """) + f'"{S["del_reminder_title"]}"' + vue(r""",
        {
          confirmButtonText: """) + f'"{S["confirm_ok"]}"' + vue(r""",
          cancelButtonText: """) + f'"{S["cancel"]}"' + vue(r""",
          customClass: "cm-del-msgbox",
          confirmButtonClass: "cm-del-msgbox__ok",
          cancelButtonClass: "cm-del-msgbox__cancel",
          showClose: true,
          distinguishCancelAndClose: true
        }
      );
    },
    userSpaceTo(row) {
      if (!row || !row.user_id) return null;
      return minibiliUserSpaceRoute(row.user_id);
    },
    videoLink(row) {
      if (!row || !row.video_id) return null;
      return {
        name: "video",
        params: { aid: formatVideoBvid(row.video_id) },
        query: { mb_cid: String(row.id) }
      };
    },
    videoHref(row) {
      const to = this.videoLink(row);
      if (!to) return "";
      return this.$router.resolve(to).href;
    },
    openVideoTab(row) {
      const href = this.videoHref(row);
      if (href) window.open(href, "_blank", "noopener,noreferrer");
    },
    async loadVideos() {
      try {
        const items = [];
        let cursor = "";
        for (let i = 0; i < 20; i += 1) {
          const res = await mbListMyVideos(cursor ? { cursor } : undefined);
          items.push(...(res.items || []));
          if (!res.next_cursor) break;
          cursor = res.next_cursor;
        }
        this.videoOptions = items;
      } catch {
        this.videoOptions = [];
      }
    },
    async fetchList() {
      if (!this.showListArea || !this.isMinibiliMode || !getAccessToken()) return;
      this.loading = true;
      this.loadError = "";
      try {
        const params = {
          page: this.page,
          page_size: this.pageSize,
          sort: this.sortKey
        };
        if (this.searchApplied) params.q = this.searchApplied;
        if (this.videoFilter) params.video_id = Number(this.videoFilter);
        const res = await mbListCreatorComments(params);
        this.rows = res.items || [];
        this.total = Number(res.total) || 0;
        this.totalPages = Number(res.total_pages) || 0;
        this.selectedIds = [];
        this.selectAll = false;
      } catch (e) {
        this.loadError = (e && e.message) || """) + f'"{S["load_fail"]}"' + vue(r""";
        this.rows = [];
      } finally {
        this.loading = false;
      }
    },
    onSearch() {
      this.searchApplied = this.searchQ;
      this.page = 1;
      void this.fetchList();
    },
    onFilterChange() {
      this.page = 1;
      void this.fetchList();
    },
    setSort(key) {
      if (this.sortKey === key) return;
      this.sortKey = key;
      if (this.showListArea) {
        this.page = 1;
        void this.fetchList();
      }
    },
    goPage(p) {
      if (p < 1 || p > this.totalPages || p === this.page) return;
      this.page = p;
      void this.fetchList();
    },
    onSelectAllChange() {
      if (this.selectAll) {
        this.selectedIds = this.rows.map((r) => r.id);
      } else {
        this.selectedIds = [];
      }
    },
    onReport() {
      ElMessage.info(""") + f'"{S["report_soon"]}"' + vue(r""");
    },
    onPendingPick() {
      ElMessage.info(""") + f'"{S["pick_soon"]}"' + vue(r""");
    },
    onPendingIgnore() {
      ElMessage.info(""") + f'"{S["ignore_soon"]}"' + vue(r""");
    },
    async onRowDelete(row) {
      if (!row || !row.id) return;
      try {
        await this.confirmDeleteDialog();
      } catch {
        return;
      }
      this.deleting = true;
      try {
        await mbDeleteComment(row.id);
        ElMessage.success(""") + f'"{S["del_done"]}"' + vue(r""");
        if (this.rows.length <= 1 && this.page > 1) {
          this.page -= 1;
        }
        await this.fetchList();
      } catch (e) {
        ElMessage.error((e && e.message) || """) + f'"{S["del_fail"]}"' + vue(r""");
      } finally {
        this.deleting = false;
      }
    },
    async onBatchDelete() {
      if (!this.selectedIds.length) return;
      try {
        await this.confirmDeleteDialog();
      } catch {
        return;
      }
      this.deleting = true;
      const ids = [...this.selectedIds];
      try {
        for (const id of ids) {
          await mbDeleteComment(id);
        }
        ElMessage.success(""") + f'"{S["del_done"]}"' + vue(r""");
        if (this.rows.length <= ids.length && this.page > 1) {
          this.page -= 1;
        }
        await this.fetchList();
      } catch (e) {
        ElMessage.error((e && e.message) || """) + f'"{S["del_fail"]}"' + vue(r""");
      } finally {
        this.deleting = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "./commentManage.scss";
</style>
""")

# Fix accidental motion.* tags from template builder
content = content.replace("<motion ", "<motion ").replace("<motion ", "<div ")
content = content.replace("</motion>", "</motion>")
import re
content = re.sub(r"</?motion\b", lambda m: m.group(0).replace("motion", "motion"), content)
content = content.replace("<motion ", "<div ").replace("</motion>", "</div>")

TARGET.write_text(content, encoding="utf-8")
print("OK", TARGET)
