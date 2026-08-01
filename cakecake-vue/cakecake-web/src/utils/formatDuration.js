/** 视频时长：分:秒；超过 1 小时为 时:分:秒 */
export function formatDuration(sec) {
  const s = Math.max(0, Math.floor(Number(sec) || 0));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const ss = s % 60;
  const pad = n => String(n).padStart(2, "0");
  if (h > 0) {
    return `${h}:${pad(m)}:${pad(ss)}`;
  }
  return `${m}:${pad(ss)}`;
}
