<template>
  <div class="adm-panel">
    <div class="adm-panel__head">
      <h2>热搜运营</h2>
      <div class="adm-panel__actions">
        <el-button :loading="boardLoading" @click="loadDashboard">刷新</el-button>
        <el-button type="primary" @click="openCreate">新增规则</el-button>
      </div>
    </div>
    <p class="adm-panel__desc">
      左侧主站展示榜可拖拽排序；右侧 Redis 榜可置顶、屏蔽、加热或移除。
    </p>

    <div v-loading="boardLoading" class="adm-board">
      <section class="adm-board__col adm-board__col--merged">
        <header class="adm-board__head">
          <h3>主站展示榜</h3>
          <div class="adm-board__head-right">
            <span v-if="customOrder" class="adm-board__tag">自定义排序</span>
            <button
              v-if="customOrder"
              type="button"
              class="adm-board__reset"
              @click="resetDisplayOrder"
            >
              恢复默认
            </button>
            <span class="adm-board__hint">拖拽把手调整顺序</span>
          </div>
        </header>
        <ol v-if="merged.length" class="adm-hot-list">
          <li
            v-for="(item, index) in merged"
            :key="mergedItemKey(item, index)"
            class="adm-hot-list__row"
            :class="{
              'adm-hot-list__row--dragging': dragIndex === index,
              'adm-hot-list__row--over': overIndex === index && dragIndex !== null
            }"
            @dragover.prevent="onDragOver(index)"
            @drop.prevent="onDrop(index)"
          >
            <span
              class="adm-hot-list__handle"
              title="拖拽排序"
              draggable="true"
              @dragstart="onDragStart(index, $event)"
              @dragend="onDragEnd"
            >⋮⋮</span>
            <span class="adm-hot-list__rank" :class="rankClass(item.rank)">{{
              item.rank
            }}</span>
            <span class="adm-hot-list__title">{{ item.title }}</span>
            <span v-if="item.badge === '热'" class="adm-hot-list__badge adm-hot-list__badge--hot"
              >热</span
            >
            <span
              v-else-if="item.badge === '新'"
              class="adm-hot-list__badge adm-hot-list__badge--new"
              >新</span
            >
            <span
              v-else-if="item.badge === '荐'"
              class="adm-hot-list__badge adm-hot-list__badge--rec"
              >荐</span
            >
            <span class="adm-hot-list__src" :class="`adm-hot-list__src--${item.source}`">{{
              sourceLabel(item.source)
            }}</span>
            <button
              v-if="item.op_id && item.source !== 'auto'"
              type="button"
              class="adm-hot-list__revoke"
              @click="cancelOpById(item.op_id, item.title, cancelLabel(item.source))"
            >
              {{ cancelLabel(item.source) }}
            </button>
          </li>
        </ol>
        <p v-if="reorderSaving" class="adm-board__saving">正在保存排序…</p>
        <p v-else-if="merged.length" class="adm-board__saving adm-board__saving--idle">
          共 {{ merged.length }} 条，按最终展示顺序排列
        </p>
        <p v-else class="adm-board__empty">暂无数据，可先配置规则或等待用户搜索产生 Redis 榜</p>
      </section>

      <section class="adm-board__col adm-board__col--redis">
        <header class="adm-board__head">
          <h3>Redis 自动榜</h3>
          <span class="adm-board__hint">真实搜索次数（ZSET 分数）</span>
        </header>
        <div v-if="redisRows.length" class="adm-redis-table-wrap">
          <table class="adm-redis-table">
            <thead>
              <tr>
                <th>#</th>
                <th>关键词</th>
                <th>热度</th>
                <th>状态</th>
                <th>干预</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in redisRows" :key="`r_${row.keyword}`">
                <td class="adm-redis-table__rank">{{ row.rank }}</td>
                <td>
                  <div class="adm-redis-table__kw">{{ row.title }}</div>
                  <div v-if="row.keyword !== row.title" class="adm-redis-table__norm">
                    {{ row.keyword }}
                  </div>
                </td>
                <td class="adm-redis-table__score">{{ formatScore(row.score) }}</td>
                <td>
                  <span v-if="row.blocked" class="adm-state adm-state--block">已屏蔽</span>
                  <span v-else-if="row.pinned" class="adm-state adm-state--pin">已置顶</span>
                  <span v-else-if="row.manual" class="adm-state adm-state--manual">人工</span>
                  <span v-else class="adm-state adm-state--auto">自动</span>
                </td>
                <td class="adm-redis-table__ops">
                  <template v-if="isPinned(row)">
                    <el-button link type="danger" @click="cancelOp(row, '置顶')">
                      取消置顶
                    </el-button>
                  </template>
                  <el-button v-else link type="primary" @click="quickPin(row)">置顶</el-button>

                  <template v-if="isManual(row)">
                    <el-button link type="danger" @click="cancelOp(row, '人工')">
                      取消人工
                    </el-button>
                  </template>
                  <el-button v-else link type="warning" @click="quickManual(row)">人工</el-button>

                  <template v-if="isBlocked(row)">
                    <el-button link type="danger" @click="cancelOp(row, '屏蔽')">
                      取消屏蔽
                    </el-button>
                  </template>
                  <el-button v-else link type="info" @click="quickBlock(row)">屏蔽</el-button>

                  <el-button link @click="boostRow(row)">+5</el-button>
                  <el-button link type="danger" @click="removeRedis(row)">移除</el-button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="adm-board__empty">Redis 暂无搜索统计</p>
      </section>
    </div>

    <div class="adm-rules">
      <header class="adm-rules__head">
        <h3>干预规则</h3>
        <div class="adm-rules__filter">
          <button
            v-for="f in opFilters"
            :key="f.id"
            type="button"
            class="adm-rules__tab"
            :class="{ 'adm-rules__tab--on': opFilter === f.id }"
            @click="opFilter = f.id"
          >
            {{ f.label }}
          </button>
        </div>
      </header>
      <el-table v-loading="loading" :data="filteredOps" border stripe size="small">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="op_type" label="类型" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.op_type === 'pin'" type="danger" size="small">置顶</el-tag>
            <el-tag v-else-if="row.op_type === 'block'" type="info" size="small">屏蔽</el-tag>
            <el-tag v-else type="warning" size="small">人工</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="keyword" label="关键词" min-width="100" />
        <el-table-column prop="display_title" label="展示文案" min-width="100" />
        <el-table-column prop="pin_rank" label="排位" width="60" />
        <el-table-column prop="badge" label="角标" width="56" />
        <el-table-column label="启用" width="56">
          <template #default="{ row }">{{ row.enabled ? "是" : "否" }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="onDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '编辑规则' : '新增规则'"
      width="480px"
      destroy-on-close
    >
      <el-form label-width="88px">
        <el-form-item label="类型" required>
          <el-select v-model="form.op_type" style="width: 100%">
            <el-option label="置顶" value="pin" />
            <el-option label="人工词条" value="manual" />
            <el-option label="屏蔽" value="block" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键词" required>
          <el-input v-model="form.keyword" placeholder="匹配/屏蔽用" />
        </el-form-item>
        <el-form-item label="展示文案">
          <el-input v-model="form.display_title" placeholder="留空则用关键词" />
        </el-form-item>
        <el-form-item v-if="form.op_type !== 'block'" label="排位">
          <el-input-number v-model="form.pin_rank" :min="1" :max="20" />
        </el-form-item>
        <el-form-item v-if="form.op_type !== 'block'" label="角标">
          <el-select v-model="form.badge" clearable style="width: 100%">
            <el-option label="无" value="" />
            <el-option label="热" value="热" />
            <el-option label="新" value="新" />
            <el-option label="荐" value="荐" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import {
  adminBoostHotSearchRedis,
  adminCreateHotSearchOp,
  adminDeleteHotSearchOp,
  adminHotSearchDashboard,
  adminQuickHotSearchOp,
  adminReorderHotSearch,
  adminResetHotSearchDisplayOrder,
  adminRemoveHotSearchRedis,
  adminUpdateHotSearchOp
} from "@/api/admin";
import { ElMessage, ElMessageBox } from "element-plus";

