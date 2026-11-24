#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_DIR="$ROOT_DIR/apps/frontend/out"
TARGET_DIR="$ROOT_DIR/apps/backend/internal/frontend/web"

source "$ROOT_DIR/scripts/lib/cli.sh"

copy_frontend() {
  if [[ ! -d "$SOURCE_DIR" || ! -f "$SOURCE_DIR/index.html" ]]; then
    printf 'Frontend export is missing or incomplete: %s\n' "$SOURCE_DIR" >&2
    return 1
  fi

  local staging_dir
  staging_dir="$(mktemp -d "$(dirname "$TARGET_DIR")/.frontend.XXXXXX")"

  if ! cp -a "$SOURCE_DIR"/. "$staging_dir/"; then
    rm -rf -- "$staging_dir"
    return 1
  fi

  find "$staging_dir" -type d -empty -delete

  rm -rf -- "$TARGET_DIR"
  mv -- "$staging_dir" "$TARGET_DIR"
}

cli_init "Embed frontend assets" "local" 1
cli_header
cli_roadmap "Copy static frontend into backend"
run_step "Copy static frontend into backend" copy_frontend

printf '\n%sEmbedded frontend%s\n' "$CLI_BOLD$CLI_CYAN" "$CLI_RESET"
cli_artifact "$TARGET_DIR"
cli_footer
