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
	//Check if the variable set ID is valid (not empty)
	if vs.ID == "" {
		w.log.Info("Reconcile Variable Sets", "msg", "variable set ID is empty, cannot remove variable set from workspace")
		return fmt.Errorf("invalid variable set ID")
	}

	//Handle global variable sets
	//If a variable set is global it will not be removed
	if vs.Global {
		w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("variable set %s is global and will not be removed", vs.ID))
		return nil
	}

	w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("removing variable set %s from workspace", vs.ID))

	workspace := &tfc.Workspace{
		ID: w.instance.Status.WorkspaceID,
	}

	options := &tfc.VariableSetRemoveFromWorkspacesOptions{
		Workspaces: []*tfc.Workspace{workspace},
	}

	//Remove the variable set from the workspace
	err := w.tfClient.Client.VariableSets.RemoveFromWorkspaces(ctx, vs.ID, options)
	if err != nil {
		w.log.Error(err, "Reconcile Variable Sets", "msg", fmt.Sprintf("failed to remove variable set %s from workspace", vs.ID))
		return err
	}

	w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("successfully removed variable set %s from workspace", vs.ID))
	return nil
}

func (r *WorkspaceReconciler) reconcileVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	w.log.Info("Reconcile Variable Sets", "msg", "new reconciliation event")

	//Fetch the variable sets that are currently applied to the workspace
	currentVariableSets, err := w.getVariableSets(ctx, w, workspace)
	if err != nil {
		w.log.Error(err, "Reconcile Variable Sets", "msg", "failed to get workspace variable sets")
		return err
	}

	//Retrieve variable sets that are in the spec i.e. operator-managed variable sets
	//This handles variable sets that were manually created in HCP, for eg
	//a variable set that is applied to a project
	specVariableSets := w.instance.Spec.VariableSets
	specVariableSetIDs := make(map[string]struct{})
	for _, vs := range specVariableSets {
		specVariableSetIDs[vs.ID] = struct{}{}
	}

	//Remove variable sets that are not in the spec
	//If the variable set is no longer in the spec,
	//it is safe to assume the user would like to remove it from the ws
	for id, vs := range currentVariableSets {
		//Skip global variable sets
		if vs.Global {
			w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("variable set %s is global and cannot be removed", id))
			continue
		}

		//Remove variable sets that are not in the spec
		if _, inSpec := specVariableSetIDs[id]; !inSpec {
			w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("variable set %s is not defined in the spec...removing", id))
			if err := r.removeVariableSetFromWorkspace(ctx, w, vs); err != nil {
				return fmt.Errorf("failed to remove variable set %s: %w", id, err)
			}
		}
	}

	//Apply variable sets that are in the spec but not yet applied to the workspace
	//If a variable set is in the spec and has not yet been applied it will now be applied
	for _, vs := range specVariableSets {
		if _, exists := currentVariableSets[vs.ID]; !exists {
			options := &tfc.VariableSetApplyToWorkspacesOptions{
				Workspaces: []*tfc.Workspace{workspace},
			}
			err := w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, vs.ID, options)
			if err != nil {
				w.log.Error(err, "Reconcile Variable Sets", "msg", "failed to apply variable set", "id", vs.ID)
				return err
			}
			w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("applied variable set %s to workspace", vs.ID))
		}
	}

	w.log.Info("Reconcile Variable Sets", "msg", "successfully reconciled variable sets for workspace")
	return nil
}
