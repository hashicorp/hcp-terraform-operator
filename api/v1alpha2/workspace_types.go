package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretKeyRef refers to a Kubernetes Secret object within the same namespace as the Workspace object
type SecretKeyRef struct {
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef"`
}

// WorkspaceSpec defines the desired state of Workspace
type WorkspaceSpec struct {
	// Organization name where the Workspace will be created
	// More information: https://www.terraform.io/cloud-docs/users-teams-organizations/organizations
	Organization string `json:"organization"`
	// API Token to be used for API calls
	// More information: https://www.terraform.io/cloud-docs/users-teams-organizations/api-tokens
	Token SecretKeyRef `json:"token"`
	// The display name of the workspace.
	// More information: https://www.terraform.io/cloud-docs/workspaces/settings#name
	//+kubebuilder:validation:Pattern="^[a-zA-Z0-9_-]+$"
	Name string `json:"name"`

	// Define either change will be applied automatically(auto) or require an operator to confirm(manual).
	// More information: https://www.terraform.io/cloud-docs/workspaces/settings#auto-apply-and-manual-apply
	//+kubebuilder:validation:Pattern="^(auto|manual)$"
	//+kubebuilder:default=manual
	//+optional
	ApplyMethod string `json:"applyMethod"`
	// Workspace description
	//+optional
	Description string `json:"description,omitempty"`
	// Define where the Terraform code will be executed.
	// More information: https://www.terraform.io/cloud-docs/workspaces/settings#execution-mode
	//+kubebuilder:validation:Pattern="^(agent|local|remote)$"
	//+kubebuilder:default=remote
	//+optional
	ExecutionMode string `json:"executionMode"`
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
