.PHONY: build test run-api
build:
	go build -o bin/api-server ./cmd/api-server
	go build -o bin/scheduler ./cmd/scheduler
	go build -o bin/worker ./cmd/worker
	go build -o bin/notifier ./cmd/notifier

test:
	go test ./...

run-api:
	go run ./cmd/api-server
