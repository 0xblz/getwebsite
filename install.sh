#!/bin/bash
set -e

REPO="0xblz/getwebsite"
BINARY="getwebsite"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case "$OS" in
    linux)  OS="linux" ;;
    darwin) OS="darwin" ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

# Get latest release tag
LATEST=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
    echo "Could not determine latest release. Building from source..."

    # Check for Go
    if ! command -v go &> /dev/null; then
        echo "Go is not installed. Please install Go 1.21+ first:"
        echo "  https://go.dev/dl/"
        exit 1
    fi

    TMPDIR=$(mktemp -d)
    trap "rm -rf $TMPDIR" EXIT

    echo "Cloning repository..."
    git clone --depth 1 "https://github.com/${REPO}.git" "$TMPDIR/getwebsite"

    echo "Building..."
    cd "$TMPDIR/getwebsite"
    go build -o "$BINARY" ./cmd/getwebsite

    echo "Installing to ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY" "${INSTALL_DIR}/${BINARY}"
    else
        sudo mv "$BINARY" "${INSTALL_DIR}/${BINARY}"
    fi

    echo "Done! Run 'getwebsite <url>' to get started."
    exit 0
fi

FILENAME="${BINARY}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${FILENAME}"

echo "Downloading ${BINARY} ${LATEST} for ${OS}/${ARCH}..."

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

curl -sSL "$DOWNLOAD_URL" -o "${TMPDIR}/${FILENAME}"
tar -xzf "${TMPDIR}/${FILENAME}" -C "$TMPDIR"

echo "Installing to ${INSTALL_DIR}..."
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"

echo "Done! Run 'getwebsite <url>' to get started."
