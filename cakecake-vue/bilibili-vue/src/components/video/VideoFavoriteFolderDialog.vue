<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="video-fav-folder-dialog-overlay"
      role="presentation"
      @click.self="onClose"
    >
      <div
        ref="dialogRoot"
        class="video-fav-folder-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="video-fav-folder-dialog-title"
        tabindex="-1"
        @click="onDialogClick"
        @keydown.esc="onEsc"
      >
        <button
          type="button"
          class="video-fav-folder-dialog__close"
          aria-label="关闭"
          :disabled="loading"
          @click="onClose"
        >
          ×
        </button>
        <h2 id="video-fav-folder-dialog-title" class="video-fav-folder-dialog__title">
          添加到收藏夹
        </h2>

        <div v-if="listLoading" class="video-fav-folder-dialog__loading">
          加载中…
        </div>
        <div v-else-if="loadError" class="video-fav-folder-dialog__loading">
          {{ loadError }}
        </div>
        <ul v-else class="video-fav-folder-dialog__list" role="listbox" aria-label="收藏夹">
          <li
            v-for="item in items"
            :key="item.id"
            class="video-fav-folder-dialog__row"
          >
            <button
              type="button"
              class="video-fav-folder-dialog__row-btn"
              role="option"
              :aria-selected="isChecked(item.id)"
              @click="toggleFolder(item)"
            >
              <span
                class="video-fav-folder-dialog__chk"
                :class="{ 'is-on': isChecked(item.id) }"
                aria-hidden="true"
              >
                <svg
                  v-if="isChecked(item.id)"
                  class="video-fav-folder-dialog__chk-ico"
                  viewBox="0 0 16 16"
                  fill="none"
                >
                  <path
                    d="M3.5 8.2 6.4 11 12.5 5"
                    stroke="currentColor"
                    stroke-width="1.8"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </span>
              <span class="video-fav-folder-dialog__name">{{ item.title }}</span>
              <span class="video-fav-folder-dialog__count">{{
                displayCountLabel(item)
              }}</span>
            </button>
          </li>

          <li class="video-fav-folder-dialog__row video-fav-folder-dialog__row--new">
            <button
              v-if="!createOpen"
              type="button"
              class="video-fav-folder-dialog__new-btn"
              @click.stop="openCreate"
            >
              <span class="video-fav-folder-dialog__new-plus" aria-hidden="true">+</span>
              <span>新建收藏夹</span>
            </button>
            <div
              v-else
              ref="createBlock"
              class="video-fav-folder-dialog__create"
              @click.stop
            >
              <div
                v-if="createTipVisible"
                class="video-fav-folder-dialog__create-tip"
                role="status"
              >
                <span>点击弹窗内其他区域或 ESC 键，取消新建收藏夹</span>
                <button
                  type="button"
                  class="video-fav-folder-dialog__create-tip-close"
                  aria-label="关闭提示"
                  @click="createTipVisible = false"
                >
                  ×
                </button>
              </div>
              <div class="video-fav-folder-dialog__create-row">
                <input
                  ref="createInput"
                  v-model.trim="createTitle"
                  type="text"
                  class="video-fav-folder-dialog__create-input"
                  maxlength="15"
                  placeholder="最多可输入15个字"
                  :disabled="createSaving"
                  @keydown.enter.prevent="submitCreate"
                />
                <button
                  type="button"
                  class="video-fav-folder-dialog__create-submit"
                  :disabled="createSaving || !createTitle"
                  @click="submitCreate"
                >
                  {{ createSaving ? "新建中…" : "新建" }}
                </button>
              </div>
            </div>
          </li>
        </ul>

        <div class="video-fav-folder-dialog__foot">
          <button
            type="button"
            class="video-fav-folder-dialog__confirm"
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
</template>

<script>
import { ElMessage } from "element-plus";
import {
  mbCreateFavoriteFolder,
  mbGetVideoFavoritePicker
} from "@/api/minibili";

const FOLDER_CAPACITY = 999;

function sameIdSet(a, b) {
  const sa = [...a].sort((x, y) => x - y);
  const sb = [...b].sort((x, y) => x - y);
  if (sa.length !== sb.length) return false;
  return sa.every((v, i) => v === sb[i]);
}

function mapPickerItem(row) {
  return {
    id: Number(row.id),
    title: String(row.title || ""),
    is_default: !!row.is_default,
    base_count: Number(row.video_count) || 0,
    initial_selected: !!row.selected
  };
}

