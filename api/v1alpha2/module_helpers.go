// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
)

func (m *Module) GetNamespace() string {
	return m.ObjectMeta.Namespace
}

func (m *Module) GetToken() corev1.SecretKeySelector {
	return *m.Spec.Token.SecretKeyRef
}
