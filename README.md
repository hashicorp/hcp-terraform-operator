<a href="https://cloud.hashicorp.com/products/terraform">
    <img src=".github/tf_logo.png" alt="Terraform logo" title="Terraform Cloud" align="left" height="50" />
</a>

# Kubernetes Operator for Terraform Cloud / Enterprise

[![GitHub release (with filter)](https://img.shields.io/github/v/release/hashicorp/terraform-cloud-operator)](https://github.com/hashicorp/terraform-cloud-operator/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/hashicorp/terraform-cloud-operator)](https://hub.docker.com/r/hashicorp/terraform-cloud-operator)
[![GitHub](https://img.shields.io/github/license/hashicorp/terraform-cloud-operator)](https://github.com/hashicorp/terraform-cloud-operator/blob/main/LICENSE)


> **Warning**
> Please note that this is a beta version still undergoing final testing before the official release.

Kubernetes Operator allows managing Terraform Cloud / Enterprise resources via Kubernetes Custom Resources.

> **Note**
> _From this point forward, the terms Terraform Cloud or TFC can be used interchangeably with Terraform Enterprise or TFE in all documents, provided that the contrary is indicated._

The Operator can manage the following types of resources:

- `AgentPool` manages [Terraform Cloud Agent Pools](https://developer.hashicorp.com/terraform/cloud-docs/agents/agent-pools) and [Terraform Cloud Agent Tokens](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens#agent-api-tokens)
- `Module` implements [API-driven Run Workflows](https://developer.hashicorp.com/terraform/cloud-docs/run/api)
- `Workspace` manages [Terraform Cloud Workspaces](https://developer.hashicorp.com/terraform/cloud-docs/workspaces)

## :warning: Beta :warning:

Welcome to the `v2-beta` version of the Kubernetes Operator for Terraform Cloud the successor of [`v1`](https://github.com/hashicorp/terraform-k8s). This new iteration of the Operator is being developed based on feedback from issues that practitioners have experienced with the first version.

We are still working on finalizing the project's life-cycle processes and thus we ask you to use the following instructions to install and upgrade the Helm chart for the Operator beta. The rest instructions remain valid.

Please, take into account the beta stage of this project and DO NOT use it in your production or critical environment. It has not been "battle tested" yet.

We deeply appreciate everyone who is participating in the Operator beta and looking forward to hearing your feedback.

## Getting started

To get started see our tutorials on the HashiCorp Developer Portal:

- [Deploy infrastructure with the Terraform Cloud Kubernetes Operator v2](https://developer.hashicorp.com/terraform/tutorials/kubernetes/kubernetes-operator-v2)
- [Manage agent pools with the Terraform Cloud Kubernetes Operator v2](https://developer.hashicorp.com/terraform/tutorials/kubernetes/kubernetes-operator-v2-agentpool)
- [Terraform Cloud Kubernetes Operator v2 Migration Guide](https://developer.hashicorp.com/terraform/cloud-docs/integrations/kubernetes/ops-v2-migration)

## Documentation

### Supported Features

The full list of supported Terraform Cloud features can be found [here](./docs/features.md).

### Installation

The Operator provides [Helm chart](../charts/terraform-cloud-operator) as a first-class method of installation on Kubernetes.

Three simple commands to install the Operator:

```console
$ helm repo add hashicorp https://helm.releases.hashicorp.com
$ helm repo update
$ helm install demo hashicorp/terraform-cloud-operator --wait --version 2.0.0-beta7
```

More detailed information about the installation and available values can be found [here](../charts/terraform-cloud-operator).

### Usage

General usage documentation can be found [here](./docs/usage.md).

Controllers usage guides:

- [AgentPool](./docs/agentpool.md)
- [Module](./docs/module.md)
- [Workspace](./docs/workspace.md)

### Metrics

The Operator exposes metrics in the [Prometheus](https://prometheus.io/) format for each controller. More information can be found [here](./docs/metrics.md).

### API reference

API reference documentation can be found [here](./docs/api-reference.md).

### Frequently Asked Questions

FAQ can be found [here](./docs/faq.md).

### Examples

YAML manifests examples live [here](./docs/examples/).

## Operator Options

Global options:

- `sync-period` -- the minimum frequency at which watched resources are reconciled. Format: 5s, 1m, etc. Default: `5m`.
- `namespace` -- Namespace to watch. Default: `watch all namespaces`.

`AgentPool` controller has the following options:

- `agent-pool-workers` -- the number of the Agent Pool controller workers. Default: `1`.

`Module` controller has the following options:

- `module-workers` -- the number of the Module controller workers. Default: `1`.

`Workspace` controller has the following options:

- `workspace-workers` -- the number of the Workspace controller workers. Default: `1`.

In order to change the default values of the options, use the corresponding Helm chart value.

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
    $ kubectl get module <NAME>
    $ kubectl get workspace <NAME>
    ```

- check the CR events:

    ```console
    $ kubectl describe agentpool <NAME>
    $ kubectl describe module <NAME>
    $ kubectl describe workspace <NAME>
    ```

If you believe you've found a bug and cannot find an existing issue, feel free to open a new issue! Be sure to include as much information as you can about your environment.

## Security Reporting

If you think you've found a security vulnerability, we'd love to hear from you.

Follow the instructions in [SECURITY.md](.github/SECURITY.md) to make a report.

## Experimental Status

By using the software in this repository (the "Software"), you acknowledge that: (1) the Software is still in development, may change, and has not been released as a commercial product by HashiCorp and is not currently supported in any way by HashiCorp; (2) the Software is provided on an "as-is" basis, and may include bugs, errors, or other issues; (3) the Software is NOT INTENDED FOR PRODUCTION USE, use of the Software may result in unexpected results, loss of data, or other unexpected results, and HashiCorp disclaims any and all liability resulting from use of the Software; and (4) HashiCorp reserves all rights to make all decisions about the features, functionality and commercial release (or non-release) of the Software, at any time and without any obligation or liability whatsoever.
