# Annotations and Labels used by HCP Terraform Operator

## Annotations

| Annotation key | Target resources | Possible values | Description |
| --- | --- | --- | --- |
| `workspace.app.terraform.io/run-new` | Workspace | `"true"` | Set this annotation to `"true"` to trigger a new run. Example: `kubectl annotate workspace <WORKSPACE-NAME> workspace.app.terraform.io/run-new="true"`. |
| `workspace.app.terraform.io/run-type` | Workspace | `plan`, `apply`, `refresh` | Specifies the run type. Changing this annotation does not start a new run. Refer to [Run Modes and Options](https://developer.hashicorp.com/terraform/cloud-docs/run/modes-and-options) for more information. Defaults to `"plan"`. |
| `workspace.app.terraform.io/run-terraform-version` | Workspace | Any valid Terraform version | Specifies the Terraform version to use. Changing this annotation does not start a new run. Only valid when the annotation `workspace.app.terraform.io/run-type` is set to `plan`. Defaults to the Workspace version. |
| `app.terraform.io/paused` | CRD[All] | `"true"` | Set this annotation to `"true"` to pause reconciliation for the custom resource. While paused, the operator will skip reconciliation for the annotated resource, even if the custom resource changes. Deletion logic will still be executed. Example: `kubectl annotate workspace <WORKSPACE-NAME> app.terraform.io/paused="true"`. |

## Labels

| Label key | Target resources | Possible values | Description |
| --- | --- | --- | --- |
| `app.terraform.io/crd-schema-version` | CRD[All] | A valid calendar versioning tag format: `vYY.MM.PATCH`. | The label is used to version the HCP Operator CRD. The version is updated whenever there is a change in the schema, following the [calendar versioning](https://calver.org/) approach. |
| `agentpool.app.terraform.io/pool-name` | Pod[Agent] | Any valid AgentPool name | Associate the resource with a specific agent pool by specifying the name of the agent pool. |
| `agentpool.app.terraform.io/pool-id` | Pod[Agent] | Any valid AgentPool ID | Associate the resource with a specific agent pool by specifying the ID of the agent pool. |
