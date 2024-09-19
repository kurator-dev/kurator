---
title: "Blue/Green Deployment"
linkTitle: "Blue/Green Deployment"
weight: 50
description: >
  A comprehensive guide on Kurator's Blue/Green Deployment, providing an overview and quick start guide.
---

## Introduction

In Blue/Green Deployment, there are two separate live production environments - the blue environment and the green environment. The blue environment runs the existing version receiving real-time traffic, while the green environment hosts the new release. At any given time, only one of the environments is live with real traffic.

The key benefit of Blue/Green Deployment is that if issues arise in the new version, traffic can be instantaneously switched back to the blue environment running the old version, avoiding any downtime and resulting losses. This allows seamless rollback to the previous known-good release in the event validation fails.

- **Use Case**: If issues are encountered that prevent the new version from functioning properly, the testing process should immediately switch the traffic back to the previously stable legacy release. This ensures users continue receiving an optimal service experience without interruption while the new release issues are addressed.
- **Functionality**: Provides configuration of Blue/Green Deployment and triggers a Blue/Green Deploymenton new release.

By allowing users to deploy applications and their Blue/Green Deployment configurations in a single place, Kurator streamlines Blue/Green Deployment through automated GitOps workflows for unified deployment and validation.
