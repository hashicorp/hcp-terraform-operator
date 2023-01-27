// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (ap *AgentPool) NeedToAddFinalizer(finalizer string) bool {
	return ap.ObjectMeta.DeletionTimestamp.IsZero() && !controllerutil.ContainsFinalizer(ap, finalizer)
}

func (ap *AgentPool) IsDeletionCandidate(finalizer string) bool {
	return !ap.ObjectMeta.DeletionTimestamp.IsZero() && controllerutil.ContainsFinalizer(ap, finalizer)
}

func (ap *AgentPool) IsCreationCandidate() bool {
	return ap.Status.AgentPoolID == ""
}
