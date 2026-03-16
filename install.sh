#!/bin/sh
set -e

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    ARCH="arm64"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

URL="https://github.com/neozmmv/Lighthouse/releases/latest/download/lighthouse-${OS}-${ARCH}"

echo "Downloading Lighthouse for ${OS}/${ARCH}..."
curl -fsSL "$URL" -o /tmp/lighthouse
chmod +x /tmp/lighthouse
sudo mv /tmp/lighthouse /usr/local/bin/lighthouse

echo "Lighthouse installed! Run 'lighthouse up' to get started."