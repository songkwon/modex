FROM golang:1.22-alpine AS build
WORKDIR /src
# Container builds don't inherit the host proxy; override with a reachable mirror.
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/docs-mcp-server ./cmd/docs-mcp-server

FROM alpine:3.20
COPY --from=build /out/docs-mcp-server /usr/local/bin/docs-mcp-server
CMD ["docs-mcp-server"]
