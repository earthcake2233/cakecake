/** 生产环境可关闭网页端视频文件上传（仍允许保存稿件元数据到服务端） */

function envTruthy(v) {
  const s = String(v ?? "")
    .trim()
    .toLowerCase();
  return s === "true" || s === "1" || s === "yes" || s === "on";
}

export function isVideoUploadDisabled() {
  return envTruthy(import.meta.env.VITE_VIDEO_UPLOAD_DISABLED);
}

export const VIDEO_UPLOAD_DISABLED_TITLE =
  import.meta.env.VITE_VIDEO_UPLOAD_DISABLED_TITLE ||
  "视频文件需线下处理";

export const VIDEO_UPLOAD_DISABLED_MESSAGE =
  import.meta.env.VITE_VIDEO_UPLOAD_DISABLED_MESSAGE ||
  "当前服务器暂不支持在网页上传与转码视频文件。你仍可填写标题、简介、分区、标签和封面并保存到服务器；视频 MP4 将由管理员在本地处理后关联。";

export const STORAGE_METADATA_ONLY = "creator_center_metadata_only";

/** @returns {boolean} true = 已拦截（应中止视频文件上传） */
export function guardVideoFileUploadDisabled(notify) {
  if (!isVideoUploadDisabled()) return false;
  const text = `${VIDEO_UPLOAD_DISABLED_TITLE}：${VIDEO_UPLOAD_DISABLED_MESSAGE}`;
  if (typeof notify === "function") {
    notify(text);
  }
  return true;
}

/** @deprecated 使用 guardVideoFileUploadDisabled */
export function guardVideoUploadDisabled(notify) {
  return guardVideoFileUploadDisabled(notify);
}
