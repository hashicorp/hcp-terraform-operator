// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (r *WorkspaceReconciler) reconcileRuns(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	w.log.Info("Reconcile Runs", "msg", "new reconciliation event")

	if runNew, ok := w.instance.Annotations[workspaceAnnotationRunNew]; ok && runNew == annotationTrue {
		w.log.Info("Reconcile Runs", "msg", "trigger a new run")
		runType := runTypeDefault
		if rt, ok := w.instance.Annotations[workspaceAnnotationRunType]; ok {
			runType = rt
		}
		options := tfc.RunCreateOptions{
			Message:   tfc.String(runMessage),
			Workspace: workspace,
		}

		switch runType {
		case runTypePlan:
			options.PlanOnly = tfc.Bool(true)
			if t, ok := w.instance.Annotations[workspaceAnnotationRunTerraformVersion]; ok {
				options.TerraformVersion = tfc.String(t)
			}
			return r.triggerPlanRun(ctx, w, options)
		case runTypeApply:
			options.PlanOnly = tfc.Bool(false)
			return r.triggerApplyRun(ctx, w, options)
		case runTypeRefresh:
			options.RefreshOnly = tfc.Bool(true)
			return r.triggerApplyRun(ctx, w, options)
		default:
			// Throw an error message here but don't return.
			w.log.Error(fmt.Errorf("run type %q is not valid", runType), "Reconcile Runs", "msg", "no new run will be triggered")
		}
	}

	if err := r.reconcileCurrentRun(ctx, w, workspace); err != nil {
		return err
	}

	if err := r.reconcilePlanRun(ctx, w); err != nil {
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

	if isRetryEnabled(w) {
		if _, ok := runStatusUnsuccessful[run.Status]; ok {
			w.log.Info("Reconcile Runs", "msg", "ongoing non-speculative run is unsuccessful, retrying it")

			if err = r.retryFailedApplyRun(ctx, w, workspace, run); err != nil {
				return err
			}
		}
	}

	// when the current run succeeds, we reset the failed counter for a next retry
	if _, ok := runStatusComplete[run.Status]; ok {
		r.resetRetryStatus(ctx, w)
	}

	return nil
}

func (r *WorkspaceReconciler) reconcilePlanRun(ctx context.Context, w *workspaceInstance) error {
	if w.instance.Status.Plan == nil {
		w.log.Info("Reconcile Runs", "msg", "there are no ongoing speculative runs")
		return nil
	}

	if !w.instance.Status.Plan.RunCompleted() {
		w.log.Info("Reconcile Runs", "msg", "get the speculative run status")
		run, err := w.tfClient.Client.Runs.Read(ctx, w.instance.Status.Plan.ID)
		if err != nil {
			w.log.Error(err, "Reconcile Runs", "msg", "failed to get the speculative run status")
			return err
		}
		w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("successfully got the speculative run status %s", run.Status))
		w.instance.Status.Plan.ID = run.ID
		w.instance.Status.Plan.Status = string(run.Status)
	}

	return nil
}

func (r *WorkspaceReconciler) triggerApplyRun(ctx context.Context, w *workspaceInstance, options tfc.RunCreateOptions) error {
	w.log.Info("Reconcile Runs", "msg", "create a new apply run")

	run, err := w.tfClient.Client.Runs.Create(ctx, options)
	if err != nil {
		w.log.Error(err, "Reconcile Runs", "msg", "failed to create a apply new run")
		return err
	}
	w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("successfully created a new apply run %s", run.ID))

	// Set annotation `workspace.app.terraform.io/run-new` to false if a new run was successfully triggered.
	w.instance.Annotations[workspaceAnnotationRunNew] = annotationFalse
	err = r.Update(ctx, &w.instance)
	if err != nil {
		w.log.Error(err, "Reconcile Runs", "msg", "failed to update instance")
		return err
	}

	// Update status
	if w.instance.Status.Run == nil {
		w.instance.Status.Run = &appv1alpha2.RunStatus{}
	}
	w.instance.Status.Run.ID = run.ID
	w.instance.Status.Run.Status = string(run.Status)
	w.instance.Status.Run.ConfigurationVersion = run.ConfigurationVersion.ID

	// WARNING: there is a race limit here in case the run fails very fast and the initial status returned
	// by the Runs.Create funtion is Errored. In this case the run is never retried.
	// TODO: loop back ? I don't like loops so maybe the best would be to change the reconcile runs function to
	// make sure we didn't miss a retry

	return nil
}

func (r *WorkspaceReconciler) triggerPlanRun(ctx context.Context, w *workspaceInstance, options tfc.RunCreateOptions) error {
	w.log.Info("Reconcile Runs", "msg", "create a new plan run")
	run, err := w.tfClient.Client.Runs.Create(ctx, options)
	if err != nil {
		w.log.Error(err, "Reconcile Runs", "msg", "failed to create a new plan run")
		return err
	}
	w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("successfully created a new plan run %s", run.ID))

	// Set annotation `workspace.app.terraform.io/run-new` to false if a new run was successfully triggered.
	w.instance.Annotations[workspaceAnnotationRunNew] = annotationFalse
	err = r.Update(ctx, &w.instance)
	if err != nil {
		w.log.Error(err, "Reconcile Runs", "msg", "failed to update instance")
		return err
	}

	// Update status
	w.instance.Status.Plan = &appv1alpha2.PlanStatus{
		ID:               run.ID,
		Status:           string(run.Status),
		TerraformVersion: run.TerraformVersion,
	}

	return nil
}
