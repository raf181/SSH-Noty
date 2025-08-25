#!/usr/bin/env bash
set -euo pipefail

# Usage examples:
#   curl -fsSL https://raw.githubusercontent.com/raf181/SSH-Noty/main/install.sh | sudo bash -s -- --webhook="https://hooks.slack.com/services/..."

OWNER_REPO="raf181/SSH-Noty"
BIN_NAME="ssh-noti"
INSTALL_DIR="/opt/ssh-noti"
STATE_DIR="$INSTALL_DIR/state"

WEBHOOK=""
MODE="realtime"
TAG=""           # When set, download this specific release tag (e.g., v0.1.0 or nightly)
DOWNLOAD_ONLY=0  # When 1, only download the binary and exit
ARCH_OVERRIDE=""
PREFIX=""        # When set, install under this directory instead of /opt/ssh-noti
NO_SYSTEMD=0     # When 1, skip systemd unit installation
SERVICE_USER="root" # Default to running the service as root (can be overridden)

while [[ $# -gt 0 ]]; do
  case "$1" in
    --webhook=*) WEBHOOK="${1#*=}"; shift ;;
    --mode=*) MODE="${1#*=}"; shift ;;
    --tag=*) TAG="${1#*=}"; shift ;;
    --download-only) DOWNLOAD_ONLY=1; shift ;;
    --arch=*) ARCH_OVERRIDE="${1#*=}"; shift ;;
    --prefix=*) PREFIX="${1#*=}"; shift ;;
    --no-systemd) NO_SYSTEMD=1; shift ;;
    --service-user=*) SERVICE_USER="${1#*=}"; shift ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

# Root requirement: only if installing to system directories and managing systemd
if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
  if [[ "$DOWNLOAD_ONLY" -eq 1 ]]; then
    : # allowed
  elif [[ -n "$PREFIX" ]]; then
    echo "Running without root; will install under PREFIX=$PREFIX and skip systemd/user creation." >&2
    NO_SYSTEMD=1
  else
    echo "Please run as root (or use --download-only, or --prefix to install in a user-writable dir)." >&2
    exit 1
  fi
fi

command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 1; }

# Determine install root
if [[ -n "$PREFIX" ]]; then
  INSTALL_DIR="$PREFIX"
  STATE_DIR="$INSTALL_DIR/state"
fi

# Create directories when installing (not in download-only)
if [[ "$DOWNLOAD_ONLY" -ne 1 ]]; then
  echo "Creating directories..."
  mkdir -p "$INSTALL_DIR" "$STATE_DIR"
fi

# Determine release tag
if [[ -z "$TAG" ]]; then
  echo "Fetching latest release for $OWNER_REPO..."
  LATEST_URL=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/${OWNER_REPO}/releases/latest")
  LATEST_TAG=${LATEST_URL##*/}
else
  LATEST_TAG="$TAG"
  echo "Using specified release tag: $LATEST_TAG"
fi

ASSET_URL_AMD64="https://github.com/${OWNER_REPO}/releases/download/${LATEST_TAG}/ssh-noti_linux_amd64"
ASSET_URL_ARM64="https://github.com/${OWNER_REPO}/releases/download/${LATEST_TAG}/ssh-noti_linux_arm64"

# Determine architecture
ARCH=$(uname -m)
if [[ -n "$ARCH_OVERRIDE" ]]; then
  ARCH="$ARCH_OVERRIDE"
fi
case "$ARCH" in
  x86_64|amd64) DL_URL="$ASSET_URL_AMD64" ;;
  aarch64|arm64) DL_URL="$ASSET_URL_ARM64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

if [[ -z "$DL_URL" || "$DL_URL" == "null" ]]; then
  echo "No binary URL constructed for arch $ARCH" >&2
  exit 1
fi

# Validate URL exists (avoid downloading 404 HTML as binary)
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -L "$DL_URL") || true
if [[ "$HTTP_STATUS" != "200" ]]; then
  echo "Release asset not found (HTTP $HTTP_STATUS) at: $DL_URL" >&2
  echo "- Ensure a release/tag '${LATEST_TAG}' exists with Linux binaries, or use --tag=nightly."
  exit 1
fi

# Download binary
TMP_BIN=$(mktemp)
echo "Downloading $DL_URL ..."
curl -fsSL "$DL_URL" -o "$TMP_BIN"

