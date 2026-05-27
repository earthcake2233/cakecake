<template>
  <div
    class="filter-wrap"
    :class="{
      'filter-wrap--folded': !fold && !viewOnly,
      'filter-wrap--view-only': viewOnly
    }"
  >
    <template v-if="!viewOnly">
    <ul class="filter-type clearfix order">
      <li
        v-for="opt in orderOptions"
        :key="`order_${opt.id}`"
        class="filter-item"
        :class="{ active: order === opt.id }"
      >
        <a href="javascript:;" @click.prevent="pickOrder(opt.id)">{{ opt.name }}</a>
      </li>
    </ul>
    <ul v-show="fold" class="filter-type clearfix duration">
      <li
        v-for="opt in durationOptions"
        :key="`dur_${opt.id}`"
        class="filter-item"
        :class="{ active: duration === opt.id }"
      >
        <a href="javascript:;" @click.prevent="pickDuration(opt.id)">{{ opt.name }}</a>
      </li>
    </ul>
    <ul v-show="fold" class="filter-type clearfix tids_1">
      <li
        v-for="opt in zoneOptions"
        :key="`zone_${opt.id}`"
        class="filter-item"
        :class="{ active: zone === opt.id }"
      >
        <a href="javascript:;" @click.prevent="pickZone(opt.id)">{{ opt.name }}</a>
      </li>
    </ul>
    <a v-if="fold" class="fold up" href="javascript:;" @click.prevent="fold = false">
      收起筛选
      <i class="arrow-up"></i>
    </a>
    <a v-else class="fold down" href="javascript:;" @click.prevent="fold = true">
      更多筛选
      <i class="arrow-down"></i>
    </a>
    </template>
    <div class="switch-wrap">
      <button
        type="button"
        class="switch-item aver type"
        :class="{ active: viewMode === 'grid' }"
        title="网格"
        @click="setViewMode('grid')"
      >
        <svg
          class="switch-ico"
          width="16"
          height="16"
          viewBox="0 0 16 16"
          aria-hidden="true"
        >
          <rect x="1" y="1" width="4" height="4" rx="0.5" />
          <rect x="6" y="1" width="4" height="4" rx="0.5" />
          <rect x="11" y="1" width="4" height="4" rx="0.5" />
          <rect x="1" y="6" width="4" height="4" rx="0.5" />
          <rect x="6" y="6" width="4" height="4" rx="0.5" />
          <rect x="11" y="6" width="4" height="4" rx="0.5" />
          <rect x="1" y="11" width="4" height="4" rx="0.5" />
          <rect x="6" y="11" width="4" height="4" rx="0.5" />
          <rect x="11" y="11" width="4" height="4" rx="0.5" />
        </svg>
      </button>
      <button
        type="button"
        class="switch-item imgleft type"
        :class="{ active: viewMode === 'list' }"
        title="列表"
        @click="setViewMode('list')"
      >
        <svg
          class="switch-ico"
          width="16"
          height="16"
          viewBox="0 0 16 16"
          aria-hidden="true"
        >
          <rect x="1" y="2" width="3" height="3" rx="0.5" />
          <rect x="6" y="2.5" width="9" height="2" rx="0.5" />
          <rect x="1" y="7" width="3" height="3" rx="0.5" />
          <rect x="6" y="7.5" width="9" height="2" rx="0.5" />
          <rect x="1" y="12" width="3" height="3" rx="0.5" />
          <rect x="6" y="12.5" width="9" height="2" rx="0.5" />
        </svg>
      </button>
    </div>
  </div>
</template>

<script>
import {
  SEARCH_ORDER_OPTIONS,
  SEARCH_DURATION_OPTIONS,
  SEARCH_ZONE_OPTIONS,
  DEFAULT_VIDEO_FILTERS,
  DEFAULT_SEARCH_VIDEO_VIEW
} from "@/utils/searchFilters";

export default {
  props: {
    modelValue: {
      type: Object,
      default: () => ({ ...DEFAULT_VIDEO_FILTERS })
    },
    viewMode: {
      type: String,
      default: DEFAULT_SEARCH_VIDEO_VIEW
    },
    /** 仅展示网格/列表切换（综合搜索视频区） */
    viewOnly: {
      type: Boolean,
      default: false
    }
  },
  emits: ["update:modelValue", "change", "view-change"],
  data() {
    return {
      orderOptions: SEARCH_ORDER_OPTIONS,
      durationOptions: SEARCH_DURATION_OPTIONS,
      zoneOptions: SEARCH_ZONE_OPTIONS,
      fold: true
    };
  },
  computed: {
    order() {
      return (this.modelValue && this.modelValue.order) || "default";
    },
    duration() {
      return (this.modelValue && this.modelValue.duration) || "all";
    },
    zone() {
      return (this.modelValue && this.modelValue.zone) || "";
    }
  },
  methods: {
    emitChange(patch) {
      const next = {
        order: this.order,
        duration: this.duration,
        zone: this.zone,
        ...patch
      };
      this.$emit("update:modelValue", next);
      this.$emit("change", next);
    },
    pickOrder(id) {
      this.emitChange({ order: id });
    },
    pickDuration(id) {
      this.emitChange({ duration: id });
    },
    pickZone(id) {
      this.emitChange({ zone: id });
    },
    setViewMode(mode) {
      if (mode === this.viewMode) {
        return;
      }
      this.$emit("view-change", mode);
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../../style/mixin";

.switch-wrap {
  .switch-item {
    position: absolute;
    top: 0;
    margin: 0;
    padding: 0;
    border: none;
    background: transparent;
    cursor: pointer;
    line-height: 0;
    color: #c9ccd0;

    &.aver {
      right: 26px;
    }
    &.imgleft {
      right: 0;
    }
    &.active {
      color: $blue;
    }
    &:hover {
      color: $blue;
    }
  }

  .switch-ico {
    display: block;
    fill: currentColor;
  }
}
</style>
