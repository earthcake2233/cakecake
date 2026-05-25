<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="mb-cvft-overlay"
      role="presentation"
      @click.self="onClose"
    >
      <div
        ref="dialogRoot"
        class="mb-cvft-dialog"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        tabindex="-1"
        @keydown.esc="onEsc"
      >
        <button
          type="button"
          class="mb-cvft-dialog__close"
          aria-label="关闭"
          :disabled="loading"
          @click="onClose"
        >
          ×
        </button>
        <h2 :id="titleId" class="mb-cvft-dialog__title">{{ dialogTitle }}</h2>

        <div v-if="listLoading" class="mb-cvft-dialog__loading">加载中…</div>
        <div v-else class="mb-cvft-dialog__body">
          <div class="mb-cvft-dialog__new-zone">
            <button
              type="button"
              class="mb-cvft-dialog__new-dashed"
              :disabled="folderCreateSaving"
              @click.stop="openFolderCreate"
            >
              <span class="mb-cvft-dialog__new-plus" aria-hidden="true">+</span>
              <span>新建收藏夹</span>
            </button>
          </div>

          <ul class="mb-cvft-dialog__list" role="listbox">
            <li v-for="item in listItems" :key="item.id" class="mb-cvft-dialog__row">
              <button
                type="button"
                class="mb-cvft-dialog__row-btn"
                role="option"
                :aria-selected="selectedId === item.id"
                @click="selectedId = item.id"
              >
                <span
                  class="mb-cvft-dialog__radio"
                  :class="{ 'is-on': selectedId === item.id }"
                  aria-hidden="true"
                />
                <span class="mb-cvft-dialog__name">{{ item.title }}</span>
                <span class="mb-cvft-dialog__count">{{ item.count_label }}</span>
              </button>
            </li>
          </ul>
        </div>

        <div class="mb-cvft-dialog__foot">
          <button
            type="button"
            class="mb-cvft-dialog__confirm"
            :class="{ 'is-active': canConfirm }"
            :disabled="!canConfirm || loading || listLoading"
            @click="onConfirm"
          >
            确定
          </button>
        </div>
      </div>
    </div>
  </Teleport>

  <MbCollectFolderCreateDialog
    v-model="folderCreateOpen"
    :loading="folderCreateSaving"
    @submit="onFolderCreateSubmit"
  />
</template>

<script>
import { ElMessage } from "element-plus";
import {
  mbCreateFavoriteFolder,
  mbListMyFavoriteFolders
} from "@/api/minibili";
import MbCollectFolderCreateDialog from "@/components/minibili/MbCollectFolderCreateDialog.vue";
import { showMbDarkToast } from "@/utils/mbToast";

const FOLDER_CAPACITY = 999;

export default {
  name: "MbCollectVideoFolderTransferDialog",
  components: { MbCollectFolderCreateDialog },
  props: {
    modelValue: { type: Boolean, default: false },
    mode: {
      type: String,
      default: "copy",
      validator: (v) => v === "copy" || v === "move"
    },
    fromFolderId: { type: Number, default: null },
    videoCount: { type: Number, default: 1 },
    loading: { type: Boolean, default: false }
  },
  emits: ["update:modelValue", "confirm", "cancel", "folder-created"],
  data() {
    return {
      titleId: "mb-cvft-dialog-title",
      listLoading: false,
      items: [],
      selectedId: null,
      folderCreateOpen: false,
      folderCreateSaving: false
    };
  },
  computed: {
    dialogTitle() {
      const n = Math.max(1, Number(this.videoCount) || 1);
      return this.mode === "move"
        ? `将${n}个视频移动至`
        : `将${n}个视频复制至`;
    },
    listItems() {
      const from = Number(this.fromFolderId);
      if (this.mode !== "move" || !from) {
        return this.items;
      }
      return this.items.filter((row) => Number(row.id) !== from);
    },
    canConfirm() {
      return this.selectedId != null && this.selectedId > 0;
    }
  },
  watch: {
    modelValue(open) {
      if (open) {
        this.folderCreateOpen = false;
        this.folderCreateSaving = false;
        this.loadFolders();
        this.$nextTick(() => {
          this.$refs.dialogRoot?.focus();
        });
      } else {
        this.folderCreateOpen = false;
        this.folderCreateSaving = false;
        this.selectedId = null;
      }
    },
    mode() {
      if (this.modelValue) {
        this.loadFolders();
      }
    }
  },
  methods: {
    folderCountLabel(row) {
      const n = Number(row.video_count) || 0;
      if (row.is_default) {
        return String(n);
      }
      return `${n}/${FOLDER_CAPACITY}`;
    },
    mapFolderRow(row) {
      return {
        id: Number(row.id),
        title: String(row.title || ""),
        is_default: !!row.is_default,
        video_count: Number(row.video_count) || 0,
        count_label: this.folderCountLabel(row)
      };
    },
    async loadFolders() {
      this.listLoading = true;
      try {
        const res = await mbListMyFavoriteFolders();
        this.items = (Array.isArray(res.items) ? res.items : []).map((row) =>
          this.mapFolderRow(row)
        );
        this.selectedId = this.listItems.length ? this.listItems[0].id : null;
      } catch (e) {
        this.items = [];
        this.selectedId = null;
        ElMessage.error((e && e.message) || "加载收藏夹失败");
      } finally {
        this.listLoading = false;
      }
    },
    openFolderCreate() {
      if (this.folderCreateSaving) return;
      this.folderCreateOpen = true;
    },
    async onFolderCreateSubmit(payload) {
      if (this.folderCreateSaving) return;
      this.folderCreateSaving = true;
      try {
        const row = await mbCreateFavoriteFolder({
          title: payload.title,
          description: payload.description || "",
          is_public: payload.is_public,
          cover: payload.cover || null
        });
        const item = this.mapFolderRow(row);
        this.items = [...this.items, item];
        this.selectedId = item.id;
        this.folderCreateOpen = false;
        this.$emit("folder-created", item);
        showMbDarkToast("收藏夹创建成功");
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "创建失败";
        ElMessage.error(String(msg));
      } finally {
        this.folderCreateSaving = false;
      }
    },
    onEsc() {
      if (!this.modelValue || this.loading || this.folderCreateOpen) return;
      this.onClose();
    },
    onClose() {
      if (this.loading || this.folderCreateSaving) return;
      this.$emit("update:modelValue", false);
      this.$emit("cancel");
    },
    onConfirm() {
      if (!this.canConfirm || this.loading || this.listLoading) return;
      this.$emit("confirm", Number(this.selectedId));
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/styles/mb-collect-video-folder-transfer-dialog.scss";
</style>
