// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (r *ModuleReconciler) deleteModule(ctx context.Context, m *moduleInstance) error {
	// Due to historical reasons, there are two fields intended to control the same behavior:
	// 'destroyOnDeletion' and 'deletionPolicy'. While both serve the same purpose, 'deletionPolicy'
	// aligns with the approach the operator uses across all other controllers.
	//
	// To support both fields during the transition period, the following logic is applied:
	// | destroyOnDeletion | deletionPolicy   | action                    |
	// |-------------------+------------------+---------------------------|
	// | false (default)   | retain (default) | remove CR only            |
	// | true              | retain (default) | remove CR and run destroy |
	// | false (default)   | destroy          | remove CR and run destroy |
	// | true              | destroy          | remove CR and run destroy |
	//
	// The rationale behind this logic is: we run destroy only when either or both fields explicitly require it.
	// If both fields use their default values, we only remove the CR.
	//
	// Note: 'destroyOnDeletion' is marked as deprecated and will be removed in the future.
	if m.instance.Spec.DeletionPolicy != "" {
		m.log.Info("Reconcile Module", "msg", fmt.Sprintf("deletion policy is %s", m.instance.Spec.DeletionPolicy))
	}

	if !m.instance.Spec.DestroyOnDeletion {
		switch m.instance.Spec.DeletionPolicy {
		case appv1alpha2.ModuleDeletionPolicyRetain:
			m.log.Info("Delete Module", "msg", "no need to run destroy run, deleting object")
			return r.removeFinalizer(ctx, m)
		case appv1alpha2.ModuleDeletionPolicyDestroy:
			return r.destroyModule(ctx, m)
		}
		m.log.Info("Delete Module", "msg", "no need to run destroy run, deleting object")
		return r.removeFinalizer(ctx, m)
	}

	return r.destroyModule(ctx, m)
}

func (r *ModuleReconciler) destroyModule(ctx context.Context, m *moduleInstance) error {
	// check whether a Run was ever running, if no then there is nothing to delete,
	// so delete the Kubernetes object without running the 'Destroy' run
	if m.instance.Status.Run == nil {
		m.log.Info("Delete Module", "msg", "run is empty, removing finalizer")
		return r.removeFinalizer(ctx, m)
	}

	// if 'status.destroyRunID' is empty we first check if there is another ongoing 'Destroy' run and if so,
	// update the status with the run status. Otherwise, execute a new 'Destroy' run.
	if m.instance.Status.DestroyRunID == "" {
		m.log.Info("Delete Module", "msg", "get workspace")
		ws, err := m.tfClient.Client.Workspaces.ReadByID(ctx, m.instance.Status.WorkspaceID)
		if err != nil {
			m.log.Info("Delete Module", "msg", fmt.Sprintf("failed to get workspace: %s", m.instance.Status.WorkspaceID))
			return err
		}
		m.log.Info("Delete Module", "msg", "successfully got workspace")
		if ws.CurrentRun != nil {
			m.log.Info("Delete Module", "msg", "get current run")
			// Have to read the individual run here, since the one associated with workspace doesn't contain the necessary info
			cr, err := m.tfClient.Client.Runs.Read(ctx, ws.CurrentRun.ID)
			if err != nil {
				m.log.Info("Delete Module", "msg", fmt.Sprintf("failed to get current run: %s", ws.CurrentRun.ID))
				return err
			}
			if cr.IsDestroy {
				m.log.Info("Delete Module", "msg", fmt.Sprintf("current run %s is destroy", cr.ID))
				if _, ok := runStatusComplete[cr.Status]; ok {
					m.log.Info("Delete Module", "msg", "current destroy run finished")
					return r.removeFinalizer(ctx, m)
				}
				return r.updateStatusDestroy(ctx, &m.instance, cr)
			}
			m.log.Info("Delete Module", "msg", "current run is not destroy")
		}

		m.log.Info("Delete Module", "msg", "destroy on deletion, create a new destroy run")
		run, err := m.tfClient.Client.Runs.Create(ctx, tfc.RunCreateOptions{
			IsDestroy: tfc.Bool(true),
			Message:   tfc.String(runMessage),
			Workspace: &tfc.Workspace{
				ID: m.instance.Status.WorkspaceID,
			},
		})
		if err != nil {
			m.log.Error(err, "Delete Module", "msg", "failed to create a new destroy run")
			return err
		}
		m.log.Info("Delete Module", "msg", "successfully created a new destroy run")
		return r.updateStatusDestroy(ctx, &m.instance, run)
	}

	if waitRunToComplete(m.instance.Status.Run) {
		m.log.Info("Delete Module", "msg", "get destroy run status")
		run, err := m.tfClient.Client.Runs.Read(ctx, m.instance.Status.Run.ID)
		if err != nil {
			m.log.Error(err, "Delete Module", "msg", "failed to get destroy run status")
			return err
		}
		m.log.Info("Reconcile Run", "msg", fmt.Sprintf("successfully got destroy run status: %s", run.Status))

		if _, ok := runStatusComplete[run.Status]; ok {
			m.log.Info("Delete Module", "msg", "destroy run finished")
			return r.removeFinalizer(ctx, m)
		}

		return r.updateStatusDestroy(ctx, &m.instance, run)
	}

	return nil
}
