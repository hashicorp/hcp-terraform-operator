// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Agent Token is a secret token that a HCP Terraform Agent is used to connect to the HCP Terraform Agent Pool.
// In `spec` only the field `Name` is allowed, the rest are used in `status`.
// More infromation:
//   - https://developer.hashicorp.com/terraform/cloud-docs/agents
type AgentToken struct {
	// Agent Token name.
	//
	//+kubebuilder:validation:MinLength:=1
	Name string `json:"name"`
	// Agent Token ID.
	//
	//+kubebuilder:validation:Pattern:="^at-[a-zA-Z0-9]+$"
	//+optional
	ID string `json:"id,omitempty"`
	// Timestamp of when the agent token was created.
	//
	//+optional
	CreatedAt *int64 `json:"createdAt,omitempty"`
	// Timestamp of when the agent token was last used.
	//
	//+optional
	LastUsedAt *int64 `json:"lastUsedAt,omitempty"`
}

// TargetWorkspace is the name or ID of the workspace you want autoscale against.
type TargetWorkspace struct {
	// Workspace ID
	//
	//+optional
	ID string `json:"id,omitempty"`
	// Workspace Name
	//
	//+kubebuilder:validation:MinLength:=1
	//+optional
	Name string `json:"name,omitempty"`

	// Wildcard Name to match match workspace names using `*` on name suffix, prefix, or both.
	//
	//+kubebuilder:validation:MinLength:=1
	//+optional
	WildcardName string `json:"wildcardName,omitempty"`
}

// AgentDeploymentAutoscaling allows you to configure the operator
// to scale the deployment for an AgentPool up and down to meet demand.
type AgentDeploymentAutoscaling struct {
	// MaxReplicas is the maximum number of replicas for the Agent deployment.
	MaxReplicas *int32 `json:"maxReplicas"`

	// MinReplicas is the minimum number of replicas for the Agent deployment.
	MinReplicas *int32 `json:"minReplicas"`

	// TargetWorkspaces is a list of HCP Terraform Workspaces which
	// the agent pool should scale up to meet demand. When this field
	// is ommited the autoscaler will target all workspaces that are
	// associated with the AgentPool.
	//
	//+optional
	TargetWorkspaces *[]TargetWorkspace `json:"targetWorkspaces"`

	// CooldownPeriodSeconds is the time to wait between scaling events. Defaults to 300.
	//
	//+optional
	//+kubebuilder:default:=300
	CooldownPeriodSeconds *int32 `json:"cooldownPeriodSeconds,omitempty"`

	// CoolDownPeriod configures the period to wait between scaling up and scaling down
	//+optional
	CooldownPeriod *AgentDeploymentAutoscalingCooldownPeriod `json:"cooldownPeriod,omitempty"`
}

// AgentDeploymentAutoscalingCooldownPeriod configures the period to wait between scaling up and scaling down
type AgentDeploymentAutoscalingCooldownPeriod struct {
	// ScaleUpSeconds is the time to wait before scaling up.
	//+optional
	ScaleUpSeconds *int32 `json:"scaleUpSeconds,omitempty"`

	// ScaleDownSeconds is the time to wait before scaling down.
	//+optional
	ScaleDownSeconds *int32 `json:"scaleDownSeconds,omitempty"`
}

type AgentDeployment struct {
	Replicas *int32      `json:"replicas,omitempty"`
	Spec     *v1.PodSpec `json:"spec,omitempty"`

	//Labels are intended to specify identifying attributes of objects
	//that are meaningful and relevant to users.These are applied to the
	//pod template within the AgentDeployment, helping to organize and select
	//pods.
	//+optional
	Labels map[string]string `json:"labels,omitempty"`

	//Annotations are used to provide metadata. They are applied to the pod
	//template within the AgentDeployment.
	//+optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// AgentPoolSpec defines the desired state of AgentPool.
type AgentPoolSpec struct {
	// Agent Pool name.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/agents/agent-pools
	//
	//+kubebuilder:validation:MinLength:=1
	Name string `json:"name"`
	// Organization name where the Workspace will be created.
	// More information:
	//   - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations
	//
	//+kubebuilder:validation:MinLength:=1
	Organization string `json:"organization"`
	// API Token to be used for API calls.
	Token Token `json:"token"`

	// List of the agent tokens to generate.
	//
	//+kubebuilder:validation:MinItems:=1
	//+optional
	AgentTokens []*AgentToken `json:"agentTokens,omitempty"`

	// Agent deployment settings
	//+optional
	AgentDeployment *AgentDeployment `json:"agentDeployment,omitempty"`

	// Agent deployment settings
	//+optional
	AgentDeploymentAutoscaling *AgentDeploymentAutoscaling `json:"autoscaling,omitempty"`
}

// AgentDeploymentAutoscalingStatus
type AgentDeploymentAutoscalingStatus struct {
	// Desired number of agent replicas
	//+optional
	DesiredReplicas *int32 `json:"desiredReplicas,omitempty"`

	// Last time the agent pool was scaledx
	//+optional
	LastScalingEvent *metav1.Time `json:"lastScalingEvent,omitempty"`
}

// AgentPoolStatus defines the observed state of AgentPool.
type AgentPoolStatus struct {
	// Real world state generation.
	ObservedGeneration int64 `json:"observedGeneration"`
	// Agent Pool ID that is managed by the controller.
	AgentPoolID string `json:"agentPoolID"`
	// List of the agent tokens generated by the controller.
	//
	//+optional
	AgentTokens []*AgentToken `json:"agentTokens,omitempty"`
	// Name of the agent deployment generated by the controller.
	//
	//+optional
	AgentDeploymentName string `json:"agentDeploymentName,omitempty"`
	// Autoscaling Status
	//
	//+optional
	AgentDeploymentAutoscalingStatus *AgentDeploymentAutoscalingStatus `json:"autoscaling,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AgentPool is the Schema for the agentpools API.
type AgentPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentPoolSpec   `json:"spec"`
	Status AgentPoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AgentPoolList contains a list of AgentPool.
type AgentPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentPool{}, &AgentPoolList{})
}
