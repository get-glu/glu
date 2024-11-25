# Guides

This section contains guides for exploring pre-built Glu pipelines in action.

## Glu Orchestrated GitOps Pipeline

In this guide, we will create a GitOps pipeline for deploying some simple applications to multiple "environments".
For the purposes of keeping the illustration simple to create (and destroy), our environments are simply two separate pods in the same cluster. However, Glu can extend to multiple namespaces, clusters, or even non-Kubernetes deployment environments.

In this guide, we will:

- Deploy a bunch of applications and CD components to a Kubernetes-in-Docker cluster on our local machine.
- Visit our applications in their different "environments" to see what is deployed.
- Look at the Glu pipeline UI to see the latest version of our application, as well as the versions deployed to each environment.
- Trigger a promotion in the UI to update one target environment, to version it is configured to promote from.
- Observe that our promoted environment has updated to the new version.

All the content in this guide works around our [GitOps Example Repository](https://github.com/get-glu/gitops-example).
You can follow the README there, however, we will go over the steps again here in this guide.

The repository contains a script, which will:

- Spin up a Kind cluster to run all of our components.
- Deploy FluxCD into the cluster to perform CD for our environments.
- Add two Flux Git sources pointed at two separate "environment" folders (`staging` and `production`).
- Flux will then deploy our demo application into both of these "environments".
- Deploy a Glu-implemented pipeline for visualizing and progressing promotions across our environments.

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
3. Make a note of your fork's GitHub URL (likely `https://github.com/{your_username}/gitops-example`).
4. Generate a [GitHub Personal Access Token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens) (if you want to experiment with promotions).

> You will need at least read and write contents scope (`contents:write`).

Once you have done the above, you can run the start script found in the root of your local fork.
The script will prompt you for your fork's repository URL and access token (given you want to perform promotions).

```console
./start.sh

Enter your target gitops repo URL [default: https://github.com/get-glu/gitops-example]:
Enter your GitHub personal access token [default: <empty> (read-only pipeline)]:
Creating cluster...
```

This process can take a few minutes in total, sit tight!

Once finished, you will be presented with a link to take you to the Glu UI.

```console
##########################################
#                                        #
# Pipeline Ready: http://localhost:30080 #
#                                        #
##########################################
```

Following the link, you should see a view similar to the one in the screenshot below.
From here, we need to select a pipeline to get started.

<img src="/images/guides/gitops-pipeline/dashboard-welcome.png" alt="Dashboard Welcome Screen" />

Head to the pipeline drop down at the top-left of the dashboard.
Then select the `gitops-example-app` pipeline.

<img src="/images/guides/gitops-pipeline/pipeline-selection-dropdown.png" alt="Pipeline selection dropdown" />

You will notice there are three phase "oci", "staging" and "production".

<img src="/images/guides/gitops-pipeline/pipeline-dashboard-view.png" alt="Pipeline dashboard view" />

If we click on a phase (for example, "staging") we can see an expanded view of the phases details in a pane below.

<img src="/images/guides/gitops-pipeline/pipeline-staging-phase.png" alt="Pipeline staging phase expanded view" />

Notice that the "staging" and "production" phases have a `url` label on them.
These contain links to localhost ports that are exposed on our kind cluster.
You can follow these links to see the "staging" and "production" deployed applications.

Click on the URL for the "staging" phase (it should be [https://localhost:30081](https://localhost:30081)).
The page should open in a new tab and show a small welcome screen with some details regarding the deployed application.
Notice we can see details regarding the built version of the application (Git and OCI digests).

<img src="/images/guides/gitops-pipeline/application-landing-page-staging.png" alt="Staging application landing page" />

Navigating back to the Glu pipeline view, we will now visit our "production" application.

<img src="/images/guides/gitops-pipeline/pipeline-production-phase.png" alt="Pipeline view with production url underlined" />

Click on the URL for the "production" phase (it should be [https://localhost:30082](https://localhost:30082)).

<img src="/images/guides/gitops-pipeline/application-landing-page-production.png" alt="Production application landing page" />

This page is similar to our staging application phase.
It contains all the same information, but it lacks all the styles.
This is because "staging" is currently ahead of "production".

We're going to now trigger a promotion.
Head back to the glu dashboard and click on the up arrow in the "production" phase node.

<img src="/images/guides/gitops-pipeline/pipeline-promotion-modal.png" alt="Promotion dialogue" />

You will be presented with a modal, with a cancel and promote button.
Click on `Promote` to trigger a promotion from "staging" to "production".

> Congratulations, a promotion is in progress!

This will have triggered a promotion to take place.
In this particular example, this will have updated a manifest in our forked Git repository.
Try navigating in your browser to your forked repository in GitHub.
You should see a new commit labelled `Update production`.
Clicking on this, you should see a git patch similar to the following.

<img src="/images/guides/gitops-pipeline/git-diff-production.png" alt="Git patch to production deployment manifest" />

Eventually, once FluxCD has caught up with a new revision you should see it take effect in the [production application](https://localhost:30082).

<img src="/images/guides/gitops-pipeline/application-landing-page-production-promoted.png" alt="Git patch to production deployment manifest" />

### Next Steps

This guide has walked through an interactive example of a pipeline implemented in Glu.
In the next guide, we will look into how this particular pipeline is implemented in Go, using the Glu framework.

TODO(georgemac): Write the guide and link it here.
