GO ?= go
BIN := bin/griffin

.PHONY: build test demo fmt

build:
	$(GO) build -o $(BIN) ./cmd/griffin

test:
	$(GO) test ./...

demo: build
	./$(BIN) demo --once

fmt:
	$(GO) fmt ./...
