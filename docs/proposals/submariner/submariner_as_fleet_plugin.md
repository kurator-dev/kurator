---
title: Add submariner as fleet plugin for multi-cluster networking support
authors:
- @Flying-Tom
reviewers:
approvers:

creation-date: 2024-07-31


---

## Add submariner as fleet plugin for multi-cluster networking support

### Summary

The installation of [submariner](https://submariner.io/) can be currently done by the [kurator CLI](cmd/kurator/app/install/submariner/submariner.go). A more elegant way to install submariner is to use the fleet manager as a Fleet plugin, which will enhance the user experience and the robustness of the installation process.

### Motivation

By implementng this feature, users can finish the installation of submariner by the fleet manager as a Fleet plugin, which will make the installation of submariner more convenient and unified.

#### Goals

1. Expand the fleet manager's capabilities to install submariner as a Fleet plugin.
2. Simplify user configuration and provide customizations for submariner.

### Proposal

We propose introducing a new plugin configuration for submariner in the fleet manager. The plugin configuration will be added to the `pkg/fleet-manager/manifests/plugins/submariner.yaml` file. `reconcileSubmarinerPlugin` needs to be implemented and registered in `reconcilePlugins` of `pkg/fleet-manager/fleet_plugin.go`. To be more specific, methods as `renderSubmarinerPlugin` will be implemented and modifications of Fleet API will be introduced to support the plugin configuration.

### Design Details

Plugin config of submariner will be added to `pkg/fleet-manager/manifests/plugins/submariner.yaml` as follows:

```yaml
type: default
repo: https://submariner-io.github.io/submariner-charts/charts
name: submariner
version: 0.14.9
targetNamespace: submariner
```

[Fleet API](pkg/apis/fleet/v1alpha1/types.go) will be extended to support the plugin configuration:

```go
type SubMarinerConfig struct {
 // Chart defines the helm chart config of the submariner.
 // default value is
 //
 // ```yaml
 // chart:
 //   repository: https://submariner-io.github.io/submariner-charts/charts
 //   name: submariner
 //   version: 0.14.9
 //   targetNamespace: submariner
 // ```
 //
 // +optional
 Chart *ChartConfig `json:"chart,omitempty"`
 // ExtraArgs is the set of extra arguments for submariner, and example will be provided in the future.
 //
 // +optional
 ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}
```

#### Test Plan

During the development phase, UTs will be added to cover the core functionalities and edge cases. Post-development, integration tests will be designed to ensure the proper operation of the submariner plugin. Examples demonstrating the submariner plugin will be provided.
