apiVersion: v1
kind: ServiceAccount
metadata:
  name: "{{ .ServiceAccountName }}"
  namespace: "{{ .PipelineNamespace }}"
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: "{{ .RoleBindingName }}"
  namespace: "{{ .PipelineNamespace }}"
subjects:
- kind: ServiceAccount
  name: "{{ .ServiceAccountName }}"
  namespace: "{{ .PipelineNamespace }}"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tekton-triggers-eventlistener-roles # add role for handle EventListener, configmap, secret and so on. `tekton-triggers-eventlistener-roles` is provided by Tekton
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: "{{ .ClusterRoleBindingName }}"
subjects:
- kind: ServiceAccount
  name: "{{ .ServiceAccountName }}"
  namespace: "{{ .PipelineNamespace }}"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tekton-triggers-eventlistener-clusterroles # add role for handle pod.  `tekton-triggers-eventlistener-clusterroles` is provided by Tekton
