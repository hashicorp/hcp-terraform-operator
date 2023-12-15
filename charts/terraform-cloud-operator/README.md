# Installation

Use the option `--version VERSION` with `helm install` and `helm upgrade` commands to specify the version you want to install.

## Steps

1. Add the Helm repository

    ```console
    $ helm repo add hashicorp https://helm.releases.hashicorp.com
    ```

2. Update your local Helm chart repository cache

    ```console
    $ helm repo update
    ```

3. Install

    ```console
    $ helm install demo hashicorp/terraform-cloud-operator \
      --version 2.1.0 \
      --namespace tfc-operator-system \
      --create-namespace
    ```

Below are examples of the Operator installation/upgrade Helm chart with options.

### Install with options

```console
$ helm install demo hashicorp/terraform-cloud-operator \
  --version 2.1.0 \
  --namespace tfc-operator-system \
  --create-namespace \
  --set operator.syncPeriod=10m \
  --set 'operator.watchedNamespaces={white,blue,red}' \
  --set controllers.agentPool.workers=5 \
  --set controllers.module.workers=5 \
  --set controllers.project.workers=5 \
  --set controllers.workspace.workers=5
```

In the above example, the Operator will watch 3 namespaces in the Kubernetes cluster: `white`, `blue`, and `red`.

### Install to work with Terraform Enterprise

If targeting a Terraform Enterprise instance rather than Terraform Cloud, set the API endpoint URL using the `operator.tfeAddress` value:

```console
$ helm install demo hashicorp/terraform-cloud-operator \
  --version 2.1.0 \
  --set operator.tfeAddress="https://tfe-api.my-company.com"
```

If you encounter a TLS-related issue using the Operator with Terraform Enterprise, you may need to configure your own CA certificates using the [`customCAcertificates`](./README.md#values) value, or by skipping TLS verification using the [`operator.skipTLSVerify`](./README.md#values) value.

For more information, please refer to the [FAQ](./../../docs/faq.md#general-questions).

### Upgrade with options

```console
$ helm upgrade demo hashicorp/terraform-cloud-operator \
  --version 2.1.0 \
  --namespace tfc-operator-system \
  --set operator.syncPeriod=5m \
  --set controllers.agentPool.workers=5 \
  --set controllers.module.workers=10 \
  --set controllers.project.workers=2 \
  --set controllers.workspace.workers=20
```

In the above example, the Operator will watch all namespaces in the Kubernetes cluster.

<!-- TODO: automate Values section generation: https://github.com/norwoodj/helm-docs -->
# Values

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| replicaCount | int | 2 | Number of Terraform Cloud Operator replicas. |
| operator.image.repository | string | "hashicorp/terraform-cloud-operator" | Image repository. |
| operator.image.pullPolicy | string | "IfNotPresent" | Image pull policy. |
| operator.image.tag | string | "" | Image tag. |
| operator.resources.limits.cpu | string | "500m" | Limits as a maximum amount of CPU to be used by a container. |
| operator.resources.limits.memory | string | "128Mi" | Limits as a maximum amount of memory to be used by a container. |
| operator.resources.requests.cpu | string | "50m" | Guaranteed minimum amount of CPU to be used by a container. |
| operator.resources.requests.memory | string | "64Mi" | Guaranteed minimum amount of memory to be used by a container. |
| operator.syncPeriod | string | "5m" | The minimum frequency at which watched resources are reconciled. Format: 5s, 1m, etc. |
| operator.watchedNamespaces | list | [] | List of namespaces the controllers should watch. |
| operator.tfeAddress | string | "" | The API URL of a Terraform Enterprise instance. |
| operator.skipTLSVerify | bool | false | Whether or not to ignore TLS certification warnings. |
| kubeRbacProxy.image.repository | string | "quay.io/brancz/kube-rbac-proxy" | Image repository. |
| kubeRbacProxy.image.pullPolicy | string | "IfNotPresent" | Image pull policy. |
| kubeRbacProxy.image.tag | string | "v0.14.4" | Image tag. |
| kubeRbacProxy.resources.limits.cpu | string | "500m" | Limits as a maximum amount of CPU to be used by a container. |
| kubeRbacProxy.resources.limits.memory | string | "128Mi" | Limits as a maximum amount of memory to be used by a container. |
| kubeRbacProxy.resources.requests.cpu | string | "50m" | Guaranteed minimum amount of CPU to be used by a container. |
| kubeRbacProxy.resources.requests.memory | string | "64Mi" | Guaranteed minimum amount of memory to be used by a container. |
| controllers.agentPool.workers | int | 1 | The number of the Agent Pool controller workers. |
| controllers.module.workers | int | 1 | The number of the Module controller workers. |
| controllers.project.workers | int | 1 | The number of the Project controller workers. |
| controllers.workspace.workers | int | 1 | The number of the Workspace controller workers. |
| customCAcertificates | string | "" | Custom Certificate Authority bundle to validate API TLS certificates. Expects a path to a CRT file containing concatenated certificates. |
