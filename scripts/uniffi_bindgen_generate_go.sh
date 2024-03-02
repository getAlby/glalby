#!/bin/bash

set -e

if ! command -v cross &> /dev/null; then
  echo "cross-rs is required to build bindings. Install it by running:"
  echo "  cargo install cross --git https://github.com/cross-rs/cross"
  exit 1
fi

uniffi-bindgen-go src/glalby.udl -o . -c ./uniffi.toml

build_lib() {
  local TOOL=$1
  local TARGET=$2
  local OUTPUT_FILE=$3

  $TOOL build --release --target $TARGET || exit 1
  mkdir -p "glalby/$TARGET" || exit 1
  cp "target/$TARGET/release/$OUTPUT_FILE" "glalby/$TARGET/" || exit 1
}

# If we're running on macOS, build the macOS library using the host compiler.
# Cross compilation is not supported (needs more complex setup).
if [[ "$OSTYPE" == "darwin"* ]]; then
  build_lib "cargo" "aarch64-apple-darwin" "libglalby_bindings.dylib"
fi

build_lib "cross" "x86_64-unknown-linux-gnu" "libglalby_bindings.so"
build_lib "cross" "x86_64-pc-windows-gnu" "glalby_bindings.dll"
