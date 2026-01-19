.PHONY: build test

build:
	go build ./cmd/ossre

test:
	go test ./...
