# goct — build & dev tasks
# Note: the SQLite store (mattn/go-sqlite3) needs cgo, so builds default to
# CGO_ENABLED=1. Building with CGO_ENABLED=0 compiles, but the store fails at
# runtime (go-sqlite3 falls back to a stub).

BINARY      := goct
PKG         := ./cmd/goct          # primary CLI / daemon binary
CLOUDFN_PKG := .                   # root package main: cloud-function entrypoint
BIN_DIR     := bin

CGO_ENABLED ?= 1
GOFLAGS     ?=
LDFLAGS     ?= -s -w
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
CONFIG      ?= config.yaml

# Docker
IMAGE       ?= goct
TAG         ?= $(VERSION)

export CGO_ENABLED

.DEFAULT_GOAL := help

## build: compile the CLI/daemon binary into bin/goct
.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(PKG)

## build-cloudfn: compile the cloud-function entrypoint (root package) into bin/goct-cloudfn
.PHONY: build-cloudfn
build-cloudfn:
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-cloudfn $(CLOUDFN_PKG)

## run: build and run ONCE (one-shot, exits) — the default/cloud-function mode; CONFIG=... ARGS=...
.PHONY: run
run: build
	$(BIN_DIR)/$(BINARY) --config $(CONFIG) $(ARGS)

## run-daemon: build and run as a long-lived daemon (healthcheck on :8081)
.PHONY: run-daemon
run-daemon: build
	$(BIN_DIR)/$(BINARY) daemon --config $(CONFIG) $(ARGS)

## test: run the test suite
.PHONY: test
test:
	go test ./... $(GOFLAGS)

## vet: run go vet
.PHONY: vet
vet:
	go vet ./...

## fmt: format all Go sources
.PHONY: fmt
fmt:
	gofmt -s -w .

## tidy: sync go.mod / go.sum
.PHONY: tidy
tidy:
	go mod tidy

## check: fmt-check, vet and test (CI-style gate)
.PHONY: check
check: vet test
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)

## docker-build: build the container image (IMAGE:TAG)
.PHONY: docker-build
docker-build:
	docker build -t $(IMAGE):$(TAG) -t $(IMAGE):latest .

## docker-run: run the image ONCE (one-shot, exits); mount ./config.yaml and pass TELEGRAM_APITOKEN
.PHONY: docker-run
docker-run:
	docker run --rm \
		-e TELEGRAM_APITOKEN=$(TELEGRAM_APITOKEN) \
		-v $(PWD)/$(CONFIG):/app/config.yaml:ro \
		$(IMAGE):$(TAG)

## docker-run-daemon: run the image as a daemon (publishes healthcheck port 8081)
.PHONY: docker-run-daemon
docker-run-daemon:
	docker run --rm -p 8081:8081 \
		-e TELEGRAM_APITOKEN=$(TELEGRAM_APITOKEN) \
		-v $(PWD)/$(CONFIG):/app/config.yaml:ro \
		$(IMAGE):$(TAG) daemon --config /app/config.yaml

## clean: remove build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

## help: list available targets
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | sort
