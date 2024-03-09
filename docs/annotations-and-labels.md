# Annotations and Labels used by Terraform Cloud Operator

## Annotations

| Annotation key | Target resources | Possible values | Description |
|----------------|---------------------|-----------------|-------------|
| `workspace.app.terraform.io/run-at` | Workspace | any | |
| `workspace.app.terraform.io/run-restarted-at` | Workspace | any | |
| `workspace.app.terraform.io/run-type` | Workspace | `plan`, `apply`, `refresh` | |
| `workspace.app.terraform.io/run-terraform-version` | Workspace | | |

## Labels

Terraform Cloud Operator uses no labels.
