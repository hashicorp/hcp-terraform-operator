// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-tfe"
	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (r *workspaceInstance) getVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) (map[string]*tfc.VariableSet, error) {
	workspaceVariableSets := make(map[string]*tfc.VariableSet)

	w.log.Info("Reconcile Variable Sets", "msg", "fetching workspace variable sets")

	listOpts := &tfc.VariableSetListOptions{
		ListOptions: tfc.ListOptions{
			PageSize: maxPageSize,
		},
	}

	//Fetch paginated variable sets
	for {
		v, err := w.tfClient.Client.VariableSets.ListForWorkspace(ctx, w.instance.Status.WorkspaceID, listOpts)
		if err != nil {
			w.log.Error(err, "Reconcile Variable Sets", "msg", "failed to get workspace variable sets")
			return nil, err
		}

		//Store variable sets in the map, use ID as the key
		for _, vs := range v.Items {
			workspaceVariableSets[vs.ID] = vs
		}

		if v.NextPage == 0 {
			break
		}

		listOpts.PageNumber = v.NextPage
	}

	w.log.Info("Reconcile Variable Sets", "msg", "successfully fetched workspace variable sets")
	return workspaceVariableSets, nil
}

func (r *WorkspaceReconciler) removeVariableSetFromWorkspace(ctx context.Context, w *workspaceInstance, vs *tfc.VariableSet) error {
	// Skip global variable sets
	if vs.Global {
		w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("skipping global variable set %s", vs.ID))
		return nil
	}

	w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("removing variable set %s from workspace", vs.ID))

	options := &tfc.VariableSetRemoveFromWorkspacesOptions{
		Workspaces: []*tfc.Workspace{{ID: w.instance.Status.WorkspaceID}},
	}

	//Removing variable set
	err := w.tfClient.Client.VariableSets.RemoveFromWorkspaces(ctx, vs.ID, options)
	if err != nil {
		w.log.Error(err, "Reconcile Variable Sets", "msg", fmt.Sprintf("failed to remove variable set %s from workspace", vs.ID))
		return err
	}

	w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("successfully removed variable set %s from workspace", vs.ID))
	return nil
}

