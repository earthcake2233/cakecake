#!/usr/bin/env bash
# Optimize README screenshots to WebP (target <= ~300 KB each).
#
# - backs up the original PNGs under docs/images/bench/originals/
# - converts docs/images/*.png to docs/images/*.webp (libwebp, quality 78,
#   retries at quality 65 if still above the size cap)
# - removes the original PNGs after a successful conversion
#
# Usage: ./scripts/optimize-screenshots.sh [SRC_DIR]

set -euo pipefail

SRC_DIR="${1:-docs/images}"
BACKUP_DIR="$SRC_DIR/bench/originals"
Q1=78
Q2=65
MAX_KB=350

command -v ffmpeg >/dev/null 2>&1 || {
  echo "error: ffmpeg is required (libwebp encoder)" >&2
  exit 1
}

mkdir -p "$BACKUP_DIR"

for png in "$SRC_DIR"/*.png; do
  [ -e "$png" ] || continue
  base=$(basename "$png" .png)
  webp="$SRC_DIR/$base.webp"

  if [ ! -f "$BACKUP_DIR/$base.png" ]; then
    cp "$png" "$BACKUP_DIR/$base.png"
  fi

  ffmpeg -hide_banner -y -i "$png" -c:v libwebp -quality "$Q1" -compression_level 6 "$webp" >/dev/null 2>&1
  size_kb=$(( $(stat -c%s "$webp") / 1024 ))
  if [ "$size_kb" -gt "$MAX_KB" ]; then
    ffmpeg -hide_banner -y -i "$png" -c:v libwebp -quality "$Q2" -compression_level 6 "$webp" >/dev/null 2>&1
    size_kb=$(( $(stat -c%s "$webp") / 1024 ))
  fi

  echo "$base.png -> $base.webp (${size_kb} KB)"
  rm "$png"
done

echo "Done. Originals backed up in $BACKUP_DIR"
