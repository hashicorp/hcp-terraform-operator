// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-tfe/v2/api/models"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func (r *WorkspaceReconciler) reconcileRuns(ctx context.Context, w *workspaceInstance, workspace models.Workspacesable) error {
	w.log.Info("Reconcile Runs", "msg", "new reconciliation event")

	if runNew, ok := w.instance.Annotations[WorkspaceAnnotationRunNew]; ok && runNew == MetaTrue {
		w.log.Info("Reconcile Runs", "msg", "trigger a new run")

		runType := RunTypeDefault
		if rt, ok := w.instance.Annotations[WorkspaceAnnotationRunType]; ok {
			runType = rt
		}

		options := models.NewRuns_attributes()
		options.SetMessage(new(runMessage))

		switch runType {
		case RunTypePlan:
			options.SetPlanOnly(new(true))
			if t, ok := w.instance.Annotations[WorkspaceAnnotationRunTerraformVersion]; ok {
				options.SetTerraformVersion(new(t))
			}
			return r.triggerPlanRun(ctx, w, pointer.ValueOf(workspace.GetId()), options)
		case RunTypeApply:
			options.SetPlanOnly(new(false))
			return r.triggerApplyRun(ctx, w, pointer.ValueOf(workspace.GetId()), options)
		case RunTypeRefresh:
			options.SetRefreshOnly(new(true))
			return r.triggerApplyRun(ctx, w, pointer.ValueOf(workspace.GetId()), options)
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

func (r *WorkspaceReconciler) reconcileCurrentRun(ctx context.Context, w *workspaceInstance, workspace models.Workspacesable) error {
	var currentRunID string
	if cr := workspace.GetRelationships().GetCurrentRun(); cr != nil && cr.GetData() != nil {
		currentRunID = pointer.ValueOf(cr.GetData().GetId())
	}
	if currentRunID == "" {
		w.log.Info("Reconcile Runs", "msg", "there are no ongoing non-speculative runs")
		return nil
	}

	if w.instance.Status.Run != nil {
		if currentRunID == w.instance.Status.Run.ID && w.instance.Status.Run.RunCompleted() {
			w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("the run %s status is synchronized no actions is required", currentRunID))
			return nil
		}
	}

	w.log.Info("Reconcile Runs", "msg", "get the ongoing non-speculative run status")
	resp, err := w.tfClient.Client.API.Runs().ById(currentRunID).Get(ctx, nil)
	if err != nil {
		w.log.Error(err, "Reconcile Runs", "msg", "failed to get the ongoing non-speculative run status")
		return err
	}
	run := resp.GetData()

	w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("successfully got the ongoing non-speculative run status %s", runAttributesStatus(run.GetAttributes())))

	w.updateWorkspaceStatusRun(run)

	return nil
}

func (r *WorkspaceReconciler) reconcilePlanRun(ctx context.Context, w *workspaceInstance) error {
	if w.instance.Status.Plan == nil {
		w.log.Info("Reconcile Runs", "msg", "there are no ongoing speculative runs")
		return nil
	}

	if !w.instance.Status.Plan.RunCompleted() {
		w.log.Info("Reconcile Runs", "msg", "get the speculative run status")
		resp, err := w.tfClient.Client.API.Runs().ById(w.instance.Status.Plan.ID).Get(ctx, nil)
		if err != nil {
			w.log.Error(err, "Reconcile Runs", "msg", "failed to get the speculative run status")
			return err
		}
		run := resp.GetData()

		w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("successfully got the speculative run status %s", runAttributesStatus(run.GetAttributes())))

		w.instance.Status.Plan.ID = pointer.ValueOf(run.GetId())
		w.instance.Status.Plan.Status = runAttributesStatus(run.GetAttributes())
	}

	return nil
}

func (r *WorkspaceReconciler) triggerApplyRun(ctx context.Context, w *workspaceInstance, workspaceID string, options models.Runs_attributesable) error {
	w.log.Info("Reconcile Runs", "msg", "create a new apply run")

	ws := models.NewWorkspacesHasOne_data()
	ws.SetId(new(workspaceID))
	ws.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIER_TYPE))
	wsRel := models.NewWorkspacesHasOne()
	wsRel.SetData(ws)
	rels := models.NewRuns_relationships()
	rels.SetWorkspace(wsRel)
	data := models.NewRuns()
	data.SetTypeEscaped(new(models.RUNS_RUNS_TYPE))
	data.SetAttributes(options)
	data.SetRelationships(rels)
	body := models.NewRunsEnvelope()
	body.SetData(data)

	resp, err := w.tfClient.Client.API.Runs().Post(ctx, body, nil)
	if err != nil {
		w.log.Error(err, "Reconcile Runs", "msg", "failed to create a apply new run")
		return err
	}
	run := resp.GetData()

	w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("successfully created a new apply run %s", pointer.ValueOf(run.GetId())))

	// Set annotation `workspace.app.terraform.io/run-new` to false if a new run was successfully triggered.
	w.instance.Annotations[WorkspaceAnnotationRunNew] = metaFalse
	err = r.Update(ctx, &w.instance)
	if err != nil {
		w.log.Error(err, "Reconcile Runs", "msg", "failed to update instance")
		return err
	}

	// Update status
	w.updateWorkspaceStatusRun(run)

	return nil
}

func (r *WorkspaceReconciler) triggerPlanRun(ctx context.Context, w *workspaceInstance, workspaceID string, options models.Runs_attributesable) error {
	w.log.Info("Reconcile Runs", "msg", "create a new plan run")

	ws := models.NewWorkspacesHasOne_data()
	ws.SetId(new(workspaceID))
	ws.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIER_TYPE))
	wsRel := models.NewWorkspacesHasOne()
	wsRel.SetData(ws)
	rels := models.NewRuns_relationships()
	rels.SetWorkspace(wsRel)
	data := models.NewRuns()
	data.SetTypeEscaped(new(models.RUNS_RUNS_TYPE))
	data.SetAttributes(options)
	data.SetRelationships(rels)
	body := models.NewRunsEnvelope()
	body.SetData(data)

	resp, err := w.tfClient.Client.API.Runs().Post(ctx, body, nil)
	if err != nil {
		w.log.Error(err, "Reconcile Runs", "msg", "failed to create a new plan run")
		return err
	}
	run := resp.GetData()

	w.log.Info("Reconcile Runs", "msg", fmt.Sprintf("successfully created a new plan run %s", pointer.ValueOf(run.GetId())))

	// Set annotation `workspace.app.terraform.io/run-new` to false if a new run was successfully triggered.
	w.instance.Annotations[WorkspaceAnnotationRunNew] = metaFalse
	err = r.Update(ctx, &w.instance)
	if err != nil {
		w.log.Error(err, "Reconcile Runs", "msg", "failed to update instance")
		return err
	}

	// Update status
	w.instance.Status.Plan = &appv1alpha2.PlanStatus{
		ID:               pointer.ValueOf(run.GetId()),
		Status:           runAttributesStatus(run.GetAttributes()),
		TerraformVersion: pointer.ValueOf(run.GetAttributes().GetTerraformVersion()),
	}

	return nil
}
