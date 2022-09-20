package controllers

import (
	"context"
	"fmt"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"

	tfc "github.com/hashicorp/go-tfe"
)

func (r *WorkspaceReconciler) getAgentPoolIDByName(ctx context.Context, instance *appv1alpha2.Workspace) (string, error) {
	agentPoolName := instance.Spec.AgentPool.Name

	agentPoolIDs, err := r.tfClient.Client.AgentPools.List(ctx, instance.Spec.Organization, &tfc.AgentPoolListOptions{
		Query: agentPoolName,
	})
	if err != nil {
		return "", err
	}

	for _, a := range agentPoolIDs.Items {
		if a.Name == agentPoolName {
			return a.ID, nil
		}
	}

	return "", fmt.Errorf("agent pool ID not found for agent pool name %q", agentPoolName)
}

func (r *WorkspaceReconciler) getAgentPoolID(ctx context.Context, instance *appv1alpha2.Workspace) (string, error) {
	specAgentPool := instance.Spec.AgentPool

	if specAgentPool.Name != "" {
		r.log.Info("Reconcile Agent Pool", "msg", "getting agent pool ID by name")
		return r.getAgentPoolIDByName(ctx, instance)
	}

	r.log.Info("Reconcile Agent Pool", "msg", "getting agent pool ID from the spec.AgentPool.ID")
	return specAgentPool.ID, nil
}
