#!/usr/bin/env python3
"""Bootstrap or refresh the local .env with strong random secrets.

Usage:
  python3 scripts/init_env.py            # create .env from .env.example (refuses to overwrite)
  python3 scripts/init_env.py --refresh  # merge .env.example into an existing .env:
                                         # preserve real values, fill empty/missing secrets,
                                         # keep extra local keys (e.g. GOCACHE)

Security contract:
  - never prints or logs secret values;
  - writes .env with mode 0600;
  - .env is gitignored; never commit it.
"""

import argparse
import os
import re
import secrets
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EXAMPLE = ROOT / ".env.example"
TARGET = ROOT / ".env"

SECRET_GENERATORS = {
    "JWT_SECRET": lambda: secrets.token_hex(32),
    "MYSQL_ROOT_PASSWORD": lambda: secrets.token_hex(16),
    "ADMIN_SEED_PASSWORD": lambda: secrets.token_hex(16),
    "METRICS_TOKEN": lambda: secrets.token_hex(32),
    "GRAFANA_ADMIN_PASSWORD": lambda: secrets.token_hex(16),
}

METRICS_SECRET_FILE = ROOT / ".secrets" / "metrics_token"
METRICS_SECRET_CLOUD_FILE = ROOT / ".secrets" / "metrics_token_cloud"

PLACEHOLDER_MARKS = (
    "change-me",
    "changeme",
    "cakecake_dev",
    "user:password@",
    "replace-me",
    "请替换",
    "<generate",
)

KEY_RE = re.compile(r"^([A-Z][A-Z0-9_]*)\s*=(.*)$")
COMMENT_KEY_RE = re.compile(r"^#\s*([A-Z][A-Z0-9_]*)\s*=")


def parse_env(text):
    entries = []
    values = {}
    commented_keys = set()
    for line in text.splitlines():
        m = KEY_RE.match(line)
        if m:
            key, raw = m.group(1), m.group(2).strip()
            entries.append((key, line))
            values[key] = raw
            continue
        cm = COMMENT_KEY_RE.match(line)
        if cm:
            commented_keys.add(cm.group(1))
            entries.append((None, line))
            continue
        entries.append((None, line))
    return entries, values, commented_keys


def is_placeholder(value):
    low = value.lower()
    return any(mark in low for mark in PLACEHOLDER_MARKS)


def extract_root_password(dsn):
    """Return the root password embedded in a local-dev DSN, if any."""
    if not dsn.startswith("root:"):
        return None
    rest = dsn[len("root:"):]
    host = rest.find("@tcp(")
    if host <= 0:
        return None
    pw = rest[:host]
    if not pw or is_placeholder(pw):
        return None
    return pw


def local_dsn(root_password, database):
    return (
        f"root:{root_password}@tcp(127.0.0.1:3306)/{database}"
        "?charset=utf8mb4&parseTime=True&loc=Local"
    )


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--refresh", action="store_true", help="merge into an existing .env")
    args = parser.parse_args()

    if not EXAMPLE.exists():
        sys.exit(f"missing {EXAMPLE.relative_to(ROOT)}")
    example_entries, example_values, commented_keys = parse_env(EXAMPLE.read_text(encoding="utf-8"))

    existing = {}
    if TARGET.exists():
        _, existing, _ = parse_env(TARGET.read_text(encoding="utf-8"))

    if not args.refresh and TARGET.exists():
        sys.exit(f"{TARGET.relative_to(ROOT)} already exists; run with --refresh to update it")

    root_pw = existing.get("MYSQL_ROOT_PASSWORD")
    if (not root_pw or is_placeholder(root_pw)) and existing.get("MYSQL_DSN"):
        root_pw = extract_root_password(existing["MYSQL_DSN"])
    if not root_pw or is_placeholder(root_pw):
        root_pw = SECRET_GENERATORS["MYSQL_ROOT_PASSWORD"]()

    known_keys = set(example_values) | commented_keys
    out = []
    summary = {}
    written = {}

    for key, line in example_entries:
        if key is None:
            out.append(line)
            continue
        example = example_values[key]
        current = existing.get(key)

        if key == "MYSQL_ROOT_PASSWORD" and (not current or is_placeholder(current)) and root_pw:
            value = root_pw
            summary[key] = "derived from MYSQL_DSN"
        elif current and not is_placeholder(current):
            value = current
            summary[key] = "preserved"
        elif key in SECRET_GENERATORS:
            value = SECRET_GENERATORS[key]()
            summary[key] = "generated"
        elif key == "MYSQL_DSN" and root_pw:
            value = local_dsn(root_pw, example_values.get("MYSQL_DATABASE", "cakecake"))
            summary[key] = "generated from MYSQL_ROOT_PASSWORD"
        else:
            value = example
            summary.setdefault(key, "empty/default")

        out.append(f"{key}={value}")
        written[key] = value

    # Keep extra local keys (Go toolchain vars etc.) in original order, without leaking values.
    extra_keys = [k for k in existing if k not in known_keys]
    if extra_keys:
        out.append("")
        out.append("# ── 其它（原有 .env 独有项，原样保留）──")
        for k in extra_keys:
            out.append(f"{k}={existing[k]}")

    TARGET.write_text("\n".join(out) + "\n", encoding="utf-8")
    os.chmod(TARGET, 0o600)

    # Prometheus uses credentials_file (Docker secret) instead of env expansion,
    # which is unreliable for authorization fields. Mode 0644 matches Docker
    # secret semantics so the non-root container user can read it; the .env
    # itself stays 0600.
    metrics_token = written["METRICS_TOKEN"]
    METRICS_SECRET_FILE.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    METRICS_SECRET_FILE.write_text(metrics_token.rstrip("\n") + "\n", encoding="utf-8")
    os.chmod(METRICS_SECRET_FILE, 0o644)

    # 云服务器可用独立 token（METRICS_TOKEN_CLOUD）；未设置时与本地一致。
    cloud_token = existing.get("METRICS_TOKEN_CLOUD")
    if not cloud_token or is_placeholder(cloud_token):
        cloud_token = metrics_token
    METRICS_SECRET_CLOUD_FILE.write_text(cloud_token.rstrip("\n") + "\n", encoding="utf-8")
    os.chmod(METRICS_SECRET_CLOUD_FILE, 0o644)

    print(f"wrote {TARGET.relative_to(ROOT)} (mode 0600)")
    print(f"wrote {METRICS_SECRET_FILE.relative_to(ROOT)} (mode 0644, Docker secret semantics)")
    print(f"wrote {METRICS_SECRET_CLOUD_FILE.relative_to(ROOT)} (mode 0644, Docker secret semantics)")
    print("secret status (values intentionally not printed):")
    for key in sorted(set(summary) | set(SECRET_GENERATORS)):
        print(f"  {key}: {summary.get(key, 'empty/default')}")


if __name__ == "__main__":
    main()
