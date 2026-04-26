#!/bin/sh
# burrowd installer
# Usage: curl -fsSL https://raw.githubusercontent.com/miniaxolotl/burrow/production/scripts/install.sh | sh
#        curl -fsSL ... | sh -s -- --local   # Install to ~/.local/bin (no sudo)

set -e

REPO="miniaxolotl/burrow"
BINARY="burrowd"
LOCAL_INSTALL=""

info() { printf '  \033[1;34m>\033[0m %s\n' "$*"; }
warn() { printf '  \033[1;33m>\033[0m %s\n' "$*"; }
err()  { printf '  \033[1;31m!\033[0m %s\n' "$*" >&2; exit 1; }

need() {
    command -v "$1" >/dev/null 2>&1 || err "Required tool '$1' not found. Please install it and try again."
}

parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --local|-l)
                LOCAL_INSTALL="1"
                ;;
            --help|-h)
                echo "Usage: install.sh [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --local, -l    Install to ~/.local/bin (no sudo required)"
                echo "  --help, -h     Show this help message"
                exit 0
                ;;
            *)
                warn "Unknown option: $1"
                ;;
        esac
        shift
    done
}

detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux)  OS="linux" ;;
        Darwin) OS="darwin" ;;
        *)      err "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64)  ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)             err "Unsupported architecture: $ARCH" ;;
    esac
}

fetch_latest_tag() {
    need curl

    TAG="$(curl -fsSI "https://github.com/${REPO}/releases/latest" 2>/dev/null \
        | grep -i '^location:' \
        | head -1 \
        | sed 's|.*/tag/||' \
        | tr -d '\r\n')"

    [ -n "$TAG" ] || err "Could not determine latest release. Check https://github.com/${REPO}/releases"
}

verify_checksum() {
    local asset_path="$1"
    local asset_name
    asset_name="$(basename "$asset_path")"
    local checksums_url="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"
    local checksums_path="${TMPDIR}/checksums.txt"

    if ! curl -fsSL --max-time 10 "$checksums_url" -o "$checksums_path" 2>/dev/null; then
        warn "No checksums.txt found — skipping integrity check"
        return
    fi

    local expected
    expected="$(grep "  ${asset_name}$\| ${asset_name}$" "$checksums_path" | awk '{print $1}')"
    if [ -z "$expected" ]; then
        warn "No checksum entry for ${asset_name} — skipping integrity check"
        return
    fi

    info "Verifying checksum..."
    if command -v sha256sum >/dev/null 2>&1; then
        echo "${expected}  ${asset_path}" | sha256sum -c --quiet \
            || err "Checksum verification failed."
    elif command -v shasum >/dev/null 2>&1; then
        echo "${expected}  ${asset_path}" | shasum -a 256 -q -c \
            || err "Checksum verification failed."
    else
        warn "Neither sha256sum nor shasum available — skipping integrity check"
    fi
}

install() {
    VERSION="${TAG#v}"
    ASSET="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"

    TMPDIR="$(mktemp -d)"
    trap 'rm -rf "$TMPDIR"' EXIT

    info "Downloading ${BINARY} ${TAG} for ${OS}/${ARCH}..."
    curl -fsSL "$DOWNLOAD_URL" -o "${TMPDIR}/${ASSET}" \
        || err "Download failed. Check https://github.com/${REPO}/releases/tag/${TAG}"

    verify_checksum "${TMPDIR}/${ASSET}"

    info "Extracting..."
    tar -xzf "${TMPDIR}/${ASSET}" -C "$TMPDIR"
    chmod +x "${TMPDIR}/${BINARY}"

    if [ -n "$LOCAL_INSTALL" ]; then
        INSTALL_DIR="${HOME}/.local/bin"
        mkdir -p "$INSTALL_DIR"
        info "Installing to ${INSTALL_DIR} (--local mode)..."
    elif [ -w /usr/local/bin ]; then
        INSTALL_DIR="/usr/local/bin"
    elif command -v sudo >/dev/null 2>&1; then
        info "Installing to /usr/local/bin (requires sudo)..."
        if [ -t 0 ]; then
            sudo mv "${TMPDIR}/${BINARY}" "/usr/local/bin/${BINARY}" && \
                info "Installed ${BINARY} to /usr/local/bin/${BINARY}" && return
        elif [ -e /dev/tty ]; then
            sudo mv "${TMPDIR}/${BINARY}" "/usr/local/bin/${BINARY}" </dev/tty && \
                info "Installed ${BINARY} to /usr/local/bin/${BINARY}" && return
        fi
        warn "sudo failed, falling back to ~/.local/bin"
        INSTALL_DIR="${HOME}/.local/bin"
        mkdir -p "$INSTALL_DIR"
    else
        INSTALL_DIR="${HOME}/.local/bin"
        mkdir -p "$INSTALL_DIR"
        info "Installing to ${INSTALL_DIR} (no sudo available)..."
    fi

    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    info "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"

    case ":$PATH:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            warn "Add ${INSTALL_DIR} to your PATH:"
            echo ""
            echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
            echo ""
            ;;
    esac
}

main() {
    parse_args "$@"
    info "burrowd installer"
    detect_platform
    fetch_latest_tag
    install
    info "Done. Run '${BINARY} --version' to verify."
}

main "$@"
