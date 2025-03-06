// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	tfc "github.com/hashicorp/go-tfe"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Custom permissions let you assign specific, finer-grained permissions to a team than the broader fixed permission sets provide.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#custom-project-permissions
//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#general-workspace-permissions
type CustomProjectPermissions struct {
	// Project access.
	// Must be one of the following values: `delete`, `read`, `update`.
	// Default: `read`.
	//
	//+kubebuilder:validation:Enum:=delete;read;update
	//+kubebuilder:default:=read
	//+optional
	ProjectAccess tfc.ProjectSettingsPermissionType `json:"projectAccess,omitempty"`
	// Team management.
	// Must be one of the following values: `manage`, `none`, `read`.
	// Default: `none`.
	//
	//+kubebuilder:validation:Enum:=manage;none;read
	//+kubebuilder:default:=none
	//+optional
	TeamManagement tfc.ProjectTeamsPermissionType `json:"teamManagement,omitempty"`
	// Allow users to create workspaces in the project.
	// This grants read access to all workspaces in the project.
	// Default: `false`.
	//
	//+kubebuilder:default:=false
	//+optional
	CreateWorkspace bool `json:"createWorkspace,omitempty"`
	// Allows users to delete workspaces in the project.
	// Default: `false`.
	//
	//+kubebuilder:default:=false
	//+optional
	DeleteWorkspace bool `json:"deleteWorkspace,omitempty"`
	// Allows users to move workspaces out of the project.
	// A user must have this permission on both the source and destination project to successfully move a workspace from one project to another.
	// Default: `false`.
	//
	//+kubebuilder:default:=false
	//+optional
	MoveWorkspace bool `json:"moveWorkspace,omitempty"`
	// Allows users to manually lock the workspace to temporarily prevent runs.
	// When a workspace's execution mode is set to "local", users must have this permission to perform local CLI runs using the workspace's state.
	// Default: `false`.
	//
	//+kubebuilder:default:=false
	//+optional
	LockWorkspace bool `json:"lockWorkspace,omitempty"`
	// Run access.
	// Must be one of the following values: `apply`, `plan`, `read`.
	// Default: `read`.
	//
	//+kubebuilder:validation:Enum:=apply;plan;read
	//+kubebuilder:default:=read
	//+optional
	Runs tfc.WorkspaceRunsPermissionType `json:"runs,omitempty"`
	// Manage Workspace Run Tasks.
	// Default: `false`.
	//
	//+kubebuilder:validation:default:=false
	//+optional
	RunTasks bool `json:"runTasks,omitempty"`
	// Download Sentinel mocks.
	// Must be one of the following values: `none`, `read`.
	// Default: `none`.
	//
	//+kubebuilder:validation:Enum:=none;read
	//+kubebuilder:default:=none
	//+optional
	SentinelMocks tfc.WorkspaceSentinelMocksPermissionType `json:"sentinelMocks,omitempty"`
	// State access.
	// Must be one of the following values: `none`, `read`, `read-outputs`, `write`.
	// Default: `none`.
	//
	//+kubebuilder:validation:Enum:=none;read;read-outputs;write
	//+kubebuilder:default:=none
	//+optional
	StateVersions tfc.WorkspaceStateVersionsPermissionType `json:"stateVersions,omitempty"`
	// Variable access.
	// Must be one of the following values: `none`, `read`, `write`.
	// Default: `none`.
	//
	//+kubebuilder:validation:Enum:=none;read;write
	//+kubebuilder:default:=none
	//+optional
	Variables tfc.WorkspaceVariablesPermissionType `json:"variables,omitempty"`
}

// HCP Terraform's access model is team-based. In order to perform an action within a HCP Terraform organization,
// users must belong to a team that has been granted the appropriate permissions.
// You can assign project-specific permissions to teams.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/organize-workspaces-with-projects#permissions
//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#project-permissions
type ProjectTeamAccess struct {
	// Team to grant access.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/teams
	Team Team `json:"team"`
	// There are two ways to choose which permissions a given team has on a project: fixed permission sets, and custom permissions.
	// Must be one of the following values: `admin`, `custom`, `maintain`, `read`, `write`.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#project-permissions
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#general-project-permissions
	//
	//+kubebuilder:validation:Enum:=admin;custom;maintain;read;write
	Access tfc.TeamProjectAccessType `json:"access"`
	// Custom permissions let you assign specific, finer-grained permissions to a team than the broader fixed permission sets provide.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#custom-project-permissions
	//
	//+optional
	Custom *CustomProjectPermissions `json:"custom,omitempty"`
}

// DeletionPolicy defines the strategy the Kubernetes operator uses when you delete a project, either manually or by a system event.
//
// You must use one of the following values:
// - `retain`: When the custom resource is deleted, the operator will not delete the associated project.
// - `destroy`: Performs a destroy operation to remove the project. The project must be empty.
// - `force`: Forcefully and immediately removes all resources in the project. Once this is completed the operator deletes the project.
type ProjectDeletionPolicy string

const (
	ProjectDeletionPolicyRetain  ProjectDeletionPolicy = "retain"
	ProjectDeletionPolicyDestroy ProjectDeletionPolicy = "destroy"
	ProjectDeletionPolicyForce   ProjectDeletionPolicy = "force"
)

// ProjectSpec defines the desired state of Project.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/organize-workspaces-with-projects
type ProjectSpec struct {
	// Organization name where the Workspace will be created.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations
	//
	//+kubebuilder:validation:MinLength:=1
	Organization string `json:"organization"`
	// API Token to be used for API calls.
	Token Token `json:"token"`
	// Name of the Project.
	//
	//+kubebuilder:validation:MinLength:=1
	Name string `json:"name"`

	// HCP Terraform's access model is team-based. In order to perform an action within a HCP Terraform organization,
	// users must belong to a team that has been granted the appropriate permissions.
	// You can assign project-specific permissions to teams.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/organize-workspaces-with-projects#permissions
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#project-permissions
	//
	//+kubebuilder:validation:MinItems:=1
	//+optional
	TeamAccess []*ProjectTeamAccess `json:"teamAccess,omitempty"`
	// DeletionPolicyForProject defines the strategy the Kubernetes operator uses when you delete a project, either manually or by a system event.
	//
	// You must use one of the following values:
	// - `retain`:  When the custom resource is deleted, the operator will not delete the associated project.
	// - `destroy`: Performs a destroy operation to remove the project. The project must be empty.
	// - `force`: Forcefully and immediately removes all resources in the project. Once this is completed the operator deletes the project.
	// Default: `retain`.
	//
	//+kubebuilder:validation:Enum:=retain;destroy;force
	//+kubebuilder:default=retain
	//+optional
	DeletionPolicy ProjectDeletionPolicy `json:"deletionPolicy,omitempty"`
}

// ProjectStatus defines the observed state of Project.
type ProjectStatus struct {
	// Real world state generation.
	ObservedGeneration int64 `json:"observedGeneration"`
	// Project ID.
	ID string `json:"id"`
	// Project name.
	Name string `json:"name"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Project Name",type=string,JSONPath=`.status.name`
//+kubebuilder:printcolumn:name="Project ID",type=string,JSONPath=`.status.id`

// Project manages HCP Terraform Projects.
// More information:
// - https://developer.hashicorp.com/terraform/cloud-docs/projects/manage
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec"`
	Status ProjectStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
