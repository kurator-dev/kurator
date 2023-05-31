apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: HelmRepository
metadata:
  name: "{{ $.ResourceName }}"
  namespace: "{{ .Fleet.Namespace }}"
  labels:
    app.kubernetes.io/managed-by: fleet-manager
    fleet.kurator.dev/name: "{{ .Fleet.Name }}"
    fleet.kurator.dev/plugin: "{{ .Name }}"
    fleet.kurator.dev/component: "{{ .Component }}"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
spec:
  type: "{{ .Chart.Type }}"
  interval: 5m0s
  url: "{{ .Chart.Repo }}"
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: "{{ .ResourceName }}"
  namespace: "{{ .Fleet.Namespace }}"
  labels:
    app.kubernetes.io/managed-by: fleet-manager
    fleet.kurator.dev/name: "{{ .Fleet.Name }}"
    fleet.kurator.dev/plugin: "{{ .Name }}"
    fleet.kurator.dev/component: "{{ .Component }}"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
spec:
  chart:
    spec:
      chart: "{{ .Chart.Name }}"
      version: "{{ .Chart.Version }}"
      sourceRef:
        kind: HelmRepository
        name: "{{ .ResourceName }}"
{{- if or .Chart.Values  .Values }}
  values:
    {{- merge .Values .Chart.Values | toYaml | nindent 4 }}
{{- end }}
  interval: 1m0s
  install:
    createNamespace: true
  targetNamespace: "{{ .Chart.TargetNamespace }}"
  storageNamespace: "{{ .StorageNamespace }}"
  timeout: 15m0s
{{- if .Cluster }}
  kubeConfig:
    secretRef:
      name: {{ .Cluster.SecretName }}
      key: {{ .Cluster.SecretKey }}
{{- end }}
