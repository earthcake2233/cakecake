export const SITE_BRAND = "cakecake";
export const SITE_TITLE_TAGLINE = "(゜-゜)つロ 干杯~";
/** 与 B 站 tab 格式一致：品牌 + 口号 + 域名后缀 */
export const SITE_TITLE_SUFFIX = `${SITE_BRAND} ${SITE_TITLE_TAGLINE}-${SITE_BRAND}`;
export const SITE_HOME_TITLE = SITE_TITLE_SUFFIX;

/** 浏览器标签标题：{页面名} - cakecake (゜-゜)つロ 干杯~-cakecake */
export function buildDocumentTitle(pageLabel) {
  const label = String(pageLabel ?? "").trim();
  if (!label) {
    return SITE_HOME_TITLE;
  }
  return `${label} - ${SITE_HOME_TITLE}`;
}
