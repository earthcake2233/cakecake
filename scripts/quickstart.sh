#!/usr/bin/env bash
# cakecake one-command quickstart.
#
# Downloads the official compose file and .env defaults from the cakecake
# repository, generates random secrets, then starts the full stack with the
# published Docker Hub images
# (MySQL/Redis/RabbitMQ/ES + backend + frontend). No source clone or local
# build required. Idempotent: existing docker-compose.yml / .env are kept.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/earthcake2233/cakecake/main/scripts/quickstart.sh | bash

set -euo pipefail

REPO_RAW="https://raw.githubusercontent.com/earthcake2233/cakecake/main"

say() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
die() { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }
gen_hex() { head -c "${1:-16}" /dev/urandom | od -An -tx1 | tr -d ' \n'; }
fill_secrets() {
  sed -i "s/^JWT_SECRET=$/JWT_SECRET=$(gen_hex 32)/" .env
  sed -i "s/^MYSQL_ROOT_PASSWORD=$/MYSQL_ROOT_PASSWORD=$(gen_hex 16)/" .env
  sed -i "s/^ADMIN_SEED_PASSWORD=$/ADMIN_SEED_PASSWORD=$(gen_hex 16)/" .env
  sed -i "s/^METRICS_TOKEN=$/METRICS_TOKEN=$(gen_hex 32)/" .env
  sed -i "s/^GRAFANA_ADMIN_PASSWORD=$/GRAFANA_ADMIN_PASSWORD=$(gen_hex 16)/" .env
  mkdir -p .secrets
  token=$(sed -n 's/^METRICS_TOKEN=//p' .env | head -1)
  [ -n "$token" ] || return 1
  printf '%s\n' "$token" > .secrets/metrics_token
  printf '%s\n' "$token" > .secrets/metrics_token_cloud
  chmod 700 .secrets
  chmod 644 .secrets/metrics_token .secrets/metrics_token_cloud
  chmod 600 .env
}

command -v docker >/dev/null 2>&1 || die "docker not found. Install Docker first: https://docs.docker.com/get-docker/"
docker compose version >/dev/null 2>&1 || die "docker compose plugin not found (Docker Desktop 4.x+ / Compose v2 required)"

if [ ! -f docker-compose.yml ]; then
  say "Downloading docker-compose.yml"
  curl -fsSL "$REPO_RAW/docker-compose.yml" -o docker-compose.yml
else
  say "docker-compose.yml already exists, keeping it"
fi

if [ ! -f .env ]; then
  say "Creating .env from .env.example"
  curl -fsSL "$REPO_RAW/.env.example" -o .env.example
  cp .env.example .env
  fill_secrets
else
  say ".env already exists, keeping it"
  fill_secrets
fi

say "Pulling images (first run may take a while)"
docker compose pull

say "Starting cakecake"
docker compose up -d

cat <<'EOF'

cakecake is starting:
  Web       http://localhost:8888
  API       http://localhost:8080   (health: /api/v1/health)
  Swagger   http://localhost:8080/swagger/

Demo users (password: demo123456):
  暗猫の祝福 / 加载超时请稍后 / Baka恶魔 / 三栗lili / Yeuoly / 科学超电磁炮F / 泛式大大

Admin console: http://localhost:8888/#/admin/login  (admin, password from .env ADMIN_SEED_PASSWORD)

Stop:   docker compose down
Reset:  docker compose down -v
Update: re-run this script
EOF
