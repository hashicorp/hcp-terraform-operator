# API Reference

## Packages
- [app.terraform.io/v1alpha2](#appterraformiov1alpha2)


## app.terraform.io/v1alpha2

Package v1alpha2 contains API Schema definitions for the app v1alpha2 API group

### Resource Types
- [AgentPool](#agentpool)
- [Module](#module)
- [Project](#project)
- [Workspace](#workspace)



#### AgentDeployment





_Appears in:_
- [AgentPoolSpec](#agentpoolspec)

| Field | Description |
| --- | --- |
| `replicas` _integer_ |  |
| `spec` _[PodSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#podspec-v1-core)_ |  |
| `annotations` _object (keys:string, values:string)_ | The annotations that the operator will apply to the pod template in the deployment. |
| `labels` _object (keys:string, values:string)_ | The labels that the operator will apply to the pod template in the deployment. |


#### AgentDeploymentAutoscaling



AgentDeploymentAutoscaling allows you to configure the operator
to scale the deployment for an AgentPool up and down to meet demand.

_Appears in:_
- [AgentPoolSpec](#agentpoolspec)

| Field | Description |
| --- | --- |
| `maxReplicas` _integer_ | MaxReplicas is the maximum number of replicas for the Agent deployment. |
| `minReplicas` _integer_ | MinReplicas is the minimum number of replicas for the Agent deployment. |
| `targetWorkspaces` _[TargetWorkspace](#targetworkspace)_ | DEPRECATED: This field has been deprecated since 2.9.0 and will be removed in future versions.<br />TargetWorkspaces is a list of HCP Terraform Workspaces which<br />the agent pool should scale up to meet demand. When this field<br />is ommited the autoscaler will target all workspaces that are<br />associated with the AgentPool. |
| `cooldownPeriodSeconds` _integer_ | CooldownPeriodSeconds is the time to wait between scaling events. Defaults to 300. |
| `cooldownPeriod` _[AgentDeploymentAutoscalingCooldownPeriod](#agentdeploymentautoscalingcooldownperiod)_ | CoolDownPeriod configures the period to wait between scaling up and scaling down |


#### AgentDeploymentAutoscalingCooldownPeriod



AgentDeploymentAutoscalingCooldownPeriod configures the period to wait between scaling up and scaling down

_Appears in:_
- [AgentDeploymentAutoscaling](#agentdeploymentautoscaling)

| Field | Description |
| --- | --- |
| `scaleUpSeconds` _integer_ | ScaleUpSeconds is the time to wait before scaling up. |
| `scaleDownSeconds` _integer_ | ScaleDownSeconds is the time to wait before scaling down. |


#### AgentDeploymentAutoscalingStatus



AgentDeploymentAutoscalingStatus

_Appears in:_
- [AgentPoolStatus](#agentpoolstatus)

| Field | Description |
| --- | --- |
| `desiredReplicas` _integer_ | Desired number of agent replicas |
| `lastScalingEvent` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#time-v1-meta)_ | Last time the agent pool was scaledx |


#### AgentPool



AgentPool manages HCP Terraform Agent Pools, HCP Terraform Agent Tokens and can perform HCP Terraform Agent scaling.
More infromation:
  - https://developer.hashicorp.com/terraform/cloud-docs/agents/agent-pools
  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens#agent-api-tokens
  - https://developer.hashicorp.com/terraform/cloud-docs/agents



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `app.terraform.io/v1alpha2`
| `kind` _string_ | `AgentPool`
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[AgentPoolSpec](#agentpoolspec)_ |  |


#### AgentPoolDeletionPolicy

_Underlying type:_ _string_

DeletionPolicy defines the strategy the Kubernetes operator uses when you delete a resource, either manually or by a system event.
You must use one of the following values:
- `retain`: When you delete the custom resource, the operator does not delete the agent pool.
- `destroy`: The operator will attempt to remove the managed HCP Terraform agent pool.

_Appears in:_
- [AgentPoolSpec](#agentpoolspec)



#### AgentPoolSpec



AgentPoolSpec defines the desired state of AgentPool.

_Appears in:_
- [AgentPool](#agentpool)

| Field | Description |
| --- | --- |
| `name` _string_ | Agent Pool name.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/agents/agent-pools |
| `organization` _string_ | Organization name where the Workspace will be created.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations |
| `token` _[Token](#token)_ | API Token to be used for API calls. |
| `agentTokens` _[AgentToken](#agenttoken) array_ | List of the agent tokens to generate. |
| `agentDeployment` _[AgentDeployment](#agentdeployment)_ | Agent deployment settings |
| `autoscaling` _[AgentDeploymentAutoscaling](#agentdeploymentautoscaling)_ | Agent deployment settings |
| `deletionPolicy` _[AgentPoolDeletionPolicy](#agentpooldeletionpolicy)_ | The Deletion Policy specifies the behavior of the custom resource and its associated agent pool when the custom resource is deleted.<br />- `retain`: When you delete the custom resource, the operator will remove only the custom resource.<br />  The HCP Terraform agent pool will be retained. The managed tokens will remain active on the HCP Terraform side; however, the corresponding secrets and managed agents will be removed.<br />- `destroy`: The operator will attempt to remove the managed HCP Terraform agent pool.<br />  On success, the managed agents and the corresponding secret with tokens will be removed along with the custom resource.<br />  On failure, the managed agents will be scaled down to 0, and the managed tokens, along with the corresponding secret, will be removed. The operator will continue attempting to remove the agent pool until it succeeds.<br />Default: `retain`. |




#### AgentToken



Agent Token is a secret token that a HCP Terraform Agent is used to connect to the HCP Terraform Agent Pool.
In `spec` only the field `Name` is allowed, the rest are used in `status`.
More infromation:
  - https://developer.hashicorp.com/terraform/cloud-docs/agents

_Appears in:_
- [AgentPoolSpec](#agentpoolspec)
- [AgentPoolStatus](#agentpoolstatus)

| Field | Description |
| --- | --- |
| `name` _string_ | Agent Token name. |
| `id` _string_ | Agent Token ID. |
| `createdAt` _integer_ | Timestamp of when the agent token was created. |
| `lastUsedAt` _integer_ | Timestamp of when the agent token was last used. |


#### ConfigurationVersionStatus



A configuration version is a resource used to reference the uploaded configuration files.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/api-docs/configuration-versions
  - https://developer.hashicorp.com/terraform/cloud-docs/run/api

_Appears in:_
- [ModuleStatus](#modulestatus)

| Field | Description |
| --- | --- |
| `id` _string_ | Configuration Version ID. |


#### ConsumerWorkspace



ConsumerWorkspace allows access to the state for specific workspaces within the same organization.
Only one of the fields `ID` or `Name` is allowed.
At least one of the fields `ID` or `Name` is mandatory.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/state#remote-state-access-controls

_Appears in:_
- [RemoteStateSharing](#remotestatesharing)

| Field | Description |
| --- | --- |
| `id` _string_ | Consumer Workspace ID.<br />Must match pattern: `^ws-[a-zA-Z0-9]+$` |
| `name` _string_ | Consumer Workspace name. |


#### CustomPermissions



Custom permissions let you assign specific, finer-grained permissions to a team than the broader fixed permission sets provide.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#custom-workspace-permissions

_Appears in:_
- [TeamAccess](#teamaccess)

| Field | Description |
| --- | --- |
| `runs` _string_ | Run access.<br />Must be one of the following values: `apply`, `plan`, `read`.<br />Default: `read`. |
| `runTasks` _boolean_ | Manage Workspace Run Tasks.<br />Default: `false`. |
| `sentinel` _string_ | Download Sentinel mocks.<br />Must be one of the following values: `none`, `read`.<br />Default: `none`. |
| `stateVersions` _string_ | State access.<br />Must be one of the following values: `none`, `read`, `read-outputs`, `write`.<br />Default: `none`. |
| `variables` _string_ | Variable access.<br />Must be one of the following values: `none`, `read`, `write`.<br />Default: `none`. |
| `workspaceLocking` _boolean_ | Lock/unlock workspace.<br />Default: `false`. |


#### CustomProjectPermissions



Custom permissions let you assign specific, finer-grained permissions to a team than the broader fixed permission sets provide.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#custom-project-permissions
  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#general-workspace-permissions

_Appears in:_
- [ProjectTeamAccess](#projectteamaccess)

| Field | Description |
| --- | --- |
| `projectAccess` _[ProjectSettingsPermissionType](#projectsettingspermissiontype)_ | Project access.<br />Must be one of the following values: `delete`, `read`, `update`.<br />Default: `read`. |
| `teamManagement` _[ProjectTeamsPermissionType](#projectteamspermissiontype)_ | Team management.<br />Must be one of the following values: `manage`, `none`, `read`.<br />Default: `none`. |
| `createWorkspace` _boolean_ | Allow users to create workspaces in the project.<br />This grants read access to all workspaces in the project.<br />Default: `false`. |
| `deleteWorkspace` _boolean_ | Allows users to delete workspaces in the project.<br />Default: `false`. |
| `moveWorkspace` _boolean_ | Allows users to move workspaces out of the project.<br />A user must have this permission on both the source and destination project to successfully move a workspace from one project to another.<br />Default: `false`. |
| `lockWorkspace` _boolean_ | Allows users to manually lock the workspace to temporarily prevent runs.<br />When a workspace's execution mode is set to "local", users must have this permission to perform local CLI runs using the workspace's state.<br />Default: `false`. |
| `runs` _[WorkspaceRunsPermissionType](#workspacerunspermissiontype)_ | Run access.<br />Must be one of the following values: `apply`, `plan`, `read`.<br />Default: `read`. |
| `runTasks` _boolean_ | Manage Workspace Run Tasks.<br />Default: `false`. |
| `sentinelMocks` _[WorkspaceSentinelMocksPermissionType](#workspacesentinelmockspermissiontype)_ | Download Sentinel mocks.<br />Must be one of the following values: `none`, `read`.<br />Default: `none`. |
| `stateVersions` _[WorkspaceStateVersionsPermissionType](#workspacestateversionspermissiontype)_ | State access.<br />Must be one of the following values: `none`, `read`, `read-outputs`, `write`.<br />Default: `none`. |
| `variables` _[WorkspaceVariablesPermissionType](#workspacevariablespermissiontype)_ | Variable access.<br />Must be one of the following values: `none`, `read`, `write`.<br />Default: `none`. |


#### DeletionPolicy

_Underlying type:_ _string_

DeletionPolicy defines the strategy the Kubernetes operator uses when you delete a resource, either manually or by a system event.


You must use one of the following values:
- `retain`: When you delete the custom resource, the operator does not delete the workspace.
- `soft`: Attempts to delete the associated workspace only if it does not contain any managed resources.
- `destroy`: Executes a destroy operation to remove all resources managed by the associated workspace. Once the destruction of these resources is successful, the operator deletes the workspace, and then deletes the custom resource.
- `force`: Forcefully and immediately deletes the workspace and the custom resource.

_Appears in:_
- [WorkspaceSpec](#workspacespec)



#### Module



Module implements API-driven Run Workflows.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/run/api



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `app.terraform.io/v1alpha2`
| `kind` _string_ | `Module`
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ModuleSpec](#modulespec)_ |  |


#### ModuleDeletionPolicy

_Underlying type:_ _string_

Deletion Policy defines the strategies for resource deletion in the Kubernetes operator.
It controls how the operator should handle the deletion of resources when triggered by
a user action or system event.


There is one possible value:
- `retain`: When the custom resource is deleted, the associated module is retained. `destroyOnDeletion` must be set to false. Default value.
- `destroy`: Executes a destroy operation. Removes all resources and the module.

_Appears in:_
- [ModuleSpec](#modulespec)



#### ModuleOutput



Module outputs to store in ConfigMap(non-sensitive) or Secret(sensitive).

_Appears in:_
- [ModuleSpec](#modulespec)

| Field | Description |
| --- | --- |
| `name` _string_ | Output name must match with the module output. |
| `sensitive` _boolean_ | Specify whether or not the output is sensitive.<br />Default: `false`. |


#### ModuleSource



Module source and version to execute.

_Appears in:_
- [ModuleSpec](#modulespec)

| Field | Description |
| --- | --- |
| `source` _string_ | Non local Terraform module source.<br />More information:<br />  - https://developer.hashicorp.com/terraform/language/modules/sources |
| `version` _string_ | Terraform module version. |


#### ModuleSpec



ModuleSpec defines the desired state of Module.

_Appears in:_
- [Module](#module)

| Field | Description |
| --- | --- |
| `organization` _string_ | Organization name where the Workspace will be created.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations |
| `token` _[Token](#token)_ | API Token to be used for API calls. |
| `module` _[ModuleSource](#modulesource)_ | Module source and version to execute. |
| `workspace` _[ModuleWorkspace](#moduleworkspace)_ | Workspace to execute the module. |
| `name` _string_ | Name of the module that will be uploaded and executed.<br />Default: `this`. |
| `variables` _[ModuleVariable](#modulevariable) array_ | Variables to pass to the module, they must exist in the Workspace. |
| `outputs` _[ModuleOutput](#moduleoutput) array_ | Module outputs to store in ConfigMap(non-sensitive) or Secret(sensitive). |
| `destroyOnDeletion` _boolean_ | DEPRECATED: Specify whether or not to execute a Destroy run when the object is deleted from the Kubernetes.<br />Default: `false`. |
| `restartedAt` _string_ | Allows executing a new Run without changing any Workspace or Module attributes.<br />Example: kubectl patch <KIND> <NAME> --type=merge --patch '\{"spec": \{"restartedAt": "'\`date -u -Iseconds\`'"\}\}' |
| `deletionPolicy` _[ModuleDeletionPolicy](#moduledeletionpolicy)_ | Deletion Policy defines the strategies for resource deletion in the Kubernetes operator.<br />It controls how the operator should handle the deletion of resources when triggered by<br />a user action or system event.<br /><br />There is one possible value:<br />- `retain`: When the custom resource is deleted, the associated module is retained. `destroyOnDeletion` must be set to false.<br />- `destroy`: Executes a destroy operation. Removes all resources and the module.<br />Default: `retain`. |




#### ModuleVariable



Variables to pass to the module.

_Appears in:_
- [ModuleSpec](#modulespec)

| Field | Description |
| --- | --- |
| `name` _string_ | Variable name must exist in the Workspace. |


#### ModuleWorkspace



Workspace to execute the module.
Only one of the fields `ID` or `Name` is allowed.
At least one of the fields `ID` or `Name` is mandatory.

_Appears in:_
- [ModuleSpec](#modulespec)

| Field | Description |
| --- | --- |
| `id` _string_ | Module Workspace ID.<br />Must match pattern: `^ws-[a-zA-Z0-9]+$` |
| `name` _string_ | Module Workspace Name. |


#### Notification



Notifications allow you to send messages to other applications based on run and workspace events.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/notifications

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `name` _string_ | Notification name. |
| `type` _[NotificationDestinationType](#notificationdestinationtype)_ | The type of the notification.<br />Must be one of the following values: `email`, `generic`, `microsoft-teams`, `slack`. |
| `enabled` _boolean_ | Whether the notification configuration should be enabled or not.<br />Default: `true`. |
| `token` _string_ | The token of the notification. |
| `triggers` _[NotificationTrigger](#notificationtrigger) array_ | The list of run events that will trigger notifications.<br />Trigger represents the different TFC notifications that can be sent as a run's progress transitions between different states.<br />There are two categories of triggers:<br />  - Health Events: `assessment:check_failure`, `assessment:drifted`, `assessment:failed`.<br />  - Run Events: `run:applying`, `run:completed`, `run:created`, `run:errored`, `run:needs_attention`, `run:planning`. |
| `url` _string_ | The URL of the notification.<br />Must match pattern: `^https?://.*` |
| `emailAddresses` _string array_ | The list of email addresses that will receive notification emails.<br />It is only available for Terraform Enterprise users. It is not available in HCP Terraform. |
| `emailUsers` _string array_ | The list of users belonging to the organization that will receive notification emails. |


#### NotificationTrigger

_Underlying type:_ _string_

NotificationTrigger represents the different TFC notifications that can be sent as a run's progress transitions between different states.
This must be aligned with go-tfe type `NotificationTriggerType`.
Must be one of the following values: `run:applying`, `assessment:check_failure`, `run:completed`, `run:created`, `assessment:drifted`, `run:errored`, `assessment:failed`, `run:needs_attention`, `run:planning`.

_Appears in:_
- [Notification](#notification)



#### OutputStatus



Outputs status.

_Appears in:_
- [ModuleStatus](#modulestatus)

| Field | Description |
| --- | --- |
| `runID` _string_ | Run ID of the latest run that updated the outputs. |


#### PlanStatus





_Appears in:_
- [WorkspaceStatus](#workspacestatus)

| Field | Description |
| --- | --- |
| `id` _string_ | Latest plan-only/speculative plan HCP Terraform run ID. |
| `terraformVersion` _string_ | The version of Terraform to use for this run. |


#### Project



Project manages HCP Terraform Projects.
More information:
- https://developer.hashicorp.com/terraform/cloud-docs/projects/manage



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `app.terraform.io/v1alpha2`
| `kind` _string_ | `Project`
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ProjectSpec](#projectspec)_ |  |


#### ProjectDeletionPolicy

_Underlying type:_ _string_

DeletionPolicy defines the strategy the Kubernetes operator uses when you delete a project, either manually or by a system event.


You must use one of the following values:
- `retain`: When the custom resource is deleted, the operator will not delete the associated project.
- `soft`: Attempts to remove the project. The project must be empty.

_Appears in:_
- [ProjectSpec](#projectspec)



#### ProjectSpec



ProjectSpec defines the desired state of Project.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/organize-workspaces-with-projects

_Appears in:_
- [Project](#project)

| Field | Description |
| --- | --- |
| `organization` _string_ | Organization name where the Workspace will be created.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations |
| `token` _[Token](#token)_ | API Token to be used for API calls. |
| `name` _string_ | Name of the Project. |
| `teamAccess` _[ProjectTeamAccess](#projectteamaccess) array_ | HCP Terraform's access model is team-based. In order to perform an action within a HCP Terraform organization,<br />users must belong to a team that has been granted the appropriate permissions.<br />You can assign project-specific permissions to teams.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/organize-workspaces-with-projects#permissions<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#project-permissions |
| `deletionPolicy` _[ProjectDeletionPolicy](#projectdeletionpolicy)_ | DeletionPolicy defines the strategy the Kubernetes operator uses when you delete a project, either manually or by a system event.<br /><br />You must use one of the following values:<br />- `retain`:  When the custom resource is deleted, the operator will not delete the associated project.<br />- `soft`: Attempts to remove the project. The project must be empty.<br />Default: `retain`. |




#### ProjectTeamAccess



HCP Terraform's access model is team-based. In order to perform an action within a HCP Terraform organization,
users must belong to a team that has been granted the appropriate permissions.
You can assign project-specific permissions to teams.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/organize-workspaces-with-projects#permissions
  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#project-permissions

_Appears in:_
- [ProjectSpec](#projectspec)

| Field | Description |
| --- | --- |
| `team` _[Team](#team)_ | Team to grant access.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/teams |
| `access` _[TeamProjectAccessType](#teamprojectaccesstype)_ | There are two ways to choose which permissions a given team has on a project: fixed permission sets, and custom permissions.<br />Must be one of the following values: `admin`, `custom`, `maintain`, `read`, `write`.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#project-permissions<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#general-project-permissions |
| `custom` _[CustomProjectPermissions](#customprojectpermissions)_ | Custom permissions let you assign specific, finer-grained permissions to a team than the broader fixed permission sets provide.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#custom-project-permissions |


#### RemoteStateSharing



RemoteStateSharing allows remote state access between workspaces.
By default, new workspaces in HCP Terraform do not allow other workspaces to access their state.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/state#accessing-state-from-other-workspaces

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `allWorkspaces` _boolean_ | Allow access to the state for all workspaces within the same organization.<br />Default: `false`. |
| `workspaces` _[ConsumerWorkspace](#consumerworkspace) array_ | Allow access to the state for specific workspaces within the same organization. |


#### RetryPolicy



RetryPolicy allows you to configure retry behavior for failed runs on the workspace.
It will apply for the latest current run of the operator.

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `backoffLimit` _integer_ | Limit is the maximum number of retries for failed runs. If set to a negative number, no limit will be applied.<br />Default: `0`. |


#### RetryStatus



RetryStatus contains the status of the retry of the latest run on the workspace. How many attempts are left and
possibly a time to wait for the next attempt.

_Appears in:_
- [WorkspaceStatus](#workspacestatus)

| Field | Description |
| --- | --- |
| `failed` _integer_ | Failed is the number of failed attempts, counting the initial one. |


#### RunStatus





_Appears in:_
- [ModuleStatus](#modulestatus)
- [WorkspaceStatus](#workspacestatus)

| Field | Description |
| --- | --- |
| `id` _string_ | Current(both active and finished) HCP Terraform run ID. |
| `configurationVersion` _string_ | The configuration version of this run. |
| `outputRunID` _string_ | Run ID of the latest run that could update the outputs. |


#### RunTrigger



RunTrigger allows you to connect this workspace to one or more source workspaces.
These connections allow runs to queue automatically in this workspace on successful apply of runs in any of the source workspaces.
Only one of the fields `ID` or `Name` is allowed.
At least one of the fields `ID` or `Name` is mandatory.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-triggers

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `id` _string_ | Source Workspace ID.<br />Must match pattern: `^ws-[a-zA-Z0-9]+$` |
| `name` _string_ | Source Workspace Name. |


#### SSHKey



SSH key used to clone Terraform modules.
Only one of the fields `ID` or `Name` is allowed.
At least one of the fields `ID` or `Name` is mandatory.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/ssh-keys

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `id` _string_ | SSH key ID.<br />Must match pattern: `^sshkey-[a-zA-Z0-9]+$` |
| `name` _string_ | SSH key name. |


#### Tag

_Underlying type:_ _string_

Tags allows you to correlate, organize, and even filter workspaces based on the assigned tags.
Tags must be one or more characters; can include letters, numbers, colons, hyphens, and underscores; and must begin and end with a letter or number.
Must match pattern: `^[A-Za-z0-9][A-Za-z0-9:_-]*$`

_Appears in:_
- [WorkspaceSpec](#workspacespec)



#### TargetWorkspace



TargetWorkspace is the name or ID of the workspace you want autoscale against.

_Appears in:_
- [AgentDeploymentAutoscaling](#agentdeploymentautoscaling)

| Field | Description |
| --- | --- |
| `id` _string_ | Workspace ID |
| `name` _string_ | Workspace Name |
| `wildcardName` _string_ | Wildcard Name to match match workspace names using `*` on name suffix, prefix, or both. |


#### Team



Teams are groups of HCP Terraform users within an organization.
If a user belongs to at least one team in an organization, they are considered a member of that organization.
Only one of the fields `ID` or `Name` is allowed.
At least one of the fields `ID` or `Name` is mandatory.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/teams

_Appears in:_
- [ProjectTeamAccess](#projectteamaccess)
- [TeamAccess](#teamaccess)

| Field | Description |
| --- | --- |
| `id` _string_ | Team ID.<br />Must match pattern: `^team-[a-zA-Z0-9]+$` |
| `name` _string_ | Team name. |


#### TeamAccess



HCP Terraform workspaces can only be accessed by users with the correct permissions.
You can manage permissions for a workspace on a per-team basis.
When a workspace is created, only the owners team and teams with the "manage workspaces" permission can access it,
with full admin permissions. These teams' access can't be removed from a workspace.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/access

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `team` _[Team](#team)_ | Team to grant access.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/teams |
| `access` _string_ | There are two ways to choose which permissions a given team has on a workspace: fixed permission sets, and custom permissions.<br />Must be one of the following values: `admin`, `custom`, `plan`, `read`, `write`.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#workspace-permissions |
| `custom` _[CustomPermissions](#custompermissions)_ | Custom permissions let you assign specific, finer-grained permissions to a team than the broader fixed permission sets provide.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#custom-workspace-permissions |


#### Token



Token refers to a Kubernetes Secret object within the same namespace as the Workspace object

_Appears in:_
- [AgentPoolSpec](#agentpoolspec)
- [ModuleSpec](#modulespec)
- [ProjectSpec](#projectspec)
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `secretKeyRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#secretkeyselector-v1-core)_ | Selects a key of a secret in the workspace's namespace |


#### ValueFrom



ValueFrom source for the variable's value.
Cannot be used if value is not empty.

_Appears in:_
- [Variable](#variable)

| Field | Description |
| --- | --- |
| `configMapKeyRef` _[ConfigMapKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#configmapkeyselector-v1-core)_ | Selects a key of a ConfigMap. |
| `secretKeyRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#secretkeyselector-v1-core)_ | Selects a key of a Secret. |


#### Variable



Variables let you customize configurations, modify Terraform's behavior, and store information like provider credentials.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name of the variable. |
| `description` _string_ | Description of the variable. |
| `hcl` _boolean_ | Parse this field as HashiCorp Configuration Language (HCL). This allows you to interpolate values at runtime.<br />Default: `false`. |
| `sensitive` _boolean_ | Sensitive variables are never shown in the UI or API.<br />They may appear in Terraform logs if your configuration is designed to output them.<br />Default: `false`. |
| `value` _string_ | Value of the variable. |
| `valueFrom` _[ValueFrom](#valuefrom)_ | Source for the variable's value. Cannot be used if value is not empty. |


#### VariableSetStatus





_Appears in:_
- [WorkspaceStatus](#workspacestatus)

| Field | Description |
| --- | --- |
| `id` _string_ |  |
| `name` _string_ |  |


#### VariableStatus





_Appears in:_
- [WorkspaceStatus](#workspacestatus)

| Field | Description |
| --- | --- |
| `name` _string_ | Name of the variable. |
| `id` _string_ | ID of the variable. |
| `versionID` _string_ | VersionID is a hash of the variable on the TFC end. |
| `valueID` _string_ | ValueID is a hash of the variable on the CRD end. |
| `category` _string_ | Category of the variable. |


#### VersionControl



VersionControl settings for the workspace's VCS repository, enabling the UI/VCS-driven run workflow.
Omit this argument to utilize the CLI-driven and API-driven workflows, where runs are not driven by webhooks on your VCS provider.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/run/ui
  - https://developer.hashicorp.com/terraform/cloud-docs/vcs

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `oAuthTokenID` _string_ | The VCS Connection (OAuth Connection + Token) to use.<br />Must match pattern: `^ot-[a-zA-Z0-9]+$` |
| `repository` _string_ | A reference to your VCS repository in the format `<organization>/<repository>` where `<organization>` and `<repository>` refer to the organization and repository in your VCS provider. |
| `branch` _string_ | The repository branch that Run will execute from. This defaults to the repository's default branch (e.g. main). |
| `speculativePlans` _boolean_ | Whether this workspace allows automatic speculative plans on PR.<br />Default: `true`.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/run/ui#speculative-plans-on-pull-requests<br />  - https://developer.hashicorp.com/terraform/cloud-docs/run/remote-operations#speculative-plans |
| `enableFileTriggers` _boolean_ | File triggers allow you to queue runs in HCP Terraform when files in your VCS repository change.<br />Default: `false`.<br />More informarion:<br /> - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/vcs#automatic-run-triggering |
| `triggerPatterns` _string array_ | The list of pattern triggers that will queue runs in HCP Terraform when files in your VCS repository change.<br />`spec.versionControl.fileTriggersEnabled` must be set to `true`.<br />More informarion:<br /> - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/vcs#automatic-run-triggering |
| `triggerPrefixes` _string array_ | The list of pattern prefixes that will queue runs in HCP Terraform when files in your VCS repository change.<br />`spec.versionControl.fileTriggersEnabled` must be set to `true`.<br />More informarion:<br /> - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/vcs#automatic-run-triggering |


#### Workspace



Workspace manages HCP Terraform Workspaces.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `app.terraform.io/v1alpha2`
| `kind` _string_ | `Workspace`
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[WorkspaceSpec](#workspacespec)_ |  |


#### WorkspaceAgentPool



AgentPool allows HCP Terraform to communicate with isolated, private, or on-premises infrastructure.
Only one of the fields `ID` or `Name` is allowed.
At least one of the fields `ID` or `Name` is mandatory.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/agents

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `id` _string_ | Agent Pool ID.<br />Must match pattern: `^apool-[a-zA-Z0-9]+$` |
| `name` _string_ | Agent Pool name. |


#### WorkspaceProject



Projects let you organize your workspaces into groups.
Only one of the fields `ID` or `Name` is allowed.
At least one of the fields `ID` or `Name` is mandatory.
More information:
  - https://developer.hashicorp.com/terraform/tutorials/cloud/projects

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `id` _string_ | Project ID.<br />Must match pattern: `^prj-[a-zA-Z0-9]+$` |
| `name` _string_ | Project name. |


#### WorkspaceRunTask



Run tasks allow HCP Terraform to interact with external systems at specific points in the HCP Terraform run lifecycle.
Only one of the fields `ID` or `Name` is allowed.
At least one of the fields `ID` or `Name` is mandatory.
More information:
  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-tasks

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `id` _string_ | Run Task ID.<br />Must match pattern: `^task-[a-zA-Z0-9]+$` |
| `name` _string_ | Run Task Name. |
| `enforcementLevel` _string_ | Run Task Enforcement Level. Can be one of `advisory` or `mandatory`. Default: `advisory`.<br />Must be one of the following values: `advisory`, `mandatory`<br />Default: `advisory`. |
| `stage` _string_ | Run Task Stage.<br />Must be one of the following values: `pre_apply`, `pre_plan`, `post_plan`.<br />Default: `post_plan`. |


#### WorkspaceSpec



WorkspaceSpec defines the desired state of Workspace.

_Appears in:_
- [Workspace](#workspace)

| Field | Description |
| --- | --- |
| `name` _string_ | Workspace name. |
| `organization` _string_ | Organization name where the Workspace will be created.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations |
| `token` _[Token](#token)_ | API Token to be used for API calls. |
| `applyMethod` _string_ | Define either change will be applied automatically(auto) or require an operator to confirm(manual).<br />Must be one of the following values: `auto`, `manual`.<br />Default: `manual`.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#auto-apply-and-manual-apply |
| `applyRunTrigger` _string_ | Specifies the type of apply, whether manual or auto<br />Must be of value `auto` or `manual`<br />Default: `manual`<br />More information:<br />- https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#auto-apply |
| `allowDestroyPlan` _boolean_ | Allows a destroy plan to be created and applied.<br />Default: `true`.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#destruction-and-deletion |
| `description` _string_ | Workspace description. |
| `agentPool` _[WorkspaceAgentPool](#workspaceagentpool)_ | HCP Terraform Agents allow HCP Terraform to communicate with isolated, private, or on-premises infrastructure.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/agents |
| `executionMode` _string_ | Define where the Terraform code will be executed.<br />Must be one of the following values: `agent`, `local`, `remote`.<br />Default: `remote`.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#execution-mode |
| `runTasks` _[WorkspaceRunTask](#workspaceruntask) array_ | Run tasks allow HCP Terraform to interact with external systems at specific points in the HCP Terraform run lifecycle.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-tasks |
| `tags` _[Tag](#tag) array_ | Workspace tags are used to help identify and group together workspaces.<br />Tags must be one or more characters; can include letters, numbers, colons, hyphens, and underscores; and must begin and end with a letter or number. |
| `teamAccess` _[TeamAccess](#teamaccess) array_ | HCP Terraform workspaces can only be accessed by users with the correct permissions.<br />You can manage permissions for a workspace on a per-team basis.<br />When a workspace is created, only the owners team and teams with the "manage workspaces" permission can access it,<br />with full admin permissions. These teams' access can't be removed from a workspace.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/access |
| `terraformVersion` _string_ | The version of Terraform to use for this workspace.<br />If not specified, the latest available version will be used.<br />Must match pattern: `^\\d\{1\}\\.\\d\{1,2\}\\.\\d\{1,2\}$`<br />More information:<br />  - https://www.terraform.io/cloud-docs/workspaces/settings#terraform-version |
| `workingDirectory` _string_ | The directory where Terraform will execute, specified as a relative path from the root of the configuration directory.<br />More information:<br />  - https://www.terraform.io/cloud-docs/workspaces/settings#terraform-working-directory |
| `environmentVariables` _[Variable](#variable) array_ | Terraform Environment variables for all plans and applies in this workspace.<br />Variables defined within a workspace always overwrite variables from variable sets that have the same type and the same key.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables#environment-variables |
| `terraformVariables` _[Variable](#variable) array_ | Terraform variables for all plans and applies in this workspace.<br />Variables defined within a workspace always overwrite variables from variable sets that have the same type and the same key.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables#terraform-variables |
| `remoteStateSharing` _[RemoteStateSharing](#remotestatesharing)_ | Remote state access between workspaces.<br />By default, new workspaces in HCP Terraform do not allow other workspaces to access their state.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/state#accessing-state-from-other-workspaces |
| `retryPolicy` _[RetryPolicy](#retrypolicy)_ | Retry Policy allows you to specify how the operator should retry failed runs automatically. |
| `runTriggers` _[RunTrigger](#runtrigger) array_ | Run triggers allow you to connect this workspace to one or more source workspaces.<br />These connections allow runs to queue automatically in this workspace on successful apply of runs in any of the source workspaces.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-triggers |
| `versionControl` _[VersionControl](#versioncontrol)_ | Settings for the workspace's VCS repository, enabling the UI/VCS-driven run workflow.<br />Omit this argument to utilize the CLI-driven and API-driven workflows, where runs are not driven by webhooks on your VCS provider.<br />More information:<br />  - https://www.terraform.io/cloud-docs/run/ui<br />  - https://www.terraform.io/cloud-docs/vcs |
| `sshKey` _[SSHKey](#sshkey)_ | SSH key used to clone Terraform modules.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/ssh-keys |
| `notifications` _[Notification](#notification) array_ | Notifications allow you to send messages to other applications based on run and workspace events.<br />More information:<br />  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/notifications |
| `project` _[WorkspaceProject](#workspaceproject)_ | Projects let you organize your workspaces into groups.<br />Default: default organization project.<br />More information:<br />  - https://developer.hashicorp.com/terraform/tutorials/cloud/projects |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | The Deletion Policy specifies the behavior of the custom resource and its associated workspace when the custom resource is deleted.<br />- `retain`: When you delete the custom resource, the operator does not delete the workspace.<br />- `soft`: Attempts to delete the associated workspace only if it does not contain any managed resources.<br />- `destroy`: Executes a destroy operation to remove all resources managed by the associated workspace. Once the destruction of these resources is successful, the operator deletes the workspace, and then deletes the custom resource.<br />- `force`: Forcefully and immediately deletes the workspace and the custom resource.<br />Default: `retain`. |
| `variableSets` _[WorkspaceVariableSet](#workspacevariableset) array_ | HCP Terraform variable sets let you reuse variables in an efficient and centralized way.<br />More information<br />  - https://developer.hashicorp.com/terraform/tutorials/cloud/cloud-multiple-variable-sets |




#### WorkspaceVariableSet





_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `id` _string_ | ID of the variable set.<br />Must match pattern: `varset-[a-zA-Z0-9]+$`<br />More information:<br />  - https://developer.hashicorp.com/terraform/tutorials/cloud/cloud-multiple-variable-sets |
| `name` _string_ | Name of the variable set.<br />More information:<br />  - https://developer.hashicorp.com/terraform/tutorials/cloud/cloud-multiple-variable-sets |


