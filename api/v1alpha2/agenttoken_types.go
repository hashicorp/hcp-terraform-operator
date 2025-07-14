// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AgentTokenManagementPolicy string

const (
	AgentTokenManagementPolicySelf  AgentTokenManagementPolicy = "self"
	AgentTokenManagementPolicyOwner AgentTokenManagementPolicy = "owner"
)

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
	//+kubebuilder:validation:Enum:=retain;destroy
	//+kubebuilder:default=retain
	//+optional
	DeletionPolicy AgentTokenDeletionPolicy `json:"deletionPolicy,omitempty"`
	AgentPool      AgentPoolRef             `json:"agentPool"`
	//+kubebuilder:validation:Enum:=self;owner
	//+kubebuilder:default=self
	//+optional
	ManagementPolicy AgentTokenManagementPolicy `json:"managementPolicy,omitempty"`
	//+kubebuilder:validation:MinItems:=1
	AgentTokens []AgentAPIToken `json:"agentTokens"`
}

// AgentTokenStatus defines the observed state of AgentToken.
type AgentTokenStatus struct {
	AgentPool *AgentPoolRef `json:"agentPool,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Agent Pool Name",type=string,JSONPath=`.status.agentPool.name`
//+kubebuilder:printcolumn:name="Agent Pool ID",type=string,JSONPath=`.status.agentPool.id`
//+kubebuilder:metadata:labels="app.terraform.io/crd-schema-version=v25.9.0"

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
