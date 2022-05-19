#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# This script is copied from karmada project

# util::install_kubectl will install the given version kubectl
function util::install_kubectl {
    local KUBECTL_VERSION=${1}
    local ARCH=${2}
    local OS=${3:-linux}
    if [ -z "$KUBECTL_VERSION" ]; then
      KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
    fi
    echo "Installing 'kubectl ${KUBECTL_VERSION}' for you"
    curl --retry 5 -sSLo ./kubectl -w "%{http_code}" https://dl.k8s.io/release/"$KUBECTL_VERSION"/bin/"$OS"/"$ARCH"/kubectl | grep '200' > /dev/null
    ret=$?
    if [ ${ret} -eq 0 ]; then
        chmod +x ./kubectl
        mkdir -p ~/.local/bin/
        mv ./kubectl ~/.local/bin/kubectl

        export PATH=$PATH:~/.local/bin
    else
        echo "Failed to install kubectl, can not download the binary file at https://dl.k8s.io/release/$KUBECTL_VERSION/bin/$OS/$ARCH/kubectl"
        exit 1
    fi
}

# util::wait_for_condition blocks until the provided condition becomes true
# Arguments:
#  - 1: message indicating what conditions is being waited for (e.g. 'ok')
#  - 2: a string representing an eval'able condition.  When eval'd it should not output
#       anything to stdout or stderr.
#  - 3: optional timeout in seconds. If not provided, waits forever.
# Returns:
#  1 if the condition is not met before the timeout
function util::wait_for_condition() {
  local msg=$1
  # condition should be a string that can be eval'd.
  local condition=$2
  local timeout=${3:-}

  local start_msg="Waiting for ${msg}"
  local error_msg="[ERROR] Timeout waiting for condition ${msg}"

  local counter=0
  while ! eval ${condition}; do
    if [[ "${counter}" = "0" ]]; then
      echo -n "${start_msg}"
    fi

    if [[ -z "${timeout}" || "${counter}" -lt "${timeout}" ]]; then
      counter=$((counter + 1))
      if [[ -n "${timeout}" ]]; then
        echo -n '.'
      fi
      sleep 1
    else
      echo -e "\n${error_msg}"
      return 1
    fi
  done

  if [[ "${counter}" != "0" && -n "${timeout}" ]]; then
    echo ' done'
  fi
}

# util::wait_file_exist checks if a file exists, if not, wait until timeout
function util::wait_file_exist() {
    local file_path=${1}
    local timeout=${2}
    local error_msg="[ERROR] Timeout waiting for file exist ${file_path}"
    for ((time=0; time<${timeout}; time++)); do
        if [[ -e ${file_path} ]]; then
            return 0
        fi
        sleep 1
    done
    echo -e "\n${error_msg}"
    return 1
}

# util::wait_pod_ready waits for pod state becomes ready until timeout.
# Parmeters:
#  - $1: pod label, such as "app=etcd"
#  - $2: pod namespace, such as "karmada-system"
#  - $3: time out, such as "200s"
function util::wait_pod_ready() {
    local pod_label=$1
    local pod_namespace=$2

    echo "wait the $pod_label ready..."
    set +e
    util::kubectl_with_retry wait --for=condition=Ready --timeout=30s pods -l app=${pod_label} -n ${pod_namespace}
    ret=$?
    set -e
    if [ $ret -ne 0 ];then
      echo "kubectl describe info:"
      kubectl describe pod -l app=${pod_label} -n ${pod_namespace}
      echo "kubectl logs info:"
      kubectl logs -l app=${pod_label} -n ${pod_namespace}
    fi
    return ${ret}
}

# util::wait_apiservice_ready waits for apiservice state becomes Available until timeout.
# Parmeters:
#  - $1: apiservice label, such as "app=etcd"
#  - $3: time out, such as "200s"
function util::wait_apiservice_ready() {
    local apiservice_label=$1

    echo "wait the $apiservice_label Available..."
    set +e
    util::kubectl_with_retry wait --for=condition=Available --timeout=30s apiservices -l app=${apiservice_label}
    ret=$?
    set -e
    if [ $ret -ne 0 ];then
      echo "kubectl describe info:"
      kubectl describe apiservices -l app=${apiservice_label}
    fi
    return ${ret}
}

