apiVersion: v1
data:
  cni.yaml: |-
{{ .CNIYaml | indent 4 }}
kind: ConfigMap
metadata:
  name: {{ .Name }}-crs-cni
  namespace: {{ .Namespace }}
  labels:
    infra.kurator.dev/cluster-name: {{ .Name }}
    infra.kurator.dev/cluster-namespace: {{ .Namespace }}
---
apiVersion: addons.cluster.x-k8s.io/v1beta1
kind: ClusterResourceSet
metadata:
  name: {{ .Name }}-crs-cni
  namespace: {{ .Namespace }}
  labels:
    infra.kurator.dev/cluster-name: {{ .Name }}
    infra.kurator.dev/cluster-namespace: {{ .Namespace }}
spec:
  clusterSelector:
    matchLabels:
      infra.kurator.dev/cluster-name: {{ .Name }}
      infra.kurator.dev/cluster-namespace: {{ .Namespace }}
  resources:
    - kind: ConfigMap
      name: {{ .Name }}-crs-cni
  strategy: ApplyOnce
