#!/bin/bash

set -e

# Function to download and extract a specific library
download_library() {
    local asset=$1
    local target_dir=$2
    local lib_pattern=$3
    local os_type=$4

    echo "Downloading asset: $asset"

    # Create temp directory
    local temp_dir=$(mktemp -d)
    cd "$temp_dir"

    # Download the asset
    local download_url="https://github.com/LadybugDB/ladybug/releases/latest/download/$asset"
    echo "   Downloading from: $download_url"

    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$asset" "$download_url"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$asset" "$download_url"
    else
        echo "ERROR: Neither curl nor wget is available"
        exit 1
    fi

    # Extract the asset
    case "$asset" in
        *.tar.gz)
            tar -xzf "$asset"
            ;;
        *)
            unzip -q "$asset"
            ;;
    esac

    # Find and copy library file
    local lib_file=$(find . -name "$lib_pattern" | head -1)
    if [ -n "$lib_file" ]; then
        # Create target directory
        mkdir -p "$OLDPWD/$target_dir"

        cp "$lib_file" "$OLDPWD/$target_dir/"
        echo "Copied $lib_pattern to $target_dir"

        # For Windows, also look for .lib if it exists
        if [ "$os_type" = "windows" ]; then
            local lib_import=$(find . -name "lbug_shared.lib" -o -name "lbug.lib" | head -1)
            if [ -n "$lib_import" ]; then
                cp "$lib_import" "$OLDPWD/$target_dir/"
                echo "Copied $(basename "$lib_import") to $target_dir"
            fi
        fi
    else
        echo "ERROR: Library file ($lib_pattern) not found in the extracted files"
        cd "$OLDPWD"
        rm -rf "$temp_dir"
        exit 1
    fi

    # Cleanup
    cd "$OLDPWD"
    rm -rf "$temp_dir"
}

# Check if we should download all libraries
if [ -n "$DOWNLOAD_ALL_LIBS" ]; then
    echo "Downloading all libraries for all platforms..."

    # Create temp directory for header file
    temp_dir=$(mktemp -d)
    cd "$temp_dir"

    # Download one package to get lbug.h (use osx-universal as it's reliable)
    curl -L -o "liblbug-osx-universal.tar.gz" "https://github.com/LadybugDB/ladybug/releases/latest/download/liblbug-osx-universal.tar.gz"
    tar -xzf "liblbug-osx-universal.tar.gz"
    lbug_file=$(find . -name "lbug.h" | head -1)
    if [ -n "$lbug_file" ]; then
        cp "$lbug_file" "$OLDPWD"
        echo "Copied lbug.h to project root"
    else
        echo "ERROR: lbug.h not found"
        exit 1
    fi
    cd "$OLDPWD"
    rm -rf "$temp_dir"

    # Download all platform libraries
    download_library "liblbug-linux-x86_64.tar.gz" "lib/dynamic/linux-amd64" "liblbug.so" "linux"
    download_library "liblbug-linux-aarch64.tar.gz" "lib/dynamic/linux-arm64" "liblbug.so" "linux"
    download_library "liblbug-osx-universal.tar.gz" "lib/dynamic/osx" "liblbug.dylib" "osx"
    download_library "liblbug-windows-x86_64.zip" "lib/dynamic/windows" "lbug_shared.dll" "windows"

    echo "All libraries downloaded successfully!"
    exit 0
fi

# Detect OS
os=$(uname -s)
case $os in
    Linux) os="linux" ;;
    Darwin) os="osx" ;;
    MINGW*|CYGWIN*) os="windows" ;;
    *) echo "ERROR: Unsupported OS: $os"; exit 1 ;;
esac

# Detect Architecture
arch=$(uname -m)
case $arch in
    x86_64) arch="x86_64" ;;
    aarch64|arm64) arch="aarch64" ;;
    *) echo "ERROR: Unsupported architecture: $arch"; exit 1 ;;
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
echo "Target Directory: $target_dir"

# Determine asset name and library pattern
if [ "$os" = "osx" ]; then
    asset="liblbug-osx-universal.tar.gz"
    lib_pattern="liblbug.dylib"
elif [ "$os" = "windows" ]; then
    if [ "$arch" != "x86_64" ]; then
        echo "ERROR: Windows only supports x86_64 architecture"
        exit 1
    fi
    asset="liblbug-windows-x86_64.zip"
    lib_pattern="lbug_shared.dll"
else
    asset="liblbug-linux-${arch}.tar.gz"
    lib_pattern="liblbug.so"
fi

echo "Detected OS: $os, Architecture: $arch"

# Create temp directory for header file
temp_dir=$(mktemp -d)
cd "$temp_dir"

# Download the asset to get lbug.h
download_url="https://github.com/LadybugDB/ladybug/releases/latest/download/$asset"
echo "   Downloading from: $download_url"

if command -v curl >/dev/null 2>&1; then
    curl -L -o "$asset" "$download_url"
elif command -v wget >/dev/null 2>&1; then
    wget -O "$asset" "$download_url"
else
    echo "ERROR: Neither curl nor wget is available"
    exit 1
fi

# Extract to get lbug.h
    case "$asset" in
        *.tar.gz)
            tar -xzf "$asset"
            ;;
        *)
            unzip -q "$asset"
            ;;
    esac

# Find and copy lbug.h
lbug_file=$(find . -name "lbug.h" | head -1)
if [ -n "$lbug_file" ]; then
    cp "$lbug_file" "$OLDPWD"
    echo "Copied lbug.h to project root"
else
    echo "ERROR: lbug.h not found in the extracted files"
    exit 1
fi

# Cleanup header temp directory
cd "$OLDPWD"
rm -rf "$temp_dir"

# Download the platform-specific library
download_library "$asset" "$target_dir" "$lib_pattern" "$os"

echo "Done!"