# util::create_cluster creates a kubernetes cluster
# util::create_cluster creates a kind cluster and don't wait for control plane node to be ready.
# Parmeters:
#  - $1: cluster name, such as "host"
#  - $2: KUBECONFIG file, such as "/var/run/host.config"
#  - $3: node docker image to use for booting the cluster, such as "kindest/node:v1.19.1"
#  - $4: log file path, such as "/tmp/logs/"
function util::create_cluster() {
  local cluster_name=${1}
  local kubeconfig=${2}
  local kind_image=${3}
  local log_path=${4}
  local cluster_config=${5:-}

  mkdir -p ${log_path}
  rm -rf "${log_path}/${cluster_name}.log"
  rm -f "${kubeconfig}"
  nohup kind delete cluster --name="${cluster_name}" >> "${log_path}"/"${cluster_name}".log 2>&1 && kind create cluster --name "${cluster_name}" --kubeconfig="${kubeconfig}" --image="${kind_image}" --config="${cluster_config}" >> "${log_path}"/"${cluster_name}".log 2>&1 &
  echo "Creating cluster ${cluster_name} and the log file is in ${log_path}/${cluster_name}.log"
}

# This function returns the IP address of a docker instance
# Parameters:
#  - $1: docker instance name

function util::get_docker_native_ipaddress(){
  local container_name=$1
  docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${container_name}"
}

# This function returns the IP address and port of a specific docker instance's host IP
# Parameters:
#  - $1: docker instance name
# Note:
#   Use for getting host IP and port for cluster
#   "6443/tcp" assumes that API server port is 6443 and protocol is TCP

function util::get_docker_host_ip_port(){
  local container_name=$1
  docker inspect --format='{{range $key, $value := index .NetworkSettings.Ports "6443/tcp"}}{{if eq $key 0}}{{$value.HostIp}}:{{$value.HostPort}}{{end}}{{end}}' "${container_name}"
}

# util::check_clusters_ready checks if a cluster is ready, if not, wait until timeout
function util::check_clusters_ready() {
  local kubeconfig_path=${1}
  local context_name=${2}

  echo "Waiting for kubeconfig file ${kubeconfig_path} and clusters ${context_name} to be ready..."
  util::wait_file_exist "${kubeconfig_path}" 300
  util::wait_for_condition 'running' "docker inspect --format='{{.State.Status}}' ${context_name}-control-plane &> /dev/null" 300

  kubectl config rename-context "kind-${context_name}" "${context_name}" --kubeconfig="${kubeconfig_path}"

  local os_name
  os_name=$(go env GOOS)
  local container_ip_port
  case $os_name in
    linux) container_ip_port=$(util::get_docker_native_ipaddress "${context_name}-control-plane")":6443"
    ;;
    darwin) container_ip_port=$(util::get_docker_host_ip_port "${context_name}-control-plane")
    ;;
    *)
        echo "OS ${os_name} does NOT support for getting container ip in installation script"
        exit 1
  esac
  kubectl config set-cluster "kind-${context_name}" --server="https://${container_ip_port}" --kubeconfig="${kubeconfig_path}"

  util::wait_for_condition 'ok' "kubectl --kubeconfig ${kubeconfig_path} --context ${context_name} get --raw=/healthz &> /dev/null" 300
}


# util::add_routes will add routes for given kind cluster
# Parameters:
#  - $1: name of the kind cluster want to add routes
#  - $2: the kubeconfig path of the cluster wanted to be connected
#  - $3: the context in kubeconfig of the cluster wanted to be connected
function util::add_routes() {
  unset IFS
  routes=$(kubectl --kubeconfig ${2} --context ${3} get nodes -o jsonpath='{range .items[*]}ip route add {.spec.podCIDR} via {.status.addresses[?(.type=="InternalIP")].address}{"\n"}{end}')
  echo "Connecting cluster ${1} to ${2} ${3}"

  IFS=$'\n'
  for n in $(kind get nodes --name "${1}"); do
    for r in $routes; do
      echo "exec cmd in docker $n $r"
      eval "docker exec $n $r"
    done
  done
  unset IFS
}


