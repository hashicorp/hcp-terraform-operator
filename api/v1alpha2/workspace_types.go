package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretKeySelector refers to an value in a Kubernetes Secret object
type SecretKeySelector struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// SecretKeyRef refers to a Kubernetes Secret object
type SecretKeyRef struct {
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef"`
}

// WorkspaceSpec defines the desired state of Workspace
type WorkspaceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Organization name where the workspace will be created
	Organization string `json:"organization"`
	// API Token to be used for API calls
	Token SecretKeyRef `json:"token"`
	// Workspace name
	Name string `json:"name"`
}

// WorkspaceStatus defines the observed state of Workspace
type WorkspaceStatus struct {
	// Workspace ID that is managed by the controller
	WorkspaceID string `json:"workspaceID"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Workspace is the Schema for the workspaces API
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceSpec   `json:"spec,omitempty"`
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
