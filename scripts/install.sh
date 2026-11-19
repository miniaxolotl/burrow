#!/bin/sh
# burrowd installer
# Usage: curl -fsSL https://raw.githubusercontent.com/miniaxolotl/burrow/production/scripts/install.sh | sh
#        curl -fsSL ... | sh -s -- --local   # Install to ~/.local/bin (no sudo)

set -e

REPO="miniaxolotl/burrow"
BINARY="burrowd"
# Update these together after reviewing a delivered binary release.
TAG="v0.3.23"
CHECKSUMS_SHA256="ea9c74f470ac11a70e74b8e9bd8c9ba4287a0cc7ed553c371d0dca47865cfe76"
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

verify_checksum() {
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum < "$1") || err "Could not checksum $1"
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 < "$1") || err "Could not checksum $1"
    else
        err "Install sha256sum or shasum to verify this release."
    fi
    [ "${actual%% *}" = "$2" ] || err "Checksum mismatch: $1"
}

install() {
    VERSION="${TAG#v}"
    ASSET="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
    RELEASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

    TMPDIR="$(mktemp -d)"
    trap 'rm -rf "$TMPDIR"' EXIT

    curl -fsSL --max-time 30 "${RELEASE_URL}/checksums.txt" -o "${TMPDIR}/checksums.txt" \
        || err "Cannot download the reviewed checksum inventory for ${TAG}."
    verify_checksum "${TMPDIR}/checksums.txt" "$CHECKSUMS_SHA256"
    expected=$(awk -v asset="$ASSET" '$2 == asset { count++; digest=$1 }
        END { if (count != 1 || length(digest) != 64 || digest ~ /[^0-9a-f]/) exit 1; print digest }' \
        "${TMPDIR}/checksums.txt") || err "Missing or invalid checksum entry for ${ASSET}."

    info "Downloading ${BINARY} ${TAG} for ${OS}/${ARCH}..."
    curl -fsSL "${RELEASE_URL}/${ASSET}" -o "${TMPDIR}/${ASSET}" \
        || err "Download failed. Check https://github.com/${REPO}/releases/tag/${TAG}"

    verify_checksum "${TMPDIR}/${ASSET}" "$expected"

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
    need curl
    install
    info "Done. Run '${BINARY} --version' to verify."
}

main "$@"
