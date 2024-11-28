<div>
  <img align="left" src="./.github/images/stu.png" alt="Stu - The Glu mascot" width="200" />
  <br>
  <h3>Glu</h3>
  <p>
    <em>
      A Deployment pipeline framework that sticks
    </em>
  </p>
  <p>
    Glu is the missing piece in your CI/CD toolbelt.
    It is a framework for orchestrating, manipulating and introspecting deployment configuration stored in version control.
  </p>
  <br>
</div>

[![Go Reference](https://pkg.go.dev/badge/github.com/get-glu/glu.svg)](https://pkg.go.dev/github.com/get-glu/glu)

## What Is It?

Glu is a framework built to help manage and coordinate multi-environment deployments using configuration stored in Git.

<p align="center">
https://github.com/user-attachments/assets/b6f4d20b-e166-4ec9-8aa3-0c4d5e0cf410
</p>

Glu has an opinionated set of models and abstractions, which when combined, allow you to build consistent command-line and server processes for orchestrating the progression of applications and configuration across target environments.

## Whats Included

- A CLI for interacting with the state of your pipelines.
- An API for interacting with the state of your pipelines.
- An optional UI for visualizing and interacting with the state of your pipelines.

## Use Cases

Use it to implement anything that involes automating updates to Git repositories via commits and pull-requests.

- ✅ Track new versions of applications in source repositories (OCI, Helm etc) and trigger updates to target configuration repositories (Git).
- ⌛️ Coordinate any combination of scheduled, event driven or manually triggered promotions from one environment to the next.
- 🔍 Expose a single pane of glass to compare and manipulate the state of your resources in one environment to the next.
- 🗓️ Export standardized telemetry which ties together your entire end-to-end CI/CD and promotion pipeline sequence of events.

## Getting Started

1. Install Glu

```
go get github.com/get-glu/glu
```

2. Follow the [GitOps Example Repository](https://github.com/get-glu/gitops-example) to see Glu in action.

3. Implement your own pipelines using the [Glu SDK](https://pkg.go.dev/github.com/get-glu/glu).

## Development

See [DEVELOPMENT.md](./DEVELOPMENT.md) for more information.

## Roadmap Ideas

In the future, we plan to support more functionality, such as:

- New sources:
  - Helm
  - Webhook / API
  - Kubernetes (direct to cluster)
- Progressive delivery (think Kargo / Argo Rollouts)
  - Ability to guard promotion with condition checks on resource status
  - Expose status via Go function definitions on resource types
- Pinning, history, and rollback
  - Ability to view past states for phases
  - Be able to pin phases to current or manually overridden states
  - Rollback phases to previously known states

## Built By

The team at [Flipt](https://flipt.io). We built Glu to power our own internal promotion pipelines and open sourced it so that others can benefit from it.
