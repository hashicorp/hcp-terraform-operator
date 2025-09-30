# Metrics

The Operator exposes metrics in the [Prometheus](https://prometheus.io/) format for each controller. They are available at the standard `/metrics` path over the HTTPS port `8443`.

The metrics are protected by [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy). This allows providing RBAC-based access to the metrics within the Kubernetes cluster.

## Available Metrics

The Operator exposes all metrics provided by the controller-runtime by default. The full list you can find on the [Kubebuilder documentation](https://book.kubebuilder.io/reference/metrics-reference.html).

Starting with version `2.10.0`, the operator introduces HCP Terraform–specific metrics. These metrics use the prefix `hcp_tf_*`. Below is the full list of HCP Terraform–specific metrics.

| Metric name | Type | Description | Controller | Status |
|-------------|------|-------------|------------|--------|
| `hcp_tf_runs{run_status="<HCP Terraform Run Status>"}` | Gauge | Pending runs by statuses. | RunsCollector | Alpha |
| `hcp_tf_runs_total` | Gauge | Total number of pending Runs by statuses. | RunsCollector | Alpha |

## Scraping Metrics

How metrics are scraped will depend on how you operate your Prometheus server. The below example assumes that the [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator) is being used to run Prometheus.

If the Operator is deployed with helm, a Kubernetes Cluster IP service resource is created. This service should be used as a target for Prometheus. The service name builds by the following template: `{{ .Release.Name }}-controller-manager-metrics-service`

Below is an example of the Prometheus Operator ConfigMap to scrape metrics from the Operator Helm release named `tfc-operator`:

```yaml
apiVersion: v1
data:
  ...
  prometheus.yml: |
    ...
    - job_name: tfc-operator
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      scheme: https
      scrape_interval: 1m
      scrape_timeout: 10s
      static_configs:
      - targets:
        - tfc-operator-controller-manager-metrics-service:8443
      tls_config:
        insecure_skip_verify: true
```
