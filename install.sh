#!/usr/bin/env bash
# Universal POSE 1-Liner Installer & Upgrader
# Usage: curl -fsSLO https://raw.githubusercontent.com/oseiaspereira88/pose/main/install.sh && bash install.sh
set -euo pipefail

REPO="oseiaspereira88/pose"
TARGET_DIR="${1:-.}"

# A release bundle ships this script beside the native binary. Prefer that
# binary: going to the provider first would install the *previous* release into
# the target and then validate it with the *new* engine, which is how every
# release ended up gated on the release before it (spec
# pose-installer-local-binary-precedence). Only an executable file literally
# beside this script qualifies — never a PATH lookup, and never when the script
# arrives on stdin (`curl | bash`), where it has no directory of its own.
SCRIPT_PATH="${BASH_SOURCE[0]:-}"
BUNDLED_BIN=""
if [[ -n "$SCRIPT_PATH" && -f "$SCRIPT_PATH" ]]; then
  candidate="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)/pose"
  [[ -x "$candidate" ]] && BUNDLED_BIN="$candidate"
fi

if [[ -n "$BUNDLED_BIN" ]]; then
  POSE_BIN="$BUNDLED_BIN"
  VERSION="$("$POSE_BIN" version | awk 'NR==1{sub(/-dev$/, "", $2); print $2}')"
  echo "[pose-installer] using the native binary beside this script (v${VERSION})"
else
  echo "[pose-installer] fetching latest release version from GitHub (${REPO})..."
  LATEST_TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | head -n1 | sed -E 's/.*"([^"]+)".*/\1/' || true)

  if [[ -z "$LATEST_TAG" ]]; then
    LATEST_TAG="v0.16.3"
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
fi

if [[ -d "$TARGET_DIR/.git" ]] || [[ -d "$TARGET_DIR/.pose" ]]; then
  echo "[pose-installer] upgrading & syncing POSE instance at ${TARGET_DIR}..."
  # `upgrade` and `check` operate on the current instance and take no
  # positional directory; calling them with one only printed a usage banner,
  # so neither the migration nor the final gate ever ran. There is nothing to
  # migrate before the first install, so skip it on a target without `.pose`.
  if [[ -d "$TARGET_DIR/.pose" ]]; then
    (cd "$TARGET_DIR" && "$POSE_BIN" upgrade) || true
  fi
  "$POSE_BIN" install "$TARGET_DIR" --force
  (cd "$TARGET_DIR" && "$POSE_BIN" check --strict)
fi

echo "[pose-installer] SUCCESS! POSE v${VERSION} ready."
