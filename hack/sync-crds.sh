#!/bin/bash

# Copyright Istio Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# shellcheck disable=SC2046,SC2086

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
HELM_CRD_BASE=${REPO_ROOT}/manifests/charts/cluster-operator/crds
WEBHOOK_BASE=${REPO_ROOT}/manifests/charts/cluster-operator/templates
CLUSTER_API_PROVIDER_VERSION=${CLUSTER_API_PROVIDER_VERSION:-'v1.2.5'}
AWS_PROVIDER_VERSION=${AWS_PROVIDER_VERSION:-'v2.0.0'}
# set default ca prevent deleting crd fail
CA_BUNDLE=${CA_BUNDLE:-'LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUQ0ekNDQXN1Z0F3SUJBZ0lVQ3lxYVFvUktQM1ZtVk52Q0hTazM1UDBnQW5Vd0RRWUpLb1pJaHZjTkFRRUwKQlFBd1hERUxNQWtHQTFVRUJoTUNlSGd4Q2pBSUJnTlZCQWdNQVhneENqQUlCZ05WQkFjTUFYZ3hDakFJQmdOVgpCQW9NQVhneENqQUlCZ05WQkFzTUFYZ3hDekFKQmdOVkJBTU1BbU5oTVJBd0RnWUpLb1pJaHZjTkFRa0JGZ0Y0Ck1CNFhEVEl5TVRFeE1qQTNORFV3TUZvWERUSTNNVEV4TVRBM05EVXdNRm93UmpFWE1CVUdBMVVFQ2hNT2MzbHoKZEdWdE9tMWhjM1JsY25NeEt6QXBCZ05WQkFNTUlpb3VhM1Z5WVhSdmNpMXplWE4wWlcwdWMzWmpMbU5zZFhOMApaWEl1Ykc5allXd3dnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFDYThhYk1IVklNCkkxNFp4SDUraHc4SlIwSEVucmlxV0RPMTdTd0hoY21VRHc4emd5d1hwNGdjdTA0SDB6cUFYcTZvNU9hd1lJbFkKSjVsVGloMnBhaGV2K0N5VDhka0pPSURTcE5uMThaZnl2UTkvYVZDdDhiMkJCMTJQcHVwSU1PWEV0dXJ3TFpmMgpCMVlYdzBPQkNJVXVydE96MW9zSlRjZUw1dlUvdkFyNVUvQzN6YVVITFlXTUlQY2tkR2U0dk1FbkdqSjhxUEg0CkxyTUpiZkFnUmZuNmJTQVJQNE01SHYrOVFNTnZjVTY3eWhUcEl0KzlHcU9seG9hbjMzb1J6NnNWeFMwWkFybnEKVm1vWmVZVWJobFhFek5HeXlMQ1UxeWQyY2d0aG1jR1lmb0NwTjFwYW81MUdVWlp5MnRYYWRia3Iraitmdjh0ZQpMOVJrVHgzdzZXaVJBZ01CQUFHamdiSXdnYTh3RGdZRFZSMFBBUUgvQkFRREFnV2dNQjBHQTFVZEpRUVdNQlFHCkNDc0dBUVVGQndNQ0JnZ3JCZ0VGQlFjREFUQU1CZ05WSFJNQkFmOEVBakFBTUIwR0ExVWREZ1FXQkJTYXNuK2cKUEtsdnM2RE9KS2FoWWgydFpyRDlOREFmQmdOVkhTTUVHREFXZ0JTeFBxQWovZXNLV2lpemY5c3EwS3Fxd0hsQQpEekF3QmdOVkhSRUVLVEFuZ2hRcUxtdDFjbUYwYjNJdGMzbHpkR1Z0TG5OMlk0SUpiRzlqWVd4b2IzTjBod1IvCkFBQUJNQTBHQ1NxR1NJYjNEUUVCQ3dVQUE0SUJBUUIydWNuWlBKV1h5WGVkSDBYdEZreHpoTlhUeU0zdkttREEKSWk4TDViZmNVelh4VUVqb1B5aldYRVR0QVNtbTBNS3NUdTdDK3ptalM4QVNOUWZvRjdVcFM0bGNpakc5TXY0RAp1WEdNZnlJTU1hcUUrRWJFR2NJdWVKMEZBMjVZT2tDdzFza1BFVkdVTERZZENVWE1wMDZ1R2ZsR0hiL0JPMm4rCmMvbk5ZdDR4RmtySEhKaUNZM3FXQkhhN0FLVXAxZHU1bnN6Mm4yK3E5WkN0Nmp3YThKaC9zUlZDZFlDNC9tWVkKTWc5VkFjMDZlcTA2dmFWY1pxZU5UbnR1TGFVY3R2KzkvL2NSTzFncS9wbEFFZ015R2V3YUxYWHptcGNlWnV1RwpmbTlzRDdXeHltSlpXaGdEaFAvUmpwZm1rcHNFaGNUcm1leHJFbTJEZ01QVDVwdWhuQmQwCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K'}

# gen crds
rm -rf ${HELM_CRD_BASE}/*.x-k8s.io.yaml
CRD_OUTPUT_DIR=${HELM_CRD_BASE} CLUSTER_API_PROVIDER_VERSION=${CLUSTER_API_PROVIDER_VERSION} AWS_PROVIDER_VERSION=${AWS_PROVIDER_VERSION} go run cmd/crd-gen-tool/main.go

# sed cert
sed -i "s|caBundle: Cg==|caBundle: ${CA_BUNDLE}|g" $(find ${HELM_CRD_BASE} -type f)

# capi
sed -i "s|capi-serving-cert|kurator-serving-cert|g" $(find ${HELM_CRD_BASE} -type f)
sed -i "s|capi-system|{{ .Release.Namespace }}|g" $(find ${HELM_CRD_BASE} -type f)
sed -i "s|capi-webhook-service|kurator-webhook-service|g" $(find ${HELM_CRD_BASE} -type f)

# bootstrap
sed -i "s|capi-kubeadm-bootstrap-serving-cert|kurator-serving-cert|g" $(find ${HELM_CRD_BASE} -type f)
sed -i "s|capi-kubeadm-bootstrap-system|{{ .Release.Namespace }}|g" $(find ${HELM_CRD_BASE} -type f)
sed -i "s|capi-kubeadm-bootstrap-webhook-service|kurator-webhook-service|g" $(find ${HELM_CRD_BASE} -type f)

# kubeadm-control-plane
sed -i "s|capi-kubeadm-control-plane-serving-cert|kurator-serving-cert|g" $(find ${HELM_CRD_BASE} -type f)
sed -i "s|capi-kubeadm-control-plane-system|{{ .Release.Namespace }}|g" $(find ${HELM_CRD_BASE} -type f)
sed -i "s|capi-kubeadm-control-plane-webhook-service|kurator-webhook-service|g" $(find ${HELM_CRD_BASE} -type f)

# capa
sed -i "s|capa-serving-cert|kurator-serving-cert|g" $(find ${HELM_CRD_BASE} -type f)
sed -i "s|capa-system|{{ .Release.Namespace }}|g" $(find ${HELM_CRD_BASE} -type f)
sed -i "s|capa-webhook-service|kurator-webhook-service|g" $(find ${HELM_CRD_BASE} -type f)

mv ${HELM_CRD_BASE}/*-webhook-configuration.yaml ${WEBHOOK_BASE}