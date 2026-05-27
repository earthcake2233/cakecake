<template>
  <div class="adm-panel">
    <div class="adm-panel__head">
      <h2>
        专栏审核
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
      <el-table v-loading="loading" :data="rows" border stripe class="adm-article-table">
        <el-table-column prop="id" label="ID" width="64" />
        <el-table-column label="封面" width="108">
          <template #default="{ row }">
            <img v-if="row.cover_url" :src="row.cover_url" class="adm-thumb" alt="" />
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="160" show-overflow-tooltip />
        <el-table-column prop="uploader_name" label="作者" width="100" show-overflow-tooltip />
        <el-table-column label="状态" width="88" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="提交时间" width="88" align="center" class-name="adm-col-time">
          <template #default="{ row }">
            <div v-if="formatTimeParts(row.created_at)" class="adm-time-cell">
              <span class="adm-time-cell__date">{{ formatTimeParts(row.created_at).date }}</span>
              <span class="adm-time-cell__time">{{ formatTimeParts(row.created_at).time }}</span>
            </div>
            <span v-else>—</span>
          </template>
        </el-table-column>
        <el-table-column
          label="操作"
          :width="actionColWidth"
          align="center"
          class-name="adm-col-actions"
        >
          <template #default="{ row }">
            <div class="adm-row-actions">
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
            </div>
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
      :title="detail ? `审核 #${detail.id}` : '专栏审核'"
      width="760px"
      destroy-on-close
      @closed="detail = null"
    >
      <template v-if="detail">
        <div class="adm-review">
          <div class="adm-review__media">
            <img
              v-if="detail.cover_url"
              :src="detail.cover_url"
              class="adm-review__cover"
              alt=""
            />
          </div>
          <div class="adm-review__meta">
            <h3>{{ detail.title }}</h3>
            <p><strong>作者：</strong>{{ detail.uploader_name || detail.user_id }}</p>
            <p><strong>状态：</strong>{{ statusLabel(detail.status) }}</p>
            <p v-if="detail.fail_reason"><strong>原因：</strong>{{ detail.fail_reason }}</p>
          </div>
          <div class="adm-review__body">
            <strong>正文预览</strong>
            <div class="adm-review__html" v-html="detail.body_html || ''" />
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
          删除专栏
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="rejectVisible" title="驳回专栏" width="480px" destroy-on-close>
      <el-form label-width="72px">
        <el-form-item label="驳回理由" required>
          <el-input
            v-model="rejectReason"
            type="textarea"
            :rows="4"
            maxlength="500"
            show-word-limit
            placeholder="请填写驳回原因，作者可在稿件管理中看到"
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
  adminApproveArticle,
  adminDeleteArticle,
  adminGetArticle,
  adminListArticles,
  adminRejectArticle
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
  computed: {
    actionColWidth() {
      const wide =
        this.statusFilter === "pending_review" || this.statusFilter === "all";
      return wide ? 148 : 108;
    }
  },
  created() {
    this.load();
  },
  methods: {
    async load() {
      this.loading = true;
      try {
        const body = await adminListArticles({
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
        draft: "草稿"
      };
      return m[st] || st;
    },
    statusTagType(st) {
      if (st === "pending_review") return "warning";
      if (st === "published") return "success";
      if (st === "rejected") return "danger";
      return "info";
    },
    canDelete(row) {
      const st = row && row.status;
      return st === "published" || st === "rejected";
    },
    formatTime(t) {
      const p = this.formatTimeParts(t);
      if (!p) return "—";
      return `${p.date} ${p.time}`;
    },
    formatTimeParts(t) {
      if (!t) return null;
      const d = new Date(t);
      if (Number.isNaN(d.getTime())) return null;
      const pad = (x) => String(x).padStart(2, "0");
      return {
        date: `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`,
        time: `${pad(d.getHours())}:${pad(d.getMinutes())}`
      };
    },
    async openDetail(row) {
      const body = await adminGetArticle(row.id);
      this.detail = body.data;
      this.detailVisible = true;
    },
    async onApprove(row) {
      await ElMessageBox.confirm(`确定通过并发布「${row.title}」？`, "确认通过");
      this.acting = true;
      try {
        await adminApproveArticle(row.id);
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
        await adminRejectArticle(row.id, reason);
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
        `确定删除「${row.title}」？将同步删除数据库记录与 OSS 上的封面/正文图片，且不可恢复。`,
        "确认删除",
        { type: "warning" }
      );
      this.acting = true;
      try {
        await adminDeleteArticle(row.id);
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
  overflow: visible;
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
  max-width: 100%;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}
.adm-article-table {
  width: 100%;
  table-layout: fixed;
}
.adm-article-table :deep(.adm-col-time .cell),
.adm-article-table :deep(.adm-col-actions .cell) {
  padding-left: 6px;
  padding-right: 6px;
}
.adm-row-actions {
  display: inline-flex;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: center;
  gap: 0;
  white-space: nowrap;
}
.adm-row-actions :deep(.el-button.is-link) {
  padding: 0 4px;
  margin: 0;
  height: auto;
}
.adm-time-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  line-height: 1.35;
  gap: 2px;
  @include sc(12px, #61666d);
}
.adm-time-cell__date,
.adm-time-cell__time {
  white-space: nowrap;
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
.adm-review__body {
  margin-top: 16px;
  strong {
    display: block;
    margin-bottom: 8px;
    @include sc(13px, #18191c);
  }
}
.adm-review__html {
  max-height: 360px;
  overflow: auto;
  padding: 12px;
  border: 1px solid #e3e5e7;
  border-radius: 6px;
  background: #fafafa;
  @include sc(14px, #18191c);
  line-height: 1.6;
  :deep(img) {
    max-width: 100%;
    height: auto;
  }
  :deep(pre) {
    overflow-x: auto;
  }
}
</style>
