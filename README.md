Glu - Progressive delivery that sticks
--------------------------------------

Glu is the missing piece in your CI/CD toolbelt.
It is a framework for orchestrating, manipulating and introspecting the state of configuration Git repositories.

![Glu Illustration Diagram](./glu.svg)

## Use Cases

Use it to implement anything that involes automating updates to Git repositories via commits and pull-requests.

### Docker and OCI image version updates (think Renovate / Kargo)

TODO(georgemac): Give demonstration (e.g. with FluxCD for the delivery component).
Demonstrate how the API can be used for introspection of pipeline state.

## Development

### Glu Framework

The core framework is implemented in Go and is designed to be embedded in your own application.

// TODO(mark): more details.

### Glu UI

The Glu UI is a React application that allows you to view and interact with the state of your pipelines.

```
cd ui
npm install
npm start
```

This will start a local server which can be viewed in your browser at http://localhost:1234.

## Roadmap

In the future we plan to support more use-case, such as:

- (In Progress) Progressive delivery (think Kargo / Argo Rollouts)
