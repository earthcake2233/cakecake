<template>
  <div class="adm-panel">
    <!-- ===== 头部：标题 + 描述内嵌 ===== -->
    <div class="syscfg-hero">
      <div class="syscfg-hero__text">
        <h2>运行时配置</h2>
        <p>
          优先从 <code>system_configs</code> 表动态读取，未设置时回退到环境变量默认值。
          保存后 <strong>≤30s</strong> 全节点同步生效。
        </p>
      </div>
      <div class="syscfg-hero__actions">
        <el-tooltip content="从服务器刷新" placement="top">
          <el-button :loading="loading" circle size="small" @click="loadConfigs">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 2v6h-6"/><path d="M3 12a9 9 0 0 1 15.36-6.36L21 8M3 22v-6h6"/><path d="M21 12a9 9 0 0 1-15.36 6.36L3 16"/></svg>
          </el-button>
        </el-tooltip>
        <el-badge :value="changeCount" :hidden="changeCount === 0" class="syscfg-save-badge">
          <el-button type="primary" :loading="saving" :disabled="changeCount === 0" @click="saveConfigs">
            保存配置
          </el-button>
        </el-badge>
      </div>
    </div>

    <div v-loading="loading" class="syscfg-grid">
      <!-- ============ AI 助手 ============ -->
      <section class="syscfg-card">
        <div class="syscfg-card__header">
          <div class="syscfg-card__icon"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2a4 4 0 1 0 0 8 4 4 0 0 0 0-8z"/><path d="M6 21v-2a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v2"/></svg></div>
          <div>
            <h3>AI 助手</h3>
            <p>控制 AI 对话功能的开关、配额与超时策略</p>
          </div>
        </div>
        <div class="syscfg-card__body">
          <div class="syscfg-row" :class="{ 'syscfg-row--changed': cKeyChanged('agent_enabled') }">
            <div class="syscfg-row__info">
              <span class="syscfg-row__label">AI 助手总开关</span>
              <span class="syscfg-row__env">env: AGENT_ENABLED</span>
            </div>
            <div class="syscfg-row__ctrl">
              <el-switch v-model="form.agent_enabled" active-text="开启" inactive-text="关闭" />
              <el-button v-if="cKeyChanged('agent_enabled')" text size="small" class="syscfg-row__reset" @click="resetKey('agent_enabled')">还原</el-button>
            </div>
            <p class="syscfg-row__hint">关闭后所有用户无法发起 AI 对话，已存在的会话将不可用</p>
          </div>

          <div class="syscfg-row" :class="{ 'syscfg-row--changed': cKeyChanged('agent_daily_quota') }">
            <div class="syscfg-row__info">
              <span class="syscfg-row__label">每日对话上限</span>
              <span class="syscfg-row__env">env: AGENT_DAILY_QUOTA=80</span>
            </div>
            <div class="syscfg-row__ctrl">
              <el-input-number v-model.number="form.agent_daily_quota" :min="0" :max="9999" controls-position="right" />
              <span class="syscfg-row__unit">次/天/用户</span>
              <el-button v-if="cKeyChanged('agent_daily_quota')" text size="small" class="syscfg-row__reset" @click="resetKey('agent_daily_quota')">还原</el-button>
            </div>
            <p class="syscfg-row__hint">每个用户每天最多发起的 AI 对话次数，设为 0 表示不限制</p>
          </div>

          <div class="syscfg-row" :class="{ 'syscfg-row--changed': cKeyChanged('agent_max_history') }">
            <div class="syscfg-row__info">
              <span class="syscfg-row__label">历史消息条数</span>
              <span class="syscfg-row__env">env: AGENT_MAX_HISTORY=20</span>
            </div>
            <div class="syscfg-row__ctrl">
              <el-input-number v-model.number="form.agent_max_history" :min="1" :max="200" controls-position="right" />
              <span class="syscfg-row__unit">条</span>
              <el-button v-if="cKeyChanged('agent_max_history')" text size="small" class="syscfg-row__reset" @click="resetKey('agent_max_history')">还原</el-button>
            </div>
            <p class="syscfg-row__hint">AI 对话中保持的上下文历史消息数，越多的上下文消耗更多 tokens</p>
          </div>

          <div class="syscfg-row" :class="{ 'syscfg-row--changed': cKeyChanged('agent_history_ttl') }">
            <div class="syscfg-row__info">
              <span class="syscfg-row__label">历史消息过期时间</span>
              <span class="syscfg-row__env">env: AGENT_HISTORY_TTL=720h</span>
            </div>
            <div class="syscfg-row__ctrl">
              <el-input v-model="form.agent_history_ttl" style="width:120px" />
              <span class="syscfg-row__unit">小时 (h) / 天 (d)</span>
              <el-button v-if="cKeyChanged('agent_history_ttl')" text size="small" class="syscfg-row__reset" @click="resetKey('agent_history_ttl')">还原</el-button>
            </div>
            <p class="syscfg-row__hint">超过此时间的历史消息会被后台清理任务自动移除，720h = 30 天</p>
          </div>

          <div class="syscfg-row" :class="{ 'syscfg-row--changed': cKeyChanged('agent_request_timeout') }">
            <div class="syscfg-row__info">
              <span class="syscfg-row__label">LLM 请求超时</span>
              <span class="syscfg-row__env">env: AGENT_REQUEST_TIMEOUT=90s</span>
            </div>
            <div class="syscfg-row__ctrl">
              <el-input v-model="form.agent_request_timeout" style="width:120px" />
              <span class="syscfg-row__unit">秒 (s) / 毫秒 (ms)</span>
              <el-button v-if="cKeyChanged('agent_request_timeout')" text size="small" class="syscfg-row__reset" @click="resetKey('agent_request_timeout')">还原</el-button>
            </div>
            <p class="syscfg-row__hint">AI 大模型 API 调用超时时间，模型响应慢时可适当调大</p>
          </div>
        </div>
      </section>

      <!-- ============ 全局限流 ============ -->
      <section class="syscfg-card">
        <div class="syscfg-card__header">
          <div class="syscfg-card__icon"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg></div>
          <div>
            <h3>全局限流</h3>
            <p>令牌桶限流，保护后端服务不被突发流量打垮</p>
          </div>
        </div>
        <div class="syscfg-card__body">
          <div class="syscfg-row" :class="{ 'syscfg-row--changed': cKeyChanged('rate_limit_enabled') }">
            <div class="syscfg-row__info">
              <span class="syscfg-row__label">限流总开关</span>
              <span class="syscfg-row__env">env: RATE_LIMIT_ENABLED=false</span>
            </div>
            <div class="syscfg-row__ctrl">
              <el-switch v-model="form.rate_limit_enabled" active-text="开启" inactive-text="关闭" />
              <el-button v-if="cKeyChanged('rate_limit_enabled')" text size="small" class="syscfg-row__reset" @click="resetKey('rate_limit_enabled')">还原</el-button>
            </div>
            <p class="syscfg-row__hint">开启后使用令牌桶算法对全局限流，关闭后所有请求直接放行</p>
          </div>

          <div v-if="form.rate_limit_enabled" class="syscfg-rate-preview">
            <div class="syscfg-rate-bar">
              <div class="syscfg-rate-bar__fill" :style="{ width: rateBarWidth + '%' }"></div>
            </div>
            <p class="syscfg-rate-preview__text">
              每秒约产生 <strong>{{ form.rate_limit_rate }}</strong> 个令牌，
              桶容量 <strong>{{ form.rate_limit_burst }}</strong>，
              突发峰值允许 <strong>{{ form.rate_limit_burst.toLocaleString() }}</strong> 个并发请求
            </p>
          </div>

          <div class="syscfg-row" :class="{ 'syscfg-row--changed': cKeyChanged('rate_limit_rate') }">
            <div class="syscfg-row__info">
              <span class="syscfg-row__label">令牌桶速率</span>
              <span class="syscfg-row__env">env: RATE_LIMIT_RATE=20</span>
            </div>
            <div class="syscfg-row__ctrl">
              <el-input-number v-model.number="form.rate_limit_rate" :min="0.1" :step="1" :precision="1" controls-position="right" />
              <span class="syscfg-row__unit">请求/秒</span>
              <el-button v-if="cKeyChanged('rate_limit_rate')" text size="small" class="syscfg-row__reset" @click="resetKey('rate_limit_rate')">还原</el-button>
            </div>
            <p class="syscfg-row__hint">令牌每秒补充的速率，每 1/rate 秒放入一个令牌</p>
          </div>

          <div class="syscfg-row" :class="{ 'syscfg-row--changed': cKeyChanged('rate_limit_burst') }">
            <div class="syscfg-row__info">
              <span class="syscfg-row__label">令牌桶容量</span>
              <span class="syscfg-row__env">env: RATE_LIMIT_BURST=50</span>
            </div>
            <div class="syscfg-row__ctrl">
              <el-input-number v-model.number="form.rate_limit_burst" :min="1" :max="10000" controls-position="right" />
              <span class="syscfg-row__unit">个</span>
              <el-button v-if="cKeyChanged('rate_limit_burst')" text size="small" class="syscfg-row__reset" @click="resetKey('rate_limit_burst')">还原</el-button>
            </div>
            <p class="syscfg-row__hint">桶内最多积累的令牌数，决定了短时间内的突发并发上限</p>
          </div>
        </div>
      </section>
    </div>

    <!-- 底栏保存条 -->
    <transition name="syscfg-fade">
      <div v-if="changeCount > 0" class="syscfg-sticky-bar">
        <div class="syscfg-sticky-bar__inner">
          <span class="syscfg-sticky-bar__info">
            <el-icon><WarningFilled /></el-icon>
            您有 <strong>{{ changeCount }}</strong> 项配置未保存
          </span>
          <div class="syscfg-sticky-bar__actions">
            <el-button @click="discardAll">放弃修改</el-button>
            <el-button type="primary" :loading="saving" @click="saveConfigs">
              保存 {{ changeCount }} 项配置
            </el-button>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<script>
