// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (r *WorkspaceReconciler) resetRetryStatus(ctx context.Context, w *workspaceInstance) error {
	if !isRetryEnabled(w) {
		return nil
	}
	w.instance.Status.Retry = &appv1alpha2.RetryStatus{
		Failed: 0,
	}
	return nil
}

func isRetryEnabled(w *workspaceInstance) bool {
	return w.instance.Spec.RetryPolicy != nil && w.instance.Spec.RetryPolicy.BackoffLimit != 0
}

func (r *WorkspaceReconciler) retryFailedApplyRun(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace, failedRun *tfc.Run) error {
	retriedRun, err := r.retryFailedRun(ctx, w, workspace, failedRun)
	if err != nil {
		return err
	}

	// when no run is returned, it means the backoff limit was reached
	if retriedRun == nil {
		return nil
	}

	w.updateWorkspaceStatusRun(retriedRun)
	// WARNING: there is a race limit here in case the run fails very fast and the initial status returned
	// by the Runs.Create funtion is Errored. In this case the run is never retried.
	// TODO: loop back ? I don't like loops so maybe the best would be to change the reconcile runs function to
	// make sure we didn't miss a retry

	return nil
}

func (r *WorkspaceReconciler) retryFailedDestroyRun(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace, failedRun *tfc.Run) error {
	retriedRun, err := r.retryFailedRun(ctx, w, workspace, failedRun)
	if err != nil {
		return err
	}

	// when no run is returned, it means the backoff limit was reached
	if retriedRun == nil {
		return nil
	}

	w.instance.Status.DestroyRunID = retriedRun.ID
	w.updateWorkspaceStatusRun(retriedRun)

	return nil
}

func (r *WorkspaceReconciler) retryFailedRun(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace, failedRun *tfc.Run) (*tfc.Run, error) {
	if w.instance.Status.Retry == nil {
		w.instance.Status.Retry = &appv1alpha2.RetryStatus{
			Failed: 0,
		}
	}
	w.instance.Status.Retry.Failed++

	if w.instance.Spec.RetryPolicy.BackoffLimit < 0 || w.instance.Status.Retry.Failed <= w.instance.Spec.RetryPolicy.BackoffLimit {

		options := tfc.RunCreateOptions{
			Message:     tfc.String(runMessage),
			Workspace:   workspace,
			IsDestroy:   tfc.Bool(failedRun.IsDestroy),
			RefreshOnly: tfc.Bool(failedRun.RefreshOnly),
		}
		retriedRun, err := w.tfClient.Client.Runs.Create(ctx, options)
		if err != nil {
			w.log.Error(err, "Retry Runs", "msg", "failed to create a new apply run for retry")
			return nil, err
		}
		w.log.Info("Retry Runs", "msg", fmt.Sprintf("successfully created a new apply run %s for to retry failed %s", retriedRun.ID, failedRun.ID))

		return retriedRun, nil
	} else {
		w.log.Info("Retry Runs", "msg", "backoff limit was reached, skip retry")
		return nil, nil
	}
}
