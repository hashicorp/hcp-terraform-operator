// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StackProject defines the project where the Stack will be created.
// Only one of the fields `ID` or `Name` is allowed.
// At least one of the fields `ID` or `Name` is mandatory.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/stacks
type StackProject struct {
	// Project ID.
	// Must match pattern: `^prj-[a-zA-Z0-9]+$`
	//
	//+kubebuilder:validation:Pattern:="^prj-[a-zA-Z0-9]+$"
	//+optional
	ID string `json:"id,omitempty"`
	// Project name.
	//
	//+kubebuilder:validation:MinLength:=1
	//+optional
	Name string `json:"name,omitempty"`
}

// StackVCSRepo defines the VCS repository configuration for the Stack.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/stacks
type StackVCSRepo struct {
	// The VCS Connection (OAuth Connection + Token) to use.
	// Must match pattern: `^ot-[a-zA-Z0-9]+$`
	//
	//+kubebuilder:validation:Pattern:="^ot-[a-zA-Z0-9]+$"
	Identifier string `json:"identifier"`
	// The repository branch that Stack will execute from.
	//
	//+kubebuilder:validation:MinLength:=1
	//+optional
	Branch string `json:"branch,omitempty"`
	// The path to the Stack configuration file in the repository.
	//
	//+kubebuilder:validation:MinLength:=1
	//+optional
	Path string `json:"path,omitempty"`
	// GitHub App installation ID. Required for GitHub App connections.
	//
	//+optional
	GHAInstallationID string `json:"ghaInstallationId,omitempty"`
}

// StackDeployment defines deployment configuration for the Stack.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/stacks
type StackDeployment struct {
	// Names of the deployments to create.
	//
	//+kubebuilder:validation:MinItems:=1
	Names []string `json:"names"`
}

// StackDeletionPolicy defines the strategy the Kubernetes operator uses when you delete a stack, either manually or by a system event.
//
// You must use one of the following values:
// - `retain`: When you delete the custom resource, the operator does not delete the stack.
// - `delete`: The operator will attempt to delete the stack and all its deployments.
type StackDeletionPolicy string

const (
	StackDeletionPolicyRetain StackDeletionPolicy = "retain"
	StackDeletionPolicyDelete StackDeletionPolicy = "delete"
)

// StackSpec defines the desired state of Stack.
type StackSpec struct {
	// Stack name.
	//
	//+kubebuilder:validation:MinLength:=1
	Name string `json:"name"`
	// Organization name where the Stack will be created.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations
	//
	//+kubebuilder:validation:MinLength:=1
	Organization string `json:"organization"`
	// API Token to be used for API calls.
	Token Token `json:"token"`
	// Project where the Stack will be created.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/stacks
	//
	//+optional
	Project *StackProject `json:"project,omitempty"`
	// VCS repository configuration for the Stack.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/stacks
	//
	//+optional
	VCSRepo *StackVCSRepo `json:"vcsRepo,omitempty"`
	// Stack description.
	//
	//+kubebuilder:validation:MinLength:=1
	//+optional
	Description string `json:"description,omitempty"`
	// Terraform version to use for this stack.
	// If not specified, the latest available version will be used.
	// Must match pattern: `^\d{1}\.\d{1,2}\.\d{1,2}$`
	//
	//+kubebuilder:validation:Pattern:="^\\d{1}\\.\\d{1,2}\\.\\d{1,2}$"
	//+optional
	TerraformVersion string `json:"terraformVersion,omitempty"`
	// Terraform Environment variables for all deployments in this stack.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/stacks
	//
	//+kubebuilder:validation:MinItems:=1
	//+optional
	EnvironmentVariables []Variable `json:"environmentVariables,omitempty"`
	// Terraform variables for all deployments in this stack.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/stacks
	//
	//+kubebuilder:validation:MinItems:=1
	//+optional
	TerraformVariables []Variable `json:"terraformVariables,omitempty"`
	// Deployment configuration for the Stack.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/stacks
	//
	//+optional
	Deployment *StackDeployment `json:"deployment,omitempty"`
	// The Deletion Policy specifies the behavior of the custom resource and its associated stack when the custom resource is deleted.
	// - `retain`: When you delete the custom resource, the operator does not delete the stack.
	// - `delete`: The operator will attempt to delete the stack and all its deployments.
	// Default: `retain`.
	//
	//+kubebuilder:validation:Enum:=retain;delete
	//+kubebuilder:default=retain
	//+optional
	DeletionPolicy StackDeletionPolicy `json:"deletionPolicy,omitempty"`
}

// DeploymentStatus defines the status of a Stack deployment.
type DeploymentStatus struct {
	// Deployment name.
	Name string `json:"name"`
	// Deployment ID.
	ID string `json:"id"`
	// Deployment status.
	Status string `json:"status"`
	// Last updated timestamp.
	//
	//+optional
	UpdatedAt int64 `json:"updatedAt,omitempty"`
}

// StackStatus defines the observed state of Stack.
type StackStatus struct {
	// Real world state generation.
	//
	//+optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Stack ID that is managed by the controller.
	StackID string `json:"stackID"`
	// Stack last update timestamp.
	//
	//+optional
	UpdatedAt int64 `json:"updatedAt,omitempty"`
	// Stack Terraform version.
	//
	//+kubebuilder:validation:Pattern:="^\\d{1}\\.\\d{1,2}\\.\\d{1,2}$"
	//+optional
	TerraformVersion string `json:"terraformVersion,omitempty"`
	// Deployments status.
	//
	//+optional
	Deployments []DeploymentStatus `json:"deployments,omitempty"`
	// Default organization project ID.
	//
	//+optional
	DefaultProjectID string `json:"defaultProjectID,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Stack ID",type=string,JSONPath=`.status.stackID`
//+kubebuilder:metadata:labels="app.terraform.io/crd-schema-version=v25.4.0"

// Stack manages HCP Terraform Stacks.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/stacks
type Stack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StackSpec   `json:"spec"`
	Status StackStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StackList contains a list of Stack
type StackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Stack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stack{}, &StackList{})
}

// Made with Bob
