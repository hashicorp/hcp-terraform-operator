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

func (w *workspaceInstance) getOrgVariableSets(ctx context.Context) (map[string]*tfc.VariableSet, error) {
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

func (w *workspaceInstance) removeVariableSetFromWorkspace(ctx context.Context, vs *tfc.VariableSet) error {
	if vs.Global {
		w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("skipping global variable set %s", vs.ID))
		return nil
	}
	w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("removing variable set %s from workspace", vs.ID))
	return w.tfClient.Client.VariableSets.RemoveFromWorkspaces(ctx, vs.ID, &tfc.VariableSetRemoveFromWorkspacesOptions{
		Workspaces: []*tfc.Workspace{{ID: w.instance.Status.WorkspaceID}},
	})
}

func (w *workspaceInstance) applyVariableSetsToWorkspace(ctx context.Context, vs *tfc.VariableSet) error {
	if vs.Global {
		return nil
	}
	w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("applying missing variable set %s to workspace", vs.ID))
	return w.tfClient.Client.VariableSets.ApplyToWorkspaces(ctx, vs.ID, &tfc.VariableSetApplyToWorkspacesOptions{
		Workspaces: []*tfc.Workspace{{ID: w.instance.Status.WorkspaceID}},
	})

}

func (w *workspaceInstance) addOrUpdateVariableSetsStatus(vs appv1alpha2.VariableSetStatus) {
	if slices.ContainsFunc(w.instance.Status.VariableSets, func(v appv1alpha2.VariableSetStatus) bool {
		return v.ID == vs.ID
	}) {
		return
	}
	w.instance.Status.VariableSets = append(w.instance.Status.VariableSets, vs)
}

func (w *workspaceInstance) deleteVariableSetsStatus(id string) {
	w.instance.Status.VariableSets = slices.DeleteFunc(w.instance.Status.VariableSets, func(vs appv1alpha2.VariableSetStatus) bool {
		return vs.ID == id
	})
}

func (r *WorkspaceReconciler) reconcileVariableSets(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Variable Sets", "msg", "new reconciliation event")

	if len(w.instance.Spec.VariableSets) == 0 && len(w.instance.Status.VariableSets) == 0 {
		return nil
	}

	workspaceVariableSets, err := w.getOrgVariableSets(ctx)
	if err != nil {
		return err
	}

	specVariableSets := make(map[string]*tfc.VariableSet)
	statusVariableSets := make(map[string]struct{})

	for _, vs := range w.instance.Spec.VariableSets {
		if vs.Name != "" {
			f := false
			for _, wvs := range workspaceVariableSets {
				if vs.Name == wvs.Name {
					specVariableSets[wvs.ID] = wvs
					f = true
				}
			}
			if !f {
				return fmt.Errorf("variable set %s does not exist ", vs.Name)
			}
		}
		if vs.ID != "" {
			if wvs, ok := workspaceVariableSets[vs.ID]; ok {
				specVariableSets[vs.ID] = wvs
			} else {
				return fmt.Errorf("variable set %s does not exist ", vs.ID)
			}
		}
	}

	for _, vs := range w.instance.Status.VariableSets {
		statusVariableSets[vs.ID] = struct{}{}
	}

	for id, vs := range specVariableSets {
		if _, ok := statusVariableSets[id]; ok {
			if slices.ContainsFunc(vs.Workspaces, func(ws *tfc.Workspace) bool {
				return ws.ID == w.instance.Status.WorkspaceID
			}) {
				delete(statusVariableSets, id)
				continue
			}
		}
		err := w.applyVariableSetsToWorkspace(ctx, vs)
		if err != nil {
			w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("failed to apply variable set %s", id))
			return err
		}
		w.addOrUpdateVariableSetsStatus(appv1alpha2.VariableSetStatus{ID: vs.ID, Name: vs.Name})
		delete(statusVariableSets, id)
	}

	for id := range statusVariableSets {
		if vs, ok := workspaceVariableSets[id]; ok {
			err := w.removeVariableSetFromWorkspace(ctx, vs)
			if err != nil {
				w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("failed to remove variable set %s", id))
				return err
			}
		}
		w.deleteVariableSetsStatus(id)
	}

	return nil
}
