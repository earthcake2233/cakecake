/** 去掉 URL 上已有的 v= 缓存参数 */
export function stripImageCacheBust(url) {
  const u = String(url || "").trim();
  if (!u) {
    return u;
  }
  const q = u.indexOf("?");
  if (q < 0) {
    return u;
  }
  const base = u.slice(0, q);
  const params = u
    .slice(q + 1)
    .split("&")
    .filter(p => p && !p.startsWith("v="));
  return params.length ? `${base}?${params.join("&")}` : base;
}

/** 为 OSS 等固定路径图片追加缓存破坏参数，避免换图后仍显示旧图 */
export function withImageCacheBust(url, bust) {
  const base = stripImageCacheBust(url);
  if (!base) {
    return "";
  }
  if (bust == null || bust === "" || bust === 0) {
    return base;
  }
  const sep = base.includes("?") ? "&" : "?";
  return `${base}${sep}v=${encodeURIComponent(String(bust))}`;
}

/** 展示用户头像：优先 API 自带 v=，必要时再叠加前端 bust */
export function resolveUserAvatarUrl(rawUrl, bust = 0) {
  const u = String(rawUrl || "").trim();
  if (!u) {
    return "";
  }
  if (/[?&]v=\d+/.test(u)) {
    return bust ? withImageCacheBust(u, bust) : u;
  }
  return withImageCacheBust(u, bust);
}