import { WarningFilled } from "@element-plus/icons-vue";
import { adminListSystemConfigs, adminUpdateSystemConfigs } from "@/api/admin";

export default {
  name: "SystemConfigManage",
  components: { WarningFilled },
  data() {
    return {
      loading: false,
      saving: false,
      form: {
        agent_enabled: false,
        agent_daily_quota: 80,
        agent_max_history: 20,
        agent_history_ttl: "720h",
        agent_request_timeout: "90s",
        rate_limit_enabled: false,
        rate_limit_rate: 20,
        rate_limit_burst: 50,
      },
      original: null,
    };
  },
  computed: {
    changeCount() {
      if (!this.original) return 0;
      return Object.keys(this.original).filter((k) => this.cKeyChanged(k)).length;
    },
    rateBarWidth() {
      const rate = this.form.rate_limit_rate || 1;
      const burst = this.form.rate_limit_burst || 1;
      return Math.min(100, (rate / burst) * 100);
    },
  },
  created() {
    this.loadConfigs();
  },
  methods: {
    cKeyChanged(key) {
      if (!this.original) return false;
      return String(this.form[key] ?? "") !== String(this.original[key] ?? "");
    },
    async loadConfigs() {
      this.loading = true;
      try {
        const body = await adminListSystemConfigs();
        const raw = body.data || {};
        this.original = { ...raw };
        this.form.agent_enabled = raw.agent_enabled === "true";
        this.form.agent_daily_quota = parseInt(raw.agent_daily_quota, 10) || 80;
        this.form.agent_max_history = parseInt(raw.agent_max_history, 10) || 20;
        this.form.agent_history_ttl = raw.agent_history_ttl || "720h";
        this.form.agent_request_timeout = raw.agent_request_timeout || "90s";
        this.form.rate_limit_enabled = raw.rate_limit_enabled === "true";
        this.form.rate_limit_rate = parseFloat(raw.rate_limit_rate) || 20;
        this.form.rate_limit_burst = parseInt(raw.rate_limit_burst, 10) || 50;
      } catch {
        // error already toasted
      } finally {
        this.loading = false;
      }
    },
    resetKey(key) {
      if (!this.original) return;
      const raw = this.original[key];
      if (key === "agent_enabled" || key === "rate_limit_enabled") {
        this.form[key] = raw === "true";
      } else if (key === "agent_daily_quota" || key === "agent_max_history" || key === "rate_limit_burst") {
        this.form[key] = parseInt(raw, 10) || 0;
      } else if (key === "rate_limit_rate") {
        this.form[key] = parseFloat(raw) || 0;
      } else {
        this.form[key] = raw || "";
      }
    },
    discardAll() {
      if (!this.original) return;
      Object.keys(this.original).forEach((k) => this.resetKey(k));
    },
    async saveConfigs() {
      if (this.changeCount === 0) return;
      this.saving = true;
      try {
        const configs = {
          agent_enabled: this.form.agent_enabled ? "true" : "false",
          agent_daily_quota: String(this.form.agent_daily_quota),
          agent_max_history: String(this.form.agent_max_history),
          agent_history_ttl: this.form.agent_history_ttl,
          agent_request_timeout: this.form.agent_request_timeout,
          rate_limit_enabled: this.form.rate_limit_enabled ? "true" : "false",
          rate_limit_rate: String(this.form.rate_limit_rate),
          rate_limit_burst: String(this.form.rate_limit_burst),
        };
        await adminUpdateSystemConfigs(configs);
        this.original = { ...configs };
        this.$message.success("配置已保存，将在 30s 内服务器同步生效");
      } catch {
        // error already toasted
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>

<style lang="scss" scoped>
@import "@/style/mixin";

/* ===== Hero Header ===== */
.syscfg-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}
.syscfg-hero__text {
  h2 {
    font-size: 20px;
    font-weight: 600;
    color: #1f2a3a;
    margin: 0 0 4px;
  }
  p {
    font-size: 13px;
    color: #8a929a;
    margin: 0;
    line-height: 1.5;
    code {
      background: #eef2f5;
      padding: 1px 6px;
      border-radius: 4px;
      font-size: 12px;
      color: #1d77b2;
    }
    strong {
      color: #f59e0b;
    }
  }
}
.syscfg-hero__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  padding-top: 2px;
}

/* ===== Grid ===== */
.syscfg-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* ===== Card ===== */
.syscfg-card {
  background: #fff;
  border: 1px solid #e3e5e7;
  border-radius: 10px;
  overflow: hidden;
  transition: box-shadow 0.2s;
  &:hover {
    box-shadow: 0 2px 12px rgba(0,0,0,0.06);
  }
}
.syscfg-card__header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px 0;
  h3 {
    font-size: 16px;
    font-weight: 600;
    color: #1f2a3a;
    margin: 0;
  }
  p {
    font-size: 13px;
    color: #8a929a;
    margin: 2px 0 0;
  }
}
.syscfg-card__icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: #f0f7ff;
  color: #409eff;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.syscfg-card__body {
  padding: 12px 20px 4px;
}

