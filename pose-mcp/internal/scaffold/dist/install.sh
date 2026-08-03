#!/usr/bin/env bash
# Universal POSE 1-Liner Installer & Upgrader
# Usage: curl -fsSL https://raw.githubusercontent.com/oseiaspereira88/pose/main/install.sh | bash
set -euo pipefail

REPO="oseiaspereira88/pose"
TARGET_DIR="${1:-.}"

echo "[pose-installer] fetching latest release version from GitHub (${REPO})..."
LATEST_TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | head -n1 | sed -E 's/.*"([^"]+)".*/\1/' || true)

if [[ -z "$LATEST_TAG" ]]; then
  LATEST_TAG="v0.16.1"
fi

VERSION="${LATEST_TAG#v}"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

ASSET="pose_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${ASSET}"

echo "[pose-installer] downloading POSE v${VERSION} (${OS}_${ARCH})..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

if curl -fsSL "$URL" -o "$TMP_DIR/$ASSET"; then
  tar -xzf "$TMP_DIR/$ASSET" -C "$TMP_DIR" pose
  INSTALL_DIR="${HOME}/.local/bin"
  mkdir -p "$INSTALL_DIR"
  install -m 0755 "$TMP_DIR/pose" "$INSTALL_DIR/pose"
  echo "[pose-installer] POSE binary updated to ${INSTALL_DIR}/pose (v${VERSION})"
  POSE_BIN="$INSTALL_DIR/pose"
elif command -v pose >/dev/null 2>&1; then
  POSE_BIN="pose"
else
  echo "[pose-installer] Error: failed to download release asset from ${URL}" >&2
  exit 1
fi

if [[ -d "$TARGET_DIR/.git" ]] || [[ -d "$TARGET_DIR/.pose" ]]; then
  echo "[pose-installer] upgrading & syncing POSE instance at ${TARGET_DIR}..."
  "$POSE_BIN" upgrade "$TARGET_DIR" || true
  "$POSE_BIN" install "$TARGET_DIR" --force
  "$POSE_BIN" check --strict "$TARGET_DIR" || true
fi

echo "[pose-installer] SUCCESS! POSE v${VERSION} ready."
