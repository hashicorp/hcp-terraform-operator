// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"

	tfc "github.com/hashicorp/go-tfe"
)

func (r *workspaceInstance) getVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) (map[string]*tfc.VariableSet, error) {
	//variableSets := w.instance.Spec.VariableSets

	//No variable sets
	// if workspace.VariableSet == nil {
	// 	return nil, nil
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

func (r *WorkspaceReconciler) reconcileVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	w.log.Info("Reconcile Variable Sets", "msg", "new reconciliation event")

	variableSets, err := w.getVariableSets(ctx, w, workspace)
	if err != nil {
		w.log.Error(err, "Reconcile Variable Sets", "msg", "failed to get workspace variable sets")
		return err
	}

	//Reconciling varsets now
	//Global = no need to apply
	//Not global = apply, if that is the varset specified
	for _, vs := range variableSets {
		//If the variable set is not global, we might need to apply it to the workspace
		if !vs.Global {
			options := &tfc.VariableSetApplyToWorkspacesOptions{
				Workspaces: []*tfc.Workspace{workspace},
			}
			err := w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, vs.ID, options)
			if err != nil {
				w.log.Error(err, "Reconcile Variable Sets", "msg", "failed to apply variable set", "id", vs.ID)
				return err
			}
			w.log.Info("Applied variable set", "id", vs.ID)
		} else {
			w.log.Info("Skipping global variable set", "id", vs.ID)
		}
	}

	w.log.Info("Reconcile Variable Sets", "msg", "successfully reconciled variable sets for workspace")

	return nil
}
