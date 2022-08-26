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
PROM_OUT_PATH=out/prom
KUBE_PROM_VER=v0.10.0
KUBE_PROM_CFG_FILE=kube-prometheus.jsonnet
PROM_MANIFESTS_PATH=manifests/profiles/prom/

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
lint: golangci-lint lint-copyright lint-markdown

.PHONY: lint-markdown
lint-markdown:
	markdownlint docs --ignore docs/install-components -c common/config/mdl.json
	markdownlint ./README.md -c common/config/mdl.json

lint-copyright:
	@${FINDFILES} \( -name '*.go' -o -name '*.cc' -o -name '*.h' -o -name '*.proto' -o -name '*.py' -o -name '*.sh' \) \( ! \( -name '*.gen.go' -o -name '*.pb.go' -o -name '*_pb2.py' \) \) -print0 |\
		${XARGS} hack/lint_copyright_banner.sh

fix-copyright:
	@${FINDFILES} \( -name '*.go' -o -name '*.cc' -o -name '*.h' -o -name '*.proto' -o -name '*.py' -o -name '*.sh' \) \( ! \( -name '*.gen.go' -o -name '*.pb.go' -o -name '*_pb2.py' \) \) -print0 |\
		${XARGS} hack/fix_copyright_banner.sh

golangci-lint:
	hack/golangci-lint.sh

.PHONY: gen-prom
gen-prom: clean
	rm -rf ${PROM_MANIFESTS_PATH}
	mkdir -p ${PROM_MANIFESTS_PATH}
	mkdir -p ${PROM_OUT_PATH}
	cp manifests/jsonnet/kube-prometheus.jsonnet ${PROM_OUT_PATH}
	hack/gen-prom.sh ${PROM_OUT_PATH} ${KUBE_PROM_VER} ${KUBE_PROM_CFG_FILE}
	cp -r ${PROM_OUT_PATH}/manifests/* ${PROM_MANIFESTS_PATH}

.PHONY: test
test: clean tidy
	go test ./...

.PHONY: clean
clean:
	go clean -testcache
	go clean -cache
	rm -rf $(OUT_PATH)
	rm -rf $(PROM_OUT_PATH)

.PHONY: gen
gen: \
	tidy \
	fix-copyright \
	gen-prom

.PHONY: gen-check
gen-check: gen
	hack/gen-check.sh

.PHONY: serve
serve:
	hack/local-docsite-up.sh
