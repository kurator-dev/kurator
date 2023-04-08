# API Reference

## Packages
- [fleet.kurator.dev/v1alpha1](#fleetkuratordevv1alpha1)


## fleet.kurator.dev/v1alpha1

Package v1alpha1 contains API Schema definitions for the fleet v1alpha1 API group

### Resource Types
- [Fleet](#fleet)
- [FleetList](#fleetlist)



#### Fleet



Fleet represents a group of clusters, it is to consistently manage a group of clusters.

_Appears in:_
- [FleetList](#fleetlist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `fleet.kurator.dev/v1alpha1`
| `kind` _string_ | `Fleet`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[FleetSpec](#fleetspec)_ |  |


#### FleetList



FleetList contains a list of fleets.



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `fleet.kurator.dev/v1alpha1`
| `kind` _string_ | `FleetList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Fleet](#fleet) array_ |  |


#### FleetSpec



FleetSpec defines the desired state of the fleet

_Appears in:_
- [Fleet](#fleet)

| Field | Description |
| --- | --- |
| `clusters` _[ObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectreference-v1-core) array_ | Clusters represents the clusters that would be registered to the fleet. Note: only kurator cluster is supported now TODO: add attached cluster support? |




