# Usage

## Prerequisites

- The Operator requires a Terraform Cloud [organization](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations) name and a [team 'owners' token](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens#team-api-tokens) in order to access the Terraform Cloud API.
- The API token must be stored in a Kubernetes secret.
- A single instance of the Operator can manage Terraform Cloud resources for different organizations and/or different API tokens. For that purpose, the organization name and a reference to the corresponding Kubernetes secret are shipped within the custom resource.

Below are examples of how to create a Kubernetes secret and store the API token there. The examples assume that the API token is already known.

1. `kubectl` command

    ```console
    $ kubectl create secret generic tfc-operator --from-literal=token=APIt0k3n
    ```

2. YAML manifest
    - Encode the API token

        ```console
        $ echo -n "APIt0k3n" | base64
        ```

    - Create a YAML manifest and paste the encoded token from the previous step

        ```yaml
        apiVersion: v1
        kind: Secret
        metadata:
          name: tfc-operator
        type: Opaque
        data:
          token: QVBJdDBrM24=
        ```

    - Apply YAML manifest

        ```console
        $ kubectl apply -f secret.yaml
        ```

For more information about Kubernetes secrets please refer to the [Kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/secret/). Please use the approach that is matching with the best practices which are accepted in your organization.

Controllers usage guides:
  - [AgentPool](../docs/agentpool.md)
  - [Module](../docs/module.md)
  - [Workspace](../docs/workspace.md)
