/** Format 硬币 for display (supports one decimal from UP 10% share). */
export function formatCoinBalance(value) {
  const n = Number(value);
  if (!Number.isFinite(n) || n < 0) {
    return "0";
  }
  const rounded = Math.round(n * 10) / 10;
  if (Math.abs(rounded - Math.round(rounded)) < 1e-6) {
    return String(Math.round(rounded));
  }
  return rounded.toFixed(1);
}

export function coinBalanceNumber(value) {
  const n = Number(value);
  if (!Number.isFinite(n) || n < 0) {
    return 0;
  }
  return Math.round(n * 10) / 10;
}
