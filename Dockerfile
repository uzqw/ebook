ARG NODE_IMAGE=node:22-alpine3.21
ARG GOLANG_IMAGE=golang:1.25-bookworm
ARG RUNTIME_IMAGE=ubuntu:24.04

# Shared builder inputs and static assets. This stage usually stays cached for
# frontend-only or backend-only changes, so the runtime can reuse its layers.
FROM ${RUNTIME_IMAGE} AS assets
RUN sed -i 's/archive.ubuntu.com/mirrors.ustc.edu.cn/g' /etc/apt/sources.list.d/ubuntu.sources \
  && sed -i 's/security.ubuntu.com/mirrors.ustc.edu.cn/g' /etc/apt/sources.list.d/ubuntu.sources
RUN rm -f /etc/apt/apt.conf.d/docker-clean \
  && echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache
RUN --mount=type=cache,id=ebook-reader-apt-cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,id=ebook-reader-apt-lists,target=/var/lib/apt/lists,sharing=locked \
    apt-get update \
    && apt-get install -y --no-install-recommends \
       ca-certificates \
       curl \
       tzdata \
       fonts-droid-fallback

WORKDIR /app
COPY fonts/ /app/fonts/
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod 0755 /usr/local/bin/docker-entrypoint.sh \
  && mkdir -p /app/pb_data/tmp \
  && chmod 1777 /app/pb_data/tmp

# 1. Build frontend
FROM ${NODE_IMAGE} AS frontend-builder
ARG CACHE_ROOT=/pi-build-cache
ARG FRONTEND_CACHE_ID=ebook-reader-uzqw-frontend
WORKDIR /src
COPY package.json package-lock.json ./
RUN --mount=type=cache,id=${FRONTEND_CACHE_ID}-npm,target=${CACHE_ROOT}/frontend/npm,sharing=locked \
    npm config set registry https://registry.npmmirror.com && npm ci
COPY index.html tsconfig.json tsconfig.app.json tsconfig.node.json vite.config.ts postcss.config.js tailwind.config.js ./
COPY src ./src
RUN npm run build

# 2. Build Go backend
FROM ${GOLANG_IMAGE} AS backend-builder
ARG CACHE_ROOT=/pi-build-cache
ARG BACKEND_CACHE_ID=ebook-reader-uzqw-backend
ENV DEBIAN_FRONTEND=noninteractive
RUN rm -f /etc/apt/apt.conf.d/docker-clean \
  && echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache
RUN --mount=type=cache,id=ebook-reader-backend-apt-cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,id=ebook-reader-backend-apt-lists,target=/var/lib/apt/lists,sharing=locked \
    apt-get update \
    && apt-get install -y --no-install-recommends \
       ca-certificates \
       build-essential
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,id=${BACKEND_CACHE_ID}-gomod,target=${CACHE_ROOT}/backend/gomod,sharing=locked \
    go mod download
COPY backend/ ./
RUN --mount=type=cache,id=${BACKEND_CACHE_ID}-gomod,target=${CACHE_ROOT}/backend/gomod,sharing=locked \
    --mount=type=cache,id=${BACKEND_CACHE_ID}-gobuild,target=${CACHE_ROOT}/backend/gobuild,sharing=locked \
    CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags='-s -w -extldflags "-static"' -o ebook-pocketbase ./cmd/ebook-pocketbase

# 3. Final runtime image
FROM ${RUNTIME_IMAGE} AS runtime
COPY --from=assets / /
ENV POCKETBASE_HOST=0.0.0.0 \
    POCKETBASE_PORT=18093 \
    POCKETBASE_DATA_DIR=/app/pb_data \
    PUBLIC_DIR=/app/dist \
    TMPDIR=/app/pb_data/tmp \
    PB_BIN=/usr/local/bin/ebook-pocketbase \
    EPUB_RENDER_FONT=/app/fonts/DroidSansFallback.ttf

WORKDIR /app
COPY --from=frontend-builder /src/dist/ /app/dist/
COPY --from=backend-builder /src/ebook-pocketbase /usr/local/bin/ebook-pocketbase

EXPOSE 18093
VOLUME ["/app/pb_data"]
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["serve"]