if [[ "$DOWNLOAD_ONLY" -eq 1 ]]; then
  OUT_NAME="$PWD/${BIN_NAME}"
  if [[ "$ARCH" == "arm64" || "$ARCH" == "aarch64" ]]; then
    OUT_NAME+="_linux_arm64"
  else
    OUT_NAME+="_linux_amd64"
  fi
  install -m 0755 "$TMP_BIN" "$OUT_NAME"
  rm -f "$TMP_BIN"
  echo "Downloaded binary to: $OUT_NAME"
  exit 0
fi

install -m 0755 "$TMP_BIN" "$INSTALL_DIR/$BIN_NAME"
rm -f "$TMP_BIN"

# Write config if missing
if [[ ! -f "$INSTALL_DIR/config.json" ]]; then
  cat > "$INSTALL_DIR/config.json" <<JSON
{
  "slack_webhook": "${WEBHOOK}",
  "mode": "${MODE}",
  "sources": { "prefer": "auto", "file_paths": ["/var/log/auth.log", "/var/log/secure"], "systemd_units": ["sshd.service", "ssh.service"] },
  "rules": { "notify_success": true, "notify_failure": true, "notify_invalid_user": true, "notify_root_login": true, "exclude_users": [], "exclude_ips": [], "include_ips": [] },
  "rate_limit": { "window_seconds": 60, "max_events_per_window": 20, "dedup_window_seconds": 30 },
  "batch": { "window_seconds": 3600, "min_failed_threshold": 5 },
  "geoip": { "enabled": false, "db_path": "/usr/share/GeoIP/GeoLite2-City.mmdb" },
  "formatting": { "concise": false, "show_key_fingerprint": true, "show_hostname": true },
  "telemetry": { "log_level": "INFO", "log_file": "" }
}
JSON
fi

# Config file permissions
if [[ "$NO_SYSTEMD" -ne 1 ]]; then
  if [[ "$SERVICE_USER" != "root" ]]; then
    if ! id -u "$SERVICE_USER" >/dev/null 2>&1; then
      useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER" || true
    fi
    chown "$SERVICE_USER":"$SERVICE_USER" "$INSTALL_DIR/config.json"
    chmod 0640 "$INSTALL_DIR/config.json"
  else
    chown root:root "$INSTALL_DIR/config.json"
    chmod 0600 "$INSTALL_DIR/config.json"
  fi
else
  chmod 0600 "$INSTALL_DIR/config.json"
fi

# Directory ownership
chown -R root:root "$INSTALL_DIR"
chmod 0755 "$INSTALL_DIR"
if [[ "$SERVICE_USER" != "root" ]]; then
  chown -R "$SERVICE_USER":"$SERVICE_USER" "$STATE_DIR"
  chmod 0700 "$STATE_DIR"
else
  chown -R root:root "$STATE_DIR"
  chmod 0700 "$STATE_DIR"
fi

# Systemd units
if [[ "$NO_SYSTEMD" -ne 1 ]] && command -v systemctl >/dev/null 2>&1; then
  UNIT_DIR="/etc/systemd/system"
  mkdir -p "$UNIT_DIR"

  cat > "$UNIT_DIR/ssh-noti.service" <<UNIT
[Unit]
Description=SSH Notifier (Go)
After=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
ExecStart=/opt/ssh-noti/ssh-noti --daemon --config=/opt/ssh-noti/config.json
Restart=always
RestartSec=5
AmbientCapabilities=
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
UNIT

  cat > "$UNIT_DIR/ssh-noti-summary.service" <<UNIT
[Unit]
Description=SSH Notifier Summary Run

[Service]
Type=oneshot
User=${SERVICE_USER}
Group=${SERVICE_USER}
ExecStart=/opt/ssh-noti/ssh-noti --batch --config=/opt/ssh-noti/config.json
UNIT

  cat > "$UNIT_DIR/ssh-noti-summary.timer" <<'UNIT'
[Unit]
Description=SSH Notifier Summary (hourly)

[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
UNIT

  chmod 0644 "$UNIT_DIR/ssh-noti.service" "$UNIT_DIR/ssh-noti-summary.service" "$UNIT_DIR/ssh-noti-summary.timer"
  systemctl daemon-reload
  systemctl enable --now ssh-noti.service
  systemctl enable --now ssh-noti-summary.timer || true
  echo "Systemd service installed and started: ssh-noti.service (as ${SERVICE_USER})"
else
  echo "systemctl not found or disabled; skipping service installation. You can run $INSTALL_DIR/$BIN_NAME --daemon manually." >&2
fi

echo "Installed ssh-noti to $INSTALL_DIR"
echo "Configure webhook in $INSTALL_DIR/config.json if not provided via --webhook"
echo "Test: $INSTALL_DIR/$BIN_NAME --test --config=$INSTALL_DIR/config.json"
