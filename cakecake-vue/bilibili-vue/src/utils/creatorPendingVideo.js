/** 创作中心：从投稿页带到发布页的本地视频 File（sessionStorage 无法存 File） */
let pendingVideoFile = null;

export function stashPendingVideoFile(file) {
  pendingVideoFile = file || null;
}

export function takePendingVideoFile() {
  const f = pendingVideoFile;
  pendingVideoFile = null;
  return f;
}

export function clearPendingVideoFile() {
  pendingVideoFile = null;
}
