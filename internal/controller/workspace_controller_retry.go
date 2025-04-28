// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (r *WorkspaceReconciler) resetRetryStatus(ctx context.Context, w *workspaceInstance) error {
	if w.instance.Spec.RetryPolicy == nil || w.instance.Spec.RetryPolicy.BackoffLimit == 0 {
		return nil
	}
	w.instance.Status.Retry = &appv1alpha2.RetryStatus{
		Failed: 0,
	}
	return nil
}
