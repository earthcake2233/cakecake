<template>
  <button
    type="button"
    class="vd-cmt-act"
    :class="[
      variant === 'like' ? 'vd-cmt-act--like' : 'vd-cmt-act--dislike',
      active ? (variant === 'like' ? 'is-liked' : 'is-disliked') : ''
    ]"
    :aria-label="variant === 'like' ? '点赞' : '踩'"
    @click="$emit('click', $event)"
  >
    <svg
      v-if="!active"
      class="vd-cmt-like-ico vd-cmt-like-ico--outline"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <path v-if="variant === 'like'" :d="thumbUpOutline" />
      <g v-else transform="translate(12 12) scale(1 -1) translate(-12 -12)">
        <path :d="thumbUpOutline" />
      </g>
    </svg>
    <svg
      v-else
      class="vd-cmt-like-ico vd-cmt-like-ico--solid"
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden="true"
    >
      <path v-if="variant === 'like'" :d="thumbUpSolid" />
      <g v-else transform="translate(12 12) scale(1 -1) translate(-12 -12)">
        <path :d="thumbUpSolid" />
      </g>
    </svg>
    <template v-if="variant === 'like' && showCount">{{ count }}</template>
  </button>
</template>

<script>
import { THUMB_UP_OUTLINE, THUMB_UP_SOLID } from "./commentThumbPaths.js";

export default {
  name: "VdCommentThumbBtn",
  props: {
    variant: {
      type: String,
      default: "like",
      validator: (v) => v === "like" || v === "dislike"
    },
    active: { type: Boolean, default: false },
    count: { type: [Number, String], default: 0 },
    showCount: { type: Boolean, default: true }
  },
  emits: ["click"],
  computed: {
    thumbUpOutline() {
      return THUMB_UP_OUTLINE;
    },
    thumbUpSolid() {
      return THUMB_UP_SOLID;
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/styles/vd-comment-list.scss";
</style>
