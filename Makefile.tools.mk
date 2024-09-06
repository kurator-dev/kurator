# Go tools directory holds the binaries of Go-based tools.
GO           := $(shell which go)
TOOLS_DIR    ?= $(PWD)/.tools

# Catch all rules for Go-based tools.
$(GO_TOOLS_DIR)/%:
	@GOBIN=$(TOOLS_DIR) $(GO) install $($(notdir $@)@v)

# Go-based tools.
golangci-lint 	:= $(TOOLS_DIR)/golangci-lint
client-gen    	:= $(TOOLS_DIR)/client-gen
informer-gen  	:= $(TOOLS_DIR)/informer-gen
lister-gen    	:= $(TOOLS_DIR)/lister-gen
register-gen  	:= $(TOOLS_DIR)/register-gen
deepcopy-gen  	:= $(TOOLS_DIR)/deepcopy-gen
controller-gen	:= $(TOOLS_DIR)/controller-gen
kustomize       := $(TOOLS_DIR)/kustomize
jb				:= $(TOOLS_DIR)/jb
gojsontoyaml    := $(TOOLS_DIR)/gojsontoyaml
jsonnet         := $(TOOLS_DIR)/jsonnet

gen-crd-api-reference-docs := $(TOOLS_DIR)/gen-crd-api-reference-docs

golangci-lint@v 	:= github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.2
client-gen@v    	:= k8s.io/code-generator/cmd/client-gen@v0.25.2
informer-gen@v  	:= k8s.io/code-generator/cmd/informer-gen@v0.25.2
lister-gen@v    	:= k8s.io/code-generator/cmd/lister-gen@v0.25.2
register-gen@v  	:= k8s.io/code-generator/cmd/register-gen@v0.25.2
deepcopy-gen@v  	:= k8s.io/code-generator/cmd/deepcopy-gen@v0.25.2
controller-gen@v 	:= sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
kustomize@v       	:= sigs.k8s.io/kustomize/kustomize/v4@v4.5.5
jb@v                := github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@v0.6.0
gojsontoyaml@v      := github.com/brancz/gojsontoyaml@v0.1.0
jsonnet@v           := github.com/google/go-jsonnet/cmd/jsonnet@v0.18.0

gen-crd-api-reference-docs@v := github.com/ahmetb/gen-crd-api-reference-docs@45bac9a # 2023-03-28
