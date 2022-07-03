<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Install Karmada](#install-karmada)
  - [Prerequisites](#prerequisites)
    - [Deploy kubernetes clusters using kurator's script](#deploy-kubernetes-clusters-using-kurators-script)
    - [KinD](#kind)
    - [kubeadm](#kubeadm)
  - [Deploy Karmada](#deploy-karmada)
  - [Add kubernetes cluster to karmada control plane](#add-kubernetes-cluster-to-karmada-control-plane)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Install Karmada

The documentation uses `Ubuntu 20.04.4 LTS` as an example.

## Prerequisites

Installing Karmada requires a kubernetes cluster. If you do not have a kubernetes cluster, you can choose any of the following methods to deploy the kubernetes cluster.

### Deploy kubernetes clusters using kurator's script

Deploy a kubernetes cluster using kurator's scripts. This script will create three clusters for you, one is used to host Karmada control plane and the other two will be joined as member clusters.
```bash
git clone https://github.com/kurator-dev/kurator.git
cd kurator
hack/local-dev-setup.sh
```

### KinD

Install docker

[Install on Ubuntu](https://docs.docker.com/engine/install/ubuntu/)


Install KinD
```bash
KIND_VERSION=v0.11.1
curl -Lo ./kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-$(uname)-amd64"
chmod +x ./kind
mv ./kind /usr/local/bin/kind
```

Create a kubernetes cluster
```bash
kind create cluster
```

For more instructions, please refer to [KinD](https://kind.sigs.k8s.io/docs/user/quick-start)

### kubeadm

Load Linux kernel modules and set kernel parameters.
```bash
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF

sudo modprobe overlay
sudo modprobe br_netfilter

# sysctl params required by setup, params persist across reboots
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

# Apply sysctl params without reboot
sudo sysctl --system
```

Install containerd
```
sudo apt-get install  containerd
```

Install Kubeadm Kubelet Kubectl
```bash
sudo apt-get update && sudo apt-get install -y apt-transport-https curl
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
cat <<EOF | sudo tee /etc/apt/sources.list.d/kubernetes.list
deb https://apt.kubernetes.io/ kubernetes-xenial main
EOF

sudo apt-get update
sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
```

Create a kubernetes cluster
```
kubeadm config images pull
kubeadm init

mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

Install CNI, take calico as an example
```
curl https://docs.projectcalico.org/manifests/calico.yaml -O
kubectl apply -f calico.yaml
kubectl -n kube-system get pod -w
```
> Notice
> 
> In the kubernetes cluster created by Kubeadm, the master node is identified as Taints. If your kubernetes cluster has only one node, you need to remove the Taints of the master node.
> 
> `kubectl taint node ${NODE_NAME} node-role.kubernetes.io/control-plane-`
> 
> `kubectl taint node ${NODE_NAME} node-role.kubernetes.io/master-`


For more instructions, please refer to [kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm)

## Deploy Karmada

Compile `kurator` from source
```bash
git clone https://github.com/kurator-dev/kurator.git
cd kurator
make kurator
```

Install karmada control plane
```bash
./kurator install karmada --kubeconfig=/root/.kube/config
```
> When deploying kubernetes using a script, the kubeconfig is kurator-host.config

karmada installation parameters can be set with `--set`, e.g.
```
./kurator install karmada --set karmada-data=/etc/Karmada-test --set port=32222 --kubeconfig .kube/config
```

## Add kubernetes cluster to karmada control plane

```bash
./kurator join karmada member1 \
    --cluster-kubeconfig=/root/.kube/kurator-members.config \
    --cluster-context=kurator-member1
```

Show members of karmada 
```
kubectl --kubeconfig /etc/karmada/karmada-apiserver.config get clusters
```

>Notice
>
> karmada v1.2.0 and below version, does not support kubernetes v1.24.0 and above version join the karmada control plane
>
> For details, please see [1961](https://github.com/karmada-io/karmada/issues/1961)
