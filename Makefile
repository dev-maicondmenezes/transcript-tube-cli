VERSION ?= $(shell tr -d '[:space:]' < VERSION)
CORE_VERSION ?= $(shell tr -d '[:space:]' < CORE_VERSION)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE := github.com/dev-maicondmenezes/transcript-tube-cli
LDFLAGS := -s -w \
	-X $(MODULE)/internal/cli.Version=$(VERSION) \
	-X $(MODULE)/internal/cli.Commit=$(COMMIT) \
	-X $(MODULE)/internal/cli.Date=$(DATE)

.PHONY: setup build test test-race cover fmt fmt-check vet check man man-check changelog release-dry-run install-local clean

setup:
	go mod download

build:
	go build ./...

test:
	go test ./...

test-race:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "arquivos não formatados:"; gofmt -l .; exit 1)

vet:
	go vet ./...

man:
	go run ./cmd/gen-man

man-check: man
	git diff --exit-code -- docs/man docs/completions

check: fmt-check vet build test man-check

changelog:
	git-cliff --unreleased --prepend CHANGELOG.md

release-dry-run:
	goreleaser release --snapshot --clean

install-local:
	go install -ldflags '$(LDFLAGS)' ./cmd/tyot

clean:
	rm -rf dist coverage.out coverage.html
