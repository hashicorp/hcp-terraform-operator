# `Module`

`Module` controller allows executing arbitrary Terraform Modules code in Terraform Cloud Workspace via Kubernetes Custom Resources.

Please refer to the [CRD](../config/crd/bases/app.terraform.io_modules.yaml) and [API Reference](./api-reference.md#module) to get the full list of available options.

Below is an example of a Module Custom Resource:

```yaml
apiVersion: app.terraform.io/v1alpha2
kind: Module
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: tfc-operator
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

```console
$ kubectl patch module <NAME> \
  --type=merge \
  --patch '{"spec": {"restartedAt": "'`date -u -Iseconds`'"}}'
```

If you have any questions, please check out the [FAQ](./faq.md#module-controller) to see if you can find answers there.

If you encounter any issues with the `AgentPool` controller please refer to the [Troubleshooting](../README.md#troubleshooting).
