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

![Kurator architecture diagram](./docs/images/kurator-arch.svg)

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
