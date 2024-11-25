// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
)

func (w *workspaceInstance) getVariableSetIDByName(ctx context.Context) (string, error) {
	variableSets := w.instance.Spec.VariableSets

	// Iterate over each variable set in the slice
	for _, variableSet := range variableSets {
		// Check if both ID and Name are provided for this specific variable set
		if variableSet.ID != "" {
			return variableSet.ID, nil
		}

		if variableSet.Name != "" {
			// Search by name if the ID is not available
			listOpts := &tfc.VariableSetListOptions{
				ListOptions: tfc.ListOptions{
					PageSize: maxPageSize,
				},
			}

			for {
				variableSetList, err := w.tfClient.Client.VariableSets.List(ctx, w.instance.Spec.Organization, listOpts)
				if err != nil {
					return "", err
				}

				// Iterate through the returned variable sets to find a match by name
				for _, vs := range variableSetList.Items {
					if vs.Name == variableSet.Name {
						return vs.ID, nil
					}
				}

				// Break if there are no more pages to fetch
				if variableSetList.NextPage == 0 {
					break
				}

				// Update page number to fetch the next set of results
				listOpts.PageNumber = variableSetList.NextPage
			}
		}
	}

	return "", fmt.Errorf("variable set ID not found for any of the provided variable set names")
}

func (w *workspaceInstance) getVariableSetID(ctx context.Context) (string, error) {
	specVariableSets := w.instance.Spec.VariableSets

	if specVariableSets == nil {
		return "", fmt.Errorf("'spec.VariableSets' is not set")
	}

	for _, vs := range specVariableSets {
		if vs.Name != "" {
			w.log.Info("Reconcile Variable Set", "msg", "getting variable set ID by name")
			return w.getVariableSetIDByName(ctx)
		}

		w.log.Info("Reconcile Variable Set", "msg", "getting variable set ID from the spec.VariableSets.ID")
		if vs.ID != "" {
			return vs.ID, nil
		}
	}

	return "", fmt.Errorf("no valid Variable Set found in 'spec.VariableSets'")

}

func (r *WorkspaceReconciler) reconcileVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	// Get the ID for the variable sets specified in the spec
	variableSetID, err := w.getVariableSetID(ctx)
	if err != nil {
		return err
	}

	// Get the variable set using its ID
	variableSet, err := w.tfClient.Client.VariableSets.Read(ctx, variableSetID, nil)
	if err != nil {
		return fmt.Errorf("failed to read variable set with ID %s: %w", variableSetID, err)
	}

	// If the variable set is global, we assume it is already applied to all workspaces
	if variableSet.Global {
		w.log.Info("Reconcile Variable Set", "msg", "variable set is global, no need to apply again")
		return nil
	}

	// List the variable sets already applied to the workspace
	variableSetsForWorkspace, err := w.tfClient.Client.VariableSets.ListForWorkspace(ctx, workspace.ID, nil)
	if err != nil {
		return fmt.Errorf("failed to list variable sets for workspace %s: %w", workspace.ID, err)
	}

	// Check if the variable set is already applied to the workspace
	isApplied := false
	for _, vs := range variableSetsForWorkspace.Items {
		if vs.ID == variableSetID {
			isApplied = true
			break
		}
	}

	// If the variable set is not applied, apply it to the workspace
	if !isApplied {
		w.log.Info("Reconcile Variable Set", "msg", fmt.Sprintf("applying variable set %s to workspace %s", variableSetID, workspace.ID))
		err = w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, variableSetID, &tfc.VariableSetApplyToWorkspacesOptions{
			Workspaces: []*tfc.Workspace{workspace},
		})
		if err != nil {
			return fmt.Errorf("failed to apply variable set %s to workspace %s: %w", variableSetID, workspace.ID, err)
		}
		w.log.Info("Reconcile Variable Set", "msg", fmt.Sprintf("successfully applied variable set %s to workspace %s", variableSetID, workspace.ID))
	}

	return nil
}
