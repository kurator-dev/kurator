---
title: "Setting Up Your Pipeline"
linkTitle: "Setting Up Your Pipeline"
weight: 10
description: >
  This document provides a step-by-step guide to setting up your pipeline using Kurator, covering prerequisites, configuration, and task customization.
---

This document provides a step-by-step guide to setting up your pipeline using Kurator, covering prerequisites, configuration, and task customization.

## Prerequisites

### Installing Components

To start using Kurator pipeline, you need to install Tekton in your Kubernetes cluster. Run the following command to install Tekton components:

```
kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml
kubectl apply --filename https://storage.googleapis.com/tekton-releases/chains/latest/release.yaml
```

### Configuring Authentication for Private Repositories

Since the first task in a pipeline often involves pulling code from a Git repository, it's essential to configure Git repository authentication. 
This can be done by creating a Kubernetes secret containing the authentication details. Use the following command to create this secret:

```
kubectl create secret generic git-credentials \
  --namespace=kurator-pipeline \
  --from-literal=.gitconfig=$'[credential "https://github.com"]\n\thelper = store' \
  --from-literal=.git-credentials='https://<username>:<your-PAT>@github.com'
```

## Pipeline Field Introduction

Understanding the core fields of a Pipeline in Kurator is crucial for effective configuration and utilization. 
A Pipeline in Kurator contains a collection of tasks, defining the workflow of a CI/CD process.

Here is an example of a typical pipeline:

```
apiVersion: pipeline.kurator.dev/v1alpha1
kind: Pipeline
metadata:
  name: test-custom-task
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
```

In the Pipeline, tasks are executed in the sequence they are listed. 
Each task encompasses detailed information regarding the specific steps that need to be undertaken. 

The tasks are structured as an array, comprising either predefinedTasks or CustomTasks.
**PredefinedTask** provides users the option to select from a set of predefined tasks and input their parameters. 
Conversely, **CustomTask** offers the flexibility to directly define a task, particularly when the desired task is not available in the list of predefined tasks.


## predefined Tasks

**Pre-configured Pipeline Templates**: These are a variety of ready-to-use pipeline templates based on best practices for common CI/CD scenarios. 
  They streamline the initial setup process and provide a quick start for users.

### Currently Supported predefined Tasks

| Task Name      | Description | Benefits | Core Parameters |
| -------------- | ----------- | -------- | --------------- |
| `git-clone`    | Clones the user's code repository into the workspace, typically the first task in the entire pipeline. | Simplifies the process of pulling code for subsequent tasks. | - `git-secret-name`: Name of the secret for Git authentication. |
| `go-test`      | Runs Go tests in specified packages with configurable environment. | Facilitates testing in Go projects. | - `packages`, `context`, `version`, `flags`, `GOOS`, `GOARCH`, `GO111MODULE`, `GOCACHE`, `GOMODCACHE` |
| `go-lint`      | Performs linting on Go source code, using golangci-lint. | Ensures coding style and common error checks. | - `package`, `context`, `flags`, `version`, `GOOS`, `GOARCH`, `GO111MODULE`, `GOCACHE`, `GOMODCACHE`, `GOLANGCI_LINT_CACHE` |
| `build-and-push-image` | Builds and pushes a Docker image using Kaniko. | Enables building and storing Docker images. | - `IMAGE`, `DOCKERFILE`, `CONTEXT`, `EXTRA_ARGS`, `BUILDER_IMAGE` |

### How to Configure predefined Tasks

To configure a predefined Task, simply reference the task template in your pipeline definition, and provide any required parameters based on your specific requirements. 
For example:

```yaml
tasks:
  - name: git-clone
    predefinedTask:
      name: git-clone
      params:
        git-secret-name: git-credentials
```
## Custom Tasks

### Introduction to Custom Tasks
- **Customization of Tasks**: This feature enables users to tailor their pipelines by incorporating both common predefined CI tasks and custom tasks, catering to a broad range of needs and enhancing adaptability.

### Custom Task Configuration
A Custom Task allows for greater flexibility by defining tasks directly within the pipeline configuration. The key fields to understand in a Custom Task are:

- `Image`: Specifies the Docker image name.
- `Command`: The entrypoint array, not executed in a shell.
- `Args`: Arguments for the entrypoint, used if the Command is not provided.
- `Env`: List of environment variables to set in the task.
- `ResourceRequirements`: Specifies compute resource requirements.
- `Script`: Contains the contents of an executable file to execute.

Example of a Custom Task:

```yaml
tasks:
  - name: custom-task-example
    customTask:
      image: 'example-image'
      command:
        - '/bin/sh'
        - '-c'
      args:
        - 'echo Hello World'
      env:
        - name: EXAMPLE_ENV
          value: 'example'
```
