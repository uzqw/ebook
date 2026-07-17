ARG NODE_IMAGE=node:22-alpine3.21
ARG GOLANG_IMAGE=golang:1.25-bookworm
ARG RUNTIME_IMAGE=ubuntu:24.04

# 1. Build frontend
FROM ${NODE_IMAGE} AS frontend-builder
WORKDIR /src
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm config set registry https://registry.npmmirror.com && npm ci
COPY index.html tsconfig.json tsconfig.app.json tsconfig.node.json vite.config.ts postcss.config.js tailwind.config.js ./
COPY src ./src
RUN npm run build

# 2. Build Go backend
FROM ${GOLANG_IMAGE} AS backend-builder
ENV DEBIAN_FRONTEND=noninteractive
RUN rm -f /etc/apt/apt.conf.d/docker-clean \
  && echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    apt-get update \
    && apt-get install -y --no-install-recommends \
       ca-certificates \
       build-essential
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY backend/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags='-s -w -extldflags "-static"' -o ebook-pocketbase ./cmd/ebook-pocketbase

# 3. Final runtime image
FROM ${RUNTIME_IMAGE} AS runtime
RUN sed -i 's/archive.ubuntu.com/mirrors.ustc.edu.cn/g' /etc/apt/sources.list.d/ubuntu.sources \
  && sed -i 's/security.ubuntu.com/mirrors.ustc.edu.cn/g' /etc/apt/sources.list.d/ubuntu.sources
RUN rm -f /etc/apt/apt.conf.d/docker-clean \
  && echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    apt-get update \
    && apt-get install -y --no-install-recommends \
       ca-certificates \
       curl \
       tzdata \
       fonts-droid-fallback

WORKDIR /app

ENV POCKETBASE_HOST=0.0.0.0 \
    POCKETBASE_PORT=18093 \
    POCKETBASE_DATA_DIR=/app/pb_data \
    POCKETBASE_HOOKS_DIR=/app/pb_hooks \
    PUBLIC_DIR=/app/dist \
    TMPDIR=/app/pb_data/tmp \
    PB_BIN=/usr/local/bin/ebook-pocketbase \
    EPUB_RENDER_FONT=/app/fonts/DroidSansFallback.ttf

COPY --from=frontend-builder /src/dist/ /app/dist/
COPY --from=backend-builder /src/ebook-pocketbase /usr/local/bin/ebook-pocketbase
COPY pb_hooks/ /app/pb_hooks/
COPY fonts/ /app/fonts/
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod 0755 /usr/local/bin/docker-entrypoint.sh \
  && mkdir -p /app/pb_data/tmp \
  && chmod 1777 /app/pb_data/tmp

EXPOSE 18093
VOLUME ["/app/pb_data"]
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["serve"]
