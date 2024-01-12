---
title: "Pipeline"
linkTitle: "Pipeline"
weight: 5
description: >
  Instructions on how to create a pipeline easily with Kurator.
---

Kurator efficiently provides a practical solution for the setup and management of CI/CD pipelines in cloud-native applications, significantly simplifying the inherent complexities of pipeline management.
The feature seamlessly integrates with [Tekton](https://tekton.dev/), providing a range of pre-configured Pipeline task templates while also allowing for the creation of custom tasks. 
Designed to be user-friendly, it is suitable for both beginners and advanced users, offering an efficient and straightforward experience in pipeline setup and management.

## Main Benefits

- **Simplified Pipeline Creation**: Offers a user-friendly interface to effortlessly set up CI/CD pipelines.
  Users only need to describe their pipeline in a single Kurator pipeline configuration.
  Kurator automatically handles the creation of necessary resources and the construction of the pipeline.
  Subsequently, users can configure relevant secret settings and set up event-triggered services routing to webhooks as needed.
  This integration with Tekton simplifies the previously complex setup process, reducing the learning curve for cloud-native development.
  The entire process is illustrated in the diagram below.

{{< image width="100%"
link="./image/use-pipeline.svg"
>}}

- **Pre-configured Pipeline Templates**: Provides a variety of ready-to-use pipeline templates based on best practices for common CI/CD scenarios, streamlining the initial setup process.

- **Customization of Tasks**: Enables users to tailor their pipelines with both common preset CI tasks and custom tasks, catering to a broad range of needs, enhancing accessibility and adaptability to various skill levels.

- **Enhanced Operational Control and Visibility**: Expands Kurator CLI capabilities, allowing users to list pipelines and view aggregated logs from the actual execution pods.

- **Enhanced Security and Compliance**: Integrates Tekton Chains for automated artifact builds, signature additions, and provenance proof, aligning with Software Supply Chain Level for Software Artifacts (SLSA) standards. It also supports automatic synchronization and uploading of images to repositories.

## Architecture

Below is the architecture diagram of the Kurator Pipeline feature.

{{< image width="100%"
link="./image/pipeline-arch.svg"
>}} 
