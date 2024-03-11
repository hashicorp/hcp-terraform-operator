# Annotations and Labels used by Terraform Cloud Operator

## Annotations

| Annotation key | Target resources | Possible values | Description |
|----------------|---------------------|-----------------|-------------|
| `workspace.app.terraform.io/run-at` | Workspace | any | Set or update this annotation to trigger a new run. Although a value can be any non-empty string, we recommend using a timestamp for better observability.<br><br> Example: ```kubectl annotate workspace <WORKSPACE-NAME> workspace.app.terraform.io/run-at=`date -u -Iseconds` --overwrite```. |
| `workspace.app.terraform.io/run-restarted-at` | Workspace | any | This annotation, set by the operator, controls whether a run has been triggered or not.<br><br> Changing this annotation **DOES NOT** trigger a new run. |
| `workspace.app.terraform.io/run-type` | Workspace | `plan`, `apply`, `refresh` | Specify the run type. More information: [Run Modes and Options](https://developer.hashicorp.com/terraform/cloud-docs/run/modes-and-options). It defaults to `plan`.<br><br> Changing this annotation **DOES NOT** trigger a new run. |
| `workspace.app.terraform.io/run-terraform-version` | Workspace | valid Terraform version | Specify the Terraform version to use. This setting works only when the annotation `workspace.app.terraform.io/run-type` is set to `plan`. It defaults to the Workspace version.<br><br> Changing this annotation **DOES NOT** trigger a new run. |

## Labels

Terraform Cloud Operator uses no labels.
