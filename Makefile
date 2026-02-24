VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

MODULE   = github.com/shiroha/subdux/internal/version
LDFLAGS  = -s -w \
           -X $(MODULE).Version=$(VERSION) \
           -X $(MODULE).Commit=$(COMMIT) \
           -X $(MODULE).BuildDate=$(BUILD_DATE)

BINARY   = subdux

.PHONY: build dev frontend docker clean

build: frontend
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/server

dev:
	@tmux new-session -d -s subdux-dev \
		'go run -ldflags="$(LDFLAGS)" ./cmd/server' \; \
		split-window -h -c web \
		'bun dev' \; \
		attach

frontend:
	cd web && bun install && bun run build

docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t subdux:$(VERSION) .

clean:
	rm -f $(BINARY)
