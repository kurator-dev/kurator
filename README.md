# Kurator

## Overview

Kurator is an open source distributed cloud native platform that helps users to build their own distributed cloud native infrastructure and facilitates enterprise digital transformation.

Kurator stands on shoulders of many popular cloud native software stacks including
[Kubernetes](https://github.com/kubernetes/kubernetes), [Istio](https://github.com/istio/istio), [Prometheus](https://github.com/prometheus/prometheus), [FluxCD](https://github.com/fluxcd/flux2), [KubeEdge](https://github.com/kubeedge/kubeedge), [Volcano](https://github.com/volcano-sh/volcano), [Karmada](https://github.com/karmada-io/karmada), [Kyverno](https://github.com/kyverno/kyverno), etc.
It provides powerful capabilities for multi-cloud and multi-cluster management, including:

- Multi-cloud, Edge-cloud, Edge-edge Synergy
- Unified Resource Orchestration
- Unified Scheduling
- Unified Traffic Management
- Unified Telemetry

## Advantages

- Infrastructure-as-Code: declarative way of infrastructure(cluster, node, vpc, etc) management on cloud, edge or on premises.
- Out of box: one button to install cloud native software stacks
- Unified management of clusters with fleet:

1. Support cluster registration and un-registration to a fleet.
1. Application customize and sync across fleet.
1. Namespaces, ServiceAccount, Service sameness across clusters of a fleet.
1. Provide service discovery and communication across clusters.
1. Aggregate metrics from all clusters of a fleet.
1. Provide policy engine to make all clusters have consistent policies.

## Architecture

<div  align="center">
    <img src="./docs/images/kurator-arch.svg" width = "80%" align="center">
</div>

## Documentation

Please visit [kurator website](https://kurator.dev/docs/) for our documentation.

## Contact

If you have any question, feel free to reach out to us in the following ways:

- [mailing group](https://groups.google.com/g/kuator-dev)
- [slack](https://join.slack.com/t/kurator-hq/shared_invite/zt-1sowqzfnl-Vu1AhxgAjSr1XnaFoogq0A)

## Contributing

If you're interested in being a contributor and want to get involved in
developing the Kurator code, please see [CONTRIBUTING](CONTRIBUTING.md) for
details on submitting patches and the contribution workflow.

## License

Kurator is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.

## report a vulnerability

If you find a vulnerability in Kurator, you can report it to our security-team in the [following way](https://github.com/kurator-dev/kurator/blob/main/community/security/report-a-vulnerability.md). We will deal with it as soon as possible.

## Semantic Versioning Specification

1.A normal version number MUST take the form X.Y.Z where X, Y, and Z are non-negative integers, and MUST NOT contain leading zeroes. X is the major version, Y is the minor version, and Z is the patch version. Each element MUST increase numerically. For instance: 1.9.0 -> 1.10.0 -> 1.11.0.

2.Once a versioned package has been released, the contents of that version MUST NOT be modified. Any modifications MUST be released as a new version. Once a package is released with a version number of x.y.z, it is considered stable.

3.Major version zero (0.y.z) is for initial development. Anything MAY change at any time. The public API SHOULD NOT be considered stable.

4.Version 1.0.0 defines the public API. The way in which the version number is incremented after this release is dependent on this public API and how it changes.

5.Patch version Z (x.y.Z | x > 0) MUST be incremented if only backward compatible bug fixes are introduced. A bug fix is defined as an internal change that fixes incorrect behavior.

6.Minor version Y (x.Y.z | x > 0) MUST be incremented if new, backward compatible functionality is introduced to the public API. It MUST be incremented if any public API functionality is marked as deprecated. It MAY be incremented if substantial new functionality or improvements are introduced within the private code. It MAY include patch level changes. Patch version MUST be reset to 0 when minor version is incremented.

7.Major version X (X.y.z | X > 0) MUST be incremented if any backward incompatible changes are introduced to the public API. It MAY also include minor and patch level changes. Patch and minor versions MUST be reset to 0 when major version is incremented.

8.A pre-release version MAY be denoted by appending a hyphen and a series of dot separated identifiers immediately following the patch version. Identifiers MUST comprise only ASCII alphanumerics and hyphens [0-9A-Za-z-]. Identifiers MUST NOT be empty. Numeric identifiers MUST NOT include leading zeroes. Pre-release versions have a lower precedence than the associated normal version. A pre-release version indicates that the version is unstable and might not satisfy the intended compatibility requirements as denoted by its associated normal version. Examples: 0.5.0-rc.0
