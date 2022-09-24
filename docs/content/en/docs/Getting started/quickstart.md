---
title: "Quick start"
linkTitle: "Quick start"
description: >
  The easest way to get start with Kurator.
---

## Setup kubernetes clusters

Deploy a kubernetes cluster using kurator's scripts. This script will create three clusters for you, one is used to host Karmada control plane and the other two will be joined as member clusters.

```bash
git clone https://github.com/kurator-dev/kurator.git
cd kurator
hack/local-dev-setup.sh
```

## Compile `kurator` from source

```bash
make kurator
```

