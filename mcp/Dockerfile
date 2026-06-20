FROM golang:1.23-alpine AS build
WORKDIR /src
# Container builds don't inherit the host proxy; override with a reachable mirror.
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/modex-mcp-server ./cmd/modex-mcp-server

FROM alpine:3.20
COPY --from=build /out/modex-mcp-server /usr/local/bin/modex-mcp-server
CMD ["modex-mcp-server"]
