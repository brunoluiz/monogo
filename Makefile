.PHONY: all lint test build

lint:
	golangci-lint run ./...

test:
	go test -v ./...

build:
	go build -o bin/app ./cmd/monogo

all: lint test build
