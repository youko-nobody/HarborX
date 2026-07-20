#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${HARBORX_AGENT_BASE_URL:-}"
TOKEN="${HARBORX_AGENT_TOKEN:-}"
INSTALL_DIR="${HARBORX_AGENT_INSTALL_DIR:-/opt/harborx-agent}"
SERVICE_NAME="harborx-agent"
INTERVAL_SECONDS="${HARBORX_AGENT_INTERVAL_SECONDS:-10}"

if [ "$(id -u)" -ne 0 ]; then
  echo "Please run as root: sudo bash scripts/install-agent.sh"
  exit 1
fi

if [ -z "$BASE_URL" ] || [ -z "$TOKEN" ]; then
  echo "HARBORX_AGENT_BASE_URL and HARBORX_AGENT_TOKEN are required."
  exit 1
fi

if command -v apt-get >/dev/null 2>&1; then
  apt-get update
  apt-get install -y ca-certificates curl git golang
elif command -v dnf >/dev/null 2>&1; then
  dnf install -y ca-certificates curl git golang
elif command -v yum >/dev/null 2>&1; then
  yum install -y ca-certificates curl git golang
else
  echo "Unsupported package manager. Install Go, git, and curl manually."
  exit 1
fi

mkdir -p "$INSTALL_DIR"
if [ -d "$INSTALL_DIR/.git" ]; then
  git -C "$INSTALL_DIR" pull --ff-only
else
  git clone "${HARBORX_REPO_URL:-https://github.com/youko-nobody/HarborX.git}" "$INSTALL_DIR"
fi

cd "$INSTALL_DIR"
go build -o harborx-agent ./cmd/agent

cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=HarborX Remote Agent
After=network-online.target
Wants=network-online.target

[Service]
WorkingDirectory=$INSTALL_DIR
Environment=HARBORX_AGENT_BASE_URL=$BASE_URL
Environment=HARBORX_AGENT_TOKEN=$TOKEN
Environment=HARBORX_AGENT_INTERVAL_SECONDS=$INTERVAL_SECONDS
ExecStart=$INSTALL_DIR/harborx-agent
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now "${SERVICE_NAME}.service"

echo "HarborX agent installed and running."
