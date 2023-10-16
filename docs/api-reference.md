# API Reference

## Packages
- [app.terraform.io/v1alpha2](#appterraformiov1alpha2)


## app.terraform.io/v1alpha2

Package v1alpha2 contains API Schema definitions for the app v1alpha2 API group

### Resource Types
- [AgentPool](#agentpool)
- [Module](#module)
- [Workspace](#workspace)



#### AgentDeployment





_Appears in:_
- [AgentPoolSpec](#agentpoolspec)

| Field | Description |
| --- | --- |
| `replicas` _integer_ |  |
| `spec` _[PodSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#podspec-v1-core)_ |  |


#### AgentDeploymentAutoscaling



AgentDeploymentAutoscaling allows you to configure the operator to scale the deployment for an AgentPool up and down to meet demand.

_Appears in:_
- [AgentPoolSpec](#agentpoolspec)

| Field | Description |
| --- | --- |
| `maxReplicas` _integer_ | MaxReplicas is the maximum number of replicas for the Agent deployment. |
| `minReplicas` _integer_ | MinReplicas is the minimum number of replicas for the Agent deployment. |
| `targetWorkspaces` _[TargetWorkspace](#targetworkspace)_ | TargetWorkspaces is a list of Terraform Cloud Workspaces which the agent pool should scale up to meet demand. When this field is ommited the autoscaler will target all workspaces that are associated with the AgentPool. |
| `cooldownPeriodSeconds` _integer_ | CooldownPeriodSeconds is the time to wait between scaling events. Defaults to 300. |


#### AgentDeploymentAutoscalingStatus



AgentDeploymentAutoscalingStatus

_Appears in:_
- [AgentPoolStatus](#agentpoolstatus)

| Field | Description |
| --- | --- |
| `desiredReplicas` _integer_ | Desired number of agent replicas |
| `lastScalingEvent` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | Last time the agent pool was scaledx |


#### AgentPool



AgentPool is the Schema for the agentpools API.



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `app.terraform.io/v1alpha2`
| `kind` _string_ | `AgentPool`
| `TypeMeta` _[TypeMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#typemeta-v1-meta)_ |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[AgentPoolSpec](#agentpoolspec)_ |  |


#### AgentPoolSpec



AgentPoolSpec defines the desired stak get ste of AgentPool.

_Appears in:_
- [AgentPool](#agentpool)

| Field | Description |
| --- | --- |
| `name` _string_ | Agent Pool name. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/agents/agent-pools |
| `organization` _string_ | Organization name where the Workspace will be created. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations |
| `token` _[Token](#token)_ | API Token to be used for API calls. |
| `agentTokens` _[AgentToken](#agenttoken) array_ | List of the agent tokens to generate. |
| `agentDeployment` _[AgentDeployment](#agentdeployment)_ | Agent deployment settings |
| `autoscaling` _[AgentDeploymentAutoscaling](#agentdeploymentautoscaling)_ | Agent deployment settings |




#### AgentToken



Agent Token is a secret token that a Terraform Cloud Agent is used to connect to the Terraform Cloud Agent Pool. In `spec` only the field `Name` is allowed, the rest are used in `status`. More infromation:   - https://developer.hashicorp.com/terraform/cloud-docs/agents

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



A configuration version is a resource used to reference the uploaded configuration files. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/api-docs/configuration-versions   - https://developer.hashicorp.com/terraform/cloud-docs/run/api

_Appears in:_
- [ModuleStatus](#modulestatus)

| Field | Description |
| --- | --- |
| `id` _string_ | Configuration Version ID. |


#### ConsumerWorkspace



ConsumerWorkspace allows access to the state for specific workspaces within the same organization. Only one of the fields `ID` or `Name` is allowed. At least one of the fields `ID` or `Name` is mandatory. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/state#remote-state-access-controls

_Appears in:_
- [RemoteStateSharing](#remotestatesharing)

| Field | Description |
| --- | --- |
| `id` _string_ | Consumer Workspace ID. Must match pattern: ^ws-[a-zA-Z0-9]+$ |
| `name` _string_ | Consumer Workspace name. |


#### CustomPermissions



Custom permissions let you assign specific, finer-grained permissions to a team than the broader fixed permission sets provide. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#custom-workspace-permissions

_Appears in:_
- [TeamAccess](#teamaccess)

| Field | Description |
| --- | --- |
| `runs` _string_ | Run access. Must be one of the following values: `apply`, `plan`, `read`. Default: `read`. |
| `runTasks` _boolean_ | Manage Workspace Run Tasks. Default: `false`. |
| `sentinel` _string_ | Download Sentinel mocks. Must be one of the following values: `none`, `read`. Default: `none`. |
| `stateVersions` _string_ | State access. Must be one of the following values: `none`, `read`, `read-outputs`, `write`. Default: `none`. |
| `variables` _string_ | Variable access. Must be one of the following values: `none`, `read`, `write`. Default: `none`. |
| `workspaceLocking` _boolean_ | Lock/unlock workspace. Default: `false`. |


#### Module



Module is the Schema for the modules API Module implements the API-driven Run Workflow More information:   - https://developer.hashicorp.com/terraform/cloud-docs/run/api



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `app.terraform.io/v1alpha2`
| `kind` _string_ | `Module`
| `TypeMeta` _[TypeMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#typemeta-v1-meta)_ |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ModuleSpec](#modulespec)_ |  |


#### ModuleOutput



Module outputs to store in ConfigMap(non-sensitive) or Secret(sensitive).

_Appears in:_
- [ModuleSpec](#modulespec)

| Field | Description |
| --- | --- |
| `name` _string_ | Output name must match with the module output. |
| `sensitive` _boolean_ | Specify whether or not the output is sensitive. Default: `false`. |


#### ModuleSource



Module source and version to execute.

_Appears in:_
- [ModuleSpec](#modulespec)

| Field | Description |
| --- | --- |
| `source` _string_ | Non local Terraform module source. More information:   - https://developer.hashicorp.com/terraform/language/modules/sources |
| `version` _string_ | Terraform module version. |


#### ModuleSpec



ModuleSpec defines the desired state of Module.

_Appears in:_
- [Module](#module)

| Field | Description |
| --- | --- |
| `organization` _string_ | Organization name where the Workspace will be created. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations |
| `token` _[Token](#token)_ | API Token to be used for API calls. |
| `module` _[ModuleSource](#modulesource)_ | Module source and version to execute. |
| `workspace` _[ModuleWorkspace](#moduleworkspace)_ | Workspace to execute the module. |
| `name` _string_ | Name of the module that will be uploaded and executed. Default: `this`. |
| `variables` _[ModuleVariable](#modulevariable) array_ | Variables to pass to the module, they must exist in the Workspace. |
| `outputs` _[ModuleOutput](#moduleoutput) array_ | Module outputs to store in ConfigMap(non-sensitive) or Secret(sensitive). |
| `destroyOnDeletion` _boolean_ | Specify whether or not to execute a Destroy run when the object is deleted from the Kubernetes. Default: `false`. |
| `restartedAt` _string_ | Allows executing a new Run without changing any Workspace or Module attributes. Example: kubectl patch <KIND> <NAME> --type=merge --patch '{"spec": {"restartedAt": "'\`date -u -Iseconds\`'"}}' |




#### ModuleVariable



Variables to pass to the module.

_Appears in:_
- [ModuleSpec](#modulespec)

| Field | Description |
| --- | --- |
| `name` _string_ | Variable name must exist in the Workspace. |


#### ModuleWorkspace



Workspace to execute the module. Only one of the fields `ID` or `Name` is allowed. At least one of the fields `ID` or `Name` is mandatory.

_Appears in:_
- [ModuleSpec](#modulespec)

| Field | Description |
| --- | --- |
| `id` _string_ | Module Workspace ID. Must match pattern: ^ws-[a-zA-Z0-9]+$ |
| `name` _string_ | Module Workspace Name. |


#### Notification



Notifications allow you to send messages to other applications based on run and workspace events. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/notifications

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `name` _string_ | Notification name. |
| `type` _NotificationDestinationType_ | The type of the notification. Must be one of the following values: `email`, `generic`, `microsoft-teams`, `slack`. |
| `enabled` _boolean_ | Whether the notification configuration should be enabled or not. Default: `true`. |
| `token` _string_ | The token of the notification. |
| `triggers` _NotificationTrigger array_ | The list of run events that will trigger notifications. Trigger represents the different TFC notifications that can be sent as a run's progress transitions between different states. There are two categories of triggers:   - Health Events: `assessment:check_failure`, `assessment:drifted`, `assessment:failed`.   - Run Events: `run:applying`, `run:completed`, `run:created`, `run:errored`, `run:needs_attention`, `run:planning`. |
| `url` _string_ | The URL of the notification. Must match pattern: ^https?://.* |
| `emailAddresses` _string array_ | The list of email addresses that will receive notification emails. It is only available for Terraform Enterprise users. It is not available in Terraform Cloud. |
| `emailUsers` _string array_ | The list of users belonging to the organization that will receive notification emails. |


#### OutputStatus



Module Outputs status.

_Appears in:_
- [ModuleStatus](#modulestatus)

| Field | Description |
| --- | --- |
| `runID` _string_ | Run ID of the latest run that updated the outputs. |


#### RemoteStateSharing



RemoteStateSharing allows remote state access between workspaces. By default, new workspaces in Terraform Cloud do not allow other workspaces to access their state. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/state#accessing-state-from-other-workspaces

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `allWorkspaces` _boolean_ | Allow access to the state for all workspaces within the same organization. Default: `false`. |
| `workspaces` _[ConsumerWorkspace](#consumerworkspace) array_ | Allow access to the state for specific workspaces within the same organization. |


#### RunStatus





_Appears in:_
- [ModuleStatus](#modulestatus)
- [WorkspaceStatus](#workspacestatus)

| Field | Description |
| --- | --- |
| `id` _string_ | Current(both active and finished) Terraform Cloud run ID. |
| `configurationVersion` _string_ |  |
| `outputRunID` _string_ | Run ID of the latest run that could update the outputs. |


#### RunTrigger



RunTrigger allows you to connect this workspace to one or more source workspaces. These connections allow runs to queue automatically in this workspace on successful apply of runs in any of the source workspaces. Only one of the fields `ID` or `Name` is allowed. At least one of the fields `ID` or `Name` is mandatory. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-triggers

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `id` _string_ | Source Workspace ID. Must match pattern: ^ws-[a-zA-Z0-9]+$ |
| `name` _string_ | Source Workspace Name. |


#### SSHKey



SSH key used to clone Terraform modules. Only one of the fields `ID` or `Name` is allowed. At least one of the fields `ID` or `Name` is mandatory. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/ssh-keys

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `id` _string_ | SSH key ID. Must match pattern: ^sshkey-[a-zA-Z0-9]+$ |
| `name` _string_ | SSH key name. |


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



Teams are groups of Terraform Cloud users within an organization. If a user belongs to at least one team in an organization, they are considered a member of that organization. Only one of the fields `ID` or `Name` is allowed. At least one of the fields `ID` or `Name` is mandatory. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/teams

_Appears in:_
- [TeamAccess](#teamaccess)

| Field | Description |
| --- | --- |
| `id` _string_ | Team ID. Must match pattern: ^team-[a-zA-Z0-9]+$ |
| `name` _string_ | Team name. |


#### TeamAccess



Terraform Cloud workspaces can only be accessed by users with the correct permissions. You can manage permissions for a workspace on a per-team basis. When a workspace is created, only the owners team and teams with the "manage workspaces" permission can access it, with full admin permissions. These teams' access can't be removed from a workspace. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/access

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `team` _[Team](#team)_ | Team to grant access. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/teams |
| `access` _string_ | There are two ways to choose which permissions a given team has on a workspace: fixed permission sets, and custom permissions. Must be one of the following values: `admin`, `custom`, `plan`, `read`, `write`. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#workspace-permissions |
| `custom` _[CustomPermissions](#custompermissions)_ | Custom permissions let you assign specific, finer-grained permissions to a team than the broader fixed permission sets provide. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#custom-workspace-permissions |


#### Token



Token refers to a Kubernetes Secret object within the same namespace as the Workspace object

_Appears in:_
- [AgentPoolSpec](#agentpoolspec)
- [ModuleSpec](#modulespec)
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `secretKeyRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#secretkeyselector-v1-core)_ | Selects a key of a secret in the workspace's namespace |


#### ValueFrom



ValueFrom source for the variable's value. Cannot be used if value is not empty.

_Appears in:_
- [Variable](#variable)

| Field | Description |
| --- | --- |
| `configMapKeyRef` _[ConfigMapKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#configmapkeyselector-v1-core)_ | Selects a key of a ConfigMap. |
| `secretKeyRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#secretkeyselector-v1-core)_ | Selects a key of a Secret. |


#### Variable



Variables let you customize configurations, modify Terraform's behavior, and store information like provider credentials. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name of the variable. |
| `description` _string_ | Description of the variable. |
| `hcl` _boolean_ | Parse this field as HashiCorp Configuration Language (HCL). This allows you to interpolate values at runtime. Default: `false`. |
| `sensitive` _boolean_ | Sensitive variables are never shown in the UI or API. They may appear in Terraform logs if your configuration is designed to output them. Default: `false`. |
| `value` _string_ | Value of the variable. |
| `valueFrom` _[ValueFrom](#valuefrom)_ | Source for the variable's value. Cannot be used if value is not empty. |


#### VersionControl



VersionControl settings for the workspace's VCS repository, enabling the UI/VCS-driven run workflow. Omit this argument to utilize the CLI-driven and API-driven workflows, where runs are not driven by webhooks on your VCS provider. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/run/ui   - https://developer.hashicorp.com/terraform/cloud-docs/vcs

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `oAuthTokenID` _string_ | The VCS Connection (OAuth Connection + Token) to use. Must match pattern: ^ot-[a-zA-Z0-9]+$ |
| `repository` _string_ | A reference to your VCS repository in the format <organization>/<repository> where <organization> and <repository> refer to the organization and repository in your VCS provider. |
| `branch` _string_ | The repository branch that Run will execute from. This defaults to the repository's default branch (e.g. main). |


#### Workspace



Workspace is the Schema for the workspaces API



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `app.terraform.io/v1alpha2`
| `kind` _string_ | `Workspace`
| `TypeMeta` _[TypeMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#typemeta-v1-meta)_ |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[WorkspaceSpec](#workspacespec)_ |  |


#### WorkspaceAgentPool



AgentPool allows Terraform Cloud to communicate with isolated, private, or on-premises infrastructure. Only one of the fields `ID` or `Name` is allowed. At least one of the fields `ID` or `Name` is mandatory. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/agents

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `id` _string_ | Agent Pool ID. Must match pattern: ^apool-[a-zA-Z0-9]+$ |
| `name` _string_ | Agent Pool name. |


#### WorkspaceRunTask



Run tasks allow Terraform Cloud to interact with external systems at specific points in the Terraform Cloud run lifecycle. Only one of the fields `ID` or `Name` is allowed. At least one of the fields `ID` or `Name` is mandatory. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-tasks

_Appears in:_
- [WorkspaceSpec](#workspacespec)

| Field | Description |
| --- | --- |
| `id` _string_ | Run Task ID. Must match pattern: ^task-[a-zA-Z0-9]+$ |
| `name` _string_ | Run Task Name. |
| `enforcementLevel` _string_ | Run Task Enforcement Level. Can be one of `advisory` or `mandatory`. Default: `advisory`. Must be one of the following values: `advisory`, `mandatory` Default: `advisory`. |
| `stage` _string_ | Run Task Stage. Must be one of the following values: `pre_apply`, `pre_plan`, `post_plan`. Default: `post_plan`. |


#### WorkspaceSpec



WorkspaceSpec defines the desired state of Workspace.

_Appears in:_
- [Workspace](#workspace)

| Field | Description |
| --- | --- |
| `name` _string_ | Workspace name. |
| `organization` _string_ | Organization name where the Workspace will be created. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations |
| `token` _[Token](#token)_ | API Token to be used for API calls. |
| `applyMethod` _string_ | Define either change will be applied automatically(auto) or require an operator to confirm(manual). Must be one of the following values: `auto`, `manual`. Default: `manual`. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#auto-apply-and-manual-apply |
| `allowDestroyPlan` _boolean_ | Allows a destroy plan to be created and applied. Default: `true`. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#destruction-and-deletion |
| `description` _string_ | Workspace description. |
| `agentPool` _[WorkspaceAgentPool](#workspaceagentpool)_ | Terraform Cloud Agents allow Terraform Cloud to communicate with isolated, private, or on-premises infrastructure. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/agents |
| `executionMode` _string_ | Define where the Terraform code will be executed. Must be one of the following values: `agent`, `local`, `remote`. Default: `remote`. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#execution-mode |
| `runTasks` _[WorkspaceRunTask](#workspaceruntask) array_ | Run tasks allow Terraform Cloud to interact with external systems at specific points in the Terraform Cloud run lifecycle. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-tasks |
| `tags` _Tag array_ | Workspace tags are used to help identify and group together workspaces. Tags must be one or more characters; can include letters, numbers, colons, hyphens, and underscores; and must begin and end with a letter or number. |
| `teamAccess` _[TeamAccess](#teamaccess) array_ | Terraform Cloud workspaces can only be accessed by users with the correct permissions. You can manage permissions for a workspace on a per-team basis. When a workspace is created, only the owners team and teams with the "manage workspaces" permission can access it, with full admin permissions. These teams' access can't be removed from a workspace. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/access |
| `terraformVersion` _string_ | The version of Terraform to use for this workspace. If not specified, the latest available version will be used. Must match pattern: ^\\d{1}\\.\\d{1,2}\\.\\d{1,2}$ More information:   - https://www.terraform.io/cloud-docs/workspaces/settings#terraform-version |
| `workingDirectory` _string_ | The directory where Terraform will execute, specified as a relative path from the root of the configuration directory. More information:   - https://www.terraform.io/cloud-docs/workspaces/settings#terraform-working-directory |
| `environmentVariables` _[Variable](#variable) array_ | Terraform Environment variables for all plans and applies in this workspace. Variables defined within a workspace always overwrite variables from variable sets that have the same type and the same key. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables#environment-variables |
| `terraformVariables` _[Variable](#variable) array_ | Terraform variables for all plans and applies in this workspace. Variables defined within a workspace always overwrite variables from variable sets that have the same type and the same key. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables#terraform-variables |
| `remoteStateSharing` _[RemoteStateSharing](#remotestatesharing)_ | Remote state access between workspaces. By default, new workspaces in Terraform Cloud do not allow other workspaces to access their state. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/state#accessing-state-from-other-workspaces |
| `runTriggers` _[RunTrigger](#runtrigger) array_ | Run triggers allow you to connect this workspace to one or more source workspaces. These connections allow runs to queue automatically in this workspace on successful apply of runs in any of the source workspaces. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-triggers |
| `versionControl` _[VersionControl](#versioncontrol)_ | Settings for the workspace's VCS repository, enabling the UI/VCS-driven run workflow. Omit this argument to utilize the CLI-driven and API-driven workflows, where runs are not driven by webhooks on your VCS provider. More information:   - https://www.terraform.io/cloud-docs/run/ui   - https://www.terraform.io/cloud-docs/vcs |
| `sshKey` _[SSHKey](#sshkey)_ | SSH key used to clone Terraform modules. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/ssh-keys |
| `notifications` _[Notification](#notification) array_ | Notifications allow you to send messages to other applications based on run and workspace events. More information:   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/notifications |




