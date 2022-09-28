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
OUT_BASE_PATH= out
OUT_PATH = $(OUT_BASE_PATH)/$(GOOS)-$(GOARCH)

LDFLAGS := "-X kurator.dev/kurator/pkg/version.gitVersion=$(GIT_VERSION) \
			-X kurator.dev/kurator/pkg/version.gitCommit=$(GIT_COMMIT_HASH) \
			-X kurator.dev/kurator/pkg/version.gitTreeState=$(GIT_TREESTATE) \
			-X kurator.dev/kurator/pkg/version.buildDate=$(BUILD_DATE)"

FINDFILES=find . \( -path ./common-protos -o -path ./.git -o -path ./out -o -path ./.github  -o -path ./hack -o -path ./licenses -o -path ./vendor \) -prune -o -type f
XARGS = xargs -0 -r

.PHONY: build
build: tidy kurator

.PHONY: tidy
tidy:
	go mod tidy -compat=1.17

.PHONY: kurator
kurator: clean
	CGO_ENABLED=0 GOOS=$(GOOS) go build \
		-ldflags $(LDFLAGS) \
		-o $(OUT_PATH)/kurator \
		cmd/kurator/main.go

.PHONY: lint
lint: golangci-lint lint-copyright lint-markdown lint-shellcheck

.PHONY: lint-markdown
lint-markdown:
	markdownlint docs --ignore docs/install-components -c common/config/mdl.json
	markdownlint ./README.md -c common/config/mdl.json

lint-copyright:
	@${FINDFILES} \( -name '*.go' -o -name '*.cc' -o -name '*.h' -o -name '*.proto' -o -name '*.py' -o -name '*.sh' \) \( ! \( -name '*.gen.go' -o -name '*.pb.go' -o -name '*_pb2.py' \) \) -print0 |\
		${XARGS} hack/lint_copyright_banner.sh

# GitHub has shellcheck pre-installed
lint-shellcheck:
	@echo Running Shellcheck linter ...
	@shellcheck hack/*.sh

fix-copyright:
	@${FINDFILES} \( -name '*.go' -o -name '*.cc' -o -name '*.h' -o -name '*.proto' -o -name '*.py' -o -name '*.sh' \) \( ! \( -name '*.gen.go' -o -name '*.pb.go' -o -name '*_pb2.py' \) \) -print0 |\
		${XARGS} hack/fix_copyright_banner.sh

golangci-lint:
	hack/golangci-lint.sh

init-gen:
	hack/init-gen-tools.sh

.PHONY: gen-prom
gen-prom: init-gen
	hack/gen-prom.sh manifests/jsonnet/prometheus/prometheus.jsonnet manifests/profiles/prom/

.PHONY: gen-prom-thanos
gen-prom-thanos: init-gen
	hack/gen-prom.sh manifests/jsonnet/prometheus/thanos.jsonnet manifests/profiles/prom-thanos/

.PHONY: gen-thanos
gen-thanos: init-gen
	hack/gen-thanos.sh

.PHONY: test
test: clean tidy
	go test ./...

.PHONY: clean
clean:
	go clean -testcache
	go clean -cache
	rm -rf $(OUT_BASE_PATH)

.PHONY: gen
gen: \
	tidy \
	fix-copyright \
	gen-thanos \
	gen-prom \
	gen-prom-thanos \
	gen-thanos

.PHONY: gen-check
gen-check: gen
	hack/gen-check.sh

.PHONY: serve
serve:
	hack/local-docsite-up.sh
