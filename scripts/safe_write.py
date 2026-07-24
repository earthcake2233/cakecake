#!/usr/bin/env python3
"""Safe UTF-8 file writer for PowerShell environments.

Usage:
  python scripts/safe_write.py --base64 <base64> --output <filepath>
  python scripts/safe_write.py --text <content> --output <filepath>
  python scripts/safe_write.py --file <source> --output <filepath>
"""

import argparse, base64, hashlib, pathlib, sys

def main():
    parser = argparse.ArgumentParser(description="Safe UTF-8 file writer")
    parser.add_argument("--base64", help="Base64-encoded content to write")
    parser.add_argument("--text", help="Plain text content to write")
    parser.add_argument("--file", help="Read content from source file")
    parser.add_argument("--output", required=True, help="Output file path")
    parser.add_argument("--encoding", default="utf-8", help="Output encoding")
    args = parser.parse_args()

    if args.base64:
        content = base64.b64decode(args.base64).decode("utf-8")
    elif args.text:
        content = args.text
    elif args.file:
        content = pathlib.Path(args.file).read_text(args.encoding)
    else:
        content = sys.stdin.read()

    out_path = pathlib.Path(args.output)
    # Strip UTF-8 BOM if present (Go compiler rejects BOM)
    if content.startswith("\ufeff"):
        content = content.lstrip("\ufeff")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(content, args.encoding)
    h = hashlib.sha256(content.encode(args.encoding)).hexdigest()[:16]
    print(f"Written {len(content)} bytes to {out_path} [{h}]")

if __name__ == "__main__":
    main()