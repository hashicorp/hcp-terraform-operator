package controllers

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

func (r *WorkspaceReconciler) getRunTriggersSources(ctx context.Context, instance *appv1alpha2.Workspace) (map[string]string, error) {
	runTriggersIDs := make(map[string]string)
	runTriggersNames := []string{}

	for _, rt := range instance.Spec.RunTriggers {
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
	workspaces, err := r.tfClient.Client.Workspaces.List(ctx, instance.Spec.Organization, &tfc.WorkspaceListOptions{})
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

func (r *WorkspaceReconciler) getRunTriggersWorkspace(ctx context.Context, instance *appv1alpha2.Workspace) (map[string]string, error) {
	runTriggers, err := r.tfClient.Client.RunTriggers.List(ctx, instance.Status.WorkspaceID, &tfc.RunTriggerListOptions{
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

func (r *WorkspaceReconciler) addRunTriggers(ctx context.Context, instance *appv1alpha2.Workspace, workspaces map[string]string) error {
	if len(workspaces) == 0 {
		return nil
	}

	for w := range workspaces {
		_, err := r.tfClient.Client.RunTriggers.Create(ctx, instance.Status.WorkspaceID, tfc.RunTriggerCreateOptions{
			Sourceable: &tfc.Workspace{ID: w},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) removeRunTriggers(ctx context.Context, instance *appv1alpha2.Workspace, workspaces map[string]string) error {
	if len(workspaces) == 0 {
		return nil
	}

	for _, w := range workspaces {
		err := r.tfClient.Client.RunTriggers.Delete(ctx, w)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) reconcileRunTriggers(ctx context.Context, instance *appv1alpha2.Workspace) error {
	r.log.Info("Reconcile Run Triggers", "msg", "new reconciliation event")

	instanceRunTriggers, err := r.getRunTriggersSources(ctx, instance)
	if err != nil {
		return nil
	}
	workspaceRunTriggers, err := r.getRunTriggersWorkspace(ctx, instance)
	if err != nil {
		return nil
	}

	addWorkspaces := getWorkspacesToAdd(instanceRunTriggers, workspaceRunTriggers)
	if len(addWorkspaces) > 0 {
		r.log.Info("Reconcile Run Triggers", "msg", "adding run triggers workspaces to the workspace")
		err = r.addRunTriggers(ctx, instance, addWorkspaces)
		if err != nil {
			return err
		}
	}

	deleteWorkspaces := getWorkspacesToDelete(instanceRunTriggers, workspaceRunTriggers)
	if len(deleteWorkspaces) > 0 {
		r.log.Info("Reconcile Run Triggers", "msg", "deleting run triggers workspaces from the workspace")
		err = r.removeRunTriggers(ctx, instance, deleteWorkspaces)
		if err != nil {
			return err
		}
	}

	return nil
}
