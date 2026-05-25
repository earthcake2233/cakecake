<template>
  <div class="mb-cmt-curated-notice" role="status">
    <img
      class="mb-cmt-curated-notice__av"
      :src="resolvedAvatar"
      width="48"
      height="48"
      alt=""
    />
    <p class="mb-cmt-curated-notice__txt">{{ text }}</p>
  </div>
</template>

<script>
import defaultAvatar from "@/assets/akari.jpg";

export default {
  name: "MbCommentCuratedNotice",
  props: {
    avatarSrc: { type: String, default: "" },
    text: {
      type: String,
      default: "评论被up主精选后，对所有人可见"
    }
  },
  computed: {
    resolvedAvatar() {
      const u = String(this.avatarSrc || "").trim();
      return u || defaultAvatar;
    }
  }
};
</script>

<style lang="scss" scoped>
.mb-cmt-curated-notice {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 0 0 16px;
  padding: 0;
}
.mb-cmt-curated-notice__av {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  object-fit: cover;
  background: #f1f2f3;
}
.mb-cmt-curated-notice__txt {
  flex: 1;
  min-width: 0;
  margin: 0;
  padding: 12px 16px;
  border-radius: 8px;
  background: #f1f2f3;
  font-size: 14px;
  line-height: 22px;
  color: #61666d;
}
</style>