function util::connect_kind_clusters() {
  C1="${1}"
  C1_KUBECONFIG="${2}"
  C2="${3}"
  C2_KUBECONFIG="${4}"
  POD_TO_POD_AND_SERVICE_CONNECTIVITY="${5}"

  C1_NODE="${C1}-control-plane"
  C2_NODE="${C2}-control-plane"
  C1_DOCKER_IP=$(docker inspect -f "{{ .NetworkSettings.Networks.kind.IPAddress }}" "${C1_NODE}")
  C2_DOCKER_IP=$(docker inspect -f "{{ .NetworkSettings.Networks.kind.IPAddress }}" "${C2_NODE}")
  if [ "${POD_TO_POD_AND_SERVICE_CONNECTIVITY}" -eq 1 ]; then
    # Set up routing rules for inter-cluster direct pod to pod & service communication
    C1_POD_CIDR=$(kubectl get --kubeconfig="${C1_KUBECONFIG}" --context="${C1}" node -ojsonpath='{.items[0].spec.podCIDR}')
    C2_POD_CIDR=$(kubectl get --kubeconfig="${C2_KUBECONFIG}" --context="${C2}" node -ojsonpath='{.items[0].spec.podCIDR}')
    C1_SVC_CIDR=$(kubectl cluster-info dump --kubeconfig="${C1_KUBECONFIG}" --context="${C1}" | sed -n 's/^.*--service-cluster-ip-range=\([^"]*\).*$/\1/p' | head -n 1)
    C2_SVC_CIDR=$(kubectl cluster-info dump --kubeconfig="${C2_KUBECONFIG}" --context="${C2}" | sed -n 's/^.*--service-cluster-ip-range=\([^"]*\).*$/\1/p' | head -n 1)
    docker exec "${C1_NODE}" ip route add "${C2_POD_CIDR}" via "${C2_DOCKER_IP}"
    docker exec "${C1_NODE}" ip route add "${C2_SVC_CIDR}" via "${C2_DOCKER_IP}"
    docker exec "${C2_NODE}" ip route add "${C1_POD_CIDR}" via "${C1_DOCKER_IP}"
    docker exec "${C2_NODE}" ip route add "${C1_SVC_CIDR}" via "${C1_DOCKER_IP}"
  fi
}


# util::install_metallb will install metallb for given kind cluster
# Parameters:
#  - $1: kubeconfig of the kind cluster want to install metallb
#  - $2: context of the kind cluster want to install metallb
function util::install_metallb() {
  if [ -z "${METALLB_IPS[*]}" ]; then
    # Take IPs from the end of the docker kind network subnet to use for MetalLB IPs
    DOCKER_KIND_SUBNET="$(docker inspect kind | jq '.[0].IPAM.Config[0].Subnet' -r)"
    METALLB_IPS=()
    while read -r ip; do
      echo $ip ==============
      METALLB_IPS+=("$ip")
    done < <(util::cidr_to_ips "$DOCKER_KIND_SUBNET" | tail -n 100)
  fi

  # Give this cluster of those IPs
  RANGE="${METALLB_IPS[0]}-${METALLB_IPS[9]}"
  METALLB_IPS=("${METALLB_IPS[@]:10}")

  echo 'apiVersion: v1
kind: ConfigMap
metadata:
  namespace: metallb-system
  name: config
data:
  config: |
    address-pools:
    - name: default
      protocol: layer2
      addresses:
      - '"$RANGE" | kubectl apply --kubeconfig="$1" --context="$2" -f - 
}

function util::cidr_to_ips() {
    CIDR="$1"
    python3 - <<EOF
from ipaddress import IPv4Network; [print(str(ip)) for ip in IPv4Network('$CIDR').hosts()]
EOF
}