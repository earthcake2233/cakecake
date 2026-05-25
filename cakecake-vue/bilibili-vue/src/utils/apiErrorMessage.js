const GENERIC_AXIOS_STATUS = /^Request failed with status code \d+$/;

/**
 * Prefer backend `{ msg }` over axios generic HTTP status messages.
 */
export function extractApiErrorMessage(error, fallback = "加载失败") {
  if (!error) return fallback;

  const data = error.response && error.response.data;
  if (data != null) {
    if (typeof data === "object") {
      const m = data.msg != null ? data.msg : data.message;
      if (typeof m === "string" && m.trim()) return m.trim();
    }
    if (typeof data === "string") {
      try {
        const parsed = JSON.parse(data);
        const m =
          parsed && (parsed.msg != null ? parsed.msg : parsed.message);
        if (typeof m === "string" && m.trim()) return m.trim();
      } catch (_) {
        if (data.trim()) return data.trim();
      }
    }
  }

  const message =
    typeof error.message === "string" ? error.message.trim() : "";
  if (message && !GENERIC_AXIOS_STATUS.test(message)) {
    return message;
  }

  return fallback;
}
