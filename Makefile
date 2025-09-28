SHELL := /bin/bash

.PHONY: all tidy build test race lint vuln release

all: tidy build test

tidy:
	go mod tidy

build:
	go build ./...

test:
	go test ./... -count=1 -cover

race:
	go test ./... -race -count=1

lint:
	staticcheck ./...

vuln:
	govulncheck ./...

release:
	goreleaser release --clean
