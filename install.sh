#!/usr/bin/env bash
set -euo pipefail

APP_NAME="yx"
REPO="${YUNXIAO_FREE_CLI_REPO:-wanghongyi/yunxiao-free-cli}"
BIN_DIR="${YUNXIAO_FREE_CLI_BIN_DIR:-$HOME/.local/bin}"
MODULE_PATH="github.com/${REPO}/cmd/yx"

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

platform_os() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *)
      echo "unsupported os: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

platform_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "unsupported arch: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

install_release_binary() {
  if ! need_cmd curl || ! need_cmd tar; then
    return 1
  fi

  local os arch url tmp
  os="$(platform_os)"
  arch="$(platform_arch)"
  url="${YUNXIAO_FREE_CLI_BINARY_URL:-https://github.com/${REPO}/releases/latest/download/yx_${os}_${arch}.tar.gz}"

  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  echo "Downloading ${url} ..."
  if ! curl -fL "$url" -o "$tmp/yx.tar.gz"; then
    return 1
  fi

  tar -xzf "$tmp/yx.tar.gz" -C "$tmp"
  if [[ ! -f "$tmp/${APP_NAME}" ]]; then
    echo "binary ${APP_NAME} not found in archive" >&2
    return 1
  fi

  mkdir -p "$BIN_DIR"
  install -m 0755 "$tmp/${APP_NAME}" "$BIN_DIR/${APP_NAME}"
  return 0
}

install_with_go() {
  if ! need_cmd go; then
    return 1
  fi
  echo "Installing via go install ${MODULE_PATH}@latest ..."
  GOBIN="$BIN_DIR" go install "${MODULE_PATH}@latest"
}

main() {
  mkdir -p "$BIN_DIR"

  if install_release_binary; then
    echo "Installed ${APP_NAME} to ${BIN_DIR}/${APP_NAME}"
  elif install_with_go; then
    echo "Installed ${APP_NAME} to ${BIN_DIR}/${APP_NAME}"
  else
    echo "Install failed: neither release download nor go install succeeded." >&2
    echo "Please install Go >= 1.22 or publish release binaries first." >&2
    exit 1
  fi

  case ":${PATH}:" in
    *":${BIN_DIR}:"*) ;;
    *)
      echo "Warning: ${BIN_DIR} is not in PATH."
      echo "Add this line to your shell profile:"
      echo "  export PATH=\"${BIN_DIR}:\$PATH\""
      ;;
  esac

  echo "Run '${APP_NAME} auth' to configure Yunxiao PAT."
}

main "$@"
