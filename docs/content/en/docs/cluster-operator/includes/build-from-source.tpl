Build cluster operator image and helm chart:

```console
VERSION=0.3-dev make gen-chart
```

Load image to kind cluster:

```console
kind load docker-image ghcr.io/kurator-dev/cluster-operator:0.3-dev --name kurator
```