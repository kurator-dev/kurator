apiVersion: v1
data:
  infraType: {{ .InfraType }}
  {{- $cniArgs := (.Network.CNI.ExtraArgs | toJson| fromJson) }}
  key1-key2: {{ $cniArgs.key1.key2 }}
  key3: {{ $cniArgs.key3 }}
kind: ConfigMap
metadata:
  name: fake-cni
  namespace: fake-namespace
