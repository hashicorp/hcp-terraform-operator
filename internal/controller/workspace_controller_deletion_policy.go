// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	corev1 "k8s.io/api/core/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (r *WorkspaceReconciler) deleteWorkspace(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("deletion policy is %s", w.instance.Spec.DeletionPolicy))

	if w.instance.Status.WorkspaceID == "" {
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("status.WorkspaceID is empty, remove finalizer %s", workspaceFinalizer))
		return r.removeFinalizer(ctx, w)
	}

	switch w.instance.Spec.DeletionPolicy {
	case appv1alpha2.DeletionPolicyRetain:
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("remove finalizer %s", workspaceFinalizer))
		return r.removeFinalizer(ctx, w)
	case appv1alpha2.DeletionPolicySoft:
		err := w.tfClient.Client.Workspaces.SafeDeleteByID(ctx, w.instance.Status.WorkspaceID)
		if err != nil {
			if err == tfc.ErrResourceNotFound {
				w.log.Info("Reconcile Workspace", "msg", "Workspace was not found, remove finalizer")
				return r.removeFinalizer(ctx, w)
			}
			if err == tfc.ErrWorkspaceNotSafeToDelete {
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
			workspace, err := w.tfClient.Client.Workspaces.ReadByID(ctx, w.instance.Status.WorkspaceID)
			if err != nil {
				return r.handleWorkspaceErrorNotFound(ctx, w, err)
			}
			if workspace.CurrentRun == nil {
				w.log.Info("Reconcile Workspace", "msg", "Workspace does not have runs, skipping destroy run, and remove finalizer")
				if err := w.tfClient.Client.Workspaces.DeleteByID(ctx, w.instance.Status.WorkspaceID); err != nil {
					return r.handleWorkspaceErrorNotFound(ctx, w, err)
				}

				w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("workspace ID %s has been deleted, remove finalizer", w.instance.Status.WorkspaceID))
				return r.removeFinalizer(ctx, w)
			}
			w.log.Info("Destroy Run", "msg", "destroy on deletion, create a new destroy run")
			run, err := w.tfClient.Client.Runs.Create(ctx, tfc.RunCreateOptions{
				IsDestroy: tfc.Bool(true),
				Message:   tfc.String(runMessage),
				Workspace: &tfc.Workspace{
					ID: w.instance.Status.WorkspaceID,
				},
			})
			if err != nil {
				w.log.Error(err, "Destroy Run", "msg", "failed to create a new destroy run")
				return err
			}
			w.log.Info("Destroy Run", "msg", fmt.Sprintf("successfully created a new destroy run: %s", run.ID))

			w.instance.Status.DestroyRunID = run.ID
			w.updateWorkspaceStatusRun(run)
			return r.Status().Update(ctx, &w.instance)
		}

		w.log.Info("Destroy Run", "msg", fmt.Sprintf("get destroy run %s", w.instance.Status.DestroyRunID))
		run, err := w.tfClient.Client.Runs.Read(ctx, w.instance.Status.DestroyRunID)
		if err != nil {
			if err == tfc.ErrResourceNotFound {
				w.log.Info("Reconcile Workspace", "msg", "Destroy run was not found, check if the workspace exists")
				if _, err := w.tfClient.Client.Workspaces.ReadByID(ctx, w.instance.Status.WorkspaceID); err != nil {
					return r.handleWorkspaceErrorNotFound(ctx, w, err)
				}
			}
			w.log.Info("Destroy Run", "msg", fmt.Sprintf("failed to get destroy run: %s", w.instance.Status.DestroyRunID))
			return err
		}

		if _, ok := runStatusComplete[run.Status]; ok {
			w.log.Info("Destroy Run", "msg", fmt.Sprintf("current destroy run %s is finished", run.ID))
			if err := w.tfClient.Client.Workspaces.DeleteByID(ctx, w.instance.Status.WorkspaceID); err != nil {
				return r.handleWorkspaceErrorNotFound(ctx, w, err)
			}

			w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("workspace ID %s has been deleted, remove finalizer", w.instance.Status.WorkspaceID))
			return r.removeFinalizer(ctx, w)
		}

		if _, ok := runStatusUnsuccessful[run.Status]; ok {
			w.log.Info("Destroy Run", "msg", fmt.Sprintf("destroy run %s is unsuccessful: %s", run.ID, run.Status))

			workspace, err := w.tfClient.Client.Workspaces.ReadByID(ctx, w.instance.Status.WorkspaceID)
			if err != nil {
				return r.handleWorkspaceErrorNotFound(ctx, w, err)
			}

			w.log.Info("Destroy Run", "msg", fmt.Sprintf("CurrentRun: %s %s %v", workspace.CurrentRun.ID, workspace.CurrentRun.Status, workspace.CurrentRun.IsDestroy))

			if workspace.CurrentRun != nil && workspace.CurrentRun.ID != w.instance.Status.DestroyRunID {

				run, err := w.tfClient.Client.Runs.Read(ctx, w.instance.Status.DestroyRunID)
				if err != nil {
					// ignore this run id, and let the next reconcile loop handle the error
					return nil
				}
				if run.IsDestroy {
					w.log.Info("Destroy Run", "msg", fmt.Sprintf("found more recent destroy run %s, updating DestroyRunID", workspace.CurrentRun.ID))

					w.instance.Status.DestroyRunID = workspace.CurrentRun.ID
					w.updateWorkspaceStatusRun(run)
					return r.Status().Update(ctx, &w.instance)
				}
			}
			if isRetryEnabled(w) {
				w.log.Info("Destroy Run", "msg", fmt.Sprintf("ongoing destroy run %s is unsuccessful, retrying it", run.ID))
				err := r.retryFailedDestroyRun(ctx, w, workspace, run)
				if err != nil {
					w.log.Info("Destroy Run", "msg", fmt.Sprintf("ongoing destroy run %s is unsuccessful, retrying it", run.ID))
					return err
				}
				return r.Status().Update(ctx, &w.instance)
			}

			return nil
		}
		w.log.Info("Destroy Run", "msg", fmt.Sprintf("destroy run %s is not finished", run.ID))

		w.updateWorkspaceStatusRun(run)
		return r.Status().Update(ctx, &w.instance)
	case appv1alpha2.DeletionPolicyForce:
		err := w.tfClient.Client.Workspaces.DeleteByID(ctx, w.instance.Status.WorkspaceID)
		if err != nil {
			if err == tfc.ErrResourceNotFound {
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
	if err == tfc.ErrResourceNotFound {
		w.log.Info("Reconcile Workspace", "msg", "Workspace was not found, remove finalizer")
		return r.removeFinalizer(ctx, w)
	}
	w.log.Error(err, "Reconcile Workspace", "msg", fmt.Sprintf("failed to handle Workspace ID %s, retry later", w.instance.Status.WorkspaceID))
	r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to handle Workspace ID %s, retry later", w.instance.Status.WorkspaceID)
	return err
}

func (w *workspaceInstance) updateWorkspaceStatusRun(run *tfc.Run) {
	if w.instance.Status.Run == nil {
		w.instance.Status.Run = &appv1alpha2.RunStatus{}
	}
	w.instance.Status.Run.ID = run.ID
	w.instance.Status.Run.Status = string(run.Status)
	w.instance.Status.Run.ConfigurationVersion = run.ConfigurationVersion.ID
}