/* ===== Row ===== */
.syscfg-row {
  padding: 12px 0;
  border-bottom: 1px solid #f0f1f3;
  transition: background 0.15s;
  &:last-child {
    border-bottom: none;
  }
  &--changed {
    background: #fefce8;
    margin: 0 -20px;
    padding: 12px 20px;
    border-bottom-color: #fde68a;
    .syscfg-row__reset {
      display: inline-flex !important;
    }
    .syscfg-row__label::after {
      content: " \2022";
      color: #f59e0b;
      font-weight: bold;
    }
  }
}
.syscfg-row__info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.syscfg-row__label {
  font-size: 14px;
  font-weight: 500;
  color: #1f2a3a;
}
.syscfg-row__env {
  font-size: 11px;
  color: #aab1b9;
  background: #f4f5f7;
  padding: 1px 6px;
  border-radius: 3px;
  font-family: monospace;
}
.syscfg-row__ctrl {
  display: flex;
  align-items: center;
  gap: 8px;
}
.syscfg-row__unit {
  font-size: 12px;
  color: #8a929a;
}
.syscfg-row__reset {
  display: none !important;
  color: #f59e0b;
}
.syscfg-row__hint {
  font-size: 12px;
  color: #8a929a;
  margin: 6px 0 0;
  line-height: 1.4;
}

