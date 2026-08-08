BINARY      := splitdns
PKG         := github.com/soulteary/splitdns
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w \
	-X $(PKG)/cmd.version=$(VERSION) \
	-X $(PKG)/cmd.commit=$(COMMIT) \
	-X $(PKG)/cmd.date=$(DATE)

.PHONY: all build test integration lint fmt fmt-check vet tidy clean install

all: fmt-check vet test build

build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) .

install:
	go install -trimpath -ldflags '$(LDFLAGS)' .

test:
	go test ./...

integration:
	go test -tags integration ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .

fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
	rm -rf dist
