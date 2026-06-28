# syntax=docker/dockerfile:1

# ---- build stage ----
# Alpine + build-base gives the C toolchain that mattn/go-sqlite3 (cgo) needs.
FROM golang:1.23-alpine AS build

RUN apk add --no-cache build-base git
WORKDIR /src

# Cache deps first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# cgo is required for the SQLite store; musl-linked binary runs on the alpine
# runtime stage below.
ENV CGO_ENABLED=1 GOOS=linux
RUN go build -ldflags "-s -w" -o /out/goct ./cmd/goct

# ---- runtime stage ----
FROM alpine:3.20

# ca-certificates: needed for HTTPS to CT logs and the Telegram API.
RUN apk add --no-cache ca-certificates \
 && adduser -D -u 10001 goct

WORKDIR /app
COPY --from=build /out/goct /usr/local/bin/goct

# Bake in a default config. Override at runtime by mounting your own file over
# /app/config.yaml (e.g. -v $PWD/config.yaml:/app/config.yaml:ro).
COPY config.yaml /app/config.yaml

# Writable dir for the SQLite db / state (mount a volume here in production).
RUN mkdir -p /app/data && chown -R goct:goct /app
USER goct

# By default goct runs all checks once and exits — the one-shot / cloud-function
# / cron-job model. Mount config.yaml at /app/config.yaml and pass
# TELEGRAM_APITOKEN via -e.
#
# To run as a long-lived daemon instead, override the command and publish the
# healthcheck port (the daemon serves /ping on :8081), e.g.:
#   docker run -p 8081:8081 goct daemon --config /app/config.yaml
EXPOSE 8081
ENTRYPOINT ["goct"]
CMD ["--config", "/app/config.yaml"]
