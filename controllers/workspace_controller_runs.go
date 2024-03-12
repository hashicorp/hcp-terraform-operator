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

	if runAt, ok := w.instance.Annotations[workspaceAnnotationRunAt]; ok {
		runType := runTypeDefault
		if rt, ok := w.instance.Annotations[workspaceAnnotationRunType]; ok {
			runType = rt
		}

		switch runType {
		case runTypePlan:
			if w.instance.Status.Plan == nil || runAt != w.instance.Status.Plan.TriggeredAt {
				return r.triggerRun(ctx, w, workspace)
			}
		default:
			if w.instance.Status.Run == nil || runAt != w.instance.Status.Run.TriggeredAt {
				return r.triggerRun(ctx, w, workspace)
			}
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
	if w.instance.Status.Run == nil || w.instance.Status.Plan == nil {
		w.log.Info("Reconcile Runs", "msg", "there are no ongoing speculative runs")
		return nil
	}

	// Update speculative run status
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

func (r *WorkspaceReconciler) triggerRun(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	runType := runTypeDefault
	if rt, ok := w.instance.Annotations[workspaceAnnotationRunType]; ok {
		runType = rt
	}

	options := tfc.RunCreateOptions{
		Message:   tfc.String("Triggered by the Kubernetes Operator"),
		Workspace: workspace,
	}

	switch runType {
	case runTypePlan:
		options.PlanOnly = tfc.Bool(true)
		if t, ok := w.instance.Annotations[workspaceAnnotationRunTerraformVersion]; ok {
			options.TerraformVersion = tfc.String(t)
		}
	case runTypeApply:
		options.PlanOnly = tfc.Bool(false)
	case runTypeRefresh:
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

	if runType == runTypePlan {
		w.instance.Status.Plan = &appv1alpha2.PlanStatus{
			ID:               run.ID,
			Status:           string(run.Status),
			TriggeredAt:      w.instance.Annotations[workspaceAnnotationRunAt],
			TerraformVersion: run.TerraformVersion,
		}

		return nil
	}

	// If a new workspace has been created and a new run is triggered via annotations,
	// then the Workspace doesn't have a current Run reference.
	if workspace.CurrentRun == nil {
		workspace.CurrentRun = &tfc.Run{}
	}
	w.instance.Status.Run.ID = run.ID
	w.instance.Status.Run.Status = string(run.Status)
	w.instance.Status.Run.ConfigurationVersion = run.ConfigurationVersion.ID
	w.instance.Status.Run.TriggeredAt = w.instance.Annotations[workspaceAnnotationRunAt]

	return nil
}
