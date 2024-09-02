VERSION ?= 1.0-dev
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

FINDFILES=find . \( -path ./common-protos -o -path ./.git -o -path ./out -o -path ./.github  -o -path ./hack -o -path ./licenses -o -path ./vendor -o -path ./.gopath \) -prune -o -type f
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

include Makefile.tools.mk

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
	docker push ${IMAGE_HUB}/fleet-manager:${IMAGE_TAG}

.PHONY: sign-image 
sign-image:
	./hack/image-sign.sh

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

mod-download-go:
	@-GOFLAGS="-mod=readonly" find -name go.mod -execdir go mod download \;
# go mod tidy is needed with Golang 1.16+ as go mod download affects go.sum
# https://github.com/golang/go/issues/43994
# exclude docs folder
	@find . -path ./docs -prune -o -name go.mod -execdir go mod tidy \;

.PHONY: mirror-licenses
mirror-licenses: mod-download-go; \
	go install istio.io/tools/cmd/license-lint@v0.0.0-20240221165422-57f6bfb4cd73;  \
	rm -fr licenses; \
	license-lint --mirror

.PHONY: lint-licenses
lint-licenses:
	@if test -d licenses; then license-lint --config common/config/license-lint.yaml; fi

.PHONY: licenses-check
licenses-check: mirror-licenses; \
    hack/licenses-check.sh

fix-copyright:
	@${FINDFILES} \( -name '*.go' -o -name '*.cc' -o -name '*.h' -o -name '*.proto' -o -name '*.py' -o -name '*.sh' \) \( ! \( -name '*.gen.go' -o -name '*.pb.go' -o -name '*_pb2.py' \) \) -print0 |\
		${XARGS} hack/fix_copyright_banner.sh

.PHONY: golangci-lint
golangci-lint: $(golangci-lint) ## Run golangci-lint
	hack/golangci-lint.sh

.PHONY: init-gen-tools
init-gen-tools: $(jb) $(gojsontoyaml) $(jsonnet)

.PHONY: gen-prom
gen-prom: init-gen-tools
	hack/gen-prom.sh manifests/jsonnet/prometheus/prometheus.jsonnet manifests/profiles/prom/

.PHONY: gen-prom-thanos
gen-prom-thanos: init-gen-tools
	hack/gen-prom.sh manifests/jsonnet/prometheus/thanos.jsonnet manifests/profiles/prom-thanos/

.PHONY: gen-thanos
gen-thanos: init-gen-tools
	hack/gen-thanos.sh

.PHONY: sync-crds
sync-crds: gen-crd
	hack/sync-crds.sh

.PHONY: gen-chart
gen-chart: sync-crds
	HELM_CHART_VERSION=$(HELM_CHART_VERSION) IMAGE_TAG=$(IMAGE_TAG) hack/gen-chart.sh

.PHONY: test
test: clean tidy
	go test ./pkg/...
	go test ./cmd/...

.PHONY: clean
clean:
	go clean -testcache
	go clean -cache
	@rm -rf $(OUT_BASE_PATH)
	@rm -rf .tools
	@rm -rf .gopath

.PHONY: gen
gen: clean \
	gen-code \
	gen-api-doc \
	tidy \
	fix-copyright \
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
init-codegen: $(kustomize) \
				$(deepcopy-gen) \
				$(client-gen) \
				$(lister-gen) \
				$(informer-gen) \
				$(register-gen) \
 			    $(controller-gen) ## Install code generation tools

.PHONY: gen-api
gen-api: gen-code gen-crd gen-api-doc

.PHONY: gen-crd
gen-crd: init-codegen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	hack/update-crdgen.sh


PACKAGE					    := kurator.dev/kurator
GOPATH_SHIM                 := ${PWD}/.gopath
PACKAGE_SHIM                := $(GOPATH_SHIM)/src/$(PACKAGE)

$(GOPATH_SHIM):
	@echo Create gopath shim... >&2
	@mkdir -p $(GOPATH_SHIM)

# learn from kyverno/kyverno project, this will allow you run client-gen everywhere without put project into GOPATH
# DO NOT REMOVE THIS `.INTERMEDIATE`
.INTERMEDIATE: $(PACKAGE_SHIM)
$(PACKAGE_SHIM): $(GOPATH_SHIM)
	@echo Create package shim... >&2
	@mkdir -p $(GOPATH_SHIM)/src/kurator.dev && ln -s -f ${PWD} $(PACKAGE_SHIM)

.PHONY: gen-code
gen-code: $(PACKAGE_SHIM) init-codegen gen-code-clean ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	hack/update-codegen.sh

.PHONY: gen-code-clean
gen-code-clean: ## Clean up generated files
	@echo "Cleaning up generated files..."
	@find pkg/apis -type f -name zz_generated* | xargs rm
	@find pkg/client-go -type f -name *.go | xargs rm

.PHONY: gen-api-doc
gen-api-doc: $(gen-crd-api-reference-docs) ## Generate API documentation
	hack/gen-api-doc.sh

.PHONY: release-artifacts
release-artifacts: ## Release artifacts
release-artifacts: build docker gen-chart
	VERSION=$(VERSION) OUT_BASE_PATH=$(OUT_BASE_PATH) hack/release-artifacts.sh
