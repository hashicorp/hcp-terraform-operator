# Installation

## :warning: Beta :warning:

By default, Helm does not display development versions (alpha, beta, and release candidate releases). In order to list them, use the following command when looking for available versions:

```console
$ helm search repo hashicorp/terraform-cloud-operator --versions --devel
```

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
      --version 2.0.0-beta7 \
      --namespace tfc-operator-system \
      --create-namespace
    ```

Below are examples of the Operator installation/upgrade Helm chart with options.

### Install with options

```console
$ helm install demo hashicorp/terraform-cloud-operator \
  --version 2.0.0-beta7 \
  --namespace tfc-operator-system \
  --create-namespace \
  --set operator.syncPeriod=10m \
  --set 'operator.watchedNamespaces={white,blue,red}' \
  --set controllers.agentPool.workers=5 \
  --set controllers.module.workers=5 \
  --set controllers.workspace.workers=5
```

In the above example, the Operator will watch 3 namespaces in the Kubernetes cluster: `white`, `blue`, and `red`.

If targeting a TFE instance rather than Terraform Cloud, set the API URL using this variable:

```
  --set operator.tfeAddress="https://tfe-api.my-company.com"
```

If the TFE instance uses a TLS certificate signed by a non-public authority or "Let's Encrypt", the chain of CAs that can validate 
that TLS certificate should be installed with the operator by setting the `customCAcertificates` chart value:

```
  --set customCAcertificates=<path-to-CA-chain-file.crt>
``` 

### Upgrade with options

```console
$ helm upgrade demo hashicorp/terraform-cloud-operator \
  --version 2.0.0-beta7 \
  --namespace tfc-operator-system \
  --set operator.syncPeriod=5m \
  --set controllers.agentPool.workers=5 \
  --set controllers.module.workers=10 \
  --set controllers.workspace.workers=20
```

In the above example, the Operator will watch all namespaces in the Kubernetes cluster.

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
| controllers.agentPool.workers | int | 1 | The number of the Agent Pool controller workers. |
| controllers.module.workers | int | 1 | The number of the Module controller workers. |
| controllers.workspace.workers | int | 1 | The number of the Workspace controller workers. |
| customCAcertificates | string | "" | Custom Certificate Authority bundle to validate API TLS certificates. Expects a path to a CRT file containing concatenated certificates. |
