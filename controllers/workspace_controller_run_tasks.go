// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"

	tfc "github.com/hashicorp/go-tfe"
)

func (r *WorkspaceReconciler) reconcileRunTasks(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Run Tasks", "msg", "new reconciliation event")

	for _, rt := range w.instance.Spec.RunTasks {
		_, err := w.tfClient.Client.WorkspaceRunTasks.Create(ctx, w.instance.Status.WorkspaceID, tfc.WorkspaceRunTaskCreateOptions{
			Type:             rt.Type,
			EnforcementLevel: tfc.TaskEnforcementLevel(rt.EnforcementLevel),
			Stage:            (*tfc.Stage)(&rt.Stage),
			RunTask: &tfc.RunTask{
				ID: rt.ID,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}
