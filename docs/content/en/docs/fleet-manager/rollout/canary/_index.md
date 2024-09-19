---
title: "Canary Deployment"
linkTitle: "Canary Deployment"
weight: 50
description: >
  A comprehensive guide on Kurator's Canary Deployment, providing an overview and quick start guide.
---

## Introduction

Canary Deployment is a software release strategy.
It refers to releasing a new software version to only a very small percentage of users first for testing, to observe if there are any issues. Based on the test results, determine whether to gradually roll out the release to more users.
It aims to maximize reducing the impact on users after a new version goes live. It is considered a safer and more reliable method of software updates.

- **Use Case**: When the system undergoes API changes that require validation through real-world usage, a Canary Deployment should be leveraged to gradually roll out and validate the changes. This incremental approach helps ensure any potential issues are identified and addressed before being exposed to all services/traffic.
- **Functionality**: Provides configuration of Canary Deployment and triggers a Canary Deployment on new release.

By allowing users to deploy applications and their canary configurations in a single place, Kurator streamlines Canary Deployment through automated GitOps workflows for unified deployment and validation.
