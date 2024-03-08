# Annotations and Labels used by Terraform Cloud Operator

## Annotations

| Annotation key | Target resources | Possible values | Description |
|----------------|---------------------|-----------------|-------------|
| `workspace.app.terraform.io/run-at` | Workspace | any | |
| `workspace.app.terraform.io/run-restarted-at` | Workspace | any | |
| `workspace.app.terraform.io/run-type` | Workspace | `plan`, `apply`, `refresh` | |

## Labels

Terraform Cloud Operator uses no labels.
