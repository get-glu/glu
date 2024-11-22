# Guides

This section contains guides for exploring pre-built Glu pipelines in action.

## Glu Orchestrated GitOps Pipeline

In this guide we will create a GitOps pipeline for deploying some simple applications to multiple "environments".
For the purposes of keeping the illustration simple to create (and destroy) our environments are simply two separate pods in the same cluster. However, Glu can definitely extend to multiple namespaces, clusters or even non-Kubernetes deployment environments.

In this guide we will:

- Deploy a bunch of applications and CD components to a Kubernetes-in-Docker cluster on our local machine.
- Visit our applications in their different "environments" to see what is deployed.
- Look at the Glu pipeline UI to see the latest version of our application, as well as the versions deployed to each environment.
- Trigger a promotion in the UI to update one target environment, to version it is configured to promote from.
- Observe that our promoted environment has updated to the new version.

All the content in this guide works around our [GitOps Example Repository](https://github.com/get-glu/gitops-example).
You can follow the README there, however, we will go over the steps again here in this guide.

The repository contains a script, which will:

- Spin up a Kind cluster to run all of our components in.
- Deploy FluxCD into the cluster to perform CD for our environments.
- Add two Flux Git sources pointed at two seperate "environment" folders (`staging` and `production`).
- Flux will then deploy our demo application into both these "environments".
- Deploy a Glu implemented pipeline for visualizing and progressing promotions across our environments.

### Requirements

- [Docker](https://www.docker.com/)
- [Kind](https://kind.sigs.k8s.io/)
- [Go](https://go.dev/)

> The start script below will also install [Timoni](https://timoni.sh/) (using `go install` to do so).
> This is used to configure our Kind cluster.
> Big love to Timoni :heart:.

### Running

Before you get started you're going to want to do the following:

1. Fork [this repo](https://github.com/get-glu/gitops-example)!
2. Clone your fork locally.
3. Make a note of your forks GitHub URL (likely `https://github.com/{your_username}/gitops-example`).
4. Generate a [GitHub Personal Access Token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens) (if you want to experiment with promotions).

> You will need at-least read and write contents scope (`contents:write`).

Once you have done the above, you can run the start script found in the root of your local fork.
The script will prompt you for your forks repository URL and access token (given you want to perform promotions).

```console
./start.sh
```