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
	Organization string `json:"organization"`
	// API Token to be used for API calls
	Token SecretKeyRef `json:"token"`
	// Workspace name
	Name string `json:"name"`
}

// WorkspaceStatus defines the observed state of Workspace
type WorkspaceStatus struct {
	// Real world state generation
	ObservedGeneration int64 `json:"observedGeneration"`
	// Workspace ID that is managed by the controller
	WorkspaceID string `json:"workspaceID"`
	// Workspace last update timestamp
	UpdateAt int64 `json:"updateAt"`
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
