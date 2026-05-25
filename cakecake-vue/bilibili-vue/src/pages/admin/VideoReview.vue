<template>
  <div class="adm-panel">
    <div class="adm-panel__head">
      <h2>
        视频审核
        <el-tag v-if="pendingCount > 0" type="warning" size="small" class="adm-badge">
          {{ pendingCount }} 待审
        </el-tag>
      </h2>
      <div class="adm-toolbar">
        <el-input
          v-model="keyword"
          placeholder="搜索标题"
          clearable
          style="width: 220px"
          @keyup.enter="onSearch"
        />
        <el-button type="primary" @click="onSearch">搜索</el-button>
      </div>
    </div>

    <div class="adm-filters">
      <button
        v-for="f in statusFilters"
        :key="f.value"
        type="button"
        class="adm-filter"
        :class="{ 'adm-filter--on': statusFilter === f.value }"
        @click="setStatusFilter(f.value)"
      >
        {{ f.label }}
      </button>
    </div>

    <div class="adm-table-wrap">
    <el-table v-loading="loading" :data="rows" border stripe class="adm-video-table">
      <el-table-column prop="id" label="ID" width="64" />
      <el-table-column label="封面" width="108">
        <template #default="{ row }">
          <img v-if="row.cover_url" :src="row.cover_url" class="adm-thumb" alt="" />
        </template>
      </el-table-column>
      <el-table-column prop="title" label="标题" min-width="140" show-overflow-tooltip />
      <el-table-column prop="uploader_name" label="UP主" width="100" show-overflow-tooltip />
      <el-table-column prop="zone" label="分区" width="72" show-overflow-tooltip />
      <el-table-column label="时长" width="72" align="center">
        <template #default="{ row }">{{ formatDuration(row.duration_sec) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="88" align="center">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="提交时间" min-width="168" show-overflow-tooltip>
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" min-width="176" align="center">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail(row)">详情</el-button>
          <template v-if="row.status === 'pending_review'">
            <el-button link type="success" @click="onApprove(row)">通过</el-button>
            <el-button link type="danger" @click="openReject(row)">驳回</el-button>
          </template>
          <el-button
            v-if="canDelete(row)"
            link
            type="danger"
            @click="onDelete(row)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    </div>

    <div v-if="totalPages > 1" class="adm-pager">
      <el-button :disabled="page <= 1" @click="goPage(page - 1)">上一页</el-button>
      <span>{{ page }} / {{ totalPages }}（共 {{ total }} 条）</span>
      <el-button :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</el-button>
    </div>

    <el-dialog
      v-model="detailVisible"
      :title="detail ? `审核 #${detail.id}` : '视频审核'"
      width="720px"
      destroy-on-close
      @closed="detail = null"
    >
      <template v-if="detail">
        <div class="adm-review">
          <div class="adm-review__media">
            <video
              v-if="detail.video_url"
              :src="detail.video_url"
              :poster="detail.cover_url"
              controls
              playsinline
              class="adm-review__video"
            />
            <img v-else-if="detail.cover_url" :src="detail.cover_url" class="adm-review__cover" alt="" />
          </div>
          <div class="adm-review__meta">
            <h3>{{ detail.title }}</h3>
            <p><strong>UP主：</strong>{{ detail.uploader_name || detail.user_id }}</p>
            <p><strong>分区：</strong>{{ detail.zone || "—" }}</p>
            <p><strong>时长：</strong>{{ formatDuration(detail.duration_sec) }}</p>
            <p><strong>状态：</strong>{{ statusLabel(detail.status) }}</p>
            <p v-if="detail.fail_reason"><strong>原因：</strong>{{ detail.fail_reason }}</p>
            <p class="adm-review__desc"><strong>简介：</strong>{{ detail.description || "（无）" }}</p>
          </div>
        </div>
      </template>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <template v-if="detail && detail.status === 'pending_review'">
          <el-button type="danger" @click="openReject(detail)">驳回</el-button>
          <el-button type="primary" :loading="acting" @click="onApprove(detail)">通过并发布</el-button>
        </template>
        <el-button
          v-if="detail && canDelete(detail)"
          type="danger"
          plain
          :loading="acting"
          @click="onDelete(detail)"
        >
          删除视频
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="rejectVisible" title="驳回视频" width="480px" destroy-on-close>
      <el-form label-width="72px">
        <el-form-item label="驳回理由" required>
          <el-input
            v-model="rejectReason"
            type="textarea"
            :rows="4"
            maxlength="500"
            show-word-limit
            placeholder="请填写驳回原因，UP主可在稿件管理中看到"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rejectVisible = false">取消</el-button>
        <el-button type="danger" :loading="acting" @click="confirmReject">确认驳回</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import {
  adminApproveVideo,
  adminDeleteVideo,
  adminGetVideo,
  adminListVideos,
  adminRejectVideo
} from "@/api/admin";
import { ElMessage, ElMessageBox } from "element-plus";

