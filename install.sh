#!/bin/sh
# install.sh — cross-platform installer for Flup
# Works on Linux (amd64/arm64), macOS (amd64/arm64), and WSL.
# Installs Go ≥1.23 automatically if missing, then builds flup from source.
#
# Usage:
#   curl -sSfL https://raw.githubusercontent.com/ankurCES/Flup/main/install.sh | sh
#   # — or —
#   wget -qO- https://raw.githubusercontent.com/ankurCES/Flup/main/install.sh | sh
#
# Environment variables:
#   INSTALL_DIR  — where to place the binary  (default: /usr/local/bin)
#   GO_VERSION   — Go version to install      (default: 1.23.0)

set -eu

REPO="github.com/ankurCES/Flup"
MODULE="${REPO}/cmd/flup"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
MIN_GO_MAJOR=1
MIN_GO_MINOR=23
GO_VERSION="${GO_VERSION:-1.23.0}"

# ── helpers ──────────────────────────────────────────────────────────────────

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
err()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || err "required tool '$1' not found"; }

detect_os() {
  case "$(uname -s)" in
    Linux*)  echo linux  ;;
    Darwin*) echo darwin  ;;
    *)       err "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)   echo amd64  ;;
    aarch64|arm64)   echo arm64  ;;
    armv7l|armhf)    echo armv6l ;;
    *)               err "unsupported arch: $(uname -m)" ;;
  esac
}

# Returns 0 if $1.$2 >= MIN_GO_MAJOR.MIN_GO_MINOR
go_version_ok() {
  _major="$1"; _minor="$2"
  [ "$_major" -gt "$MIN_GO_MAJOR" ] && return 0
  [ "$_major" -eq "$MIN_GO_MAJOR" ] && [ "$_minor" -ge "$MIN_GO_MINOR" ] && return 0
  return 1
}

# ── check / install Go ──────────────────────────────────────────────────────

ensure_go() {
  if command -v go >/dev/null 2>&1; then
    _ver="$(go version | sed 's/.*go\([0-9]*\)\.\([0-9]*\).*/\1 \2/')"
    _maj="$(echo "$_ver" | cut -d' ' -f1)"
    _min="$(echo "$_ver" | cut -d' ' -f2)"
    if go_version_ok "$_maj" "$_min"; then
      log "Go $(go version | sed 's/.*go/go/;s/ .*//' ) found — OK"
      return 0
    fi
    log "Go $(go version | sed 's/.*go/go/;s/ .*//' ) too old (need ≥${MIN_GO_MAJOR}.${MIN_GO_MINOR})"
  else
    log "Go not found"
  fi

  log "Installing Go ${GO_VERSION}..."
  _os="$(detect_os)"
  _arch="$(detect_arch)"
  _tarball="go${GO_VERSION}.${_os}-${_arch}.tar.gz"
  _url="https://go.dev/dl/${_tarball}"

  _tmpdir="$(mktemp -d)"
  trap 'rm -rf "$_tmpdir"' EXIT

  if command -v curl >/dev/null 2>&1; then
    curl -sSfL -o "${_tmpdir}/${_tarball}" "$_url"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "${_tmpdir}/${_tarball}" "$_url"
  else
    err "need curl or wget to download Go"
  fi

  _goroot="${HOME}/.local/go"
  rm -rf "$_goroot"
  mkdir -p "${HOME}/.local"
  tar -C "${HOME}/.local" -xzf "${_tmpdir}/${_tarball}"
  export PATH="${_goroot}/bin:${PATH}"
  export GOROOT="${_goroot}"
  log "Go installed to ${_goroot}"
}

# ── build & install ─────────────────────────────────────────────────────────

install_flup() {
  log "Building flup from source..."
  _tmpdir="$(mktemp -d)"

  GOBIN="$_tmpdir" go install "${MODULE}@latest" 2>&1

  if [ ! -f "${_tmpdir}/flup" ]; then
    err "build succeeded but binary not found — check module path"
  fi

  if [ -w "$INSTALL_DIR" ]; then
    mv "${_tmpdir}/flup" "${INSTALL_DIR}/flup"
  else
    log "Need sudo to install to ${INSTALL_DIR}"
    sudo mv "${_tmpdir}/flup" "${INSTALL_DIR}/flup"
  fi
  chmod +x "${INSTALL_DIR}/flup"
  rm -rf "$_tmpdir"

  log "flup installed to ${INSTALL_DIR}/flup"
}

# ── main ────────────────────────────────────────────────────────────────────

main() {
  log "Flup installer — $(detect_os)/$(detect_arch)"
  ensure_go
  install_flup
  log "Done! Run 'flup --help' to get started."
}

main
