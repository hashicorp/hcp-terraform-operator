<a href="https://cloud.hashicorp.com/products/terraform">
    <img src=".github/tf_logo.png" alt="Terraform logo" title="Terraform Cloud" align="left" height="50" />
</a>

# Kubernetes Operator for Terraform Cloud

> **Warning**
> Please note that this is a beta version still undergoing final testing before the official release.

Kubernetes Operator allows managing Terraform Cloud resources via Kubernetes Custom Resources.

The Operator can manage the following types of resources:

- `Workspace` manages [Terraform Cloud Workspaces](https://developer.hashicorp.com/terraform/cloud-docs/workspaces)
- `Module` implements [API-driven Run Workflows](https://developer.hashicorp.com/terraform/cloud-docs/run/api)

## :warning: Beta :warning:

Welcome to the `v2-beta` version of the Kubernetes Operator for Terraform Cloud the successor of [`v1`](https://github.com/hashicorp/terraform-k8s). This new iteration of the Operator is being developed based on feedback from issues that practitioners have experienced with the first version.

We are still working on finalizing the project's life-cycle processes and thus we ask you to use the following instructions to install and upgrade the Helm chart for the Operator beta. The rest instructions remain valid.

Please, take into account the beta stage of this project and DO NOT use it in your production or critical environment. It has not been "battle tested" yet.

We deeply appreciate everyone who is participating in the Operator beta and looking forward to hearing your feedback.

### Install beta version

In this example, Helm will create a new namespace `tfc-operator-system` and install the Operator to it. The Operator will watch 3 namespaces in the Kubernetes cluster: `white`, `blue`, and `red`. All other Helm values remain with their default values.

```
helm install \
  beta oci://public.ecr.aws/t8q4c9g6/terraform-cloud-operator \
  --version 0.0.1 \
  --namespace tfc-operator-system \
  --create-namespace \
  --set operator.image.repository=public.ecr.aws/t8q4c9g6/terraform-cloud-operator \
  --set operator.image.tag=2.0.0-beta1 \
  --set 'operator.watchedNamespaces={white,blue,red}'
```

> **Note**
> Please pay attention to the repository name, chart version, and image tag.

### Upgrade beta version

In this example, Helm will upgrade the existing operator installation in the `tfc-operator-system` namespace. The Operator will watch all namespaces(default value) in the Kubernetes cluster and run 5 workers for `Module` and `Workspace` controllers. All other Helm values remain with their default values.

```
helm upgrade \
  beta oci://public.ecr.aws/t8q4c9g6/terraform-cloud-operator \
  --version 0.0.1 \
  --namespace tfc-operator-system \
  --set operator.image.repository=public.ecr.aws/t8q4c9g6/terraform-cloud-operator \
  --set operator.image.tag=2.0.0-beta1 \
  --set controllers.module.workers=5 \
  --set controllers.workspace.workers=5
```

> **Note**
> Please pay attention to the repository name, chart version, and image tag.

## Installation

The Operator provides Helm charts as a first-class method of installation on Kubernetes.

### Steps

1. Add the Helm repository
    ```
    helm repo add hashicorp https://helm.releases.hashicorp.com
    ```

2. Update your local Helm chart repository cache
    ```
    helm repo update
    ```

3. Install
    ```
    helm install \
      demo hashicorp/terraform-cloud-operator \
      --namespace tfc-operator-system \
      --create-namespace
    ```

## Usage

### Prerequisites

- The Operator requires a Terraform Cloud [organization](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations) name and a [token](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens) in order to access the Terraform Cloud API.
- The API token must be stored in a Kubernetes secret.
- A single instance of the Operator can manage Terraform Cloud resources for different organizations and/or different API tokens. For that purpose, the organization name and a reference to the corresponding Kubernetes secret are shipped within the custom resource.

Below are examples of how to create a Kubernetes secret and store the API token there. The examples assumes that the API token is already known.

1. `kubectl` command
    ```
    kubectl create secret generic tfc-operator --from-literal=token=APIt0k3n
    ```

2. YAML manifest
    - Encode the API token
        ```
        echo -n "APIt0k3n" | base64
        ```
    - Create a YAML manifest and paste the encoded token from the previous step
        ```yaml
        apiVersion: v1
        kind: Secret
        metadata:
          name: operator
        type: Opaque
        data:
          token: QVBJdDBrM24=
        ```
    - Apply YAML manifest
        ```
        kubectl apply -f secret.yaml
        ```

For more information about Kubernetes secrets please refer to the [Kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/secret/).

Please use the approach that is matching with the best practices which are accepted in your organization.

### `Workspace`

`Workspace` controller allows managing Terraform Cloud Workspace via Kubernetes Custom Resources.

Below is an example of a Workspace Custom Resource:
```yaml
---
apiVersion: app.terraform.io/v1alpha2
kind: Workspace
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: operator
      key: token
  name: kubernetes-operator-demo
  description: Kubernetes Operator Automated Workspace
  applyMethod: auto
  terraformVersion: 1.3.2
  executionMode: remote
  terraformVariables:
    - name: counter
      hcl: true
      value: >
        [
        1,
        2,
        4,
        8,
        16,
        32
        ]
  tags:
    - demo
```

Once the above CR is applied, the Operator creates a new workspace under the `kubernetes-operator` organization.

Non-sensitive outputs of the workspace runs will be saved in Kubernetes ConfigMaps. Sensitive outputs of the workspace runs will be saved in Kubernetes Secrets. In both cases, the name of the corresponding Kubernetes object will be generated automatically and has the following pattern: `<metadata.name>-outputs`. For the above example, the name of ConfigMap and Secret will be `this-outputs`.

Please refer to the  [CRD](https://github.com/hashicorp/terraform-cloud-operator/blob/main/config/crd/bases/app.terraform.io_workspaces.yaml) to get the full list of available options.

### `Module`

`Module` controller allows executing arbitrary Terraform Modules code in Terraform Cloud Workspace via Kubernetes Custom Resources.

Below is an example of a Module Custom Resource:
```yaml
---
apiVersion: app.terraform.io/v1alpha2
kind: Module
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: operator
      key: token
  destroyOnDeletion: true
  module:
    source: app.terraform.io/kubernetes-operator/module-random/provider
    version: 0.0.5
  variables:
  - name: counter
  outputs:
  - name: secret
    sensitive: true
  - name: random_strings
  workspace:
    name: kubernetes-operator-demo
```

The above CR will be transformed to the following terraform code and then executed within the `kubernetes-operator-demo` workspace:

```hcl
variable "counter" {}

module "this" {
  source  = "app.terraform.io/kubernetes-operator/module-random/provider"
  version = "0.0.5"

  counter = var.counter
}

output "secret" {
    value     = module.this.secret
    sensitive = true
}

output "random_strings" {
    value     = module.this.random_strings
}
```

Non-sensitive outputs will be saved in Kubernetes ConfigMaps. Sensitive outputs will be saved in Kubernetes Secrets. In both cases, the name of the corresponding Kubernetes object will be generated automatically and has the following pattern: `<metadata.name>-module-outputs`. For the above example, the name of ConfigMap and Secret will be `this-module-outputs`.

Please note that the `Module` controller does not create a workspace or variables in the referred workspace. They must exist.

In order to restart reconciliation for a particular CR, execute the following command:
```
kubectl patch module <NAME> \
  --type=merge \
  --patch '{"spec": {"restartedAt": "'`date -u -Iseconds`'"}}'
```

Please refer to the  [CRD](https://github.com/hashicorp/terraform-cloud-operator/blob/main/config/crd/bases/app.terraform.io_modules.yaml) to get the full list of available options.

## Operator Options

Global options:

- `sync-period` -- the minimum frequency at which watched resources are reconciled. Format: 5s, 1m, etc. Default: `5m`.
- `namespace` -- Namespace to watch. Default: `watch all namespaces`.

`Workspace` controller has the following options:

- `workspace-workers` -- the number of the workspace controller workers. Default: `1`.

`Module` controller has the following options:

- `module-workers` -- the number of the module controller workers. Default: `1`.

In order to change the default values of the options, use the corresponding Helm chart value. Below is an example of the Operator installation/upgrade.

### Install with options
```
helm install \
  demo hashicorp/terraform-cloud-operator \
  --namespace tfc-operator-system \
  --create-namespace \
  --set operator.syncPeriod=10m \
  --set 'operator.watchedNamespaces={white,blue,red}' \
  --set controllers.module.workers=5 \
  --set controllers.workspace.workers=5
```

In this example, the Operator will watch 3 namespaces in the Kubernetes cluster: `white`, `blue`, and `red`.

### Upgrade with options
```
helm upgrade \
  demo hashicorp/terraform-cloud-operator \
  --namespace tfc-operator-system \
  --set operator.syncPeriod=5m \
  --set controllers.module.workers=10 \
  --set controllers.workspace.workers=20
```

In this example, the Operator will watch all namespaces in the Kubernetes cluster.

## Troubleshooting

If you encounter any issues with the Operator there are a number of ways how to troubleshoot it:

- check the Operator logs:
    ```
    kubectl logs -f <POD_NAME>
    ```

- check the CR events:
    ```
    kubectl describe workspace <NAME>
    kubectl describe module <NAME>
    ```

If you believe you've found a bug and cannot find an existing issue, feel free to open a new issue! Be sure to include as much information as you can about your environment.

## Security Reporting

If you think you've found a security vulnerability, we'd love to hear from you.

Follow the instructions in [SECURITY.md](.github/SECURITY.md) to make a report.
