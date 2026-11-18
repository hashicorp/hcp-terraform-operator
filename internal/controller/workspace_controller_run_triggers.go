// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"errors"
	"fmt"

	tfc "github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/workspaces"
	"github.com/hashicorp/go-tfe/v2/api/workspaces/item/runtriggers"
	abstractions "github.com/microsoft/kiota-abstractions-go"

	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func (r *WorkspaceReconciler) getRunTriggersSources(ctx context.Context, w *workspaceInstance) (map[string]string, error) {
	runTriggersIDs := make(map[string]string)
	runTriggersNames := []string{}

	for _, rt := range w.instance.Spec.RunTriggers {
		if rt.Name != "" {
			runTriggersNames = append(runTriggersNames, rt.Name)
		}
		if rt.ID != "" {
			runTriggersIDs[rt.ID] = ""
		}
	}

	for _, workspaceName := range runTriggersNames {
		ws, err := w.tfClient.Client.API.Organizations().ByOrganization_name(w.instance.Spec.Organization).Workspaces().ByWorkspace_name(workspaceName).Get(ctx, nil)
		if errors.Is(err, tfc.ErrNotFound) {
			return nil, fmt.Errorf("cannot find ID for Workspace %s", workspaceName)
		}
		if err != nil {
			return nil, err
		}
		runTriggersIDs[pointer.ValueOf(ws.GetData().GetId())] = ""
	}

	return runTriggersIDs, nil
}

func (r *WorkspaceReconciler) getRunTriggersWorkspace(ctx context.Context, w *workspaceInstance) (map[string]string, error) {
	rrt := make(map[string]string)
	query := &workspaces.ItemRunTriggersRequestBuilderGetQueryParameters{
		FilterrunTriggertype: new(runtriggers.INBOUND_GETFILTERRUNTRIGGERTYPEQUERYPARAMETERTYPE),
		Pagesize:             new(int32(MaxPageSize)),
	}

	for {
		runTriggers, err := w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).RunTriggers().Get(ctx,
			&abstractions.RequestConfiguration[workspaces.ItemRunTriggersRequestBuilderGetQueryParameters]{QueryParameters: query})
		if err != nil {
			return nil, err
		}

		for _, rt := range runTriggers.GetData() {
			// Sourceable ID is an ID of the sourceable Workspace
			if rt.GetRelationships() == nil || rt.GetRelationships().GetSourceable() == nil || rt.GetRelationships().GetSourceable().GetData() == nil {
				continue
			}
			rrt[pointer.ValueOf(rt.GetRelationships().GetSourceable().GetData().GetId())] = pointer.ValueOf(rt.GetId())
		}

		next := nextPage(runTriggers.GetMeta())
		if next == 0 {
			break
		}
		query.Pagenumber = &next
	}

	return rrt, nil
}

func getWorkspacesToAdd(instanceWorkspaces, runTriggersWorkspaces map[string]string) map[string]string {
	return workspaceDifference(instanceWorkspaces, runTriggersWorkspaces)
}

func getWorkspacesToDelete(instanceWorkspaces, runTriggersWorkspaces map[string]string) map[string]string {
	return workspaceDifference(runTriggersWorkspaces, instanceWorkspaces)
}

func workspaceDifference(leftWorkspace, rightWorkspace map[string]string) map[string]string {
	d := make(map[string]string)

	for kl, vl := range leftWorkspace {
		if _, ok := rightWorkspace[kl]; !ok {
			d[kl] = vl
		}
	}

	return d
}

func (r *WorkspaceReconciler) addRunTriggers(ctx context.Context, w *workspaceInstance, workspaces map[string]string) error {
	if len(workspaces) == 0 {
		return nil
	}

	for workspace := range workspaces {
		source := models.NewWorkspacesHasOne_data()
		source.SetId(new(workspace))
		source.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIER_TYPE))
		sourceRel := models.NewWorkspacesHasOne()
		sourceRel.SetData(source)
		rels := models.NewRunTriggers_relationships()
		rels.SetSourceable(sourceRel)
		data := models.NewRunTriggers()
		data.SetTypeEscaped(new(models.RUNTRIGGERS_RUNTRIGGERS_TYPE))
		data.SetRelationships(rels)
		body := models.NewRunTriggersEnvelope()
		body.SetData(data)

		_, err := w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).RunTriggers().Post(ctx, body, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) removeRunTriggers(ctx context.Context, w *workspaceInstance, workspaces map[string]string) error {
	if len(workspaces) == 0 {
		return nil
	}

	for _, workspace := range workspaces {
		err := w.tfClient.Client.API.RunTriggers().ById(workspace).Delete(ctx, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) reconcileRunTriggers(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Run Triggers", "msg", "new reconciliation event")

	instanceRunTriggers, err := r.getRunTriggersSources(ctx, w)
	if err != nil {
		return nil
	}
	workspaceRunTriggers, err := r.getRunTriggersWorkspace(ctx, w)
	if err != nil {
		return nil
	}

	addWorkspaces := getWorkspacesToAdd(instanceRunTriggers, workspaceRunTriggers)
	if len(addWorkspaces) > 0 {
		w.log.Info("Reconcile Run Triggers", "msg", "adding run triggers workspaces to the workspace")
		err = r.addRunTriggers(ctx, w, addWorkspaces)
		if err != nil {
			return err
		}
	}

	deleteWorkspaces := getWorkspacesToDelete(instanceRunTriggers, workspaceRunTriggers)
	if len(deleteWorkspaces) > 0 {
		w.log.Info("Reconcile Run Triggers", "msg", "deleting run triggers workspaces from the workspace")
		err = r.removeRunTriggers(ctx, w, deleteWorkspaces)
		if err != nil {
			return err
		}
	}

	return nil
}
