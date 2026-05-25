import "axios";

declare module "axios" {
  interface AxiosRequestConfig {
    /** 为 true 时不弹出 http 拦截器里的全局 ElMessage（由页面自行提示） */
    skipGlobalErrorToast?: boolean;
  }
}
