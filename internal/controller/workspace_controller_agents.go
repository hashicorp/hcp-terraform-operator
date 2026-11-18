// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-tfe/v2/api/organizations"
	abstractions "github.com/microsoft/kiota-abstractions-go"

	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func (r *WorkspaceReconciler) getAgentPoolIDByName(ctx context.Context, w *workspaceInstance) (string, error) {
	agentPoolName := w.instance.Spec.AgentPool.Name
	query := &organizations.ItemAgentPoolsRequestBuilderGetQueryParameters{
		Q:        &agentPoolName,
		Pagesize: new(int32(MaxPageSize)),
	}

	for {
		agentPools, err := w.tfClient.Client.API.Organizations().ByOrganization_name(w.instance.Spec.Organization).AgentPools().Get(ctx,
			&abstractions.RequestConfiguration[organizations.ItemAgentPoolsRequestBuilderGetQueryParameters]{QueryParameters: query})
		if err != nil {
			return "", err
		}

		for _, a := range agentPools.GetData() {
			if a.GetAttributes() != nil && pointer.ValueOf(a.GetAttributes().GetName()) == agentPoolName {
				return pointer.ValueOf(a.GetId()), nil
			}
		}

		next := nextPage(agentPools.GetMeta())
		if next == 0 {
			break
		}
		query.Pagenumber = &next
	}

	return "", fmt.Errorf("agent pool ID not found for agent pool name %q", agentPoolName)
}

func (r *WorkspaceReconciler) getAgentPoolID(ctx context.Context, w *workspaceInstance) (string, error) {
	specAgentPool := w.instance.Spec.AgentPool
	if specAgentPool == nil {
		return "", fmt.Errorf("'spec.agentPool' is not set")
	}

	if specAgentPool.Name != "" {
		w.log.Info("Reconcile Agent Pool", "msg", "getting agent pool ID by name")
		return r.getAgentPoolIDByName(ctx, w)
	}

	w.log.Info("Reconcile Agent Pool", "msg", "getting agent pool ID from the spec.AgentPool.ID")
	return specAgentPool.ID, nil
}
