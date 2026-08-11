#!/usr/bin/env bash
# Merge a template env file into an existing env file without touching values.
#
# Rules:
#   - keys already present in the target (including empty values) are NEVER
#     overwritten;
#   - keys present only in the template are appended with the template value;
#   - duplicate active keys in the target are deduplicated (first occurrence
#     wins, so values are never rewritten);
#   - the previous target is kept as <target>.prev;
#   - target permissions stay 0600.
#
# Usage: merge_env.sh <target.env> <template.env>
set -euo pipefail

if [ $# -ne 2 ]; then
  echo "usage: $0 <target.env> <template.env>" >&2
  exit 2
fi

TARGET=$1
TEMPLATE=$2

[ -f "$TARGET" ] || { echo "target env not found: $TARGET" >&2; exit 1; }
[ -f "$TEMPLATE" ] || { echo "template env not found: $TEMPLATE" >&2; exit 1; }

tmp=$(mktemp "${TARGET}.merge.XXXXXX")
trap 'rm -f "$tmp"' EXIT
chmod 600 "$tmp"

seen=()
removed=0

append_key() {
  local key=$1 line=$2
  for k in "${seen[@]:-}"; do
    if [ "$k" = "$key" ]; then
      return 1
    fi
  done
  seen+=("$key")
  printf '%s\n' "$line" >> "$tmp"
  return 0
}

# Copy the target verbatim except duplicate active keys (first one wins).
while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    '' | '#'*)
      printf '%s\n' "$line" >> "$tmp"
      continue
      ;;
    *=*)
      key=${line%%=*}
      case "$key" in
        '' | *[!A-Za-z0-9_]*)
          printf '%s\n' "$line" >> "$tmp"
          continue
          ;;
      esac
      if append_key "$key" "$line"; then
        :
      else
        removed=$((removed + 1))
      fi
      ;;
    *)
      printf '%s\n' "$line" >> "$tmp"
      ;;
  esac
done < "$TARGET"

added_count=0
added_names=""
while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    '' | '#'*) continue ;;
    *=*)
      key=${line%%=*}
      case "$key" in
        '' | *[!A-Za-z0-9_]*) continue ;;
      esac
      if append_key "$key" "$line"; then
        added_count=$((added_count + 1))
        added_names="${added_names:+$added_names }$key"
      fi
      ;;
  esac
done < "$TEMPLATE"

if [ "$added_count" -eq 0 ] && [ "$removed" -eq 0 ]; then
  echo "env merge: no changes (${TARGET} already up to date)"
  exit 0
fi

cp -a "$TARGET" "${TARGET}.prev"
mv "$tmp" "$TARGET"
chmod 600 "$TARGET"

echo "env merge: added ${added_count} key(s): ${added_names}"
[ "$removed" -gt 0 ] && echo "env merge: removed ${removed} duplicate line(s) (first value kept)"
echo "env merge: backup written to ${TARGET}.prev"
