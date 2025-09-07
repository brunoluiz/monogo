.PHONY: all ci lint test build

ci: lint test build
all: lint test build

lint:
	golangci-lint run --timeout 5m --color always ./...

test:
	go test -race ./...

build:
	go build -o bin/app ./cmd/monogo
