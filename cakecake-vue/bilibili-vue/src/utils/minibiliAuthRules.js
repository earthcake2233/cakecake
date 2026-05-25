/** 与后端 username.Valid 一致：3–32 个字符，支持中文、字母、数字、下划线 */

export const MINIBILI_USERNAME_RULE_HINT =
  "3–32 个字符，支持中文、英文、数字、下划线；不可含空格或特殊符号";

export const MINIBILI_USERNAME_PLACEHOLDER = "例如：小蛋糕、cake_user";

export const MINIBILI_REGISTER_PASSWORD_HINT = "至少 8 位";

function isUsernameChar(ch) {
  if (ch === "_") return true;
  if (/[0-9]/.test(ch)) return true;
  return /\p{L}/u.test(ch);
}

export function validateMinibiliUsername(username) {
  const u = String(username || "").trim();
  const chars = [...u];
  if (chars.length < 3 || chars.length > 32) {
    return "用户名为 3–32 个字符，支持中文、字母、数字、下划线";
  }
  for (const ch of chars) {
    if (!isUsernameChar(ch)) {
      return "用户名仅支持中文、字母、数字、下划线";
    }
  }
  return "";
}

export function validateMinibiliRegisterPassword(password) {
  const p = password || "";
  if (p.length < 8) {
    return "密码至少 8 位";
  }
  return "";
}

/** 从 axios 错误中取出后端 msg（含业务 4xx JSON） */
export function minibiliErrorMessage(err, fallback = "请求失败") {
  const d = err && err.response && err.response.data;
  if (d && typeof d.msg === "string" && d.msg.trim()) {
    return d.msg.trim();
  }
  if (err && err.message) {
    return String(err.message);
  }
  return fallback;
}

/** 登录失败：不把「未登录/Token」暴露给用户，统一为账号密码错误 */
export function mapMinibiliLoginFailureMessage(err) {
  const d = err && err.response && err.response.data;
  const code = d && typeof d.code === "number" ? d.code : null;
  if (code === 40100 || code === 40101) {
    return "用户名或密码错误";
  }
  const raw = minibiliErrorMessage(err, "");
  if (
    raw.includes("未登录") ||
    raw.includes("Token") ||
    raw.includes("token") ||
    raw.includes("过期")
  ) {
    return "用户名或密码错误";
  }
  return raw || "用户名或密码错误";
}
