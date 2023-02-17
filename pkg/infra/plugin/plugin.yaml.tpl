apiVersion: v1
data:
  plugin.yaml: |-
{{ .PluginYAML | indent 4 }}
kind: ConfigMap
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
  labels:
    infra.kurator.dev/cluster-name: {{ .ClusterName }}
    infra.kurator.dev/cluster-namespace: {{ .Namespace }}
---
apiVersion: addons.cluster.x-k8s.io/v1beta1
kind: ClusterResourceSet
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
  labels:
    infra.kurator.dev/cluster-name: {{ .ClusterName }}
    infra.kurator.dev/cluster-namespace: {{ .Namespace }}
spec:
  clusterSelector:
    matchLabels:
      infra.kurator.dev/cluster-name: {{ .ClusterName }}
      infra.kurator.dev/cluster-namespace: {{ .Namespace }}
  resources:
    - kind: ConfigMap
      name: {{ .Name }}
  strategy: ApplyOnce
