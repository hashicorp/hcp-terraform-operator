// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (r *ModuleReconciler) deleteModule(ctx context.Context, m *moduleInstance) error {
	m.log.Info("Reconcile Module", "msg", fmt.Sprintf("deletion policy is %s", m.instance.Spec.DeletionPolicy))

	if m.instance.Status.WorkspaceID == "" {
		m.log.Info("Reconcile Module", "msg", fmt.Sprintf("status.WorkspaceID is empty, remove finalizer %s", moduleFinalizer))
		return r.removeFinalizer(ctx, m)
	}

	switch m.instance.Spec.DeletionPolicy {
	case appv1alpha2.ModuleDeletionPolicyRetain:
		// If the deletion policy is Retain, remove the finalizer without triggering a destroy operation
		// destroyOnDeletion must be set to false
		m.log.Info("Reconcile Module", "msg", fmt.Sprintf("remove finalizer %s", moduleFinalizer))

		if m.instance.Spec.DestroyOnDeletion {
			m.log.Info("Reconcile Module", "msg", "destroyOnDeletion is true, checking for active destroy run")

			// If no destroy run exists, check for any ongoing destroy process
			if m.instance.Status.DestroyRunID == "" {
				ws, err := m.tfClient.Client.Workspaces.ReadByID(ctx, m.instance.Status.WorkspaceID)
				if err != nil {
					m.log.Error(err, "Reconcile Module", "msg", fmt.Sprintf("failed to get workspace: %s", m.instance.Status.WorkspaceID))
					return err
				}
				if ws.CurrentRun != nil {
					m.log.Info("Reconcile Module", "msg", "ongoing destroy run found, checking status")
					cr, err := m.tfClient.Client.Runs.Read(ctx, ws.CurrentRun.ID)
					if err != nil {
						m.log.Error(err, "Reconcile Module", "msg", fmt.Sprintf("failed to get current destroy run: %s", ws.CurrentRun.ID))
						return err
					}
					if cr.IsDestroy {
						if _, ok := runStatusComplete[cr.Status]; ok {
							m.log.Info("Reconcile Module", "msg", "current destroy run finished, removing finalizer")
							return r.removeFinalizer(ctx, m)
						}
						m.log.Info("Reconcile Module", "msg", fmt.Sprintf("destroy run %s still in progress", cr.ID))
						return r.updateStatusDestroy(ctx, &m.instance, cr)
					}
				}
				m.log.Info("Reconcile Module", "msg", "no active destroy run, creating a new destroy run")
				run, err := m.tfClient.Client.Runs.Create(ctx, tfc.RunCreateOptions{
					IsDestroy: tfc.Bool(true),
					Message:   tfc.String(runMessage),
					Workspace: &tfc.Workspace{
						ID: m.instance.Status.WorkspaceID,
					},
				})
				if err != nil {
					m.log.Error(err, "Reconcile Module", "msg", "failed to create a new destroy run")
					return err
				}
				return r.updateStatusDestroy(ctx, &m.instance, run)
			} else {
				m.log.Info("Reconcile Module", "msg", "destroy run already in progress, waiting for completion")
				run, err := m.tfClient.Client.Runs.Read(ctx, m.instance.Status.DestroyRunID)
				if err != nil {
					m.log.Error(err, "Reconcile Module", "msg", "failed to get destroy run status")
					return err
				}
				if _, ok := runStatusComplete[run.Status]; ok {
					m.log.Info("Reconcile Module", "msg", "destroy run finished, removing finalizer")
					return r.removeFinalizer(ctx, m)
				}
				return r.updateStatusDestroy(ctx, &m.instance, run)
			}
		} else {
			// This handles the default cases where
			// deletionPolicy : retain
			// destroyOnDeletion: false
			m.log.Info("Reconcile Module", "msg", "destroyOnDeletion is false, removing finalizer without destroying resources")
			return r.removeFinalizer(ctx, m)
		}

	case appv1alpha2.ModuleDeletionPolicyDestroy:
		// Destroy policy deletes the module and any resources
		if m.instance.Spec.DestroyOnDeletion {
			m.log.Info("Reconcile Module", "msg", "destroyOnDeletion is true, destroying associated resources")
			err := m.tfClient.Client.Workspaces.DeleteByID(ctx, m.instance.Status.WorkspaceID)
			if err != nil {
				m.log.Error(err, "Reconcile Module", "msg", "Failed to destroy associated resources")
				return err
			}
		}
		m.log.Info("Reconcile Module", "msg", "deleting module and removing finalizer")
		return r.removeFinalizer(ctx, m)
	}

	return nil
}
