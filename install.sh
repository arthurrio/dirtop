#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────────────────────────────────────
# dirtop installer
# https://github.com/arthurrio/dirtop
# ─────────────────────────────────────────────────────────────────────────────

REPO="arthurrio/dirtop"
BINARY="dirtop"
INSTALL_DIR=""

# ── Helpers ───────────────────────────────────────────────────────────────────

info()    { printf "\033[1;34m==>\033[0m %s\n" "$*"; }
success() { printf "\033[1;32m  ✓\033[0m %s\n" "$*"; }
warn()    { printf "\033[1;33m  !\033[0m %s\n" "$*" >&2; }
die()     { printf "\033[1;31merror:\033[0m %s\n" "$*" >&2; exit 1; }

# ── OS / arch detection ───────────────────────────────────────────────────────

detect_platform() {
  local os arch

  case "$(uname -s)" in
    Linux)  os="linux"  ;;
    Darwin) os="darwin" ;;
    *)      die "Unsupported operating system: $(uname -s). Only Linux and macOS are supported." ;;
  esac

  case "$(uname -m)" in
    x86_64 | amd64)          arch="amd64"  ;;
    aarch64 | arm64)         arch="arm64"  ;;
    armv7l | armv6l | armhf) arch="arm"    ;;
    i386 | i686)             arch="386"    ;;
    *)                       die "Unsupported architecture: $(uname -m)." ;;
  esac

  echo "${os}_${arch}"
}

# ── Install directory ─────────────────────────────────────────────────────────

choose_install_dir() {
  # Prefer /usr/local/bin if writable (or if running as root)
  if [ -w "/usr/local/bin" ] || [ "$(id -u)" -eq 0 ]; then
    echo "/usr/local/bin"
    return
  fi

  # Fall back to ~/.local/bin
  local local_bin="$HOME/.local/bin"
  mkdir -p "$local_bin"
  echo "$local_bin"
}

add_to_path_hint() {
  local dir="$1"
  if [[ ":$PATH:" != *":${dir}:"* ]]; then
    warn "${dir} is not in your PATH."
    warn "Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    warn ""
    warn "  export PATH=\"\$PATH:${dir}\""
    warn ""
  fi
}

# ── Method 1: go install ──────────────────────────────────────────────────────

install_via_go() {
  info "Installing via 'go install'..."
  GOBIN="$INSTALL_DIR" go install "github.com/${REPO}@latest"
  success "Installed to ${INSTALL_DIR}/${BINARY}"
}

# ── Method 2: download binary from GitHub releases ───────────────────────────

install_via_release() {
  local platform="$1"
  info "Fetching latest release from github.com/${REPO}..."

  # Resolve latest release tag
  local latest_url="https://api.github.com/repos/${REPO}/releases/latest"
  local tag

  if command -v curl &>/dev/null; then
    tag=$(curl -fsSL "$latest_url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  elif command -v wget &>/dev/null; then
    tag=$(wget -qO- "$latest_url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  else
    die "Neither curl nor wget found. Please install one of them and try again."
  fi

  if [ -z "$tag" ]; then
    die "Could not determine the latest release tag. Please check https://github.com/${REPO}/releases"
  fi

  info "Latest release: ${tag}"

  local archive="${BINARY}_${platform}.tar.gz"
  local download_url="https://github.com/${REPO}/releases/download/${tag}/${archive}"
  local tmp_dir
  tmp_dir=$(mktemp -d)
  trap 'rm -rf "$tmp_dir"' EXIT

  info "Downloading ${archive}..."
  if command -v curl &>/dev/null; then
    curl -fsSL "$download_url" -o "${tmp_dir}/${archive}"
  else
    wget -qO "${tmp_dir}/${archive}" "$download_url"
  fi

  info "Extracting..."
  tar -xzf "${tmp_dir}/${archive}" -C "$tmp_dir"

  if [ ! -f "${tmp_dir}/${BINARY}" ]; then
    die "Binary '${BINARY}' not found in archive. Archive contents may have changed."
  fi

  chmod +x "${tmp_dir}/${BINARY}"

  # Install (use sudo if needed)
  if [ -w "$INSTALL_DIR" ]; then
    mv "${tmp_dir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  else
    info "Requesting sudo to install to ${INSTALL_DIR}..."
    sudo mv "${tmp_dir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  fi

  success "Installed to ${INSTALL_DIR}/${BINARY}"
}

# ── Verify ────────────────────────────────────────────────────────────────────

verify() {
  if command -v "$BINARY" &>/dev/null; then
    success "Verified: $(command -v "$BINARY")"
  else
    warn "Binary installed but '${BINARY}' is not in the current PATH."
    add_to_path_hint "$INSTALL_DIR"
  fi
}

# ── Main ──────────────────────────────────────────────────────────────────────

main() {
  echo ""
  echo "  dirtop installer"
  echo "  https://github.com/${REPO}"
  echo ""

  INSTALL_DIR=$(choose_install_dir)
  info "Install directory: ${INSTALL_DIR}"

  # Try go install first — cleanest method for a Go tool
  if command -v go &>/dev/null; then
    install_via_go
  else
    warn "Go not found. Falling back to binary release download."
    local platform
    platform=$(detect_platform)
    info "Detected platform: ${platform}"
    install_via_release "$platform"
  fi

  add_to_path_hint "$INSTALL_DIR"
  verify

  echo ""
  success "Done! Run 'dirtop' to start monitoring your current directory."
  echo ""
  echo "  Usage:"
  echo "    dirtop                     # monitor current directory"
  echo "    dirtop ~/code/myproject    # monitor a specific path"
  echo "    dirtop -i 5                # start with 5s refresh interval"
  echo ""
  echo "  Keyboard shortcuts:"
  echo "    c  cycle chart mode     m  cycle metric     i  cycle interval     q  quit"
  echo ""
}

main
