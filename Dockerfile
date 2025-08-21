ARG GO_VERSION="1.23"
ARG GOLANGCI_LINT_VERSION="1.61"

# base downloads the necessary Go modules
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS base
WORKDIR /src
RUN --mount=src=go.mod,dst=go.mod \
  --mount=src=go.sum,dst=go.sum \
  --mount=type=cache,target=/go/pkg/mod \
  go mod download

FROM base AS test
RUN --mount=target=. \
  --mount=type=cache,target=/go/pkg/mod \
  go test ./...

FROM golangci/golangci-lint:v${GOLANGCI_LINT_VERSION}-alpine AS lint
ARG XDG_CACHE_HOME=/tmp/cache/
RUN --mount=src=go.mod,dst=go.mod \
  --mount=src=go.sum,dst=go.sum \
  --mount=type=cache,target=/go/pkg/mod \
  go mod download
RUN --mount=target=.,rw \
  --mount=type=cache,target=/tmp/cache \
  golangci-lint -v run ./...
