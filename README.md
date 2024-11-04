Glu - Progressive delivery that sticks
--------------------------------------

Glu is the missing piece in your CI/CD toolbelt.
It is a framework for orchestrating, manipulating and introspecting the state of configuration Git repositories.

![Glu Illustration Diagram](./glu.svg)

## Use Cases

Use it to implement anything that involes automating updates to Git repositories via commits and pull-requests.

### Automated dependency updates (think Dependabot / Renovate)

TODO(georgemac): Give demonstration (potentially implement re-usable package).

### Docker and OCI image version updates (think Renovate / Kargo)

TODO(georgemac): Give demonstration (e.g. with FluxCD for the delivery component).
Demonstrate how the API can be used for introspection of pipeline state.

## Roadmap

In the future we plan to support more use-case, such as:

- (In Progress) Progressive delivery (think Kargo / Argo Rollouts)
