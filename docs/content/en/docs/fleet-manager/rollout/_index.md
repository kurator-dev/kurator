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

### Key Components Overview

- **Rollout Controller**: The central control component of the Rollout system. It manages Fleet and rendering Flagger configurations according to the Rollout Policy.

- **primary pod**: The primary pod represents the stable, production-ready version of the application. It is responsible for handling user traffic and serving the live application services.

- **canary pod**: The canary pod contains an experimental version of the application that is deployed temporarily for validation when a new release is rolled out.

To use Rollout, you must first configure and install the necessary engine plugin.
Please refer to the subsequent sections for detailed guidance on plugin configuration and instructions for each specific operation.