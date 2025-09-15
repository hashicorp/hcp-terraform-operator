# `Runs Collector`

`Runs Collector` controller scrapes HCP Terraform run statuses from a given Agent Pool.

Please refer to the [CRD](../config/crd/bases/app.terraform.io_runscollectors.yaml) and [API Reference](./api-reference.md#runscollector) to get the full list of available options.

Below is a basic example of a Runs Collector Custom Resource:

```yaml
apiVersion: app.terraform.io/v1alpha2
kind: RunsCollector
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: hcp-terraform-operator
      key: token
  agentPool:
    name: multik
```

Once the above CR is applied, the Operator starts scraping run metrics from the `multik` agent pool under the `kubernetes-operator` organization.

If you have any questions, please check out the [FAQ](./faq.md#runs-collector-controller).

If you encounter any issues with the `RunsCollector` controller please refer to the [Troubleshooting](../README.md#troubleshooting).
