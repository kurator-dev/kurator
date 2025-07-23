---
title: "Install Minio"
linkTitle: "Install Minio"
weight: 40
description: >
  Instructions on installing Minio.
---

## Install Minio with Helm

Kurator use helm chart from bitnami, more [details](https://github.com/bitnami/charts) can be found here.

Setup [Minio](https://min.io/) with following command:

```console
cat <<EOF | helm install minio oci://registry-1.docker.io/bitnamicharts/minio -n monitoring --create-namespace -f -
auth:
  rootPassword: minio123
  rootUser: minio
defaultBuckets: thanos,velero
accessKey:
  password: minio
secretKey:
  password: minio123
service:
  type: LoadBalancer
EOF
```

Check the controller status:

```console
kubectl get po -n monitoring
```

*Optional*, Create a secret for Thanos:

```console
export MINIO_SERVICE_IP=$(kubectl get svc --namespace monitoring minio --template "{{ range (index .status.loadBalancer.ingress 0) }}{{ . }}{{ end }}")
cat <<EOF > objstore.yaml
type: S3
config:
  bucket: "thanos"
  endpoint: "${MINIO_SERVICE_IP}:9000"
  access_key: "minio"
  insecure: true
  signature_version2: false
  secret_key: "minio123"
EOF
```

```console
kubectl create secret generic thanos-objstore --from-file=objstore.yml=./objstore.yaml
```

*Optional*, Create a secret for Velero:

```console
kubectl create secret generic minio-credentials --from-literal=access-key=minio --from-literal=secret-key=minio123
```

confirm your minio service ip

```console
export MINIO_SERVICE_IP=$(kubectl get svc --namespace monitoring minio --template "{{ range (index .status.loadBalancer.ingress 0) }}{{ . }}{{ end }}")
echo $MINIO_SERVICE_IP
```

## Cleanup

```bash
helm delete minio -n monitoring

kubectl delete secret thanos-objstore
```
