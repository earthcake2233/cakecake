import { getAccessToken } from "@/utils/authTokens";
import {
  mbAddMySearchHistory,
  mbGetMySearchHistory,
  mbPutMySearchHistory
} from "@/api/minibili";

const STORAGE_KEY = "minibili_search_history";
const MAX_ITEMS = 20;

/** 与后端 searchhist.Norm 一致：小写、去空格 */
export function searchKeywordNorm(keyword) {
  return String(keyword ?? "")
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "");
}

function normalizeList(arr) {
  const seen = new Set();
  const out = [];
  for (const item of arr || []) {
    const k = String(item ?? "").trim();
    if (!k) {
      continue;
    }
    const norm = searchKeywordNorm(k);
    if (!norm || seen.has(norm)) {
      continue;
    }
    seen.add(norm);
    out.push(k);
    if (out.length >= MAX_ITEMS) {
      break;
    }
  }
  return out;
}

function loadLocalSearchHistory() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return [];
    }
    const arr = JSON.parse(raw);
    return normalizeList(Array.isArray(arr) ? arr : []);
  } catch {
    return [];
  }
}

function persistLocal(list) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(normalizeList(list)));
  } catch {
    /* ignore quota */
  }
}

/** 合并：服务端优先，再补本地未同步项 */
function mergeHistory(serverList, localList) {
  return normalizeList([...(serverList || []), ...(localList || [])]);
}

let syncPromise = null;

/** 登录用户：拉取服务端历史并与本地合并 */
export async function syncSearchHistoryFromServer() {
  if (!getAccessToken()) {
    return loadLocalSearchHistory();
  }
  if (syncPromise) {
    return syncPromise;
  }
  syncPromise = (async () => {
    try {
      const local = loadLocalSearchHistory();
      const res = await mbGetMySearchHistory();
      const server = normalizeList(res.keywords);
      let merged = server;
      if (local.length) {
        merged = mergeHistory(server, local);
        if (merged.length) {
          await mbPutMySearchHistory(merged);
        }
      }
      persistLocal(merged);
      return merged;
    } catch {
      return loadLocalSearchHistory();
    } finally {
      syncPromise = null;
    }
  })();
  return syncPromise;
}

export function loadSearchHistory() {
  return loadLocalSearchHistory();
}

export async function loadSearchHistoryAsync() {
  if (getAccessToken()) {
    return syncSearchHistoryFromServer();
  }
  return loadLocalSearchHistory();
}

export function addSearchHistory(keyword) {
  const k = String(keyword ?? "").trim();
  if (!k) {
    return loadLocalSearchHistory();
  }
  const norm = searchKeywordNorm(k);
  const next = [
    k,
    ...loadLocalSearchHistory().filter(x => searchKeywordNorm(x) !== norm)
  ].slice(0, MAX_ITEMS);
  persistLocal(next);
  if (getAccessToken()) {
    void mbAddMySearchHistory(k)
      .then(res => {
        if (res && Array.isArray(res.keywords)) {
          persistLocal(normalizeList(res.keywords));
        }
      })
      .catch(() => {});
  }
  return next;
}

export async function addSearchHistoryAsync(keyword) {
  return addSearchHistory(keyword);
}

export function removeSearchHistoryAt(index) {
  const list = loadLocalSearchHistory();
  const i = Number(index);
  if (!Number.isFinite(i) || i < 0 || i >= list.length) {
    return list;
  }
  list.splice(i, 1);
  persistLocal(list);
  if (getAccessToken()) {
    void mbPutMySearchHistory(list).catch(() => {});
  }
  return list;
}
