BINARY := kustomize-fzf
PKG := github.com/TaliaMarine/kustomize-fzf
CMD := ./cmd/kustomize-fzf
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build test vet tidy clean multi-build

all: build

build:
	GOOS=$$(go env GOOS) GOARCH=$$(go env GOARCH) go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(CMD)

# Builds archives for:
# linux  amd64
# linux  arm64
# darwin amd64
# darwin arm64
# windows amd64 (.exe)
# Output archive names: kustomize-fzf_<version>_<os>_<arch>.tar.gz
multi-build:
	@mkdir -p dist
	@for GOOS in linux darwin windows; do \
	  for GOARCH in amd64 arm64; do \
	    if [ "$$GOOS" = "windows" ] && [ "$$GOARCH" != "amd64" ]; then continue; fi; \
	    echo "Building $$GOOS/$$GOARCH"; \
	    EXT=""; if [ "$$GOOS" = "windows" ]; then EXT=".exe"; fi; \
	    OUT_BASENAME=$(BINARY)_$(VERSION)_$$GOOS_$$GOARCH; \
	    OUT=dist/$$OUT_BASENAME$$EXT; \
	    GOOS=$$GOOS GOARCH=$$GOARCH CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $$OUT $(CMD); \
	    tar -czf dist/$$OUT_BASENAME.tar.gz -C dist $$(basename $$OUT); \
	    rm $$OUT; \
	  done; \
	done

clean:
	rm -rf bin dist

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy
