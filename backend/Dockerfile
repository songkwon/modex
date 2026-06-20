# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /src

# IMPORTANT: On your development machine (with good internet), run ONCE:
#   cd backend && go mod tidy
# This populates go.sum with checksums for all deps (including minio-go/v7 for real MinIO uploads of site files).
# Commit the updated go.sum. Then Docker builds will be reliable.

# Optional: override the proxy at build time, e.g. outside China:
# docker build --build-arg GOPROXY=https://proxy.golang.org,direct ...
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

# Copy go.mod + go.sum first for better layer caching of dependencies
COPY go.mod go.sum ./

# Use BuildKit cache mount (requires Docker BuildKit or docker compose build).
# This makes "go mod download" fast on rebuilds and prevents appearing "卡住" on slow networks.
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source and build (static binary for alpine)
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/modex-api ./cmd/modex-api

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/modex-api /usr/local/bin/modex-api
EXPOSE 8671
CMD ["modex-api"]
