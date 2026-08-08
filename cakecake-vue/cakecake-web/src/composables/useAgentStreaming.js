import { getUserId } from "@/utils/authTokens";
import defaultFace from "@/assets/akari.jpg";

/**
 * Agent streaming state machine (pause / continue / regenerate + draft + tool
 * activity). Extracted from MbDmChatPanel so the panel stays a renderer and
 * the transitions are testable pure helpers.
 */

export function createAgentStreamState() {
  return {
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

export function clearReplyTimer(state) {
  if (!state || !state._agentReplyTimer) return;
  clearTimeout(state._agentReplyTimer);
  state._agentReplyTimer = null;
}

export function startReplyWait(state, notifyTimeout) {
  clearReplyTimer(state);
  state.chatAwaitingAgent = true;
  state._agentReplyTimer = setTimeout(() => {
    state.chatAwaitingAgent = false;
    state._agentReplyTimer = null;
    if (state._agentContinuePending && state._agentDraftContent) {
      state._agentContinuePending = false;
      state._agentStopped = true;
      if (typeof notifyTimeout === "function") notifyTimeout();
    }
  }, 120000);
}

export function markStop(state) {
  state.chatAwaitingAgent = false;
  state._agentStopped = true;
  state._agentContinuePending = false;
  state._pendingToolActs = [];
  state._liveToolActs = [];
  state._pendingResultData = {};
}

export function markContinueStart(state) {
  state._agentStopped = false;
  state._agentContinuePending = true;
  state._agentContinuing = true;
  state._agentLastAction = "continue";
}

export function markRegenerate(state) {
  state._agentDraftContent = "";
  state._agentStopped = false;
  state._agentRegenerating = true;
  state._agentLastAction = "regenerate";
  state._pendingToolActs = [];
  state._liveToolActs = [];
  state._pendingResultData = {};
}

export function applyAgentDelta(state, content) {
  state._agentContinuePending = false;
  state._agentDraftContent += content;
}

export function setContinueMode(state, mode) {
  if (mode === "buffer" || mode === "reprompt") {
    state._agentContinueMode = mode;
  }
}

export function trackToolStart(state, body) {
  const act = { ...body, status: "running" };
  state._pendingToolActs.push(act);
  state._liveToolActs.push(act);
}

export function trackToolEnd(state, body) {
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

export function trackToolResult(state, spanId, items) {
  state._pendingResultData[spanId] = items;
}

export function finalizeAgentMessage(state) {
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

export function resetAgentStream(state) {
  clearReplyTimer(state);
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
