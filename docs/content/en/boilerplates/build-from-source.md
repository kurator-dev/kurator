Build cluster operator image and helm chart:

```console
VERSION={{< kurator-version >}} make docker
VERSION={{< kurator-version >}} make gen-chart
```

Load image to kind cluster:

```console
kind load docker-image ghcr.io/kurator-dev/cluster-operator:{{< kurator-version >}} --name kurator-host
kind load docker-image ghcr.io/kurator-dev/fleet-manager:{{< kurator-version >}} --name kurator-host
```
