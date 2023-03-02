apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
  labels:
    cluster.kurator.dev/cluster-name: {{ .Name }}
    cluster.kurator.dev/cluster-namespace: {{ .Namespace }}
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
      {{- range $cidr := .PodCIDR}}
      - {{ $cidr }}
      {{- end }}
    services:
      cidrBlocks:
      {{- range $cidr := .ServiceCIDR}}
      - {{ $cidr }}
      {{- end }}
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: KubeadmControlPlane
    name: {{ .Name }}-kcp
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
    kind: AWSCluster
    name: {{ .Name }}
---
apiVersion: controlplane.cluster.x-k8s.io/v1beta1
kind: KubeadmControlPlane
metadata:
  name: {{ .Name }}-kcp
  namespace: {{ .Namespace }}
spec:
  kubeadmConfigSpec:
    clusterConfiguration:
      apiServer:
        extraArgs:
          cloud-provider: aws
          {{- if .EnablePodIdentity }}
          service-account-key-file: /etc/kubernetes/pki/sa-signer-pkcs8.pub
          service-account-signing-key-file: /etc/kubernetes/pki/sa-signer.key
          service-account-issuer: https://{{ .BucketName }}.s3.amazonaws.com
          api-audiences: sts.amazonaws.com
          {{- end }}
      controllerManager:
        extraArgs:
          cloud-provider: aws
    initConfiguration:
      nodeRegistration:
        kubeletExtraArgs:
          cloud-provider: aws
        name: {{ `"{{ ds.meta_data.local_hostname }}"` }}
    joinConfiguration:
      nodeRegistration:
        kubeletExtraArgs:
          cloud-provider: aws
        name: {{ `"{{ ds.meta_data.local_hostname }}"` }}
    {{- if .EnablePodIdentity }}
    preKubeadmCommands:
    - aws s3 cp s3://{{ .BucketName }}/sa-signer-pkcs8.pub /etc/kubernetes/pki/sa-signer-pkcs8.pub
    - aws s3 cp s3://{{ .BucketName }}/sa-signer.key /etc/kubernetes/pki/sa-signer.key && chmod 0600 /etc/kubernetes/pki/sa-signer.key
    {{- end }}
  machineTemplate:
    infrastructureRef:
      apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
      kind: AWSMachineTemplate
      name: {{ .Name }}-cp
  replicas: {{ .ControlPlane.Replicas }}
  version: {{ .Version }}
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
kind: AWSMachineTemplate
metadata:
  name: {{ .Name }}-cp
  namespace: {{ .Namespace }}
spec:
  template:
    spec:
      {{- if .ControlPlane.ImageOS }}
      imageLookupBaseOS: {{ .ControlPlane.ImageOS }}
      {{- end }}
      iamInstanceProfile: control-plane{{ $.StackSuffix }}
      instanceType: {{ .ControlPlane.InstanceType }}
      {{- if .ControlPlane.SSHKey }}
      sshKeyName: {{ .ControlPlane.SSHKey }}
      {{- else }}
      sshKeyName: ""
      {{- end }}
{{- range $index, $workerInstance := .Workers}}
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: {{ $.Name }}-md-{{ $index }}
  namespace: {{ $.Namespace }}
spec:
  clusterName: {{ $.Name }}
  replicas: {{ $workerInstance.Replicas }}
  selector:
    matchLabels: null
  template:
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: KubeadmConfigTemplate
          name: {{ $.Name }}-md-{{ $index }}
      clusterName: {{ $.Name }}
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
        kind: AWSMachineTemplate
        name: {{ $.Name }}-md-{{ $index }}
      version: {{ $.Version }}
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
kind: AWSMachineTemplate
metadata:
  name: {{ $.Name }}-md-{{ $index }}
  namespace: {{ $.Namespace }}
spec:
  template:
    spec:
      {{- if $workerInstance.ImageOS }}
      imageLookupBaseOS: {{ $workerInstance.ImageOS }}
      {{- end }}
      iamInstanceProfile: nodes{{ $.StackSuffix }}
      instanceType: {{ $workerInstance.InstanceType }}
      {{- if $workerInstance.SSHKey }}
      sshKeyName: {{ $workerInstance.SSHKey }}
      {{- else }}
      sshKeyName: ""
      {{- end }}
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: KubeadmConfigTemplate
metadata:
  name: {{ $.Name }}-md-{{ $index }}
  namespace: {{ $.Namespace }}
spec:
  template:
    spec:
      joinConfiguration:
        nodeRegistration:
          kubeletExtraArgs:
            cloud-provider: aws   
          name: {{ `"{{ ds.meta_data.local_hostname }}"` }}
{{- end }}
