---
title: "Quick start"
linkTitle: "Quick start"
description: >
  The easiest way to get start with Kurator.
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

## Troubleshooting

### ERROR Timeout waiting for file exist /root/.kube/kurator-host.config

See [kind Known Issues][kind-known-issues]

```console
sysctl fs.inotify.max_user_watches=524288
sysctl fs.inotify.max_user_instances=512
```

### jq: command not found

```console
apt-get install -y jq
```

[kind-known-issues]: https://kind.sigs.k8s.io/docs/user/known-issues/

