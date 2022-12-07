# `Workspace`

`Workspace` controller allows managing Terraform Cloud Workspace via Kubernetes Custom Resources.

Below is an example of a Workspace Custom Resource:

```yaml
apiVersion: app.terraform.io/v1alpha2
kind: Workspace
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: tfc-operator
      key: token
  name: workspace-demo
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

Once the above CR is applied, the Operator creates a new workspace `workspace-demo` under the `kubernetes-operator` organization.

Non-sensitive outputs of the workspace runs will be saved in Kubernetes ConfigMaps. Sensitive outputs of the workspace runs will be saved in Kubernetes Secrets. In both cases, the name of the corresponding Kubernetes object will be generated automatically and has the following pattern: `<metadata.name>-outputs`. For the above example, the name of ConfigMap and Secret will be `this-outputs`.

Please refer to the [CRD](../config/crd/bases/app.terraform.io_workspaces.yaml) to get the full list of available options.
