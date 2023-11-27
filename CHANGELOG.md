## 2.1.0 (November 27, 2023)

ENHANCEMENT:

* `Workspace`: Add the ability to configure the project for the workspace via a new field `spec.project.[id | name]`.  [[GH-300](https://github.com/hashicorp/terraform-cloud-operator/pull/300)]

BUG FIXES:

* `Module`: fix an issue when initiating foreground cascading deletion results in two destroy runs being triggered, and even after both runs are successfully executed, a module object persists in Kubernetes. [[GH-301](https://github.com/hashicorp/terraform-cloud-operator/pull/301)]

## 2.0.0 (November 06, 2023)

BUG FIXES:

* `Workspace`: fix an issue of properly handling special characters when generating string output. [[GH-289](https://github.com/hashicorp/terraform-cloud-operator/pull/289)]
* `Module`: fix an issue of properly handling special characters when generating string output. [[GH-289](https://github.com/hashicorp/terraform-cloud-operator/pull/289)]

ENHANCEMENT:

* `Helm Chart`: Add the ability to configure `kube-rbac-proxy` image and resources. [[GH-259](https://github.com/hashicorp/terraform-cloud-operator/pull/259)] [[GH-271](https://github.com/hashicorp/terraform-cloud-operator/pull/271)]
* `AgentPool`: Add the ability to use wildcard-name searches to target workspaces for autoscaling. [[GH-274](https://github.com/hashicorp/terraform-cloud-operator/pull/274)]
* `AgentPool`: Make `targetWorkspaces` field optional and default to targeting all workspaces linked to an AgentPool. [[GH-274](https://github.com/hashicorp/terraform-cloud-operator/pull/274)]
* `AgentPool`: Tweak autoscaling to take into account Planning and Applying states when computing the replica count for agents  [[GH-290](https://github.com/hashicorp/terraform-cloud-operator/pull/290)]
* `AgentPool`: Default agent pods to have a `terminationGracePeriod` of 15 minutes. [[GH-290](https://github.com/hashicorp/terraform-cloud-operator/pull/290)]

DOCS:

* Update FAQ. [[GH-271](https://github.com/hashicorp/terraform-cloud-operator/pull/271)]

DEPENDENCIES:

* Bump `sigs.k8s.io/controller-runtime` from 0.15.1 to 0.15.3. [[GH-258](https://github.com/hashicorp/terraform-cloud-operator/pull/258)] [[GH-294](https://github.com/hashicorp/terraform-cloud-operator/pull/294)]
* Bump `github.com/hashicorp/go-slug` from 0.12.1 to 0.12.2. [[GH-261](https://github.com/hashicorp/terraform-cloud-operator/pull/261)]
* Bump `k8s.io/api` from 0.27.5 to 0.27.7. [[GH-264](https://github.com/hashicorp/terraform-cloud-operator/pull/264)] [[GH-292](https://github.com/hashicorp/terraform-cloud-operator/pull/292)]
* Bump `k8s.io/apimachinery` from 0.27.5 to 0.27.7. [[GH-264](https://github.com/hashicorp/terraform-cloud-operator/pull/264)] [[GH-292](https://github.com/hashicorp/terraform-cloud-operator/pull/292)]
* Bump `k8s.io/client-go` from 0.27.5 to 0.2776. [[GH-264](https://github.com/hashicorp/terraform-cloud-operator/pull/264)] [[GH-292](https://github.com/hashicorp/terraform-cloud-operator/pull/292)]
* Bump `kube-rbac-proxy` image from `0.14.2` to `0.14.4`. [[GH-271](https://github.com/hashicorp/terraform-cloud-operator/pull/271)] [[GH-281](https://github.com/hashicorp/terraform-cloud-operator/pull/281)]
* Bump `golang.org/x/net` from 0.14.0 to 0.17.0. [[GH-272](https://github.com/hashicorp/terraform-cloud-operator/pull/272)]
* Bump `golang.org/x/sys` from 0.11.0 to 0.13.0. [[GH-272](https://github.com/hashicorp/terraform-cloud-operator/pull/272)]
* Bump `golang.org/x/term` from 0.11.0 to 0.13.0. [[GH-272](https://github.com/hashicorp/terraform-cloud-operator/pull/272)]
* Bump `golang.org/x/text` from 0.12.0 to 0.13.0. [[GH-272](https://github.com/hashicorp/terraform-cloud-operator/pull/272)]
* Bump `github.com/hashicorp/go-tfe` from 1.32.1 to 1.35.0. [[GH-273](https://github.com/hashicorp/terraform-cloud-operator/pull/273)]
* Bump `github.com/onsi/gomega` from 1.28.1 to 1.29.0. [[GH-291](https://github.com/hashicorp/terraform-cloud-operator/pull/291)]
* Bump `github.com/go-logr/logr` from 1.2.4 to 1.3.0. [[GH-293](https://github.com/hashicorp/terraform-cloud-operator/pull/293)]

## Community Contributors :raised_hands:
- @kieranbrown made their contribution in https://github.com/hashicorp/terraform-cloud-operator/pull/259
- @KamalAman for constantly providing us with a valuable feedback :rocket:

## 2.0.0-beta8 (August 29, 2023)

BUG FIXES:

* `AgentPool`: fix an issue when `plan_queued` and `apply_queued` statuses do not trigger agent scaling. [[GH-215](https://github.com/hashicorp/terraform-cloud-operator/pull/215)]
* `Helm Chart`: fix an issue with the Deployment template in the Helm chart where `name` in path `spec.template.spec.containers[0]` was duplicated. [[GH-216](https://github.com/hashicorp/terraform-cloud-operator/pull/216)]
* `Workspace`: fix an issue when the Operator panics when `spec.executionMode` is configured as `agent` but `spec.agentPool` is not set which is mandatory in this case. [[GH-242](https://github.com/hashicorp/terraform-cloud-operator/pull/242)]
* `Workspace`: fix an issue when a new Workspace is successfully created, but its `status.WorkspaceID` status fails to update with a new Workspace ID due to an error during subsequent reconciliation. Consequently, the Workspace controller continuously encounters failures while attempting to reconcile the newly created Workspace. [[GH-234](https://github.com/hashicorp/terraform-cloud-operator/pull/234)]

ENHANCEMENT:

* `Operator`: Add the ability to skip TLS certificate validation for communication between the Operator and the TFC/E endpoint. A new environment variable `TFC_TLS_SKIP_VERIFY` should be set to `true` to skip the validation. Default: `false`. [[GH-222](https://github.com/hashicorp/terraform-cloud-operator/pull/222)]
* `Helm Chart`: Add a new parameter `operator.skipTLSVerify` to configure the ability to skip TLS certificate validation for communication between the Operator and the TFC/E endpoint. Default: `false`. [[GH-222](https://github.com/hashicorp/terraform-cloud-operator/pull/222)]
* `Workspace`: Add `spec.Tags` validation to align with the TFC requirement. [[GH-234](https://github.com/hashicorp/terraform-cloud-operator/pull/234)]

DEPENDENCIES:

* Bump `github.com/hashicorp/go-tfe` from 1.29.0 to 1.32.1. [[GH-218](https://github.com/hashicorp/terraform-cloud-operator/pull/218)] [[GH-247](https://github.com/hashicorp/terraform-cloud-operator/pull/247)]
* Bump `github.com/hashicorp/go-slug` from 0.11.1 to 0.12.1. [[GH-219](https://github.com/hashicorp/terraform-cloud-operator/pull/219)] [[GH-247](https://github.com/hashicorp/terraform-cloud-operator/pull/247)]
* Bump `github.com/onsi/gomega` from 1.27.8 to 1.27.10. [[GH-247](https://github.com/hashicorp/terraform-cloud-operator/pull/247)]
* Bump `go.uber.org/zap` from 1.24.0 to 1.25.0. [[GH-247](https://github.com/hashicorp/terraform-cloud-operator/pull/247)]
* Bump `k8s.io/api` from 0.27.3 to 0.27.5. [[GH-247](https://github.com/hashicorp/terraform-cloud-operator/pull/247)] [[GH-252](https://github.com/hashicorp/terraform-cloud-operator/pull/252)]
* Bump `k8s.io/apimachinery` from 0.27.3 to 0.27.5. [[GH-247](https://github.com/hashicorp/terraform-cloud-operator/pull/247)] [[GH-252](https://github.com/hashicorp/terraform-cloud-operator/pull/252)]
* Bump `k8s.io/client-go` from 0.27.3 to 0.27.5. [[GH-247](https://github.com/hashicorp/terraform-cloud-operator/pull/247)] [[GH-252](https://github.com/hashicorp/terraform-cloud-operator/pull/252)]
* Bump `sigs.k8s.io/controller-runtime` from 0.15.0 to 0.15.1. [[GH-247](https://github.com/hashicorp/terraform-cloud-operator/pull/247)]
* Bump `kube-rbac-proxy` image from `0.13.1` to `0.14.2`. [[GH-251](https://github.com/hashicorp/terraform-cloud-operator/pull/251)]
* Bump `github.com/onsi/ginkgo/v2` from 2.11.0 to 2.12.0. [[GH-254](https://github.com/hashicorp/terraform-cloud-operator/pull/254)]

## 2.0.0-beta7 (July 07, 2023)

NOTES:
* `Helm Chart`: the Helm chart version is synced to the Terraform Cloud Operator version. [[GH-204](https://github.com/hashicorp/terraform-cloud-operator/pull/204)]

BUG FIXES:

* `Operator`: fix an issue when the operator couldn't be run on the `amd64` platform. [[GH-203](https://github.com/hashicorp/terraform-cloud-operator/pull/203)]

ENHANCEMENT:
* `Helm Chart`: `operator.image.tag` defaults to `.Chart.AppVersion`. [[GH-204](https://github.com/hashicorp/terraform-cloud-operator/pull/204)]
* `Workspace`: add event filtering to reduce the number of unnecessary reconciliations. [[GH-194](https://github.com/hashicorp/terraform-cloud-operator/pull/194)]
* `AgentPool`: add `autoscaling` field to allow configuration of a basic autoscaler for agent deployments based on pending runs. [[GH-198](https://github.com/hashicorp/terraform-cloud-operator/pull/198)]
* `Workspace`: add Terraform version utilized in the Workspace to the status: `status.TerraformVersion`. [[GH-206](https://github.com/hashicorp/terraform-cloud-operator/pull/206)]

DOCS:

* Update FAQ. [[GH-206](https://github.com/hashicorp/terraform-cloud-operator/pull/206)]

DEPENDENCIES:

* Bump `k8s.io/api` from 0.27.2 to 0.27.3. [[GH-195](https://github.com/hashicorp/terraform-cloud-operator/pull/195)]
* Bump `k8s.io/apimachinery` from 0.27.2 to 0.27.3. [[GH-195](https://github.com/hashicorp/terraform-cloud-operator/pull/195)]
* Bump `k8s.io/client-go` from 0.27.2 to 0.27.3. [[GH-195](https://github.com/hashicorp/terraform-cloud-operator/pull/195)]
* Bump `github.com/onsi/ginkgo/v2` from 2.9.5 to 2.11.0. [[GH-197](https://github.com/hashicorp/terraform-cloud-operator/pull/197)]
* Bump `github.com/onsi/gomega` from 1.27.7 to 1.27.8. [[GH-197](https://github.com/hashicorp/terraform-cloud-operator/pull/197)]
* Bump `github.com/hashicorp/go-tfe` from 1.23.0 to 1.29.0. [[GH-205](https://github.com/hashicorp/terraform-cloud-operator/pull/205)]

## 2.0.0-beta6 (June 23, 2023)

NOTES:
* `Operator`: the Operator no longer includes the global option `--config`. [[GH-185](https://github.com/hashicorp/terraform-cloud-operator/pull/185)]
* `Helm Chart`: the Helm chart no longer includes the ConfigMap `manager-config` as it has been removed. [[GH-185](https://github.com/hashicorp/terraform-cloud-operator/pull/185)]
* `Helm Chart`: the Helm chart now allows configuration of custom CA bundles [[GH-173](https://github.com/hashicorp/terraform-cloud-operator/pull/173)]

ENHANCEMENT:

* `Module`: the Run now adopts the apply method of the Workspace in which it is executed. If the apply method is set to 'manual', the Run will remain on hold until it receives manual approval or rejection for the application or cancellation of the Run. [[GH-170](https://github.com/hashicorp/terraform-cloud-operator/pull/170)]
* `Module`: add a new field `spec.name` that allows modifying the name of the module that is generated by the Operator. Default: `this`. [[GH-172](https://github.com/hashicorp/terraform-cloud-operator/pull/172)]
* `Workspace`: mark fields `.status.ObservedGeneration`, `.status.UpdateAt`, and `.status.runStatus.configurationVersion` as optional. [[GH-186](https://github.com/hashicorp/terraform-cloud-operator/pull/186)]
* `Workspace`: add an extra validation during the reconciliation to exit if the object contains the `v1` finalizer `finalizer.workspace.app.terraform.io`. [[GH-186](https://github.com/hashicorp/terraform-cloud-operator/pull/186)]

DEPENDENCIES:

* Bump `github.com/go-logr/zapr` from 1.2.3 to 1.2.4. [[GH-185](https://github.com/hashicorp/terraform-cloud-operator/pull/185)]
* Bump `github.com/onsi/ginkgo/v2` from 2.9.4 to 2.9.5. [[GH-185](https://github.com/hashicorp/terraform-cloud-operator/pull/185)]
* Bump `github.com/onsi/gomega` from 1.27.6 to 1.27.7. [[GH-185](https://github.com/hashicorp/terraform-cloud-operator/pull/185)]
* Bump `k8s.io/api` from 0.26.3 to 0.27.2. [[GH-185](https://github.com/hashicorp/terraform-cloud-operator/pull/185)]
* Bump `k8s.io/apimachinery` from 0.26.3 to 0.27.2. [[GH-185](https://github.com/hashicorp/terraform-cloud-operator/pull/185)]
* Bump `k8s.io/client-go` from 0.26.3 to 0.27.2. [[GH-185](https://github.com/hashicorp/terraform-cloud-operator/pull/185)]
* Bump `sigs.k8s.io/controller-runtime` from 0.14.6 to 0.15.0. [[GH-185](https://github.com/hashicorp/terraform-cloud-operator/pull/185)]

## 2.0.0-beta5 (April 18, 2023)

BUG FIXES:

* RBAC fixes for agent deployment [[GH-135](https://github.com/hashicorp/terraform-cloud-operator/pull/135)], [[GH-143](https://github.com/hashicorp/terraform-cloud-operator/pull/134)]

DEPENDENCIES:

* Bump sigs.k8s.io/controller-runtime from 0.14.5 to 0.14.6 [[GH-132](https://github.com/hashicorp/terraform-cloud-operator/pull/132)]

## 2.0.0-beta4 (March 28, 2023)

ENHANCEMENT:

* `AgentPool`: add custom resource `spec` validation. [[GH-79](https://github.com/hashicorp/terraform-cloud-operator/issues/79)]
* `AgentPool`: add `agentDeployment` field to spec [[GH-96](https://github.com/hashicorp/terraform-cloud-operator/pull/96)]
* `Module`: add custom resource `spec` validation. [[GH-79](https://github.com/hashicorp/terraform-cloud-operator/issues/79)]
* `Workspace`: add custom resource `spec` validation. [[GH-79](https://github.com/hashicorp/terraform-cloud-operator/issues/79)]
* `Workspace`: add `notifications` field to spec [[GH-107](https://github.com/hashicorp/terraform-cloud-operator/pull/107)]
* `Workspace`: add `runTasks` field to spec [[GH-89](https://github.com/hashicorp/terraform-cloud-operator/pull/89)]

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
