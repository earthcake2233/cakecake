<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="mb-collect-folder-dialog-overlay"
      role="presentation"
      @click.self="onCancel"
    >
      <div
        class="mb-collect-folder-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="mb-collect-folder-dialog-title"
        @click.stop
      >
        <button
          type="button"
          class="mb-collect-folder-dialog__close"
          aria-label="关闭"
          :disabled="loading"
          @click="onCancel"
        >
          ×
        </button>
        <h2 id="mb-collect-folder-dialog-title" class="mb-collect-folder-dialog__title">
          {{ dialogTitle }}
        </h2>

        <button
          type="button"
          class="mb-collect-folder-dialog__cover"
          :class="{ 'has-preview': !!coverPreviewUrl }"
          :disabled="loading"
          aria-label="上传收藏夹封面"
          @click="onPickCover"
        >
          <img
            v-if="coverPreviewUrl"
            class="mb-collect-folder-dialog__cover-img"
            :src="coverPreviewUrl"
            alt=""
          />
          <span v-else class="mb-collect-folder-dialog__cover-stack" aria-hidden="true">
            <span class="mb-collect-folder-dialog__layer mb-collect-folder-dialog__layer--back2" />
            <span class="mb-collect-folder-dialog__layer mb-collect-folder-dialog__layer--back1" />
            <span class="mb-collect-folder-dialog__layer mb-collect-folder-dialog__layer--front">
              <span class="mb-collect-folder-dialog__plus">+</span>
            </span>
          </span>
        </button>
        <input
          ref="coverInput"
          type="file"
          class="mb-collect-folder-dialog__file"
          accept="image/jpeg,image/png,image/gif,image/bmp,image/webp"
          @change="onCoverChange"
        />

        <div class="mb-collect-folder-dialog__field">
          <label class="mb-collect-folder-dialog__label">
            名称<span class="is-req">*</span>
          </label>
          <div class="mb-collect-folder-dialog__input-wrap mb-collect-folder-dialog__input-wrap--line">
            <input
              v-model.trim="draftTitle"
              type="text"
              class="mb-collect-folder-dialog__input"
              maxlength="20"
              placeholder="快来给你的收藏夹命名吧"
              :disabled="loading"
              @keydown.enter.prevent="onSubmit"
            />
            <span class="mb-collect-folder-dialog__count">{{ titleLen }} / 20</span>
          </div>
        </div>

        <div class="mb-collect-folder-dialog__switch-row">
          <span class="mb-collect-folder-dialog__label">公开</span>
          <button
            type="button"
            class="mb-collect-folder-dialog__switch"
            :class="{ 'is-on': draftPublic }"
            role="switch"
            :aria-checked="draftPublic"
            :disabled="loading"
            @click="draftPublic = !draftPublic"
          />
        </div>

        <div class="mb-collect-folder-dialog__field">
          <label class="mb-collect-folder-dialog__label">简介</label>
          <div class="mb-collect-folder-dialog__input-wrap mb-collect-folder-dialog__input-wrap--area">
            <textarea
              v-model.trim="draftDesc"
              class="mb-collect-folder-dialog__textarea"
              maxlength="200"
              rows="4"
              placeholder="可以简单描述下你的收藏夹"
              :disabled="loading"
            />
            <span class="mb-collect-folder-dialog__count">{{ descLen }} / 200</span>
          </div>
        </div>

        <div class="mb-collect-folder-dialog__actions">
          <button
            type="button"
            class="mb-collect-folder-dialog__btn mb-collect-folder-dialog__btn--cancel"
            :disabled="loading"
            @click="onCancel"
          >
            取消
          </button>
          <button
            type="button"
            class="mb-collect-folder-dialog__btn mb-collect-folder-dialog__btn--ok"
            :disabled="loading || !canSubmit"
            @click="onSubmit"
          >
            {{ submitLabel }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script>
import { ElMessage } from "element-plus";

const COVER_MAX_BYTES = 10 * 1024 * 1024;
const COVER_ACCEPT = ["image/jpeg", "image/png", "image/gif", "image/bmp", "image/webp"];

export default {
  name: "MbCollectFolderCreateDialog",
  props: {
    modelValue: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
    mode: {
      type: String,
      default: "create",
      validator: (v) => v === "create" || v === "edit"
    },
    initial: {
      type: Object,
      default: null
    }
  },
  emits: ["update:modelValue", "submit", "cancel"],
  data() {
    return {
      draftTitle: "",
      draftDesc: "",
      draftPublic: true,
      coverFile: null,
      coverPreviewUrl: "",
      coverPreviewFromFile: false
    };
  },
  computed: {
    isEdit() {
      return this.mode === "edit";
    },
    dialogTitle() {
      return this.isEdit ? "收藏夹信息" : "新建收藏夹";
    },
    submitLabel() {
      if (this.loading) {
        return this.isEdit ? "保存中…" : "创建中…";
      }
      return this.isEdit ? "保存" : "创建";
    },
    titleLen() {
      return [...String(this.draftTitle || "")].length;
    },
    descLen() {
      return [...String(this.draftDesc || "")].length;
    },
    canSubmit() {
      return this.titleLen > 0 && this.titleLen <= 20 && this.descLen <= 200;
    }
  },
  watch: {
    modelValue(open) {
      if (open) {
        this.applyInitialOrReset();
      }
    },
    initial: {
      handler() {
        if (this.modelValue && this.isEdit) {
          this.applyInitialOrReset();
        }
      },
      deep: true
    }
  },
  beforeUnmount() {
    this.revokeCoverPreview();
  },
  methods: {
    revokeCoverPreview() {
      if (this.coverPreviewFromFile && this.coverPreviewUrl) {
        URL.revokeObjectURL(this.coverPreviewUrl);
      }
      this.coverPreviewUrl = "";
      this.coverPreviewFromFile = false;
    },
    applyInitialOrReset() {
      this.revokeCoverPreview();
      this.coverFile = null;
      if (this.$refs.coverInput) {
        this.$refs.coverInput.value = "";
      }
      if (this.isEdit && this.initial) {
        this.draftTitle = String(this.initial.title || "");
        this.draftDesc = String(this.initial.description || "");
        this.draftPublic = this.initial.is_public !== false;
        const url = this.initial.cover_url || this.initial.coverUrl || "";
        this.coverPreviewUrl = url ? String(url) : "";
        this.coverPreviewFromFile = false;
        return;
      }
      this.draftTitle = "";
      this.draftDesc = "";
      this.draftPublic = true;
    },
    onPickCover() {
      if (this.loading) {
        return;
      }
      this.$refs.coverInput?.click();
    },
    onCoverChange(e) {
      const file = e.target?.files?.[0];
      if (!file) {
        return;
      }
      if (!COVER_ACCEPT.includes(file.type)) {
        ElMessage.warning("封面格式不支持，请使用 JPEG/PNG/GIF/BMP/WEBP");
        e.target.value = "";
        return;
      }
      if (file.size > COVER_MAX_BYTES) {
        ElMessage.warning("封面大小不能超过 10MB");
        e.target.value = "";
        return;
      }
      this.revokeCoverPreview();
      this.coverFile = file;
      this.coverPreviewUrl = URL.createObjectURL(file);
      this.coverPreviewFromFile = true;
    },
    onCancel() {
      if (this.loading) {
        return;
      }
      this.$emit("update:modelValue", false);
      this.$emit("cancel");
    },
    onSubmit() {
      if (!this.canSubmit || this.loading) {
        return;
      }
      this.$emit("submit", {
        title: String(this.draftTitle || "").trim(),
        description: String(this.draftDesc || "").trim(),
        is_public: !!this.draftPublic,
        cover: this.coverFile
      });
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/styles/mb-collect-folder-create-dialog.scss";
</style>
