#!/bin/sh
# Wrapper around download-liblbug.sh that places the library into lib/ where
# cgo_bundled.go expects it, then creates the versioned symlink that the
# runtime dynamic linker needs (the dylib/so embed a versioned install name).
#
# download-liblbug.sh is kept as a verbatim copy of the upstream script at:
#   https://raw.githubusercontent.com/LadybugDB/ladybug/refs/heads/main/scripts/download-liblbug.sh
# To update it: curl -fsSL <url above> -o download-liblbug.sh

# pipefail is a bashism; use -eu which POSIX sh supports.
# The script contains no pipelines, so pipefail is not needed here.
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LIB_DIR="$SCRIPT_DIR/lib"
UPSTREAM_SCRIPT="$SCRIPT_DIR/download-liblbug.sh"
UPSTREAM_URL="https://raw.githubusercontent.com/LadybugDB/ladybug/refs/heads/main/scripts/download-liblbug.sh"

# Fetch the upstream helper script if it is not already present.
if [ ! -f "$UPSTREAM_SCRIPT" ]; then
  echo "Fetching $UPSTREAM_URL ..."
  curl -fsSL "$UPSTREAM_URL" -o "$UPSTREAM_SCRIPT"
  chmod +x "$UPSTREAM_SCRIPT"
fi

# The upstream script defaults TARGET_DIR to SCRIPT_DIR/../lib because it assumes
# it lives in a scripts/ subdir. Override to put things in lib/ at the project root.
LBUG_TARGET_DIR="$LIB_DIR" bash "$UPSTREAM_SCRIPT"

# The dylib/so embed a versioned install name (e.g. @rpath/liblbug.0.dylib,
# liblbug.so.0) but the archive only contains the unversioned file.  Create a
# symlink so the runtime dynamic linker can resolve the name it expects.
OS="$(uname -s)"
case "$OS" in
  Darwin)
    if [ ! -e "$LIB_DIR/liblbug.0.dylib" ]; then
      ln -s liblbug.dylib "$LIB_DIR/liblbug.0.dylib"
      echo "Created symlink liblbug.0.dylib -> liblbug.dylib"
    fi
    ;;
  Linux)
    if [ ! -e "$LIB_DIR/liblbug.so.0" ]; then
      ln -s liblbug.so "$LIB_DIR/liblbug.so.0"
      echo "Created symlink liblbug.so.0 -> liblbug.so"
    fi
    ;;
esac

# Copy the header to the project root so it is available without a -I flag.
cp "$LIB_DIR/lbug.h" "$SCRIPT_DIR/lbug.h"
echo "Copied lbug.h to $SCRIPT_DIR"
