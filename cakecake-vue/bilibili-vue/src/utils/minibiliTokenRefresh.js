import axios from "axios";
import { getAccessToken, getRefreshToken, setTokens } from "./authTokens";

const isMinibili =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

const remoteRaw = import.meta.env.VITE_REMOTE_API_BASE;
const remoteTrim =
  remoteRaw && String(remoteRaw).trim() !== ""
    ? String(remoteRaw).replace(/\/$/, "")
    : "";

const apiBase = isMinibili ? remoteTrim || "" : "";

const REFRESH_LOCK_KEY = "minibili_refresh_lock";
const REFRESH_LOCK_TTL_MS = 20000;
const tabId = `${Date.now()}-${Math.random().toString(36).slice(2)}`;

let refreshPromise = null;

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

function readRefreshLock() {
  try {
    const raw = localStorage.getItem(REFRESH_LOCK_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function writeRefreshLock() {
  try {
    localStorage.setItem(
      REFRESH_LOCK_KEY,
      JSON.stringify({ tabId, at: Date.now() })
    );
  } catch {
    /* private mode */
  }
}

function clearRefreshLock() {
  const lock = readRefreshLock();
  if (lock && lock.tabId === tabId) {
    try {
      localStorage.removeItem(REFRESH_LOCK_KEY);
    } catch {
      /* ignore */
    }
  }
}

/** access 将过期或已过期（默认提前 60s 视为需续期） */
export function isAccessTokenExpired(skewSec = 60) {
  const t = getAccessToken();
  if (!t) {
    return true;
  }
  try {
    const body = t.split(".")[1];
    const b64 = body.replace(/-/g, "+").replace(/_/g, "/");
    const pad = b64.length % 4 ? "=".repeat(4 - (b64.length % 4)) : "";
    const p = JSON.parse(atob(b64 + pad));
    if (p.exp == null) {
      return false;
    }
    return Date.now() / 1000 >= Number(p.exp) - skewSec;
  } catch {
    return true;
  }
}

export function shouldAttemptTokenRefresh(err, config) {
  if (!isMinibili || !config) {
    return false;
  }
  if (config._mbTokenRefresh) {
    return false;
  }
  if (config.skipSessionInvalidate) {
    return false;
  }
  const url = String(config.url || "");
  if (/\/auth\/(login|refresh)\b/.test(url)) {
    return false;
  }
  if (!getRefreshToken()) {
    return false;
  }
  const st = err.response && err.response.status;
  const body = err.response && err.response.data;
  const code = body && typeof body.code === "number" ? body.code : null;
  const biz = err.minibiliApiCode;
  if (st === 401 || code === 40100 || biz === 40100) {
    return true;
  }
  return false;
}

/** 另一标签页正在 refresh 时，等待其写入新 access */
async function waitForCrossTabRefresh(prevAccess) {
  const deadline = Date.now() + REFRESH_LOCK_TTL_MS;
  while (Date.now() < deadline) {
    await sleep(250);
    const lock = readRefreshLock();
    if (!lock || Date.now() - lock.at > REFRESH_LOCK_TTL_MS) {
      break;
    }
    const next = getAccessToken();
    if (next && next !== prevAccess && !isAccessTokenExpired()) {
      return true;
    }
  }
  return !!getAccessToken() && !isAccessTokenExpired();
}

async function performRefreshRequest() {
  const rt = getRefreshToken();
  if (!rt) {
    return false;
  }
  const r = await axios.post(
    `${apiBase}/api/v1/auth/refresh`,
    { refresh_token: rt },
    {
      headers: { "Content-Type": "application/json" },
      timeout: 15000
    }
  );
  const data = r.data;
  if (data && data.code === 0 && data.data) {
    setTokens(data.data.access_token, data.data.refresh_token);
    return true;
  }
  return false;
}

/** 发请求前主动续期 access（有 refresh 且 access 将过期时） */
export async function ensureFreshAccessToken() {
  if (!isMinibili) {
    return false;
  }
  if (!getRefreshToken()) {
    return false;
  }
  if (!isAccessTokenExpired()) {
    return true;
  }
  return refreshMinibiliAccessToken();
}

/** 用 refresh_token 换新 access（单飞；多标签页通过 lock 协调） */
export async function refreshMinibiliAccessToken() {
  if (!isMinibili) {
    return false;
  }
  if (!getRefreshToken()) {
    return false;
  }
  if (!refreshPromise) {
    refreshPromise = (async () => {
      const prevAccess = getAccessToken();
      const lock = readRefreshLock();
      const now = Date.now();
      if (
        lock &&
        lock.tabId !== tabId &&
        now - lock.at < REFRESH_LOCK_TTL_MS
      ) {
        return waitForCrossTabRefresh(prevAccess);
      }
      writeRefreshLock();
      try {
        return await performRefreshRequest();
      } catch {
        return false;
      } finally {
        clearRefreshLock();
      }
    })().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

/** 页面可见 / 定时检查：避免长时间挂着 tab 后 access 过期 */
export function installMinibiliTokenAutoRefresh() {
  if (!isMinibili || typeof document === "undefined") {
    return;
  }
  const tick = () => {
    if (localStorage.getItem("signIn") !== "1") {
      return;
    }
    if (!getRefreshToken()) {
      return;
    }
    if (isAccessTokenExpired()) {
      void refreshMinibiliAccessToken();
    }
  };
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible") {
      tick();
    }
  });
  window.setInterval(tick, 45 * 60 * 1000);
}
