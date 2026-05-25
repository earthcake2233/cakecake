import { ElMessage } from "element-plus";

/** 与 B 站一致的深色居中 Toast（绕过 main.js 对 ElMessage.success 的禁用） */
export const MB_DARK_TOAST_CLASS = "mb-dark-toast";

/**
 * @param {string} message
 * @param {number} [duration=2000]
 */
export function showMbDarkToast(message, duration = 2000) {
  ElMessage({
    message,
    duration,
    customClass: MB_DARK_TOAST_CLASS
  });
}
