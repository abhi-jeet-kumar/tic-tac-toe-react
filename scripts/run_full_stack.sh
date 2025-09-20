#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
SERVER_DIR="$ROOT_DIR/server"
INFRA_DIR="$ROOT_DIR/infra"
APP_DIR="$ROOT_DIR/app"
EXPO_LOG="/tmp/expo-web.log"
EXPO_PID="/tmp/expo-web.pid"

compose_cmd() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    echo "docker compose"
  else
    echo "docker-compose"
  fi
}

die() { echo "[ERR] $*" >&2; exit 1; }

require() { command -v "$1" >/dev/null 2>&1 || die "Missing dependency: $1"; }

detect_goarch() {
  local arch
  arch=$(docker info --format '{{.Architecture}}' 2>/dev/null || echo amd64)
  case "$arch" in
    aarch64|arm64) echo arm64 ;;
    x86_64|amd64) echo amd64 ;;
    *) echo amd64 ;;
  esac
}

docker_platform() {
  local ga; ga=$(detect_goarch)
  if [ "$ga" = "arm64" ]; then echo linux/arm64; else echo linux/amd64; fi
}

build_plugin() {
  echo "[+] Building Nakama Go plugin (Linux $(detect_goarch))"
  require docker
  local goarch; goarch=$(detect_goarch)
  local dplat; dplat=$(docker_platform)
  mkdir -p "$SERVER_DIR/bin"
  # Use Debian-based golang image, force platform and PATH so 'go' is resolvable
  docker run --rm --platform "$dplat" -v "$SERVER_DIR":/work -w /work golang:1.22 \
    bash -lc "export PATH=/usr/local/go/bin:\$PATH; /usr/local/go/bin/go version; /usr/local/go/bin/go env; /usr/local/go/bin/go mod tidy; GOOS=linux GOARCH=$goarch /usr/local/go/bin/go build -buildmode=plugin -o bin/tictactoe.so ."
  ls -la "$SERVER_DIR/bin"
}

compose_up() {
  echo "[+] Starting Postgres + Nakama"
  local dc; dc=$(compose_cmd)
  (cd "$ROOT_DIR" && $dc -f "$INFRA_DIR/docker-compose.yml" up -d)
  echo "[+] Waiting for Nakama console..."
  for i in {1..60}; do
    if curl -sf http://localhost:7351/ >/dev/null 2>&1; then echo "[ok] Nakama console up"; break; fi; sleep 2; done
}

apply_migrations() {
  echo "[+] Applying SQL migrations to Postgres"
  require docker
  for f in $(ls -1 "$SERVER_DIR"/migrations/*.sql | sort); do
    echo "  - $f"
    docker exec -i ttt_postgres psql -U nakama -d nakama -v ON_ERROR_STOP=1 -f - < "$f"
  done
}

start_expo_web_bg() {
  echo "[+] Installing app deps and starting Expo Web (background)"
  require npm
  (cd "$APP_DIR" && npm install --silent)
  (cd "$APP_DIR" && npx --yes expo start --web --non-interactive --clear --port 19006 &> "$EXPO_LOG" & echo $! > "$EXPO_PID")
  sleep 5
  if lsof -i :19006 -sTCP:LISTEN >/dev/null 2>&1; then
    echo "[ok] Expo Web at http://localhost:19006"
  else
    echo "[warn] Expo not listening yet; tail $EXPO_LOG"
  fi
}

stop_all() {
  echo "[+] Stopping services"
  local dc; dc=$(compose_cmd)
  (cd "$ROOT_DIR" && $dc -f "$INFRA_DIR/docker-compose.yml" down -v || true)
  if [ -f "$EXPO_PID" ] && kill -0 "$(cat "$EXPO_PID")" 2>/dev/null; then
    kill "$(cat "$EXPO_PID")" || true
    rm -f "$EXPO_PID"
  fi
}

status() {
  echo "[status] Containers:" && docker ps --filter name=ttt_ | cat
  echo "[status] Expo:" && (lsof -i :19006 -sTCP:LISTEN >/dev/null 2>&1 && echo "listening on 19006" || echo "not running")
}

usage() {
  cat <<USAGE
Usage: $(basename "$0") [up|down|status|rebuild]

  up       Build plugin, start Docker services, apply migrations, start Expo Web
  down     Stop Docker services and Expo Web
  status   Show container and Expo status
  rebuild  Rebuild the Go plugin only
USAGE
}

cmd=${1:-up}
case "$cmd" in
  up)
    build_plugin
    compose_up
    apply_migrations
    start_expo_web_bg
    echo "Nakama console: http://localhost:7351"
    echo "App (web):     http://localhost:19006"
    ;;
  down)
    stop_all
    ;;
  status)
    status
    ;;
  rebuild)
    build_plugin
    ;;
  *)
    usage; exit 1
    ;;
esac


