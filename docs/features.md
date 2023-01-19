# Supported Terraform Cloud Features

## Agent Pool

The following Terraform Cloud Agent Pool features are supported by the `AgentPool` controller of the Operator:

- Agent Pool Management -- the Operator can create, update or delete Agent Pools;
- Agent Token Management -- the Operator can create, update or delete Agent Tokens, but does not manage Agent Lifecycle.

## Module

The `Module` controller implements API-driven Run workflow by generating Terraform module code with the following features:

- Outputs -- all outputs produced by the Module run will be stored in Kubernetes ConfigMap and/or Secret;
- Source -- where to find the source code for the module;
- Version -- the module version.

## Workspace

The following Terraform Cloud Workspace features are supported by the `Workspace` controller of the Operator:

- Agents
- Allow Destroy Plan
- Apply Method
- Description
- Execution Mode
- Outputs
- Remote State Sharing
- Run Triggers
- SSH Key
- Tags
- Team Access
- Terraform Version
- VCS
- Variables
- Working Directory

More information about each feature you can find in the [Terraform Cloud documentation](https://developer.hashicorp.com/terraform/cloud-docs/workspaces).

If a feature you are looking for is not implemented yet please let us know by opening an issue with a feature request, or adding your reaction to an existing one.
