<template>
  <div class="mb-dm">
    <div class="mb-dm__table-head">
      <span class="mb-dm__th mb-dm__th--t">时间</span>
      <span class="mb-dm__th mb-dm__th--c">弹幕内容</span>
      <span class="mb-dm__th mb-dm__th--s">发送时间</span>
    </div>
    <div ref="tbodyRef" class="mb-dm__table-body">
      <div v-for="row in filteredRows" :key="row.id" class="mb-dm__tr">
        <span class="mb-dm__td mb-dm__td--t">{{ formatVideoTime(row.video_time) }}</span>
        <span class="mb-dm__td mb-dm__td--c" :title="row.content">{{ formatDmPreview(row.content) }}</span>
        <span
          class="mb-dm__td mb-dm__td--s"
          :title="row.created_at ? String(row.created_at) : ''"
        >{{ row.created_at || "—" }}</span>
      </div>
      <div v-if="!filteredRows.length && !wsHint" class="mb-dm__empty">
        {{ filterDateKey ? "该日暂无弹幕" : "暂无弹幕" }}
      </div>
    </div>
    <div class="mb-dm__foot">
      <button type="button" class="mb-dm__history-btn" @click="toggleCalendar">
        {{ calendarOpen ? "收起日历" : "查看历史弹幕" }}
      </button>
      <button
        v-if="filterDateKey"
        type="button"
        class="mb-dm__clear-filter"
        @click="clearDateFilter"
      >
        显示全部
      </button>
    </div>

    <div v-show="calendarOpen" class="mb-dm__calendar">
      <div class="mb-cal__head">
        <button type="button" class="mb-cal__nav" aria-label="上月" @click="prevMonth">‹</button>
        <span class="mb-cal__title">{{ calendarTitle }}</span>
        <button type="button" class="mb-cal__nav" aria-label="下月" @click="nextMonth">›</button>
      </div>
      <div class="mb-cal__week">
        <span v-for="w in weekLabels" :key="w" class="mb-cal__week-cell">{{ w }}</span>
      </div>
      <div class="mb-cal__grid">
        <button
          v-for="(cell, idx) in calendarCells"
          :key="'c' + idx"
          type="button"
          class="mb-cal__day"
          :class="{
            'mb-cal__day--pad': cell.pad,
            'mb-cal__day--muted': cell.muted,
            'mb-cal__day--selected': cell.selected
          }"
          :disabled="cell.pad || cell.muted"
          @click="onPickDay(cell)"
        >
          {{ cell.label }}
        </button>
      </div>
      <div class="mb-cal__foot">
        <button type="button" class="mb-dm__history-btn" @click="toggleCalendar">查看历史弹幕</button>
      </div>
    </div>

    <div v-if="wsHint" class="mb-dm__hint">{{ wsHint }}</div>
  </div>
</template>

<script>
const CH_MONTH = [
  "一月",
  "二月",
  "三月",
  "四月",
  "五月",
  "六月",
  "七月",
  "八月",
  "九月",
  "十月",
  "十一月",
  "十二月"
];
const WEEK_LABELS = ["日", "一", "二", "三", "四", "五", "六"];

function pad2(n) {
  return String(n).padStart(2, "0");
}

function ymd(d) {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
}

export default {
  name: "MinibiliDanmakuFeed",
  props: {
    videoId: { type: Number, required: true },
    catalog: {
      type: Array,
      default: () => []
    },
    wsHint: { type: String, default: "" }
  },
  data() {
    const t = new Date();
    return {
      calendarOpen: false,
      filterDateKey: "",
      viewYear: t.getFullYear(),
      viewMonth: t.getMonth(),
      weekLabels: WEEK_LABELS
    };
  },
  computed: {
    sortedRows() {
      const rows = (this.catalog || []).slice();
      rows.sort((a, b) => (Number(b.id) || 0) - (Number(a.id) || 0));
      return rows;
    },
    filteredRows() {
      const key = this.filterDateKey;
      if (!key) return this.sortedRows;
      return this.sortedRows.filter(r => this.rowYmd(r) === key);
    },
    calendarTitle() {
      return `${this.viewYear}年 ${CH_MONTH[this.viewMonth]}`;
    },
    todayYmd() {
      return ymd(new Date());
    },
    calendarCells() {
      const y = this.viewYear;
      const m = this.viewMonth;
      const first = new Date(y, m, 1);
      const startPad = first.getDay();
      const daysInMonth = new Date(y, m + 1, 0).getDate();
      const cells = [];
      for (let i = 0; i < startPad; i++) {
        cells.push({ pad: true, label: "" });
      }
      const today = this.todayYmd;
      for (let d = 1; d <= daysInMonth; d++) {
        const key = `${y}-${pad2(m + 1)}-${pad2(d)}`;
        const muted = key > today;
        const selected = this.filterDateKey === key;
        cells.push({
          pad: false,
          muted,
          selected,
          label: String(d),
          ymd: key
        });
      }
      while (cells.length % 7 !== 0) {
        cells.push({ pad: true, label: "" });
      }
      return cells;
    }
  },
  methods: {
    rowYmd(row) {
      const s = String(row.created_at || "").trim();
      if (s.length >= 10) return s.slice(0, 10);
      return "";
    },
    formatVideoTime(sec) {
      const s = Math.max(0, Number(sec) || 0);
      const m = Math.floor(s / 60);
      const r = Math.floor(s % 60);
      return `${String(m).padStart(2, "0")}:${String(r).padStart(2, "0")}`;
    },
    /** 列表内最多展示 10 个字，超出用 …（完整内容用 title） */
    formatDmPreview(text) {
      const s = String(text ?? "");
      const chars = Array.from(s);
      if (chars.length <= 10) return s;
      return `${chars.slice(0, 10).join("")}...`;
    },
    toggleCalendar() {
      this.calendarOpen = !this.calendarOpen;
    },
    clearDateFilter() {
      this.filterDateKey = "";
    },
    prevMonth() {
      if (this.viewMonth === 0) {
        this.viewYear -= 1;
        this.viewMonth = 11;
      } else {
        this.viewMonth -= 1;
      }
    },
    nextMonth() {
      const t = new Date();
      const maxY = t.getFullYear();
      const maxM = t.getMonth();
      if (this.viewYear > maxY || (this.viewYear === maxY && this.viewMonth >= maxM)) {
        return;
      }
      if (this.viewMonth === 11) {
        this.viewYear += 1;
        this.viewMonth = 0;
      } else {
        this.viewMonth += 1;
      }
    },
    onPickDay(cell) {
      if (!cell || cell.pad || cell.muted || !cell.ymd) return;
      this.filterDateKey = cell.ymd;
      this.calendarOpen = false;
      this.$nextTick(() => {
        const el = this.$refs.tbodyRef;
        if (el) el.scrollTop = 0;
      });
    }
  }
};
</script>

