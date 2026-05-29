#!/bin/sh
# Wrapper around upstream download-liblbug.sh (same pattern as go-ladybug).
# Downloads prebuilt liblbug into a local cache and writes CMake env flags.
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

ENV_FILE="${1:-$PROJECT_DIR/.cache/lbug-prebuilt.env}"
CACHE_LIB_DIR="${LBUG_TARGET_DIR:-$PROJECT_DIR/.cache/lbug-prebuilt/lib}"
LIB_KIND="${LBUG_LIB_KIND:-static}"
UPSTREAM_SCRIPT="$SCRIPT_DIR/download-liblbug.sh"
UPSTREAM_URL="https://raw.githubusercontent.com/LadybugDB/ladybug/refs/heads/main/scripts/download-liblbug.sh"

# Fetch the upstream helper if needed.
if [ ! -f "$UPSTREAM_SCRIPT" ]; then
  echo "Fetching $UPSTREAM_URL ..."
  curl -fsSL "$UPSTREAM_URL" -o "$UPSTREAM_SCRIPT"
  chmod +x "$UPSTREAM_SCRIPT"
fi

LBUG_TARGET_DIR="$CACHE_LIB_DIR" LBUG_LIB_KIND="$LIB_KIND" bash "$UPSTREAM_SCRIPT"

OS="$(uname -s)"
if [ "$LIB_KIND" = "shared" ]; then
  case "$OS" in
    Darwin)
      LIB_PATH="$CACHE_LIB_DIR/liblbug.dylib"
      ;;
    Linux)
      LIB_PATH="$CACHE_LIB_DIR/liblbug.so"
      ;;
    MINGW*|MSYS*|CYGWIN*)
      LIB_PATH="$CACHE_LIB_DIR/lbug_shared.dll"
      ;;
    *)
      echo "Unsupported OS: $OS" >&2
      exit 1
      ;;
  esac
else
  case "$OS" in
    MINGW*|MSYS*|CYGWIN*)
      LIB_PATH="$CACHE_LIB_DIR/lbug.lib"
      ;;
    *)
      LIB_PATH="$CACHE_LIB_DIR/liblbug.a"
      ;;
  esac
fi

if [ ! -f "$LIB_PATH" ]; then
  echo "Expected precompiled library not found at $LIB_PATH" >&2
  exit 1
fi

mkdir -p "$(dirname "$ENV_FILE")"
if [ "$LIB_KIND" = "shared" ]; then
  cat > "$ENV_FILE" <<EOF
LBUG_C_API_LIB_PATH=$LIB_PATH
EOF
else
  cat > "$ENV_FILE" <<EOF
EXTRA_CMAKE_FLAGS=-DBUILD_LBUG=FALSE -DBUILD_SHELL=FALSE -DLBUG_API_USE_PRECOMPILED_LIB=TRUE -DLBUG_API_PRECOMPILED_LIB_PATH=$LIB_PATH
EOF
fi

echo "Wrote $ENV_FILE"
echo "Resolved precompiled library: $LIB_PATH"
