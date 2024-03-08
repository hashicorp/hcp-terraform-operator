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

	if runAt, ok := w.instance.Annotations[workspaceAnnotationRunAt]; ok && runAt != w.instance.Annotations[workspaceAnnotationRestartedAt] {
		w.log.Info("Reconcile Runs", "msg", "NEW RUN")

		options := tfc.RunCreateOptions{
			Message:   tfc.String("Triggered by the Kubernetes Operator"),
			Workspace: workspace,
		}

		runType := runTypeDefault
		if rt, ok := w.instance.Annotations[workspaceAnnotationRunType]; ok {
			runType = rt
		}

		switch runType {
		case runTypePlanOnly:
			options.PlanOnly = tfc.Bool(true)
			// TODO:
			// - Handle Terraform version, annotation: `workspace.app.terraform.io/run-terraform-version`
		case runTypePlanAndApply:
			options.PlanOnly = tfc.Bool(false)
		case runTypeRefreshState:
			options.RefreshOnly = tfc.Bool(true)
		default:
			return fmt.Errorf("run type %q is not valid", runType)
		}

		w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("create a new run type %s", runType))
		run, err := w.tfClient.Client.Runs.Create(ctx, options)
		if err != nil {
			w.log.Error(err, "Reconcile Runs", "msg", "failed to create a new run")
			return err
		}
		w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("successfully created a new run %s type %s", run.ID, runType))

		// Update annotations
		w.instance.Annotations[workspaceAnnotationRestartedAt] = w.instance.Annotations[workspaceAnnotationRunAt]
		err = r.Update(ctx, &w.instance)
		if err != nil {
			w.log.Error(err, "Reconcile Runs", "msg", "failed to update instance")
			return err
		}

		if runType == runTypePlanOnly {
			if w.instance.Status.Run == nil {
				w.instance.Status.Run = &appv1alpha2.RunStatus{
					PlanOnly: &appv1alpha2.RunPlanOnlyStatus{},
				}
			}

			if w.instance.Status.Run.PlanOnly == nil {
				w.instance.Status.Run.PlanOnly = &appv1alpha2.RunPlanOnlyStatus{}
			}

			w.instance.Status.Run.PlanOnly.ID = run.ID
			w.instance.Status.Run.PlanOnly.Status = string(run.Status)
		} else {
			// workspace.CurrentRun is a relation connection and contains only RunID.
			// Update it here to avoid an unnecessary read workspace API call.
			workspace.CurrentRun.ID = run.ID
		}
	}

	if err := r.reconcileCurrentRun(ctx, w, workspace); err != nil {
		return err
	}

	if err := r.reconcileSpeculativeRun(ctx, w); err != nil {
		return err
	}

	return nil
}

func (r *WorkspaceReconciler) reconcileCurrentRun(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	if workspace.CurrentRun == nil {
		w.log.Info("Reconcile Runs", "msg", "there are no ongoing non-speculative runs")
		return nil
	}

	if w.instance.Status.Run != nil {
		if workspace.CurrentRun.ID == w.instance.Status.Run.ID && w.instance.Status.Run.RunCompleted() {
			w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("the run %s status is synchronized no actions is required", workspace.CurrentRun.ID))
			return nil
		}
	}

	w.log.Info("Reconcile Runs", "msg", "get the ongoing non-speculative run status")
	run, err := w.tfClient.Client.Runs.Read(ctx, workspace.CurrentRun.ID)
	if err != nil {
		w.log.Error(err, "Reconcile Runs", "msg", "failed to get the ongoing non-speculative run status")
		return err
	}
	w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("successfully got the ongoing non-speculative run status %s", run.Status))

	if w.instance.Status.Run == nil {
		w.instance.Status.Run = &appv1alpha2.RunStatus{}
	}

	w.instance.Status.Run.ID = run.ID
	w.instance.Status.Run.Status = string(run.Status)
	w.instance.Status.Run.ConfigurationVersion = run.ConfigurationVersion.ID

	return nil
}

func (r *WorkspaceReconciler) reconcileSpeculativeRun(ctx context.Context, w *workspaceInstance) error {
	if w.instance.Status.Run == nil || w.instance.Status.Run.PlanOnly == nil {
		w.log.Info("Reconcile Runs", "msg", "there are no ongoing speculative runs")
		return nil
	}

	// Update speculative run status
	if !w.instance.Status.Run.PlanOnly.RunCompleted() {
		w.log.Info("Reconcile Runs", "msg", "get the speculative run status")
		run, err := w.tfClient.Client.Runs.Read(ctx, w.instance.Status.Run.PlanOnly.ID)
		if err != nil {
			w.log.Error(err, "Reconcile Runs", "msg", "failed to get the speculative run status")
			return err
		}
		w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("successfully got the speculative run status %s", run.Status))
		// Update status
		w.instance.Status.Run.PlanOnly.ID = run.ID
		w.instance.Status.Run.PlanOnly.Status = string(run.Status)
	}

	return nil
}
