// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (r *workspaceInstance) getVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) (map[string]*tfc.VariableSet, error) {
	workspaceVariableSets := make(map[string]*tfc.VariableSet)

	listOpts := &tfc.VariableSetListOptions{
		ListOptions: tfc.ListOptions{
			PageSize: maxPageSize,
		},
	}

	for {
		v, err := w.tfClient.Client.VariableSets.ListForWorkspace(ctx, w.instance.Status.WorkspaceID, listOpts)
		if err != nil {
			w.log.Error(err, "Reconcile Variable Sets", "msg", "failed to get workspace variable sets")
			return nil, err
		}

		for _, vs := range v.Items {
			workspaceVariableSets[vs.ID] = vs
		}

		if v.NextPage == 0 {
			break
		}

		listOpts.PageNumber = v.NextPage
	}

	return workspaceVariableSets, nil
}

func (r *WorkspaceReconciler) removeVariableSetFromWorkspace(ctx context.Context, w *workspaceInstance, vs *tfc.VariableSet) error {
	if vs.Global {
		w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("skipping global variable set %s", vs.ID))
		return nil
	}

	options := &tfc.VariableSetRemoveFromWorkspacesOptions{
		Workspaces: []*tfc.Workspace{{ID: w.instance.Status.WorkspaceID}},
	}

	err := w.tfClient.Client.VariableSets.RemoveFromWorkspaces(ctx, vs.ID, options)
	if err != nil {
		return err
	}

	return nil
}

func (r *WorkspaceReconciler) reconcileVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	w.log.Info("Reconcile Variable Sets", "msg", "starting reconciliation")

	if w.instance.Status.VariableSets == nil {
		w.instance.Status.VariableSets = make(map[string]appv1alpha2.VariableSetStatus)
	}

	specVariableSets := make(map[string]appv1alpha2.VariableSetStatus)
	statusVariableSets := make(map[string]appv1alpha2.VariableSetStatus)

	for _, vs := range w.instance.Spec.VariableSets {
		specVariableSets[vs.ID] = appv1alpha2.VariableSetStatus{
			ID: vs.ID,
		}
	}

	if len(specVariableSets) == 0 && len(w.instance.Status.VariableSets) == 0 {
		return nil
	}

	workspaceVariableSets, err := w.getVariableSets(ctx, w, workspace)
	if err != nil {
		return err
	}

	for id, statusVS := range w.instance.Status.VariableSets {
		if _, exists := workspaceVariableSets[id]; exists {
			statusVariableSets[id] = statusVS
		}
	}

	//If the spec is not empty and status is empty, i.e. no variable sets have been applied yet
	// 1 | 0
	if len(specVariableSets) > 0 && len(statusVariableSets) == 0 {
		w.log.Info("Reconcile Variable Sets", "msg", "applying variable sets to workspace")
		for id, specVS := range specVariableSets {
			options := &tfc.VariableSetApplyToWorkspacesOptions{
				Workspaces: []*tfc.Workspace{workspace},
			}
			err := w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, id, options)
			if err != nil {
				w.log.Error(err, "Reconcile Variable Sets", "msg", fmt.Sprintf("Failed to apply variable set %s", id))
				return err
			}

			w.instance.Status.VariableSets[id] = specVS
		}
		return nil
	}

	//If the spec is empty and status is not empty
	//0 | 1
	if len(specVariableSets) == 0 && len(statusVariableSets) > 0 {
		w.log.Info("Reconcile Variable Sets", "msg", "removing variable sets from workspace as they are no longer in the spec")
		for id := range statusVariableSets {
			if _, ok := specVariableSets[id]; !ok {
				if vs, ok := workspaceVariableSets[id]; ok {
					w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Removing variable set %s from workspace", id))

					err := r.removeVariableSetFromWorkspace(ctx, w, vs)
					if err != nil {
						return err
					}

					delete(w.instance.Status.VariableSets, id)
					w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Successfully removed variable set %s from workspace", id))
				}
			}
		}
		return nil
	}

	//If both spec and status are not empty
	// 1 | 1
	w.log.Info("Reconcile Variable Sets", "msg", "reconciling spec and status")
	w.log.Info("Reconcile Variable Sets", "msg", "Current statusVariableSets", "statusVariableSets", statusVariableSets)

	for id, specVS := range specVariableSets {
		if _, exists := statusVariableSets[id]; !exists {
			w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Applying missing variable set %s to workspace", id))
			options := &tfc.VariableSetApplyToWorkspacesOptions{
				Workspaces: []*tfc.Workspace{workspace},
			}
			err := w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, id, options)
			if err != nil {
				w.log.Error(err, "Reconcile Variable Sets", "msg", fmt.Sprintf("Failed to apply variable set %s", id))
				return err
			}

			w.instance.Status.VariableSets[id] = specVS
		}
	}

	//Remove variable sets from workspace that are in status but not in spec
	//0 | 1
	for id := range statusVariableSets {
		if _, exists := specVariableSets[id]; !exists {
			if vs, ok := workspaceVariableSets[id]; ok {
				err := r.removeVariableSetFromWorkspace(ctx, w, vs)
				if err != nil {
					return err
				}
				delete(w.instance.Status.VariableSets, id)
			}
		}
	}

	return nil
}
