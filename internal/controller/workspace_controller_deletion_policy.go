// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tfc "github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
	corev1 "k8s.io/api/core/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func (r *WorkspaceReconciler) deleteWorkspace(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("deletion policy is %s", w.instance.Spec.DeletionPolicy))

	if w.instance.Status.WorkspaceID == "" {
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("status.WorkspaceID is empty, remove finalizer %s", workspaceFinalizer))
		return r.removeFinalizer(ctx, w)
	}

	workspaceRequest := w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID)

	switch w.instance.Spec.DeletionPolicy {
	case appv1alpha2.DeletionPolicyRetain:
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("remove finalizer %s", workspaceFinalizer))
		return r.removeFinalizer(ctx, w)
	case appv1alpha2.DeletionPolicySoft:
		err := workspaceRequest.Actions().SafeDelete().Post(ctx, nil)
		if err != nil {
			if errors.Is(err, tfc.ErrNotFound) {
				w.log.Info("Reconcile Workspace", "msg", "Workspace was not found, remove finalizer")
				return r.removeFinalizer(ctx, w)
			}
			// Locked and still processing workspaces are reported as different errors.
			var apiErr *tfc.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == 409 && !strings.Contains(apiErr.Error(), "locked") && !strings.Contains(apiErr.Error(), "being processed") {
				w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("Workspace ID %s is still managing resources, retry later", w.instance.Status.WorkspaceID))
				return nil
			}
			w.log.Error(err, "Reconcile Workspace", "msg", fmt.Sprintf("failed to soft delete Workspace ID %s, retry later", w.instance.Status.WorkspaceID))
			r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to safe delete Workspace ID %s, retry later", w.instance.Status.WorkspaceID)
			return err
		}
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("workspace ID %s has been deleted, remove finalizer", w.instance.Status.WorkspaceID))
		return r.removeFinalizer(ctx, w)
	case appv1alpha2.DeletionPolicyDestroy:
		if w.instance.Status.DestroyRunID == "" {
			workspace, err := r.readWorkspace(ctx, w)
			if err != nil {
				return r.handleWorkspaceErrorNotFound(ctx, w, err)
			}

			if cr := workspace.GetRelationships().GetCurrentRun(); cr == nil || cr.GetData() == nil || pointer.ValueOf(cr.GetData().GetId()) == "" {
				w.log.Info("Reconcile Workspace", "msg", "Workspace does not have runs, skipping destroy run, and remove finalizer")
				if err := workspaceRequest.Delete(ctx, nil); err != nil {
					return r.handleWorkspaceErrorNotFound(ctx, w, err)
				}
				w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("workspace ID %s has been deleted, remove finalizer", w.instance.Status.WorkspaceID))
				return r.removeFinalizer(ctx, w)
			}

			w.log.Info("Destroy Run", "msg", "destroy on deletion, create a new destroy run")
			attrs := models.NewRuns_attributes()
			attrs.SetIsDestroy(new(true))
			attrs.SetMessage(new(runMessage))
			rels := models.NewRuns_relationships()
			ws := models.NewWorkspacesHasOne_data()
			ws.SetId(new(w.instance.Status.WorkspaceID))
			ws.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIER_TYPE))
			wsRel := models.NewWorkspacesHasOne()
			wsRel.SetData(ws)
			rels.SetWorkspace(wsRel)
			data := models.NewRuns()
			data.SetTypeEscaped(new(models.RUNS_RUNS_TYPE))
			data.SetAttributes(attrs)
			data.SetRelationships(rels)
			body := models.NewRunsEnvelope()
			body.SetData(data)

			resp, err := w.tfClient.Client.API.Runs().Post(ctx, body, nil)
			if err != nil {
				w.log.Error(err, "Destroy Run", "msg", "failed to create a new destroy run")
				return err
			}
			run := resp.GetData()

			w.log.Info("Destroy Run", "msg", fmt.Sprintf("successfully created a new destroy run: %s", pointer.ValueOf(run.GetId())))
			w.instance.Status.DestroyRunID = pointer.ValueOf(run.GetId())
			w.updateWorkspaceStatusRun(run)
			return r.Status().Update(ctx, &w.instance)
		}

		w.log.Info("Destroy Run", "msg", fmt.Sprintf("get destroy run %s", w.instance.Status.DestroyRunID))
		runResp, err := w.tfClient.Client.API.Runs().ById(w.instance.Status.DestroyRunID).Get(ctx, nil)
		if err != nil {
			if errors.Is(err, tfc.ErrNotFound) {
				w.log.Info("Reconcile Workspace", "msg", "Destroy run was not found, check if the workspace exists")
				if _, err := r.readWorkspace(ctx, w); err != nil {
					return r.handleWorkspaceErrorNotFound(ctx, w, err)
				}
			}
			w.log.Info("Destroy Run", "msg", fmt.Sprintf("failed to get destroy run: %s", w.instance.Status.DestroyRunID))
			return err
		}
		run := runResp.GetData()
		runStatus := runAttributesStatus(run.GetAttributes())

		if _, ok := runStatusComplete[runStatus]; ok {
			w.log.Info("Destroy Run", "msg", fmt.Sprintf("current destroy run %s is finished", pointer.ValueOf(run.GetId())))
			if err := workspaceRequest.Delete(ctx, nil); err != nil {
				return r.handleWorkspaceErrorNotFound(ctx, w, err)
			}
			w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("workspace ID %s has been deleted, remove finalizer", w.instance.Status.WorkspaceID))
			return r.removeFinalizer(ctx, w)
		}

		if _, ok := runStatusUnsuccessful[runStatus]; ok {
			w.log.Info("Destroy Run", "msg", fmt.Sprintf("destroy run %s is unsuccessful: %s", pointer.ValueOf(run.GetId()), runStatus))
			return nil
		}

		w.log.Info("Destroy Run", "msg", fmt.Sprintf("destroy run %s is not finished", pointer.ValueOf(run.GetId())))
		w.updateWorkspaceStatusRun(run)
		return r.Status().Update(ctx, &w.instance)
	case appv1alpha2.DeletionPolicyForce:
		err := workspaceRequest.Delete(ctx, nil)
		if err != nil {
			if errors.Is(err, tfc.ErrNotFound) {
				w.log.Info("Reconcile Workspace", "msg", "Workspace was not found, remove finalizer")
				return r.removeFinalizer(ctx, w)
			}
			w.log.Error(err, "Reconcile Workspace", "msg", fmt.Sprintf("failed to force delete Workspace ID %s, retry later", w.instance.Status.WorkspaceID))
			r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to force delete Workspace ID %s, retry later", w.instance.Status.WorkspaceID)
			return err
		}
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("workspace ID %s has been deleted, remove finalizer", w.instance.Status.WorkspaceID))
		return r.removeFinalizer(ctx, w)
	}

	return nil
}

