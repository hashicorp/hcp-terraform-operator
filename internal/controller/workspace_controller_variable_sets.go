// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/slice"
)

func (r *workspaceInstance) getVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) (map[string]*tfc.VariableSet, error) {
	workspaceVariableSets := make(map[string]*tfc.VariableSet)

	listOpts := &tfc.VariableSetListOptions{
		ListOptions: tfc.ListOptions{
			PageSize: maxPageSize,
		},
	}

	for {
		v, err := w.tfClient.Client.VariableSets.List(ctx, w.instance.Spec.Organization, listOpts)
		if err != nil {
			w.log.Info("Reconcile Variable Sets", "msg", "failed to get workspace variable sets")
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

	if len(w.instance.Spec.VariableSets) == 0 && len(w.instance.Status.VariableSets) == 0 {
		return nil
	}

	if w.instance.Status.VariableSets == nil {
		w.instance.Status.VariableSets = make([]appv1alpha2.VariableSetStatus, 0)
	}

	specVariableSets := make(map[string]appv1alpha2.VariableSetStatus)
	statusVariableSets := make(map[string]appv1alpha2.VariableSetStatus)

	for _, vs := range w.instance.Spec.VariableSets {
		specVariableSets[vs.ID] = appv1alpha2.VariableSetStatus{
			ID: vs.ID,
		}
	}

	workspaceVariableSets, err := w.getVariableSets(ctx, w, workspace)
	if err != nil {
		return err
	}

	for _, statusVS := range w.instance.Status.VariableSets {
		//if _, exists := workspaceVariableSets[id]; exists {
		statusVariableSets[statusVS.ID] = statusVS
		//}
	}

	//If the spec is not empty and status is empty, i.e. no variable sets have been applied yet
	// 1 | 0
	if len(specVariableSets) > 0 && len(statusVariableSets) == 0 {
		for id, specVS := range specVariableSets {
			if workspaceVS, ok := workspaceVariableSets[id]; ok {
				if !workspaceVS.Global {
					w.log.Info("Reconcile Variable Sets", "msg", "applying variable sets to workspace")
					options := &tfc.VariableSetApplyToWorkspacesOptions{
						Workspaces: []*tfc.Workspace{workspace},
					}
					err := w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, id, options)
					if err != nil {
						w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Failed to apply variable set %s", id))
						return err
					}
				}

			} else {
				return fmt.Errorf("Variable set %s does not exist ", id)
			}
			//w.instance.Status.VariableSets[id] = specVS
			w.instance.Status.VariableSets = append(w.instance.Status.VariableSets, specVS)
		}
		return nil
	}

	//If the spec is empty and status is not empty
	//0 | 1
	if len(specVariableSets) == 0 && len(statusVariableSets) > 0 {
		for id := range statusVariableSets {
			if vs, ok := workspaceVariableSets[id]; ok {
				if !vs.Global {
					w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Removing variable set %s from workspace", id))

					err := r.removeVariableSetFromWorkspace(ctx, w, vs)
					if err != nil {
						return err
					}
					w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Successfully removed variable set %s from workspace", id))
				}
			}

			for v, DeleteVS := range w.instance.Status.VariableSets {
				if DeleteVS.ID == id {
					w.instance.Status.VariableSets = slice.RemoveFromSlice(w.instance.Status.VariableSets, v)
					break
				}
			}

		}
		return nil
	}

	//If both spec and status are not empty
	// 1 | 1
	w.log.Info("Reconcile Variable Sets", "msg", "reconciling spec and status")

	for id, specVS := range specVariableSets {
		if workspaceVS, ok := workspaceVariableSets[id]; ok {

			if _, exists := statusVariableSets[id]; !exists {
				w.instance.Status.VariableSets = append(w.instance.Status.VariableSets, specVS)
			}

			if _, exists := statusVariableSets[id]; !exists {
				if !workspaceVS.Global {
					w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Applying missing variable set %s to workspace", id))
					options := &tfc.VariableSetApplyToWorkspacesOptions{
						Workspaces: []*tfc.Workspace{workspace},
					}
					err := w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, id, options)
					if err != nil {
						w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Failed to apply variable set %s", id))
						return err
					}
				}
			} else {
				if !workspaceVS.Global {
					applied := false
					for _, ws := range workspaceVS.Workspaces {
						if ws.ID == w.instance.Status.WorkspaceID {
							applied = true
							break
						}
					}
					if !applied {
						options := &tfc.VariableSetApplyToWorkspacesOptions{
							Workspaces: []*tfc.Workspace{workspace},
						}
						err := w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, id, options)
						if err != nil {
							w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Failed to apply variable set %s", id))
							return err
						}
					}
				}
			}
			//w.instance.Status.VariableSets[id] = specVS
			//w.instance.Status.VariableSets = append(w.instance.Status.VariableSets, specVS)
		} else {
			return fmt.Errorf("variable set %s does not exist ", id)
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
			}
			for v, DeleteVS := range w.instance.Status.VariableSets {
				if DeleteVS.ID == id {
					w.instance.Status.VariableSets = slice.RemoveFromSlice(w.instance.Status.VariableSets, v)
					break
				}
			}
		}
	}

	return nil
}
