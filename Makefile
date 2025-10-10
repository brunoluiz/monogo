all: lint test build

.PHONY: lint
lint:
	golangci-lint run --timeout 5m --color always ./...

.PHONY: format
format:
	golangci-lint fmt --enable gofumpt,goimports ./...
	prettier --write .

.PHONY: test
test:
	go test -race ./...

.PHONY: build
build:
	go build -o bin/app ./cmd/monogo
