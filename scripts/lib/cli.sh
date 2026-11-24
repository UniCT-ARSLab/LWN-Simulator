#!/usr/bin/env bash

set -Eeuo pipefail

if [[ -t 1 && -t 2 && -z "${NO_COLOR:-}" && "${TERM:-dumb}" != "dumb" ]]; then
  readonly CLI_RESET=$'\033[0m'
  readonly CLI_BOLD=$'\033[1m'
  readonly CLI_DIM=$'\033[2m'
  readonly CLI_CYAN=$'\033[36m'
  readonly CLI_BLUE=$'\033[34m'
  readonly CLI_GREEN=$'\033[32m'
  readonly CLI_YELLOW=$'\033[33m'
  readonly CLI_RED=$'\033[31m'
  readonly CLI_INTERACTIVE=true
else
  readonly CLI_RESET=""
  readonly CLI_BOLD=""
  readonly CLI_DIM=""
  readonly CLI_CYAN=""
  readonly CLI_BLUE=""
  readonly CLI_GREEN=""
  readonly CLI_YELLOW=""
  readonly CLI_RED=""
  readonly CLI_INTERACTIVE=false
fi

CLI_LOG_DIR=""
CLI_ACTIVE_PID=""
CLI_CURRENT_STEP=0
CLI_TOTAL_STEPS=0
CLI_TITLE=""
CLI_VERSION=""
CLI_KEEP_LOGS=false

cli_init() {
  CLI_TITLE="$1"
  CLI_VERSION="$2"
  CLI_TOTAL_STEPS="$3"
  CLI_CURRENT_STEP=0

  local log_parent="${LWN_BUILD_LOG_DIR:-${TMPDIR:-/tmp}}"

  mkdir -p -- "$log_parent"
  CLI_LOG_DIR="$(mktemp -d "$log_parent/lwn-simulator-cli.XXXXXX")"
  CLI_KEEP_LOGS="${LWN_KEEP_BUILD_LOGS:-false}"

  trap cli_cleanup EXIT
  trap cli_interrupt INT TERM
}

cli_cleanup() {
  if [[ "$CLI_KEEP_LOGS" != true && -n "${CLI_LOG_DIR:-}" && -d "$CLI_LOG_DIR" ]]; then
    rm -rf -- "$CLI_LOG_DIR"
  fi
}

cli_interrupt() {
  CLI_KEEP_LOGS=true

  if [[ -n "${CLI_ACTIVE_PID:-}" ]]; then
    kill "$CLI_ACTIVE_PID" 2>/dev/null || true
  fi

  printf '\n%s✖ Interrupted%s\n' "$CLI_RED" "$CLI_RESET" >&2
  printf '%sBuild logs: %s%s\n' "$CLI_YELLOW" "$CLI_LOG_DIR" "$CLI_RESET" >&2
  exit 130
}

cli_header() {
  printf '\n%s%s%s\n' "$CLI_BOLD$CLI_CYAN" "$CLI_TITLE" "$CLI_RESET"
  printf '%sVersion %s%s\n\n' "$CLI_DIM" "$CLI_VERSION" "$CLI_RESET"
}

cli_roadmap() {
  printf '%sRoadmap%s\n' "$CLI_BOLD$CLI_BLUE" "$CLI_RESET"

  local index=1
  local step

  for step in "$@"; do
    printf '  %s%02d%s  %s\n' "$CLI_DIM" "$index" "$CLI_RESET" "$step"
    index=$((index + 1))
  done

  printf '\n'
}

run_in_dir() {
  local directory="$1"
  shift

  (cd "$directory" && "$@")
}

run_step() {
  local label="$1"
  shift

  CLI_CURRENT_STEP=$((CLI_CURRENT_STEP + 1))

  local log_file
  local process_status=0
  local frame_index=0
  local process_id
  local frame
  local -a frames=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")

  log_file="$CLI_LOG_DIR/step-${CLI_CURRENT_STEP}.log"

  {
    printf 'Command:'
    printf ' %q' "$@"
    printf '\n'
    printf 'Started: %s\n\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  } >"$log_file"

  "$@" >>"$log_file" 2>&1 &
  process_id=$!
  CLI_ACTIVE_PID="$process_id"

  if [[ "$CLI_INTERACTIVE" == true ]]; then
    while kill -0 "$process_id" 2>/dev/null; do
      frame="${frames[$frame_index]}"
      printf '\r\033[K  %s%s%s %02d/%02d  %s' \
        "$CLI_CYAN" "$frame" "$CLI_RESET" "$CLI_CURRENT_STEP" "$CLI_TOTAL_STEPS" "$label"
      frame_index=$(((frame_index + 1) % ${#frames[@]}))
      sleep 0.08
    done
  else
    printf '  [%02d/%02d] %s...\n' "$CLI_CURRENT_STEP" "$CLI_TOTAL_STEPS" "$label"
  fi

  wait "$process_id" || process_status=$?
  CLI_ACTIVE_PID=""

  if ((process_status == 0)); then
    if [[ "$CLI_INTERACTIVE" == true ]]; then
      printf '\r\033[K  %s✔%s %02d/%02d  %s\n' \
        "$CLI_GREEN" "$CLI_RESET" "$CLI_CURRENT_STEP" "$CLI_TOTAL_STEPS" "$label"
    else
      printf '  [OK] %02d/%02d  %s\n' "$CLI_CURRENT_STEP" "$CLI_TOTAL_STEPS" "$label"
    fi
    return 0
  fi

  printf '\r\033[K  %s✖%s %02d/%02d  %s\n' \
    "$CLI_RED" "$CLI_RESET" "$CLI_CURRENT_STEP" "$CLI_TOTAL_STEPS" "$label" >&2
  printf '%sLast output from failed step:%s\n' "$CLI_YELLOW" "$CLI_RESET" >&2
  tail -n 30 "$log_file" | sed 's/^/    /' >&2
  CLI_KEEP_LOGS=true
  printf '%sFull build log: %s%s\n' "$CLI_YELLOW" "$CLI_LOG_DIR" "$CLI_RESET" >&2

  return "$process_status"
}

cli_artifact() {
  local path="$1"

  if [[ ! -e "$path" ]]; then
    return 0
  fi

  printf '  %s•%s %s (%s)\n' "$CLI_GREEN" "$CLI_RESET" "$path" "$(du -sh -- "$path" | cut -f1)"
}

cli_footer() {
  printf '\n%sCompleted successfully%s\n' "$CLI_BOLD$CLI_GREEN" "$CLI_RESET"
}
