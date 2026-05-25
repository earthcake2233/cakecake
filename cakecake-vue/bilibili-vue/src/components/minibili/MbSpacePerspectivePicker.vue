<template>
  <div
    class="mb-space-perspective-picker"
    @mouseenter="onMenuEnter"
    @mouseleave="onMenuLeave"
  >
    <button
      type="button"
      class="mb-space-perspective-picker__trigger"
      :aria-expanded="menuOpen ? 'true' : 'false'"
      :aria-label="t.perspective.menuAria"
      @click.stop="onTriggerClick"
    >
      <span>{{ t.perspective.triggerSelf }}</span>
      <span class="mb-space-perspective-picker__chev" aria-hidden="true">&#9662;</span>
    </button>
    <ul
      v-show="menuOpen"
      class="mb-space-perspective-picker__menu"
      role="menu"
      @mouseenter="onMenuEnter"
      @mouseleave="onMenuLeave"
      @click.stop
    >
      <li role="none">
        <button
          type="button"
          role="menuitem"
          @click="selectMode('fan')"
        >
          {{ t.perspective.fan }}
        </button>
      </li>
      <li role="none">
        <button
          type="button"
          role="menuitem"
          @click="selectMode('visitor')"
        >
          {{ t.perspective.visitor }}
        </button>
      </li>
    </ul>
  </div>
</template>

<script>
import { personalSpaceZhCN } from "@/i18n/personalSpace.zh-CN";

export default {
  name: "MbSpacePerspectivePicker",
  props: {
    modelValue: {
      type: String,
      default: "self"
    }
  },
  emits: ["update:modelValue"],
  data() {
    return {
      t: personalSpaceZhCN,
      menuOpen: false,
      _menuCloseTimer: null
    };
  },
  beforeUnmount() {
    this.clearMenuCloseTimer();
  },
  methods: {
    clearMenuCloseTimer() {
      if (this._menuCloseTimer) {
        clearTimeout(this._menuCloseTimer);
        this._menuCloseTimer = null;
      }
    },
    onMenuEnter() {
      this.clearMenuCloseTimer();
      this.menuOpen = true;
    },
    onMenuLeave() {
      this.clearMenuCloseTimer();
      this._menuCloseTimer = setTimeout(() => {
        this.menuOpen = false;
        this._menuCloseTimer = null;
      }, 140);
    },
    onTriggerClick() {
      this.menuOpen = !this.menuOpen;
    },
    selectMode(mode) {
      this.menuOpen = false;
      if (mode === "fan" || mode === "visitor") {
        this.$emit("update:modelValue", mode);
      }
    }
  }
};
</script>

<style lang="scss" scoped>
.mb-space-perspective-picker {
  position: relative;
  flex-shrink: 0;
}

.mb-space-perspective-picker__trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 12px;
  border: 1px solid rgba(255, 255, 255, 0.65);
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.22);
  color: #fff;
  font-size: 13px;
  line-height: 1;
  cursor: pointer;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.25);
  backdrop-filter: blur(4px);
  white-space: nowrap;

  &:hover {
    background: rgba(0, 0, 0, 0.32);
  }
}

.mb-space-perspective-picker__chev {
  font-size: 11px;
  opacity: 0.9;
  transform: translateY(1px);
}

.mb-space-perspective-picker__menu {
  position: absolute;
  right: 0;
  bottom: calc(100% + 8px);
  top: auto;
  z-index: 30;
  min-width: 132px;
  margin: 0;
  padding: 6px 0;
  list-style: none;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);

  button {
    display: block;
    width: 100%;
    padding: 9px 16px;
    border: none;
    background: transparent;
    color: #61666d;
    font-size: 14px;
    line-height: 1.3;
    text-align: center;
    cursor: pointer;
    white-space: nowrap;

    &:hover {
      background: #f6f7f8;
      color: #18191c;
    }
  }
}
</style>
