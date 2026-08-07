<template>
  <div class="adm-feedback">
    <header class="adm-feedback__head">
      <h2>AI 助手反馈</h2>
      <button type="button" class="adm-feedback__refresh" @click="load">
        刷新
      </button>
    </header>
    <p v-if="loading" class="adm-feedback__hint">加载中…</p>
    <p v-else-if="!items.length" class="adm-feedback__hint">暂无反馈数据</p>
    <table v-else class="adm-feedback__table">
      <thead>
        <tr>
          <th>时间</th>
          <th>用户 ID</th>
          <th>反馈</th>
          <th>消息内容</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="it in items" :key="it.id">
          <td>{{ it.created_at }}</td>
          <td>{{ it.user_id }}</td>
          <td>
            <span
              class="adm-feedback__badge"
              :class="it.feedback === 'like' ? 'is-like' : 'is-dislike'"
            >
              {{ it.feedback === "like" ? "有用" : "没用" }}
            </span>
          </td>
          <td class="adm-feedback__content">
            {{ preview(it.message_content) }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script>
import { adminListAgentFeedbacks } from "@/api/admin";

export default {
  name: "AgentFeedbacks",
  data() {
    return {
      items: [],
      loading: false
    };
  },
  mounted() {
    void this.load();
  },
  methods: {
    async load() {
      this.loading = true;
      try {
        const r = await adminListAgentFeedbacks();
        const d = (r && r.data) || {};
        this.items = d.items || [];
      } finally {
        this.loading = false;
      }
    },
    preview(s) {
      const text = String(s || "").replace(/\s+/g, " ").trim();
      return text.length > 120 ? text.slice(0, 120) + "…" : text || "(空)";
    }
  }
};
</script>

<style scoped>
.adm-feedback {
  padding: 20px;
  box-sizing: border-box;
}
.adm-feedback__head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}
.adm-feedback__refresh {
  padding: 4px 14px;
  border: 1px solid #00aeec;
  border-radius: 6px;
  background: transparent;
  color: #00aeec;
  cursor: pointer;
}
.adm-feedback__hint {
  color: #9499a0;
}
.adm-feedback__table {
  width: 100%;
  border-collapse: collapse;
  background: #fff;
}
.adm-feedback__table th,
.adm-feedback__table td {
  border: 1px solid #e3e5e7;
  padding: 8px 10px;
  text-align: left;
  font-size: 13px;
  vertical-align: top;
}
.adm-feedback__table th {
  background: #f5f6f7;
}
.adm-feedback__badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 10px;
  font-size: 12px;
}
.adm-feedback__badge.is-like {
  background: rgba(0, 174, 236, 0.12);
  color: #00aeec;
}
.adm-feedback__badge.is-dislike {
  background: rgba(240, 64, 64, 0.1);
  color: #f04040;
}
.adm-feedback__content {
  max-width: 520px;
  word-break: break-all;
}
</style>
