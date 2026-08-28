BINARY := idpforge-server
DIST   := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

PLATFORMS := linux/amd64 linux/arm64 windows/amd64 windows/arm64

.PHONY: all build web clean test vet fmt release $(PLATFORMS)

all: build

# web builds the Next.js admin console as a static export and copies it
# into internal/webui/dist, where go:embed picks it up. Run this before any
# `go build` that should include a working admin UI; without it, the
# checked-in placeholder page is served instead.
web:
	cd web && npm ci && npm run build
	rm -rf internal/webui/dist
	mkdir -p internal/webui/dist
	cp -r web/out/. internal/webui/dist/

build: web
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY) ./cmd/server

test:
	CGO_ENABLED=0 go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

release: web $(PLATFORMS)

$(PLATFORMS):
	$(eval OS := $(word 1,$(subst /, ,$@)))
	$(eval ARCH := $(word 2,$(subst /, ,$@)))
	$(eval EXT := $(if $(filter windows,$(OS)),.exe,))
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags "$(LDFLAGS)" \
		-o $(DIST)/$(BINARY)-$(OS)-$(ARCH)$(EXT) ./cmd/server
	tar -C $(DIST) -czf $(DIST)/$(BINARY)-$(OS)-$(ARCH).tar.gz $(BINARY)-$(OS)-$(ARCH)$(EXT)

clean:
	rm -rf $(DIST) internal/webui/dist web/out web/.next
