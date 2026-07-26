#!/usr/bin/env sh
set -eu

mode=${1:-runtime}
case "$mode" in
  runtime|ci) ;;
  *)
    printf 'DOCTOR_ERROR invalid-mode=%s expected=runtime-or-ci\n' "$mode" >&2
    exit 2
    ;;
esac

missing=0
for command_name in task node npm go docker realpath shellcheck; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    printf 'DEPENDENCY_MISSING command=%s\n' "$command_name" >&2
    missing=1
  fi
done
[ "$missing" -eq 0 ] || exit 3

docker compose version >/dev/null 2>&1 || {
  printf '%s\n' 'DEPENDENCY_MISSING command=docker-compose-v2' >&2
  exit 3
}

printf 'OK mode=%s node=%s go=%s\n' \
  "$mode" "$(node --version)" "$(go version | awk '{print $3}')"

if [ "$mode" = ci ]; then
  exit 0
fi

command -v curl >/dev/null 2>&1 || {
  printf '%s\n' 'DEPENDENCY_MISSING command=curl' >&2
  exit 3
}

[ -f .env ] || {
  printf '%s\n' 'DEPENDENCY_MISSING file=.env run=task-setup' >&2
  exit 3
}

docker info >/dev/null 2>&1 || {
  printf '%s\n' 'DEPENDENCY_MISSING service=docker-daemon' >&2
  exit 3
}

docker network inspect cicd-observability >/dev/null 2>&1 || {
  printf '%s\n' 'DEPENDENCY_MISSING network=cicd-observability run=cicd-setup' >&2
  exit 3
}

data_path=$(./scripts/resolve-data-path.sh)
probe=$data_path
while [ ! -e "$probe" ]; do
  parent=$(dirname -- "$probe")
  [ "$parent" != "$probe" ] || break
  probe=$parent
done

[ -d "$probe" ] && [ -w "$probe" ] || {
  printf 'DEPENDENCY_MISSING writable-data-parent=%s\n' "$probe" >&2
  exit 3
}

if [ -e "$data_path" ]; then
  [ -d "$data_path" ] && [ -w "$data_path" ] || {
    printf 'DEPENDENCY_MISSING writable-data-path=%s\n' "$data_path" >&2
    exit 3
  }
fi

printf 'OK network=cicd-observability data=%s\n' "$data_path"
