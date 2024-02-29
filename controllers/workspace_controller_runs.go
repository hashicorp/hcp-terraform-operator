// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

func (r *WorkspaceReconciler) reconcileRuns(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	w.log.Info("Reconcile Runs", "msg", "new reconciliation event")

	if workspace.CurrentRun == nil {
		w.log.Info("Reconcile Runs", "msg", "there is no current run")
		return nil
	}

	if w.instance.Status.Run == nil {
		w.instance.Status.Run = &appv1alpha2.RunStatus{}
	}

	// Update current run status
	if workspace.CurrentRun.ID != w.instance.Status.Run.ID || !w.instance.Status.Run.RunCompleted() {
		w.log.Info("Reconcile Runs", "msg", "get the current run status")
		run, err := w.tfClient.Client.Runs.Read(ctx, workspace.CurrentRun.ID)
		if err != nil {
			w.log.Error(err, "Reconcile Runs", "msg", "failed to get the current run status")
			return err
		}
		w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("successfully got the current run status %s", run.Status))
		// Update status
		w.instance.Status.Run.ID = run.ID
		w.instance.Status.Run.Status = string(run.Status)
		w.instance.Status.Run.ConfigurationVersion = run.ConfigurationVersion.ID
	}

	return nil
}
