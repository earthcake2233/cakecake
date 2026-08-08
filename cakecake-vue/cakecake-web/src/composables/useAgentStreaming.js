import { reactive } from "vue";
import { getUserId } from "@/utils/authTokens";
import defaultFace from "@/assets/akari.jpg";

/**
 * Agent streaming state machine (pause / continue / regenerate + draft + tool
 * activity) as a true Vue 3 composable: the state is reactive and every
 * transition lives here, so MbDmChatPanel only renders and forwards transport
 * events.
 */
export function useAgentStreaming({ notifyTimeout } = {}) {
  const state = reactive({
    chatAwaitingAgent: false,
    _agentDraftContent: "",
    _agentStopped: false,
    _agentContinuePending: false,
    _agentContinuing: false,
    _agentRegenerating: false,
    _agentLastAction: "",
    _agentReplyTimer: null,
    _agentContinueMode: "buffer",
    _pendingResultData: {},
    _pendingToolActs: [],
    _liveToolActs: []
  });

  function clearWait() {
    if (!state._agentReplyTimer) return;
    clearTimeout(state._agentReplyTimer);
    state._agentReplyTimer = null;
  }

  function startWait() {
    clearWait();
    state.chatAwaitingAgent = true;
    state._agentReplyTimer = setTimeout(() => {
      state.chatAwaitingAgent = false;
      state._agentReplyTimer = null;
      if (state._agentContinuePending && state._agentDraftContent) {
        state._agentContinuePending = false;
        state._agentStopped = true;
        if (typeof notifyTimeout === "function") notifyTimeout();
      }
      if (state._agentRegenerating) {
        // Safety net: a regenerate that produced no reply (e.g. a conversation
        // with no user message) must never leave the bubble stuck forever.
        state._agentRegenerating = false;
        if (typeof notifyTimeout === "function") notifyTimeout();
      }
    }, 120000);
  }

  function stop() {
    state.chatAwaitingAgent = false;
    state._agentStopped = true;
    state._agentContinuePending = false;
    state._pendingToolActs = [];
    state._liveToolActs = [];
    state._pendingResultData = {};
  }

  function startContinue() {
    state._agentStopped = false;
    state._agentContinuePending = true;
    state._agentContinuing = true;
    state._agentLastAction = "continue";
  }

  function startRegenerate() {
    state._agentDraftContent = "";
    state._agentStopped = false;
    state._agentRegenerating = true;
    state._agentLastAction = "regenerate";
    state._pendingToolActs = [];
    state._liveToolActs = [];
    state._pendingResultData = {};
  }

  function onDelta(content) {
    state._agentContinuePending = false;
    state._agentDraftContent += content;
  }

  function setMode(mode) {
    if (mode === "buffer" || mode === "reprompt") {
      state._agentContinueMode = mode;
    }
  }

  function onToolStart(body) {
    const act = { ...body, status: "running" };
    state._pendingToolActs.push(act);
    state._liveToolActs.push(act);
  }

  function onToolEnd(body) {
    const done = { ...body, status: "done" };
    const idx = state._pendingToolActs.findIndex(t => t.span_id === body.span_id);
    if (idx >= 0) {
      state._pendingToolActs[idx] = { ...state._pendingToolActs[idx], ...done };
      state._liveToolActs[idx] = { ...state._liveToolActs[idx], ...done };
    } else {
      state._pendingToolActs.push(done);
      state._liveToolActs.push(done);
    }
  }

  function onToolResult(spanId, items) {
    state._pendingResultData[spanId] = items;
  }

  function onFinalMessage() {
    state._agentDraftContent = "";
    state._agentStopped = false;
    state._agentContinuePending = false;
    state._agentContinuing = false;
    state._agentRegenerating = false;
    if (state._pendingToolActs.length) {
      state._pendingToolActs.forEach(t => {
        if (t.status === "running") t.status = "done";
      });
      state._liveToolActs.forEach(t => {
        if (t.status === "running") t.status = "done";
      });
      state._pendingToolActs = [];
      state._liveToolActs = [];
      state._pendingResultData = {};
    }
  }

  function reset() {
    clearWait();
    state.chatAwaitingAgent = false;
    state._agentDraftContent = "";
    state._agentStopped = false;
    state._agentContinuing = false;
    state._agentRegenerating = false;
    state._agentContinuePending = false;
    state._agentLastAction = "";
    state._agentContinueMode = "buffer";
    state._pendingToolActs = [];
    state._liveToolActs = [];
    state._pendingResultData = {};
  }

  return {
    state,
    clearWait,
    startWait,
    stop,
    startContinue,
    startRegenerate,
    onDelta,
    setMode,
    onToolStart,
    onToolEnd,
    onToolResult,
    onFinalMessage,
    reset
  };
}

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

