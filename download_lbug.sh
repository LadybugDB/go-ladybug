#!/bin/bash

set -e

# Helper function to download a file
download_file() {
    local url=$1
    local output=$2

    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$output" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$output" "$url"
    else
        echo "ERROR: Neither curl nor wget is available"
        exit 1
    fi
}

# Helper function to extract an archive
extract_archive() {
    local archive=$1

    case "$archive" in
        *.tar.gz)
            tar -xzf "$archive"
            ;;
        *)
            unzip -q "$archive"
            ;;
    esac
}

detect_soname() {
    local lib_file=$1

    if command -v objdump >/dev/null 2>&1; then
        objdump -p "$lib_file" 2>/dev/null | awk '$1 == "SONAME" { print $2; exit }'
    elif command -v readelf >/dev/null 2>&1; then
        readelf -d "$lib_file" 2>/dev/null | awk -F'[][]' '/SONAME/ { print $2; exit }'
    fi
}

detect_install_name() {
    local lib_file=$1

    if command -v otool >/dev/null 2>&1; then
        otool -D "$lib_file" 2>/dev/null | awk 'NR == 2 { print $1; exit }'
    fi
}

# Function to download and extract a specific library
download_library() {
    local asset=$1
    local target_dir=$2
    local lib_pattern=$3
    local os_type=$4
    local copy_header=$5  # Optional: if set, also copy lbug.h
    local header_dest=$6  # Optional: where to copy lbug.h

    echo "Downloading asset: $asset"

    # Create temp directory
    local temp_dir=$(mktemp -d)
    cd "$temp_dir"

    # Download the asset
    local download_url="https://github.com/LadybugDB/ladybug/releases/latest/download/$asset"
    echo "   Downloading from: $download_url"
    download_file "$download_url" "$asset"

    # Extract the asset
    extract_archive "$asset"

    # Copy header file if requested
    if [ -n "$copy_header" ]; then
        local header_file=$(find . -name "lbug.h" | head -1)
        if [ -n "$header_file" ]; then
            mkdir -p "$OLDPWD/$header_dest"
            cp "$header_file" "$OLDPWD/$header_dest/"
            echo "Copied lbug.h to $header_dest"
        else
            echo "WARNING: lbug.h not found in the extracted files"
        fi
    fi

    mkdir -p "$OLDPWD/$target_dir"

    # Linux shared libraries typically ship as a symlink chain:
    #   liblbug.so -> liblbug.so.0 -> liblbug.so.0.x.y
    # Preserve those names so the runtime loader can satisfy the SONAME.
    if [ "$os_type" = "linux" ]; then
        local found_lib=0
        local lib_list="$temp_dir/lib_files.txt"
        local copied_main_lib=""
        find . \( -type f -o -type l \) -name "${lib_pattern}*" | sort > "$lib_list"

        while IFS= read -r lib_file; do
            found_lib=1
            cp -a "$lib_file" "$OLDPWD/$target_dir/"
            echo "Copied $(basename "$lib_file") to $target_dir"
            if [ "$(basename "$lib_file")" = "$lib_pattern" ]; then
                copied_main_lib="$OLDPWD/$target_dir/$lib_pattern"
            fi
        done < "$lib_list"

        if [ "$found_lib" -eq 0 ]; then
            echo "ERROR: Library file (${lib_pattern}*) not found in the extracted files"
            cd "$OLDPWD"
            rm -rf "$temp_dir"
            exit 1
        fi

        if [ -n "$copied_main_lib" ]; then
            local soname=$(detect_soname "$copied_main_lib")
            if [ -n "$soname" ] && [ "$soname" != "$lib_pattern" ] && [ ! -e "$OLDPWD/$target_dir/$soname" ]; then
                ln -s "$lib_pattern" "$OLDPWD/$target_dir/$soname"
                echo "Created $soname -> $lib_pattern in $target_dir"
            fi
        fi
    else
        # Find and copy library file
        local lib_file=$(find . -name "$lib_pattern" | head -1)
        if [ -n "$lib_file" ]; then
            cp "$lib_file" "$OLDPWD/$target_dir/"
            echo "Copied $lib_pattern to $target_dir"

            if [ "$os_type" = "osx" ]; then
                local copied_lib="$OLDPWD/$target_dir/$lib_pattern"
                local install_name=$(detect_install_name "$copied_lib")
                local install_basename=""

                if [ -n "$install_name" ]; then
                    install_basename=$(basename "$install_name")
                fi

                if [ -n "$install_basename" ] && [ "$install_basename" != "$lib_pattern" ] && [ ! -e "$OLDPWD/$target_dir/$install_basename" ]; then
                    ln -s "$lib_pattern" "$OLDPWD/$target_dir/$install_basename"
                    echo "Created $install_basename -> $lib_pattern in $target_dir"
                fi
            fi

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
    fi

    # Cleanup
    cd "$OLDPWD"
    rm -rf "$temp_dir"
}

# Parse arguments
out_dir=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -out)
            out_dir="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

# Detect OS and Architecture
os=$(uname -s)
arch=$(uname -m)

case $os in
    Linux) os="linux" ;;
    Darwin) os="osx" ;;
    MINGW*|CYGWIN*) os="windows" ;;
    *) echo "ERROR: Unsupported OS: $os"; exit 1 ;;
esac

case $arch in
    x86_64) arch="x86_64" ;;
    aarch64|arm64) arch="aarch64" ;;
    *) echo "ERROR: Unsupported architecture: $arch"; exit 1 ;;
esac

# Construct target directory based on cgo_shared.go expectations or use provided out_dir
if [ -n "$out_dir" ]; then
    target_dir="$out_dir"
else
    # Map architecture for path construction
    path_arch="$arch"
    [ "$arch" = "x86_64" ] && path_arch="amd64"
    [ "$arch" = "aarch64" ] && path_arch="arm64"

    # Build platform-specific path
    case "$os" in
        linux)   platform="linux-${path_arch}" ;;
        osx)     platform="osx" ;;
        windows) platform="windows" ;;
    esac
    target_dir="lib/dynamic/$platform"
fi

echo "Detected OS: $os, Architecture: $arch"
echo "Target Directory: $target_dir"

# Determine asset name and library pattern
case "$os" in
    osx)
        asset="liblbug-osx-universal.tar.gz"
        lib_pattern="liblbug.dylib"
        ;;
    windows)
        if [ "$arch" != "x86_64" ]; then
            echo "ERROR: Windows only supports x86_64 architecture"
            exit 1
        fi
        asset="liblbug-windows-x86_64.zip"
        lib_pattern="lbug_shared.dll"
        ;;
    linux)
        asset="liblbug-linux-${arch}.tar.gz"
        lib_pattern="liblbug.so"
        ;;
esac

# Download the platform-specific library
# Only extract header if downloading to go-ladybug source (no -out flag)
if [ -n "$out_dir" ]; then
    # External directory - don't copy header
    download_library "$asset" "$target_dir" "$lib_pattern" "$os"
else
    # go-ladybug source tree - copy header to project root
    download_library "$asset" "$target_dir" "$lib_pattern" "$os" "yes" "."
fi

echo "Done!"
