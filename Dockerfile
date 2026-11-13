# syntax=docker/dockerfile:1

# --- 1. frontend ---
FROM node:22-alpine AS web
# .git is not in the build context; pass it: docker build --build-arg VERSION=$(git describe --tags --always)
ARG VERSION=dev
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
# imported by the frontend (About / changelog), lives outside web/
COPY CHANGELOG.md /src/CHANGELOG.md
# vite.config.js writes to ../internal/web/dist
RUN mkdir -p /src/internal/web/dist && VERSION=$VERSION npm run build

# --- 2. backend (static, no cgo) ---
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/internal/web/dist ./internal/web/dist
ARG TARGETOS TARGETARCH VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/pigura ./cmd/pigura

# --- 3. runtime ---
FROM alpine:3.22
# Thumbnail tools are optional but recommended: libvips (images), libheif (HEIC), ffmpeg (video).
RUN apk add --no-cache vips-tools libheif-tools ffmpeg ca-certificates tzdata \
 && addgroup -S -g 10001 pigura && adduser -S -u 10001 -G pigura -H -h /data pigura \
 && install -d -o pigura -g pigura /data
COPY --from=build /out/pigura /usr/local/bin/pigura

ENV PIGURA_ADDR=0.0.0.0:8080 \
    PIGURA_DB=/data/pigura.db \
    PIGURA_CACHE=/data/cache \
    PIGURA_SCAN_BASE=/media \
    PIGURA_LIBRARY=/media/photos \
    PIGURA_SCAN_INTERVAL=10m \
    GOMEMLIMIT=150MiB

USER pigura
VOLUME ["/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8080/api/auth/status >/dev/null || exit 1
ENTRYPOINT ["pigura"]
