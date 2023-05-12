# Installation

The Operator provides [Helm chart](../charts/terraform-cloud-operator) as a first-class method of installation on Kubernetes.

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
    $ helm install \
      demo hashicorp/terraform-cloud-operator \
      --namespace tfc-operator-system \
      --create-namespace
    ```

Below are examples of the Operator installation/upgrade Helm chart with options.

### Install with options

```console
$ helm install \
  demo hashicorp/terraform-cloud-operator \
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
$ helm upgrade \
  demo hashicorp/terraform-cloud-operator \
  --namespace tfc-operator-system \
  --set operator.syncPeriod=5m \
  --set controllers.agentPool.workers=5 \
  --set controllers.module.workers=10 \
  --set controllers.workspace.workers=20
```

In the above example, the Operator will watch all namespaces in the Kubernetes cluster.
