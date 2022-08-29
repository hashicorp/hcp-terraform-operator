package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentPool allows Terraform Cloud to communicate with isolated, private, or on-premises infrastructure.
// More information:
// - https://www.terraform.io/cloud-docs/agents
type AgentPool struct {
	// Agent Pool ID.
	//+kubebuilder:validation:Pattern="^apool-[a-zA-Z0-9]+$"
	//+optional
	ID string `json:"id,omitempty"`
	// Agent Pool name.
	//+optional
	Name string `json:"name,omitempty"`
}

// Token refers to a Kubernetes Secret object within the same namespace as the Workspace object
type Token struct {
	// Selects a key of a secret in the workspace's namespace
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef"`
}

// Source for the variable's value. Cannot be used if value is not empty.
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
//  - https://www.terraform.io/cloud-docs/workspaces/variables
type Variable struct {
	// Name of the variable.
	Name string `json:"name"`
	// Description of the variable.
	//+optional
	Description string `json:"description,omitempty"`
	// Parse this field as HashiCorp Configuration Language (HCL). This allows you to interpolate values at runtime.
	//+kubebuilder:default:=false
	//+optional
	HCL bool `json:"hcl"`
	// Sensitive variables are never shown in the UI or API. They may appear in Terraform logs if your configuration is designed to output them.
	//+kubebuilder:default:=false
	//+optional
	Sensitive bool `json:"sensitive"`
	// Value of the variable.
	//+optional
	Value string `json:"value,omitempty"`
	// Source for the variable's value. Cannot be used if value is not empty.
	//+optional
	ValueFrom *ValueFrom `json:"valueFrom,omitempty"`
}

// WorkspaceSpec defines the desired state of Workspace
type WorkspaceSpec struct {
	// API Token to be used for API calls
	Token Token `json:"token"`
	// Workspace name
	Name string `json:"name"`
	// Organization name where the Workspace will be created
	// More information: https://www.terraform.io/cloud-docs/users-teams-organizations/organizations
	Organization string `json:"organization"`

	// Define either change will be applied automatically(auto) or require an operator to confirm(manual).
	// More information: https://www.terraform.io/cloud-docs/workspaces/settings#auto-apply-and-manual-apply
	//+kubebuilder:validation:Pattern="^(auto|manual)$"
	//+kubebuilder:default=manual
	//+optional
	ApplyMethod string `json:"applyMethod"`
	// Workspace description
	//+optional
	Description string `json:"description,omitempty"`
	// Terraform Cloud Agents allow Terraform Cloud to communicate with isolated, private, or on-premises infrastructure.
	// More information: https://www.terraform.io/cloud-docs/agents
	//+optional
	AgentPool *AgentPool `json:"agentPool,omitempty"`
	// Define where the Terraform code will be executed.
	// More information: https://www.terraform.io/cloud-docs/workspaces/settings#execution-mode
	//+kubebuilder:validation:Pattern="^(agent|local|remote)$"
	//+kubebuilder:default=remote
	//+optional
	ExecutionMode string `json:"executionMode"`
	// Workspace tags are used to help identify and group together workspaces.
	//+optional
	Tags []string `json:"tags,omitempty"`
	// The version of Terraform to use for this workspace.
	// If not specified, the latest available version will be used.
	// More information: https://www.terraform.io/cloud-docs/workspaces/settings#terraform-version
	//+kubebuilder:validation:Pattern="^\\d{1}\\.\\d{1,2}\\.\\d{1,2}$"
	//+optional
	TerraformVersion string `json:"terraformVersion,omitempty"`
	// The directory where Terraform will execute, specified as a relative path from the root of the configuration directory.
	// More information: https://www.terraform.io/cloud-docs/workspaces/settings#terraform-working-directory
	//+optional
	WorkingDirectory string `json:"workingDirectory,omitempty"`
	// Terraform Environment variables for all plans and applies in this workspace.
	// Variables defined within a workspace always overwrite variables from variable sets that have the same type and the same key.
	// More information:
	//  - https://www.terraform.io/cloud-docs/workspaces/variables
	//  - https://www.terraform.io/cloud-docs/workspaces/variables##environment-variables
	//+optional
	EnvironmentVariables []Variable `json:"environmentVariables,omitempty"`
	// Terraform variables for all plans and applies in this workspace.
	// Variables defined within a workspace always overwrite variables from variable sets that have the same type and the same key.
	// More information:
	//  - https://www.terraform.io/cloud-docs/workspaces/variables
	//  - https://www.terraform.io/cloud-docs/workspaces/variables#terraform-variables
	//+optional
	TerraformVariables []Variable `json:"terraformVariables,omitempty"`
}

// WorkspaceStatus defines the observed state of Workspace
type WorkspaceStatus struct {
	// Real world state generation
	ObservedGeneration int64 `json:"observedGeneration"`
	// Workspace last update timestamp
	UpdateAt int64 `json:"updateAt"`
	// Workspace ID that is managed by the controller
	WorkspaceID string `json:"workspaceID"`
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
