SHELL := /bin/bash

.PHONY: all tidy build test race lint vuln release test-integration test-integration-smoke test-integration-benchmarks

all: tidy build test

tidy:
	go mod tidy

build:
	go build ./...

test:
	go test ./... -count=1 -cover

test-integration:
	go test ./test/integration/... -count=1 -v -timeout=10m

test-integration-smoke:
	go test ./test/integration/... -run="TestIntegrationSmokeTest" -count=1 -v -timeout=5m

test-integration-benchmarks:
	go test ./test/integration/... -run="TestIntegrationBenchmarks" -count=1 -v -timeout=15m

test-integration-regression:
	go test ./test/integration/... -run="TestIntegrationRegression" -count=1 -v -timeout=10m

test-all: test test-integration-smoke

race:
	go test ./... -race -count=1

lint:
	staticcheck ./...

lint-integration:
	staticcheck ./test/integration/...

vuln:
	govulncheck ./...

release:
	goreleaser release --clean
