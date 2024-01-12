---
title: "Unified Rollout"
linkTitle: "Unified Rollout"
weight: 50
description: >
  This is your go-to guide for Rollout uniformly with Kurator. In this guide, we will introduce how to use Kurator to rollout uniformly with Fleet.
---

Kurator provides a unified Rollout system across multiple clusters supported by Fleet.

By leveraging [Flagger](https://docs.flagger.app/), Kurator can perform Rollout quickly. We extended the [Kurator Application configuration](https://kurator.dev/docs/fleet-manager/application/) to include Rollout configurations. This allows users to deploy applications and their Rollout configurations in a single place.

## Architecture

Below is the architecture diagram of the Kurator Rollout feature.

{{< image width="100%"
link="./image/rollout.svg"
>}}

- **Rollout Controller**: The central control component of the Rollout system. It manages Fleet and rendering Flagger configurations according to the Rollout Policy.

- **primary pod**: The primary pod represents the stable, production-ready version of the application. It is responsible for handling user traffic and serving the live application services.

- **canary pod**: The canary pod contains an experimental version of the application that is deployed temporarily for validation when a new release is rolled out.

To better illustrate how traffic flows between different components, here is the Traffic Routing Diagram for the Kurator Rollout.

{{< image width="100%"
link="./image/traffic.svg"
>}}

In the diagram above, `service-primary` and `service-canary` are both created by the kurator rollout plugin.
`service-primary` is a deep copy of the original service. By modifying the service's selector and creating a virtual service, the plugin migrates traffic from the original pods to pods-primary without interrupting the service.
The `pod-canary` in Kurator is only created when a new deployment is triggered. It performs testing of the new release through gradual traffic shifts driven by changes to the virtual service rules.

To use Rollout, you must first configure and install the necessary engine plugin.
Please refer to the subsequent sections for detailed guidance on plugin configuration and instructions for each specific operation.
