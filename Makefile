VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

MODULE   = github.com/shiroha/subdux/internal/version
LDFLAGS  = -s -w \
           -X $(MODULE).Version=$(VERSION) \
           -X $(MODULE).Commit=$(COMMIT) \
           -X $(MODULE).BuildDate=$(BUILD_DATE)

BINARY   = subdux
GO_FILES = $(shell find . -path './web' -prune -o -name '*.go' -print)

.PHONY: build dev frontend-deps frontend frontend-lint fmt fmt-check vet test check docker clean

build: frontend
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/server

dev:
	@tmux kill-session -t subdux-dev 2>/dev/null || true
	@tmux new-session -d -s subdux-dev \
		'go run -ldflags="$(LDFLAGS)" ./cmd/server' \; \
		split-window -h -t subdux-dev -c '$(CURDIR)/web' \
		'bun run dev' \; \
		attach -t subdux-dev

frontend-deps:
	cd web && bun install

frontend: frontend-deps
	cd web && bun run build

frontend-lint: frontend-deps
	cd web && bun run lint

fmt:
	gofmt -w $(GO_FILES)

fmt-check:
	@files="$$(gofmt -l $(GO_FILES))"; \
	if [ -n "$$files" ]; then \
		echo "Go files are not gofmt-formatted:"; \
		echo "$$files"; \
		exit 1; \
	fi

vet: frontend
	go vet ./...

test: frontend
	go test ./...

check: fmt-check frontend-lint frontend
	go vet ./...
	go test ./...

docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t subdux:$(VERSION) .

clean:
	rm -f $(BINARY)
