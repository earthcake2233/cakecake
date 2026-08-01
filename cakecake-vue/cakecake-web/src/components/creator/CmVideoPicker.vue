<template>
  <div ref="root" class="cm-vp" :class="{ 'is-open': open }">
    <button type="button" class="cm-vp__trigger" @click="toggle">
      <span class="cm-vp__label">{{ displayLabel }}</span>
      <span class="cm-vp__chev" aria-hidden="true" />
    </button>
    <div v-show="open" class="cm-vp__panel">
      <div class="cm-vp__search">
        <input
          v-model.trim="filterQ"
          type="search"
          class="cm-vp__search-input"
          :placeholder="searchPlaceholder"
          autocomplete="off"
          @keydown.stop
        />
        <span class="cm-vp__search-ico" aria-hidden="true" />
      </div>
      <ul class="cm-vp__list">
        <li>
          <button
            type="button"
            class="cm-vp__item"
            :class="{ 'is-on': modelValue === '' }"
            @click="pick('')"
          >
            <span class="cm-vp__item-t">{{ allLabel }}</span>
            <span v-if="modelValue === ''" class="cm-vp__check" aria-hidden="true">✓</span>
          </button>
        </li>
        <li v-for="v in filteredOptions" :key="v.id">
          <button
            type="button"
            class="cm-vp__item"
            :class="{ 'is-on': String(modelValue) === String(v.id) }"
            @click="pick(v.id)"
          >
            <span class="cm-vp__item-t">{{ optionTitle(v) }}</span>
            <span
              v-if="String(modelValue) === String(v.id)"
              class="cm-vp__check"
              aria-hidden="true"
            >✓</span>
          </button>
        </li>
        <li v-if="!filteredOptions.length && filterQ" class="cm-vp__empty-li">
          无匹配视频
        </li>
      </ul>
    </div>
  </div>
</template>

<script>
export default {
  name: "CmVideoPicker",
  props: {
    modelValue: { type: [String, Number], default: "" },
    options: { type: Array, default: () => [] },
    allLabel: { type: String, default: "全部视频" },
    searchPlaceholder: { type: String, default: "输入视频搜索关键字" }
  },
  emits: ["update:modelValue", "change"],
  data() {
    return {
      open: false,
      filterQ: ""
    };
  },
  computed: {
    displayLabel() {
      if (this.modelValue === "" || this.modelValue == null) {
        return this.allLabel;
      }
      const id = String(this.modelValue);
      const hit = (this.options || []).find((v) => String(v.id) === id);
      return hit ? this.optionTitle(hit) : this.allLabel;
    },
    filteredOptions() {
      const q = this.filterQ.trim().toLowerCase();
      const list = this.options || [];
      if (!q) return list;
      return list.filter((v) => {
        const t = String(v.title || "").toLowerCase();
        return t.includes(q) || String(v.id).includes(q);
      });
    }
  },
  mounted() {
    document.addEventListener("click", this.onDocClick);
  },
  beforeUnmount() {
    document.removeEventListener("click", this.onDocClick);
  },
  methods: {
    optionTitle(v) {
      const t = String((v && v.title) || "").trim() || `稿件 ${v.id}`;
      return t.length > 28 ? `${t.slice(0, 28)}…` : t;
    },
    toggle() {
      this.open = !this.open;
      if (this.open) this.filterQ = "";
    },
    pick(id) {
      const val = id === "" ? "" : String(id);
      this.$emit("update:modelValue", val);
      this.$emit("change", val);
      this.open = false;
      this.filterQ = "";
    },
    onDocClick(e) {
      const root = this.$refs.root;
      if (!root || root.contains(e.target)) return;
      this.open = false;
    }
  }
};
</script>

<style lang="scss" scoped>
.cm-vp {
  position: relative;
  min-width: 140px;
}
.cm-vp__trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  min-width: 140px;
  height: 32px;
  padding: 0 10px 0 12px;
  border: 1px solid #e3e5e7;
  border-radius: 4px;
  background: #fff;
  font-size: 14px;
  color: #18191c;
  cursor: pointer;
  &:hover {
    border-color: #00aeec;
  }
}
.cm-vp.is-open .cm-vp__trigger {
  border-color: #00aeec;
}
.cm-vp__label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
}
.cm-vp__chev {
  flex-shrink: 0;
  width: 0;
  height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 5px solid #9499a0;
  transition: transform 0.15s;
}
.cm-vp.is-open .cm-vp__chev {
  transform: rotate(180deg);
}
.cm-vp__panel {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  z-index: 30;
  width: 280px;
  max-height: 320px;
  background: #fff;
  border: 1px solid #e3e5e7;
  border-radius: 6px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
  overflow: hidden;
}
.cm-vp__search {
  position: relative;
  padding: 10px 10px 6px;
  border-bottom: 1px solid #f1f2f3;
}
.cm-vp__search-input {
  width: 100%;
  height: 32px;
  padding: 0 32px 0 10px;
  border: 1px solid #e3e5e7;
  border-radius: 4px;
  font-size: 13px;
  outline: none;
  &:focus {
    border-color: #00aeec;
  }
}
.cm-vp__search-ico {
  position: absolute;
  right: 18px;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  background: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%239499a0' stroke-width='2'%3E%3Ccircle cx='11' cy='11' r='7'/%3E%3Cpath d='M18 18l-4-4'/%3E%3C/svg%3E")
    center / contain no-repeat;
  pointer-events: none;
}
.cm-vp__list {
  list-style: none;
  margin: 0;
  padding: 4px 0;
  max-height: 260px;
  overflow-y: auto;
}
.cm-vp__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  padding: 10px 14px;
  border: none;
  background: none;
  font-size: 14px;
  color: #18191c;
  text-align: left;
  cursor: pointer;
  &:hover {
    background: #f6f7f8;
  }
  &.is-on {
    color: #00aeec;
  }
}
.cm-vp__item-t {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cm-vp__check {
  flex-shrink: 0;
  font-size: 14px;
  color: #00aeec;
}
.cm-vp__empty-li {
  padding: 12px 14px;
  font-size: 13px;
  color: #9499a0;
  text-align: center;
}
</style>
