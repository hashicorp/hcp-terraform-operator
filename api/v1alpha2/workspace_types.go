// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentPool allows Terraform Cloud to communicate with isolated, private, or on-premises infrastructure.
// Only one of the fields `ID` or `Name` is allowed.
//
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/agents
type WorkspaceAgentPool struct {
	// Agent Pool ID.
	//
	//+kubebuilder:validation:Pattern="^apool-[a-zA-Z0-9]+$"
	//+optional
	ID string `json:"id,omitempty"`
	// Agent Pool name.
	//
	//+kubebuilder:validation:MinLength=1
	//+optional
	Name string `json:"name,omitempty"`
}

// ConsumerWorkspace allows access to the state for specific workspaces within the same organization.
// Only one of the fields `ID` or `Name` is allowed.
//
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/state#remote-state-access-controls
type ConsumerWorkspace struct {
	// Consumer Workspace ID.
	//
	//+kubebuilder:validation:Pattern="^ws-[a-zA-Z0-9]+$"
	//+optional
	ID string `json:"id,omitempty"`
	// Consumer Workspace name.
	//
	//+kubebuilder:validation:MinLength=1
	//+optional
	Name string `json:"name,omitempty"`
}

// RemoteStateSharing allows remote state access between workspaces.
// By default, new workspaces in Terraform Cloud do not allow other workspaces to access their state.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/state#accessing-state-from-other-workspaces
type RemoteStateSharing struct {
	// Allow access to the state for all workspaces within the same organization.
	//
	//+kubebuilder:default:=false
	//+optional
	AllWorkspaces bool `json:"allWorkspaces,omitempty"`
	// Allow access to the state for specific workspaces within the same organization.
	//
	//+kubebuilder:validation:MinItems=1
	//+optional
	Workspaces []*ConsumerWorkspace `json:"workspaces,omitempty"`
}

// RunTrigger allows you to connect this workspace to one or more source workspaces.
// These connections allow runs to queue automatically in this workspace on successful apply of runs in any of the source workspaces.
// Only one of the fields `ID` or `Name` is allowed.
//
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-triggers
type RunTrigger struct {
	// Source Workspace ID.
	//
	//+kubebuilder:validation:Pattern="^ws-[a-zA-Z0-9]+$"
	//+optional
	ID string `json:"id,omitempty"`
	// Source Workspace Name.
	//
	//+kubebuilder:validation:MinLength=1
	//+optional
	Name string `json:"name,omitempty"`
}

// Teams are groups of Terraform Cloud users within an organization.
// If a user belongs to at least one team in an organization, they are considered a member of that organization.
// Only one of the fields `ID` or `Name` is allowed.
//
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/teams
type Team struct {
	// Team ID.
	//
	//+kubebuilder:validation:Pattern="^team-[a-zA-Z0-9]+$"
	//+optional
	ID string `json:"id,omitempty"`
	// Team name.
	//
	//+kubebuilder:validation:MinLength=1
	//+optional
	Name string `json:"name,omitempty"`
}

// Custom permissions let you assign specific, finer-grained permissions to a team than the broader fixed permission sets provide.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#custom-workspace-permissions
//
// +optional
type CustomPermissions struct {
	//+kubebuilder:validation:Pattern="^(apply|plan|read)$"
	//+kubebuilder:default:=read
	//+optional
	Runs string `json:"runs,omitempty"`
	//+kubebuilder:validation:default:=false
	//+optional
	RunTasks bool `json:"runTasks,omitempty"`
	//+kubebuilder:validation:Pattern="^(none|read)$"
	//+kubebuilder:default:=none
	//+optional
	Sentinel string `json:"sentinel,omitempty"`
	//+kubebuilder:validation:Pattern="^(none|read|read-outputs|write)$"
	//+kubebuilder:default:=none
	//+optional
	StateVersions string `json:"stateVersions,omitempty"`
	//+kubebuilder:validation:Pattern="^(none|read|write)$"
	//+kubebuilder:default:=none
	//+optional
	Variables string `json:"variables,omitempty"`
	//+kubebuilder:default:=false
	//+optional
	WorkspaceLocking bool `json:"workspaceLocking,omitempty"`
}

