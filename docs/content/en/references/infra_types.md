# API Reference

## Packages
- [infrastructure.cluster.x-k8s.io/v1alpha1](#infrastructureclusterx-k8siov1alpha1)


## infrastructure.cluster.x-k8s.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the cluster v1alpha1 API group

### Resource Types
- [CustomCluster](#customcluster)
- [CustomMachine](#custommachine)



#### CNIConfig





_Appears in:_
- [CustomClusterSpec](#customclusterspec)

| Field | Description |
| --- | --- |
| `type` _string_ | Type is the type of CNI. The default value is calico and can be ["calico", "cilium", "canal", "flannel"] |


#### ControlPlaneConfig





_Appears in:_
- [CustomClusterSpec](#customclusterspec)

| Field | Description |
| --- | --- |
| `address` _string_ | same as `ControlPlaneEndpoint` |
| `certSANs` _string array_ | CertSANs sets extra Subject Alternative Names for the API Server signing cert. |


#### CustomCluster



CustomCluster represents the parameters for a cluster in supplement of Cluster API.



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `infrastructure.cluster.x-k8s.io/v1alpha1`
| `kind` _string_ | `CustomCluster`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[CustomClusterSpec](#customclusterspec)_ | Specification of the desired behavior of the kurator cluster. |


#### CustomClusterPhase

_Underlying type:_ `string`



_Appears in:_
- [CustomClusterStatus](#customclusterstatus)



#### CustomClusterSpec



CustomClusterSpec defines the desired state of a kurator cluster.

_Appears in:_
- [CustomCluster](#customcluster)

| Field | Description |
| --- | --- |
| `machineRef` _[ObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectreference-v1-core)_ | MachineRef is the reference of nodes for provisioning a kurator cluster. |
| `cni` _[CNIConfig](#cniconfig)_ | CNIConfig is the configuration for the CNI of the cluster. |
| `controlPlaneConfig` _[ControlPlaneConfig](#controlplaneconfig)_ | ControlPlaneConfig contains control plane configuration. |




#### CustomMachine



CustomMachine is the schema for kubernetes nodes.



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `infrastructure.cluster.x-k8s.io/v1alpha1`
| `kind` _string_ | `CustomMachine`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[CustomMachineSpec](#custommachinespec)_ | Specification of the desired behavior of the kurator cluster. |


#### CustomMachineSpec



CustomMachineSpec defines kubernetes cluster master and nodes.

_Appears in:_
- [CustomMachine](#custommachine)

| Field | Description |
| --- | --- |
| `master` _[Machine](#machine) array_ |  |
| `node` _[Machine](#machine) array_ |  |




#### Machine



Machine defines a node.

_Appears in:_
- [CustomMachineSpec](#custommachinespec)

| Field | Description |
| --- | --- |
| `hostName` _string_ | HostName is the hostname of the machine. |
| `privateIP` _string_ | PrivateIP is the private ip address of the machine. |
| `publicIP` _string_ | PublicIP specifies the public IP. |
| `region` _string_ | Region specifies the region where the machine resides. |
| `zone` _string_ | Region specifies the zone where the machine resides. |
| `sshKey` _[ObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectreference-v1-core)_ | SSHKeyName is the name of the ssh key to attach to the instance. Valid values are empty string (do not use SSH keys), a valid SSH key name, or omitted (use the default SSH key name) |
| `labels` _object (keys:string, values:string)_ | AdditionalTags is an optional set of tags to add to an instance. |


