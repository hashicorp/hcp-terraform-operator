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
