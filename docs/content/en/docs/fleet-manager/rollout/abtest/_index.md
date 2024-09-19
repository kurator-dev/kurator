---
title: "A/B Testing"
linkTitle: "A/B Testing"
weight: 50
description: >
  A comprehensive guide on Kurator's A/B Testing, providing an overview and quick start guide.
---

## Introduction

A/B Testing is a method of comparing two versions of an application to validate which performs better.
It essentially involves a controlled experiment where users are randomly allocated into groups at the same time, with each group experiencing a different version of the application.
The metrics from their usage are then analyzed to select the superior version based on the results. The A/B Testing can also be used to route selective users to the new version, allowing their real-world feedback on the new release to be gathered.

- **Use Case**: There are two application services with identical backend functionality but different frontend UIs. It is now necessary to validate which UI design leads to a better user experience. In this scenario, A/B Testing should be used to deploy both versions of the service in a live environment. The UI that demonstrates superior user metrics and outcomes can then be selected for full release.
- **Functionality**: Provide configuration of A/B Testing and trigger an A/B Testing on new release.

By allowing users to deploy applications and their A/B Testing configurations in a single place, Kurator streamlines A/B Testing through automated GitOps workflows for unified deployment and validation.
