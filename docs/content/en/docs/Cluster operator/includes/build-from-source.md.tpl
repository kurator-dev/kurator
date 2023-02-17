Build cluster operator image:

```console
make docker
```

Build cluster operator helm chart:

```console
make gen-chart
```

Load image to kind cluster:

```console
kind load docker-image ghcr.io/kurator-dev/cluster-operator:latest --name kurator
```