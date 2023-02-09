// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Module source and version to execute.
type ModuleSource struct {
	// Non local Terraform module source.
	// More information:
	//   - https://developer.hashicorp.com/terraform/language/modules/sources
	//
	//+kubebuilder:validation:MinLength:=1
	Source string `json:"source"`
	// Terraform module version.
	//
	//+kubebuilder:validation:MinLength:=1
	//+optional
	Version string `json:"version,omitempty"`
}

// Workspace to execute the module.
// Only one of the fields `ID` or `Name` is allowed.
// At least one of the fields `ID` or `Name` is mandatory.
type ModuleWorkspace struct {
	// Module Workspace ID.
	//
	//+kubebuilder:validation:Pattern="^ws-[a-zA-Z0-9]+$"
	//+optional
	ID string `json:"id,omitempty"`
	// Module Workspace Name.
	//
	//+kubebuilder:validation:MinLength:=1
	//+optional
	Name string `json:"name,omitempty"`
}

// A configuration version is a resource used to reference the uploaded configuration files.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/api-docs/configuration-versions
//   - https://developer.hashicorp.com/terraform/cloud-docs/run/api
type ConfigurationVersionStatus struct {
	// Configuration Version ID.
	ID string `json:"id"`
	// Configuration Version Status.
	Status string `json:"status"`
}

// Module Outputs status.
type OutputStatus struct {
	// Run ID of the latest run that updated the outputs.
	RunID string `json:"runID"`
}

// Variables to pass to the module.
type ModuleVariable struct {
	// Variable name must exist in the Workspace.
	//
	//+kubebuilder:validation:MinLength:=1
	Name string `json:"name"`
}

// Module outputs to store in ConfigMap(non-sensitive) or Secret(sensitive).
type ModuleOutput struct {
	// Output name must match with the module output.
	//
	//+kubebuilder:validation:MinLength:=1
	Name string `json:"name"`
	// Specify whether or not the output is sensitive.
	//
	//+kubebuilder:default:=false
	//+optional
	Sensitive bool `json:"sensitive,omitempty"`
}

// ModuleSpec defines the desired state of Module.
type ModuleSpec struct {
	// API Token to be used for API calls.
	Token Token `json:"token"`
	// Organization name where the Workspace will be created.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations
	Organization string `json:"organization"`
	// Module source and version to execute.
	Module *ModuleSource `json:"module"`
	// Workspace to execute the module.
	Workspace *ModuleWorkspace `json:"workspace"`
	// Variables to pass to the module, they must exist in the Workspace.
	//
	//+kubebuilder:validation:MinItems:=1
	// +optional
	Variables []ModuleVariable `json:"variables,omitempty"`
	// Module outputs to store in ConfigMap(non-sensitive) or Secret(sensitive).
	//
	//+kubebuilder:validation:MinItems:=1
	//+optional
	Outputs []ModuleOutput `json:"outputs,omitempty"`
	// Specify whether or not to execute a Destroy run when the object is deleted from the Kubernetes.
	//
	//+kubebuilder:default:=false
	//+optional
	DestroyOnDeletion bool `json:"destroyOnDeletion,omitempty"`
	// Allows executing a new Run without changing any Workspace or Module attributes.
	// Example: kubectl patch <KIND> <NAME> --type=merge --patch '{"spec": {"restartedAt": "'`date -u -Iseconds`'"}}'
	//
	//+kubebuilder:validation:MinLength:=1
	//+optional
	RestartedAt string `json:"restartedAt,omitempty"`
}

// ModuleStatus defines the observed state of Module.
type ModuleStatus struct {
	ObservedGeneration int64 `json:"observedGeneration"`
	// Workspace ID where the module is running.
	WorkspaceID string `json:"workspaceID"`
	// A configuration version is a resource used to reference the uploaded configuration files.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/api-docs/configuration-versions
	//   - https://developer.hashicorp.com/terraform/cloud-docs/run/api
	ConfigurationVersion *ConfigurationVersionStatus `json:"configurationVersion,omitempty"`
	// Workspace Runs status.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/run/states
	Run *RunStatus `json:"run,omitempty"`
	// Module Outputs status.
	Output *OutputStatus `json:"output,omitempty"`
	// Workspace Destroy Run status.
	//
	//+optional
	DestroyRunID string `json:"destroyRunID,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="CV Status",type=string,JSONPath=`.status.configurationVersion.status`
//+kubebuilder:printcolumn:name="Run Status",type=string,JSONPath=`.status.run.status`

// Module is the Schema for the modules API
// Module implements the API-driven Run Workflow
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/run/api
type Module struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModuleSpec   `json:"spec"`
	Status ModuleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ModuleList contains a list of Module
type ModuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Module `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Module{}, &ModuleList{})
}