<style scoped lang="scss">
.mb-dm {
  margin-bottom: 0;
  padding: 0 0 8px;
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-height: 0;
  width: 100%;
}
.mb-dm__table-head {
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr) minmax(132px, 38%);
  gap: 6px;
  padding: 8px 4px 6px;
  font-size: 12px;
  font-weight: 600;
  color: #18191c;
  border-bottom: 1px solid #e3e5e7;
  flex-shrink: 0;
}
.mb-dm__th--c {
  padding-left: 2px;
}
.mb-dm__th--s {
  text-align: center;
}
.mb-dm__table-body {
  flex: 1 1 auto;
  min-height: 120px;
  overflow-y: auto;
  font-size: 12px;
  line-height: 1.45;
  padding: 4px 0 8px;
}
.mb-dm__tr {
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr) minmax(132px, 38%);
  gap: 6px;
  align-items: start;
  padding: 6px 4px;
  border-bottom: 1px solid #f1f2f3;
}
.mb-dm__td--t {
  color: #61666d;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.mb-dm__td--c {
  color: #18191c;
  white-space: nowrap;
  overflow: hidden;
  min-width: 0;
}
.mb-dm__td--s {
  color: #9499a0;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  text-align: center;
  min-width: 0;
}
.mb-dm__empty {
  color: #9499a0;
  text-align: center;
  padding: 24px 0;
  font-size: 12px;
}
.mb-dm__foot {
  flex-shrink: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 6px 0 8px;
  border-bottom: 1px solid #e3e5e7;
}
.mb-dm__history-btn {
  border: none;
  background: none;
  color: #00aeec;
  font-size: 12px;
  cursor: pointer;
  padding: 4px 8px;
}
.mb-dm__history-btn:hover {
  text-decoration: underline;
}
.mb-dm__clear-filter {
  border: none;
  background: none;
  color: #9499a0;
  font-size: 12px;
  cursor: pointer;
  padding: 4px 8px;
}
.mb-dm__clear-filter:hover {
  color: #61666d;
}
.mb-dm__calendar {
  flex-shrink: 0;
  padding: 10px 8px 12px;
  background: #fff;
  border: 1px solid #e3e5e7;
  border-radius: 4px;
  margin-top: 6px;
}
.mb-cal__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  padding: 0 4px;
}
.mb-cal__title {
  font-size: 13px;
  font-weight: 600;
  color: #18191c;
}
.mb-cal__nav {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 4px;
  background: #f4f5f7;
  color: #61666d;
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
}
.mb-cal__nav:hover {
  background: #ebedf0;
}
.mb-cal__week {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  text-align: center;
  font-size: 12px;
  color: #18191c;
  margin-bottom: 6px;
}
.mb-cal__week-cell {
  padding: 4px 0;
}
.mb-cal__grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
}
.mb-cal__day {
  height: 30px;
  border: 1px solid transparent;
  border-radius: 2px;
  background: #fff;
  font-size: 12px;
  color: #18191c;
  cursor: pointer;
  padding: 0;
}
.mb-cal__day--pad {
  visibility: hidden;
  pointer-events: none;
}
.mb-cal__day--muted {
  color: #c0c4cc;
  cursor: default;
}
.mb-cal__day--selected:not(.mb-cal__day--muted) {
  background: #dff6fc;
  border-color: #9499a0;
  color: #00a1d6;
  font-weight: 600;
}
.mb-cal__day:hover:not(:disabled) {
  background: #f4f5f7;
}
.mb-cal__foot {
  margin-top: 10px;
  padding-top: 8px;
  border-top: 1px solid #ebedf0;
  text-align: center;
}
.mb-dm__hint {
  margin-top: 8px;
  font-size: 12px;
  color: #f56c6c;
}
</style>
