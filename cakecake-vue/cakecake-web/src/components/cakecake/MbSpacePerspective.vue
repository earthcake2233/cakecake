<template>
  <div
    v-if="isPreview"
    class="mb-space-perspective-banner"
    role="status"
  >
    <p class="mb-space-perspective-banner__text">{{ bannerText }}</p>
    <button
      type="button"
      class="mb-space-perspective-banner__close"
      @click="closePreview"
    >
      {{ t.perspective.closePreview }}
    </button>
  </div>
</template>

<script>
import { personalSpaceZhCN } from "@/i18n/personalSpace.zh-CN";

export default {
  name: "MbSpacePerspective",
  props: {
    modelValue: {
      type: String,
      default: "self"
    }
  },
  emits: ["update:modelValue"],
  data() {
    return {
      t: personalSpaceZhCN
    };
  },
  computed: {
    isPreview() {
      return this.modelValue === "fan" || this.modelValue === "visitor";
    },
    bannerText() {
      if (this.modelValue === "visitor") {
        return this.t.perspective.bannerVisitor;
      }
      if (this.modelValue === "fan") {
        return this.t.perspective.bannerFan;
      }
      return "";
    }
  },
  methods: {
    closePreview() {
      this.$emit("update:modelValue", "self");
    }
  }
};
</script>

<style lang="scss" scoped>
.mb-space-perspective-banner {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  z-index: 9;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 20px;
  min-height: 40px;
  padding: 8px 16px;
  box-sizing: border-box;
  background: #e3f5ff;
  color: #18191c;
  font-size: 14px;
  line-height: 1.4;
}

.mb-space-perspective-banner__text {
  margin: 0;
  text-align: center;
}

.mb-space-perspective-banner__close {
  flex-shrink: 0;
  min-width: 88px;
  height: 30px;
  padding: 0 14px;
  border: none;
  border-radius: 6px;
  background: #fff;
  color: #18191c;
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);

  &:hover {
    background: #f6f7f8;
  }
}
</style>
