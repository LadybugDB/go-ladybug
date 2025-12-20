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


# Map to Go conventions for variable usage, but custom path construction
if [ "$os" = "osx" ]; then
    go_os="darwin"
else
    go_os="$os"
fi

if [ "$arch" = "x86_64" ]; then
    go_arch="amd64"
elif [ "$arch" = "aarch64" ]; then
    go_arch="arm64"
else
    go_arch="$arch"
fi

# Construct target directory based on cgo_shared.go expectations
if [ "$go_os" = "linux" ]; then
    platform="linux-${go_arch}"
elif [ "$go_os" = "darwin" ]; then
    platform="osx"
elif [ "$go_os" = "windows" ]; then
    platform="windows"
else
    platform="${go_os}_${go_arch}"
fi

target_dir="lib/dynamic/$platform"
echo "🎯 Target Directory: $target_dir"

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

# Find and copy library file based on OS
case $os in
    linux)
        lib_pattern="liblbug.so"
        ;;
    osx)
        lib_pattern="liblbug.dylib"
        ;;
    windows)
        lib_pattern="lbug.dll"
        ;;
esac

lib_file=$(find . -name "$lib_pattern" | head -1)
if [ -n "$lib_file" ]; then
    # Create target directory
    mkdir -p "$OLDPWD/$target_dir"
    
    cp "$lib_file" "$OLDPWD/$target_dir/"
    echo "✅ Copied $lib_pattern to $target_dir"
    
    # For Windows, also look for .lib if it exists
    if [ "$os" = "windows" ]; then
        lib_import=$(find . -name "lbug.lib" | head -1)
        if [ -n "$lib_import" ]; then
            cp "$lib_import" "$OLDPWD/$target_dir/"
            echo "✅ Copied lbug.lib to $target_dir"
        fi
    fi
else
    echo "❌ Library file ($lib_pattern) not found in the extracted files"
    exit 1
fi

# Cleanup
cd "$OLDPWD"
rm -rf "$temp_dir"

echo "🎉 Done!"