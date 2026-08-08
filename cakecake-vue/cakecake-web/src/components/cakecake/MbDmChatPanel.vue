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
              <MbDmMessageItem
                :item="m"
                :feedback-map="_feedbackMap"
                :copied-msg-id="_copiedMsgId"
                @copy="copyMessage"
                @feedback="kind => setFeedback(m, kind)"
                @regenerate="regenerateReply"
                @switch-version="idx => switchVersion(m, idx)"
                @pick-suggestion="sendSuggestion"
                @open-item="goToItem"
                @avatar-error="onAvatarError"
              />
          </template>
          </template>
          <AgentToolActivities
            v-if="agentStream.chatAwaitingAgent"
            :acts="agentStream._liveToolActs"
            live
          />
          <div
            v-if="agentStream.chatAwaitingAgent && (!agentStream._agentDraftContent || agentStream._agentContinuePending)"
            class="msg-chat-loading msg-chat-loading--typing"
          >
            {{ agentStream._agentRegenerating ? "AI 正在重新生成…" : agentStream._agentContinueMode === "reprompt" ? "AI 正在补全回复…" : "AI 正在输入…" }}
          </div>
          <button
            v-if="agentStream.chatAwaitingAgent"
            type="button"
            class="msg-chat-stop"
            @click="stopAgentReply"
          >
            停止生成
          </button>
          <button
            v-if="agentStream._agentStopped"
            type="button"
            class="msg-chat-stop msg-chat-stop--primary"
            @click="continueAgentReply"
          >
            继续生成
          </button>
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
              :disabled="chatPosting || agentStream.chatAwaitingAgent || !chatDraftTrimmed"
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
  mbPostAgentFeedback,
  mbWsChatUrl
} from "@/api/cakecake";
import { getAccessToken, getUserId } from "@/utils/authTokens";
import defaultFace from "@/assets/akari.jpg";
import gochatIllus from "@/assets/gochat.png";
import muteIcon from "@/assets/mute.png";
import { refreshMessageUnread } from "@/utils/messageUnread";
import { buildVersionGroups, useAgentStreaming } from "@/composables/useAgentStreaming";
import { animateScrollTo, isNearBottom } from "@/composables/useChatScroll";
import AgentToolActivities from "./AgentToolActivities.vue";
import MbDmMessageItem from "./MbDmMessageItem.vue";

/** 每次向服务端请求的历史消息条数 */
const DM_MESSAGE_PAGE_SIZE = 30;

