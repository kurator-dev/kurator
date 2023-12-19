apiVersion: v1
kind: ServiceAccount
metadata:
  name: "{{ .ServiceAccountName }}"
  namespace: "{{ .PipelineNamespace }}"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: "{{ .BroadResourceRoleBindingName }}"
  namespace: "{{ .PipelineNamespace }}"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
subjects:
- kind: ServiceAccount
  name: "{{ .ServiceAccountName }}"
  namespace: "{{ .PipelineNamespace }}"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tekton-triggers-eventlistener-roles # add role for handle broad-resource, such as eventListener, triggers, configmaps and so on. `tekton-triggers-eventlistener-roles` is provided by Tekton
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: "{{ .SecretRoleBindingName }}"
  namespace: "{{ .PipelineNamespace }}"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
subjects:
- kind: ServiceAccount
  name: "{{ .ServiceAccountName }}"
  namespace: "{{ .PipelineNamespace }}"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tekton-triggers-eventlistener-clusterroles # add role for handle secret, clustertriggerbinding and clusterinterceptors. `tekton-triggers-eventlistener-clusterroles` is provided by Tekton
