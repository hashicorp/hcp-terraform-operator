<a href="https://cloud.hashicorp.com/products/terraform">
    <img src=".github/hcp-terraform-logo.png" alt="HCP Terraform" title="HCP Terraform" align="left" height="60" />
</a>

# HCP Terraform Operator for Kubernetes

[![GitHub release (with filter)](https://img.shields.io/github/v/release/hashicorp/hcp-terraform-operator)](https://github.com/hashicorp/hcp-terraform-operator/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/hashicorp/hcp-terraform-operator)](https://hub.docker.com/r/hashicorp/hcp-terraform-operator) [![Docker Pulls](https://img.shields.io/docker/pulls/hashicorp/terraform-cloud-operator)](https://hub.docker.com/r/hashicorp/terraform-cloud-operator)
[![GitHub](https://img.shields.io/github/license/hashicorp/hcp-terraform-operator)](https://github.com/hashicorp/hcp-terraform-operator/blob/main/LICENSE)

Kubernetes Operator allows managing HCP Terraform / Terraform Enterprise resources via Kubernetes Custom Resources.

> **Note**
> _From this point forward, the terms HCP Terraform can be used interchangeably with Terraform Enterprise in all documents, provided that the contrary is indicated._

The Operator can manage the following types of resources:

- `AgentPool` manages [HCP Terraform Agent Pools](https://developer.hashicorp.com/terraform/cloud-docs/agents/agent-pools), [HCP Terraform Agent Tokens](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens#agent-api-tokens) and can perform TFC agent scaling
- `AgentToken` manages [HCP Terraform Agent Tokens](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens#agent-api-tokens)
- `Module` implements [API-driven Run Workflows](https://developer.hashicorp.com/terraform/cloud-docs/run/api)
- `Project` manages [HCP Terraform Projects](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/organize-workspaces-with-projects)
- `Runs Collector` Runs scrapes HCP Terraform run statuses from a given Agent Pool and exposes them as Prometheus-compatible metrics. Learn more about [Runs](https://developer.hashicorp.com/terraform/cloud-docs/run/remote-operations).
- `Workspace` manages [HCP Terraform Workspaces](https://developer.hashicorp.com/terraform/cloud-docs/workspaces)

## Getting started

To get started see our tutorials on the HashiCorp Developer Portal:

- [HCP Terraform Operator for Kubernetes overview](https://developer.hashicorp.com/terraform/cloud-docs/integrations/kubernetes)
- [Deploy infrastructure with the HCP Terraform Operator for Kubernetes](https://developer.hashicorp.com/terraform/tutorials/kubernetes/kubernetes-operator-v2)
- [Manage agent pools with the HCP Terraform Operator for Kubernetes](https://developer.hashicorp.com/terraform/tutorials/kubernetes/kubernetes-operator-v2-agentpool)
- [HCP Terraform Operator for Kubernetes Migration Guide](https://developer.hashicorp.com/terraform/cloud-docs/integrations/kubernetes/ops-v2-migration)

## Documentation

### Supported Features

The full list of supported HCP Terraform Operator features can be found on our [Developer portal](https://developer.hashicorp.com/terraform/cloud-docs/integrations/kubernetes#supported-terraform-cloud-features).

### Installation

The Operator provides [Helm chart](./charts/hcp-terraform-operator) as a first-class method of installation on Kubernetes.

Three simple commands to install the Operator:

```console
$ helm repo add hashicorp https://helm.releases.hashicorp.com
$ helm repo update
$ helm install demo hashicorp/hcp-terraform-operator --wait --version 2.9.2
```

More detailed information about the installation and available values can be found [here](./charts/hcp-terraform-operator/README.md).

### Usage

General usage documentation can be found [here](./docs/usage.md).

Controllers usage guides:

- [AgentPool](./docs/agentpool.md)
- [AgentToken](./docs/agenttoken.md)
- [Module](./docs/module.md)
- [Project](./docs/project.md)
- [Runs Collector](./docs/runs_collector.md)
- [Workspace](./docs/workspace.md)


### Annotations and Labels

Annotations and Labels used by HCP Terraform Operator can be found [here](./docs/annotations-and-labels.md).

### Metrics

The Operator exposes metrics in the [Prometheus](https://prometheus.io/) format for each controller. More information can be found [here](./docs/metrics.md).

### API reference

API reference documentation can be found [here](./docs/api-reference.md).

### Frequently Asked Questions

FAQ can be found [here](./docs/faq.md).

### Examples

YAML manifests examples live [here](./docs/examples/).

### Community Contribution

If you come across articles, videos, how-tos, or any other resources that could assist individuals in adopting and utilizing the operator with greater efficiency, kindly inform us by initiating a [pull request](https://github.com/hashicorp/hcp-terraform-operator/pulls) and placing a link within this designated section.

Your participation matters. Thank you for being a part of our community! :raised_hands:

## Troubleshooting

If you encounter any issues with the Operator there are a number of ways how to troubleshoot it:

- check the Operator logs:

    ```console
    $ kubectl logs -f <POD_NAME>
    ```

    Logs for a specific CR can be identified with the following pattern:

    ```json
    {"<KIND>": "<NAMESPACE>/<METADATA.NAME>", "msg": "..."}
    ```

    For example:

    ```text
    2023-01-05T12:11:31Z INFO Agent Pool Controller	{"agentpool": "default/this", "msg": "successfully reconcilied agent pool"}
    ```

- check the CR:

    ```console
    $ kubectl get agentpool <NAME>
    $ kubectl get agenttoken <NAME>
    $ kubectl get module <NAME>
    $ kubectl get project <NAME>
    $ kubectl get runscollector <NAME>
    $ kubectl get workspace <NAME>
    ```

- check the CR events:

    ```console
    $ kubectl describe agentpool <NAME>
    $ kubectl describe agenttoken <NAME>
    $ kubectl describe module <NAME>
    $ kubectl describe project <NAME>
    $ kubectl describe runscollector <NAME>
    $ kubectl describe workspace <NAME>
    ```

If you believe you've found a bug and cannot find an existing issue, feel free to open a new issue! Be sure to include as much information as you can about your environment.

## Contributing to the Operator

We appreciate your enthusiasm for participating in the development of the HCP Terraform Operator. To contribute, please read the [contribution guidelines](./CONTRIBUTING.md).

## Security Reporting

If you think you've found a security vulnerability, we'd love to hear from you.

Follow the instructions in [SECURITY.md](.github/SECURITY.md) to make a report.