export default {
  name: "VideoFavoriteFolderDialog",
  props: {
    modelValue: { type: Boolean, default: false },
    videoId: { type: Number, default: null },
    loading: { type: Boolean, default: false }
  },
  emits: ["update:modelValue", "confirm", "cancel"],
  data() {
    return {
      listLoading: false,
      loadError: "",
      items: [],
      checkedIds: [],
      initialCheckedIds: [],
      createOpen: false,
      createTitle: "",
      createSaving: false,
      createTipVisible: true
    };
  },
  computed: {
    canConfirm() {
      return !sameIdSet(this.checkedIds, this.initialCheckedIds);
    }
  },
  watch: {
    modelValue(open) {
      if (open) {
        this.resetCreate();
        this.loadPicker();
        this.$nextTick(() => {
          this.$refs.dialogRoot?.focus();
        });
      } else {
        this.resetCreate();
      }
    },
    videoId() {
      if (this.modelValue) {
        this.loadPicker();
      }
    }
  },
  methods: {
    displayCountLabel(item) {
      const delta =
        (this.isChecked(item.id) ? 1 : 0) - (item.initial_selected ? 1 : 0);
      const n = Math.max(0, item.base_count + delta);
      if (item.is_default) {
        return String(n);
      }
      return `${n}/${FOLDER_CAPACITY}`;
    },
    isChecked(id) {
      return this.checkedIds.includes(Number(id));
    },
    toggleFolder(item) {
      const n = Number(item.id);
      const willCheck = !this.isChecked(n);
      if (willCheck) {
        const delta =
          1 - (item.initial_selected ? 1 : 0);
        const next = item.base_count + delta;
        if (!item.is_default && next > FOLDER_CAPACITY) {
          ElMessage.warning("该收藏夹已满");
          return;
        }
        this.checkedIds = [...this.checkedIds, n];
      } else {
        this.checkedIds = this.checkedIds.filter((x) => x !== n);
      }
    },
    openCreate() {
      this.createOpen = true;
      this.createTitle = "";
      this.createTipVisible = true;
      this.$nextTick(() => {
        this.$refs.createInput?.focus();
      });
    },
    cancelCreate() {
      this.createOpen = false;
      this.createTitle = "";
      this.createSaving = false;
    },
    resetCreate() {
      this.createOpen = false;
      this.createTitle = "";
      this.createSaving = false;
      this.createTipVisible = true;
    },
    onDialogClick(e) {
      if (!this.createOpen) return;
      const block = this.$refs.createBlock;
      if (block && block.contains(e.target)) return;
      this.cancelCreate();
    },
    onEsc() {
      if (!this.modelValue || this.loading) return;
      if (this.createOpen) {
        this.cancelCreate();
        return;
      }
      this.onClose();
    },
    async submitCreate() {
      const title = String(this.createTitle || "").trim();
      if (!title || this.createSaving) return;
      if ([...title].length > 15) {
        ElMessage.warning("名称最多 15 个字");
        return;
      }
      this.createSaving = true;
      try {
        const row = await mbCreateFavoriteFolder({
          title,
          is_public: true
        });
        const item = mapPickerItem({
          id: row.id,
          title: row.title,
          is_default: row.is_default,
          video_count: row.video_count,
          selected: false
        });
        this.items = [...this.items, item];
        if (!this.isChecked(item.id)) {
          this.checkedIds = [...this.checkedIds, item.id];
        }
        this.cancelCreate();
        ElMessage.success("收藏夹已创建");
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "创建失败";
        ElMessage.error(String(msg));
      } finally {
        this.createSaving = false;
      }
    },
    async loadPicker() {
      if (this.videoId == null) {
        this.loadError = "视频无效";
        return;
      }
      this.listLoading = true;
      this.loadError = "";
      this.resetCreate();
      try {
        const res = await mbGetVideoFavoritePicker(this.videoId);
        this.items = (Array.isArray(res.items) ? res.items : []).map(
          mapPickerItem
        );
        const selected = this.items
          .filter((row) => row.initial_selected)
          .map((row) => row.id);
        let checked = [...selected];
        if (!checked.length) {
          const def = this.items.find((row) => row.is_default);
          if (def) {
            checked = [def.id];
          }
        }
        this.checkedIds = checked;
        this.initialCheckedIds = [...selected];
      } catch (e) {
        this.items = [];
        this.checkedIds = [];
        this.initialCheckedIds = [];
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "加载收藏夹失败";
        this.loadError = String(msg);
        ElMessage.error(this.loadError);
      } finally {
        this.listLoading = false;
      }
    },
    onClose() {
      if (this.loading) return;
      this.$emit("update:modelValue", false);
      this.$emit("cancel");
    },
    onConfirm() {
      if (!this.canConfirm || this.loading || this.listLoading) return;
      this.$emit("confirm", [...this.checkedIds]);
    }
  }
};
</script>

<style lang="scss" scoped>
$vf-blue: #00aeec;
$vf-text: #18191c;
$vf-muted: #9499a0;
$vf-border: #e3e5e7;

.video-fav-folder-dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: 10060;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  box-sizing: border-box;
  background: rgba(0, 0, 0, 0.45);
}

.video-fav-folder-dialog {
  position: relative;
  width: 100%;
  max-width: 480px;
  padding: 20px 0 16px;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.18);
  box-sizing: border-box;
  outline: none;
}

