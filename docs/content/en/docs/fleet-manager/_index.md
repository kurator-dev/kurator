---
title: "Manage Clusters Consistently with Fleet Manager"
linkTitle: "Fleet manager"
weight: 4
description: >
  Learn how to manage clusters with fleet manager.
---

Fleet manager aims to manage a group of clusters consistently. The clusters group is called `fleet`.
Fleet allows you manage clusters distributed in any cloud easily and consistently.


## What can fleet manager do

1. Provide a logic unit `Fleet` that represents a groups of physical clusters.
1. Fleet control plane lifecycle management.
1. Support cluster registration and unregistration to a fleet.
1. Application orchestration across fleet.
1. Namespaces, ServiceAccount, Service sameness across clusters of a fleet.
1. Provide service discovery and communication across clusters.
1. Aggregate metrics from all clusters of a fleet.



## Architecture

The overall architecture of Kurator fleet manager is shown as below:

{{< image width="100%"
    link="./image/fleet-manager.svg"
    >}}


The Kurator Fleet Manager runs as a kubernetes operator, it is in charge of fleet control plane lifecycle management and also responsible for cluster registration and un registration.
