#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
resolver="$script_dir/resolve-data-path.sh"
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

assert_equal() {
  expected=$1
  actual=$2
  label=$3
  if [ "$expected" != "$actual" ]; then
    printf 'FAIL %s\nexpected: %s\nactual:   %s\n' \
      "$label" "$expected" "$actual" >&2
    exit 1
  fi
}

assert_rejected() {
  label=$1
  shift
  if "$@" >/dev/null 2>&1; then
    printf 'FAIL %s: unsafe path was accepted\n' "$label" >&2
    exit 1
  fi
}

mkdir -p "$tmp_dir/home" "$tmp_dir/xdg" "$tmp_dir/external/apps"

actual=$(env HOME="$tmp_dir/home" APP_DATA_ROOT="$tmp_dir/external/apps" \
  XDG_DATA_HOME="$tmp_dir/ignored" "$resolver")
assert_equal \
  "$tmp_dir/external/apps/ebook-reader/pb_data" \
  "$actual" \
  'explicit APP_DATA_ROOT'

actual=$(env -u APP_DATA_ROOT HOME="$tmp_dir/home" \
  XDG_DATA_HOME="$tmp_dir/xdg" "$resolver")
assert_equal \
  "$tmp_dir/xdg/uzqw/apps/ebook-reader/pb_data" \
  "$actual" \
  'XDG_DATA_HOME fallback'

actual=$(env -u APP_DATA_ROOT -u XDG_DATA_HOME HOME="$tmp_dir/home" \
  "$resolver")
assert_equal \
  "$tmp_dir/home/.local/share/uzqw/apps/ebook-reader/pb_data" \
  "$actual" \
  'HOME fallback'

assert_rejected 'relative APP_DATA_ROOT' \
  env HOME="$tmp_dir/home" APP_DATA_ROOT=relative/path "$resolver"
assert_rejected 'filesystem root' \
  env HOME="$tmp_dir/home" APP_DATA_ROOT=/ "$resolver"
assert_rejected 'home directory' \
  env HOME="$tmp_dir/home" APP_DATA_ROOT="$tmp_dir/home" "$resolver"
assert_rejected 'repository directory' \
  env HOME="$tmp_dir/home" APP_DATA_ROOT="$script_dir/.." "$resolver"

ln -s "$script_dir/.." "$tmp_dir/repo-link"
assert_rejected 'repository symlink' \
  env HOME="$tmp_dir/home" APP_DATA_ROOT="$tmp_dir/repo-link" "$resolver"

printf 'PASS data path resolution\n'
