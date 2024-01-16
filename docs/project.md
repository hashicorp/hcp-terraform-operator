# `Project`

`Project` controller allows managing Terraform Cloud Projects via Kubernetes Custom Resources.

Please refer to the [CRD](../config/crd/bases/app.terraform.io_projects.yaml) and [API Reference](./api-reference.md#project) to get the full list of available options.

Below is a basic example of a Project Custom Resource:

```yaml
apiVersion: app.terraform.io/v1alpha2
kind: Project
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: tfc-operator
      key: token
  name: project-demo
```

Once the above CR is applied, the Operator creates a new project `project-demo` under the `kubernetes-operator` organization.

The example can be extended with team access permission support:

```yaml
apiVersion: app.terraform.io/v1alpha2
kind: Project
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: tfc-operator
      key: token
  name: project-demo
  teamAccess:
  - team:
      name: demo
    access: admin
```

The team `demo` will get `Admin` [permission group](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions) access to workspaces under the project `project-demo`.

If you have any questions, please check out the [FAQ](./faq.md#project-controller).

If you encounter any issues with the `Project` controller please refer to the [Troubleshooting](../README.md#troubleshooting).
