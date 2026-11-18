// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"errors"

	"github.com/hashicorp/go-tfe/v2/api/models"
)

func (r *ModuleReconciler) getWorkspaceByName(ctx context.Context, m *moduleInstance) (models.Workspacesable, error) {
	resp, err := m.tfClient.Client.API.Organizations().
		ByOrganization_name(m.instance.Spec.Organization).
		Workspaces().
		ByWorkspace_name(m.instance.Spec.Workspace.Name).
		Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	return resp.GetData(), nil
}

func (r *ModuleReconciler) getWorkspaceByID(ctx context.Context, m *moduleInstance) (models.Workspacesable, error) {
	resp, err := m.tfClient.Client.API.Workspaces().ByWorkspace_id(m.instance.Spec.Workspace.ID).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	return resp.GetData(), nil
}

func (r *ModuleReconciler) getWorkspace(ctx context.Context, m *moduleInstance) (models.Workspacesable, error) {
	specWorkspace := m.instance.Spec.Workspace

	if specWorkspace == nil {
		return models.NewWorkspaces(), errors.New("instance.Spec.Workspace is nil")
	}

	if specWorkspace.Name != "" {
		m.log.Info("Reconcile Module Workspace", "msg", "getting workspace by name")
		return r.getWorkspaceByName(ctx, m)
	}

	m.log.Info("Reconcile Module Workspace", "msg", "getting workspace by ID")
	return r.getWorkspaceByID(ctx, m)
}
