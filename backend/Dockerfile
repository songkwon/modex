FROM golang:1.22-alpine AS build
WORKDIR /src

# Copy go.mod + go.sum first for better layer caching of dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/modex-api ./cmd/modex-api

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/modex-api /usr/local/bin/modex-api
EXPOSE 8671
CMD ["modex-api"]
