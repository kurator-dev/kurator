---
title: "Unified Backup, Restore, and Migration"
linkTitle: "Unified Backup"
weight: 40
description: >
  The easiest way to manage multi cluster Unified Backup, Restore, and Migration with Fleet.
---

Kurator provides a unified solution for backup, restore, and migration of applications and their related cluster resources across multiple Fleet clusters. 
This approach addresses the challenges of managing these tasks across various environments, ensuring clarity and consistency for users.

Through its integration with [Velero](https://velero.io/), Kurator delivers a one-click solution with a unified status view. This empowers users with efficient management and clear visibility into application backups, restores, and migrations across Fleet clusters.

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

### Key Components Overview

- **Kurator Fleet Manager**: The core management component, responsible for managing fleet and backup engines.

- **Backup Engine**: This component handles the backup and restore processes.

- **Object Storage**: The external storage solutions, such as Minio or cloud providers, where the backups are stored.


To use Kurator for backup, restore, and migration scenarios, you must first configure and install the necessary backup engine plugin.
Please refer to the subsequent sections for detailed guidance on plugin configuration and instructions for each specific operation.