.video-fav-folder-dialog__close {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 1;
  width: 32px;
  height: 32px;
  border: none;
  padding: 0;
  border-radius: 6px;
  background: transparent;
  color: $vf-muted;
  font-size: 22px;
  line-height: 1;
  cursor: pointer;

  &:hover:not(:disabled) {
    color: $vf-text;
    background: rgba(0, 0, 0, 0.05);
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.video-fav-folder-dialog__title {
  margin: 0 0 8px;
  padding: 0 48px;
  text-align: center;
  font-size: 16px;
  font-weight: 600;
  color: $vf-text;
  line-height: 1.4;
}

.video-fav-folder-dialog__loading {
  min-height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  color: $vf-muted;
}

.video-fav-folder-dialog__list {
  list-style: none;
  margin: 0;
  padding: 4px 0;
  max-height: 360px;
  overflow-y: auto;
  border-top: 1px solid $vf-border;
  border-bottom: 1px solid $vf-border;
}

.video-fav-folder-dialog__row {
  margin: 0;
  padding: 0;
}

.video-fav-folder-dialog__row-btn {
  display: flex;
  align-items: center;
  width: 100%;
  min-height: 48px;
  padding: 10px 20px;
  border: none;
  background: transparent;
  cursor: pointer;
  text-align: left;
  box-sizing: border-box;
  transition: background 0.12s ease;

  &:hover {
    background: #f6f7f8;
  }
}

.video-fav-folder-dialog__chk {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  margin-right: 12px;
  border: 1px solid #ccd0d7;
  border-radius: 4px;
  background: #fff;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  transition:
    border-color 0.12s ease,
    background 0.12s ease;

  &.is-on {
    border-color: $vf-blue;
    background: $vf-blue;
  }
}

.video-fav-folder-dialog__chk-ico {
  width: 14px;
  height: 14px;
}

.video-fav-folder-dialog__name {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  color: $vf-text;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.video-fav-folder-dialog__count {
  flex-shrink: 0;
  margin-left: 12px;
  font-size: 13px;
  color: $vf-muted;
  line-height: 1.4;
}

.video-fav-folder-dialog__row--new {
  border-top: 1px solid $vf-border;
}

.video-fav-folder-dialog__new-btn {
  display: flex;
  align-items: center;
  width: 100%;
  min-height: 48px;
  padding: 10px 20px;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 14px;
  color: $vf-text;
  text-align: left;
  box-sizing: border-box;

  &:hover {
    background: #f6f7f8;
  }
}

.video-fav-folder-dialog__new-plus {
  flex-shrink: 0;
  width: 18px;
  margin-right: 12px;
  font-size: 20px;
  font-weight: 300;
  line-height: 1;
  color: $vf-muted;
  text-align: center;
}

.video-fav-folder-dialog__create {
  position: relative;
  padding: 12px 20px 14px;
}

.video-fav-folder-dialog__create-tip {
  position: absolute;
  left: 20px;
  bottom: calc(100% + 6px);
  display: flex;
  align-items: flex-start;
  gap: 6px;
  width: max-content;
  max-width: min(300px, calc(100% - 40px));
  padding: 8px 10px;
  border-radius: 6px;
  background: $vf-blue;
  color: #fff;
  font-size: 12px;
  line-height: 1.4;
  box-shadow: 0 4px 12px rgba(0, 174, 236, 0.35);
  z-index: 2;

  > span {
    flex: 1;
    min-width: 0;
  }

  &::after {
    content: "";
    position: absolute;
    left: 24px;
    bottom: -6px;
    border: 6px solid transparent;
    border-top-color: $vf-blue;
  }
}

.video-fav-folder-dialog__create-tip-close {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  margin: 0;
  padding: 0;
  border: none;
  background: transparent;
  color: #fff;
  font-size: 16px;
  line-height: 1;
  cursor: pointer;
  opacity: 0.85;

  &:hover {
    opacity: 1;
  }
}

.video-fav-folder-dialog__create-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.video-fav-folder-dialog__create-input {
  flex: 1;
  min-width: 0;
  height: 36px;
  padding: 0 12px;
  border: 1px solid $vf-blue;
  border-radius: 6px;
  font-size: 14px;
  color: $vf-text;
  outline: none;
  box-sizing: border-box;

  &::placeholder {
    color: #c9ccd0;
  }

  &:focus {
    box-shadow: 0 0 0 2px rgba(0, 174, 236, 0.15);
  }

  &:disabled {
    background: #f6f7f8;
    cursor: not-allowed;
  }
}

.video-fav-folder-dialog__create-submit {
  flex-shrink: 0;
  height: 36px;
  padding: 0 16px;
  border: none;
  border-radius: 6px;
  background: rgba(0, 174, 236, 0.12);
  color: $vf-blue;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.12s ease;

  &:hover:not(:disabled) {
    background: rgba(0, 174, 236, 0.2);
  }

  &:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
}

.video-fav-folder-dialog__foot {
  padding: 16px 20px 0;
}

.video-fav-folder-dialog__confirm {
  display: block;
  width: 100%;
  height: 40px;
  border: none;
  border-radius: 6px;
  background: #e3e5e7;
  color: #fff;
  font-size: 15px;
  font-weight: 500;
  cursor: not-allowed;
  transition: background 0.15s ease;

  &.is-active {
    background: $vf-blue;
    cursor: pointer;

    &:hover:not(:disabled) {
      opacity: 0.92;
    }
  }

  &:disabled {
    opacity: 1;
  }
}
</style>
