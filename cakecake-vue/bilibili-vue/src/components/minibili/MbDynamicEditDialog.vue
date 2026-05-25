<template>
  <Teleport v-if="modelValue || leaveConfirmVisible" to="body">
    <div
      v-if="modelValue"
      class="mb-dyn-edit-overlay"
      role="presentation"
      @click.self="requestClose"
    >
      <div
        class="mb-dyn-edit-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="mb-dyn-edit-title"
        @click.stop
      >
        <header class="mb-dyn-edit-dialog__head">
          <h2 id="mb-dyn-edit-title" class="mb-dyn-edit-dialog__title">
            编辑动态
          </h2>
          <button
            type="button"
            class="mb-dyn-edit-dialog__close"
            aria-label="关闭"
            @click="requestClose"
          >
            ×
          </button>
        </header>

        <div class="mb-dyn-edit-dialog__body">
          <div class="mb-dyn-editor__compose">
            <input
              v-model="draftTitle"
              type="text"
              class="mb-dyn-editor__title"
              maxlength="20"
              placeholder="好的标题更容易获得支持，选填20字"
              @input="emitDirty"
            />
            <textarea
              v-model="draftContent"
              class="mb-dyn-editor__content"
              rows="4"
              maxlength="233"
              placeholder="有什么想和大家分享的？"
              @input="emitDirty"
            />
          </div>

          <div v-if="draftImageMode" class="mb-dyn-editor__media">
            <div class="mb-dyn-editor__media-grid">
              <div
                v-for="(item, ix) in draftImagePreviews"
                :key="'edit-img-' + ix"
                class="mb-dyn-editor__media-item"
              >
                <img :src="item.url" alt="" />
                <button
                  type="button"
                  class="mb-dyn-editor__media-remove"
                  aria-label="移除图片"
                  @click="removeDraftImage(ix)"
                >
                  ×
                </button>
              </div>
              <button
                v-if="draftImagePreviews.length < maxDraftImages"
                type="button"
                class="mb-dyn-editor__media-add"
                aria-label="添加图片"
                @click="openDraftImagePicker"
              >
                <span class="mb-dyn-editor__media-add-plus">+</span>
              </button>
            </div>
          </div>
          <input
            ref="draftImageInput"
            type="file"
            accept="image/jpeg,image/png,image/webp,image/gif"
            multiple
            class="mb-dyn-editor__file-input"
            @change="onDraftImagesSelected"
          />

          <div class="mb-dyn-editor__bar">
            <div class="mb-dyn-editor__tools" aria-label="插入内容">
              <button
                type="button"
                class="mb-dyn-editor__tool"
                :class="{ 'is-on': draftImageMode }"
                title="图片"
                aria-label="图片"
                @click="onDraftImageToolClick"
              >
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  aria-hidden="true"
                >
                  <rect
                    x="4"
                    y="5"
                    width="16"
                    height="14"
                    rx="2"
                    stroke="currentColor"
                    stroke-width="1.5"
                  />
                  <circle cx="9" cy="10" r="1.5" fill="currentColor" />
                  <path
                    d="M6 17l4.5-4 3 2.5L18 11"
                    stroke="currentColor"
                    stroke-width="1.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </button>
            </div>
            <div class="mb-dyn-editor__right">
              <span class="mb-dyn-editor__count">{{ draftCharCount }}/233</span>
              <button
                type="button"
                class="mb-dyn-editor__submit"
                :disabled="!canPublishDynamic || publishSubmitting"
                @click="onPublish"
              >
                {{ publishSubmitting ? "发布中…" : "发布" }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 自建离开确认层，避免 ElMessageBox 层级低于编辑弹窗 -->
    <div
      v-if="leaveConfirmVisible"
      class="mb-dyn-leave-overlay"
      role="presentation"
      @click.self="onLeaveStay"
    >
      <div
        class="mb-dyn-leave-dialog"
        role="alertdialog"
        aria-labelledby="mb-dyn-leave-title"
        aria-describedby="mb-dyn-leave-desc"
        @click.stop
      >
        <button
          type="button"
          class="mb-dyn-leave-dialog__close"
          aria-label="关闭"
          @click="onLeaveStay"
        >
          ×
        </button>
        <h3 id="mb-dyn-leave-title" class="mb-dyn-leave-dialog__title">
          {{ leaveTitle }}
        </h3>
        <p id="mb-dyn-leave-desc" class="mb-dyn-leave-dialog__msg">
          {{ leaveMsg }}
        </p>
        <div class="mb-dyn-leave-dialog__btns">
          <button
            type="button"
            class="mb-dyn-leave-dialog__btn mb-dyn-leave-dialog__btn--ghost"
            @click="onLeaveGo"
          >
            直接离开
          </button>
          <button
            type="button"
            class="mb-dyn-leave-dialog__btn mb-dyn-leave-dialog__btn--primary"
            @click="onLeaveStay"
          >
            继续修改
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script>
import { ElMessage } from "element-plus";
import { mbPutMyUserDynamic } from "@/api/minibili";
import { clearStuckPageOverlays } from "@/utils/clearPageOverlays";

const LEAVE_TITLE = "确认取消修改？";
const LEAVE_MSG = "离开后此次编辑无法保存";

export default {
  name: "MbDynamicEditDialog",
  props: {
    modelValue: { type: Boolean, default: false },
    dynamicId: { type: Number, default: 0 },
    initialTitle: { type: String, default: "" },
    initialContent: { type: String, default: "" },
    initialImages: { type: Array, default: () => [] }
  },
  emits: ["update:modelValue", "published", "dirty-change"],
  data() {
    return {
      leaveTitle: LEAVE_TITLE,
      leaveMsg: LEAVE_MSG,
      draftTitle: "",
      draftContent: "",
      draftImageMode: false,
      draftImageFiles: [],
      draftImagePreviews: [],
      maxDraftImages: 9,
      publishSubmitting: false,
      baselineKey: "",
      leaveConfirmVisible: false,
      _leavePromise: null
    };
  },
  computed: {
    draftCharCount() {
      return (
        String(this.draftTitle || "").length +
        String(this.draftContent || "").length
      );
    },
    canPublishDynamic() {
      return (
        !!String(this.draftTitle || "").trim() ||
        !!String(this.draftContent || "").trim() ||
        this.draftImagePreviews.length > 0
      );
    },
    isDirty() {
      return this.baselineKey !== this.currentSnapshotKey();
    }
  },
  watch: {
    modelValue(open) {
      if (open) {
        this.bootstrapFromProps();
      } else {
        this.clearDraftImages();
        this.leaveConfirmVisible = false;
        this._leavePromise = null;
      }
    },
    initialTitle() {
      if (this.modelValue) this.bootstrapFromProps();
    },
    initialContent() {
      if (this.modelValue) this.bootstrapFromProps();
    },
    initialImages() {
      if (this.modelValue) this.bootstrapFromProps();
    },
    isDirty(v) {
      this.$emit("dirty-change", v);
    }
  },
  methods: {
    currentSnapshotKey() {
      const keep = this.draftImagePreviews.map(p =>
        p.file ? `f:${p.file.name}:${p.file.size}` : `u:${p.url}`
      );
      return [
        String(this.draftTitle || "").trim(),
        String(this.draftContent || "").trim(),
        keep.join("|")
      ].join("\n");
    },
    bootstrapFromProps() {
      this.clearDraftImages();
      this.draftTitle = String(this.initialTitle || "");
      this.draftContent = String(this.initialContent || "");
      const imgs = Array.isArray(this.initialImages) ? this.initialImages : [];
      for (const u of imgs) {
        const url = String(u || "").trim();
        if (!url) continue;
        this.draftImagePreviews.push({ url, file: null });
      }
      this.draftImageMode = this.draftImagePreviews.length > 0;
      this.draftImageFiles = [];
      this.baselineKey = this.currentSnapshotKey();
      this.emitDirty();
    },
    emitDirty() {
      this.$emit("dirty-change", this.isDirty);
    },
    dismissLeaveConfirm(action) {
      this.leaveConfirmVisible = false;
      const p = this._leavePromise;
      this._leavePromise = null;
      if (p) {
        p.reject(action);
      }
    },
    promptLeave() {
      if (this.leaveConfirmVisible) {
        return Promise.reject("close");
      }
      return new Promise((resolve, reject) => {
        this._leavePromise = { resolve, reject };
        this.leaveConfirmVisible = true;
      });
    },
    onLeaveStay() {
      this.dismissLeaveConfirm("close");
    },
    onLeaveGo() {
      this.dismissLeaveConfirm("cancel");
    },
    async requestClose() {
      if (!this.isDirty) {
        this.$emit("update:modelValue", false);
        return;
      }
      try {
        await this.promptLeave();
      } catch (action) {
        if (action === "cancel") {
          this.$emit("update:modelValue", false);
        }
      }
    },
    onDraftImageToolClick() {
      if (!this.draftImageMode) {
        this.draftImageMode = true;
        this.openDraftImagePicker();
        return;
      }
      this.openDraftImagePicker();
    },
    openDraftImagePicker() {
      const el = this.$refs.draftImageInput;
      if (el) el.click();
    },
    onDraftImagesSelected(ev) {
      const input = ev && ev.target;
      const picked = input && input.files ? Array.from(input.files) : [];
      if (!picked.length) return;
      const remain = this.maxDraftImages - this.draftImagePreviews.length;
      if (remain <= 0) {
        ElMessage.warning(`最多上传 ${this.maxDraftImages} 张图片`);
        return;
      }
      const slice = picked.slice(0, remain);
      for (const file of slice) {
        if (!file.type.startsWith("image/")) continue;
        const url = URL.createObjectURL(file);
        this.draftImageFiles.push(file);
        this.draftImagePreviews.push({ url, file });
      }
      this.draftImageMode = true;
      this.emitDirty();
      if (input) input.value = "";
    },
    removeDraftImage(ix) {
      const i = Number(ix);
      if (!Number.isFinite(i) || i < 0) return;
      const prev = this.draftImagePreviews[i];
      if (prev && prev.file && prev.url) {
        try {
          URL.revokeObjectURL(prev.url);
        } catch {
          /* ignore */
        }
      }
      if (prev && prev.file) {
        const fi = this.draftImageFiles.indexOf(prev.file);
        if (fi >= 0) this.draftImageFiles.splice(fi, 1);
      }
      this.draftImagePreviews.splice(i, 1);
      if (!this.draftImagePreviews.length) {
        this.draftImageMode = false;
      }
      this.emitDirty();
    },
    clearDraftImages() {
      for (const p of this.draftImagePreviews) {
        if (p && p.file && p.url) {
          try {
            URL.revokeObjectURL(p.url);
          } catch {
            /* ignore */
          }
        }
      }
      this.draftImagePreviews = [];
      this.draftImageFiles = [];
      this.draftImageMode = false;
    },
    async onPublish() {
      if (!this.canPublishDynamic || this.publishSubmitting) return;
      const id = Number(this.dynamicId) || 0;
      if (!id) {
        ElMessage.error("无效的动态 ID");
        return;
      }
      const keepImages = this.draftImagePreviews
        .filter(p => !p.file)
        .map(p => String(p.url || "").trim())
        .filter(Boolean);
      const newFiles = this.draftImagePreviews
        .filter(p => p.file)
        .map(p => p.file);
      this.publishSubmitting = true;
      try {
        const item = await mbPutMyUserDynamic(id, {
          title: this.draftTitle,
          content: this.draftContent,
          keepImages,
          images: newFiles
        });
        this.baselineKey = this.currentSnapshotKey();
        this.emitDirty();
        ElMessage.success("动态已更新");
        this.$emit("published", item);
        this.$emit("update:modelValue", false);
      } catch (e) {
        ElMessage.error((e && e.message) || "发布失败");
      } finally {
        this.publishSubmitting = false;
      }
    },
    /** 路由离开时由父组件调用：true=允许离开，false=留在当前页 */
    confirmLeaveIfDirty() {
      if (!this.modelValue || !this.isDirty) {
        return Promise.resolve(true);
      }
      return this.promptLeave()
        .then(() => false)
        .catch(action => action === "cancel");
    }
  },
  beforeUnmount() {
    this.leaveConfirmVisible = false;
    this._leavePromise = null;
    this.clearDraftImages();
    clearStuckPageOverlays();
  }
};
</script>

<style lang="scss" scoped>
@import "@/pages/minibili/dynamics.scss";

.mb-dyn-edit-overlay {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 48px 16px 24px;
  background: rgba(0, 0, 0, 0.45);
  overflow-y: auto;
  box-sizing: border-box;
}

.mb-dyn-edit-dialog {
  width: 100%;
  max-width: 640px;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.18);
  box-sizing: border-box;
}

