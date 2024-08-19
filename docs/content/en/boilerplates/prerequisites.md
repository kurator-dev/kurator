### Setup kubernetes clusters with [Kind](https://kind.sigs.k8s.io/)

Download kurator source and enter kurator directory.

```bash
git clone https://github.com/kurator-dev/kurator.git
cd kurator
```

Deploy a kubernetes cluster using kurator's scripts.

```console
$ hack/local-dev-setup.sh
reparing kind config in path: /tmp/tmp.xxxxx
...
Local clusters is running.

To start using your host cluster, run:
  export KUBECONFIG=/root/.kube/kurator-host.config

To manage your remote clusters, run:
  export KUBECONFIG=/root/.kube/kurator-member1.config or export KUBECONFIG=/root/.kube/kurator-member2.config
```

When the console displays the above content, it indicates your Kind cluster is ready, and at this point, the host cluster `kurator-host` will be used.

In addition, you can view the configuration files of the created clusters through this command.

```console
$ ls /root/.kube/ | grep kurator
kurator-host.config
kurator-member1.config
kurator-member2.config
```

### Install cert manager

Kurator cluster operator depends on [cert manager CA injector](https://cert-manager.io/docs/concepts/ca-injector).

***Please make sure cert manager is ready before install cluster operator***

```console
helm repo add jetstack https://charts.jetstack.io
helm repo update
kubectl create namespace cert-manager
helm install -n cert-manager cert-manager jetstack/cert-manager --set crds.enabled=true --version v1.15.3
```
