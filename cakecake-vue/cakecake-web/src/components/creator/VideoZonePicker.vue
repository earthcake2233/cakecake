<template>
  <div ref="rootRef" class="vp-zone-picker">
    <button
      id="vp-zone"
      type="button"
      class="vp-zone-trigger"
      :class="{ 'vp-zone-trigger--placeholder': !modelValue }"
      aria-haspopup="listbox"
      :aria-expanded="open ? 'true' : 'false'"
      @click="toggleOpen"
    >
      <span class="vp-zone-trigger__text">{{ displayLabel || "请选择分区" }}</span>
      <span class="vp-zone-trigger__arrow" :class="{ open }" aria-hidden="true" />
    </button>
    <div v-show="open" class="vp-zone-panel" role="listbox">
      <div
        v-for="cat in categories"
        :key="cat.name"
        class="vp-zone-group"
        :class="{ 'vp-zone-group--expanded': expandedParent === cat.name }"
      >
        <div class="vp-zone-parent-row">
          <button
            v-if="cat.items && cat.items.length"
            type="button"
            class="vp-zone-expand"
            :aria-label="expandedParent === cat.name ? '收起' : '展开'"
            @click.stop="toggleExpand(cat.name)"
          >
            <span class="vp-zone-expand__icon" />
          </button>
          <span v-else class="vp-zone-expand vp-zone-expand--spacer" aria-hidden="true" />
          <button
            type="button"
            class="vp-zone-parent"
            :class="{ 'vp-zone-parent--on': modelValue === cat.name }"
            @click="selectZone(cat.name)"
          >
            {{ cat.name }}
          </button>
        </div>
        <div
          v-if="cat.items && cat.items.length && expandedParent === cat.name"
          class="vp-zone-children"
        >
          <button
            v-for="sub in cat.items"
            :key="sub"
            type="button"
            class="vp-zone-child"
            :class="{
              'vp-zone-child--on': modelValue === `${cat.name}-${sub}`
            }"
            @click="selectZone(`${cat.name}-${sub}`)"
          >
            {{ sub }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import {
  VIDEO_ZONE_CATEGORIES,
  formatVideoZoneLabel
} from "@/constants/videoZones";

export default {
  name: "VideoZonePicker",
  props: {
    modelValue: {
      type: String,
      default: ""
    }
  },
  emits: ["update:modelValue"],
  data() {
    return {
      open: false,
      expandedParent: ""
    };
  },
  computed: {
    categories() {
      return VIDEO_ZONE_CATEGORIES;
    },
    displayLabel() {
      return formatVideoZoneLabel(this.modelValue);
    }
  },
  watch: {
    modelValue: {
      immediate: true,
      handler(v) {
        this.syncExpandedFromValue(v);
      }
    }
  },
  mounted() {
    document.addEventListener("click", this.onDocClick);
  },
  beforeUnmount() {
    document.removeEventListener("click", this.onDocClick);
  },
  methods: {
    syncExpandedFromValue(v) {
      const z = String(v || "").trim();
      if (!z) return;
      const idx = z.indexOf("-");
      this.expandedParent = idx > 0 ? z.slice(0, idx) : z;
    },
    toggleOpen() {
      this.open = !this.open;
      if (this.open) {
        this.syncExpandedFromValue(this.modelValue);
      }
    },
    toggleExpand(name) {
      this.expandedParent = this.expandedParent === name ? "" : name;
    },
    selectZone(value) {
      this.$emit("update:modelValue", value);
      this.open = false;
      this.syncExpandedFromValue(value);
    },
    onDocClick(e) {
      if (!this.open) return;
      const root = this.$refs.rootRef;
      if (root && !root.contains(e.target)) {
        this.open = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
$c-blue: #00a1d6;
$c-text: #18191c;
$c-sub: #9499a0;
$c-line: #e3e5e7;
$c-panel: #fff;

.vp-zone-picker {
  position: relative;
  width: 100%;
  max-width: 420px;
}

.vp-zone-trigger {
  width: 100%;
  height: 36px;
  padding: 0 32px 0 12px;
  border: 1px solid $c-line;
  border-radius: 4px;
  background: $c-panel;
  font-size: 14px;
  color: $c-text;
  text-align: left;
  cursor: pointer;
  position: relative;
  box-sizing: border-box;
  &:hover {
    border-color: #c9ccd0;
  }
  &--placeholder .vp-zone-trigger__text {
    color: $c-sub;
  }
}

.vp-zone-trigger__text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.vp-zone-trigger__arrow {
  position: absolute;
  right: 12px;
  top: 50%;
  width: 0;
  height: 0;
  margin-top: -3px;
  border-left: 5px solid transparent;
  border-right: 5px solid transparent;
  border-top: 6px solid $c-sub;
  transition: transform 0.15s;
  &.open {
    transform: rotate(180deg);
  }
}

.vp-zone-panel {
  position: absolute;
  z-index: 120;
  left: 0;
  right: 0;
  top: calc(100% + 4px);
  max-height: 320px;
  overflow-y: auto;
  background: $c-panel;
  border: 1px solid $c-line;
  border-radius: 6px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  padding: 6px 0;
  box-sizing: border-box;
}

.vp-zone-group {
  border-bottom: 1px solid #f1f2f3;
  &:last-child {
    border-bottom: none;
  }
}

.vp-zone-parent-row {
  display: flex;
  align-items: stretch;
  min-height: 36px;
}

.vp-zone-expand {
  flex: 0 0 36px;
  width: 36px;
  border: none;
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  &--spacer {
    cursor: default;
    pointer-events: none;
  }
  &:hover:not(.vp-zone-expand--spacer) {
    background: #f6f7f8;
  }
}

.vp-zone-expand__icon {
  display: block;
  width: 6px;
  height: 6px;
  border-right: 1.5px solid #61666d;
  border-bottom: 1.5px solid #61666d;
  transform: rotate(-45deg);
  transition: transform 0.15s;
  .vp-zone-group--expanded & {
    transform: rotate(45deg);
  }
}

.vp-zone-parent {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  text-align: left;
  font-size: 14px;
  font-weight: 600;
  color: $c-text;
  padding: 8px 12px 8px 0;
  cursor: pointer;
  &:hover {
    color: $c-blue;
  }
  &--on {
    color: $c-blue;
    background: #e3f3ff;
  }
}

.vp-zone-children {
  padding: 0 0 6px 36px;
}

.vp-zone-child {
  display: block;
  width: 100%;
  border: none;
  background: transparent;
  text-align: left;
  font-size: 13px;
  color: #61666d;
  padding: 7px 12px;
  border-radius: 4px;
  cursor: pointer;
  box-sizing: border-box;
  &:hover {
    color: $c-blue;
    background: #f6f7f8;
  }
  &--on {
    color: $c-blue;
    background: #e3f3ff;
    font-weight: 500;
  }
}
</style>
