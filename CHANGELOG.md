## 2.9.0 (April 24, 2025)

BREAKING CHANGES:

* `AgentPool`: The new field, `spec.deletionPolicy`, is set to `retain` by default, which changes the previous default controller behavior when resources are deleted. The previous behavior corresponded to the `destroy` deletion policy value. This change is considered safer in cases of accidental resource deletion, planned migration, or other scenarios involving the deletion of a custom resource. [[GH-584](https://github.com/hashicorp/hcp-terraform-operator/pull/584)]
* `Projects`: The new field, `spec.deletionPolicy` is set to `retain` by default, which changes the previous default controller when projects are deleted. The previous behavior corresponded to the `soft` deletion policy. This change takes into account migration and overall better management of resources. [[GH-569](https://github.com/hashicorp/hcp-terraform-operator/pull/569)]

NOTES:

* `AgentPool`: The field `spec.autoscaling.targetWorkspaces` is deprecated and will be removed in a future release. The motivation behind this change is that the field does not guarantee newly provisioned agents will pick up runs only for the listed workspaces. Additionally, using this field could lead to situations where agents are not scaled to process runs for unlisted workspaces. [[GH-561](https://github.com/hashicorp/hcp-terraform-operator/pull/561)]
* `Module`: The field `spec.destroyOnDeletion` is deprecated and will be removed in a future release. This change is motivated by the introduction of a new field, `spec.deletionPolicy`, which aligns with the approach the operator uses across all other controllers. The default value of `spec.deletionPolicy` is `retain`, which replaces the default value `false` of `spec.destroyOnDeletion`. The corresponding value `true` of `spec.destroyOnDeletion` is replaced with `destroy` in `spec.deletionPolicy`. [[GH-582](https://github.com/hashicorp/hcp-terraform-operator/pull/582)]
* The `AgentPool` CRD has been changed. Please follow the Helm chart instructions on how to upgrade it. [[GH-494](https://github.com/hashicorp/hcp-terraform-operator/pull/494)]
* The `Module` CRD has been changed. Please follow the Helm chart instructions on how to upgrade it. [[GH-582](https://github.com/hashicorp/hcp-terraform-operator/pull/582)]
* The `Project` CRD has been changed. Please follow the Helm chart instructions on how to upgrade it.  [[GH-569](https://github.com/hashicorp/hcp-terraform-operator/pull/569)]
* The `Workspace` CRD has been changed. Please follow the Helm chart instructions on how to upgrade it. [[GH-558](https://github.com/hashicorp/hcp-terraform-operator/pull/558)]

BUG FIXES:

* `AgentPool`: Add a new scaling algorithm that accounts for speculative plans when calculating pending runs. This algorithm is compatible with HCP Terraform and TFE v202409-1 and later. [[GH-561](https://github.com/hashicorp/hcp-terraform-operator/pull/561)]
* `AgentPool`: Fix an issue where, in some circumstances, a newly created CR with the autoscaling feature enabled does not update its status while handling runs. [[GH-580](https://github.com/hashicorp/hcp-terraform-operator/pull/580)]

FEATURES:

* `AgentPool`: Add a new field, `spec.deletionPolicy`, that specifies the behavior of the custom resource and its associated agent pool when the custom resource is deleted. [[GH-584](https://github.com/hashicorp/hcp-terraform-operator/pull/584)]
* `Module`: Add a new field `spec.deletionPolicy`, that specifies the behavior of a custom resource and its associated module when the custom resource is deleted. This field aims to replace `spec.destroyOnDeletion`. [[GH-582](https://github.com/hashicorp/hcp-terraform-operator/pull/582)]
* `Project`: Add a new field, `spec.deletionPolicy`, that specifies the behavior of a custom resource and its associated project when the custom resource is deleted. [[GH-569](https://github.com/hashicorp/hcp-terraform-operator/pull/569)]

ENHANCEMENTS:

* `Workspace`: Add the ability to enable auto apply for run triggers via a new optional field `spec.applyRunTrigger`. This feature is available in HCP Terraform and Terraform Enterprise starting with version v202401-1. [[GH-558](https://github.com/hashicorp/hcp-terraform-operator/pull/558)]
* `Workspace`: Add new fields, `spec.versionControl.enableFileTriggers`, `spec.versionControl.triggerPatterns` and `spec.versionControl.triggerPrefixes`, which specify whether a new run should be triggered when files in the specified paths change in the connected repository. [[GH-496](https://github.com/hashicorp/hcp-terraform-operator/pull/496)] [[GH-578](https://github.com/hashicorp/hcp-terraform-operator/pull/578)]

DEPENDENCIES:

* Bump `k8s.io/api` from 0.31.6 to 0.32.3. [[GH-586](https://github.com/hashicorp/hcp-terraform-operator/pull/586)] [[GH-587](https://github.com/hashicorp/hcp-terraform-operator/pull/587)]
* Bump `k8s.io/apimachinery` from 0.31.6 to 0.32.3. [[GH-586](https://github.com/hashicorp/hcp-terraform-operator/pull/586)] [[GH-587](https://github.com/hashicorp/hcp-terraform-operator/pull/587)]
* Bump `k8s.io/client-go` from 0.31.6 to 0.32.3. [[GH-586](https://github.com/hashicorp/hcp-terraform-operator/pull/586)] [[GH-587](https://github.com/hashicorp/hcp-terraform-operator/pull/587)]
* Bump `sigs.k8s.io/controller-runtime` from 0.19.7 to 0.20.4. [[GH-586](https://github.com/hashicorp/hcp-terraform-operator/pull/586)]

## 2.8.1 (March 12, 2025)

ENHANCEMENTS:

* `Workspace`: Add support for attaching variable sets to a workspace referenced by its name. [[GH-570](https://github.com/hashicorp/hcp-terraform-operator/pull/570)]

DEPENDENCIES:

* Bump `kube-rbac-proxy` from 0.18.2 to 0.19.0. [[GH-560](https://github.com/hashicorp/hcp-terraform-operator/pull/560)]
* Bump `github.com/hashicorp/go-tfe` from 1.71.0 to 1.76.0. [[GH-566](https://github.com/hashicorp/hcp-terraform-operator/pull/566)]
* Bump `github.com/hashicorp/go-slug` from 0.16.3 to 0.16.4. [[GH-566](https://github.com/hashicorp/hcp-terraform-operator/pull/566)]
* Bump `k8s.io/api` from 0.31.3 to 0.31.6. [[GH-571](https://github.com/hashicorp/hcp-terraform-operator/pull/571)]
* Bump `k8s.io/apimachinery` from 0.31.3 to 0.31.6. [[GH-571](https://github.com/hashicorp/hcp-terraform-operator/pull/571)]
* Bump `k8s.io/client-go` from 0.31.3 to 0.31.6. [[GH-571](https://github.com/hashicorp/hcp-terraform-operator/pull/571)]
* Bump `sigs.k8s.io/controller-runtimeg` from 0.19.2 to 0.19.7. [[GH-571](https://github.com/hashicorp/hcp-terraform-operator/pull/571)]

## 2.8.0 (February 10, 2025)

NOTES:

* The `Workspace` CRD has been changed. Please follow the Helm chart instructions on how to upgrade it. [[GH-497](https://github.com/hashicorp/hcp-terraform-operator/pull/497)]

ENHANCEMENTS:

* `Workspace`: Add the ability to attach variable sets to a workspace via a new optional field `spec.variableSets`. [[GH-497](https://github.com/hashicorp/hcp-terraform-operator/pull/497)]

DEPENDENCIES:

* Bump `github.com/hashicorp/go-slug` from 0.16.1 to 0.16.3. [[GH-549](https://github.com/hashicorp/hcp-terraform-operator/pull/549)]

## 2.7.1 (December 04, 2024)

BREAKING CHANGES:

* `Helm Chart`: The `customCAcertificates` value has been replaced to accept a base64-encoded CA bundle instead of a file path. This change aims to simplify the installation/upgrade workflow. [[GH-516](https://github.com/hashicorp/hcp-terraform-operator/pull/516)]

NOTES:

* `Helm Chart`: The default value of `operator.syncPeriod` has changed from 5 minutes to 1 hour to reduce unnecessary reconciliation. [[GH-512](https://github.com/hashicorp/hcp-terraform-operator/pull/512)]

BUG FIXES:

* `Workspace`: Fix an issue where `spec.agentPool` can be set even when `spec.executionMode` is not set to `agent`. [[GH-504](https://github.com/hashicorp/hcp-terraform-operator/pull/504)]
* `Helm Chart`: Fix an issue that prevented custom CA certificates from being attached to the pod volume. The `customCAcertificates` value now refers to a base64-encoded CRT bundle instead of a file path. [[GH-516](https://github.com/hashicorp/hcp-terraform-operator/pull/516)]

ENHANCEMENTS:

* `Helm Chart`: Add the ability to configure `affinity` and `tolerations` for the Deployment of the operator. [[GH-495](https://github.com/hashicorp/hcp-terraform-operator/pull/495)]
* `Helm Chart`: Add the ability to configure additional labels for the Operator pod. [[GH-522](https://github.com/hashicorp/hcp-terraform-operator/pull/522)]

DEPENDENCIES:

* Bump `kube-rbac-proxy` from 0.18.0 to 0.18.2. [[GH-514](https://github.com/hashicorp/hcp-terraform-operator/pull/514)] [[GH-531](https://github.com/hashicorp/hcp-terraform-operator/pull/531)]
* Bump `github.com/hashicorp/go-tfe` from 1.62.0 to 1.71.0. [[GH-508](https://github.com/hashicorp/hcp-terraform-operator/pull/508)] [[GH-532](https://github.com/hashicorp/hcp-terraform-operator/pull/532)]
* Bump `github.com/hashicorp/go-slug` from 0.15.2 to 0.16.1. [[GH-508](https://github.com/hashicorp/hcp-terraform-operator/pull/508)] [[GH-519](https://github.com/hashicorp/hcp-terraform-operator/pull/519)]
* Bump `k8s.io/api` from 0.30.3 to 0.31.3. [[GH-525](https://github.com/hashicorp/hcp-terraform-operator/pull/525)] [[GH-527](https://github.com/hashicorp/hcp-terraform-operator/pull/527)]
* Bump `k8s.io/apimachinery` from 0.30.3 to 0.31.3. [[GH-525](https://github.com/hashicorp/hcp-terraform-operator/pull/525)] [[GH-526](https://github.com/hashicorp/hcp-terraform-operator/pull/526)]
* Bump `sigs.k8s.io/controller-runtime` from 0.18.4 to 0.19.2. [[GH-525](https://github.com/hashicorp/hcp-terraform-operator/pull/525)]
* Bump `k8s.io/client-go` from 0.30.3 to 0.31.3. [[GH-525](https://github.com/hashicorp/hcp-terraform-operator/pull/525)] [[GH-527](https://github.com/hashicorp/hcp-terraform-operator/pull/527)]

## Community Contributors :raised_hands:

- @baptman21 made their contribution in https://github.com/hashicorp/hcp-terraform-operator/pull/495
- @mlflr made their contribution in https://github.com/hashicorp/hcp-terraform-operator/pull/522

## 2.7.0 (October 24, 2024)

BREAKING CHANGES:

* `Workspace`: The new field, `spec.deletionPolicy`, is set to `retain` by default, which changes the previous default controller behavior when resources are deleted. The previous behavior corresponded to the `force` deletion policy value. This change is considered safer in cases of accidental resource deletion, planned migration, or other scenarios involving the deletion of a custom resource. [[GH-470](https://github.com/hashicorp/hcp-terraform-operator/pull/470)]

NOTES:

* The `AgentPool` CRD has been changed. Please follow the Helm chart instructions on how to upgrade it. [[GH-494](https://github.com/hashicorp/hcp-terraform-operator/pull/494)]
* The `Workspace` CRD has been changed. Please follow the Helm chart instructions on how to upgrade it. [[GH-470](https://github.com/hashicorp/hcp-terraform-operator/pull/470)] [[GH-478](https://github.com/hashicorp/hcp-terraform-operator/pull/478)] [[GH-481](https://github.com/hashicorp/hcp-terraform-operator/pull/481)]

FEATURES:

* `Helm Chart`: Add a new value called `controllers.project.syncPeriod` to set the CLI option `--project-sync-period`. [[GH-479](https://github.com/hashicorp/hcp-terraform-operator/pull/479)]
* `Helm Chart`: Add a new value called `controllers.module.syncPeriod` to set the CLI option `--module-sync-period`. [[GH-480](https://github.com/hashicorp/hcp-terraform-operator/pull/480)]
* `Module`: Add a new CLI option called `--module-sync-period` to set the time interval for re-queuing Module resources once they are successfully reconciled. [[GH-480](https://github.com/hashicorp/hcp-terraform-operator/pull/480)]
* `Project`: Add a new CLI option called `--project-sync-period` to set the time interval for re-queuing Project resources once they are successfully reconciled. [[GH-479](https://github.com/hashicorp/hcp-terraform-operator/pull/479)]
* `Workspace`: Add a new field, `spec.deletionPolicy`, that specifies the behavior of the custom resource and its associated workspace when the custom resource is deleted. [[GH-470](https://github.com/hashicorp/hcp-terraform-operator/pull/470)]
* `Workspace`: Add the `destroy` deletion policy. The `spec.allowDestroyPlan` must be set to `true` for the controller to execute a destroy run. [[GH-489](https://github.com/hashicorp/hcp-terraform-operator/pull/489)]

ENHANCEMENTS:

* `Helm Chart`: Add the ability to configure the creation of the RBAC role and role binding. [[GH-491](https://github.com/hashicorp/hcp-terraform-operator/pull/491)]
* `AgentPool`: Update agent token reconciliation logic to reduce the number of API calls. [[GH-487](https://github.com/hashicorp/hcp-terraform-operator/pull/487)]
* `AgentPool`: Add the ability to configure labels and annotations for agent deployments. [[GH-494](https://github.com/hashicorp/hcp-terraform-operator/pull/494)]
* `Module`: Update reconciliation logic to proceed to the next step immediately after the configuration version is uploaded successfully and reduce delays in output fetching. [[GH-482](https://github.com/hashicorp/hcp-terraform-operator/pull/482)]
* `Workspace`: Update Notifications reconciliation logic to reduce the number of API calls. [[GH-477](https://github.com/hashicorp/hcp-terraform-operator/pull/477)]
* `Workspace`: Update SSH key reconciliation logic to reduce the number of API calls. [[GH-478](https://github.com/hashicorp/hcp-terraform-operator/pull/478)]
* `Workspace`: Update the default project ID reconciliation logic to avoid making an API call each time a workspace object is updated. [[GH-481](https://github.com/hashicorp/hcp-terraform-operator/pull/481)]

DEPENDENCIES:

* Bump `k8s.io/api` from 0.29.4 to 0.30.3. [[GH-452](https://github.com/hashicorp/hcp-terraform-operator/pull/452)] [[GH-466](https://github.com/hashicorp/hcp-terraform-operator/pull/466)]
* Bump `k8s.io/apimachinery` from 0.29.4 to 0.30.3. [[GH-452](https://github.com/hashicorp/hcp-terraform-operator/pull/452)] [[GH-456](https://github.com/hashicorp/hcp-terraform-operator/pull/456)]
* Bump `k8s.io/client-go` from 0.29.4 to 0.30.3. [[GH-452](https://github.com/hashicorp/hcp-terraform-operator/pull/452)] [[GH-466](https://github.com/hashicorp/hcp-terraform-operator/pull/466)]
* Bump `sigs.k8s.io/controller-runtime` from 0.17.3 to 0.18.4. [[GH-452](https://github.com/hashicorp/hcp-terraform-operator/pull/452)]

## Community Contributors :raised_hands:

- @domino-sbr1 made their contribution in https://github.com/hashicorp/hcp-terraform-operator/pull/491

## 2.6.1 (August 07, 2024)

BUG FIXES:

* `Workspace`: Fix an issue where the controller fails to update CR Status when CR gets modified during the reconciliation. [[GH-457](https://github.com/hashicorp/hcp-terraform-operator/pull/457)]
* `Workspace`: Fix an issue where, in some circumstances, the controller cannot properly handle the deletion event. [[GH-460](https://github.com/hashicorp/hcp-terraform-operator/pull/460)]

ENHANCEMENTS:

* `Helm Chart`: Add the ability to configure the Deployment priority class. [[GH-451](https://github.com/hashicorp/hcp-terraform-operator/pull/451)]

## Community Contributors :raised_hands:

- @vadim-kubasov made their contribution in https://github.com/hashicorp/hcp-terraform-operator/pull/451

## 2.6.0 (July 30, 2024)

NOTES:

* The `AgentPool` CRD has been changed. Please follow the Helm chart instructions on how to upgrade it. [[GH-441](https://github.com/hashicorp/hcp-terraform-operator/pull/441)]

BUG FIXES:

* `Project`: Fix an issue where calls to paginated API endpoints were only fetching the first page of results. [[GH-426](https://github.com/hashicorp/hcp-terraform-operator/pull/426)]
* `AgentPool`: Fix an issue where calls to paginated API endpoints were only fetching the first page of results. [[GH-426](https://github.com/hashicorp/hcp-terraform-operator/pull/426)]
* `Workspace`: Fix an issue where calls to paginated API endpoints were only fetching the first page of results. [[GH-426](https://github.com/hashicorp/hcp-terraform-operator/pull/426)]

ENHANCEMENTS:

* `Helm Chart`: Add the ability to configure the service account. [[GH-431](https://github.com/hashicorp/hcp-terraform-operator/pull/431)]
* `AgentPool`: Add the ability to configure scale-up and scale-down autoscaling times separately via the `cooldown.scaleUpSeconds` and `cooldown.scaleDownSeconds` attributes, respectively. [[GH-441](https://github.com/hashicorp/hcp-terraform-operator/pull/441)]

DEPENDENCIES:

* Bump `github.com/hashicorp/go-slug` from 0.15.0 to 0.15.2. [[GH-435](https://github.com/hashicorp/hcp-terraform-operator/pull/435)]
* Bump `github.com/onsi/ginkgo/v2` from 2.16.0 to 2.19.0. [[GH-415](https://github.com/hashicorp/hcp-terraform-operator/pull/415)]
* Bump `github.com/onsi/gomega` from 1.31.1 to 1.33.1. [[GH-415](https://github.com/hashicorp/hcp-terraform-operator/pull/415)]

## Community Contributors :raised_hands:

- @frgray made their contribution in https://github.com/hashicorp/hcp-terraform-operator/pull/431
- @jtdoepke made their contribution in https://github.com/hashicorp/hcp-terraform-operator/pull/426
- @omelnyk1 for sharing his Helm expertise and valuable feedback 🚀

## 2.5.0 (July 09, 2024)

NOTES:

* In upcoming releases, we shall proceed with renaming this project to HCP Terraform Operator for Kubernetes or simply HCP Terraform Operator. This measure is necessary in response to the recent announcement of [The Infrastructure Cloud](https://www.hashicorp.com/blog/introducing-the-infrastructure-cloud). The most noticeable change you can expect in version 2.6.0 is the renaming of this repository and related resources, such as the Helm chart and Docker Hub names. Please follow the changelogs for updates.

FEATURES:

* `Helm`: Add a new value called `controllers.agentPool.syncPeriod` to set the CLI option `--agent-pool-sync-period`. [[GH-421](https://github.com/hashicorp/terraform-cloud-operator/pull/421)]
* `AgentPool`: Add a new CLI option called `--agent-pool-sync-period` to set the time interval for re-queuing Agent Pool resources once they are successfully reconciled. [[GH-421](https://github.com/hashicorp/terraform-cloud-operator/pull/421)]

ENHANCEMENTS:

* `AgentPool`: Update reconciliation logic to reduce the number of API calls. The controller now reconciles custom resources after the cooldown period if applicable; otherwise, the default timer is applied. [[GH-420](https://github.com/hashicorp/terraform-cloud-operator/pull/420)]
* `AgentPool`: The agent auroscaling logic has been updated to decrease the frequency of API calls. The controller now utilizes the List Workspaces API call with filtering based on the current run status, thereby reducing the total number of API calls needed. [[GH-419](https://github.com/hashicorp/terraform-cloud-operator/pull/419)]
* `Helm Chart`: Add the ability to configure the security context of the Deployment pod and containers. [[GH-432](https://github.com/hashicorp/terraform-cloud-operator/pull/432)]

## Community Contributors :raised_hands:

- @vadim-kubasov made their contribution in https://github.com/hashicorp/terraform-cloud-operator/pull/432

## 2.4.1 (June 07, 2024)

NOTES:

* In upcoming releases, we shall proceed with renaming this project to HCP Terraform Operator for Kubernetes or simply HCP Terraform Operator. This measure is necessary in response to the recent announcement of [The Infrastructure Cloud](https://www.hashicorp.com/blog/introducing-the-infrastructure-cloud). The most noticeable change you can expect in version 2.6.0 is the renaming of this repository and related resources, such as the Helm chart and Docker Hub names. Please follow the changelogs for updates.

BUG FIXES:

* `Module`: Fix an issue where the controller cannot create ConfigMap and Secret for outputs when a Module custom resource `metadata.name` is longer than 63 characters. This issue occurs because the controller uses the custom resource name as the value for the ModuleName label in ConfigMap and Secret. The `ModuleName` label has been removed." [[GH-423](https://github.com/hashicorp/terraform-cloud-operator/pull/423)]
* `Workspace`: Fix an issue where the controller cannot create ConfigMap and Secret for outputs when a Workspace custom resource `metadata.name` is longer than 63 characters. This issue occurs because the controller uses the custom resource name as the value for the `workspaceName` label in ConfigMap and Secret. The `workspaceName` label has been removed. [[GH-423](https://github.com/hashicorp/terraform-cloud-operator/pull/423)]

DEPENDENCIES:

* Bump `github.com/hashicorp/go-tfe` from 1.49.0 to 1.55.0. [[GH-422](https://github.com/hashicorp/terraform-cloud-operator/pull/422)]
* Bump `kube-rbac-proxy` from 0.17.0 to 0.18.0. [[GH-424](https://github.com/hashicorp/terraform-cloud-operator/pull/424)]

## Community Contributors :raised_hands:

- @jtdoepke made their contribution in https://github.com/hashicorp/terraform-cloud-operator/pull/423
- @nabadger for constantly providing us with a valuable feedback :rocket:

## 2.4.0 (May 07, 2024)

NOTES:

* The `Workspace` CRD has been changed. Please follow the Helm chart instructions on how to upgrade it. [[GH-390](https://github.com/hashicorp/terraform-cloud-operator/pull/390)]
* In upcoming releases, we shall proceed with renaming this project to HCP Terraform Operator for Kubernetes or simply HCP Terraform Operator. This measure is necessary in response to the recent announcement of [The Infrastructure Cloud](https://www.hashicorp.com/blog/introducing-the-infrastructure-cloud). The most noticeable change you can expect in version 2.6.0 is the renaming of this repository and related resources, such as the Helm chart and Docker Hub names. Please follow the changelogs for updates. [[GH-393](https://github.com/hashicorp/terraform-cloud-operator/pull/393)]

BUG FIXES:

* `Workspace`: Fix an issue when the controller panics while accessing the default Project. [[GH-394](https://github.com/hashicorp/terraform-cloud-operator/pull/394)]

FEATURES:

* `Workspace`: Add a new CLI option called `--workspace-sync-period` to set the time interval for re-queuing Workspace resources once they are successfully reconciled. [[GH-391](https://github.com/hashicorp/terraform-cloud-operator/pull/391)]
* `Helm`: Add a new value called `controllers.workspace.syncPeriod` to set the CLI option `--workspace-sync-period`. [[GH-391](https://github.com/hashicorp/terraform-cloud-operator/pull/391)]

ENHANCEMENTS:

* `Workspace`: Update variables reconciliation logic to reduce the number of API calls. [[GH-390](https://github.com/hashicorp/terraform-cloud-operator/pull/390)]

DEPENDENCIES:

* Bump `github.com/hashicorp/go-tfe` from 1.47.1 to 1.49.0. [[GH-378](https://github.com/hashicorp/terraform-cloud-operator/pull/378)]
* Bump `kube-rbac-proxy` from 0.16.0 to 0.17.0. [[GH-392](https://github.com/hashicorp/terraform-cloud-operator/pull/392)]
* Bump `k8s.io/api` from 0.29.2 to 0.29.4. [[GH-399](https://github.com/hashicorp/terraform-cloud-operator/pull/399)]
* Bump `k8s.io/apimachinery` from 0.29.2 to 0.29.4. [[GH-399](https://github.com/hashicorp/terraform-cloud-operator/pull/399)]
* Bump `k8s.io/client-go` from 0.29.2 to 0.29.4. [[GH-399](https://github.com/hashicorp/terraform-cloud-operator/pull/399)]
* Bump `sigs.k8s.io/controller-runtime` from 0.17.2 to 0.17.3. [[GH-399](https://github.com/hashicorp/terraform-cloud-operator/pull/399)]

## 2.3.0 (March 18, 2024)

BUG FIXES:

* `Workspace`: Fix an issue when the boolean attribute `allowDestroyPlan`, with a default value of `true`, is set to `false` but unexpectedly gets mutated back to `true` during creation. [[GH-337](https://github.com/hashicorp/terraform-cloud-operator/pull/337)]

FEATURES:

* `Workspace`: Add annotations support to trigger a new run. New annotations: `workspace.app.terraform.io/run-new`, `workspace.app.terraform.io/run-type`, `workspace.app.terraform.io/run-terraform-version`. [[GH-364](https://github.com/hashicorp/terraform-cloud-operator/pull/364)]

NOTES:

* The `Workspace` CRD has been changed. Please follow the Helm chart instructions on how to upgrade it. [[GH-342](https://github.com/hashicorp/terraform-cloud-operator/pull/342)] [[GH-364](https://github.com/hashicorp/terraform-cloud-operator/pull/364)]

ENHANCEMENTS:

* `Helm Chart`: Add a new attribute `imagePullSecrets` to enable pulling the Operator images from private repositories. [[GH-326](https://github.com/hashicorp/terraform-cloud-operator/pull/326)]
* `Module`: The output will be synchronized when the run status is either `applied` or `planned_and_finished`, whereas previously it was only `applied`. [[GH-351](https://github.com/hashicorp/terraform-cloud-operator/pull/351)]
* `Workspace`: Add the ability to configure automatic speculative plan on pull requests when using Version Control via a new optional field `spec.versionControl.speculativePlans`. Default to `true`. [[GH-342](https://github.com/hashicorp/terraform-cloud-operator/pull/342)]
* `Workspace`: The output will be synchronized when the run status is either `applied` or `planned_and_finished`, whereas previously it was only `applied`. [[GH-345](https://github.com/hashicorp/terraform-cloud-operator/pull/345)]
* `Workspace`: The output will be synchronized faster and require fewer API calls. [[GH-345](https://github.com/hashicorp/terraform-cloud-operator/pull/345)]
* `Workspace`: The `status` now includes the current configuration version in `status.run.configurationVersion`. [[GH-353](https://github.com/hashicorp/terraform-cloud-operator/pull/353)]
* `Workspace`: The controller will reconcile the workspace more frequently during incomplete runs to synchronize outputs faster. [[GH-353](https://github.com/hashicorp/terraform-cloud-operator/pull/353)]
* `Operator`: Add a new option `--version` to print out the version of the operator. [[GH-365](https://github.com/hashicorp/terraform-cloud-operator/pull/365)]

DEPENDENCIES:

* Bump `kube-rbac-proxy` image from 0.15.0 to 0.16.0. [[GH-335](https://github.com/hashicorp/terraform-cloud-operator/pull/335)]
* Bump `github.com/onsi/ginkgo/v2` from 2.13.2 to 2.16.0. [[GH-328](https://github.com/hashicorp/terraform-cloud-operator/pull/328)] [[GH-363](https://github.com/hashicorp/terraform-cloud-operator/pull/363)]
* Bump `github.com/onsi/gomega` from 1.29.0 to 1.31.1. [[GH-328](https://github.com/hashicorp/terraform-cloud-operator/pull/328)] [[GH-329](https://github.com/hashicorp/terraform-cloud-operator/pull/329)]
* Bump `github.com/hashicorp/go-tfe` from 1.41.0 to 1.47.1. [[GH-332](https://github.com/hashicorp/terraform-cloud-operator/pull/332)] [[GH-354](https://github.com/hashicorp/terraform-cloud-operator/pull/354)] [[GH-366](https://github.com/hashicorp/terraform-cloud-operator/pull/366)]
* Bump `github.com/hashicorp/go-slug` from 0.13.3 to 0.14.0. [[GH-332](https://github.com/hashicorp/terraform-cloud-operator/pull/332)] [[GH-354](https://github.com/hashicorp/terraform-cloud-operator/pull/354)]
* Bump `sigs.k8s.io/controller-runtime` from 0.15.3 to 0.17.2. [[GH-340](https://github.com/hashicorp/terraform-cloud-operator/pull/340)] [[GH-358](https://github.com/hashicorp/terraform-cloud-operator/pull/358)]
* Bump `k8s.io/api` from 0.27.8 to 0.29.1. [[GH-340](https://github.com/hashicorp/terraform-cloud-operator/pull/340)] [[GH-356](https://github.com/hashicorp/terraform-cloud-operator/pull/356)]
* Bump `k8s.io/apimachinery` from 0.27.8 to 0.29.2. [[GH-340](https://github.com/hashicorp/terraform-cloud-operator/pull/340)] [[GH-356](https://github.com/hashicorp/terraform-cloud-operator/pull/356)]
* Bump `k8s.io/client-go` from 0.27.8 to 0.29.2. [[GH-340](https://github.com/hashicorp/terraform-cloud-operator/pull/340)] [[GH-356](https://github.com/hashicorp/terraform-cloud-operator/pull/356)]
* Bump `go.uber.org/zap` from 1.26.0 to 1.27.0. [[GH-355](https://github.com/hashicorp/terraform-cloud-operator/pull/355)]
* Bump `google.golang.org/protobuf` from 1.31.0 to 1.33.0. [[GH-367](https://github.com/hashicorp/terraform-cloud-operator/pull/367)]

## Community Contributors :raised_hands:

- @bFekete made their contribution in https://github.com/hashicorp/terraform-cloud-operator/pull/326

## 2.2.0 (January 16, 2024)

FEATURES:

* `Project`: add a new controller `Project` that allows managing Terraform Cloud Projects. [[GH-309](https://github.com/hashicorp/terraform-cloud-operator/pull/309)].

DEPENDENCIES:

* Bump `k8s.io/api` from 0.27.7 to 0.27.8. [[GH-306](https://github.com/hashicorp/terraform-cloud-operator/pull/306)]
* Bump `k8s.io/apimachinery` from 0.27.7 to 0.27.8. [[GH-306](https://github.com/hashicorp/terraform-cloud-operator/pull/306)]
* Bump `k8s.io/client-go` from 0.27.7 to 0.27.8. [[GH-306](https://github.com/hashicorp/terraform-cloud-operator/pull/306)]
* Bump `github.com/go-logr/zapr` from 1.2.4 to 1.3.0. [[GH-305](https://github.com/hashicorp/terraform-cloud-operator/pull/305)]
* Bump `github.com/onsi/ginkgo/v2` from 2.13.0 to 2.13.2. [[GH-307](https://github.com/hashicorp/terraform-cloud-operator/pull/307)]
* Bump `github.com/hashicorp/go-tfe` from 1.37.0 to 1.41.0. [[GH-316](https://github.com/hashicorp/terraform-cloud-operator/pull/316)]
* Bump `github.com/hashicorp/go-slug` from 0.12.2 to 0.13.3. [[GH-316](https://github.com/hashicorp/terraform-cloud-operator/pull/316)]
* Bump `github.com/go-logr/logr` from 1.3.0 to 1.4.1. [[GH-317](https://github.com/hashicorp/terraform-cloud-operator/pull/317)]
* Bump `kube-rbac-proxy` image from 0.14.4 to 0.15.0. [[GH-320](https://github.com/hashicorp/terraform-cloud-operator/pull/320)]

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
* Bump `k8s.io/client-go` from 0.27.5 to 0.27.7. [[GH-264](https://github.com/hashicorp/terraform-cloud-operator/pull/264)] [[GH-292](https://github.com/hashicorp/terraform-cloud-operator/pull/292)]
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

