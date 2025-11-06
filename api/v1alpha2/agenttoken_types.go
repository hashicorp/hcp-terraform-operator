// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The Management Policy defines how the controller will manage tokens in the specified Agent Pool.
// - `merge`  — the controller will manage its tokens alongside any existing tokens in the pool, without modifying or deleting tokens it does not own.
// - `owner`  — the controller assumes full ownership of all agent tokens in the pool, managing and potentially modifying or deleting all tokens, including those not created by it.
type AgentTokenManagementPolicy string

const (
	AgentTokenManagementPolicyMerge AgentTokenManagementPolicy = "merge"
	AgentTokenManagementPolicyOwner AgentTokenManagementPolicy = "owner"
)

// The Deletion Policy defines how managed tokens and Kubernetes Secrets should be handled when the custom resource is deleted.
//   - `retain`: When the custom resource is deleted, the operator will remove only the resource itself.
//     The managed HCP Terraform Agent tokens will remain active on the HCP Terraform side, and the corresponding Kubernetes Secret will not be modified.
//   - `destroy`: The operator will attempt to delete the managed HCP Terraform Agent tokens and remove the corresponding Kubernetes Secret.
type AgentTokenDeletionPolicy string

const (
	AgentTokenDeletionPolicyRetain  AgentTokenDeletionPolicy = "retain"
	AgentTokenDeletionPolicyDestroy AgentTokenDeletionPolicy = "destroy"
)

// AgentTokenSpec defines the desired state of AgentToken.
type AgentTokenSpec struct {
	// Organization name where the Workspace will be created.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations
	//
	//+kubebuilder:validation:MinLength:=1
	Organization string `json:"organization"`
	// API Token to be used for API calls.
	Token Token `json:"token"`
	// The Deletion Policy defines how managed tokens and Kubernetes Secrets should be handled when the custom resource is deleted.
	// - `retain`: When the custom resource is deleted, the operator will remove only the resource itself.
	//   The managed HCP Terraform Agent tokens will remain active on the HCP Terraform side, and the corresponding Kubernetes Secret will not be modified.
	// - `destroy`: The operator will attempt to delete the managed HCP Terraform Agent tokens and remove the corresponding Kubernetes Secret.
	// Default: `retain`.
	//
	//+kubebuilder:validation:Enum:=retain;destroy
	//+kubebuilder:default=retain
	//+optional
	DeletionPolicy AgentTokenDeletionPolicy `json:"deletionPolicy,omitempty"`
	// The Agent Pool name or ID where the tokens will be managed.
	AgentPool AgentPoolRef `json:"agentPool"`
	// The Management Policy defines how the controller will manage tokens in the specified Agent Pool.
	// - `merge`  — the controller will manage its tokens alongside any existing tokens in the pool, without modifying or deleting tokens it does not own.
	// - `owner`  — the controller assumes full ownership of all agent tokens in the pool, managing and potentially modifying or deleting all tokens, including those not created by it.
	// Default: `merge`.
	//
	//+kubebuilder:validation:Enum:=merge;owner
	//+kubebuilder:default=merge
	//+optional
	ManagementPolicy AgentTokenManagementPolicy `json:"managementPolicy,omitempty"`
	// List of the HCP Terraform Agent tokens to manage.
	//
	//+kubebuilder:validation:MinItems:=1
	AgentTokens []AgentAPIToken `json:"agentTokens"`
	// secretName specifies the name of the Kubernetes Secret
	// where the HCP Terraform Agent tokens are stored.
	//
	//+kubebuilder:validation:MinLength:=1
	SecretName string `json:"secretName"`
}

// AgentTokenStatus defines the observed state of AgentToken.
type AgentTokenStatus struct {
	// Real world state generation.
	ObservedGeneration int64 `json:"observedGeneration"`
	// Agent Pool where tokens are managed by the controller.
	AgentPool *AgentPoolRef `json:"agentPool,omitempty"`
	// List of the agent tokens managed by the controller.
	//
	//+optional
	AgentTokens []*AgentAPIToken `json:"agentTokens,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Pool Name",type=string,JSONPath=`.status.agentPool.name`
//+kubebuilder:printcolumn:name="Pool ID",type=string,JSONPath=`.status.agentPool.id`
//+kubebuilder:metadata:labels="app.terraform.io/crd-schema-version=v25.11.0"

// AgentToken manages HCP Terraform Agent Tokens.
// More information:
// - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens#agent-api-tokens
type AgentToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentTokenSpec   `json:"spec"`
	Status AgentTokenStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AgentTokenList contains a list of AgentToken.
type AgentTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentToken `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentToken{}, &AgentTokenList{})
}
