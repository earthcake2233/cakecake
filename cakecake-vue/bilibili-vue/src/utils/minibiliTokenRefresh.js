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

let refreshPromise = null;

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

/** 用 refresh_token 换新 access（单飞，并发 401 只刷一次） */
export async function refreshMinibiliAccessToken() {
  if (!isMinibili) {
    return false;
  }
  const rt = getRefreshToken();
  if (!rt) {
    return false;
  }
  if (!refreshPromise) {
    refreshPromise = (async () => {
      try {
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
      } catch {
        return false;
      }
    })().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}