// Terraform Cloud workspaces can only be accessed by users with the correct permissions.
// You can manage permissions for a workspace on a per-team basis.
// When a workspace is created, only the owners team and teams with the "manage workspaces" permission can access it,
// with full admin permissions. These teams' access can't be removed from a workspace.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/access
type TeamAccess struct {
	// Team to grant access.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/teams
	Team Team `json:"team"`
	// There are two ways to choose which permissions a given team has on a workspace: fixed permission sets, and custom permissions.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#workspace-permissions
	//
	//+kubebuilder:validation:Pattern="^(admin|custom|plan|read|write)$"
	Access string `json:"access"`
	// Custom permissions let you assign specific, finer-grained permissions to a team than the broader fixed permission sets provide.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#custom-workspace-permissions
	//
	//+optional
	Custom CustomPermissions `json:"custom,omitempty"`
}

// Token refers to a Kubernetes Secret object within the same namespace as the Workspace object
type Token struct {
	// Selects a key of a secret in the workspace's namespace
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef"`
}

// ValueFrom source for the variable's value. Cannot be used if value is not empty.
type ValueFrom struct {
	// Selects a key of a ConfigMap.
	//+optional
	ConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	// Selects a key of a secret in the workspace's namespace
	//+optional
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// Variables let you customize configurations, modify Terraform's behavior, and store information like provider credentials.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables
type Variable struct {
	// Name of the variable.
	//
	//+kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Description of the variable.
	//
	//+optional
	Description string `json:"description,omitempty"`
	// Parse this field as HashiCorp Configuration Language (HCL). This allows you to interpolate values at runtime.
	//
	//+kubebuilder:default:=false
	//+optional
	HCL bool `json:"hcl,omitempty"`
	// Sensitive variables are never shown in the UI or API. They may appear in Terraform logs if your configuration is designed to output them.
	//
	//+kubebuilder:default:=false
	//+optional
	Sensitive bool `json:"sensitive,omitempty"`
	// Value of the variable.
	//
	//+optional
	Value string `json:"value,omitempty"`
	// Source for the variable's value. Cannot be used if value is not empty.
	//
	//+optional
	ValueFrom *ValueFrom `json:"valueFrom,omitempty"`
}

// VersionControl settings for the workspace's VCS repository, enabling the UI/VCS-driven run workflow.
// Omit this argument to utilize the CLI-driven and API-driven workflows, where runs are not driven by webhooks on your VCS provider.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/run/ui
//   - https://developer.hashicorp.com/terraform/cloud-docs/vcs
type VersionControl struct {
	// The VCS Connection (OAuth Connection + Token) to use.
	//
	//+kubebuilder:validation:Pattern="^ot-[a-zA-Z0-9]+$"
	OAuthTokenID string `json:"oAuthTokenID,omitempty"`
	// A reference to your VCS repository in the format <organization>/<repository> where <organization> and <repository> refer to the organization and repository in your VCS provider.
	Repository string `json:"repository,omitempty"`
	// The repository branch that Run will execute from. This defaults to the repository's default branch (e.g. main).
	//+kubebuilder:validation:MinLength=1
	//+optional
	Branch string `json:"branch,omitempty"`
}

// SSH key used to clone Terraform modules
// Only one of the fields `ID` or `Name` is allowed.
//
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/ssh-keys
type SSHKey struct {
	//+kubebuilder:validation:Pattern="^sshkey-[a-zA-Z0-9]+$"
	//+optional
	ID string `json:"id,omitempty"`
	//+kubebuilder:validation:MinLength=1
	//+optional
	Name string `json:"name,omitempty"`
}

// WorkspaceSpec defines the desired state of Workspace
type WorkspaceSpec struct {
	// API Token to be used for API calls
	Token Token `json:"token"`
	// Workspace name
	Name string `json:"name"`
	// Organization name where the Workspace will be created
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations
	Organization string `json:"organization"`

	// Define either change will be applied automatically(auto) or require an operator to confirm(manual).
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#auto-apply-and-manual-apply
	//
	//+kubebuilder:validation:Pattern="^(auto|manual)$"
	//+kubebuilder:default=manual
	//+optional
	ApplyMethod string `json:"applyMethod,omitempty"`
	// Allows a destroy plan to be created and applied.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#destruction-and-deletion
	//
	//+kubebuilder:default=true
	//+optional
	AllowDestroyPlan bool `json:"allowDestroyPlan,omitempty"`
	// Workspace description
	//
	//+kubebuilder:validation:MinLength=1
	//+optional
	Description string `json:"description,omitempty"`
	// Terraform Cloud Agents allow Terraform Cloud to communicate with isolated, private, or on-premises infrastructure.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/agents
	//
	//+optional
	AgentPool *WorkspaceAgentPool `json:"agentPool,omitempty"`
	// Define where the Terraform code will be executed.
	// More information:
	//  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#execution-mode
	//
	//+kubebuilder:validation:Pattern="^(agent|local|remote)$"
	//+kubebuilder:default=remote
	//+optional
	ExecutionMode string `json:"executionMode,omitempty"`
	// Workspace tags are used to help identify and group together workspaces.
	//
	//+kubebuilder:validation:MinItems=1
	//+optional
	Tags []string `json:"tags,omitempty"`
	// Terraform Cloud workspaces can only be accessed by users with the correct permissions.
	// You can manage permissions for a workspace on a per-team basis.
	// When a workspace is created, only the owners team and teams with the "manage workspaces" permission can access it,
	// with full admin permissions. These teams' access can't be removed from a workspace.
	// More information:
	//  - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/access
	//
	//+kubebuilder:validation:MinItems=1
	//+optional
	TeamAccess []*TeamAccess `json:"teamAccess,omitempty"`
	// The version of Terraform to use for this workspace.
	// If not specified, the latest available version will be used.
	// More information:
	//  - https://www.terraform.io/cloud-docs/workspaces/settings#terraform-version
	//
	//+kubebuilder:validation:Pattern="^\\d{1}\\.\\d{1,2}\\.\\d{1,2}$"
	//+optional
	TerraformVersion string `json:"terraformVersion,omitempty"`
	// The directory where Terraform will execute, specified as a relative path from the root of the configuration directory.
	// More information:
	//  - https://www.terraform.io/cloud-docs/workspaces/settings#terraform-working-directory
	//
	//+kubebuilder:validation:MinLength=1
	//+optional
	WorkingDirectory string `json:"workingDirectory,omitempty"`
	// Terraform Environment variables for all plans and applies in this workspace.
	// Variables defined within a workspace always overwrite variables from variable sets that have the same type and the same key.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables
	//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables#environment-variables
	//
	//+kubebuilder:validation:MinItems=1
	//+optional
	EnvironmentVariables []Variable `json:"environmentVariables,omitempty"`
	// Terraform variables for all plans and applies in this workspace.
	// Variables defined within a workspace always overwrite variables from variable sets that have the same type and the same key.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables
	//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables#terraform-variables
	//
	//+kubebuilder:validation:MinItems=1
	//+optional
	TerraformVariables []Variable `json:"terraformVariables,omitempty"`
	// Remote state access between workspaces.
	// By default, new workspaces in Terraform Cloud do not allow other workspaces to access their state.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/state#accessing-state-from-other-workspaces
	//+optional
	RemoteStateSharing *RemoteStateSharing `json:"remoteStateSharing,omitempty"`
	// Run triggers allow you to connect this workspace to one or more source workspaces.
	// These connections allow runs to queue automatically in this workspace on successful apply of runs in any of the source workspaces.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-triggers
	//
	//+kubebuilder:validation:MinItems=1
	//+optional
	RunTriggers []RunTrigger `json:"runTriggers,omitempty"`
	// Settings for the workspace's VCS repository, enabling the UI/VCS-driven run workflow.
	// Omit this argument to utilize the CLI-driven and API-driven workflows, where runs are not driven by webhooks on your VCS provider.
	// More information:
	//  - https://www.terraform.io/cloud-docs/run/ui
	//  - https://www.terraform.io/cloud-docs/vcs
	//
	//+optional
	VersionControl *VersionControl `json:"versionControl,omitempty"`
	// SSH key used to clone Terraform modules.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/ssh-keys
	//
	//+optional
	SSHKey *SSHKey `json:"sshKey,omitempty"`
}

type RunStatus struct {
	// Current(both active and finished) Terraform Cloud run ID.
	//+optional
	ID string `json:"id,omitempty"`
	// Current(both active and finished) Terraform Cloud run status.
	//+optional
	Status string `json:"status,omitempty"`
	//+optional
	ConfigurationVersion string `json:"configurationVersion"`
	// Run ID of the latest run that could update the outputs.
	//+optional
	OutputRunID string `json:"outputRunID,omitempty"`
}

// WorkspaceStatus defines the observed state of Workspace
type WorkspaceStatus struct {
	// Real world state generation
	ObservedGeneration int64 `json:"observedGeneration"`
	// Workspace last update timestamp
	UpdateAt int64 `json:"updateAt"`
	// Workspace ID that is managed by the controller
	WorkspaceID string `json:"workspaceID"`

	// Workspace Runs status
	//+optional
	Run RunStatus `json:"runStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Workspace ID",type=string,JSONPath=`.status.workspaceID`

// Workspace is the Schema for the workspaces API
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceSpec   `json:"spec"`
	Status WorkspaceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorkspaceList contains a list of Workspace
type WorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workspace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workspace{}, &WorkspaceList{})
}
