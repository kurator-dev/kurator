---
title: "Unified Backup, Restore, and Migrate"
linkTitle: "Unified Backup"
weight: 40
description: >
  The easiest way to manage multi cluster Unified Backup, Restore, and Migration with Fleet.
---

Kurator introduces a unified solution for backup, restore, and migration across multiple clusters in Fleet.
This feature, integrated with [Velero](https://velero.io/), simplifies the process of managing backups, restoring data, and migrating resources across clusters through a streamlined one-click operation.

The following are the main benefits of this feature.

- **Simplified Multi-cluster Operations**: Streamlining management and operational tasks across various clusters for easier resource handling.

- **Backup Support with Scheduled and Immediate Options**: Automate regular backups for data protection and compliance, along with immediate backup options for on-demand needs.

- **One-stop, Flexible Disaster Recovery Solution**: Providing a robust and flexible solution for disaster recovery, allowing tailored recovery of specific resources in specific clusters to ensure operational continuity in adverse scenarios.

- **Effortless Cluster Resource Migration**: Facilitating smooth migration of resources across multiple clusters for workload balancing or transitions to new environments.

- **Unified Status View**: Offering a clear, unified view of resources and backup statuses across clusters, enhancing visibility and control.

## Architecture

Below is the architecture diagram of the Kurator Fleet Backup feature.

{{< image width="100%"
link="./image/backup-arch.svg"
>}}

The diagram illustrates:

- **Fleet Manager**: The central management entity, responsible for orchestrating the operations.
- **Velero Integration**: How the backup and restore processes are handled using Velero.
- **Object Storage**: The external storage solutions, such as Minio or cloud providers, where the backups are stored.
- **Interactions**: The arrows depict the flow of data and interactions between various components.

To use Kurator for backup, restore, and migration scenarios, you must first configure and install the necessary backup engine plugin.
Please refer to the subsequent sections for detailed guidance on plugin configuration and instructions for each specific operation.
