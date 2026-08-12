<template>
  <div class="tdl-manage">
    <header class="tdl-manage__head">
      <div>
        <h2>转码死信</h2>
        <p>转码重试耗尽的任务会进入死信队列并记录于此；可按状态筛选并重新入队。</p>
      </div>
      <div class="tdl-manage__actions">
        <el-select
          v-model="status"
          class="tdl-manage__status"
          placeholder="全部状态"
          @change="reload"
        >
          <el-option label="全部" value="" />
          <el-option label="待处理" value="pending" />
          <el-option label="已重放" value="requeued" />
          <el-option label="已处理" value="processed" />
        </el-select>
        <el-button :loading="loading" @click="reload">刷新</el-button>
      </div>
    </header>

    <el-table :data="items" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="video_id" label="视频ID" width="90" />
      <el-table-column prop="reason" label="失败原因" min-width="220" show-overflow-tooltip />
      <el-table-column prop="retry_count" label="重试次数" width="90" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row)" size="small">{{ statusLabel(row) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="170">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column prop="requeued_count" label="重放次数" width="90" />
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button
            type="primary"
            size="small"
            :loading="requeuingId === row.id"
            :disabled="requeuingId !== null && requeuingId !== row.id"
            @click="requeue(row)"
          >
            重新入队
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      class="tdl-manage__pager"
      layout="total, prev, pager, next"
      :total="total"
      :page-size="pageSize"
      :current-page="page"
      @current-change="changePage"
    />
  </div>
</template>

<script>
import { ElMessage, ElMessageBox } from "element-plus";
import {
  adminListTranscodeDeadLetters,
  adminRequeueTranscodeDeadLetter
} from "@/api/admin";

export default {
  name: "TranscodeDeadLetterManage",
  data() {
    return {
      loading: false,
      items: [],
      total: 0,
      page: 1,
      pageSize: 20,
      status: "",
      requeuingId: null
    };
  },
  created() {
    this.reload();
  },
  methods: {
    async reload() {
      this.loading = true;
      try {
        const r = await adminListTranscodeDeadLetters({
          page: this.page,
          page_size: this.pageSize,
          status: this.status
        });
        this.items = r.data.items || [];
        this.total = r.data.total || 0;
      } catch {
        ElMessage.error("加载转码死信失败");
      } finally {
        this.loading = false;
      }
    },
    changePage(p) {
      this.page = p;
      this.reload();
    },
    statusLabel(row) {
      if (row.requeued_at) return "已重放";
      if (row.processed_at) return "已处理";
      return "待处理";
    },
    statusType(row) {
      if (row.requeued_at) return "warning";
      if (row.processed_at) return "info";
      return "danger";
    },
    formatTime(t) {
      return t ? String(t).replace("T", " ").slice(0, 19) : "—";
    },
    async requeue(row) {
      try {
        await ElMessageBox.confirm(
          `确定将死信 #${row.id}（视频 ${row.video_id}）重新入队吗？视频将回到处理中状态。`,
          "重新入队",
          { type: "warning" }
        );
      } catch {
        return;
      }
      this.requeuingId = row.id;
      try {
        await adminRequeueTranscodeDeadLetter(row.id);
        ElMessage.success("已重新入队");
        await this.reload();
      } catch (e) {
        ElMessage.error(e?.response?.data?.msg || "重新入队失败");
      } finally {
        this.requeuingId = null;
      }
    }
  }
};
</script>

<style scoped>
.tdl-manage__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}
.tdl-manage__head h2 {
  margin: 0 0 4px;
}
.tdl-manage__head p {
  margin: 0;
  color: #888;
  font-size: 13px;
}
.tdl-manage__actions {
  display: flex;
  gap: 8px;
}
.tdl-manage__status {
  width: 140px;
}
.tdl-manage__pager {
  margin-top: 16px;
  justify-content: flex-end;
}
</style>
