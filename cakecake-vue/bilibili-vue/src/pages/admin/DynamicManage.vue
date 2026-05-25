<template>
  <div class="adm-panel">
    <div class="adm-panel__head">
      <h2>
        动态管理
        <el-tag type="info" size="small" class="adm-badge">无需审核</el-tag>
      </h2>
      <div class="adm-toolbar">
        <el-input
          v-model="keyword"
          placeholder="搜索标题或正文"
          clearable
          style="width: 240px"
          @keyup.enter="onSearch"
        />
        <el-button type="primary" @click="onSearch">搜索</el-button>
      </div>
    </div>

    <p class="adm-hint">
      用户发布的图文动态即时公开，不在此审核。运营可查看内容并删除违规动态（同步删除数据库记录与 OSS 图片）。
    </p>

    <div class="adm-table-wrap">
      <el-table v-loading="loading" :data="rows" border stripe class="adm-dyn-table">
        <el-table-column prop="id" label="ID" width="64" />
        <el-table-column label="图片" width="108">
          <template #default="{ row }">
            <img v-if="row.cover_url" :src="row.cover_url" class="adm-thumb" alt="" />
            <span v-else class="adm-no-cover">无图</span>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="120" show-overflow-tooltip />
        <el-table-column prop="content" label="正文" min-width="160" show-overflow-tooltip />
        <el-table-column prop="uploader_name" label="作者" width="100" show-overflow-tooltip />
        <el-table-column label="互动" width="100" align="center">
          <template #default="{ row }">
            <span class="adm-stat">赞 {{ row.like_count || 0 }}</span>
            <span class="adm-stat">评 {{ row.comment_count || 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column label="发布时间" min-width="168" show-overflow-tooltip>
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" min-width="140" align="center">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
            <el-button link type="danger" @click="onDelete(row)">删除</el-button>
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
      :title="detail ? `动态 #${detail.id}` : '动态详情'"
      width="720px"
      destroy-on-close
      @closed="detail = null"
    >
      <template v-if="detail">
        <div class="adm-review">
          <div class="adm-review__meta">
            <h3>{{ detail.title || "（无标题）" }}</h3>
            <p><strong>作者：</strong>{{ detail.uploader_name || detail.user_id }}</p>
            <p><strong>点赞 / 评论：</strong>{{ detail.like_count || 0 }} / {{ detail.comment_count || 0 }}</p>
            <p><strong>发布时间：</strong>{{ formatTime(detail.created_at) }}</p>
            <p v-if="publicLink">
              <strong>前台链接：</strong>
              <a :href="publicLink" target="_blank" rel="noopener noreferrer">{{ publicLink }}</a>
            </p>
            <p class="adm-review__content"><strong>正文：</strong>{{ detail.content || "（无）" }}</p>
          </div>
          <div v-if="detail.images && detail.images.length" class="adm-dyn-images">
            <img
              v-for="(url, i) in detail.images"
              :key="i"
              :src="url"
              class="adm-dyn-img"
              alt=""
            />
          </div>
        </div>
      </template>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button
          v-if="detail"
          type="danger"
          :loading="acting"
          @click="onDelete(detail)"
        >
          删除动态
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import {
  adminDeleteDynamic,
  adminGetDynamic,
  adminListDynamics
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
      keyword: "",
      detailVisible: false,
      detail: null
    };
  },
  computed: {
    publicLink() {
      if (!this.detail || !this.detail.id) return "";
      const base = window.location.origin + window.location.pathname;
      return `${base}#/minibili/dynamic/${this.detail.id}`;
    }
  },
  created() {
    this.load();
  },
  methods: {
    async load() {
      this.loading = true;
      try {
        const body = await adminListDynamics({
          page: this.page,
          page_size: this.pageSize,
          q: this.keyword.trim()
        });
        const d = body.data || {};
        this.rows = d.items || [];
        this.page = d.page || 1;
        this.total = d.total || 0;
        this.totalPages = d.total_pages || 1;
      } finally {
        this.loading = false;
      }
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
    formatTime(t) {
      if (!t) return "—";
      const d = new Date(t);
      if (Number.isNaN(d.getTime())) return String(t);
      const pad = (x) => String(x).padStart(2, "0");
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
    },
    async openDetail(row) {
      const body = await adminGetDynamic(row.id);
      this.detail = body.data;
      this.detailVisible = true;
    },
    async onDelete(row) {
      await ElMessageBox.confirm(
        `确定删除动态 #${row.id}？将同步删除数据库记录、评论、点赞及 OSS 上的全部图片，且不可恢复。`,
        "确认删除",
        { type: "warning" }
      );
      this.acting = true;
      try {
        await adminDeleteDynamic(row.id);
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
  margin-bottom: 12px;
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
.adm-hint {
  margin: 0 0 16px;
  @include sc(13px, #61666d);
  line-height: 1.5;
}
.adm-table-wrap {
  width: 100%;
  overflow-x: auto;
}
.adm-dyn-table {
  width: 100%;
  min-width: 880px;
}
.adm-thumb {
  width: 96px;
  height: 54px;
  object-fit: cover;
  border-radius: 4px;
}
.adm-no-cover {
  @include sc(12px, #99a2aa);
}
.adm-stat {
  display: block;
  @include sc(12px, #61666d);
}
.adm-pager {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-top: 16px;
  @include sc(13px, #61666d);
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
  a {
    color: $blue;
    word-break: break-all;
  }
}
.adm-review__content {
  white-space: pre-wrap;
  word-break: break-word;
}
.adm-dyn-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
}
.adm-dyn-img {
  width: 120px;
  height: 120px;
  object-fit: cover;
  border-radius: 6px;
  border: 1px solid #e3e5e7;
}
</style>
