// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import corev1 "k8s.io/api/core/v1"

type Instance interface {
	GetNamespace() string
	GetToken() corev1.SecretKeySelector
}
