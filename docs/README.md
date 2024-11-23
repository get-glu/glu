# Welcome <!-- {docsify-ignore-all} -->

> Glu is progressive delivery as code. Build, test, and visualize your pipelines locally, then deploy them to your environments.


Glu is a framework written and consumed in the Go programming language.
It is intended to glue together configuration repositories and artifact sources using code. You write your deployment pipelines in code and Glu will orchestrate them across your environments.

It runs in your existing CI/CD environments (GitHub Actions, GitLab CI, etc.) and integrates with your existing source control management systems (Git).

<p align="center">
  <img src="/images/dashboard.png" alt="Glu Dashboard" title="Glu Dashboard" width="75%" />
</p>

It includes an optional UI for visualizing and interacting with your deployment pipelines across multiple environments in a single pane of glass.

## Why Glu?

- Deployment pipelines today are increasingly complex and brittle. Deployment scripts are often written in a variety of languages (Bash, Python, etc.) and are difficult to maintain.
- Tracking deployments across multiple environments is difficult. Answering questions like "What is deployed where?" is cumbersome.
- Tools such as ArgoCD, Flux, and others exist, but they solve a small part of the problem, are often very opinionated, and are Kubernetes-centric.
- YAML sucks for defining deployment pipelines.

## Goals

The goals outlined here are defined to set a target for the project to work towards.
Glu has not yet reached all of these goals, but hopefully, it gives an idea of our intended direction.

They're also not immutable and we would love your help in shaping them.

- Allow engineers to define application delivery across multiple environments as `code`.
- Create a single pane of glass for inspecting and interacting with promotion pipelines.
- Support defining complex rules to guard automated promotion using strategies such as smoke testing and performance evaluation.
- Simple and quick integration with common existing CI/CD and source control management platforms.
- Expose standardized telemetry for the entire end-to-end delivery of changes in your system.

<!-- TODO: add screenshots and more content about the problem we're solving -->
