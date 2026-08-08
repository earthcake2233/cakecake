<template>
  <div
    class="msg-chat-row"
    :class="{ 'msg-chat-row--mine': item.is_mine }"
    :data-msg-id="item.id != null ? String(item.id) : ''"
  >
    <img class="msg-chat-face" :src="item.face" alt="" width="32" height="32" />
    <div
      v-if="item.is_agent && item.isStreaming && !item.content"
      class="msg-chat-bubble msg-chat-bubble--md msg-chat-bubble--streaming"
    >
      正在重新生成…
    </div>
    <div
      v-else
      class="msg-chat-bubble"
      :class="{ 'msg-chat-bubble--md': item.is_agent }"
      v-html="renderMsgContent(item)"
    ></div>
  </div>
  <div
    v-if="item.is_agent && item.id !== 'agent-draft' && !item.isStreaming"
    class="msg-chat-actions"
  >
      <button
        type="button"
        class="msg-chat-action"
        title="复制"
        @click.stop="$emit('copy', item)"
      >
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="9" y="9" width="11" height="11" rx="2" />
          <path d="M5 15V5a2 2 0 0 1 2-2h10" />
        </svg>
        <span v-if="copiedMsgId === item.id" class="msg-chat-action__tip">已复制</span>
      </button>
      <button
        type="button"
        class="msg-chat-action"
        :class="{ 'is-active is-active--like': feedbackOf(item) === 'like' }"
        title="有帮助"
        @click.stop="$emit('feedback', 'like')"
      >
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M7 10v11H4a1 1 0 0 1-1-1v-9a1 1 0 0 1 1-1h3zm0 0l4-7a2 2 0 0 1 2 2v4h6a2 2 0 0 1 2 2.2l-1.2 6A2 2 0 0 1 17.8 21H7" />
        </svg>
      </button>
      <button
        type="button"
        class="msg-chat-action"
        :class="{ 'is-active is-active--dislike': feedbackOf(item) === 'dislike' }"
        title="没帮助"
        @click.stop="$emit('feedback', 'dislike')"
      >
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M17 14V3h3a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1h-3zm0 0l-4 7a2 2 0 0 1-2-2v-4H5a2 2 0 0 1-2-2.2L4.2 6A2 2 0 0 1 6.2 3H17" />
        </svg>
      </button>
      <button
        type="button"
        class="msg-chat-action"
        title="重新生成"
        @click.stop="$emit('regenerate', item)"
      >
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 12a9 9 0 1 1-2.64-6.36" />
          <path d="M21 3v6h-6" />
        </svg>
      </button>
  </div>
  <AgentVersionSwitcher
    v-if="item.is_agent && !item.isStreaming"
    :index="item.versionIndex"
    :count="item.versions && item.versions.length"
    @switch="$emit('switch-version', $event)"
  />
  <AgentToolActivities :acts="item.toolActivities" />
  <div v-if="resultCards.length" class="msg-tool-results">
      <div v-for="(card, ii) in resultCards" :key="ii" class="msg-result-card" @click.stop="$emit('open-item', card)">
          <template v-if="card.cover || card.cover_url">
            <img class="msg-result-card__cover" :src="card.cover || card.cover_url" />
            <div class="msg-result-card__body">
              <div class="msg-result-card__title">{{ card.title }}</div>
              <div class="msg-result-card__meta">
                {{ card.author || card.uploader_name || card.uploader || "unknown" }}
                &middot; {{ formatPlayCount(card.play_count || card.plays || 0) }}
              </div>
            </div>
          </template>
          <template v-else-if="card.user_name && card.type">
            <span class="msg-result-card__badge msg-result-card__badge--danmaku">弹幕</span>
            <img class="msg-result-card__avatar" :src="card.user_avatar || defaultAvatarSvg" @error="$emit('avatar-error', $event)" />
            <div class="msg-result-card__body">
              <div class="msg-result-card__content">{{ card.content }}</div>
              <div class="msg-result-card__meta">{{ card.user_name }} &middot; {{ formatVideoTime(card.video_time) }}</div>
            </div>
          </template>
          <template v-else-if="card.user_name && !card.type">
            <span class="msg-result-card__badge msg-result-card__badge--comment">评论</span>
            <img class="msg-result-card__avatar" :src="card.user_avatar || defaultAvatarSvg" @error="$emit('avatar-error', $event)" />
            <div class="msg-result-card__body">
              <div class="msg-result-card__content">{{ card.content }}</div>
              <div class="msg-result-card__meta">{{ card.user_name }} &middot; {{ card.like_count || 0 }}赞</div>
            </div>
          </template>
          <template v-else>
            <div class="msg-result-card__content">{{ card.content }}</div>
            <div class="msg-result-card__meta">{{ card.user_name || "匿名" }}</div>
          </template>
      </div>
  </div>
  <AgentSuggestionChips
    v-if="item.is_agent && !item.isStreaming && item.id !== 'agent-draft'"
    :suggestions="suggestionsFor(item)"
    @pick="$emit('pick-suggestion', $event)"
  />
