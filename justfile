set positional-arguments := true

# list every recipe
default:
    @just --list

# go build -o bin/duty ./cmd/duty
build:
    go build -o bin/duty ./cmd/duty

# full suite with coverage; extra args pass through, e.g. `just test -run TestClaim`
test *args:
    go test ./tests/... -coverpkg=./internal/... "$@"

# full suite under the race detector
race *args:
    go test ./tests/... -coverpkg=./internal/... -race "$@"

# golangci-lint, pinned via go run
lint *args:
    go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run "$@"

# gofumpt, in place
fmt *args:
    go run mvdan.cc/gofumpt@latest -w . "$@"

# go vet ./...
vet *args:
    go vet ./... "$@"

# build, then run the live TUI on this repo's own tree
tui: build
    ./bin/duty tui

# GoReleaser dry run: builds every target, skips publish
snapshot:
    go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean --skip=publish

# govulncheck ./...
vuln:
    go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# the pre-commit gate: fmt-check + vet + lint + test
check:
    #!/usr/bin/env bash
    set -euo pipefail
    unformatted=$(go run mvdan.cc/gofumpt@latest -l .)
    if [ -n "$unformatted" ]; then
        echo "gofumpt needs to be run on:"
        echo "$unformatted"
        exit 1
    fi
    just vet
    just lint
    just test -count=1

# remove build and release output
clean:
    rm -rf bin dist