/* ===== Rate Preview ===== */
.syscfg-rate-preview {
  background: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 8px;
  padding: 12px 16px;
  margin: 8px 0 12px;
}
.syscfg-rate-bar {
  height: 6px;
  background: #e0e7ef;
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 8px;
}
.syscfg-rate-bar__fill {
  height: 100%;
  background: linear-gradient(90deg, #60a5fa, #3b82f6);
  border-radius: 3px;
  transition: width 0.3s ease;
}
.syscfg-rate-preview__text {
  font-size: 12px;
  color: #475569;
  margin: 0;
  strong {
    color: #1d4ed8;
  }
}

/* ===== Sticky Save Bar ===== */
.syscfg-sticky-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: #fff;
  border-top: 1px solid #e3e5e7;
  box-shadow: 0 -4px 12px rgba(0,0,0,0.08);
  z-index: 100;
  padding: 12px 0;
}
.syscfg-sticky-bar__inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.syscfg-sticky-bar__info {
  font-size: 14px;
  color: #f59e0b;
  display: flex;
  align-items: center;
  gap: 6px;
  strong {
    color: #d97706;
  }
}
.syscfg-sticky-bar__actions {
  display: flex;
  gap: 8px;
}

/* ===== Transition ===== */
.syscfg-fade-enter-active,
.syscfg-fade-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}
.syscfg-fade-enter-from,
.syscfg-fade-leave-to {
  opacity: 0;
  transform: translateY(20px);
}

/* ===== Badge ===== */
.syscfg-save-badge {
  .el-badge__content {
    background: #f59e0b;
    border: none;
  }
}
</style>