func (r *WorkspaceReconciler) handleWorkspaceErrorNotFound(ctx context.Context, w *workspaceInstance, err error) error {
	if errors.Is(err, tfc.ErrNotFound) {
		w.log.Info("Reconcile Workspace", "msg", "Workspace was not found, remove finalizer")
		return r.removeFinalizer(ctx, w)
	}

	w.log.Error(err, "Reconcile Workspace", "msg", fmt.Sprintf("failed to handle Workspace ID %s, retry later", w.instance.Status.WorkspaceID))
	r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to handle Workspace ID %s, retry later", w.instance.Status.WorkspaceID)
	return err
}

func (w *workspaceInstance) updateWorkspaceStatusRun(run models.Runsable) {
	if w.instance.Status.Run == nil {
		w.instance.Status.Run = &appv1alpha2.RunStatus{}
	}

	w.instance.Status.Run.ID = pointer.ValueOf(run.GetId())
	w.instance.Status.Run.Status = runAttributesStatus(run.GetAttributes())
	w.instance.Status.Run.ConfigurationVersion = ""
	if run.GetRelationships() != nil && run.GetRelationships().GetConfigurationVersion() != nil && run.GetRelationships().GetConfigurationVersion().GetData() != nil {
		cv := run.GetRelationships().GetConfigurationVersion()
		w.instance.Status.Run.ConfigurationVersion = pointer.ValueOf(cv.GetData().GetId())
	}
}
