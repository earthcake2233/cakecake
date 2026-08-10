#!/usr/bin/env bash
# Merge a template env file into an existing env file without touching values.
#
# Rules:
#   - keys already present in the target (including empty values) are NEVER
#     overwritten;
#   - keys present only in the template are appended with the template value;
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

cp -a "$TARGET" "$tmp"

added=()
while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    '' | '#'*) continue ;;
    *=*)
      key=${line%%=*}
      case "$key" in
        '' | *[!A-Za-z0-9_]*) continue ;;
      esac
      if ! grep -qE "^[#]?${key}=" "$TARGET"; then
        printf '%s\n' "$line" >> "$tmp"
        added+=("$key")
      fi
      ;;
  esac
done < "$TEMPLATE"

if [ ${#added[@]} -eq 0 ]; then
  echo "env merge: no changes (${TARGET} already up to date)"
  exit 0
fi

cp -a "$TARGET" "${TARGET}.prev"
mv "$tmp" "$TARGET"
chmod 600 "$TARGET"

echo "env merge: added ${#added[@]} key(s): ${added[*]}"
echo "env merge: backup written to ${TARGET}.prev"
