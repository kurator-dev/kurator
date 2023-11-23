---
title: "Kubernetes Cluster Management: Kurator or Kubespray"
date: 2023-05-27
linkTitle: "Kubernetes Cluster Management: Kurator or Kubespray"
---

As cloud computing technology rapidly evolves, Kubernetes has become the de facto standard in the container orchestration domain. 
Users can automate the deployment, scaling, and management of containerized applications through Kubernetes. 
However, creating highly reliable Kubernetes clusters in different cloud environments can be complex and time-consuming. 
In response, many users have started looking for tools that can automate the deployment and management of Kubernetes clusters, 
with Kubespray and Kurator being typical representatives of such open-source tools. 
This article will compare these two tools.

## Kubespray

Kubespray is an open-source project aimed at helping users deploy and manage Kubernetes clusters in multi-cloud environments. 
To achieve this, Kubespray utilizes Ansible, a trusted open-source automation tool used for automated application deployment, configuration management, and task execution.
Based on this, Kubespray can deploy on various cloud platforms such as AWS, GCE, Azure, OpenStack, as well as on bare metal hardware. Additionally, Kubespray has the following advantages:

- Support for high-availability clusters

- Composable properties like network plugins

- Support for various popular Linux distributions

- Continuous integration testing

Using Kubespray, users can choose to execute an Ansible script, 
which then communicates with each target host via SSH protocol and performs tasks such as cluster deployment, cleanup, and upgrades based on the script.

{{< image width="100%"
    link="./image/kubespray-arch.svg"
    >}}

## Kurator

Kurator, developed by a cloud-native team, draws from years of excellent practice in the field of distributed cloud-native technology. 
While Kurator manages the lifecycle of local data center clusters based on Kubespray, 
its main difference lies in the more understandable and configurable cloud-native declarative approach to cluster management.
Specifically, Kurator has designed declarative APIs to express the desired state of a Kubernetes cluster (such as cluster version, node scale, network configuration, etc.) and manages the cluster lifecycle through the Cluster Operator. This approach greatly simplifies user operations: users only need to declare the desired state in the API object, and all remaining tasks can be automatically completed by Kurator's Cluster Operator. In distributed cloud scenarios, this declarative method provides higher automation and better scalability, making management and operation more convenient and efficient.

The following example shows how to use Kurator to deploy a local data center Kubernetes cluster:

1. Install Kurator's Cluster Operator on machines with an already installed Kubernetes cluster.
2. Create a secret containing SSH keys (an object in Kubernetes used to store sensitive data).
3. Declare CRDs (Custom Resource Definitions) containing machine and cluster information.
4. Apply these CRDs to the cluster, and the Cluster Operator will start the automatic installation of the target cluster.

**Here are some CRD instances used in the above steps:**

- CustomMachine for declaring target cluster host information

- CustomCluster for declaring cluster properties like network plugins and high availability

- KubeadmControlPlane for declaring cluster control plane configurations

The following code example shows how to define 'CustomMachine' and 'CustomCluster'. 

```console
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: CustomMachine
metadata:
  name: cc-custommachine
  namespace: default
spec:
  master:
    - hostName: master1
      publicIP: 200.x.x.1 
      privateIP: 192.x.x.1 
      sshKey:
        apiVersion: v1
        kind: Secret
        name: cluster-secret
  node:
    - hostName: node1
      publicIP: 200.x.x.2
      privateIP: 192.x.x.2
      sshKey:
        apiVersion: v1
        kind: Secret
        name: cluster-secret
  ...
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: CustomCluster
metadata:
  name: cc-customcluster
  namespace: default
spec:
  cni:
    type: cilium
  controlPlaneConfig:
    address: 192.x.x.0
    certSANs: [200.x.x.1,200.x.x.2]
  machineRef:
    apiVersion: cluster.kurator.dev/v1alpha1
    kind: CustomMachine
    name: cc-custommachine
    namespace: default
  ...
```

### Implementation details of the Cluster Operator

In a Kubernetes environment, once these custom resources (CRs) are created or updated, related events are sent to the API server. 
The Cluster Operator listens to these CR-related events and creates corresponding manager workers based on the desired state (defined by the spec fields) to adjust the state, i.e., 
from the current state to the desired state.

These manager workers, created and managed by the Cluster Operator, are special Pods capable of executing the corresponding Ansible cluster management commands to adjust the state.
Depending on the difference between the current state and the desired state, the Cluster Operator creates different workers for different adjustments. 
The Cluster Operator also monitors the completion of these workers to confirm task completion and state updates.

Apart from cluster creation and deletion, processes like cluster node scaling and upgrades follow the same procedure. 
For example, to upgrade, one simply modifies the version field in the KubeadmControlPlane. 
Similarly, for scaling, one just adds or deletes node information fields in CustomMachine.

The entire process is illustrated in the following diagram:

{{< image width="100%"
    link="./image/customcluster-controller.svg"
    >}}

## Comparison of Kubespray and Kurator

