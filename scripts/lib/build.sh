#!/usr/bin/env bash

set -Eeuo pipefail

ensure_dependencies() {
  run_step "Install and verify workspace dependencies" \
    run_in_dir "$ROOT_DIR" yarn install --immutable
}

build_contracts() {
  run_step "Generate API contracts" run_in_dir "$ROOT_DIR/packages/contracts" yarn build
}

build_components() {
  run_step "Build UI components" run_in_dir "$ROOT_DIR/packages/ui-components" yarn build
}

build_sdk() {
  run_step "Build SDK" run_in_dir "$ROOT_DIR/packages/sdk" yarn build
}

build_frontend() {
  run_step "Build frontend" run_in_dir "$ROOT_DIR/apps/frontend" env VERSION="$VERSION" yarn build
}

lwn_target_os() {
  local target_os="${LWN_TARGET_OS:-}"

  if [[ -z "$target_os" ]]; then
    case "$(uname -s)" in
      Darwin) target_os=darwin ;;
      Linux) target_os=linux ;;
      *)
        printf 'Unable to detect target OS from host: %s\n' "$(uname -s)" >&2
        return 1
        ;;
    esac
  fi

  case "$target_os" in
    linux|darwin)
      printf '%s\n' "$target_os"
      ;;
    *)
      printf 'Unsupported target OS: %s\n' "$target_os" >&2
      return 1
      ;;
  esac
}

lwn_target_arch() {
  if [[ -n "${LWN_TARGET_ARCH:-}" ]]; then
    printf '%s\n' "$LWN_TARGET_ARCH"
    return 0
  fi

  case "$(uname -m)" in
    x86_64) printf 'x64\n' ;;
    arm64|aarch64) printf 'arm64\n' ;;
    *)
      printf 'Unable to detect target architecture from host: %s\n' "$(uname -m)" >&2
      return 1
      ;;
  esac
}

lwn_electron_arch() {
  local target_os
  target_os="$(lwn_target_os)"

  case "$(lwn_target_arch)" in
    x64|arm64)
      lwn_target_arch
      ;;
    armv7l)
      if [[ "$target_os" != linux ]]; then
        printf 'Architecture armv7l is only supported for Linux\n' >&2
        return 1
      fi
      lwn_target_arch
      ;;
    *)
      printf 'Unsupported %s target architecture: %s\n' "$target_os" "$(lwn_target_arch)" >&2
      return 1
      ;;
  esac
}

lwn_go_arch() {
  case "$(lwn_target_arch)" in
    x64)
      printf 'amd64\n'
      ;;
    arm64)
      printf 'arm64\n'
      ;;
    armv7l)
      printf 'arm\n'
      ;;
    *)
      printf 'Unsupported %s target architecture: %s\n' "$(lwn_target_os)" "$(lwn_target_arch)" >&2
      return 1
      ;;
  esac
}

electron_target_files() {
  local output_dir="$1"
  local target="$2"
  local -a artifacts

  shopt -s nullglob

  case "$target" in
    AppImage)
      artifacts=("$output_dir"/*.AppImage)
      ;;
    deb)
      artifacts=("$output_dir"/*.deb)
      ;;
    rpm)
      artifacts=("$output_dir"/*.rpm)
      ;;
    dmg)
      artifacts=("$output_dir"/*.dmg)
      ;;
    zip)
      artifacts=("$output_dir"/*.zip)
      ;;
    *)
      printf 'Unsupported Electron target: %s\n' "$target" >&2
      shopt -u nullglob
      return 1
      ;;
  esac

  if (( ${#artifacts[@]} != 1 )); then
    printf 'Expected exactly one %s artifact in %s, found %d\n' \
      "$target" "$output_dir" "${#artifacts[@]}" >&2
    shopt -u nullglob
    return 1
  fi

  if ! test -s "${artifacts[0]}"; then
    shopt -u nullglob
    return 1
  fi

  shopt -u nullglob
  printf '%s\n' "${artifacts[0]}"
}

publish_electron_artifacts() {
  local output_dir="$1"
  local release_dir="$2"
  local raw_targets="${LWN_ELECTRON_TARGETS:-AppImage}"
  local -a targets
  local target source filename mode

  IFS=',' read -r -a targets <<< "$raw_targets"
  mkdir -p -- "$release_dir"
  shopt -s nullglob

  for target in "${targets[@]}"; do
    target="${target//[[:space:]]/}"
    source="$(electron_target_files "$output_dir" "$target")"
    filename="$(basename "$source")"
    mode=0644
    [[ "$target" == AppImage ]] && mode=0755

    case "$target" in
      AppImage) rm -f -- "$release_dir"/*.AppImage ;;
      deb) rm -f -- "$release_dir"/*.deb ;;
      rpm) rm -f -- "$release_dir"/*.rpm ;;
      dmg) rm -f -- "$release_dir"/*.dmg ;;
      zip) rm -f -- "$release_dir"/*.zip ;;
    esac

    install -m "$mode" -- "$source" "$release_dir/$filename.tmp"
    mv -- "$release_dir/$filename.tmp" "$release_dir/$filename"
  done

  shopt -u nullglob
  rm -rf -- "$output_dir"
}

verify_electron_artifacts() {
  local release_dir="$1"
  local raw_targets="${LWN_ELECTRON_TARGETS:-AppImage}"
  local -a targets
  local target

  IFS=',' read -r -a targets <<< "$raw_targets"
  for target in "${targets[@]}"; do
    target="${target//[[:space:]]/}"
    electron_target_files "$release_dir" "$target" >/dev/null
  done
}

build_backend() {
  local output_path="$1"
  local go_arch
  local -a go_env

  go_arch="$(lwn_go_arch)"
  go_env=(GOOS="$(lwn_target_os)" GOARCH="$go_arch" CGO_ENABLED=0)
  if [[ "$(lwn_target_arch)" == armv7l ]]; then
    go_env+=(GOARM=7)
  fi

  run_step "Build backend ($go_arch)" run_in_dir "$ROOT_DIR/apps/backend" env \
    "${go_env[@]}" \
    go build \
    -ldflags "-X lwn-simulator-backend/version.AppVersion=$VERSION" \
    -o "$output_path" \
    cmd/server/main.go
}
