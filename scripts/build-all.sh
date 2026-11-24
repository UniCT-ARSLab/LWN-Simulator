#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="$(node -p "require('$ROOT_DIR/package.json').version")"

source "$ROOT_DIR/scripts/lib/cli.sh"
source "$ROOT_DIR/scripts/lib/build.sh"

TARGET_OS="$(lwn_target_os)"
TARGET_ARCH="$(lwn_electron_arch)"

if [[ "$TARGET_OS" == linux ]]; then
  DEFAULT_ELECTRON_TARGETS="AppImage"
  ELECTRON_PLATFORM="linux"
else
  DEFAULT_ELECTRON_TARGETS="dmg,zip"
  ELECTRON_PLATFORM="mac"
fi

export LWN_ELECTRON_TARGETS="${LWN_ELECTRON_TARGETS:-$DEFAULT_ELECTRON_TARGETS}"

DESKTOP_ASSETS="$ROOT_DIR/apps/desktop/assets"
DESKTOP_RELEASE_DIR="$ROOT_DIR/releases/desktop"
BUILDER_OUTPUT="$DESKTOP_RELEASE_DIR/build"
SERVER_RELEASE_DIR="$ROOT_DIR/releases/server"

prepare_releases() {
  mkdir -p "$SERVER_RELEASE_DIR"

  mkdir -p "$DESKTOP_RELEASE_DIR"
  rm -rf -- "$BUILDER_OUTPUT"
  find "$DESKTOP_RELEASE_DIR" -maxdepth 1 -type f \( \
    -name '*.AppImage' -o -name '*.deb' -o -name '*.rpm' -o -name '*.dmg' -o -name '*.zip' \
  \) -delete
  find "$DESKTOP_RELEASE_DIR" -maxdepth 1 -type d -name '*.app' -prune -exec rm -rf -- {} +
  rm -f -- "$SERVER_RELEASE_DIR/lwn-server" "$ROOT_DIR/releases/SHA256SUMS"
  mkdir -p "$DESKTOP_ASSETS"
}

build_launcher() {
  run_step "Build desktop launcher" run_in_dir "$ROOT_DIR/apps/launcher" env VERSION="$VERSION" yarn build
}

build_electron() {
  local electron_arch
  local raw_targets="$LWN_ELECTRON_TARGETS"
  local -a targets electron_args

  electron_arch="$TARGET_ARCH"
  IFS=',' read -r -a targets <<< "$raw_targets"
  electron_args=("--$ELECTRON_PLATFORM")
  electron_args+=("${targets[@]}")
  electron_args+=("--$electron_arch")

  run_step "Package Electron $TARGET_OS artifacts ($electron_arch)" \
    run_in_dir "$ROOT_DIR/apps/desktop" env VERSION="$VERSION" yarn build "${electron_args[@]}"
}

publish_releases() {
  local server_path="$SERVER_RELEASE_DIR/lwn-server"
  local server_stage="$server_path.tmp"

  install -m 0755 -- "$DESKTOP_ASSETS/lwn-server" "$server_stage"
  if ! publish_electron_artifacts "$BUILDER_OUTPUT" "$DESKTOP_RELEASE_DIR"; then
    rm -f -- "$server_stage"
    return 1
  fi

  mv -- "$server_stage" "$server_path"
}

verify_releases() {
  test -s "$SERVER_RELEASE_DIR/lwn-server"
  test -x "$SERVER_RELEASE_DIR/lwn-server"
  verify_electron_artifacts "$DESKTOP_RELEASE_DIR"
}

cli_init "LWN-Simulator $TARGET_OS release" "$VERSION" 11
cli_header
cli_roadmap \
  "Prepare release directories" \
  "Verify workspace dependencies" \
  "Generate API contracts" \
  "Build UI components" \
  "Build SDK" \
  "Build frontend" \
  "Build backend" \
  "Build desktop launcher" \
  "Package Electron $TARGET_OS artifacts" \
  "Publish release artifacts" \
  "Verify release artifacts"

run_step "Prepare release directories" prepare_releases
ensure_dependencies
build_contracts
build_components
build_sdk
build_frontend
build_backend "$DESKTOP_ASSETS/lwn-server"
build_launcher
build_electron
run_step "Publish release artifacts" publish_releases
run_step "Verify release artifacts" verify_releases

printf '\n%sRelease artifacts%s\n' "$CLI_BOLD$CLI_CYAN" "$CLI_RESET"
cli_artifact "$SERVER_RELEASE_DIR/lwn-server"
for artifact in "$DESKTOP_RELEASE_DIR"/*; do
  cli_artifact "$artifact"
done
cli_footer
