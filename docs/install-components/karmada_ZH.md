<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [安装 Karmada](#%E5%AE%89%E8%A3%85-karmada)
  - [先决条件](#%E5%85%88%E5%86%B3%E6%9D%A1%E4%BB%B6)
    - [使用 kurator 的脚本部署 Kubernetes 集群](#%E4%BD%BF%E7%94%A8-kurator-%E7%9A%84%E8%84%9A%E6%9C%AC%E9%83%A8%E7%BD%B2-kubernetes-%E9%9B%86%E7%BE%A4)
    - [KinD](#kind)
    - [kubeadm](#kubeadm)
  - [部署 Karmada](#%E9%83%A8%E7%BD%B2-karmada)
  - [把 kubernetes 集群加入 karmada 控制面](#%E6%8A%8A-kubernetes-%E9%9B%86%E7%BE%A4%E5%8A%A0%E5%85%A5-karmada-%E6%8E%A7%E5%88%B6%E9%9D%A2)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# 安装 Karmada

文档以 `Ubuntu 20.04.4 LTS` 为例。

## 先决条件

安装 Karmada 需要一个 kubernetes 集群，如果您还没有 kubernetes 集群，可以选择下列任意方式部署 kubernetes 集群。

### 使用 kurator 的脚本部署 Kubernetes 集群

该脚本将为您创建三个集群，一个用于托管 Karmada 控制面，另外两个将作为成员集群加入。
```bash
git clone https://github.com/kurator-dev/kurator.git
cd kurator
hack/local-dev-setup.sh
```

### KinD
安装 docker

[Install on Ubuntu](https://docs.docker.com/engine/install/ubuntu/)

安装 KinD
```bash
KIND_VERSION=v0.11.1
curl -Lo ./kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-$(uname)-amd64"
chmod +x ./kind
mv ./kind /usr/local/bin/kind  
```

创建 kubernetes 集群
```bash
kind create cluster
```

更多使用说明请参考 [KinD](https://kind.sigs.k8s.io/docs/user/quick-start)

### kubeadm

加载 Linux 内核模块和设置内核参数
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

安装 containerd
```
sudo apt-get install  containerd
```

安装 Kubeadm Kubelet Kubectl
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

创建 kubernetes 集群
```
kubeadm config images pull
kubeadm init

mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

安装 CNI ，以 calico 为例
```
curl https://docs.projectcalico.org/manifests/calico.yaml -O
kubectl apply -f calico.yaml
kubectl -n kube-system get pod -w
```
> 注意
> 
> Kubeadm 创建的 kubernetes 集群，master 节点被标识为 Taints ，如果如果您的 kubernetes 集群只有一个节点，需要去掉 master 节点的 Taints
> 
> `kubectl taint node ${NODE_NAME} node-role.kubernetes.io/control-plane-`
> 
> `kubectl taint node ${NODE_NAME} node-role.kubernetes.io/master-`


更多使用说明请参考 [kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm)

## 部署 Karmada

从源码编译 `kurator`
```bash
git clone https://github.com/kurator-dev/kurator.git
cd kurator
make kurator
```

安装 Karmada
```bash
kurator install karmada --kubeconfig=/root/.kube/config
```
> 使用脚本部署 kubernetes 时，kubeconfig 是 kurator-host.config


可以使用 `--set` 设置 karmada 安装参数，例如
```
./kurator install karmada --set karmada-data=/etc/Karmada-test --set port=32222 --kubeconfig .kube/config
```

## 把 kubernetes 集群加入 karmada 控制面

```bash
kurator join karmada member1 \
    --cluster-kubeconfig=/root/.kube/kurator-members.config \
    --cluster-context=kurator-member1
```

查看 karmada 成员集群
```
kubectl --kubeconfig /etc/karmada/karmada-apiserver.config get clusters
```

>注意
>
> karmada v1.2.0 及以下版本，不支持 kubernetes v1.24.0 及以上版本加入 karmada 控制面
>
> 详情请看 [1961](https://github.com/karmada-io/karmada/issues/1961)
