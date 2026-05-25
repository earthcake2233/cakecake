import { ElMessage } from "element-plus";
import { mbToggleWatchLater } from "@/api/minibili";
import { getAccessToken } from "@/utils/authTokens";

export function openLoginModal(store) {
  store.commit("login/SET_LOGIN_TAB", 0);
  store.commit("login/OPEN_LOGIN_MODAL");
}

/** @returns {boolean|null} 切换后是否在稍后再看；未登录或失败为 null */
export async function toggleWatchLaterVideo(store, videoId) {
  const id = Number(videoId);
  if (!Number.isFinite(id) || id <= 0) {
    return null;
  }
  if (!getAccessToken()) {
    openLoginModal(store);
    return null;
  }
  try {
    const res = await mbToggleWatchLater(id);
    const on = !!res.in_watch_later;
    ElMessage.success(on ? "已加入稍后再看" : "已移出稍后再看");
    return on;
  } catch (e) {
    ElMessage.error((e && e.message) || "稍后再看操作失败");
    return null;
  }
}
