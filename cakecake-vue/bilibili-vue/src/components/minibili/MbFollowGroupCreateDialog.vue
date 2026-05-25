<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="mb-follow-group-dialog-overlay"
      role="presentation"
      @click.self="onCancel"
    >
      <div
        class="mb-follow-group-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="mb-follow-group-dialog-title"
        @click.stop
      >
        <button
          type="button"
          class="mb-follow-group-dialog__close"
          aria-label="关闭"
          :disabled="loading"
          @click="onCancel"
        >
          ×
        </button>
        <h2 id="mb-follow-group-dialog-title" class="mb-follow-group-dialog__title">
          {{ dialogTitle }}
        </h2>

        <div class="mb-follow-group-dialog__input-wrap">
          <input
            ref="nameInput"
            v-model.trim="draftName"
            type="text"
            class="mb-follow-group-dialog__input"
            :maxlength="nameMax"
            :placeholder="inputPlaceholder"
            :disabled="loading"
            @keydown.enter.prevent="onSubmit"
          />
          <span class="mb-follow-group-dialog__count">{{ nameLen }} / {{ nameMax }}</span>
        </div>

        <div class="mb-follow-group-dialog__actions">
          <button
            type="button"
            class="mb-follow-group-dialog__btn mb-follow-group-dialog__btn--cancel"
            :disabled="loading"
            @click="onCancel"
          >
            取消
          </button>
          <button
            type="button"
            class="mb-follow-group-dialog__btn mb-follow-group-dialog__btn--ok"
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
const NAME_MAX = 16;

export default {
  name: "MbFollowGroupCreateDialog",
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
      draftName: "",
      nameMax: NAME_MAX
    };
  },
  computed: {
    isEdit() {
      return this.mode === "edit";
    },
    dialogTitle() {
      return this.isEdit ? "编辑分组名称" : "新建分组";
    },
    inputPlaceholder() {
      return this.isEdit ? "" : "快来给你的分组命名吧";
    },
    submitLabel() {
      if (this.loading) {
        return this.isEdit ? "保存中…" : "提交中…";
      }
      return "确定";
    },
    nameLen() {
      return [...String(this.draftName || "")].length;
    },
    canSubmit() {
      return this.nameLen > 0 && this.nameLen <= NAME_MAX;
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
  methods: {
    applyInitialOrReset() {
      if (this.isEdit && this.initial) {
        this.draftName = String(this.initial.name || "");
      } else {
        this.draftName = "";
      }
      this.$nextTick(() => {
        this.$refs.nameInput?.focus();
      });
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
        name: String(this.draftName || "").trim()
      });
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/styles/mb-follow-group-create-dialog.scss";
</style>