Both Kurator and Kubespray can deploy production-ready Kubernetes clusters in various cloud environments. 
Kubespray has the previously mentioned advantages, including high-availability options. Kurator inherits Kubespray's capabilities and also supports high availability. 
However, these two differ significantly in technical implementation, user experience, and project vision and community.

### Technical Implementation

In Kubespray, cluster configuration is mainly done through inventory files and variable files.
The inventory file defines the hosts that Ansible needs to manage, while the variable file customizes the Kubernetes cluster. 
In terms of cluster management, Kubespray relies on executing a series of Ansible playbook commands. 
Each type of cluster operation has a corresponding Ansible script, covering cluster deployment, scaling, upgrades, and lifecycle management. 
Executing these scripts requires using the Ansible-playbook command, including the script, access permissions, and other parameter information.

In contrast, Kurator's implementation method differs. In Kurator, all configuration information is unified in API objects.
This means users do not need to manage these configurations from Ansible's perspective but through declaring API object states. 
For example, once a user declares the desired state of an API object, the Cluster Operator automatically triggers and executes the corresponding operation. 
Users do not need to know the specific Ansible scripts used, making the operation more concise and intuitive.

Overall, while Kubespray provides more flexible customization, 
Kurator offers a simpler and more intuitive cluster configuration and management method with better scalability in a cloud-native environment.

### User Experience

As mentioned above, in Kubespray, users need to configure inventory files and variable files, adjust, and execute corresponding Ansible commands based on their needs.

Kurator, on the other hand, uses declarative configuration to manage local Kubernetes clusters, aligning highly with Kubernetes' core design philosophy.
Therefore, for Kubernetes users, Kurator is easier to understand and has a lower learning curve. 
Compared to Kurator's approach of merely describing the desired state, Kubespray's use of Ansible commands may pose a learning challenge for users without Ansible experience.

Additionally, by using API objects, Kurator can save cluster information and management operation records, facilitating user review and tracking.
Since the current cluster information is preserved, Kurator can also perform pre-checks before executing operations. 
For instance, when a user wants to upgrade the Kubernetes version of a cluster, Kurator can judge the appropriateness of the upgrade span before starting the operation.

With the Operator pattern, Kurator can automate the creation and management of clusters.
If a cluster operation fails, users can delete the erroneous worker, and Kurator will immediately automatically create a new,
functionally identical cluster management worker, ensuring operation idempotency, i.e., repeated operations do not change the system state.

This highly automated approach helps reduce the time and cost of manual intervention, thereby lowering the incidence of human error and improving overall efficiency.

### Project Vision and Community

In terms of community vision, Kubespray is positioned to deploy production-ready Kubernetes clusters in various cloud environments 
and does not focus on managing clusters in distributed cloud environments. 
In contrast, Kurator aims to be a one-click distributed cloud-native suite. 
Beyond supporting cluster deployment in various cloud environments, 
Kurator's broader goal is to help users build a personalized distributed cloud-native infrastructure 
to support business distributed upgrades across different cloud and edge environments. 
Therefore, Kurator's latest version introduces the concept of "fleet" to provide the ability to consistently manage clusters distributed in any cloud environment.

Additionally, there is a clear difference in the stages of community development. 
Kubespray is currently very active and mature, with a large number of contributors and users. 
In contrast, Kurator is still in its early stages but is full of potential and innovation. 
The Kurator community gathers experienced open-source project contributors and highly values and respects the discussions and contributions of each participant. 
For newcomers to Kubernetes, Kurator can help easily create clusters and integrate common open-source tools, facilitating a better experience with cloud-native technology. 
For developers seeking breakthroughs and innovation, Kurator also offers ample space for exploration.

## Conclusion and Outlook

The following comparison table summarizes the differences between Kubespray and Kurator in various aspects:

| Comparison Aspect | Kubespray | Kurator |
| ----------------- | --------- | ------- |
| Underlying Implementation | Ansible | Ansible |
| Multi-Cloud Environment Cluster Deployment | ✔️ | ✔️ |
| Reliability | High | High |
| Cluster Configuration and Management | Inventory/Variable Files + Ansible Commands | API Objects + Automated Cluster Operator |
| User Friendliness | High customization capability, possibly requiring more learning | Simplified configuration management, reducing learning costs |
| Community | Active and mature community | Early stage, full of innovation and development potential |
| Suitable Scenarios | Deploying and managing clusters | Building distributed cloud platforms |

From the comparison above, Kubespray and Kurator each have their strengths and characteristics. 
Kubespray is a mature project with an active community and a high degree of cluster customization. 
However, Kurator offers more simplified and user-friendly configuration management, making it easier for users to get started and use. 
Although Kurator's community is currently not as large as Kubespray's, it is full of innovation and development potential. 
Therefore, whether you are a newcomer to Kubernetes or a developer seeking innovation and potential, Kurator can meet your needs.

In future development plans, Kurator will further strengthen its management of the Kubernetes cluster lifecycle, 
enhancing cluster management capabilities, providing cross-cloud, cross-region, cross-cluster unified and consistent policy management to ensure security and compliance
