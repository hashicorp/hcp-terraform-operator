## 2.0.0-beta4 (Unreleased)

ENHANCEMENT:

* `AgentPool`: add custom resource `spec` validation. [[GH-79](https://github.com/hashicorp/terraform-cloud-operator/issues/79)]
* `Module`: add custom resource `spec` validation. [[GH-79](https://github.com/hashicorp/terraform-cloud-operator/issues/79)]
* `Workspace`: add custom resource `spec` validation. [[GH-79](https://github.com/hashicorp/terraform-cloud-operator/issues/79)]

BUG FIXES:

* `Module`: fix an issue when custom resource fails if it refers to Workspace by ID. [[GH-77](https://github.com/hashicorp/terraform-cloud-operator/issues/77)]

DEPENDENCIES:

* Bump `github.com/onsi/ginkgo/v2` from 2.7.0 to 2.8.0. [[GH-73](https://github.com/hashicorp/terraform-cloud-operator/issues/73)]
* Bump `sigs.k8s.io/controller-runtime` to 0.14.3. [[GH-78](https://github.com/hashicorp/terraform-cloud-operator/issues/78)]

DOCS:

* Update controllers documentation. [[GH-76](https://github.com/hashicorp/terraform-cloud-operator/issues/76)] [[GH-79](https://github.com/hashicorp/terraform-cloud-operator/issues/79)]
* Update FAQ. [[GH-76](https://github.com/hashicorp/terraform-cloud-operator/issues/76)]
* Update API Reference. [[GH-76](https://github.com/hashicorp/terraform-cloud-operator/issues/76)] [[GH-79](https://github.com/hashicorp/terraform-cloud-operator/issues/79)]
* Add examples. [[GH-76](https://github.com/hashicorp/terraform-cloud-operator/issues/76)]

## 2.0.0-beta3 (January 25, 2023)

FEATURES:

* `Operator`: add support for Terraform Enterprise endpoints via Helm chart variable `operator.tfeAddress`.

BUG FIXES:

* `AgentPool`: fix an issue when manually created agent tokens are not removed from Agent Pool during the reconciliation.

DOCS:

* Update documentation.
* Add FAQ.
* Reorganize documentation structure.
