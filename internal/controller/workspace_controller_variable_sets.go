// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	abstractions "github.com/microsoft/kiota-abstractions-go"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func (w *workspaceInstance) getOrgVariableSets(ctx context.Context) (map[string]models.Varsetsable, error) {
	variableSets := make(map[string]models.Varsetsable)
	query := &organizations.ItemVarsetsRequestBuilderGetQueryParameters{
		Pagesize: new(int32(MaxPageSize)),
	}

	for {
		v, err := w.tfClient.Client.API.Organizations().ByOrganization_name(w.instance.Spec.Organization).Varsets().Get(ctx,
			&abstractions.RequestConfiguration[organizations.ItemVarsetsRequestBuilderGetQueryParameters]{QueryParameters: query})
		if err != nil {
			w.log.Error(err, "Reconcile Variable Sets", "msg", "failed to get variable sets")
			return nil, err
		}

		for _, vs := range v.GetData() {
			variableSets[pointer.ValueOf(vs.GetId())] = vs
		}

		next := nextPage(v.GetMeta())
		if next == 0 {
			break
		}
		query.Pagenumber = &next
	}

	return variableSets, nil
}

func (w *workspaceInstance) removeVariableSetFromWorkspace(ctx context.Context, vs models.Varsetsable) error {
	id := pointer.ValueOf(vs.GetId())
	if pointer.ValueOf(vs.GetAttributes().GetGlobal()) {
		w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("skipping global variable set %s", id))
		return nil
	}

	w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("removing variable set %s from workspace", id))
	ws := models.NewWorkspacesIdentifierArrayDocument_data()
	ws.SetId(new(w.instance.Status.WorkspaceID))
	ws.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIERARRAYDOCUMENT_DATA_TYPE))
	body := models.NewWorkspacesIdentifierArrayDocument()
	body.SetData([]models.WorkspacesIdentifierArrayDocument_dataable{ws})
	return w.tfClient.Client.API.Varsets().ByVarset_id(id).Relationships().Workspaces().Delete(ctx, body, nil)
}

func (w *workspaceInstance) applyVariableSetsToWorkspace(ctx context.Context, vs models.Varsetsable) error {
	id := pointer.ValueOf(vs.GetId())
	if pointer.ValueOf(vs.GetAttributes().GetGlobal()) {
		return nil
	}

	w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("applying missing variable set %s to workspace", id))
	ws := models.NewWorkspacesIdentifierArrayDocument_data()
	ws.SetId(new(w.instance.Status.WorkspaceID))
	ws.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIERARRAYDOCUMENT_DATA_TYPE))
	body := models.NewWorkspacesIdentifierArrayDocument()
	body.SetData([]models.WorkspacesIdentifierArrayDocument_dataable{ws})
	return w.tfClient.Client.API.Varsets().ByVarset_id(id).Relationships().Workspaces().Post(ctx, body, nil)
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

	specVariableSets := make(map[string]models.Varsetsable)
	statusVariableSets := make(map[string]struct{})

	for _, vs := range w.instance.Spec.VariableSets {
		if vs.Name != "" {
			f := false
			for _, wvs := range workspaceVariableSets {
				if vs.Name == pointer.ValueOf(wvs.GetAttributes().GetName()) {
					specVariableSets[pointer.ValueOf(wvs.GetId())] = wvs
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
			if vs.GetRelationships() != nil && vs.GetRelationships().GetWorkspaces() != nil && slices.ContainsFunc(vs.GetRelationships().GetWorkspaces().GetData(), func(ws models.WorkspacesIdentifierable) bool {
				return pointer.ValueOf(ws.GetId()) == w.instance.Status.WorkspaceID
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
		w.addOrUpdateVariableSetsStatus(appv1alpha2.VariableSetStatus{ID: pointer.ValueOf(vs.GetId()), Name: pointer.ValueOf(vs.GetAttributes().GetName())})
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
