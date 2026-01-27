# Stack

The `Stack` custom resource allows you to manage HCP Terraform Stacks through Kubernetes.

## Overview

HCP Terraform Stacks is a feature that enables infrastructure orchestration at scale. The Stack CRD provides a Kubernetes-native way to create and manage HCP Terraform Stacks.

More information:
- [HCP Terraform Stacks Documentation](https://developer.hashicorp.com/terraform/cloud-docs/stacks)

## Example Usage

### Basic Stack

```yaml
apiVersion: app.terraform.io/v1alpha2
kind: Stack
metadata:
  name: example-stack
spec:
  name: "my-stack"
  organization: "my-organization"
  token:
    secretKeyRef:
      name: tfc-operator
      key: token
  description: "Example HCP Terraform Stack"
```

### Stack with VCS Repository

```yaml
apiVersion: app.terraform.io/v1alpha2
kind: Stack
metadata:
  name: stack-with-vcs
spec:
  name: "production-stack"
  organization: "my-organization"
  token:
    secretKeyRef:
      name: tfc-operator
      key: token
  vcsRepo:
    identifier: "ot-abc123xyz"
    branch: "main"
    path: "stacks/production"
  terraformVersion: "1.7.0"
```

### Stack with Variables and Deployments

```yaml
apiVersion: app.terraform.io/v1alpha2
kind: Stack
metadata:
  name: stack-with-config
spec:
  name: "configured-stack"
  organization: "my-organization"
  token:
    secretKeyRef:
      name: tfc-operator
      key: token
  project:
    name: "infrastructure"
  environmentVariables:
    - name: "AWS_REGION"
      value: "us-west-2"
    - name: "ENVIRONMENT"
      value: "production"
  terraformVariables:
    - name: "instance_count"
      value: "5"
    - name: "tags"
      value: '{"Environment": "production"}'
      hcl: true
  deployment:
    names:
      - "production"
      - "staging"
  deletionPolicy: "retain"
```

## Spec Reference

### Stack Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Name of the Stack |
| `organization` | string | Yes | Organization name where the Stack will be created |
| `token` | [Token](#token) | Yes | API token reference for authentication |
| `project` | [StackProject](#stackproject) | No | Project where the Stack will be created |
| `vcsRepo` | [StackVCSRepo](#stackvcsrepo) | No | VCS repository configuration |
| `description` | string | No | Description of the Stack |
| `terraformVersion` | string | No | Terraform version to use (e.g., "1.7.0") |
| `environmentVariables` | [][Variable](#variable) | No | Environment variables for all deployments |
| `terraformVariables` | [][Variable](#variable) | No | Terraform variables for all deployments |
| `deployment` | [StackDeployment](#stackdeployment) | No | Deployment configuration |
| `deletionPolicy` | string | No | Deletion policy: `retain` (default) or `delete` |

### StackProject

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | No* | Project ID (pattern: `^prj-[a-zA-Z0-9]+$`) |
| `name` | string | No* | Project name |

*Either `id` or `name` must be specified, but not both.

### StackVCSRepo

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `identifier` | string | Yes | VCS connection OAuth token ID (pattern: `^ot-[a-zA-Z0-9]+$`) |
| `branch` | string | No | Repository branch (defaults to repository's default branch) |
| `path` | string | No | Path to the Stack configuration file in the repository |
| `ghaInstallationId` | string | No | GitHub App installation ID (required for GitHub App connections) |

### StackDeployment

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `names` | []string | Yes | Names of the deployments to create |

### Variable

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Variable name |
| `description` | string | No | Variable description |
| `hcl` | bool | No | Parse as HCL (default: false) |
| `sensitive` | bool | No | Mark as sensitive (default: false) |
| `value` | string | No* | Variable value |
| `valueFrom` | [ValueFrom](#valuefrom) | No* | Source for variable value |

*Either `value` or `valueFrom` must be specified, but not both.

### Token

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `secretKeyRef` | SecretKeySelector | Yes | Reference to a Kubernetes Secret containing the API token |

## Status Reference

### Stack Status

| Field | Type | Description |
|-------|------|-------------|
| `observedGeneration` | int64 | Generation of the resource that was most recently reconciled |
| `stackID` | string | HCP Terraform Stack ID |
| `updatedAt` | int64 | Last update timestamp |
| `terraformVersion` | string | Terraform version being used |
| `deployments` | [][DeploymentStatus](#deploymentstatus) | Status of deployments |
| `defaultProjectID` | string | Default organization project ID |

### DeploymentStatus

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Deployment name |
| `id` | string | Deployment ID |
| `status` | string | Deployment status |
| `updatedAt` | int64 | Last updated timestamp |

## Deletion Policy

The `deletionPolicy` field controls what happens when the Stack custom resource is deleted:

- **`retain`** (default): The Stack is retained in HCP Terraform when the custom resource is deleted
- **`delete`**: The Stack and all its deployments are deleted from HCP Terraform when the custom resource is deleted

## Annotations

The Stack controller supports the following annotations:

- `app.terraform.io/paused: "true"` - Pause reconciliation for this Stack

## Notes

- The Stack CRD is designed to work with HCP Terraform Stacks, which is a feature for infrastructure orchestration
- Ensure your HCP Terraform organization has Stacks enabled
- The API token must have appropriate permissions to manage Stacks
- This is a basic implementation; actual HCP Terraform Stacks API integration would require the go-tfe library to support Stacks endpoints

## See Also

- [Workspace](workspace.md)
- [Module](module.md)
- [Project](project.md)