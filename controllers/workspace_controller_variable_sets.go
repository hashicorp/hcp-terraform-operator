// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
)

func (r *workspaceInstance) getVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) (map[string]*tfc.VariableSet, error) {
	//variableSets := w.instance.Spec.VariableSets

	//No variable sets
	// if workspace.VariableSet == nil {
	//  return nil, nil
	// }

	//empty map
	s := make(map[string]*tfc.VariableSet)

	w.log.Info("Reconcile Variable Sets", "msg", "getting workspace variable sets")
	listOpts := &tfc.VariableSetListOptions{
		ListOptions: tfc.ListOptions{
			PageSize: maxPageSize,
		},
	}

	//Paginate through to fetch the variable sets applied to the workspace
	for {
		//Fetch the list of variable sets for this workspace
		v, err := w.tfClient.Client.VariableSets.ListForWorkspace(ctx, w.instance.Status.WorkspaceID, listOpts)
		if err != nil {
			w.log.Error(err, "Reconcile Variable Sets", "msg", "failed to get workspace variable sets")
			return nil, err
		}

		//Add variable sets to the map using ID as the key, so we can look it up easily
		for _, vs := range v.Items {
			s[vs.ID] = vs
		}

		//No more pages
		if v.NextPage == 0 {
			break
		}

		//if we have more pages then....
		listOpts.PageNumber = v.NextPage
	}

	w.log.Info("Reconcile Variables", "msg", "successfully got workspace variables")

	return s, nil

}

func (r *WorkspaceReconciler) removeVariableSetFromWorkspace(ctx context.Context, w *workspaceInstance, vs *tfc.VariableSet) error {
	// Check if the variable set ID is valid (not empty)
	if vs.ID == "" {
		w.log.Info("Reconcile Variable Sets", "msg", "variable set ID is empty, cannot remove variable set from workspace")
		return fmt.Errorf("invalid variable set ID")
	}

	w.log.Info("Reconcile Variable Sets", "msg", vs.ID)

	workspace := &tfc.Workspace{
		ID: w.instance.Status.WorkspaceID,
	}

	options := &tfc.VariableSetRemoveFromWorkspacesOptions{
		Workspaces: []*tfc.Workspace{workspace},
	}

	err := w.tfClient.Client.VariableSets.RemoveFromWorkspaces(ctx, vs.ID, options)
	if err != nil {
		return err
	}

	w.log.Info("Reconcile Variable Sets", "msg", vs.ID)
	return nil
}

func (r *WorkspaceReconciler) reconcileVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	w.log.Info("Reconcile Variable Sets", "msg", "new reconciliation event")

	// Get the current variable sets applied to the workspace
	variableSets, err := w.getVariableSets(ctx, w, workspace)
	if err != nil {
		w.log.Info("Reconcile Variable Sets", "msg", "failed to get workspace variable sets")
		return err
	}

	// Retrieve the variable sets declared in the spec
	specVariableSets := w.instance.Spec.VariableSets

	// Log all spec variable sets to ensure they are populated with valid IDs
	for _, vs := range specVariableSets {
		w.log.Info("Spec Variable Set", "msg", vs.ID)
	}

	// Remove variable sets that are in the workspace but not in the spec (un-apply them)
	for _, vs := range variableSets {
		if vs.ID == "" {
			w.log.Info("Reconcile Variable Sets", "msg", vs)
			continue // Skip invalid variable sets
		}

		// Check if the variable set is in the spec (iterate through the spec variable sets)
		found := false
		for _, specVS := range specVariableSets {
			if specVS.ID == vs.ID {
				found = true
				break
			}
		}

		if !found {
			// If the variable set is not in the spec anymore, un-apply it from the workspace
			if err := r.removeVariableSetFromWorkspace(ctx, w, vs); err != nil {
				return err
			}
		}
	}

	// Apply variable sets that are in the spec but not applied yet
	for _, vs := range specVariableSets {
		if _, ok := variableSets[vs.ID]; !ok {
			// If the variable set is in the spec but not in the workspace, apply it
			options := &tfc.VariableSetApplyToWorkspacesOptions{
				Workspaces: []*tfc.Workspace{workspace},
			}
			err := w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, vs.ID, options)
			if err != nil {
				w.log.Error(err, "Reconcile Variable Sets", "msg", "failed to apply variable set", "id", vs.ID)
				return err
			}
			w.log.Info("Applied variable set", "id", vs.ID)
		}
	}

	w.log.Info("Reconcile Variable Sets", "msg", "successfully reconciled variable sets for workspace")

	return nil
}