export default {
  data() {
    return {
      loading: false,
      acting: false,
      rows: [],
      page: 1,
      pageSize: 20,
      total: 0,
      totalPages: 1,
      pendingCount: 0,
      statusFilter: "pending_review",
      keyword: "",
      statusFilters: [
        { value: "pending_review", label: "待审核" },
        { value: "published", label: "已通过" },
        { value: "rejected", label: "已驳回" },
        { value: "all", label: "全部" }
      ],
      detailVisible: false,
      detail: null,
      rejectVisible: false,
      rejectTarget: null,
      rejectReason: ""
    };
  },
  created() {
    this.load();
  },
  methods: {
    async load() {
      this.loading = true;
      try {
        const body = await adminListVideos({
          page: this.page,
          page_size: this.pageSize,
          status: this.statusFilter,
          q: this.keyword.trim()
        });
        const d = body.data || {};
        this.rows = d.items || [];
        this.page = d.page || 1;
        this.total = d.total || 0;
        this.totalPages = d.total_pages || 1;
        this.pendingCount = d.pending_count || 0;
      } finally {
        this.loading = false;
      }
    },
    setStatusFilter(v) {
      if (this.statusFilter === v) return;
      this.statusFilter = v;
      this.page = 1;
      void this.load();
    },
    onSearch() {
      this.page = 1;
      void this.load();
    },
    goPage(p) {
      if (p < 1 || p > this.totalPages || p === this.page) return;
      this.page = p;
      void this.load();
    },
    statusLabel(st) {
      const m = {
        pending_review: "待审核",
        published: "已发布",
        rejected: "已驳回",
        processing: "转码中",
        failed: "转码失败",
        draft: "草稿"
      };
      return m[st] || st;
    },
    statusTagType(st) {
      if (st === "pending_review") return "warning";
      if (st === "published") return "success";
      if (st === "rejected" || st === "failed") return "danger";
      return "info";
    },
    canDelete(row) {
      const st = row && row.status;
      return st === "published" || st === "rejected";
    },
    formatDuration(sec) {
      const n = Math.max(0, Math.floor(Number(sec) || 0));
      const m = Math.floor(n / 60);
      const s = n % 60;
      return `${m}:${String(s).padStart(2, "0")}`;
    },
    formatTime(t) {
      if (!t) return "—";
      const d = new Date(t);
      if (Number.isNaN(d.getTime())) return String(t);
      const pad = (x) => String(x).padStart(2, "0");
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
    },
    async openDetail(row) {
      const body = await adminGetVideo(row.id);
      this.detail = body.data;
      this.detailVisible = true;
    },
    async onApprove(row) {
      await ElMessageBox.confirm(`确定通过并发布「${row.title}」？`, "确认通过");
      this.acting = true;
      try {
        await adminApproveVideo(row.id);
        ElMessage.success("已通过并发布");
        this.detailVisible = false;
        await this.load();
      } finally {
        this.acting = false;
      }
    },
    openReject(row) {
      this.rejectTarget = row;
      this.rejectReason = "";
      this.rejectVisible = true;
    },
    async confirmReject() {
      const reason = this.rejectReason.trim();
      if (!reason) {
        ElMessage.warning("请填写驳回理由");
        return;
      }
      const row = this.rejectTarget;
      if (!row) return;
      this.acting = true;
      try {
        await adminRejectVideo(row.id, reason);
        ElMessage.success("已驳回");
        this.rejectVisible = false;
        this.detailVisible = false;
        await this.load();
      } finally {
        this.acting = false;
      }
    },
    async onDelete(row) {
      await ElMessageBox.confirm(
        `确定删除「${row.title}」？将同步删除数据库记录与 OSS 上的视频/封面文件，且不可恢复。`,
        "确认删除",
        { type: "warning" }
      );
      this.acting = true;
      try {
        await adminDeleteVideo(row.id);
        ElMessage.success("已删除");
        this.detailVisible = false;
        await this.load();
      } finally {
        this.acting = false;
      }
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
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 12px;
  h2 {
    margin: 0;
    @include sc(18px, #18191c);
    display: flex;
    align-items: center;
    gap: 8px;
  }
}
.adm-toolbar {
  display: flex;
  gap: 8px;
}
.adm-filters {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.adm-filter {
  border: 1px solid #e3e5e7;
  background: #f6f7f8;
  border-radius: 6px;
  padding: 6px 14px;
  cursor: pointer;
  @include sc(13px, #61666d);
  &:hover {
    color: $blue;
    border-color: #c9ccd0;
  }
}
.adm-filter--on {
  color: $blue;
  background: #e3f3ff;
  border-color: $blue;
  font-weight: 600;
}
.adm-table-wrap {
  width: 100%;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}
.adm-video-table {
  width: 100%;
  min-width: 920px;
}
.adm-thumb {
  width: 96px;
  height: 54px;
  object-fit: cover;
  border-radius: 4px;
}
.adm-pager {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-top: 16px;
  @include sc(13px, #61666d);
}
.adm-review__media {
  margin-bottom: 16px;
}
.adm-review__video {
  width: 100%;
  max-height: 360px;
  background: #000;
  border-radius: 6px;
}
.adm-review__cover {
  width: 100%;
  max-height: 200px;
  object-fit: contain;
  border-radius: 6px;
}
.adm-review__meta {
  h3 {
    margin: 0 0 12px;
    @include sc(16px, #18191c);
  }
  p {
    margin: 0 0 8px;
    @include sc(13px, #61666d);
    line-height: 1.5;
  }
}
.adm-review__desc {
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
