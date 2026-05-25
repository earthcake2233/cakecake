import { mbUnreadSummary } from "@/api/minibili";
import { getAccessToken } from "@/utils/authTokens";

const listeners = new Set();
let lastSummary = null;

export function getLastMessageUnreadSummary() {
  return lastSummary;
}

export function subscribeMessageUnread(fn) {
  listeners.add(fn);
  if (lastSummary) {
    try {
      fn(lastSummary);
    } catch {
      /* ignore */
    }
  }
  return () => listeners.delete(fn);
}

function notify(summary) {
  listeners.forEach(fn => {
    try {
      fn(summary);
    } catch {
      /* ignore */
    }
  });
}

/** 拉取并广播各分类未读数；未登录时清零 */
export async function refreshMessageUnread() {
  if (!getAccessToken()) {
    lastSummary = null;
    notify(null);
    return null;
  }
  try {
    const summary = await mbUnreadSummary();
    lastSummary = summary || {};
    notify(lastSummary);
    return lastSummary;
  } catch {
    return lastSummary;
  }
}
