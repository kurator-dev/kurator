---
title: "Enable Policy Management with fleet"
linkTitle: "Policy Management"
weight: 30
description: >
  The easiest way to enable policy management with fleet.
---

In this tutorial we’ll cover the basics of how to use [Fleet](https://kurator.dev/docs/references/fleet-api/#fleet) to manage policies on a group of clusters.

## Architecture

Fleet's multi cluster policy management is built on top [Kyverno](https://kyverno.io/), the overall architecture is shown as below:

{{< image width="100%"
    link="./image/fleet-policy.drawio.svg"
    >}}

## Prerequisites

1. Setup Fleet manager following the instructions in the [installation guide](/docs/setup/install-fleet-manager/).

1. Fleet manager depends on [FluxCD](https://fluxcd.io/flux/), setup by following the instructions in the [installation guide](/docs/setup/install-fluxcd/).

### Create a fleet with pod security policy enabled

Run following command to enable [`baseline`](https://kubernetes.io/docs/concepts/security/pod-security-standards/) pod security check:

```console
kubectl apply -f examples/fleet/policy/kyverno.yaml
```

After a while, we can see the fleet is `ready`:

```console
kubectl wait fleet quickstart --for='jsonpath='{.status.phase}'=Ready'
```

### Verify pod security policy

Run following command to create a invalid pod in the fleet:

```console
cat <<EOF | kubectl apply -f -
apiVersion: apps.kurator.dev/v1alpha1
kind: Application
metadata:
  name: kyverno-policy-demo
  namespace: default
spec:
  source:
    gitRepo:
      interval: 3m0s
      ref:
        branch: master
      timeout: 1m0s
      url: https://github.com/kurator-dev/kurator
  syncPolicy:
    - destination:
        fleet: quickstart
      kustomization:
        interval: 5m0s
        path: ./examples/fleet/policy/badpod-demo
        prune: true
        timeout: 2m0s
EOF
```

After a while you can check policy report with following command:

```console
kubectl get policyreport --kubeconfig=/root/.kube/kurator-member1.config
```

you will see warning message like following:

```console
NAME                                  PASS   FAIL   WARN   ERROR   SKIP   AGE
cpol-disallow-capabilities            6      0      0      0       0      15s
cpol-disallow-host-namespaces         5      1      0      0       0      15s
cpol-disallow-host-path               6      0      0      0       0      15s
cpol-disallow-host-ports              6      0      0      0       0      15s
cpol-disallow-host-process            6      0      0      0       0      15s
cpol-disallow-privileged-containers   6      0      0      0       0      15s
cpol-disallow-proc-mount              6      0      0      0       0      15s
cpol-disallow-selinux                 12     0      0      0       0      15s
cpol-restrict-apparmor-profiles       6      0      0      0       0      15s
cpol-restrict-seccomp                 6      0      0      0       0      15s
cpol-restrict-sysctls                 6      0      0      0       0      15s
```

check pod event:

```console
kubectl describe pod badpod --kubeconfig=/root/.kube/kurator-member1.config | grep PolicyViolation
  Warning  PolicyViolation  90s    kyverno-scan       policy disallow-host-namespaces/host-namespaces fail: validation error: Sharing the host namespaces is disallowed. The fields spec.hostNetwork, spec.hostIPC, and spec.hostPID must be unset or set to `false`. rule host-namespaces failed at path /spec/hostIPC/
```

### Apply more policies with fleet application

You can find more policies from [Kyverno](https://kyverno.io/policies/), and sync to clusters with [Fleet Application](/docs/fleet-manager/application/).

## Cleanup

Delete the fleet created

```console
kubectl delete application kyverno-policy-demo
kubectl delete fleet quickstart
```

Uninstall fleet manager:

```console
helm uninstall kurator-fleet-manager -n kurator-system
```

{{< boilerplate cleanup >}}
