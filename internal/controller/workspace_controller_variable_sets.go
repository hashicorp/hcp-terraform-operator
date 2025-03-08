// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
	"slices"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (w *workspaceInstance) getVariableSets(ctx context.Context) (map[string]*tfc.VariableSet, error) {
	variableSets := make(map[string]*tfc.VariableSet)

	listOpts := &tfc.VariableSetListOptions{
		ListOptions: tfc.ListOptions{
			PageSize: maxPageSize,
		},
	}

	for {
		v, err := w.tfClient.Client.VariableSets.List(ctx, w.instance.Spec.Organization, listOpts)
		if err != nil {
			w.log.Error(err, "Reconcile Variable Sets", "msg", "failed to get variable sets")
			return nil, err
		}

		for _, vs := range v.Items {
			variableSets[vs.ID] = vs
		}

		if v.NextPage == 0 {
			break
		}

		listOpts.PageNumber = v.NextPage
	}

	return variableSets, nil
}

func (r *WorkspaceReconciler) removeVariableSetFromWorkspace(ctx context.Context, w *workspaceInstance, vs *tfc.VariableSet) error {
	if vs.Global {
		w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("skipping global variable set %s", vs.ID))
		return nil
	}

	options := &tfc.VariableSetRemoveFromWorkspacesOptions{
		Workspaces: []*tfc.Workspace{{ID: w.instance.Status.WorkspaceID}},
	}

	return w.tfClient.Client.VariableSets.RemoveFromWorkspaces(ctx, vs.ID, options)

}

func (r *WorkspaceReconciler) applyVariableSetsToWorkspace(ctx context.Context, w *workspaceInstance, vs *tfc.VariableSet) error {

	options := &tfc.VariableSetApplyToWorkspacesOptions{
		Workspaces: []*tfc.Workspace{{ID: w.instance.Status.WorkspaceID}},
	}

	return w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, vs.ID, options)

}

func (w *workspaceInstance) updateVariableSetsStatus(status map[string]appv1alpha2.VariableSetStatus, id string) {
	if _, ok := status[id]; !ok {
		w.instance.Status.VariableSets = append(w.instance.Status.VariableSets, appv1alpha2.VariableSetStatus{ID: id})
	}
}

func (r *WorkspaceReconciler) reconcileVariableSets(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	w.log.Info("Reconcile Variable Sets", "msg", "starting reconciliation")

	if len(w.instance.Spec.VariableSets) == 0 && len(w.instance.Status.VariableSets) == 0 {
		return nil
	}

	workspaceVariableSets, err := w.getVariableSets(ctx)
	if err != nil {
		return err
	}

	if w.instance.Status.VariableSets == nil {
		w.instance.Status.VariableSets = make([]appv1alpha2.VariableSetStatus, 0)
	}

	specVariableSets := make(map[string]*tfc.VariableSet)
	statusVariableSets := make(map[string]appv1alpha2.VariableSetStatus)

	for _, vs := range w.instance.Spec.VariableSets {
		if wVS, ok := workspaceVariableSets[vs.ID]; ok {
			specVariableSets[vs.ID] = wVS
		} else {
			return fmt.Errorf("Variable set %s does not exist ", vs.ID)
		}
	}

	for _, statusVS := range w.instance.Status.VariableSets {
		statusVariableSets[statusVS.ID] = statusVS
	}

	//If both spec and status are not empty
	// 1 | 1
	w.log.Info("Reconcile Variable Sets", "msg", "Reconciling spec and status")

	for id, specVS := range specVariableSets {
		if specVS.Global {
			w.updateVariableSetsStatus(statusVariableSets, specVS.ID)
			continue
		}
		if _, exists := statusVariableSets[id]; exists {

			if slices.ContainsFunc(specVS.Workspaces, func(ws *tfc.Workspace) bool {
				return ws.ID == w.instance.Status.WorkspaceID
			}) {
				delete(statusVariableSets, id)
				continue
			}
		}

		w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Applying missing variable set %s to workspace", id))
		err := r.applyVariableSetsToWorkspace(ctx, w, specVS)
		if err != nil {
			w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Failed to apply variable set %s", id))
			return err
		}
		w.updateVariableSetsStatus(statusVariableSets, specVS.ID)
	}

	for id := range statusVariableSets {
		if vs, ok := workspaceVariableSets[id]; ok {
			w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Removing variable set %s from workspace", id))
			err := r.removeVariableSetFromWorkspace(ctx, w, vs)
			if err != nil {
				w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("Failed to remov'e variable set %s", id))
				return err
			}
		}
		w.instance.Status.VariableSets = slices.DeleteFunc(w.instance.Status.VariableSets, func(vs appv1alpha2.VariableSetStatus) bool {
			return vs.ID == id
		})
	}

	return nil
}
