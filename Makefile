.PHONY: fmt test run smoke build-benchctl

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

build-benchctl:
	@mkdir -p $(GOCACHE_DIR) $(GOMODCACHE_DIR) bin
	@$(GO_ENV) go build -o bin/benchctl ./cmd/benchctl

smoke:
	@bash ./scripts/smoke_test.sh
