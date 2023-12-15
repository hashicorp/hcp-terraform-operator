// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (p *Project) NeedToAddFinalizer(finalizer string) bool {
	return p.ObjectMeta.DeletionTimestamp.IsZero() && !controllerutil.ContainsFinalizer(p, finalizer)
}

func (p *Project) IsDeletionCandidate(finalizer string) bool {
	return !p.ObjectMeta.DeletionTimestamp.IsZero() && controllerutil.ContainsFinalizer(p, finalizer)
}

func (p *Project) IsCreationCandidate() bool {
	return p.Status.ID == ""
}
