#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${HARBORX_REPO_URL:-https://github.com/youko-nobody/HarborX.git}"
INSTALL_DIR="${HARBORX_INSTALL_DIR:-/opt/harborx}"
PORT="${HARBORX_PORT:-18080}"
HOST="${HARBORX_HOST:-0.0.0.0}"
ADMIN_PASSWORD="${HARBORX_ADMIN_PASSWORD:-}"

need_root() {
  if [ "$(id -u)" -ne 0 ]; then
    echo "Please run as root: sudo bash scripts/deploy-vps.sh"
    exit 1
  fi
}

install_packages() {
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update
    apt-get install -y ca-certificates curl git openssl
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y ca-certificates curl git openssl
  elif command -v yum >/dev/null 2>&1; then
    yum install -y ca-certificates curl git openssl
  else
    echo "Unsupported package manager. Install git, curl, openssl, and Docker manually."
    exit 1
  fi
}

install_docker() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    return
  fi
  curl -fsSL https://get.docker.com | sh
  systemctl enable --now docker
}

sync_repo() {
  if [ -d "$INSTALL_DIR/.git" ]; then
    git -C "$INSTALL_DIR" pull --ff-only
  else
    mkdir -p "$(dirname "$INSTALL_DIR")"
    git clone "$REPO_URL" "$INSTALL_DIR"
  fi
}

write_env() {
  if [ -z "$ADMIN_PASSWORD" ]; then
    ADMIN_PASSWORD="harborx-$(openssl rand -base64 18 | tr -d '=+/')"
  fi

  cat > "$INSTALL_DIR/.env" <<EOF
HARBORX_HOST=$HOST
HARBORX_PORT=$PORT
HARBORX_DATA_DIR=/app/data
HARBORX_DB_PATH=/app/data/harborx.sqlite
HARBORX_WEB_DIST_DIR=/app/web-dist
HARBORX_ADMIN_PASSWORD=$ADMIN_PASSWORD
EOF
  chmod 600 "$INSTALL_DIR/.env"
}

start_stack() {
  cd "$INSTALL_DIR"
  docker compose up -d --build
}

need_root
install_packages
install_docker
sync_repo
write_env
start_stack

echo "HarborX is running on http://SERVER_IP:$PORT"
echo "Admin username: admin"
echo "Admin password: $ADMIN_PASSWORD"
echo "Install directory: $INSTALL_DIR"
