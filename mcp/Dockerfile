# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
# Container builds don't inherit the host proxy; override with a reachable mirror.
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /out/modex-mcp-server ./cmd/modex-mcp-server

FROM alpine:3.20
COPY --from=build /out/modex-mcp-server /usr/local/bin/modex-mcp-server
EXPOSE 8787
CMD ["modex-mcp-server", "--transport", "http", "--addr", ":8787", "--path", "/mcp"]
