FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/modex-api ./cmd/modex-api

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/modex-api /usr/local/bin/modex-api
EXPOSE 8080
CMD ["modex-api"]
