<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="mb-station-dialog-overlay"
      role="presentation"
      @click.self="emitCancel"
    >
      <div class="mb-station-dialog" role="dialog" aria-modal="true" @click.stop>
        <button
          type="button"
          class="mb-station-dialog__close"
          aria-label="关闭"
          :disabled="loading"
          @click="emitCancel"
        >
          ×
        </button>
        <h2 class="mb-station-dialog__title">{{ title }}</h2>
        <p class="mb-station-dialog__body">{{ message }}</p>
        <div class="mb-station-dialog__actions">
          <button
            type="button"
            class="mb-station-dialog__btn mb-station-dialog__btn--cancel"
            :disabled="loading"
            @click="emitCancel"
          >
            {{ cancelText }}
          </button>
          <button
            type="button"
            class="mb-station-dialog__btn mb-station-dialog__btn--confirm"
            :disabled="loading"
            @click="$emit('confirm')"
          >
            {{ confirmText }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script>
export default {
  name: "MbStationDialog",
  props: {
    modelValue: { type: Boolean, default: false },
    title: { type: String, default: "" },
    message: { type: String, default: "" },
    confirmText: { type: String, default: "确定" },
    cancelText: { type: String, default: "取消" },
    loading: { type: Boolean, default: false }
  },
  emits: ["update:modelValue", "confirm", "cancel"],
  methods: {
    emitCancel() {
      if (this.loading) {
        return;
      }
      this.$emit("update:modelValue", false);
      this.$emit("cancel");
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/styles/mb-station-dialog.scss";
</style>
