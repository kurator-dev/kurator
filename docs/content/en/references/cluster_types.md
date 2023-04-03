# API Reference

## Packages
- [cluster.kurator.dev/v1alpha1](#clusterkuratordevv1alpha1)


## cluster.kurator.dev/v1alpha1

Package v1alpha1 contains API Schema definitions for the cluster v1alpha1 API group

### Resource Types
- [Cluster](#cluster)





#### CNIConfig





_Appears in:_
- [NetworkConfig](#networkconfig)

| Field | Description |
| --- | --- |
| `type` _string_ | Type is the type of CNI. |
| `extraArgs` _[JSON](#json)_ | ExtraArgs is the set of extra arguments for CNI. |


#### Cluster



Cluster is the schema for the cluster's API



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `cluster.kurator.dev/v1alpha1`
| `kind` _string_ | `Cluster`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ClusterSpec](#clusterspec)_ |  |


#### ClusterInfraType

_Underlying type:_ `string`



_Appears in:_
- [ClusterSpec](#clusterspec)



#### ClusterSpec



ClusterSpec defines the desired state of the Cluster

_Appears in:_
- [Cluster](#cluster)

| Field | Description |
| --- | --- |
| `infraType` _[ClusterInfraType](#clusterinfratype)_ | InfraType is the infra type of the cluster. |
| `credential` _[CredentialConfig](#credentialconfig)_ | Credential is the credential used to access the cloud provider. |
| `version` _string_ | Version is the Kubernetes version to use for the cluster. |
| `region` _string_ | Region is the region to deploy the cluster. |
| `network` _[NetworkConfig](#networkconfig)_ | Network is the network configuration for the cluster. |
| `master` _[MasterConfig](#masterconfig)_ | Master is the configuration for the master node. |
| `workers` _[WorkerConfig](#workerconfig) array_ | Workers is the list of worker nodes. |
| `podIdentity` _[PodIdentityConfig](#podidentityconfig)_ | PodIdentity is the configuration for the pod identity. |
| `additionalResources` _[ResourceRef](#resourceref) array_ | AdditionalResources provides a way to automatically apply a set of resouces to cluster after it's ready. Note: the resouces will only apply once. |




#### CredentialConfig





_Appears in:_
- [ClusterSpec](#clusterspec)

| Field | Description |
| --- | --- |
| `secretRef` _string_ |  |


#### MachineConfig



MachineConfig defines the configuration for the machine.

_Appears in:_
- [NodeConfig](#nodeconfig)

| Field | Description |
| --- | --- |
| `replicas` _integer_ | Replicas is the number of replicas of the machine. |
| `instanceType` _string_ | InstanceType is the type of instance to use for the instance. |
| `sshKeyName` _string_ | SSHKeyName is the name of the SSH key to use for the instance. |
| `imageOS` _string_ | ImageOS is the OS of the image to use for the instance. Defaults to "ubuntu-20.04". |
| `rootVolumeSize` _[Volume](#volume)_ | RootVolume is the root volume to attach to the instance. |
| `nonRootVolumes` _[Volume](#volume) array_ | NonRootVolumes is the list of non-root volumes to attach to the instance. |
| `extraArgs` _[JSON](#json)_ | ExtraArgs is the set of extra arguments to create Machine on different infra. |


#### MasterConfig





_Appears in:_
- [ClusterSpec](#clusterspec)

| Field | Description |
| --- | --- |
| `NodeConfig` _[NodeConfig](#nodeconfig)_ |  |


#### NetworkConfig





_Appears in:_
- [ClusterSpec](#clusterspec)

| Field | Description |
| --- | --- |
| `vpc` _[VPCConfig](#vpcconfig)_ | VPC is the configuration for the VPC. |
| `podCIDRs` _string array_ | PodCIDRs is the CIDR block for pods in this cluster. Defaults to 192.168.0.0/16. |
| `serviceCIDRs` _string array_ | ServiceCIDRs is the CIDR block for services in this cluster. Defaults to 10.96.0.0/12. |
| `cni` _[CNIConfig](#cniconfig)_ | CNI is the configuration for the CNI. |


#### NodeConfig



NodeConfig defines the configuration for the node.

_Appears in:_
- [MasterConfig](#masterconfig)
- [WorkerConfig](#workerconfig)

| Field | Description |
| --- | --- |
| `MachineConfig` _[MachineConfig](#machineconfig)_ |  |
| `NodeRegistrationConfig` _[NodeRegistrationConfig](#noderegistrationconfig)_ |  |


#### NodeRegistrationConfig



NodeRegistrationConfig defines the configuration for the node registration.

_Appears in:_
- [NodeConfig](#nodeconfig)

| Field | Description |
| --- | --- |
| `labels` _object (keys:string, values:string)_ | Labels is the set of labels to apply to the nodes. |
| `taints` _[Taint](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#taint-v1-core) array_ | Taints is the set of taints to apply to the nodes. |


#### NodeUpgradeStrategy





_Appears in:_
- [WorkerConfig](#workerconfig)

| Field | Description |
| --- | --- |
| `type` _[NodeUpgradeStrategyType](#nodeupgradestrategytype)_ | Type of node replacement strategy. Default is RollingUpdate. |
| `rollingUpdate` _[RollingUpdateNodeUpgradeStrategy](#rollingupdatenodeupgradestrategy)_ | RollingUpdate config params. Present only if NodeUpgradeStrategyType = RollingUpdate. |


#### NodeUpgradeStrategyType

_Underlying type:_ `string`



_Appears in:_
- [NodeUpgradeStrategy](#nodeupgradestrategy)



#### PodIdentityConfig



PodIdentityConfig defines the configuration for the pod identity.

_Appears in:_
- [ClusterSpec](#clusterspec)

| Field | Description |
| --- | --- |
| `enabled` _boolean_ | Enabled is true when the pod identity is enabled. |


#### ResourceRef





_Appears in:_
- [ClusterSpec](#clusterspec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the resource. |
| `kind` _string_ | Kind Of the resource. e.g. ConfigMap, Secret, etc. |


#### RollingUpdateNodeUpgradeStrategy





_Appears in:_
- [NodeUpgradeStrategy](#nodeupgradestrategy)

| Field | Description |
| --- | --- |
| `maxUnavailable` _IntOrString_ | MaxUnavailable is the maximum number of nodes that can be unavailable during the update. |
| `maxSurge` _IntOrString_ | MaxSurge is the maximum number of nodes that can be created above the desired number of nodes during the update. |
| `deletePolicy` _string_ | DeletePolicy defines the policy used to identify nodes to delete when downscaling. Valid values are "Random", "Newest" and "Oldest". Defaults to "Newest". |


#### VPCConfig





_Appears in:_
- [NetworkConfig](#networkconfig)

| Field | Description |
| --- | --- |
| `id` _string_ | ID defines a unique identifier to reference this resource. |
| `name` _string_ | Name is the name of the VPC. if not set, the name will be generated from cluster name. |
| `cidrBlock` _string_ | CIDRBlock is the CIDR block to be used when the provider creates a managed VPC. Defaults to 10.0.0.0/16. |


#### Volume





_Appears in:_
- [MachineConfig](#machineconfig)

| Field | Description |
| --- | --- |
| `type` _string_ | Type is the type of the volume (e.g. gp2, io1, etc...). |
| `size` _integer_ | Size specifies size (in Gi) of the storage device. Must be greater than the image snapshot size or 8 (whichever is greater). |


#### WorkerConfig





_Appears in:_
- [ClusterSpec](#clusterspec)

| Field | Description |
| --- | --- |
| `NodeConfig` _[NodeConfig](#nodeconfig)_ |  |
| `strategy` _[NodeUpgradeStrategy](#nodeupgradestrategy)_ | Strategy to use to replace existing nodes with new ones. |


