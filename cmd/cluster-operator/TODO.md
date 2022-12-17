# TODO

## How to install CNI?

### Option 1: ClusterResouceSet

```yaml
apiVersion: v1
data: ${CNI_RESOURCES}
kind: ConfigMap
metadata:
  name: cni-capa-quickstart-crs-0
---
apiVersion: addons.cluster.x-k8s.io/v1beta1
kind: ClusterResourceSet
metadata:
  name: capa-quickstart-crs-0
spec:
  clusterSelector:
    matchLabels:
      cni: capa-quickstart-crs-0 # same annotation on Cluster
  resources:
  - kind: ConfigMap
    name: cni-capa-quickstart-crs-0
  strategy: ApplyOnce
```

### Option 2: CRD

```yaml
apiVersion: addons.kurator.dev/v1alphha
kind: CNI
metadata:
  name: capa-quickstart-cni
spec:
  clusterName: capa-quickstart-cni
  type: calico
  values:
    calicoNetwork:
        bgp: Disabled
        ipPools:
        - cidr: 198.51.100.0/24
        encapsulation: VXLAN
```

### Option 3: OSC?

## How to support Ingresss/LoadBalancer Service? 

project [AWS Load Balancer Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller)

How to install?

- Option 1: ClusterResouceSet

- Option 2: CRD

- Option 3: guide user with doc
