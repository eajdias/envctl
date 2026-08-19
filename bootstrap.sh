#!/usr/bin/env bash
# bootstrap.sh - Universal bootstrap installer for envctl on Linux & macOS
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash -s -- run all
#   curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash -s -- doctor

set -euo pipefail

REPO="eajdias/envctl"
VERSION="${ENVCTL_VERSION:-latest}"
INSTALL_DIR="${HOME}/.local/bin"
BINARY_NAME="envctl"
TARGET_PATH="${INSTALL_DIR}/${BINARY_NAME}"

echo "================================================================"
echo "  🚀 envctl: Universal Environment Provisioner Bootstrap (Linux)"
echo "================================================================"

# 1. Architecture and OS detection
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "${ARCH}" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "[-] Unsupported architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

if [ "${OS}" != "linux" ] && [ "${OS}" != "darwin" ]; then
  echo "[-] Unsupported operating system: ${OS}. Use bootstrap.ps1 on Windows." >&2
  exit 1
fi

# 2. Check local binary in current directory
if [ -f "./envctl" ] && [ "${FORCE:-0}" -ne 1 ]; then
  echo "[+] Using local envctl binary found in current directory"
  TARGET_PATH="./envctl"
else
  mkdir -p "${INSTALL_DIR}"

  # 3. Download Release archive
  if [ "${VERSION}" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/envctl-${OS}-${ARCH}.tar.gz"
  else
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/envctl-${OS}-${ARCH}.tar.gz"
  fi

  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "${TMP_DIR}"' EXIT

  echo "[*] Downloading envctl (${VERSION}) for ${OS}-${ARCH}..."
  if curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/envctl.tar.gz" 2>/dev/null; then
    tar -xzf "${TMP_DIR}/envctl.tar.gz" -C "${TMP_DIR}"
    cp "${TMP_DIR}/envctl" "${TARGET_PATH}"
    chmod +x "${TARGET_PATH}"
    echo "[+] Successfully installed envctl to ${TARGET_PATH}"
  else
    echo "[!] Pre-built binary download failed. Checking for Go toolchain..."
    if command -v go >/dev/null 2>&1; then
      echo "[*] Building envctl from source via Go..."
      git clone --depth 1 "https://github.com/${REPO}.git" "${TMP_DIR}/source"
      (cd "${TMP_DIR}/source" && go build -ldflags "-s -w" -o "${TARGET_PATH}" ./cmd/envctl)
      chmod +x "${TARGET_PATH}"
      echo "[+] Successfully built and installed envctl to ${TARGET_PATH}"
    else
      echo "[-] Failed to download binary and Go is not installed. Please install Go or verify release." >&2
      exit 1
    fi
  fi
fi

# 4. Ensure ~/.local/bin in PATH
case ":${PATH}:" in
  *:"${INSTALL_DIR}":*) ;;
  *) export PATH="${INSTALL_DIR}:${PATH}" ;;
esac

# 5. Execute envctl
if [ $# -eq 0 ]; then
  echo "[*] Executing: ${TARGET_PATH} run all"
  "${TARGET_PATH}" run all
else
  echo "[*] Executing: ${TARGET_PATH} $*"
  "${TARGET_PATH}" "$@"
fi
