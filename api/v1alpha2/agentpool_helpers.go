// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import corev1 "k8s.io/api/core/v1"

func (ap *AgentPool) GetNamespace() string {
	return ap.ObjectMeta.Namespace
}

func (ap *AgentPool) GetToken() corev1.SecretKeySelector {
	return *ap.Spec.Token.SecretKeyRef
}

func (ap *AgentPool) IsCreationCandidate() bool {
	return ap.Status.AgentPoolID == ""
}
