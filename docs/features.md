# Supported Terraform Cloud Features

The Terraform Cloud Operator allows you to interact with various aspects of Terraform Cloud, such as agent pools, agent scaling, Terraform module execution, and workspace management, through Kubernetes controllers. These controllers enable you to automate and manage Terraform Cloud resources using custom resources in Kubernetes. Let's break down the mentioned features.

## Agent Pool

Agent pools in Terraform Cloud are used to manage the execution environment for Terraform runs. The Terraform Cloud Operator likely allows you to create and manage agent pools as part of your Kubernetes infrastructure. For example, you might create a custom resource in Kubernetes to define an agent pool and let the operator handle its provisioning and scaling.

Let's take a look at how to create a new agent pool with the name `agent-pool-development` and generate a single agent token to it with the name `token-red`:

```yaml
---
apiVersion: app.terraform.io/v1alpha2
kind: AgentPool
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: tfc-operator
      key: token
  name: agent-pool-development
  agentTokens:
    - name: token-red
```

The token that is generated, named "token-red," will be accessible within a Kubernetes secret.

We can expand our example by introducing the agent auto-scaling feature.

```yaml
---
apiVersion: app.terraform.io/v1alpha2
kind: AgentPool
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: tfc-operator
      key: token
  name: agent-pool-development
  agentTokens:
  - name: token-red
  agentDeployment:
   replicas: 1
  autoscaling:
    targetWorkspaces:
    - name: us-west-development
    - id: ws-NUVHA9feCXzAmPHx
    - wildcardName: eu-development-*
    minReplicas: 1
    maxReplicas: 3
```

The operator will ensure that at least one agent Pod is continuously running, and it can dynamically scale the number of Pods up to a maximum of three based on the workload or resource demand. To achieve this, the operator will monitor the resource demand by observing the load of the designated workspaces, which can be specified by their name, ID, or through wildcard patterns. When the workload decreases, the operator will scale down the node to release valuable resources.

To explore more advanced options, please refer to the [API reference](./api-reference.md#agentpool) documentation.

## Module

The Module controller enforces an [API-driven Run workflow](https://developer.hashicorp.com/terraform/cloud-docs/run/api) and enables the execution of Terraform modules within various workspaces as needed.

Let's take a look at how to run module `redeux/terraform-cloud-agent/kubernetes` with a specific version `1.0.1` within the designated workspace named `workspace-name`.

```yaml
---
apiVersion: app.terraform.io/v1alpha2
kind: Module
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: tfc-operator
      key: token
  module:
    source: redeux/terraform-cloud-agent/kubernetes
    version: 1.0.1
  workspace:
    name: workspace-name
  variables:
  - name: variable_a
  - name: variable_b
  outputs:
  - name: output_a
  - name: output_b
```

The operator will transmit the variables `variable_a` and `variable_b` to the module and synchronize the outputs `output_a` and `output_b` with either a Kubernetes secret or a config map, depending on the sensitivity of the output. The variables need to be accessible within the workspace.

To explore more advanced options, please refer to the [API reference](./api-reference.md#module) documentation.

## Workspace

Workspaces in Terraform Cloud are used to organize and manage Terraform configurations. The operator may allow you to create, configure, and manage workspaces directly from Kubernetes, simplifying workspace management.

Let's take a closed look at how to create a workspace.

```yaml
---
apiVersion: app.terraform.io/v1alpha2
kind: Workspace
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: tfc-operator
      key: token
  name: us-west-development
  description: US West development workspace
  terraformVersion: 1.6.2
  applyMethod: auto
  agentPool:
    name: ap-us-west-development
  terraformVariables:
    - name: nodes
      value: 2
    - name: rds-secret
      sensitive: true
      valueFrom:
        secretKeyRef:
          name: us-west-development-secrets
          key: rds-secret
  runTasks:
    - name: rt-us-west-development
      stage: pre_plan
```

The operator will establish a new workspace named `us-west-development` with Terraform version `1.6.2`. This workspace will have two variables, namely `nodes` and `rds-secret`. The variable `rds-secret` is treated as sensitive, and it will be sourced from a Kubernetes secret named `us-west-development-secrets`.

The Terraform code will be automatically executed, due to the option `applyMethod` set to `auto`, and this will occur on an agent originating from the `ap-us-west-development` agent pool. Furthermore, the run task, denoted as `rt-us-west-development`, is scheduled to run at the `pre_plan` stage.

The operator will also manage the synchronization of Terraform code execution outputs. This synchronization process will either involve a Kubernetes secret or a config map, depending on the sensitivity of the output data.

It's important to note that any external modifications made to the operator's setup will be rolled back to match the state specified in the custom resource definition.

To explore more advanced options, please refer to the [API reference](./api-reference.md#workspace) documentation.
