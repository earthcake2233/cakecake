/** 消息中心分类（顺序：我的消息优先） */
export const MESSAGE_CATEGORIES = [
  { cat: "my_message", label: "我的消息" },
  { cat: "reply_received", label: "回复我的" },
  { cat: "at_me", label: "@ 我的" },
  { cat: "like_aggregation", label: "收到的赞" },
  { cat: "system_notice", label: "系统通知" }
];

export const MESSAGE_CAT_LABELS = Object.fromEntries(
  MESSAGE_CATEGORIES.map(({ cat, label }) => [cat, label])
);

export function formatMessageUnreadBadge(n) {
  const v = Number(n) || 0;
  if (v <= 0) return "";
  return v > 99 ? "99+" : String(v);
}

/** 创作中心顶栏消息下拉顺序（与 B 站创作中心一致） */
export const CREATOR_MESSAGE_DROPDOWN = [
  { cat: "reply_received", label: "回复我的" },
  { cat: "at_me", label: "@我的" },
  { cat: "like_aggregation", label: "收到的赞" },
  { cat: "system_notice", label: "系统通知" },
  { cat: "my_message", label: "我的消息" }
];

export function sumMessageUnread(summary) {
  if (!summary || typeof summary !== "object") return 0;
  return MESSAGE_CATEGORIES.reduce(
    (acc, { cat }) => acc + (Number(summary[cat]) || 0),
    0
  );
}
