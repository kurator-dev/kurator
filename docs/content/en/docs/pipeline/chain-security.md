---
title: "Supply Chain Security"
linkTitle: "Supply Chain Security"
weight: 30
description: >
  This document describes how to use Kurator to implement supply chain security for your applications, meeting the SLSA standards about signatures and provenance.
---

This feature incorporates Tekton Chains to automatically add signatures and provide provenance following the building of artifacts
It aligns with the Software Supply Chain Level for Software Artifacts (SLSA) standards, supporting automatic synchronization and image uploads to repositories. 

The following sections will guide you through constructing images with Kurator, 
automatically signing them, uploading signatures and provenance proofs, and utilizing them.

## Prerequisites

### Creating a Test Namespace

Create a dedicated namespace in Kubernetes for pipeline resources.

```console
kubectl create ns kurator-pipeline
```

### Generating Encryption Keys

Install the cosign tool, following the instructions at [Cosign Installation](https://docs.sigstore.dev/system_config/installation/). 
Cosign, used for signing and verifying in the pipeline, is a key component in ensuring the integrity and security of your pipeline.

After installation, generate encryption keys with the following command:

```console
cosign generate-key-pair k8s://tekton-chains/signing-secrets
```

During this process, you'll be prompted to enter a password twice. 
For testing purposes, you can enter spaces. 
This command creates a key pair in the namespace and generates a public key file `cosign.pub` in your local directory.

### Configuring Image Repository Authentication

Unlike the previous document, this guide includes image construction and uploading, requiring additional permissions for the image repository. 
We use GitHub's `ghcr.io` as the image repository for testing.

#### Docker Login

Log in to Docker to obtain the authentication file `config.json`.

```console
docker login ghcr.io -u <username> -p <your-PAT>
```

Upon successful login, Docker will store your password in `/root/.docker/config.json`. 
This file will be used to create the required Kubernetes secrets for image repository authentication.

#### Creating Secrets for Task Image Repository Authentication

Create a secret for tasks to upload images to the OCI repository:

```console
kubectl create secret generic docker-credentials --from-file=/root/.docker/config.json -n kurator-pipeline
```

This secret, used as a workspace parameter in tasks, grants authentication access.

#### Creating Secrets for Chain Controller

Create a secret for the chain controller to upload signatures(sig) and attestation (att) to the OCI repository:

```console
kubectl create secret generic chain-credentials \
    --from-file=.dockerconfigjson=/root/.docker/config.json \
    --type=kubernetes.io/dockerconfigjson \
    -n kurator-pipeline
```

### Configuring Tekton Chains Parameters

To ensure proper functioning of Tekton Chains components, apply the following configurations:

```console
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"artifacts.taskrun.format": "slsa/v1"}}'
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"artifacts.taskrun.storage": "oci"}}'
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"artifacts.oci.storage": "oci"}}'
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"transparency.enabled": "true"}}'
```

With these steps completed, the preliminary setup is ready, paving the way for secure and efficient pipeline operations for supply chain security.

## Triggering the Test Pipeline

### Creating Kurator Pipeline Test Example

Create a pipeline that includes an image task using the following command:

```console
echo 'apiVersion: pipeline.kurator.dev/v1alpha1
kind: Pipeline
metadata:
  name: quick-start
  namespace: kurator-pipeline
spec:
  description: "this is a quick-start pipeline, it shows how to use customTask and predefined Task in a pipeline"
  tasks:
    - name: git-clone
      predefinedTask:
        name: git-clone
        params:
          git-secret-name: git-credentials
    - name: cat-readme
      customTask:
        image: zshusers/zsh:4.3.15
        command:
          - /bin/sh
          - -c
        args:
          - "cat $(workspaces.source.path)/README.md"
    - name: go-test
      predefinedTask:
        name: go-test
        params:
          packages: ./...
    - name: go-lint
      predefinedTask:
        name: go-lint
        params:
          packages: "./..."
          flags: "--disable=errcheck,unused,gosimple,staticcheck --verbose --timeout 10m"
    - name: build-and-push-image
      predefinedTask:
        name: build-and-push-image
        params:
          image: "<image uri>"'| kubectl apply -f -
```

Replace `<image uri>` with your image uniform resource identifier, like `ghcr.io/myName/kurator-test:0.4.1`.

### Exposing Service

Similar to the previous pipeline, expose the services automatically created by this pipeline, more details about this service can be found in [Setting Up Your Pipeline](https://kurator.dev/docs/pipeline/setting/).

```console
kubectl port-forward --address 0.0.0.0 service/el-quick-start-listener 30002:8080 -n kurator-pipeline
```

### Configuring Webhook

Set up a webhook for this pipeline as well, following the detailed process in [Setting Up Your Pipeline](https://kurator.dev/docs/pipeline/setting/).

### Triggering the Pipeline

To trigger the pipeline, consider pushing some content to the repository, such as a modification to the README file. 
The information about the received event can be observed in the terminal where the port-forwarding service is running:

```console
Forwarding from 0.0.0.0:30002 -> 8080
Handling connection for 30002
```

### Viewing Pipeline Execution Results

After the pipeline is triggered, individual pods will be created for each task in the pipeline, executing them sequentially. 
You can view the status of each task's execution with specific commands.

Similarly, after obtaining the pipeline execution name from the command:

```console
kurator pipeline execution list -n kurator-pipeline --kubeconfig /root/.kube/kurator-host.config
```

You can view the execution logs of each task in the pipeline using the following command:

```console
$ kurator pipeline execution logs <pipeline-execution>  -n kurator-pipeline --tail 10 --kubeconfig /root/.kube/kurator-host.config
INFO[2024-01-04 11:47:34] Fetching logs for TaskRun: quick-start-run-frb7l-git-clone 
INFO[2024-01-04 11:47:34] Fetching logs for container 'step-clone' in Pod 'quick-start-run-frb7l-git-clone-pod' 
INFO[2024-01-04 11:47:34] Logs from container 'step-clone':
+ cd /workspace/source/
+ git rev-parse HEAD
+ RESULT_SHA=1858f8e5129516d6e7d9ad993b1ec41cef922d18
+ EXIT_CODE=0
+ '[' 0 '!=' 0 ]
+ git log -1 '--pretty=%ct'
+ RESULT_COMMITTER_DATE=1703581193
+ printf '%s' 1703581193
+ printf '%s' 1858f8e5129516d6e7d9ad993b1ec41cef922d18
+ printf '%s' <image uri> 
INFO[2024-01-04 11:47:34] Fetching logs for TaskRun: quick-start-run-frb7l-build-and-push-image 
INFO[2024-01-04 11:47:34] Fetching logs for container 'step-build-and-push' in Pod 'quick-start-run-frb7l-build-and-push-image-pod' 
INFO[2024-01-04 11:47:34] Logs from container 'step-build-and-push':
INFO[0161] RUN chown -R app:app ./                      
INFO[0161] Cmd: /bin/sh                                 
INFO[0161] Args: [-c chown -R app:app ./]               
INFO[0161] Running: [/bin/sh -c chown -R app:app ./]    
INFO[0161] Taking snapshot of full filesystem...        
INFO[0162] USER app                                     
INFO[0162] Cmd: USER                                    
INFO[0162] CMD ["./podinfo"]                            
INFO[0162] Pushing image to <image uri>
INFO[0188] Pushed <image uri>@sha256:73c1ad5046233adb70aae2ee5df6e00f2c521e89cc980a954dc024d12add8daf  
INFO[2024-01-04 11:47:34] Fetching logs for container 'step-write-url' in Pod 'quick-start-run-frb7l-build-and-push-image-pod' 
INFO[2024-01-04 11:47:34] Logs from container 'step-write-url':
<image uri>
INFO[2024-01-04 11:47:34] Fetching logs for TaskRun: quick-start-run-frb7l-cat-readme 
INFO[2024-01-04 11:47:34] Fetching logs for container 'step-cat-readme-quick-start' in Pod 'quick-start-run-frb7l-cat-readme-pod' 
INFO[2024-01-04 11:47:34] Logs from container 'step-cat-readme-quick-start':
To delete podinfo's Helm repository and release from your cluster run:
flux -n default delete source helm podinfo
flux -n default delete helmrelease podinfo
If you wish to manage the lifecycle of your applications in a **GitOps** manner, check out
this [workflow example](https://github.com/fluxcd/flux2-kustomize-helm-example)
for multi-env deployments with Flux, Kustomize and Helm. 
INFO[2024-01-04 11:47:34] Fetching logs for TaskRun: quick-start-run-frb7l-go-test 
INFO[2024-01-04 11:47:34] Fetching logs for container 'step-unit-test' in Pod 'quick-start-run-frb7l-go-test-pod' 
INFO[2024-01-04 11:47:34] Logs from container 'step-unit-test':
--- PASS: TestInfoHandler (0.00s)
=== RUN   TestStatusHandler
--- PASS: TestStatusHandler (0.00s)
=== RUN   TestTokenHandler
--- PASS: TestTokenHandler (0.00s)
=== RUN   TestVersionHandler
--- PASS: TestVersionHandler (0.00s)
PASS
coverage: 14.4% of statements
ok  	github.com/stefanprodan/podinfo/pkg/api	1.088s	coverage: 14.4% of statements 
INFO[2024-01-04 11:47:34] Fetching logs for TaskRun: quick-start-run-frb7l-go-lint 
INFO[2024-01-04 11:47:34] Fetching logs for container 'step-lint' in Pod 'quick-start-run-frb7l-go-lint-pod' 
INFO[2024-01-04 11:47:34] Logs from container 'step-lint':
level=info msg="[config_reader] Config search paths: [./ /workspace/src /workspace / /root]"
level=info msg="[lintersdb] Active 2 linters: [govet ineffassign]"
level=info msg="[loader] Go packages loading at mode 575 (deps|imports|types_sizes|compiled_files|files|name|exports_file) took 1m0.435700461s"
level=info msg="[runner/filename_unadjuster] Pre-built 0 adjustments in 4.381259ms"
level=info msg="[linters_context/goanalysis] analyzers took 1.651993451s with top 10 stages: inspect: 842.409271ms, ctrlflow: 402.369985ms, printf: 382.21663ms, ineffassign: 12.079525ms, slog: 3.20068ms, lostcancel: 1.346293ms, copylocks: 1.333706ms, directive: 1.287065ms, bools: 976.125µs, composites: 435.523µs"
level=info msg="[runner] processing took 2.828µs with stages: max_same_issues: 349ns, skip_dirs: 325ns, nolint: 302ns, cgo: 230ns, exclude-rules: 208ns, max_from_linter: 146ns, source_code: 143ns, path_prettifier: 133ns, filename_unadjuster: 133ns, autogenerated_exclude: 131ns, skip_files: 124ns, identifier_marker: 118ns, max_per_file_from_linter: 72ns, severity-rules: 59ns, path_shortener: 56ns, sort_results: 53ns, diff: 51ns, exclude: 50ns, uniq_by_line: 50ns, fixer: 50ns, path_prefixer: 45ns"
level=info msg="[runner] linters took 3.81614172s with stages: goanalysis_metalinter: 3.816076131s"
level=info msg="File cache stats: 0 entries of total size 0B"
level=info msg="Memory: 644 samples, avg is 35.5MB, max is 435.9MB"
level=info msg="Execution took 1m4.265454941s" 
```

This command will display the logs for each task run within the pipeline, allowing you to monitor and verify the execution results and troubleshoot if necessary.

## Supply Chain Security

### Verifying Image and Signatures in Repository

After the signing process is completed successfully, log in to your GitHub account and navigate to the Packages page. 
There, you will find the specified image you've created. Clicking on it reveals details similar to the following:

{{< image width="100%"
link="./image/chain-security.png"
>}}

In the image, you can see that along with the built image from your application repository, signatures (sig) and attestations (att) are also uploaded to the OCI repository.

### Verifying the Signatures

After visiting the `ghcr.io` to view the images and corresponding `.sig` signatures and `.att` attestations under the specified package,
we can verify the signatures using the public key (`cosign.pub`) created in the cosign process:

```console
cosign verify --key cosign.pub <image uri>
cosign verify-attestation --key cosign.pub --type slsaprovenance <image uri>
```

If verification fails, explicit error messages will be displayed (e.g., signature mismatch, invalid forepart). 
If successful, detailed information about the signature, including the docker image, will be shown.

Here's an example of a successful verification output:

```console
$ cosign verify --key cosign.pub <image uri>
Verification for <image uri> --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - The claims were present in the transparency log
  - The signatures were integrated into the transparency log when the certificate was valid
  - The signatures were verified against the specified public key
[{"critical":{"identity":{"docker-reference":"<image uri>"},"image":{"docker-manifest-digest":"sha256:a4e1fb3e11f3c0ad167ed9868b7c6fcfffd7923a61e8bd15fbfdf8cda109cb58"},"type":"cosign container image signature"},"optional":null}]
```

```console
$ cosign verify-attestation --key cosign.pub --type slsaprovenance <image uri>
Verification for <image uri> --
The following checks were performed on each of these signatures:
- The cosign claims were validated
- The claims were present in the transparency log
- The signatures were integrated into the transparency log when the certificate was valid
- The signatures were verified against the specified public key
  {"payloadType":"application/vnd.in-son","payload":"eyJfdHlwZSI...GJlOGE1In19XX19","signatures":[{"keyid":"SHA256:c7r0wGda2...ZO2hTTXTp+RkWI","sig":"MEQCICn0...6+ILoL4g=="}]}
```

The payload above(`"payload":"eyJfdHlwZSI...GJlOGE1In19XX19"`) contains detailed information about the build process of the image `<image uri>`, 
like the builder, the build steps, environments, parameters, and the start and end times of the build.

This information is crucial for adhering to the SLSA security standards, as it provides complete build transparency, 
ensuring traceability and auditability of the build process, thus enhancing the security of the software supply chain. 

These details help verify the integrity and consistency of the build process, increasing trust in the software building and deployment process.

### Decoding the Attestation

To view the attestation, decode the base64 encoded JSON string(replace `'eyJfdHlwZSI...GJlOGE1In19XX19'` with your actual payload value):

```console
$ echo 'eyJfdHlwZSI...GJlOGE1In19XX19' | base64 --decode | jq
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "predicateType": "https://slsa.dev/provenance/v0.2",
  "subject": [
    {
      "name": "<image uri>",
      "digest": {
        "sha256": "a4e1fb3e11f3c0ad167ed9868b7c6fcfffd7923a61e8bd15fbfdf8cda109cb58"
      }
    }
  ],
  "predicate": {
    "builder": {
      "id": "https://tekton.dev/chains/v2"
    },
    "buildType": "tekton.dev/v1beta1/TaskRun",
    "invocation": {
      "configSource": {},
      "parameters": {
        "BUILDER_IMAGE": "gcr.io/kaniko-project/executor:v1.5.1@sha256:c6166717f7fe0b7da44908c986137ecfeab21f31ec3992f6e128fff8a94be8a5",
        "CONTEXT": "./",
        "DOCKERFILE": "./Dockerfile",
        "EXTRA_ARGS": "",
        "IMAGE": "<image uri>"
      },
      "environment": {
        "annotations": {
          "pipeline.tekton.dev/release": "30540fc"
        },
        "labels": {
          "app.kubernetes.io/managed-by": "tekton-pipelines",
          "tekton.dev/task": "kaniko-chains"
        }
      }
    },
...
}
```

The resulting payload contains detailed information about the build process of the `<image uri>`, 
such as the builder used, build steps, environments and parameters used, and the start and end times of the build.

## Cleanup

### Cleaning Up Pipeline

To remove the pipeline examples used for testing, execute:

```console
kubectl delete pipelines.pipeline.kurator.dev  -n kurator-pipeline test-predefined-task test-custom-task
```

> Please note: When the Kurator pipeline is deleted, all the resources it created, including the pods for tasks and the services for event listeners, will be deleted as well.

### Cleaning Up Cosign Secret

Delete the cosign secret with the following command. Note that this secret cannot be altered and must be recreated as needed.

```console
kubectl delete secret signing-secrets -n tekton-chains
```

### Logging Out of Docker

Finally, log out of Docker to ensure security:

```console
docker logout ghcr.io 
```
