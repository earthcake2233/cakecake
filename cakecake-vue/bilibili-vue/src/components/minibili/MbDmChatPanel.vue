<template>
  <div class="mb-dm-chat msg-main__split">
    <div class="msg-col-msg">
      <div class="msg-col-msg__hint">最近消息</div>
      <ul v-if="threadRows.length" class="msg-thread-list">
        <li
          v-for="row in threadRows"
          :key="row.id"
          class="msg-thread-item"
          :class="{ 'msg-thread-item--on': row.id === selectedConvId }"
          @click="selectConversation(row.id)"
        >
          <div class="msg-thread-item__face">
            <img :src="row.face" alt="" />
          </div>
          <div class="msg-thread-item__meta">
            <div class="msg-thread-item__top">
              <div class="msg-thread-item__name">{{ row.name }}</div>
              <span
                v-if="row.muted"
                class="msg-thread-item__mute"
                title="消息免打扰"
                aria-label="消息免打扰"
              >
                <img :src="muteIcon" alt="" width="14" height="14" />
              </span>
            </div>
            <div class="msg-thread-item__snippet">{{ row.snippet }}</div>
          </div>
          <span
            v-if="row.unread > 0"
            class="msg-thread-item__badge"
            aria-hidden="true"
            >{{ row.unread > 99 ? "99+" : row.unread }}</span
          >
          <button
            v-if="!row.is_agent"
            type="button"
            class="msg-thread-item__del"
            aria-label="删除该会话"
            title="删除会话"
            @click.stop="onDeleteConversation(row)"
          >
            ×
          </button>
        </li>
      </ul>
      <p v-else class="msg-col-msg__empty">暂无会话</p>
    </div>
    <div
      class="msg-col-detail"
      :class="{ 'msg-col-detail--chat': selectedConvId }"
      :style="selectedConvId ? { backgroundColor: '#f8f9fa' } : null"
    >
      <template v-if="!selectedConvId">
        <div class="msg-empty" aria-live="polite">
          <div class="msg-empty__art" aria-hidden="true">
            <img class="msg-empty__img" :src="gochatIllus" alt="" />
          </div>
          <p class="msg-empty__text">快找小伙伴聊天吧 ( ゜- ゜)つロ</p>
        </div>
      </template>
      <template v-else>
        <header class="msg-chat-head" style="background-color: #f8f9fa">
          <span class="msg-chat-head__name">
            {{ selectedConvPeerName }}
            <span
              v-if="selectedIsAgent"
              style="margin-left:6px;font-size:11px;color:#00a1d6;border:1px solid #00a1d6;border-radius:3px;padding:0 3px 1px;vertical-align:middle"
              >AI</span
            >
          </span>
          <div
            class="msg-chat-head-more-wrap"
            :class="{ 'is-open': headMenuOpen }"
            @click.stop
          >
            <button
              type="button"
              class="msg-chat-head__more"
              aria-label="更多"
              aria-haspopup="true"
              :aria-expanded="headMenuOpen"
              @click="toggleHeadMenu"
            >
              <span class="msg-chat-head__more-dots" aria-hidden="true">
                <i /><i /><i />
              </span>
            </button>
            <div
              v-if="headMenuOpen"
              class="msg-chat-head-menu"
              role="menu"
              @click.stop
            >
              <button
                type="button"
                class="msg-chat-head-menu__item"
                role="menuitem"
                @click="onHeadMenuPin"
              >
                {{ chatPinned ? "取消置顶聊天" : "置顶聊天" }}
              </button>
              <button
                type="button"
                class="msg-chat-head-menu__item"
                role="menuitem"
                @click="onHeadMenuMute"
              >
                {{ chatMuted ? "关闭免打扰" : "开启免打扰" }}
              </button>
              <button
                v-if="selectedIsAgent"
                type="button"
                class="msg-chat-head-menu__item"
                role="menuitem"
                @click="onHeadMenuResetAgent"
              >
                清空记录并重新开始
              </button>
              <button
                v-if="!selectedIsAgent"
                type="button"
                class="msg-chat-head-menu__item"
                role="menuitem"
                @click="onHeadMenuBlacklist"
              >
                加入黑名单
              </button>
              <button
                v-if="!selectedIsAgent"
                type="button"
                class="msg-chat-head-menu__item"
                role="menuitem"
                @click="onHeadMenuReport"
              >
                举报该用户
              </button>
            </div>
          </div>
        </header>
        <div
          ref="chatScrollEl"
          class="msg-chat-scroll"
          style="background-color: #f8f9fa"
          @scroll.passive="onChatScroll"
        >
          <div
            v-if="chatLoadingMore && chatNextCursor"
            class="msg-chat-loading msg-chat-loading--top"
          >
            加载更早的消息…
          </div>
          <div v-else-if="chatLoading && !chatMessages.length" class="msg-chat-loading">
            加载中…
          </div>
          <template v-for="(grp, gi) in chatMessageGroups" :key="'g' + gi">
            <div class="msg-chat-time">{{ grp.label }}</div>
            <template v-for="m in grp.messages" :key="m.id">
            <div class="msg-chat-row" :class="{ 'msg-chat-row--mine': m.is_mine }">
              <img
                class="msg-chat-face"
                :src="m.face"
                alt=""
                width="32"
                height="32"
              />
              <div class="msg-chat-bubble">{{ m.content }}</div>
            </div>
            <div v-if="m.toolActivities.length" class="msg-tool-activities">
              <div v-for="act in m.toolActivities" :key="act.span_id" class="msg-tool-activity">
                <span class="msg-tool-activity__status">{{ act.status === "running" ? "↻" : "✓" }}</span>
                <span class="msg-tool-activity__name">{{ act.tool_name }}</span>
                <span v-if="act.duration_ms" class="msg-tool-activity__dur">{{ act.duration_ms }}ms</span>
              </div>
            </div>
            <div v-if="Object.keys(m.toolResultData).length" class="msg-tool-results">
              <template v-for="(items, spanId) in m.toolResultData" :key="spanId">
                <div v-for="(item, ii) in items" :key="ii" class="msg-result-card" @click.stop="goToItem(item)">
                  <template v-if="item.cover || item.cover_url">
                    <img class="msg-result-card__cover" :src="item.cover || item.cover_url" />
                    <div class="msg-result-card__body">
                      <div class="msg-result-card__title">{{ item.title }}</div>
                      <div class="msg-result-card__meta">
                        {{ item.author || item.uploader_name || item.uploader || "unknown" }}
                        &middot; {{ formatPlayCount(item.play_count || item.plays || 0) }}
                      </div>
                    </div>
                  </template>
                  <template v-else-if="item.user_name && item.type">
                    <span class="msg-result-card__badge msg-result-card__badge--danmaku">弹幕</span>
                    <img class="msg-result-card__avatar" :src="item.user_avatar || defaultFace" @error="onAvatarError" />
                    <div class="msg-result-card__body">
                      <div class="msg-result-card__content">{{ item.content }}</div>
                      <div class="msg-result-card__meta">{{ item.user_name }} &middot; {{ formatVideoTime(item.video_time) }}</div>
                    </div>
                  </template>
                  <template v-else-if="item.user_name && !item.type">
                    <span class="msg-result-card__badge msg-result-card__badge--comment">评论</span>
                    <img class="msg-result-card__avatar" :src="item.user_avatar || defaultFace" @error="onAvatarError" />
                    <div class="msg-result-card__body">
                      <div class="msg-result-card__content">{{ item.content }}</div>
                      <div class="msg-result-card__meta">{{ item.user_name }} &middot; {{ item.like_count || 0 }}赞</div>
                    </div>
                  </template>
                  <template v-else>
                    <div class="msg-result-card__content">{{ item.content }}</div>
                    <div class="msg-result-card__meta">{{ item.user_name || "匿名" }}</div>
                  </template>
                </div>
              </template>
            </div>
          </template>
          </template>
          <div v-if="chatAwaitingAgent && _liveToolActs.length" class="msg-tool-activities msg-tool-activities--live">
            <div v-for="act in _liveToolActs" :key="act.span_id" class="msg-tool-activity">
              <span class="msg-tool-activity__status">{{ act.status === "running" ? "?" : "?" }}</span>
              <span class="msg-tool-activity__name">{{ act.tool_name }}</span>
              <span v-if="act.duration_ms" class="msg-tool-activity__dur">{{ act.duration_ms }}ms</span>
            </div>
          </div>
          <div
            v-if="chatAwaitingAgent"
            class="msg-chat-loading msg-chat-loading--typing"
          >
            AI 正在输入…
          </div>
        </div>
        <footer class="msg-chat-compose">
          <div class="msg-chat-compose-box">
          <div class="msg-chat-toolbar">
            <button type="button" class="msg-chat-tool" title="图片" disabled>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
                <rect
                  x="3"
                  y="5"
                  width="18"
                  height="14"
                  rx="2"
                  stroke="currentColor"
                  stroke-width="1.5"
                />
                <circle cx="8.5" cy="10" r="1.5" fill="currentColor" />
                <path
                  d="M3 16l5-5 4 4 3-3 6 6"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linejoin="round"
                />
              </svg>
            </button>
            <button type="button" class="msg-chat-tool" title="表情" disabled>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
                <circle
                  cx="12"
                  cy="12"
                  r="9"
                  stroke="currentColor"
                  stroke-width="1.5"
                />
                <path
                  d="M8 14s1.5 2 4 2 4-2 4-2"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linecap="round"
                />
                <circle cx="9" cy="10" r="1" fill="currentColor" />
                <circle cx="15" cy="10" r="1" fill="currentColor" />
              </svg>
            </button>
          </div>
          <textarea
            v-model="chatDraft"
            class="msg-chat-input"
            rows="3"
            maxlength="500"
            placeholder="请输入消息内容"
            @keydown.enter.exact.prevent="sendChatMessage"
          />
          <div class="msg-chat-compose-foot">
            <span class="msg-chat-counter">{{ chatDraft.length }}/500</span>
            <button
              type="button"
              class="msg-chat-send"
              :disabled="chatPosting || chatAwaitingAgent || !chatDraftTrimmed"
              @click="sendChatMessage"
            >
              发送
            </button>
          </div>
          </div>
        </footer>
      </template>
    </div>

    <Teleport to="body">
      <div
        v-if="blacklistDialogOpen"
        class="msg-dm-modal-overlay"
        role="dialog"
        aria-modal="true"
        aria-labelledby="msg-dm-blacklist-title"
      >
        <div
          class="msg-dm-modal-overlay__backdrop"
          aria-hidden="true"
          @click="closeBlacklistDialog"
        />
        <div class="msg-dm-modal">
          <button
            type="button"
            class="msg-dm-modal__close"
            aria-label="关闭"
            @click="closeBlacklistDialog"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              aria-hidden="true"
            >
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
          <h2 id="msg-dm-blacklist-title" class="msg-dm-modal__title">
            加入黑名单
          </h2>
          <p class="msg-dm-modal__desc">
            加入黑名单后，将自动解除关注关系和对该用户的合集订阅关系，禁止该用户与我互动或查看我的空间
          </p>
          <div class="msg-dm-modal__actions">
            <button
              type="button"
              class="msg-dm-modal__btn msg-dm-modal__btn--ghost"
              @click="closeBlacklistDialog"
            >
              取消
            </button>
            <button
              type="button"
              class="msg-dm-modal__btn msg-dm-modal__btn--primary"
              :disabled="blacklistSubmitting"
              @click="confirmBlacklist"
            >
              确定
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script>
import { ElMessage, ElMessageBox } from "element-plus";
import {
  mbBlockUser,
  mbCreateDmConversation,
  mbDeleteDmConversation,
  mbListDmConversations,
  mbListDmMessages,
  mbPatchDmConversationSettings,
  mbPostDmMessage,
  mbResetDmAgentConversation,
  mbWsChatUrl
} from "@/api/minibili";
import { getAccessToken, getUserId } from "@/utils/authTokens";
import defaultFace from "@/assets/akari.jpg";
import gochatIllus from "@/assets/gochat.png";
import muteIcon from "@/assets/mute.png";
import { refreshMessageUnread } from "@/utils/messageUnread";

