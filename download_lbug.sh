#!/bin/bash

set -e

# Detect OS
os=$(uname -s)
case $os in
    Linux) os="linux" ;;
    Darwin) os="osx" ;;
    MINGW*|CYGWIN*) os="windows" ;;
    *) echo "❌ Unsupported OS: $os"; exit 1 ;;
esac

# Detect Architecture
arch=$(uname -m)
case $arch in
    x86_64) arch="x86_64" ;;
    aarch64|arm64) arch="aarch64" ;;
    *) echo "❌ Unsupported architecture: $arch"; exit 1 ;;
esac

# Determine asset name
if [ "$os" = "osx" ]; then
    asset="liblbug-osx-universal.tar.gz"
    ext="tar.gz"
elif [ "$os" = "windows" ]; then
    if [ "$arch" != "x86_64" ]; then
        echo "❌ Windows only supports x86_64 architecture"
        exit 1
    fi
    asset="liblbug-windows-x86_64.zip"
    ext="zip"
else
    asset="liblbug-linux-${arch}.tar.gz"
    ext="tar.gz"
fi

echo "🔍 Detected OS: $os, Architecture: $arch"
echo "📦 Downloading asset: $asset"

# Create temp directory
temp_dir=$(mktemp -d)
cd "$temp_dir"

# Download the asset
download_url="https://github.com/LadybugDB/ladybug/releases/latest/download/$asset"
echo "   Downloading from: $download_url"

if command -v curl >/dev/null 2>&1; then
    curl -L -o "$asset" "$download_url"
elif command -v wget >/dev/null 2>&1; then
    wget -O "$asset" "$download_url"
else
    echo "❌ Neither curl nor wget is available"
    exit 1
fi

# Extract the asset
if [ "$ext" = "tar.gz" ]; then
    tar -xzf "$asset"
else
    unzip "$asset"
fi

# Find and copy lbug.h
lbug_file=$(find . -name "lbug.h" | head -1)
if [ -n "$lbug_file" ]; then
    cp "$lbug_file" "$OLDPWD"
    echo "✅ Copied lbug.h to project root"
else
    echo "❌ lbug.h not found in the extracted files"
    exit 1
fi

# Cleanup
cd "$OLDPWD"
rm -rf "$temp_dir"

echo "🎉 Done!"