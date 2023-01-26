// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
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

	if len(runTriggersNames) == 0 {
		return runTriggersIDs, nil
	}

	// Get Workspace IDs for the Run Triggers that we passed by Name
	workspaces, err := w.tfClient.Client.Workspaces.List(ctx, w.instance.Spec.Organization, &tfc.WorkspaceListOptions{})
	if err != nil {
		return nil, err
	}

	workspacesID := make(map[string]string)
	for _, ws := range workspaces.Items {
		workspacesID[ws.Name] = ws.ID
	}
	for _, n := range runTriggersNames {
		if id, ok := workspacesID[n]; ok {
			runTriggersIDs[id] = ""
		} else {
			return nil, fmt.Errorf("cannot find ID for Workspace %s", n)
		}
	}

	return runTriggersIDs, nil
}

func (r *WorkspaceReconciler) getRunTriggersWorkspace(ctx context.Context, w *workspaceInstance) (map[string]string, error) {
	runTriggers, err := w.tfClient.Client.RunTriggers.List(ctx, w.instance.Status.WorkspaceID, &tfc.RunTriggerListOptions{
		RunTriggerType: tfc.RunTriggerInbound,
		Include:        []tfc.RunTriggerIncludeOpt{tfc.RunTriggerSourceable},
	})
	if err != nil {
		return nil, err
	}
	rrt := make(map[string]string)
	for _, rt := range runTriggers.Items {
		// Sourceable.ID is an ID of the sourceable Workspace
		rrt[rt.Sourceable.ID] = rt.ID
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
		_, err := w.tfClient.Client.RunTriggers.Create(ctx, w.instance.Status.WorkspaceID, tfc.RunTriggerCreateOptions{
			Sourceable: &tfc.Workspace{ID: workspace},
		})
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
		err := w.tfClient.Client.RunTriggers.Delete(ctx, workspace)
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
