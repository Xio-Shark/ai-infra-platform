.PHONY: fmt test run run-gateway smoke build-benchctl build-gateway

GOCACHE_DIR := $(CURDIR)/.tmp/go-cache
GOMODCACHE_DIR := $(CURDIR)/.tmp/go-mod-cache
GO_ENV := GOCACHE=$(GOCACHE_DIR) GOMODCACHE=$(GOMODCACHE_DIR)

fmt:
	@mkdir -p $(GOCACHE_DIR) $(GOMODCACHE_DIR)
	@gofmt -w $$(find . -name '*.go' -not -path './bin/*' -not -path './.tmp/*')

test:
	@mkdir -p $(GOCACHE_DIR) $(GOMODCACHE_DIR)
	@$(GO_ENV) go test -race ./...

run:
	@mkdir -p $(GOCACHE_DIR) $(GOMODCACHE_DIR)
	@$(GO_ENV) go run ./cmd/api-server

run-gateway:
	@mkdir -p $(GOCACHE_DIR) $(GOMODCACHE_DIR)
	@$(GO_ENV) go run ./cmd/gateway

build-benchctl:
	@mkdir -p $(GOCACHE_DIR) $(GOMODCACHE_DIR) bin
	@$(GO_ENV) go build -o bin/benchctl ./cmd/benchctl

build-gateway:
	@mkdir -p $(GOCACHE_DIR) $(GOMODCACHE_DIR) bin
	@$(GO_ENV) go build -o bin/gateway ./cmd/gateway

smoke:
	@bash ./scripts/smoke_test.sh
