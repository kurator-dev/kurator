GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
SOURCES := $(shell find . -type f  -name '*.go')

GIT_VERSION ?= $(shell git describe --tags --dirty --always)
GIT_COMMIT_HASH ?= $(shell git rev-parse HEAD)
GIT_TREESTATE = "clean"
GIT_DIFF = $(shell git diff --quiet >/dev/null 2>&1; if [ $$? -eq 1 ]; then echo "1"; fi)
ifeq ($(GIT_DIFF), 1)
    GIT_TREESTATE = "dirty"
endif

BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
OUT_PATH = out/$(GOOS)-$(GOARCH)

LDFLAGS := "-X kurator.dev/kurator/pkg/version.gitVersion=$(GIT_VERSION) \
			-X kurator.dev/kurator/pkg/version.gitCommit=$(GIT_COMMIT_HASH) \
			-X kurator.dev/kurator/pkg/version.gitTreeState=$(GIT_TREESTATE) \
			-X kurator.dev/kurator/pkg/version.buildDate=$(BUILD_DATE)"

.PHONY: build
build: kurator

.PHONY: kurator
kurator: clean
	CGO_ENABLED=0 GOOS=$(GOOS) go build \
		-ldflags $(LDFLAGS) \
		-o $(OUT_PATH)/kurator \
		cmd/kurator/main.go

.PHONY: test
test: 
	go test ./...

.PHONY: clean
clean:
	rm -rf $(OUT_PATH)