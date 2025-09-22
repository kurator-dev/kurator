/*
Copyright 2022-2025 Kurator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package render

const GitCloneTaskContent = `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: {{ .PredefinedTaskName }}
  namespace: {{ .Namespace }}
  labels:
    app.kubernetes.io/version: "0.9"
  annotations:
    tekton.dev/pipelines.minVersion: "0.38.0"
    tekton.dev/categories: Git
    tekton.dev/tags: git
    tekton.dev/displayName: "git clone"
    tekton.dev/platforms: "linux/amd64,linux/s390x,linux/ppc64le,linux/arm64"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
spec:
  description: >-
    These Tasks are Git tasks to work with repositories used by other tasks
    in your Pipeline.
    The git-clone Task will clone a repo from the provided url into the
    source Workspace. By default the repo will be cloned into the root of
    your Workspace. You can clone into a subdirectory by setting this Task's
    subdirectory param. This Task also supports sparse checkouts. To perform
    a sparse checkout, pass a list of comma separated directory patterns to
    this Task's sparseCheckoutDirectories param.
  workspaces:
  - name: source
    description: The git repo will be cloned onto the volume backing this Workspace.
  - name: ssh-directory
    optional: true
    description: |
      A .ssh directory with private key, known_hosts, config, etc. Copied to
      the user's home before git commands are executed. Used to authenticate
      with the git remote when performing the clone. Binding a Secret to this
      Workspace is strongly recommended over other volume types.
  - name: basic-auth
    optional: true
    description: |
      A Workspace containing a .gitconfig and .git-credentials file. These
      will be copied to the user's home before any git commands are run. Any
      other files in this Workspace are ignored. It is strongly recommended
      to use ssh-directory over basic-auth whenever possible and to bind a
      Secret to this Workspace over other volume types.
  - name: ssl-ca-directory
    optional: true
    description: |
      A workspace containing CA certificates, this will be used by Git to
      verify the peer with when fetching or pushing over HTTPS.
  params:
  - name: url
    description: Repository URL to clone from.
    type: string
  - name: revision
    description: Revision to checkout. (branch, tag, sha, ref, etc...)
    type: string
    default: ""
  - name: refspec
    description: Refspec to fetch before checking out revision.
    default: ""
  - name: submodules
    description: Initialize and fetch git submodules.
    type: string
    default: "true"
  - name: depth
    description: Perform a shallow clone, fetching only the most recent N commits.
    type: string
    default: "1"
  - name: sslVerify
    description: Set the "http.sslVerify" global git config. Setting this to "false" is not advised unless you are sure that you trust your git remote.
    type: string
    default: "true"
  - name: crtFileName
    description: file name of mounted crt using ssl-ca-directory workspace. default value is ca-bundle.crt.
    type: string
    default: "ca-bundle.crt"
  - name: subdirectory
    description: Subdirectory inside the "source" Workspace to clone the repo into.
    type: string
    default: ""
  - name: sparseCheckoutDirectories
    description: Define the directory patterns to match or exclude when performing a sparse checkout.
    type: string
    default: ""
  - name: deleteExisting
    description: Clean out the contents of the destination directory if it already exists before cloning.
    type: string
    default: "true"
  - name: httpProxy
    description: HTTP proxy server for non-SSL requests.
    type: string
    default: ""
  - name: httpsProxy
    description: HTTPS proxy server for SSL requests.
    type: string
    default: ""
  - name: noProxy
    description: Opt out of proxying HTTP/HTTPS requests.
    type: string
    default: ""
  - name: verbose
    description: Log the commands that are executed during "git-clone"'s operation.
    type: string
    default: "true"
  - name: gitInitImage
    description: The image providing the git-init binary that this Task runs.
    type: string
    default: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.40.2"
  - name: userHome
    description: |
      Absolute path to the user's home directory.
    type: string
    default: "/home/git"
  results:
  - name: commit
    description: The precise commit SHA that was fetched by this Task.
  - name: url
    description: The precise URL that was fetched by this Task.
  - name: committer-date
    description: The epoch timestamp of the commit that was fetched by this Task.
  steps:
  - name: clone
    image: "$(params.gitInitImage)"
    env:
    - name: HOME
      value: "$(params.userHome)"
    - name: PARAM_URL
      value: $(params.url)
    - name: PARAM_REVISION
      value: $(params.revision)
    - name: PARAM_REFSPEC
      value: $(params.refspec)
    - name: PARAM_SUBMODULES
      value: $(params.submodules)
    - name: PARAM_DEPTH
      value: $(params.depth)
    - name: PARAM_SSL_VERIFY
      value: $(params.sslVerify)
    - name: PARAM_CRT_FILENAME
      value: $(params.crtFileName)
    - name: PARAM_SUBDIRECTORY
      value: $(params.subdirectory)
    - name: PARAM_DELETE_EXISTING
      value: $(params.deleteExisting)
    - name: PARAM_HTTP_PROXY
      value: $(params.httpProxy)
    - name: PARAM_HTTPS_PROXY
      value: $(params.httpsProxy)
    - name: PARAM_NO_PROXY
      value: $(params.noProxy)
    - name: PARAM_VERBOSE
      value: $(params.verbose)
    - name: PARAM_SPARSE_CHECKOUT_DIRECTORIES
      value: $(params.sparseCheckoutDirectories)
    - name: PARAM_USER_HOME
      value: $(params.userHome)
    - name: WORKSPACE_SOURCE_PATH
      value: $(workspaces.source.path)
    - name: WORKSPACE_SSH_DIRECTORY_BOUND
      value: $(workspaces.ssh-directory.bound)
    - name: WORKSPACE_SSH_DIRECTORY_PATH
      value: $(workspaces.ssh-directory.path)
    - name: WORKSPACE_BASIC_AUTH_DIRECTORY_BOUND
      value: $(workspaces.basic-auth.bound)
    - name: WORKSPACE_BASIC_AUTH_DIRECTORY_PATH
      value: $(workspaces.basic-auth.path)
    - name: WORKSPACE_SSL_CA_DIRECTORY_BOUND
      value: $(workspaces.ssl-ca-directory.bound)
    - name: WORKSPACE_SSL_CA_DIRECTORY_PATH
      value: $(workspaces.ssl-ca-directory.path)
    securityContext:
      runAsNonRoot: true
      runAsUser: 65532
    script: |
      #!/usr/bin/env sh
      set -eu
      if [ "${PARAM_VERBOSE}" = "true" ] ; then
        set -x
      fi
      if [ "${WORKSPACE_BASIC_AUTH_DIRECTORY_BOUND}" = "true" ] ; then
        cp "${WORKSPACE_BASIC_AUTH_DIRECTORY_PATH}/.git-credentials" "${PARAM_USER_HOME}/.git-credentials"
        cp "${WORKSPACE_BASIC_AUTH_DIRECTORY_PATH}/.gitconfig" "${PARAM_USER_HOME}/.gitconfig"
        chmod 400 "${PARAM_USER_HOME}/.git-credentials"
        chmod 400 "${PARAM_USER_HOME}/.gitconfig"
      fi
      if [ "${WORKSPACE_SSH_DIRECTORY_BOUND}" = "true" ] ; then
        cp -R "${WORKSPACE_SSH_DIRECTORY_PATH}" "${PARAM_USER_HOME}"/.ssh
        chmod 700 "${PARAM_USER_HOME}"/.ssh
        chmod -R 400 "${PARAM_USER_HOME}"/.ssh/*
      fi
      if [ "${WORKSPACE_SSL_CA_DIRECTORY_BOUND}" = "true" ] ; then
         export GIT_SSL_CAPATH="${WORKSPACE_SSL_CA_DIRECTORY_PATH}"
         if [ "${PARAM_CRT_FILENAME}" != "" ] ; then
            export GIT_SSL_CAINFO="${WORKSPACE_SSL_CA_DIRECTORY_PATH}/${PARAM_CRT_FILENAME}"
         fi
      fi
      CHECKOUT_DIR="${WORKSPACE_SOURCE_PATH}/${PARAM_SUBDIRECTORY}"
      cleandir() {
        # Delete any existing contents of the repo directory if it exists.
        #
        # We don't just "rm -rf ${CHECKOUT_DIR}" because ${CHECKOUT_DIR} might be "/"
        # or the root of a mounted volume.
        if [ -d "${CHECKOUT_DIR}" ] ; then
          # Delete non-hidden files and directories
          rm -rf "${CHECKOUT_DIR:?}"/*
          # Delete files and directories starting with . but excluding ..
          rm -rf "${CHECKOUT_DIR}"/.[!.]*
          # Delete files and directories starting with .. plus any other character
          rm -rf "${CHECKOUT_DIR}"/..?*
        fi
      }
      if [ "${PARAM_DELETE_EXISTING}" = "true" ] ; then
        cleandir || true
      fi
      test -z "${PARAM_HTTP_PROXY}" || export HTTP_PROXY="${PARAM_HTTP_PROXY}"
      test -z "${PARAM_HTTPS_PROXY}" || export HTTPS_PROXY="${PARAM_HTTPS_PROXY}"
      test -z "${PARAM_NO_PROXY}" || export NO_PROXY="${PARAM_NO_PROXY}"
      git config --global --add safe.directory "${WORKSPACE_SOURCE_PATH}"
      /ko-app/git-init \
        -url="${PARAM_URL}" \
        -revision="${PARAM_REVISION}" \
        -refspec="${PARAM_REFSPEC}" \
        -path="${CHECKOUT_DIR}" \
        -sslVerify="${PARAM_SSL_VERIFY}" \
        -submodules="${PARAM_SUBMODULES}" \
        -depth="${PARAM_DEPTH}" \
        -sparseCheckoutDirectories="${PARAM_SPARSE_CHECKOUT_DIRECTORIES}"
      cd "${CHECKOUT_DIR}"
      RESULT_SHA="$(git rev-parse HEAD)"
      EXIT_CODE="$?"
      if [ "${EXIT_CODE}" != 0 ] ; then
        exit "${EXIT_CODE}"
      fi
      RESULT_COMMITTER_DATE="$(git log -1 --pretty=%ct)"
      printf "%s" "${RESULT_COMMITTER_DATE}" > "$(results.committer-date.path)"
      printf "%s" "${RESULT_SHA}" > "$(results.commit.path)"
      printf "%s" "${PARAM_URL}" > "$(results.url.path)"
`

const GoTestTaskContent = `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: {{ .PredefinedTaskName }}
  namespace: {{ .Namespace }}
  labels:
    app.kubernetes.io/version: "0.2"
  annotations:
    tekton.dev/pipelines.minVersion: "0.12.1"
    tekton.dev/categories: Testing
    tekton.dev/tags: test
    tekton.dev/displayName: "golang test"
    tekton.dev/platforms: "linux/amd64,linux/s390x,linux/ppc64le"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
spec:
  description: >-
    This Task is Golang task to test Go projects.

  params:
  - name: package
    description: package (and its children) under test
    default: "{{ default "." .Params.package }}"
  - name: packages
    description: "packages to test (default: ./...)"
    default: "{{ default "./..." .Params.packages }}"
  - name: context
    description: path to the directory to use as context.
    default: "{{ default "." .Params.context }}"
  - name: version
    description: golang version to use for tests
    default: "{{ default "latest" .Params.version }}"
  - name: flags
    description: flags to use for the test command
    default: "{{ default "-race -cover -v" .Params.flags }}"
  - name: GOOS
    description: "running program's operating system target"
    default: "{{ default "linux" .Params.GOOS }}"
  - name: GOARCH
    description: "running program's architecture target"
    default: "{{ default "amd64" .Params.GOARCH }}"
  - name: GO111MODULE
    description: "value of module support"
    default: "{{ default "auto" .Params.GO111MODULE }}"
  - name: GOCACHE
    description: "Go caching directory path"
    default: "{{ default "" .Params.GOCACHE }}"
  - name: GOMODCACHE
    description: "Go mod caching directory path"
    default: "{{ default "" .Params.GOMODCACHE }}"
  workspaces:
  - name: source
  steps:
  - name: unit-test
    image: docker.io/library/golang:$(params.version)
    workingDir: $(workspaces.source.path)
    script: |
      if [ ! -e $GOPATH/src/$(params.package)/go.mod ];then
         SRC_PATH="$GOPATH/src/$(params.package)"
         mkdir -p $SRC_PATH
         cp -R "$(workspaces.source.path)/$(params.context)"/* $SRC_PATH
         cd $SRC_PATH
      fi
      go test $(params.flags) $(params.packages)
    env:
    - name: GOOS
      value: "$(params.GOOS)"
    - name: GOARCH
      value: "$(params.GOARCH)"
    - name: GO111MODULE
      value: "$(params.GO111MODULE)"
    - name: GOCACHE
      value: "$(params.GOCACHE)"
    - name: GOMODCACHE
      value: "$(params.GOMODCACHE)"
`

const GoLintTaskContent = `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: {{ .PredefinedTaskName }}
  namespace: {{ .Namespace }}
  labels:
    app.kubernetes.io/version: "0.2"
  annotations:
    tekton.dev/pipelines.minVersion: "0.12.1"
    tekton.dev/categories: Code Quality
    tekton.dev/tags: lint
    tekton.dev/displayName: "golangci lint"
    tekton.dev/platforms: "linux/amd64"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
spec:
  description: >-
    This Task is Golang task to validate Go projects.

  params:
  - name: package
    description: base package (and its children) under validation
    default: "{{ default "." .Params.package }}"
  - name: context
    description: path to the directory to use as context.
    default: "{{ default "." .Params.context }}"
  - name: flags
    description: flags to use for the lint command
    default: "{{ default "--verbose" .Params.flags }}"
  - name: version
    description: golangci-lint version to use
    default: "{{ default "latest" .Params.version }}"
  - name: GOOS
    description: "running program's operating system target"
    default: "{{ default "linux" .Params.GOOS }}"
  - name: GOARCH
    description: "running program's architecture target"
    default: "{{ default "amd64" .Params.GOARCH }}"
  - name: GO111MODULE
    description: "value of module support"
    default: "{{ default "auto" .Params.GO111MODULE }}"
  - name: GOCACHE
    description: "Go caching directory path"
    default: "{{ default "" .Params.GOCACHE }}"
  - name: GOMODCACHE
    description: "Go mod caching directory path"
    default: "{{ default "" .Params.GOMODCACHE }}"
  - name: GOLANGCI_LINT_CACHE
    description: "golangci-lint cache path"
    default: "{{ default "" .Params.GOLANGCI_LINT_CACHE }}"
  workspaces:
  - name: source
    mountPath: /workspace/src/$(params.package)
  steps:
  - name: lint
    image: docker.io/golangci/golangci-lint:$(params.version)
    workingDir: $(workspaces.source.path)/$(params.context)
    script: |
      golangci-lint run $(params.flags)
    env:
    - name: GOPATH
      value: /workspace
    - name: GOOS
      value: "$(params.GOOS)"
    - name: GOARCH
      value: "$(params.GOARCH)"
    - name: GO111MODULE
      value: "$(params.GO111MODULE)"
    - name: GOCACHE
      value: "$(params.GOCACHE)"
    - name: GOMODCACHE
      value: "$(params.GOMODCACHE)"
    - name: GOLANGCI_LINT_CACHE
      value: "$(params.GOLANGCI_LINT_CACHE)"
`
const BuildPushImageContent = `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: {{ .PredefinedTaskName }}
  namespace: {{ .Namespace }}
  labels:
    app.kubernetes.io/version: "0.6"
  annotations:
    tekton.dev/pipelines.minVersion: "0.17.0"
    tekton.dev/categories: Image Build
    tekton.dev/tags: image-build
    tekton.dev/displayName: "Build and upload container image using Kaniko"
    tekton.dev/platforms: "linux/amd64,linux/arm64,linux/ppc64le"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
spec:
  description: >-
    This Task builds a simple Dockerfile with kaniko and pushes to a registry.
    This Task stores the image name and digest as results, allowing Tekton Chains to pick up
    that an image was built & sign it.
  params:
  - name: IMAGE # This is a parameter that must be set.
    description: Name (reference) of the image to build.
    default: {{ default "Unknown" .Params.image }}
  - name: DOCKERFILE
    description: Path to the Dockerfile to build.
    default: {{ default "./Dockerfile" .Params.dockerfile }}
  - name: CONTEXT
    description: The build context used by Kaniko.
    default: {{ default "./" .Params.context }}
  - name: EXTRA_ARGS # more details see https://github.com/GoogleContainerTools/kaniko?tab=readme-ov-file#additional-flags
    type: array
    default: {{ default "[]" .Params.extra_args }}
  - name: BUILDER_IMAGE
    description: The image on which builds will run (default is v1.19.2 debug)
    default: {{ default "gcr.io/kaniko-project/executor@sha256:899886a2db1c127ff1565d5c7b1e574af1810bbdad048e9850e4f40b5848d79c" .Params.builder_image }}
  workspaces:
  - name: source
    description: Holds the context and Dockerfile
  - name: dockerconfig
    description: Includes a docker "config.json"
    optional: true
    mountPath: /kaniko/.docker
  results:
  - name: IMAGE_DIGEST
    description: Digest of the image just built.
  - name: IMAGE_URL
    description: URL of the image just built.
  steps:
  - name: build-and-push
    workingDir: $(workspaces.source.path)
    image: $(params.BUILDER_IMAGE)
    args:
    - $(params.EXTRA_ARGS)
    - --dockerfile=$(params.DOCKERFILE)
    - --context=$(workspaces.source.path)/$(params.CONTEXT) # The user does not need to care the workspace and the source.
    - --destination=$(params.IMAGE)
    - --digest-file=$(results.IMAGE_DIGEST.path)
    - --ignore-path=/product_uuid # in case kind cluster run failed, see https://github.com/GoogleContainerTools/kaniko/issues/2164
    # kaniko assumes it is running as root, which means this example fails on platforms
    # that default to run containers as random uid (like OpenShift). Adding this securityContext
    # makes it explicit that it needs to run as root.
    securityContext:
      runAsUser: 0
  - name: write-url
    image: docker.io/library/bash:5.1.4@sha256:c523c636b722339f41b6a431b44588ab2f762c5de5ec3bd7964420ff982fb1d9
    script: |
      set -e
      image="$(params.IMAGE)"
      echo -n "${image}" | tee "$(results.IMAGE_URL.path)"
`
