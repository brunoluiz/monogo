.PHONY: all lint test build

lint:
	golangci-lint run ./...

test:
	go test -race ./...

build:
	go build -o bin/app ./cmd/monogo

all: lint test build
