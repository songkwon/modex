# docsctl CI image: docsctl binary on top of Node, so doc repos can build
# framework sites (vitepress/vuepress/fumadocs) and push to modex from one image.
#
# Build (from repo root):
#   docker build -t ghcr.io/modex/docsctl:latest -f tools/docsctl/Dockerfile tools/docsctl
#
# Publish to your registry and reference it in deploy/ci/modex-docs.gitlab-ci.yml.

FROM golang:1.22-bookworm AS build
WORKDIR /src
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/docsctl ./cmd/docsctl

FROM node:20-bookworm-slim
# git is handy for repos that pull submodules during build (e.g. rd-doc sparklers)
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/docsctl /usr/local/bin/docsctl
WORKDIR /workspace
ENTRYPOINT []
CMD ["docsctl"]
