<template>
  <CreatorShell>
    <div class="ap-wrap">
      <div class="ap-panel">
        <div class="ap-tabs">
          <button
            type="button"
            class="ap-tab"
            :class="{ on: tab === 'all' }"
            @click="tab = 'all'"
          >
            全部({{ totalCount }})
          </button>
          <button
            type="button"
            class="ap-tab"
            :class="{ on: tab === 'processing' }"
            @click="tab = 'processing'"
          >
            进行中({{ processingCount }})
          </button>
          <button
            type="button"
            class="ap-tab"
            :class="{ on: tab === 'completed' }"
            @click="tab = 'completed'"
          >
            已完成({{ completedCount }})
          </button>
        </div>

        <div v-if="filteredAppeals.length" class="ap-list">
          <div
            v-for="row in filteredAppeals"
            :key="row.id"
            class="ap-row"
          >
            <span class="ap-row-title">{{ row.title }}</span>
            <span class="ap-row-meta">{{ appealStatusText(row.status) }}</span>
          </div>
        </div>

        <div v-else class="ap-empty">
          <img
            class="ap-empty-img"
            :src="emptyIllus"
            alt=""
          />
          <p class="ap-empty-txt">{{ emptyHint }}</p>
        </div>
      </div>
    </div>
  </CreatorShell>
</template>

<script>
import CreatorShell from "@/components/creator/CreatorShell.vue";
import { CREATOR_APPEALS } from "./creatorAppealMock.js";
import emptyIllus from "@/assets/err-no-list.716e40d2.png";

export default {
  name: "AppealPage",
  components: { CreatorShell },
  data() {
    return {
      tab: "all",
      appeals: [...CREATOR_APPEALS],
      emptyIllus
    };
  },
  computed: {
    totalCount() {
      return this.appeals.length;
    },
    processingCount() {
      return this.appeals.filter((a) => a.status === "processing").length;
    },
    completedCount() {
      return this.appeals.filter((a) => a.status === "completed").length;
    },
    filteredAppeals() {
      if (this.tab === "all") return this.appeals;
      if (this.tab === "processing") {
        return this.appeals.filter((a) => a.status === "processing");
      }
      return this.appeals.filter((a) => a.status === "completed");
    },
    /** 仅在「全部」且一条都没有时用第一句；「进行中」「已完成」空列表一律第二句 */
    emptyHint() {
      if (this.tab === "all" && this.totalCount === 0) {
        return "你还没有发起过申诉(\"▔□▔)/";
      }
      return "没有该类型的申诉(\"▔□▔)";
    }
  },
  methods: {
    appealStatusText(status) {
      return status === "completed" ? "已完成" : "进行中";
    }
  }
};
</script>

<style lang="scss" scoped>
$c-blue: #00a1d6;
$c-text: #18191c;
$c-sub: #99a2aa;
$c-line: #e3e5e7;

.ap-wrap {
  max-width: 1120px;
  margin: 0 auto;
}

.ap-panel {
  background: #fff;
  border: 1px solid $c-line;
  border-radius: 8px;
  min-height: 420px;
  box-sizing: border-box;
}

.ap-tabs {
  display: flex;
  align-items: center;
  gap: 28px;
  padding: 0 24px;
  border-bottom: 1px solid $c-line;
}

.ap-tab {
  position: relative;
  padding: 16px 2px 14px;
  margin-bottom: -1px;
  border: none;
  background: none;
  font-size: 15px;
  color: #505050;
  cursor: pointer;
  border-bottom: 3px solid transparent;
}
.ap-tab:hover {
  color: $c-blue;
}
.ap-tab.on {
  color: $c-blue;
  font-weight: 600;
  border-bottom-color: $c-blue;
}

.ap-list {
  padding: 16px 24px 24px;
}

.ap-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 0;
  border-bottom: 1px solid #f0f1f2;
  font-size: 14px;
}
.ap-row:last-child {
  border-bottom: none;
}

.ap-row-title {
  color: $c-text;
}

.ap-row-meta {
  color: $c-sub;
  font-size: 13px;
}

.ap-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px 64px;
}

.ap-empty-img {
  width: 280px;
  max-width: 86%;
  height: auto;
  object-fit: contain;
  display: block;
}

.ap-empty-txt {
  margin: 20px 0 0;
  font-size: 14px;
  color: $c-sub;
  text-align: center;
  line-height: 1.6;
}
</style>
