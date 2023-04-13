VERSION ?= 0.4-dev
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
SOURCES := $(shell find . -type f  -name '*.go')
GIT_COMMIT_HASH ?= $(shell git rev-parse HEAD)
GIT_TREESTATE = "clean"
GIT_DIFF = $(shell git diff --quiet >/dev/null 2>&1; if [ $$? -eq 1 ]; then echo "1"; fi)
ifeq ($(GIT_DIFF), 1)
    GIT_TREESTATE = "dirty"
endif

BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
OUT_BASE_PATH= out
OUT_PATH = $(OUT_BASE_PATH)/$(GOOS)-$(GOARCH)

LDFLAGS := "-X kurator.dev/kurator/pkg/version.gitVersion=$(VERSION) \
			-X kurator.dev/kurator/pkg/version.gitCommit=$(GIT_COMMIT_HASH) \
			-X kurator.dev/kurator/pkg/version.gitTreeState=$(GIT_TREESTATE) \
			-X kurator.dev/kurator/pkg/version.buildDate=$(BUILD_DATE)"
GO_BUILD=CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS)
DOCKER_BUILD=docker build --build-arg BASE_VERSION=nonroot --build-arg BASE_IMAGE=gcr.io/distroless/static

FINDFILES=find . \( -path ./common-protos -o -path ./.git -o -path ./out -o -path ./.github  -o -path ./hack -o -path ./licenses -o -path ./vendor \) -prune -o -type f
XARGS = xargs -0 -r

IMAGE_HUB ?= ghcr.io/kurator-dev
IMAGE_TAG ?= $(VERSION)

HELM_CHART_VERSION ?= $(VERSION)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
	GOBIN=$(shell go env GOPATH)/bin
else
	GOBIN=$(shell go env GOBIN)
endif
export PATH := $(GOBIN):$(PATH)

.PHONY: build
build: tidy kurator cluster-operator fleet-manager

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: kurator
kurator:
	$(GO_BUILD) -o $(OUT_PATH)/kurator cmd/kurator/main.go

.PHONY: cluster-operator
cluster-operator:
	$(GO_BUILD) -o $(OUT_PATH)/cluster-operator cmd/cluster-operator/main.go

.PHONY: fleet-manager
fleet-manager:
	$(GO_BUILD) -o $(OUT_PATH)/fleet-manager cmd/fleet-manager/main.go

.PHONY: docker
docker: docker.cluster-operator docker.fleet-manager

.PHONY: docker.cluster-operator
docker.cluster-operator: cluster-operator
	cp ./cmd/cluster-operator/Dockerfile $(OUT_PATH)/
	cd $(OUT_PATH)/ && $(DOCKER_BUILD) -t ${IMAGE_HUB}/cluster-operator:${IMAGE_TAG} .

.PHONY: docker.fleet-manager
docker.fleet-manager: fleet-manager
	cp ./cmd/fleet-manager/Dockerfile $(OUT_PATH)/
	cd $(OUT_PATH)/ && $(DOCKER_BUILD) -t ${IMAGE_HUB}/fleet-manager:${IMAGE_TAG} .

.PHONY: docker-push
docker-push: docker
	docker push ${IMAGE_HUB}/cluster-operator:${IMAGE_TAG}

.PHONY: lint
lint: golangci-lint lint-copyright lint-markdown lint-shellcheck

.PHONY: lint-markdown
lint-markdown:
	markdownlint docs --ignore docs/content/en/references --ignore docs/node_modules -c common/config/mdl.json
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

.PHONY: sync-crds
sync-crds: gen-crd
	hack/sync-crds.sh

.PHONY: gen-chart
gen-chart: sync-crds
	HELM_CHART_VERSION=$(HELM_CHART_VERSION) IMAGE_TAG=$(IMAGE_TAG) hack/gen-chart.sh

release-chart: gen-chart
	rm -rf $(OUT_BASE_PATH)/charts/*.

.PHONY: test
test: clean tidy
	go test ./...

.PHONY: clean
clean:
	go clean -testcache
	go clean -cache
	rm -rf $(OUT_BASE_PATH)

.PHONY: gen
gen: clean \
	gen-code \
	gen-api-doc \
	tidy \
	fix-copyright \
	gen-thanos \
	gen-prom \
	gen-prom-thanos \
	gen-thanos \
	gen-chart

.PHONY: gen-check
gen-check: gen
	hack/gen-check.sh

.PHONY: doc.serve
doc.serve:
	KURATOR_VERSION=$(VERSION) hack/local-docsite-up.sh

.PHONY: doc.build
doc.build:
	KURATOR_VERSION=$(VERSION) hack/local-docsite-build.sh

PHONY: init-codegen
init-codegen:
	hack/init-codegen.sh

.PHONY: gen-api
gen-api: gen-code gen-crd gen-api-doc

.PHONY: gen-crd
gen-crd: init-codegen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	hack/update-crdgen.sh

.PHONY: gen-code
gen-code: init-codegen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	hack/update-codegen.sh

.PHONY: gen-api-doc
gen-api-doc: ## Generate API documentation
	hack/gen-api-doc.sh

.PHONY: release-artifacts
release-artifacts: ## Release artifacts
release-artifacts: build docker gen-chart
	VERSION=$(VERSION) OUT_BASE_PATH=$(OUT_BASE_PATH) hack/release-artifacts.sh