</template>

<script>
import MarkdownIt from "markdown-it";
import DOMPurify from "dompurify";
import hljs from "highlight.js";
import defaultFace from "@/assets/akari.jpg";
import AgentToolActivities from "./AgentToolActivities.vue";
import AgentVersionSwitcher from "./AgentVersionSwitcher.vue";
import AgentSuggestionChips from "./AgentSuggestionChips.vue";

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  highlight(str, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return (
          '<pre class="hljs"><code class="language-' +
          lang +
          '">' +
          hljs.highlight(str, { language: lang, ignoreIllegals: true }).value +
          "</code></pre>"
        );
      } catch (e) {
        /* fall through to escaped plain block */
      }
    }
    return "<pre class=\"hljs\"><code>" + md.utils.escapeHtml(str) + "</code></pre>";
  }
});

export default {
  name: "MbDmMessageItem",
  components: {
    AgentToolActivities,
    AgentVersionSwitcher,
    AgentSuggestionChips
  },
  props: {
    item: { type: Object, required: true },
    feedbackMap: { type: Object, default: () => ({}) },
    copiedMsgId: { type: [Number, String], default: null }
  },
  emits: ["copy", "feedback", "regenerate", "switch-version", "pick-suggestion", "open-item", "avatar-error"],
  data() {
    return {
      defaultAvatarSvg: defaultFace
    };
  },
  computed: {
    resultCards() {
      const out = [];
      const seen = new Set();
      for (const items of Object.values(this.item.toolResultData || {})) {
        for (const card of items || []) {
          const key =
            card && (card.id != null || card.video_id != null)
              ? "id:" + (card.id != null ? card.id : card.video_id)
              : "t:" + String((card && (card.title || card.content)) || "");
          if (!key || seen.has(key)) continue;
          seen.add(key);
          out.push(card);
        }
      }
      return out;
    }
  },
  methods: {
    escapeHtmlText(str) {
      return String(str || "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
    },
    renderMsgContent(m) {
      if (!m || !m.content) return "";
      if (!m.is_agent) return this.escapeHtmlText(m.content);
      // The display marker line (【展示】...) is machine-only and must never
      // surface to the user (defensive: the backend strips it on persist).
      const cleaned = String(m.content)
        .replace(/^[^\n]*【展示】[^\n]*$/gm, "")
        .trim();
      return DOMPurify.sanitize(md.render(cleaned));
    },
    feedbackOf(m) {
      return (m && this.feedbackMap[String(m.id)]) || "";
    },
    suggestionsFor(m) {
      if (!m || !m.is_agent) return [];
      if (Array.isArray(m.suggestions) && m.suggestions.length) {
        return m.suggestions.slice(0, 3);
      }
      return [];
    },
    formatVideoTime(sec) {
      if (sec == null || sec === undefined) return "";
      const min = Math.floor(sec / 60);
      const s = Math.floor(sec % 60);
      return min + ":" + String(s).padStart(2, "0");
    },
    formatPlayCount(n) {
      if (n >= 10000) return (n / 10000).toFixed(1) + "万";
      if (n >= 1000) return (n / 1000).toFixed(1) + "千";
      return String(n);
    }
  }
};
</script>
