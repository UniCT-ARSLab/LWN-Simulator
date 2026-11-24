#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="$(node -p "require('$ROOT_DIR/package.json').version")"

source "$ROOT_DIR/scripts/lib/cli.sh"
source "$ROOT_DIR/scripts/lib/build.sh"

DESKTOP_ASSETS="$ROOT_DIR/apps/desktop/assets"
RELEASE_DIR="$ROOT_DIR/releases/desktop"
BUILDER_OUTPUT="$RELEASE_DIR/build"

prepare_release() {
  mkdir -p "$RELEASE_DIR"
  rm -rf -- "$BUILDER_OUTPUT"
  mkdir -p "$DESKTOP_ASSETS"
}

build_launcher() {
  run_step "Build desktop launcher" run_in_dir "$ROOT_DIR/apps/launcher" env VERSION="$VERSION" yarn build
}

build_electron() {
  local electron_arch
  local raw_targets="${LWN_ELECTRON_TARGETS:-AppImage}"
  local -a targets electron_args

  electron_arch="$(lwn_electron_arch)"
  IFS=',' read -r -a targets <<< "$raw_targets"
  electron_args=(--linux)
  electron_args+=("${targets[@]}")
  electron_args+=("--$electron_arch")

  run_step "Package Electron Linux artifacts ($electron_arch)" \
    run_in_dir "$ROOT_DIR/apps/desktop" env VERSION="$VERSION" yarn build "${electron_args[@]}"
}

copy_release() {
  publish_electron_artifacts "$BUILDER_OUTPUT" "$RELEASE_DIR"
}

verify_release() {
  verify_electron_artifacts "$RELEASE_DIR"
}

cli_init "LWN-Simulator desktop release" "$VERSION" 11
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
  "Package Electron Linux artifacts" \
  "Publish release artifact" \
  "Verify release artifact"

run_step "Prepare release directories" prepare_release
ensure_dependencies
build_contracts
build_components
build_sdk
build_frontend
build_backend "$DESKTOP_ASSETS/lwn-server"
build_launcher
build_electron
run_step "Publish release artifact" copy_release
run_step "Verify release artifact" verify_release

printf '\n%sRelease artifacts%s\n' "$CLI_BOLD$CLI_CYAN" "$CLI_RESET"
for artifact in "$RELEASE_DIR"/*; do
  cli_artifact "$artifact"
done
cli_footer
