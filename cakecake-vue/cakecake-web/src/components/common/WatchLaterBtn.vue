<template>
  <button
    v-if="resolvedId > 0"
    type="button"
    class="watch-later-trigger w-later home-wl-btn"
    :class="{ 'home-wl-btn--on': active }"
    :disabled="pending"
    aria-label="稍后再看"
    @click.stop.prevent="onClick"
  >
    <span class="home-wl-btn__inner">
      <span class="home-wl-btn__ico-wrap">
        <img class="home-wl-btn__ico" :src="thumbLaterIco" alt="" />
      </span>
      <span class="home-wl-btn__txt">稍后再看</span>
    </span>
  </button>
</template>

<script>
import thumbLaterIco from "@/assets/personal_space/latertowatch.png";
import { toggleWatchLaterVideo } from "@/utils/watchLaterAction";

export default {
  name: "WatchLaterBtn",
  props: {
    videoId: {
      type: [Number, String],
      default: 0
    },
    inWatchLater: {
      type: Boolean,
      default: false
    }
  },
  data() {
    return {
      thumbLaterIco,
      pending: false,
      active: false
    };
  },
  watch: {
    inWatchLater: {
      immediate: true,
      handler(v) {
        this.active = !!v;
      }
    }
  },
  computed: {
    resolvedId() {
      const id = Number(this.videoId);
      return Number.isFinite(id) && id > 0 ? id : 0;
    }
  },
  methods: {
    async onClick() {
      if (this.pending) {
        return;
      }
      this.pending = true;
      try {
        const on = await toggleWatchLaterVideo(this.$store, this.resolvedId);
        if (on !== null) {
          this.active = on;
        }
      } finally {
        this.pending = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
/* 默认隐藏；由父级 .video-thumb-hover:hover 在全局样式中显示 */
.home-wl-btn.watch-later-trigger {
  position: absolute;
  right: 6px;
  bottom: 6px;
  z-index: 20;
  width: auto;
  min-width: 22px;
  height: 22px;
  padding: 0;
  border: none;
  background: none !important;
  background-image: none !important;
  cursor: pointer;
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
  transition: opacity 0.2s ease, visibility 0.2s ease;

  .home-wl-btn__inner {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 0;
    height: 22px;
    border-radius: 4px;
    overflow: hidden;
    background: rgba(0, 0, 0, 0.72);
    transition: max-width 0.2s ease;
    max-width: 22px;
  }

  .home-wl-btn__ico-wrap {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
  }

  .home-wl-btn__ico {
    width: 14px;
    height: 14px;
    display: block;
  }

  .home-wl-btn__txt {
    display: block;
    max-width: 0;
    overflow: hidden;
    white-space: nowrap;
    font-size: 12px;
    line-height: 22px;
    color: #fff;
    opacity: 0;
    transition: max-width 0.2s ease, opacity 0.15s ease, padding 0.2s ease;
    padding: 0;
  }

  &:hover,
  &:focus-visible {
    .home-wl-btn__inner {
      max-width: 88px;
    }
    .home-wl-btn__txt {
      max-width: 64px;
      opacity: 1;
      padding-right: 6px;
    }
  }

  &.home-wl-btn--on .home-wl-btn__ico-wrap {
    background: rgba(0, 174, 236, 0.35);
  }
}
</style>