export default {
  name: "MbDmChatPanel",
  components: {
    AgentToolActivities,
    MbDmMessageItem
  },
  setup() {
    const stream = useAgentStreaming({
      notifyTimeout: () => {
        ElMessage.warning("续写未完成，可重试或复制已生成内容");
      }
    });
    return { agentStream: stream.state, stream };
  },
  props: {
    peerIdFromRoute: { type: Number, default: 0 }
  },
  data() {
    return {
        defaultAvatarSvg: defaultFace,
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
      _versionSel: {},
      _copiedMsgId: null,
      _copiedTimer: null,
      _feedbackMap: {},
      deletingConvId: 0,
      resettingAgent: false,
      chatDraft: "",
      chatWs: null,
      _chatWsRetryTimer: null,
      _wsReconnectAttempts: 0,
      _pendingWsControls: [],
      _userScrolledUp: false,
    };
  },
  computed: {
    threadRows() {
      return (this.dmConversations || []).map(c => ({
        id: Number(c.id),
        name: c.peer_name || "用户",
        snippet: this.stripPreviewMd(c.last_preview) || "暂无消息",
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
      const conv = this.dmConversations.find(
        c => Number(c.id) === Number(this.selectedConvId)
      );
      return buildVersionGroups(
        this.chatMessages,
        this._versionSel,
        this.agentStream,
        conv,
        () => this.agentFaceForDraft()
      );
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
    this.loadAgentFeedback();
    document.addEventListener("click", this.onDocumentClick);
  },
  updated() {
    this.$nextTick(() => {
      const el = this.$refs.chatScrollEl;
      if (el) this.enhanceRenderedMd(el);
    });
  },
  beforeUnmount() {
    this.clearAgentReplyTimer();
    this.disconnectChatWs();
    document.removeEventListener("click", this.onDocumentClick);
  },
  methods: {
    async copyMessage(m) {
      if (!m || !m.content) return;
      await this.copyPlainText(m.content);
      this._copiedMsgId = m.id;
      if (this._copiedTimer) clearTimeout(this._copiedTimer);
      this._copiedTimer = setTimeout(() => {
        this._copiedMsgId = null;
        this._copiedTimer = null;
      }, 1500);
    },
    async copyPlainText(text) {
      if (!text) return;
      try {
        await navigator.clipboard.writeText(text);
      } catch {
        const ta = document.createElement("textarea");
        ta.value = text;
        ta.style.position = "fixed";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.select();
        try {
          document.execCommand("copy");
        } catch {
          /* ignore */
        }
        document.body.removeChild(ta);
      }
    },
    enhanceRenderedMd(rootEl) {
      if (!rootEl) return;
      rootEl.querySelectorAll(".msg-chat-bubble--md pre").forEach(pre => {
        if (pre.querySelector(".msg-code-copy")) return;
        pre.style.position = "relative";
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "msg-code-copy";
        btn.textContent = "复制";
        btn.addEventListener("click", ev => {
          ev.stopPropagation();
          const code = pre.querySelector("code");
          const text = (code && code.innerText) || pre.innerText || "";
          void this.copyPlainText(text).then(() => {
            btn.textContent = "已复制";
            setTimeout(() => {
              btn.textContent = "复制";
            }, 1500);
          });
        });
        pre.appendChild(btn);
      });
    },
    async setFeedback(m, kind) {
      if (!m || m.id == null || m.id === "agent-draft") return;
      const key = String(m.id);
      const cur = this._feedbackMap[key] || "";
      const next = cur === kind ? "" : kind;
      this._feedbackMap = { ...this._feedbackMap, [key]: next };
      try {
        const payload = next === "" ? cur : next;
        await mbPostAgentFeedback(Number(m.id), payload);
        try {
          localStorage.setItem(
            "cakecake_agent_feedback",
            JSON.stringify(this._feedbackMap)
          );
        } catch {
          /* ignore */
        }
      } catch (e) {
        this._feedbackMap = { ...this._feedbackMap, [key]: cur };
        ElMessage.error((e && e.message) || "反馈失败，请稍后再试");
      }
    },
    loadAgentFeedback() {
      try {
        const s = localStorage.getItem("cakecake_agent_feedback");
        if (s) this._feedbackMap = JSON.parse(s) || {};
      } catch {
        /* ignore */
      }
    },
    sendSuggestion(text) {
      if (!text || this.chatPosting || this.agentStream.chatAwaitingAgent) return;
      this.chatDraft = text;
      void this.sendChatMessage();
    },
    sendWsControl(payload) {
      if (!this.chatWs || this.chatWs.readyState !== WebSocket.OPEN) {
        // Queue the control so a stop/continue clicked during a reconnect is
        // not silently lost (otherwise the LLM keeps streaming and the reply
        // appears fully generated at once).
        this._pendingWsControls.push(payload);
        this.connectChatWs();
        return;
      }
      try {
        this.chatWs.send(JSON.stringify(payload));
      } catch {
        /* ignore */
      }
    },
    stopAgentReply() {
      this.sendWsControl({ type: "agent_cancel" });
      // Keep the in-place rewrite flag: a paused regenerate must keep hiding
      // the previous version until the new reply is persisted, otherwise the
      // old content (and a duplicate draft bubble) would flash back.
      // _agentContinuing stays true so a paused continue can resume in place.
      this.stream.stop();
    },
    continueAgentReply() {
      if (this.agentStream.chatAwaitingAgent || this.agentStream._agentContinuePending) return;
      const cid = Number(this.selectedConvId);
      if (!cid) return;
      // The draft lives server-side now; the backend replays its buffer or
      // re-prompts from the draft, and tells us which mode via
      // agent_continue_mode.
      this.sendWsControl({ type: "agent_continue", conversation_id: cid });
      this.stream.startContinue();
      this.stream.startWait();
    },
    switchVersion(m, idx) {
      if (!m || !m.groupId || !m.versions) return;
      const max = m.versions.length - 1;
      const next = Math.max(0, Math.min(max, idx));
      if (next === m.versionIndex) return;
      this._versionSel = { ...this._versionSel, [m.groupId]: next };
    },
    regenerateReply() {
      if (this.agentStream.chatAwaitingAgent || this.chatPosting || this.agentStream._agentRegenerating) return;
      const cid = Number(this.selectedConvId);
      if (!cid) return;
      this.sendWsControl({
        type: "agent_regenerate",
        conversation_id: cid
      });
      this.stream.startRegenerate();
      this.stream.startWait();
      this.$nextTick(() => {
        const msgs = this.chatMessages || [];
        for (let i = msgs.length - 1; i >= 0; i--) {
          const raw = msgs[i];
          if (String(raw.role) !== "assistant" && !raw.is_agent) {
            this.scrollToMessageTop({ id: raw.id });
            // Once the jump animation settles, let the regenerated reply
            // auto-follow the typewriter (manual up-scroll still wins later).
            setTimeout(() => {
              this._userScrolledUp = false;
            }, 700);
            break;
          }
        }
      });
    },
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
      this.stream.reset();
      this._versionSel = {};
      this.closeHeadMenu();
      this.selectedConvId = cid;
      const hit = this.dmConversations.find(c => Number(c.id) === cid);
      this.selectedPeerName = hit ? hit.peer_name : "";
      this.selectedPeerId = hit ? Number(hit.peer_id) || 0 : 0;
      this.syncChatPrefsFromConv();
      this.chatMessages = [];
      this.chatNextCursor = "";
      this._userScrolledUp = false;
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
      if (!el) {
        return;
      }
      const nearBottom = isNearBottom(el);
      this._userScrolledUp = !nearBottom;
      if (!this.chatNextCursor || this.chatLoadingMore || this.chatLoading) {
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
          this.stream.reset();
          this._versionSel = {};
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
      if (!el || this._userScrolledUp) return;
      el.scrollTop = el.scrollHeight;
    },
    appendMessageIfNew(msg) {
      if (!msg || msg.id == null) return;
      const cid = Number(msg.conversation_id);
      if (cid !== Number(this.selectedConvId)) return;
      if (this.chatMessages.some(m => Number(m.id) === Number(msg.id))) return;
      this.chatMessages = [...this.chatMessages, msg];
      const me = getUserId();
      const isAgentReply =
        this.selectedIsAgent && me != null && Number(msg.sender_id) !== Number(me);
      if (
        isAgentReply
      ) {
        this.clearAgentReplyTimer();
        this.agentStream.chatAwaitingAgent = false;
      }
      this.$nextTick(() => {
        if (isAgentReply) {
          if (this.agentStream._agentLastAction === "continue") {
            this.scrollToMessageTop(msg);
          } else if (this.agentStream._agentLastAction === "new") {
            // Stream finished: land on the last question row (the reply's
            // first sentence is right below it) with a smooth upward move,
            // instead of staying glued to the answer's tail.
            this._userScrolledUp = false;
            const lastUser = [...this.chatMessages]
              .reverse()
              .find(
                m =>
                  String(m.role) === "user" && Number(m.id) < Number(msg.id)
              );
            if (lastUser) {
              this.scrollToMessageTop(lastUser);
            } else {
              this.scrollToMessageTop(msg);
            }
          } else if (this.agentStream._agentLastAction !== "regenerate") {
            this.scrollChatToBottom();
          }
          // regenerate: keep the view where the in-place rewrite happened.
          this.agentStream._agentLastAction = "";
        } else {
          this.scrollChatToBottom();
        }
      });
    },
    scrollToMessageTop(msg) {
      const el = this.$refs.chatScrollEl;
      if (!el || !msg || msg.id == null) return;
      const target = el.querySelector(`.msg-chat-row[data-msg-id="${String(msg.id)}"]`);
      if (!target) {
        this.scrollChatToBottom();
        return;
      }
      const top =
        target.getBoundingClientRect().top -
        el.getBoundingClientRect().top +
        el.scrollTop -
        12;
      animateScrollTo(el, Math.max(0, top));
    },
    clearAgentReplyTimer() {
      this.stream.clearWait();
    },
    startAgentReplyWait() {
      this.stream.startWait();
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
        const pending = this._pendingWsControls || [];
        this._pendingWsControls = [];
        for (const p of pending) {
          try {
            this.chatWs.send(JSON.stringify(p));
          } catch {
            /* ignore */
          }
        }
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
        this.stream.onToolStart(data.body);
        return;
      }
      if (data.type === "tool_call_end" && data.body) {
        this.stream.onToolEnd(data.body);
        return;
      }
      if (data.type === "tool_result_data" && data.body) {
        this.stream.onToolResult(data.body.span_id, data.body.items);
        return;
      }
      if (data.type === "agent_continue_mode" && data.mode) {
        this.stream.setMode(data.mode);
        return;
      }
      if (data.type === "agent_delta" && data.body && typeof data.body.content === "string") {
        if (this.agentStream._agentStopped) return;
        this.stream.onDelta(data.body.content);
        this.$nextTick(() => this.scrollChatToBottom());
        return;
      }
      if (
        data.type === "agent_suggestions" &&
        data.message_id &&
        Array.isArray(data.suggestions)
      ) {
        const mid = Number(data.message_id);
        const idx = this.chatMessages.findIndex(
          (m) => Number(m.id) === mid
        );
        if (idx >= 0) {
          this.chatMessages = this.chatMessages.map((m, i) =>
            i === idx
              ? { ...m, suggestions: JSON.stringify(data.suggestions) }
              : m
          );
        }
        return;
      }
      if (!data || typeof data !== "object") return;
      if (data.type === "dm_message" && data.message) {
        // dm_message means the LLM finished processing all results: clear the
        // streaming draft and any in-flight tool activities.
        this.stream.onFinalMessage();
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
        this.stream.reset();
        this._versionSel = {};
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
    stripPreviewMd(s) {
      return String(s || "")
        .replace(/```[\s\S]*?```/g, " ")
        .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
        .replace(/\*\*|__|`/g, "")
        .replace(/^[#>+\-*]\s*/gm, "")
        .replace(/\s+/g, " ")
        .trim();
    },
    agentFaceForDraft() {
      const conv = this.dmConversations.find(
        c => Number(c.id) === Number(this.selectedConvId)
      );
      if (conv && conv.peer_avatar) return conv.peer_avatar;
      const msgs = this.chatMessages || [];
      for (let i = msgs.length - 1; i >= 0; i--) {
        const m = msgs[i];
        if (m.sender_avatar && String(m.role) === "assistant") {
          return m.sender_avatar;
        }
      }
      return defaultFace;
    },
    onAvatarError(e) {
      if (!e.target._fallback) {
        e.target._fallback = true;
        e.target.src = this.defaultAvatarSvg;
      }
    },
    async sendChatMessage() {
      const text = this.chatDraftTrimmed;
      if (
        !text ||
        !this.selectedConvId ||
        this.chatPosting ||
        this.agentStream.chatAwaitingAgent
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
        this._userScrolledUp = false;
        this.stream.reset();
        this.agentStream._agentLastAction = "new";
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

<style lang="scss" src="@/pages/cakecake/messages-dm-chat.scss"></style>
