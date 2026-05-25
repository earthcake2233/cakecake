import axios from "axios";
import { ElMessage } from "element-plus";
import { getAccessToken } from "./authTokens";
import { invalidateMinibiliSessionFromHttp } from "./minibiliAuthSync";
import {
  refreshMinibiliAccessToken,
  shouldAttemptTokenRefresh
} from "./minibiliTokenRefresh";
import { extractApiErrorMessage } from "./apiErrorMessage";

const isMinibili =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

const remoteRaw = import.meta.env.VITE_REMOTE_API_BASE;
const remoteTrim =
  remoteRaw && String(remoteRaw).trim() !== ""
    ? String(remoteRaw).replace(/\/$/, "")
    : "";

/** 未配置 VITE_REMOTE_API_BASE 时走相对路径，便于 Vite dev 代理到后端（Rule R-FE-7） */
/** 非 Minibili 演示模式：请配置 VITE_REMOTE_API_BASE，勿使用已废弃的第三方 Mock 域名 */
const defaultBase = isMinibili ? remoteTrim || "" : remoteTrim || "";

const service = axios.create({
  withCredentials: false,
  baseURL: defaultBase,
  timeout: 15000
});

function shouldInvalidateMinibiliSession(err) {
  if (!isMinibili || !err) {
    return false;
  }
  const cfg = err.config || {};
  if (cfg.skipSessionInvalidate) {
    return false;
  }
  const url = String(cfg.url || "");
  if (/\/auth\/(login|refresh)\b/.test(url)) {
    return false;
  }
  const st = err.response && err.response.status;
  const body = err.response && err.response.data;
  const code = body && typeof body.code === "number" ? body.code : null;
  const biz = err.minibiliApiCode;
  if (st === 401) {
    return true;
  }
  if (code === 40100 || biz === 40100) {
    return true;
  }
  return false;
}

service.interceptors.request.use(
  config => {
    /** FormData 必须由浏览器自动带 multipart boundary；勿沿用 application/json */
    if (config.data instanceof FormData && config.headers) {
      if (typeof config.headers.delete === "function") {
        config.headers.delete("Content-Type");
        config.headers.delete("content-type");
      } else {
        delete config.headers["Content-Type"];
        delete config.headers["content-type"];
      }
    }
    if (isMinibili) {
      const t = getAccessToken();
      if (t && !config.headers.Authorization) {
        config.headers.Authorization = `Bearer ${t}`;
      }
    }
    return config;
  },
  error => {
    ElMessage({
      message: "加载超时",
      type: "error",
      center: true
    });
    return Promise.reject(error);
  }
);

service.interceptors.response.use(
  response => {
    const data = response.data;
    if (
      isMinibili &&
      data &&
      typeof data.code === "number" &&
      data.code !== 0
    ) {
      const msg = data.msg || "请求失败";
      /** 挂上 config / response，便于鉴权失败时统一登出 */
      return Promise.reject(
        Object.assign(new Error(msg), {
          config: response.config,
          response,
          minibiliApiCode:
            data && typeof data.code === "number" ? data.code : undefined
        })
      );
    }
    return data;
  },
  async error => {
    const config = error.config;
    if (shouldAttemptTokenRefresh(error, config)) {
      const ok = await refreshMinibiliAccessToken();
      if (ok) {
        config._mbTokenRefresh = true;
        config.headers = config.headers || {};
        config.headers.Authorization = `Bearer ${getAccessToken()}`;
        return service(config);
      }
    }
    if (shouldInvalidateMinibiliSession(error)) {
      invalidateMinibiliSessionFromHttp();
    }
    const msg = extractApiErrorMessage(error, "加载失败");
    const skip = error.config && error.config.skipGlobalErrorToast;
    if (!skip) {
      ElMessage({
        message: String(msg),
        type: "error",
        center: true
      });
    }
    const enriched =
      error instanceof Error ? error : new Error(String(msg));
    enriched.message = msg;
    if (error.config) enriched.config = error.config;
    if (error.response) enriched.response = error.response;
    if (error.minibiliApiCode != null) {
      enriched.minibiliApiCode = error.minibiliApiCode;
    }
    return Promise.reject(enriched);
  }
);

export default service;
