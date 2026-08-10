#!/usr/bin/env python3
"""Guard docker-compose.yml against environment drift.

Ensures the repo-wide rule "new env var -> update .env.example" also covers the
Compose stack:
  1. every *active* key in .env.example is forwarded by the backend service;
  2. every ${VAR} referenced in docker-compose.yml is declared in .env.example
     (active or commented, so optional vars are documented too).

Usage:
  python scripts/check_compose_env.py

Requires `docker compose config` (runs on CI; validates the compose file as a
side effect). Exits non-zero with a diff when drift is found.
"""

import json
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
ENV_EXAMPLE = ROOT / ".env.example"
COMPOSE = ROOT / "docker-compose.yml"

# Compose-infra-only vars: consumed by compose services (ports/credentials of
# MySQL/Redis/RabbitMQ/ES) but intentionally NOT forwarded to the backend
# container. Adding a new var here requires updating this allowlist as well.
COMPOSE_ONLY_VARS = {
    "MYSQL_ROOT_PASSWORD",
    "MYSQL_DATABASE",
    "MYSQL_PORT",
    "REDIS_PORT",
    "RABBITMQ_PORT",
    "RABBITMQ_MGMT_PORT",
    "ES_PORT",
    "BACKEND_PORT",
    "WEB_PORT",
    "PROMETHEUS_PORT",
    "ALERTMANAGER_PORT",
    "GRAFANA_PORT",
    "GRAFANA_ADMIN_USER",
    "GRAFANA_ADMIN_PASSWORD",
    "PROMETHEUS_RETENTION",
    "PROMETHEUS_CLUSTER",
    "ALERTMANAGER_WEBHOOK_URL",
    # Consumed by the web image build (VITE_* is a Vite build-time flag).
    "VITE_VIDEO_UPLOAD_DISABLED",
}


def env_example_keys():
    """Return (active_keys, declared_keys) parsed from .env.example."""
    active = set()
    declared = set()
    for line in ENV_EXAMPLE.read_text(encoding="utf-8").splitlines():
        m = re.match(r"^#?\s*([A-Z][A-Z0-9_]*)\s*=", line)
        if not m:
            continue
        key = m.group(1)
        declared.add(key)
        if not line.lstrip().startswith("#"):
            active.add(key)
    return active, declared


def compose_refs():
    """Return all ${VAR} interpolation references in docker-compose.yml."""
    refs = set()
    for line in COMPOSE.read_text(encoding="utf-8").splitlines():
        if line.lstrip().startswith("#"):
            continue
        line = line.replace("$${", "")
        refs.update(re.findall(r"\$\{([A-Z][A-Z0-9_]*)(?:[:-][^}]*)?\}", line))
    return refs


def backend_env_keys():
    """Resolve the backend service environment keys via `docker compose config`."""
    try:
        proc = subprocess.run(
            ["docker", "compose", "config", "--format", "json"],
            cwd=ROOT,
            capture_output=True,
            text=True,
            check=False,
        )
    except FileNotFoundError:
        proc = None
    if proc is not None and proc.returncode == 0:
        cfg = json.loads(proc.stdout)
        env = cfg["services"]["backend"].get("environment") or {}
        return set(env.keys())
    if proc is not None:
        print(
            f"warning: docker compose config failed ({proc.returncode}), "
            "falling back to direct YAML parse",
            file=sys.stderr,
        )
    try:
        import yaml
    except ImportError as exc:
        sys.exit(
            "docker compose unavailable and pyyaml not installed: "
            f"{exc}"
        )
    cfg = yaml.safe_load(COMPOSE.read_text(encoding="utf-8"))
    env = cfg["services"]["backend"].get("environment") or {}
    return set(env.keys())


def main():
    active, declared = env_example_keys()
    refs = compose_refs()
    env_keys = backend_env_keys()

    problems = []
    missing = sorted((active - COMPOSE_ONLY_VARS) - env_keys)
    if missing:
        problems.append(
            "active .env.example vars missing from compose backend environment: "
            + ", ".join(missing)
        )
    undeclared = sorted(refs - declared)
    if undeclared:
        problems.append(
            "compose ${VAR} refs not declared in .env.example: " + ", ".join(undeclared)
        )

    if problems:
        print("\n".join(problems), file=sys.stderr)
        print(
            "Hint: update docker-compose.yml and/or .env.example together (repo rule: "
            "every env var must live in .env.example).",
            file=sys.stderr,
        )
        sys.exit(1)
    print("compose env drift check OK")


if __name__ == "__main__":
    main()
