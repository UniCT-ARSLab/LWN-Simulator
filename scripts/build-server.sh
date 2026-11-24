#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="$(node -p "require('$ROOT_DIR/package.json').version")"

source "$ROOT_DIR/scripts/lib/cli.sh"
source "$ROOT_DIR/scripts/lib/build.sh"

RELEASE_DIR="$ROOT_DIR/releases/server"

prepare_release() {
  mkdir -p "$RELEASE_DIR"
}

cli_init "LWN-Simulator server release" "$VERSION" 8
cli_header
cli_roadmap \
  "Prepare release directory" \
  "Verify workspace dependencies" \
  "Generate API contracts" \
  "Build UI components" \
  "Build SDK" \
  "Build frontend" \
  "Build backend" \
  "Verify release artifact"

run_step "Prepare release directory" prepare_release
ensure_dependencies
build_contracts
build_components
build_sdk
build_frontend
build_backend "$RELEASE_DIR/lwn-server"
run_step "Verify release artifact" test -x "$RELEASE_DIR/lwn-server"

printf '\n%sRelease artifacts%s\n' "$CLI_BOLD$CLI_CYAN" "$CLI_RESET"
cli_artifact "$RELEASE_DIR/lwn-server"
cli_footer