func (r *WorkspaceReconciler) reconcileVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	w.log.Info("Reconcile Variable Sets", "msg", "starting reconciliation")

	//If VariableSets in status are nil, initialize as an empty map
	if w.instance.Status.VariableSets == nil {
		w.instance.Status.VariableSets = make(map[string]appv1alpha2.VariableSetStatus)
	}

	//Fetch all current variable sets from workspace
	workspaceVariableSets, err := w.getVariableSets(ctx, w, workspace)
	if err != nil {
		return err
	}

	//Maps to store spec and status variable sets
	specVariableSets := make(map[string]appv1alpha2.VariableSetStatus)
	statusVariableSets := make(map[string]appv1alpha2.VariableSetStatus)

	//The spec contains the variable sets that will be applied to the workspace
	for _, vs := range w.instance.Spec.VariableSets {
		specVariableSets[vs.ID] = appv1alpha2.VariableSetStatus{
			ID:      vs.ID,
			Applied: false,      //Set to false, because initially they are not applied
			Source:  "operator", //Flag to indicate the varset is managed by the operator
		}
	}

	//Update status with the variable sets that are currently applied to the workspace
	for id, statusVS := range w.instance.Status.VariableSets {
		statusVariableSets[id] = statusVS
	}

	//Reconcile the spec and status variable sets
	//If both spec and status are empty
	// 0 | 0
	if len(specVariableSets) == 0 && len(statusVariableSets) == 0 {
		// Log that both are empty and no action is needed
		w.log.Info("Reconcile Variable Sets", "msg", "both spec and status are empty, nothing to do")
		return nil
	}

	//If the spec is empty and status is not empty
	// 0 | 1
	if len(specVariableSets) == 0 && len(statusVariableSets) > 0 {
		w.log.Info("Reconcile Variable Sets", "msg", "removing variable sets from workspace as they are no longer in the spec")
		//Remove all variable sets in status - as they're no longer in the spec
		for id := range statusVariableSets {
			var variableSetToRemove *tfe.VariableSet
			//Search for the corresponding variable set in the workspace
			for _, vs := range workspaceVariableSets {
				if vs.ID == id {
					variableSetToRemove = vs
					break
				}
			}

			//If found, remove it from the workspace i.e. no longer applied to the workspace
			if variableSetToRemove != nil {
				err := r.removeVariableSetFromWorkspace(ctx, w, variableSetToRemove)
				if err != nil {
					return err
				}
				//Update the status after removing the variable set from the workspace
				delete(w.instance.Status.VariableSets, id)
			} else {
				//Log that the variable set is not found in the workspace
				w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Variable set %s not found in workspace", id))
			}
		}
		return nil
	}

	//If the spec is not empty and status is empty, i.e. no variable sets have been applied yet
	// 1 | 0
	if len(specVariableSets) > 0 && len(statusVariableSets) == 0 {
		w.log.Info("Reconcile Variable Sets", "msg", "applying all variable sets from spec to workspace")
		//Apply all variable sets from the spec to the workspace
		for id, specVS := range specVariableSets {
			options := &tfc.VariableSetApplyToWorkspacesOptions{
				Workspaces: []*tfc.Workspace{workspace},
			}
			//Apply variable set to workspace
			err := w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, id, options)
			if err != nil {
				w.log.Error(err, "Reconcile Variable Sets", "msg", fmt.Sprintf("Failed to apply variable set %s", id))
				return err
			}

			//Update the status after applying the variable set
			w.instance.Status.VariableSets[id] = specVS
		}
		return nil
	}

	//If both spec and status are not empty
	// 1 | 1
	w.log.Info("Reconcile Variable Sets", "msg", "reconciling spec and status")
	w.log.Info("Reconcile Variable Sets", "msg", "Current statusVariableSets", "statusVariableSets", statusVariableSets)

	//Apply any new variable sets from spec that are not in status (spec = 1 | status = 0)
	for id, specVS := range specVariableSets {
		if _, exists := statusVariableSets[id]; !exists { //Exists in spec but not in status
			w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Applying missing variable set %s to workspace", id))
			options := &tfc.VariableSetApplyToWorkspacesOptions{
				Workspaces: []*tfc.Workspace{workspace},
			}
			err := w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, id, options)
			if err != nil {
				w.log.Error(err, "Reconcile Variable Sets", "msg", fmt.Sprintf("Failed to apply variable set %s", id))
				return err
			}

			//Update the status after applying the variable set
			w.instance.Status.VariableSets[id] = specVS
		}
	}

	//Remove variable sets from workspace that are in status but not in spec (spec = 0 | status = 1)
	for id := range statusVariableSets {
		w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Removing variable set %s from workspace", id))
		if _, exists := specVariableSets[id]; !exists { //Exists in status but not in spec
			//Search for the variable set in the workspace
			var variableSetToRemove *tfe.VariableSet
			for _, vs := range workspaceVariableSets {
				if vs.ID == id {
					variableSetToRemove = vs
					break
				}
			}

			//If found, remove it from the workspace and update status
			if variableSetToRemove != nil {
				err := r.removeVariableSetFromWorkspace(ctx, w, variableSetToRemove)
				if err != nil {
					return err
				}
				//Update the status after removing the variable set from the workspace
				delete(w.instance.Status.VariableSets, id)
			} else {
				//If variable set was not found
				w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Variable set %s not found in workspace", id))
			}
		}
	}

	//Final status after reconciliation
	w.log.Info("Reconcile Variable Sets", "msg", "Updated variable sets in status", "status", w.instance.Status.VariableSets)

	w.log.Info("Reconcile Variable Sets", "msg", "Reconciliation complete")
	return nil
}