export default {
  data() {
    return {
      boardLoading: false,
      reorderSaving: false,
      dragIndex: null,
      overIndex: null,
      loading: false,
      saving: false,
      merged: [],
      redisRows: [],
      rows: [],
      customOrder: false,
      opFilter: "all",
      opFilters: [
        { id: "all", label: "全部" },
        { id: "pin", label: "置顶" },
        { id: "manual", label: "人工" },
        { id: "block", label: "屏蔽" }
      ],
      dialogVisible: false,
      editingId: null,
      form: this.emptyForm()
    };
  },
  computed: {
    filteredOps() {
      if (this.opFilter === "all") return this.rows;
      return this.rows.filter(r => r.op_type === this.opFilter);
    }
  },
  created() {
    this.loadDashboard();
  },
  methods: {
    emptyForm() {
      return {
        op_type: "pin",
        keyword: "",
        display_title: "",
        badge: "",
        pin_rank: 1,
        enabled: true
      };
    },
    rankClass(rank) {
      if (rank === 1) return "adm-hot-list__rank--1";
      if (rank === 2) return "adm-hot-list__rank--2";
      if (rank === 3) return "adm-hot-list__rank--3";
      return "";
    },
    mergedItemKey(item, index) {
      return `m_${item.keyword || item.title || index}`;
    },
    onDragStart(index, ev) {
      this.dragIndex = index;
      this.overIndex = index;
      if (ev && ev.dataTransfer) {
        ev.dataTransfer.effectAllowed = "move";
        ev.dataTransfer.setData("text/plain", String(index));
      }
    },
    onDragOver(index) {
      if (this.dragIndex === null) return;
      this.overIndex = index;
    },
    onDragEnd() {
      this.dragIndex = null;
      this.overIndex = null;
    },
    async onDrop(index) {
      if (this.dragIndex === null || this.dragIndex === index) {
        this.onDragEnd();
        return;
      }
      const list = this.merged.slice();
      const [moved] = list.splice(this.dragIndex, 1);
      list.splice(index, 0, moved);
      this.merged = list.map((it, i) => ({ ...it, rank: i + 1 }));
      this.onDragEnd();
      await this.saveMergedOrder();
    },
    async saveMergedOrder() {
      this.reorderSaving = true;
      try {
        const items = this.merged.map(it => ({
          keyword: it.keyword || it.title,
          title: it.title,
          op_id: it.op_id || 0,
          source: it.source || "auto"
        }));
        await adminReorderHotSearch(items);
        ElMessage.success("展示顺序已保存");
        await this.loadDashboard();
      } catch (e) {
        ElMessage.error((e && e.message) || "保存排序失败");
        await this.loadDashboard();
      } finally {
        this.reorderSaving = false;
      }
    },
    async resetDisplayOrder() {
      await ElMessageBox.confirm(
        "恢复默认排序？将清除自定义拖拽顺序，按置顶规则 + Redis 热度重新合并",
        "确认"
      );
      await adminResetHotSearchDisplayOrder();
      ElMessage.success("已恢复默认排序");
      await this.loadDashboard();
    },
    sourceLabel(source) {
      if (source === "pin") return "置顶";
      if (source === "manual") return "人工";
      return "自动";
    },
    cancelLabel(source) {
      if (source === "pin") return "取消置顶";
      if (source === "manual") return "取消人工";
      if (source === "block") return "取消屏蔽";
      return "撤销";
    },
    isPinned(row) {
      return !!(row.pinned || row.op_type === "pin");
    },
    isManual(row) {
      return !!(row.manual || row.op_type === "manual");
    },
    isBlocked(row) {
      return !!(row.blocked || row.op_type === "block");
    },
    resolveOpId(row) {
      if (row.op_id) return row.op_id;
      const kw = this.keywordFromRow(row);
      const hit = this.rows.find(
        r =>
          String(r.keyword || "").trim() === kw ||
          String(r.keyword || "").trim() === String(row.keyword || "").trim()
      );
      return hit ? hit.id : null;
    },
    formatScore(score) {
      const n = Number(score);
      if (!Number.isFinite(n)) return "0";
      return n % 1 === 0 ? String(n) : n.toFixed(1);
    },
    async loadDashboard() {
      this.boardLoading = true;
      this.loading = true;
      try {
        const body = await adminHotSearchDashboard(10, 30);
        const data = body.data || {};
        this.merged = data.merged || [];
        this.redisRows = data.redis || [];
        this.rows = data.ops || [];
        this.customOrder = !!data.custom_order;
      } finally {
        this.boardLoading = false;
        this.loading = false;
      }
    },
    keywordFromRow(row) {
      return String(row.title || row.keyword || "").trim();
    },
    async cancelOpById(opId, title, actionLabel) {
      const label = actionLabel || "干预";
      const name = String(title || "").trim() || "该词条";
      await ElMessageBox.confirm(`确定${label}「${name}」？`, "确认");
      await adminDeleteHotSearchOp(opId);
      ElMessage.success(`已${label}`);
      await this.loadDashboard();
    },
    async cancelOp(row, typeLabel) {
      const opId = this.resolveOpId(row);
      if (!opId) {
        ElMessage.warning("未找到对应干预规则");
        return;
      }
      await this.cancelOpById(opId, this.keywordFromRow(row), `取消${typeLabel}`);
    },
    async quickPin(row) {
      const kw = this.keywordFromRow(row);
      await adminQuickHotSearchOp({
        keyword: kw,
        op_type: "pin",
        display_title: kw,
        pin_rank: 1,
        badge: row.badge === "热" ? "热" : "荐"
      });
      ElMessage.success("已置顶");
      await this.loadDashboard();
    },
    async quickManual(row) {
      const kw = this.keywordFromRow(row);
      await adminQuickHotSearchOp({
        keyword: kw,
        op_type: "manual",
        display_title: kw,
        pin_rank: Math.min(10, Math.max(1, row.rank || 1)),
        badge: ""
      });
      ElMessage.success("已加入人工榜");
      await this.loadDashboard();
    },
    async quickBlock(row) {
      const kw = this.keywordFromRow(row);
      await ElMessageBox.confirm(`屏蔽「${kw}」？该词将不再进入自动榜`, "确认屏蔽");
      await adminQuickHotSearchOp({ keyword: kw, op_type: "block" });
      ElMessage.success("已屏蔽");
      await this.loadDashboard();
    },
    async boostRow(row) {
      const kw = this.keywordFromRow(row);
      await adminBoostHotSearchRedis(kw, 5);
      ElMessage.success("热度 +5");
      await this.loadDashboard();
    },
    async removeRedis(row) {
      const kw = this.keywordFromRow(row);
      await ElMessageBox.confirm(
        `从 Redis 移除「${kw}」？仅删除自动统计，不影响已有干预规则`,
        "确认移除"
      );
      await adminRemoveHotSearchRedis(kw);
      ElMessage.success("已从 Redis 移除");
      await this.loadDashboard();
    },
    openCreate() {
      this.editingId = null;
      this.form = this.emptyForm();
      this.dialogVisible = true;
    },
    openEdit(row) {
      this.editingId = row.id;
      this.form = {
        op_type: row.op_type,
        keyword: row.keyword,
        display_title: row.display_title || "",
        badge: row.badge || "",
        pin_rank: row.pin_rank || 1,
        enabled: !!row.enabled
      };
      this.dialogVisible = true;
    },
    async onSave() {
      if (!String(this.form.keyword).trim()) {
        ElMessage.warning("请填写关键词");
        return;
      }
      this.saving = true;
      try {
        const payload = { ...this.form };
        if (payload.op_type === "block") {
          payload.pin_rank = 0;
          payload.badge = "";
        }
        if (this.editingId) {
          await adminUpdateHotSearchOp(this.editingId, payload);
        } else {
          await adminCreateHotSearchOp(payload);
        }
        ElMessage.success("已保存");
        this.dialogVisible = false;
        await this.loadDashboard();
      } finally {
        this.saving = false;
      }
    },
    async onDelete(row) {
      await ElMessageBox.confirm(`确定删除规则 #${row.id}？`, "确认");
      await adminDeleteHotSearchOp(row.id);
      ElMessage.success("已删除");
      await this.loadDashboard();
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/style/mixin";

.adm-panel {
  background: $white;
  border: 1px solid #e3e5e7;
  border-radius: 8px;
  padding: 20px;
}
.adm-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  h2 {
    margin: 0;
    @include sc(18px, #18191c);
  }
}
.adm-panel__actions {
  display: flex;
  gap: 8px;
}
.adm-panel__desc {
  margin: 0 0 16px;
  @include sc(12px, #9499a0);
}
.adm-board {
  display: grid;
  grid-template-columns: 1fr 1.2fr;
  gap: 16px;
  margin-bottom: 20px;
  min-height: 280px;
}
.adm-board__col {
  border: 1px solid #e3e5e7;
  border-radius: 8px;
  padding: 14px 16px;
  background: #fafbfc;
}
.adm-board__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 12px;
  h3 {
    margin: 0;
    @include sc(15px, #18191c);
    font-weight: 600;
  }
}
.adm-board__head-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}
.adm-board__tag {
  padding: 2px 8px;
  border-radius: 4px;
  background: #e3f3ff;
  @include sc(11px, $blue);
}
.adm-board__reset {
  border: none;
  background: transparent;
  cursor: pointer;
  padding: 0;
  @include sc(12px, #f25d8e);
  &:hover {
    text-decoration: underline;
  }
}
.adm-board__hint {
  @include sc(11px, #9499a0);
}
.adm-board__empty {
  margin: 24px 0;
  text-align: center;
  @include sc(13px, #9499a0);
}
.adm-hot-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.adm-hot-list__row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 0;
  border-bottom: 1px solid #eef0f2;
  transition: background 0.15s ease;
  &:last-child {
    border-bottom: none;
  }
  &--dragging {
    opacity: 0.45;
  }
  &--over {
    background: #e3f3ff;
  }
}
.adm-hot-list__handle {
  flex: 0 0 18px;
  cursor: grab;
  user-select: none;
  text-align: center;
  line-height: 1;
  letter-spacing: -2px;
  @include sc(14px, #9499a0);
  &:active {
    cursor: grabbing;
  }
}
.adm-board__saving {
  margin: 8px 0 0;
  @include sc(12px, $blue);
  &--idle {
    color: #9499a0;
  }
}
.adm-board__col--merged {
  position: relative;
}
.adm-hot-list__rank {
  flex: 0 0 22px;
  font-weight: 600;
  @include sc(14px, #9499a0);
  font-variant-numeric: tabular-nums;
  &--1 {
    color: #ff6699;
  }
  &--2 {
    color: #ff7f24;
  }
  &--3 {
    color: #ffb027;
  }
}
.adm-hot-list__title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  @include sc(14px, #18191c);
}
.adm-hot-list__badge {
  flex-shrink: 0;
  padding: 0 4px;
  border-radius: 2px;
  font-size: 10px;
  line-height: 16px;
  font-weight: 600;
  color: #fff;
  &--hot {
    background: #ff6699;
  }
  &--new {
    background: #ffb027;
  }
  &--rec {
    background: #00a1d6;
  }
}
.adm-hot-list__src {
  flex-shrink: 0;
  padding: 2px 6px;
  border-radius: 4px;
  @include sc(11px, #61666d);
  background: #eef0f2;
  &--pin {
    color: #f25d8e;
    background: #ffeef4;
  }
  &--manual {
    color: #e6a23c;
    background: #fdf6ec;
  }
  &--auto {
    color: #00a1d6;
    background: #e3f3ff;
  }
}
.adm-hot-list__revoke {
  flex-shrink: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  padding: 0 4px;
  @include sc(12px, #f25d8e);
  &:hover {
    color: #ff85ad;
  }
}
.adm-redis-table-wrap {
  overflow-x: auto;
}
.adm-redis-table {
  width: 100%;
  border-collapse: collapse;
  @include sc(13px, #18191c);
  th,
  td {
    padding: 8px 6px;
    text-align: left;
    border-bottom: 1px solid #eef0f2;
  }
  th {
    @include sc(12px, #9499a0);
    font-weight: 500;
  }
}
.adm-redis-table__rank {
  width: 28px;
  font-weight: 600;
  color: #9499a0;
}
.adm-redis-table__kw {
  font-weight: 500;
}
.adm-redis-table__norm {
  @include sc(11px, #9499a0);
  margin-top: 2px;
}
.adm-redis-table__score {
  font-variant-numeric: tabular-nums;
  color: $blue;
  font-weight: 600;
}
.adm-redis-table__ops {
  white-space: nowrap;
  .el-button + .el-button {
    margin-left: 0;
  }
}
.adm-state {
  display: inline-block;
  padding: 2px 6px;
  border-radius: 4px;
  @include sc(11px, #61666d);
  &--block {
    background: #f0f0f0;
    color: #909399;
  }
  &--pin {
    background: #ffeef4;
    color: #f25d8e;
  }
  &--manual {
    background: #fdf6ec;
    color: #e6a23c;
  }
  &--auto {
    background: #e3f3ff;
    color: $blue;
  }
}
.adm-rules__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  h3 {
    margin: 0;
    @include sc(15px, #18191c);
  }
}
.adm-rules__filter {
  display: flex;
  gap: 6px;
}
.adm-rules__tab {
  border: 1px solid #e3e5e7;
  background: $white;
  border-radius: 4px;
  padding: 4px 10px;
  cursor: pointer;
  @include sc(12px, #61666d);
  &--on {
    border-color: $blue;
    color: $blue;
    background: #e3f3ff;
  }
}
@media (max-width: 960px) {
  .adm-board {
    grid-template-columns: 1fr;
  }
}
</style>
