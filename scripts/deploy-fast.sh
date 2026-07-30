#!/usr/bin/env sh
set -eu

project=ebook-reader-uzqw
service=ebook-reader-uzqw

data_path=$(./scripts/resolve-data-path.sh)
builder_cache_path=${DOCKER_BUILDER_CACHE_PATH:-./.local/docker-build-cache}

mkdir -p "$data_path/tmp" "$builder_cache_path"
chmod 0750 "$data_path" "$builder_cache_path"
chmod 1777 "$data_path/tmp"

if ! docker network inspect cicd-observability >/dev/null 2>&1; then
  printf '%s\n' \
    'DEPENDENCY_MISSING network=cicd-observability run=cicd-setup' >&2
  exit 3
fi

DOCKER_PB_DATA_PATH="$data_path" \
  DOCKER_BUILDER_CACHE_PATH="$builder_cache_path" \
  docker compose -p "$project" \
  -f compose.yaml -f compose.build.yaml \
  -f compose.observability.yaml config --quiet

container=$(DOCKER_PB_DATA_PATH="$data_path" \
  DOCKER_BUILDER_CACHE_PATH="$builder_cache_path" \
  docker compose -p "$project" \
  -f compose.yaml -f compose.build.yaml \
  -f compose.observability.yaml ps -q "$service" 2>/dev/null || true)

if [ -n "$container" ] && \
   [ "$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || printf false)" = true ]; then
  stage_dir="$data_path/deploy-staging"
  release_dir="$data_path/deploy"
  backup_dir="$data_path/deploy.backup"
  app_dir="$data_path/deploy/app"
  backend_out="$stage_dir/backend/ebook-pocketbase"

  rm -rf "$stage_dir"
  mkdir -p "$stage_dir/frontend" "$stage_dir/backend"

  printf '%s\n' 'fast-deploy: building frontend and backend'
  npm run build
  mkdir -p .local/go-build-cache
  (
    cd backend
    GOCACHE="$PWD/../.local/go-build-cache" CGO_ENABLED=0 GOOS=linux go build -tags nocgo \
      -o "$backend_out" ./cmd/ebook-pocketbase
  )

  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete --checksum dist/ "$stage_dir/frontend/"
  else
    cp -a dist/. "$stage_dir/frontend/"
  fi
  chmod 0755 "$backend_out"

  printf '%s\n' 'fast-deploy: updating runtime artifacts'
  if [ -d "$release_dir" ]; then
    rm -rf "$backup_dir"
    mv "$release_dir" "$backup_dir"
  fi
  mv "$stage_dir" "$release_dir"
  mkdir -p "$app_dir"
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete --checksum "$release_dir/frontend/" "$app_dir/"
  else
    cp -a "$release_dir/frontend/." "$app_dir/"
  fi

  printf '%s\n' 'fast-deploy: restarting container'
  docker restart "$container" >/dev/null

  printf '%s\n' 'fast-deploy: waiting for health'
  healthy=0
  i=0
  while [ "$i" -lt 30 ]; do
    status=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' \
      "$container" 2>/dev/null || printf unknown)
    if [ "$status" = healthy ]; then
      healthy=1
      break
    fi
    i=$((i + 1))
    sleep 2
  done

  if [ "$healthy" -ne 1 ]; then
    printf '%s\n' 'fast-deploy: container is not healthy after restart' >&2
    exit 4
  fi

  printf '%s\n' 'fast-deploy: OK'
  exit 0
fi

printf '%s\n' 'fast-deploy: container not running; using image build'
DOCKER_PB_DATA_PATH="$data_path" \
  DOCKER_BUILDER_CACHE_PATH="$builder_cache_path" \
  docker compose -p "$project" \
  -f compose.yaml -f compose.build.yaml \
  -f compose.observability.yaml up -d --build
