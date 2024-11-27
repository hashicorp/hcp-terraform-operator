// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
)

func (w *workspaceInstance) getVariableSetIDs(ctx context.Context) ([]string, error) {
	variableSets := w.instance.Spec.VariableSets

	if variableSets == nil {
		return nil, fmt.Errorf("'spec.VariableSets' is not set")
	}

	var variableSetIDs []string
	var namesToLookup []string

	// Fetch all IDs and names to look up
	for _, vs := range variableSets {
		if vs.ID != "" {
			variableSetIDs = append(variableSetIDs, vs.ID)
		}
		if vs.Name != "" {
			namesToLookup = append(namesToLookup, vs.Name)
		}
	}

	if len(namesToLookup) > 0 {
		// Get all variable sets to find those referred by name
		listOpts := &tfc.VariableSetListOptions{
			ListOptions: tfc.ListOptions{
				PageSize: maxPageSize,
			},
		}

		// Get all var sets at once to optimize soln, a single API call
		for {
			variableSetList, err := w.tfClient.Client.VariableSets.List(ctx, w.instance.Spec.Organization, listOpts)
			if err != nil {
				return nil, err
			}

			// Searching by name instead
			for _, vs := range variableSetList.Items {
				for _, name := range namesToLookup {
					if vs.Name == name {
						variableSetIDs = append(variableSetIDs, vs.ID)
					}
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

	if len(variableSetIDs) == 0 {
		return nil, fmt.Errorf("no valid Variable Sets found in 'spec.VariableSets'")
	}

	return variableSetIDs, nil
}

func (r *WorkspaceReconciler) reconcileVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	// Get the ID for the variable sets specified in the spec
	variableSetIDs, err := w.getVariableSetIDs(ctx)
	if err != nil {
		return err
	}

	if len(variableSetIDs) == 0 {
		w.log.Info("Reconcile Variable Set", "msg", "no variable sets to reconcile")
		return nil
	}

	// List all variable sets once, a single API call
	listOpts := &tfc.VariableSetListOptions{
		ListOptions: tfc.ListOptions{
			PageSize: maxPageSize,
		},
	}
	variableSetList, err := w.tfClient.Client.VariableSets.List(ctx, w.instance.Spec.Organization, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list variable sets: %w", err)
	}

	// Create a map of variable set IDs to avoid multiple lookups - **Updated: Create a map for fast lookup**
	var variableSetsByID = make(map[string]*tfc.VariableSet)
	for _, vs := range variableSetList.Items {
		variableSetsByID[vs.ID] = vs
	}

	for _, variableSetID := range variableSetIDs {
		// Check if the variable set exists in the actual state
		variableSet, found := variableSetsByID[variableSetID]
		if !found {
			w.log.Info("Reconcile Variable Set", "msg", fmt.Sprintf("variable set %s not found in the organization", variableSetID))
			continue
		}

		// If the variable set is global, we can assume it is already applied to all workspaces
		if variableSet.Global {
			w.log.Info("Reconcile Variable Set", "msg", fmt.Sprintf("variable set %s is global, no need to apply again", variableSetID))
			continue
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

	}

	return nil
}
