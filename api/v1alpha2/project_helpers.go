// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import corev1 "k8s.io/api/core/v1"

func (p *Project) GetNamespace() string {
	return p.ObjectMeta.Namespace
}

func (p *Project) GetToken() corev1.SecretKeySelector {
	return *p.Spec.Token.SecretKeyRef
}

func (p *Project) IsCreationCandidate() bool {
	return p.Status.ID == ""
}
