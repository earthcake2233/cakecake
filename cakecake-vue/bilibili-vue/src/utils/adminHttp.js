import axios from "axios";
import { ElMessage } from "element-plus";
import {
  clearAdminTokens,
  getAdminAccessToken,
  getAdminRefreshToken,
  setAdminTokens
} from "./adminAuth";

const remoteRaw = import.meta.env.VITE_REMOTE_API_BASE;
const baseURL =
  remoteRaw && String(remoteRaw).trim() !== ""
    ? String(remoteRaw).replace(/\/$/, "")
    : "";

const adminHttp = axios.create({
  baseURL,
  timeout: 20000
});

adminHttp.interceptors.request.use(config => {
  const t = getAdminAccessToken();
  if (t) {
    config.headers.Authorization = `Bearer ${t}`;
  }
  if (config.data instanceof FormData && config.headers) {
    delete config.headers["Content-Type"];
    delete config.headers["content-type"];
  }
  return config;
});

let refreshPromise = null;

async function refreshAdminToken() {
  const rt = getAdminRefreshToken();
  if (!rt) {
    return false;
  }
  const res = await axios.post(`${baseURL}/api/v1/admin/auth/refresh`, {
    refresh_token: rt
  });
  const body = res.data;
  if (!body || body.code !== 0 || !body.data) {
    return false;
  }
  setAdminTokens(body.data.access_token, body.data.refresh_token);
  return true;
}

adminHttp.interceptors.response.use(
  res => {
    const body = res.data;
    if (body && typeof body.code === "number" && body.code !== 0) {
      const err = new Error(body.msg || "请求失败");
      err.minibiliApiCode = body.code;
      return Promise.reject(err);
    }
    return body;
  },
  async err => {
    const cfg = err.config || {};
    const st = err.response && err.response.status;
    if (st === 401 && !cfg._adminRetry) {
      if (!refreshPromise) {
        refreshPromise = refreshAdminToken().finally(() => {
          refreshPromise = null;
        });
      }
      const ok = await refreshPromise;
      if (ok) {
        cfg._adminRetry = true;
        cfg.headers = cfg.headers || {};
        cfg.headers.Authorization = `Bearer ${getAdminAccessToken()}`;
        return adminHttp(cfg);
      }
      clearAdminTokens();
      if (typeof window !== "undefined") {
        window.location.hash = "#/admin/login";
      }
    }
    const msg =
      (err.response && err.response.data && err.response.data.msg) ||
      err.message ||
      "请求失败";
    if (!cfg.skipGlobalErrorToast) {
      ElMessage.error(msg);
    }
    return Promise.reject(err);
  }
);

export default adminHttp;
