/**
 * Pure chat-scroll helpers extracted from MbDmChatPanel.
 */

export function animateScrollTo(el, to) {
  if (!el) return;
  const from = el.scrollTop;
  const delta = to - from;
  if (Math.abs(delta) < 1) return;
  const duration = Math.min(500, Math.max(250, Math.abs(delta) * 0.35));
  const start = performance.now();
  const ease = t => (t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2);
  const step = now => {
    const t = Math.min(1, (now - start) / duration);
    el.scrollTop = from + delta * ease(t);
    if (t < 1) requestAnimationFrame(step);
  };
  requestAnimationFrame(step);
}

export function isNearBottom(el, threshold = 80) {
  return !!el && el.scrollHeight - el.scrollTop - el.clientHeight < threshold;
}
