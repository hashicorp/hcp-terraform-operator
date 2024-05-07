# Annotations and Labels used by HCP Terraform Operator

## Annotations

| Annotation key | Target resources | Possible values | Description |
| --- | --- | --- | --- |
| `workspace.app.terraform.io/run-new` | Workspace | `"true"` | Set this annotation to `"true"` to trigger a new run. Example: `kubectl annotate workspace <WORKSPACE-NAME> workspace.app.terraform.io/run-new="true"`. |
| `workspace.app.terraform.io/run-type` | Workspace | `plan`, `apply`, `refresh` | Specifies the run type. Changing this annotation does not start a new run. Refer to [Run Modes and Options](https://developer.hashicorp.com/terraform/cloud-docs/run/modes-and-options) for more information. Defaults to `"plan"`. |
| `workspace.app.terraform.io/run-terraform-version` | Workspace | Any valid Terraform version | Specifies the Terraform version to use. Changing this annotation does not start a new run. Only valid when the annotation `workspace.app.terraform.io/run-type` is set to `plan`. Defaults to the Workspace version. |

## Labels

HCP Terraform Operator does not use any labels.
