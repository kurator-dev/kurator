### Setup kubernetes clusters with Kind

Deploy a kubernetes cluster using kurator's scripts.

```bash
git clone https://github.com/kurator-dev/kurator.git
cd kurator
hack/local-setup-cluster.sh
```

### Install cert manager

Kurator cluster operator depends on [cert manager CA injector](https://cert-manager.io/docs/concepts/ca-injector).

***Please make sure cert manager is ready before install cluster operator***

```console
helm repo add jetstack https://charts.jetstack.io
helm repo update
kubectl create namespace cert-manager
helm install -n cert-manager cert-manager jetstack/cert-manager --set installCRDs=true
```
