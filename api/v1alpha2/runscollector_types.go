// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RunsCollectorSpec struct {
	// Organization name where the Workspace will be created.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations
	//
	//+kubebuilder:validation:MinLength:=1
	Organization string `json:"organization"`
	// API Token to be used for API calls.
	Token Token `json:"token"`

	// The Agent Pool name or ID from which the controller will collect runs.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/run/states
	AgentPool *AgentPoolRef `json:"agentPool"`
}

type RunsCollectorStatus struct {
	// Real world state generation.
	ObservedGeneration int64 `json:"observedGeneration"`
	// The Agent Pool name or ID from which the controller will collect runs.
	AgentPool *AgentPoolRef `json:"agentPool,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Pool ID",type=string,JSONPath=`.status.agentPool.id`
//+kubebuilder:printcolumn:name="Pool Name",type=string,JSONPath=`.status.agentPool.name`
//+kubebuilder:metadata:labels="app.terraform.io/crd-schema-version=v25.11.0"

// Runs Collector scraptes HCP Terraform Run statuses from a given Agent Pool and exposes them as Prometheus-compatible metrics.
// More information:
//   - https://developer.hashicorp.com/terraform/cloud-docs/run/remote-operations
type RunsCollector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunsCollectorSpec   `json:"spec"`
	Status RunsCollectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RunsCollectorList contains a list of RunsCollector.
type RunsCollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RunsCollector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RunsCollector{}, &RunsCollectorList{})
}