/**
 * Pure render helper: converts raw DM messages into time-grouped bubbles and
 * merges consecutive assistant rows into version bubbles. Never mutates the
 * input messages; draft bubbles are appended when a stream is active.
 */
export function buildVersionGroups(messages, versionSel, stream, conv, agentFaceForDraft) {
  const me = getUserId();
  const groups = [];
  let curLabel = "";
  let curMsgs = [];
  const flush = () => {
    if (curMsgs.length) {
      groups.push({ label: curLabel, messages: curMsgs });
    }
  };
  let pendingAgent = null;
  for (const raw of messages || []) {
    const d = parseApiTime(raw.created_at);
    const label = `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日 ${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
    const isMine = me != null && Number(raw.sender_id) === Number(me);
    const isAgent = raw.role === "assistant" || !!raw.is_agent;
    const item = {
      id: raw.id,
      content: raw.content,
      face: isAgent
        ? raw.sender_avatar || (conv && conv.peer_avatar) || defaultFace
        : raw.sender_avatar || defaultFace,
      is_mine: isMine,
      is_agent: isAgent,
      toolActivities: raw.tool_activities
        ? JSON.parse(raw.tool_activities)
        : raw._toolActivities || [],
      toolResultData: raw.tool_result_data
        ? JSON.parse(raw.tool_result_data)
        : raw._toolResultData || {},
      suggestions: raw.suggestions ? JSON.parse(raw.suggestions) : raw._suggestions || []
    };
    if (isAgent && pendingAgent) {
      pendingAgent.versions.push(raw.content);
      pendingAgent.versionIds.push(raw.id);
      pendingAgent.versionTools.push(item.toolActivities);
      pendingAgent.versionResults.push(item.toolResultData);
      pendingAgent.versionSuggestions.push(item.suggestions);
      pendingAgent.face = item.face;
      continue;
    }
    if (isAgent) {
      item.groupId = String(raw.id);
      item.versions = [raw.content];
      item.versionIds = [raw.id];
      item.versionTools = [item.toolActivities];
      item.versionResults = [item.toolResultData];
      item.versionSuggestions = [item.suggestions];
      pendingAgent = item;
    } else {
      pendingAgent = null;
    }
    if (label !== curLabel) {
      flush();
      curLabel = label;
      curMsgs = [item];
    } else {
      curMsgs.push(item);
    }
  }
  flush();
  for (const grp of groups) {
    for (const m of grp.messages) {
      if (!m.is_agent || !m.versions || m.versions.length <= 1) continue;
      const rawSel = versionSel[m.groupId];
      const sel =
        rawSel == null || rawSel < 0 || rawSel >= m.versions.length
          ? m.versions.length - 1
          : rawSel;
      m.versionIndex = sel;
      m.content = m.versions[sel];
      m.id = m.versionIds[sel];
      m.toolActivities = m.versionTools[sel];
      m.toolResultData = m.versionResults[sel];
      m.suggestions = m.versionSuggestions[sel];
    }
  }
  const draftText = stream._agentDraftContent || "";
  let lastAgent = null;
  for (const grp of groups) {
    for (const m of grp.messages) {
      if (m.is_agent) lastAgent = m;
    }
  }
  const inPlaceDraft = stream._agentRegenerating && lastAgent;
  if ((stream.chatAwaitingAgent || stream._agentStopped) && draftText) {
    if (inPlaceDraft) {
      lastAgent.content = draftText;
      lastAgent.isStreaming = true;
      lastAgent.toolActivities = [];
      lastAgent.toolResultData = {};
      lastAgent.suggestions = [];
    } else {
      curMsgs.push({
        id: "agent-draft",
        content: draftText,
        face: agentFaceForDraft(),
        is_mine: false,
        is_agent: true,
        toolActivities: [],
        toolResultData: {},
        isStreaming: true
      });
    }
  } else if (stream._agentRegenerating && lastAgent) {
    lastAgent.content = "";
    lastAgent.isStreaming = true;
    lastAgent.toolActivities = [];
    lastAgent.toolResultData = {};
    lastAgent.suggestions = [];
  }
  return groups;
}