.mb-dyn-edit-dialog__head {
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  padding: 18px 48px 12px;
  border-bottom: 1px solid #f1f2f3;
}

.mb-dyn-edit-dialog__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #18191c;
}

.mb-dyn-edit-dialog__close {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  font-size: 22px;
  line-height: 1;
  color: #9499a0;
  cursor: pointer;
  border-radius: 6px;
  &:hover {
    color: #18191c;
    background: #f1f2f3;
  }
}

.mb-dyn-edit-dialog__body {
  padding: 12px 20px 18px;
}

.mb-dyn-editor {
  padding: 0;
}

.mb-dyn-leave-overlay {
  position: fixed;
  inset: 0;
  z-index: 3100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.35);
  box-sizing: border-box;
}

.mb-dyn-leave-dialog {
  position: relative;
  width: 100%;
  max-width: 360px;
  padding: 20px 20px 16px;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  box-sizing: border-box;
  text-align: center;
}

.mb-dyn-leave-dialog__close {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  font-size: 18px;
  line-height: 1;
  color: #9499a0;
  cursor: pointer;
  border-radius: 4px;
  &:hover {
    color: #18191c;
    background: #f1f2f3;
  }
}

.mb-dyn-leave-dialog__title {
  margin: 0 0 8px;
  font-size: 16px;
  font-weight: 600;
  color: #18191c;
}

.mb-dyn-leave-dialog__msg {
  margin: 0 0 18px;
  font-size: 13px;
  line-height: 1.5;
  color: #9499a0;
}

.mb-dyn-leave-dialog__btns {
  display: flex;
  justify-content: center;
  gap: 10px;
  flex-direction: row-reverse;
}

.mb-dyn-leave-dialog__btn {
  min-width: 88px;
  height: 32px;
  padding: 0 16px;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  border: 1px solid transparent;
  box-sizing: border-box;
}

.mb-dyn-leave-dialog__btn--ghost {
  border-color: #e3e5e7;
  background: #fff;
  color: #18191c;
  &:hover {
    border-color: #c9ccd0;
    background: #f6f7f8;
  }
}

.mb-dyn-leave-dialog__btn--primary {
  border-color: #00a1d6;
  background: #00a1d6;
  color: #fff;
  &:hover {
    background: #008ebd;
    border-color: #008ebd;
  }
}
</style>
