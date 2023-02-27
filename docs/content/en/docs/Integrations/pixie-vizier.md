---
title: "Integrate Pixie Vizier with Kurator"
---

## What is Pixie Vizier

[Pixie](https://pixielabs.ai/) is an open source observability tool for Kubernetes applications. Pixie uses eBPF to automatically capture telemetry data without the need for manual instrumentation.

Developers can use Pixie to view the high-level state of their cluster (service maps, cluster resources, application traffic) and also drill-down into more detailed views (pod state, flame graphs, individual full body application requests).

The Pixie platform consists of multiple components:

- Pixie Edge Module (PEM): Pixie's agent, installed per node. PEMs use eBPF to collect data, which is stored locally on the node.

- Vizier: Pixie’s collector, installed per cluster. Responsible for query execution and managing PEMs.

- Pixie Cloud: Used for user management, authentication, and data proxying.


In this task, we will show you how to install Pixie vizier(work with Community Cloud) with kurator.

## Prerequisites

{{% readfile "prerequisites-karmada.md" %}}

## Get an account from Community Cloud for Pixie

Visit [pixie product page](https://work.withpixie.ai/) and sign up.

After sign in, visit [pixie admin page](https://work.withpixie.ai/admin) for next step.

{{< image link="./image/pixie-admin-page.png" >}}

## Create deployment key

Create a deployment key following the directions [here](https://docs.pixielabs.ai/reference/admin/deploy-keys/#create-a-deploy-key-using-the-live-ui).

## Install Pixie Vizier

Kurator provides a very simple command to install Pixie vizier to all clusters joined to karmada.

- `--cloud-addr` sepcifies the address of the Pixie cloud instance that the vizier should be connected to.
- `--deploy-key` sepcifies the deploy key is used to link the deployed vizier to a specific user/project.

```bash
kurator install pixie vizier --deploy-key=<your_deploy_key>
```

Wait for cluster become `HEALTHY`:

{{< image link="./image/pixie.png" >}}

## Tutorials

Following the [tutorials](https://docs.pixielabs.ai/tutorials/) to experience with Pixie.
