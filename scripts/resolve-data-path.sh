#!/usr/bin/env sh
set -eu

fail() {
  printf 'DATA_PATH_ERROR %s\n' "$1" >&2
  exit 2
}

if [ -n "${APP_DATA_ROOT:-}" ]; then
  app_data_root=$APP_DATA_ROOT
else
  [ -n "${HOME:-}" ] || fail 'HOME is required when APP_DATA_ROOT is unset'
  xdg_data_home=${XDG_DATA_HOME:-"$HOME/.local/share"}
  app_data_root="$xdg_data_home/uzqw/apps"
fi

case "$app_data_root" in
  /*) ;;
  *) fail 'APP_DATA_ROOT must be absolute' ;;
esac

command -v realpath >/dev/null 2>&1 || fail 'realpath is required'
canonical_root=$(realpath -m -- "$app_data_root")
script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(realpath -m -- "$script_dir/..")

case "$canonical_root" in
  /) fail 'APP_DATA_ROOT cannot be the filesystem root' ;;
  "$repo_root"|"$repo_root"/*)
    fail 'APP_DATA_ROOT cannot be inside the repository'
    ;;
esac

if [ -n "${HOME:-}" ]; then
  canonical_home=$(realpath -m -- "$HOME")
  [ "$canonical_root" != "$canonical_home" ] || \
    fail 'APP_DATA_ROOT cannot be the home directory'
fi

printf '%s/ebook-reader/pb_data\n' "${canonical_root%/}"