/** 每次向服务端请求的历史消息条数 */
const DM_MESSAGE_PAGE_SIZE = 30;

function parseApiTime(s) {
  if (!s) return new Date();
  const m = /^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$/.exec(String(s));
  if (!m) return new Date(s);
  return new Date(
    Number(m[1]),
    Number(m[2]) - 1,
    Number(m[3]),
    Number(m[4]),
    Number(m[5]),
    Number(m[6])
  );
}

export default {
  name: "MbDmChatPanel",
  props: {
    peerIdFromRoute: { type: Number, default: 0 }
  },
  data() {
    return {
      gochatIllus,
      muteIcon,
      dmConversations: [],
      selectedConvId: null,
      selectedPeerId: 0,
      selectedPeerName: "",
      headMenuOpen: false,
      chatPinned: false,
      chatMuted: false,
      blacklistDialogOpen: false,
      blacklistSubmitting: false,
      dmSettingsPatching: false,
      chatMessages: [],
      chatNextCursor: "",
      chatLoading: false,
      chatLoadingMore: false,
      chatPosting: false,
      chatAwaitingAgent: false,
      _agentReplyTimer: null,
      deletingConvId: 0,
      resettingAgent: false,
      chatDraft: "",
      _pendingResultData: {},
      _pendingToolActs: [],
      chatWs: null,
      _chatWsRetryTimer: null,
      _liveToolActs: [],
      _wsReconnectAttempts: 0,
    };
  },
  computed: {
    threadRows() {
      return (this.dmConversations || []).map(c => ({
        id: Number(c.id),
        name: c.peer_name || "用户",
        snippet: c.last_preview || "暂无消息",
        face: c.peer_avatar || defaultFace,
        unread: Number(c.unread_count) || 0,
        muted: !!c.muted,
        pinned: !!c.pinned,
        is_agent: !!(c.is_agent || c.kind === "agent")
      }));
    },
    selectedIsAgent() {
      const hit = this.dmConversations.find(
        c => Number(c.id) === Number(this.selectedConvId)
      );
      return !!(hit && (hit.is_agent || hit.kind === "agent"));
    },
    selectedConvPeerName() {
      if (this.selectedPeerName) return this.selectedPeerName;
      const row = this.threadRows.find(r => r.id === this.selectedConvId);
      return (row && row.name) || "会话";
    },
    chatDraftTrimmed() {
      return String(this.chatDraft || "").trim();
    },
    chatMessageGroups() {
      const me = getUserId();
      const groups = [];
      let curLabel = "";
      let curMsgs = [];
      const flush = () => {
        if (curMsgs.length) {
          groups.push({ label: curLabel, messages: curMsgs });
        }
      };
      for (const raw of this.chatMessages || []) {
        const d = parseApiTime(raw.created_at);
        const label = `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日 ${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
        const isMine =
          me != null && Number(raw.sender_id) === Number(me);
        const item = {
          id: raw.id,
          content: raw.content,
          face: raw.sender_avatar || defaultFace,
          is_mine: isMine,
          toolActivities: raw.tool_activities ? JSON.parse(raw.tool_activities) : (raw._toolActivities || []),
          toolResultData: raw.tool_result_data ? JSON.parse(raw.tool_result_data) : (raw._toolResultData || {})
        };
        if (label !== curLabel) {
          flush();
          curLabel = label;
          curMsgs = [item];
        } else {
          curMsgs.push(item);
        }
      }
      flush();
      return groups;
    }
  },
  watch: {
    peerIdFromRoute: {
      immediate: true,
      handler(v) {
        const pid = Number(v) || 0;
        if (pid > 0) {
          void this.openPeerConversation(pid);
        }
      }
    }
  },
  mounted() {
    void this.loadConversations();
    this.connectChatWs();
    document.addEventListener("click", this.onDocumentClick);
  },
  beforeUnmount() {
    this.clearAgentReplyTimer();
    this.disconnectChatWs();
    document.removeEventListener("click", this.onDocumentClick);
  },
  methods: {
    async loadConversations() {
      try {
        const { items } = await mbListDmConversations();
        this.dmConversations = items || [];
      } catch (e) {
        ElMessage.error((e && e.message) || "加载会话失败");
      }
    },
    async openPeerConversation(peerId) {
      try {
        const conv = await mbCreateDmConversation(peerId);
        await this.loadConversations();
        if (conv && conv.id) {
          await this.selectConversation(Number(conv.id));
        }
      } catch (e) {
        ElMessage.error((e && e.message) || "无法发起会话");
      }
    },
    async selectConversation(id) {
      const cid = Number(id);
      if (!cid) return;
      this.clearAgentReplyTimer();
      this.chatAwaitingAgent = false;
      this.closeHeadMenu();
      this.selectedConvId = cid;
      const hit = this.dmConversations.find(c => Number(c.id) === cid);
      this.selectedPeerName = hit ? hit.peer_name : "";
      this.selectedPeerId = hit ? Number(hit.peer_id) || 0 : 0;
      this.syncChatPrefsFromConv();
      this.chatMessages = [];
      this.chatNextCursor = "";
      await this.loadChatMessages(false);
      this.$nextTick(() => this.scrollChatToBottom());
    },
    async loadChatMessages(older) {
      if (!this.selectedConvId) return;
      if (older) {
        if (!this.chatNextCursor || this.chatLoadingMore) return;
        this.chatLoadingMore = true;
      } else {
        this.chatLoading = true;
      }
      const el = this.$refs.chatScrollEl;
      const prevScrollHeight = older && el ? el.scrollHeight : 0;
      try {
        const res = await mbListDmMessages(this.selectedConvId, {
          cursor: older ? this.chatNextCursor : "",
          limit: DM_MESSAGE_PAGE_SIZE
        });
        const more = res.items || [];
        this.chatNextCursor = res.next_cursor || "";
        if (res.peer_name) this.selectedPeerName = res.peer_name;
        if (res.peer_id) this.selectedPeerId = Number(res.peer_id) || 0;
        if (older) {
          this.chatMessages = [...more, ...this.chatMessages];
          await this.$nextTick();
          if (el) {
            el.scrollTop = el.scrollHeight - prevScrollHeight;
          }
        } else {
          this.chatMessages = more;
        }
        const idx = this.dmConversations.findIndex(
          c => Number(c.id) === Number(this.selectedConvId)
        );
        if (idx >= 0) {
          this.dmConversations[idx].unread_count = 0;
        }
        void refreshMessageUnread();
      } catch (e) {
        ElMessage.error((e && e.message) || "加载消息失败");
      } finally {
        if (older) {
          this.chatLoadingMore = false;
        } else {
          this.chatLoading = false;
        }
      }
    },
    onChatScroll() {
      const el = this.$refs.chatScrollEl;
      if (!el || !this.chatNextCursor || this.chatLoadingMore || this.chatLoading) {
        return;
      }
      if (el.scrollTop < 48) {
        void this.loadChatMessages(true);
      }
    },
    async onDeleteConversation(row) {
      const cid = Number(row && row.id) || 0;
      if (!cid || this.deletingConvId) return;
      this.deletingConvId = cid;
      try {
        await mbDeleteDmConversation(cid);
        this.dmConversations = (this.dmConversations || []).filter(
          c => Number(c.id) !== cid
        );
        if (Number(this.selectedConvId) === cid) {
          this.selectedConvId = null;
          this.selectedPeerId = 0;
          this.selectedPeerName = "";
          this.chatMessages = [];
          this.chatNextCursor = "";
        }
        void refreshMessageUnread();
        ElMessage.success("已删除该会话");
      } catch (e) {
        ElMessage.error((e && e.message) || "删除失败");
      } finally {
        this.deletingConvId = 0;
      }
    },
    scrollChatToBottom() {
      const el = this.$refs.chatScrollEl;
      if (el) el.scrollTop = el.scrollHeight;
    },
    appendMessageIfNew(msg) {
      if (!msg || msg.id == null) return;
      const cid = Number(msg.conversation_id);
      if (cid !== Number(this.selectedConvId)) return;
      if (this.chatMessages.some(m => Number(m.id) === Number(msg.id))) return;
      this.chatMessages = [...this.chatMessages, msg];
      const me = getUserId();
      if (
        this.selectedIsAgent &&
        me != null &&
        Number(msg.sender_id) !== Number(me)
      ) {
        this.clearAgentReplyTimer();
        this.chatAwaitingAgent = false;
      }
      this.$nextTick(() => this.scrollChatToBottom());
    },
    clearAgentReplyTimer() {
      if (this._agentReplyTimer) {
        clearTimeout(this._agentReplyTimer);
        this._agentReplyTimer = null;
      }
    },
    startAgentReplyWait() {
      this.clearAgentReplyTimer();
      this.chatAwaitingAgent = true;
      this._agentReplyTimer = setTimeout(() => {
        this.chatAwaitingAgent = false;
        this._agentReplyTimer = null;
      }, 120000);
    },
    upsertConversation(conv) {
      this.applyConversationPatch(conv);
    },
    connectChatWs() {
      this.disconnectChatWs();
      const token = getAccessToken();
      const url = mbWsChatUrl(token);
      if (!url) return;
      const ws = new WebSocket(url);
      this.chatWs = ws;
      ws.onmessage = ev => {
        try {
          const data = JSON.parse(ev.data);
          this.onChatWsPayload(data);
        } catch {
          /* ignore */
        }
      };
      ws.onopen = () => {
        this._wsReconnectAttempts = 0;
      };
      ws.onclose = () => {
        if (this.chatWs !== ws) return;
        this._wsReconnectAttempts++;
        const delay = Math.min(1000 * Math.pow(2, this._wsReconnectAttempts - 1), 30000);
        const jitter = Math.floor(Math.random() * 1000);
        this._chatWsRetryTimer = setTimeout(() => this.connectChatWs(), delay + jitter);
      };
      ws.onerror = () => {
        ws.close();
      };
    },
    disconnectChatWs() {
      if (this._chatWsRetryTimer) {
        clearTimeout(this._chatWsRetryTimer);
        this._chatWsRetryTimer = null;
      }
      this._wsReconnectAttempts = 0;
      if (this.chatWs) {
        try {
          this.chatWs.onclose = null;
          this.chatWs.onerror = null;
          this.chatWs.onmessage = null;
          this.chatWs.close();
        } catch {
          /* ignore */
        }
        this.chatWs = null;
      }
    },
    onChatWsPayload(data) {
      if (data.type === "tool_call_start" && data.body) {
        this._pendingToolActs.push({ ...data.body, status: "running" }); this._liveToolActs.push({ ...data.body, status: "running" });
        return;
      }
      if (data.type === "tool_call_end" && data.body) {
        const idx = this._pendingToolActs.findIndex(t => t.span_id === data.body.span_id);
        if (idx >= 0) {
          this._pendingToolActs[idx] = { ...this._pendingToolActs[idx], ...data.body, status: "done" }; this._liveToolActs[idx] = { ...this._liveToolActs[idx], ...data.body, status: "done" };
        } else {
          this._pendingToolActs.push({ ...data.body, status: "done" }); this._liveToolActs.push({ ...data.body, status: "done" });
        }
        return;
      }
      if (data.type === "tool_result_data" && data.body) {
        this._pendingResultData[data.body.span_id] = data.body.items;
        return;
      }
      if (!data || typeof data !== "object") return;
      if (data.type === "dm_message" && data.message) {
        if (this._pendingToolActs.length) {
          // Mark any tools still "running" as "done" (dm_message means LLM finished processing all results)
          this._pendingToolActs.forEach(t => { if (t.status === "running") t.status = "done"; });
          data.message._toolActivities = [...this._pendingToolActs];
          data.message._toolResultData = { ...this._pendingResultData };
          // Finalize any remaining running tools before clear
          this._liveToolActs.forEach(t => { if (t.status === "running") t.status = "done"; });
          this._pendingToolActs = []; this._liveToolActs = [];
          this._pendingResultData = {};
          // Data is now persisted in DB by backend, no need for sessionStorage
        }
        this.upsertConversationFromMessage(data.message);
        this.appendMessageIfNew(data.message);
      } else if (data.type === "dm_conversation" && data.conversation) {
        this.upsertConversation(data.conversation);
      }
    },
    upsertConversationFromMessage(msg) {
      const cid = Number(msg.conversation_id);
      const hit = this.dmConversations.find(c => Number(c.id) === cid);
      if (hit) {
        hit.last_preview = msg.content;
        hit.last_message_at = msg.created_at;
        this.upsertConversation({ ...hit });
      } else {
        void this.loadConversations();
      }
    },
    onDocumentClick() {
      this.closeHeadMenu();
    },
    toggleHeadMenu() {
      this.headMenuOpen = !this.headMenuOpen;
    },
    closeHeadMenu() {
      this.headMenuOpen = false;
    },
    syncChatPrefsFromConv() {
      const hit = this.dmConversations.find(
        c => Number(c.id) === Number(this.selectedConvId)
      );
      this.chatPinned = !!(hit && hit.pinned);
      this.chatMuted = !!(hit && hit.muted);
    },
    applyConversationPatch(conv) {
      if (!conv || conv.id == null) return;
      const id = Number(conv.id);
      const list = [...this.dmConversations];
      const i = list.findIndex(c => Number(c.id) === id);
      if (i >= 0) {
        list[i] = { ...list[i], ...conv };
      } else {
        list.unshift(conv);
      }
      if (conv.pinned) {
        for (let j = 0; j < list.length; j++) {
          if (Number(list[j].id) !== id && list[j].pinned) {
            list[j] = { ...list[j], pinned: false };
          }
        }
      }
      list.sort((a, b) => {
        const pinA = a.pinned ? 1 : 0;
        const pinB = b.pinned ? 1 : 0;
        if (pinA !== pinB) return pinB - pinA;
        return String(b.last_message_at || "").localeCompare(
          String(a.last_message_at || "")
        );
      });
      this.dmConversations = list;
      if (Number(this.selectedConvId) === id) {
        this.syncChatPrefsFromConv();
      }
    },
    async patchDmSettings(body) {
      if (!this.selectedConvId || this.dmSettingsPatching) return null;
      this.dmSettingsPatching = true;
      try {
        const conv = await mbPatchDmConversationSettings(
          this.selectedConvId,
          body
        );
        this.applyConversationPatch(conv);
        return conv;
      } catch (e) {
        ElMessage.error((e && e.message) || "设置失败");
        return null;
      } finally {
        this.dmSettingsPatching = false;
      }
    },
    async onHeadMenuPin() {
      this.closeHeadMenu();
      if (this.dmSettingsPatching) return;
      const next = !this.chatPinned;
      const conv = await this.patchDmSettings({ pinned: next });
      if (conv) {
        ElMessage.success(next ? "已置顶聊天" : "已取消置顶");
      }
    },
    async onHeadMenuMute() {
      this.closeHeadMenu();
      if (this.dmSettingsPatching) return;
      const next = !this.chatMuted;
      const conv = await this.patchDmSettings({ muted: next });
      if (conv) {
        ElMessage.success(next ? "已开启免打扰" : "已关闭免打扰");
      }
    },
    onHeadMenuBlacklist() {
      this.closeHeadMenu();
      ElMessage.info("该功能即将开放");
    },
    onHeadMenuReport() {
      this.closeHeadMenu();
      ElMessage.info("该功能即将开放");
    },
    async onHeadMenuResetAgent() {
      this.closeHeadMenu();
      if (!this.selectedConvId || this.resettingAgent) return;
      try {
        await ElMessageBox.confirm(
          "将删除与该 AI 的所有聊天记录，并重新发送一条欢迎语。",
          "重新开始对话",
          {
            confirmButtonText: "确定清空",
            cancelButtonText: "取消",
            type: "warning"
          }
        );
      } catch {
        return;
      }
      this.resettingAgent = true;
      try {
        const res = await mbResetDmAgentConversation(this.selectedConvId);
        if (res && res.conversation) {
          this.applyConversationPatch(res.conversation);
        }
        this.clearAgentReplyTimer();
        this.chatAwaitingAgent = false;
        this.chatNextCursor = "";
        await this.loadChatMessages(false);
        this.$nextTick(() => this.scrollChatToBottom());
        ElMessage.success("已清空记录，对话已重新开始");
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      } finally {
        this.resettingAgent = false;
      }
    },
    closeBlacklistDialog() {
      this.blacklistDialogOpen = false;
    },
    async confirmBlacklist() {
      const peerId = Number(this.selectedPeerId) || 0;
      if (!peerId || this.blacklistSubmitting) return;
      this.blacklistSubmitting = true;
      try {
        await mbBlockUser(peerId);
        this.blacklistDialogOpen = false;
        this.selectedConvId = null;
        this.selectedPeerId = 0;
        this.selectedPeerName = "";
        this.chatMessages = [];
        ElMessage.success("已加入黑名单");
        await this.loadConversations();
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      } finally {
        this.blacklistSubmitting = false;
      }
    },
    goToItem(item) {
      var path;
      var q = {};
      if (item.cover || item.cover_url) {
        var vid1 = item.id || item.video_id;
        if (vid1) path = "/video/" + vid1;
      } else if (item.type) {
        var vid2 = item.video_id;
        if (vid2) { path = "/video/" + vid2; q.t = item.video_time || 0; }
      } else {
        var vid3 = item.video_id;
        if (vid3) { path = "/video/" + vid3; q.mb_cid = item.id; }
      }
      if (!path) return;
      this.$router.push({ path: path, query: q }).catch(function(e) {});
    },
    formatVideoTime(sec) {
      if (sec == null || sec === undefined) return "";
      var m = Math.floor(sec / 60);
      var s = Math.floor(sec % 60);
      return m + ":" + String(s).padStart(2, "0");
    },
    formatPlayCount(n) {
      if (n >= 10000) return (n / 10000).toFixed(1) + "万";
      if (n >= 1000) return (n / 1000).toFixed(1) + "千";
      return String(n);
    },
    onAvatarError(e) {
      if (e.target.src !== defaultFace) {
        e.target.src = defaultFace;
      }
    },
    async sendChatMessage() {
      const text = this.chatDraftTrimmed;
      if (
        !text ||
        !this.selectedConvId ||
        this.chatPosting ||
        this.chatAwaitingAgent
      ) {
        return;
      }
      const awaitAgent = this.selectedIsAgent;
      if (awaitAgent && (!this.chatWs || this.chatWs.readyState !== WebSocket.OPEN)) {
        this.connectChatWs();
      }
      this.chatPosting = true;
      try {
        const msg = await mbPostDmMessage(this.selectedConvId, text);
        this.chatDraft = "";
        this.appendMessageIfNew(msg);
        this.upsertConversationFromMessage(msg);
        if (awaitAgent) {
          this.startAgentReplyWait();
        }
      } catch (e) {
        ElMessage.error((e && e.message) || "发送失败");
      } finally {
        this.chatPosting = false;
      }
    }
  }
};
</script>

<style lang="scss" src="@/pages/minibili/messages-dm-chat.scss"></style>
