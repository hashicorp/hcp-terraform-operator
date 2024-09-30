// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

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
		// TODO: Implement the destroy logic
		return nil
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
