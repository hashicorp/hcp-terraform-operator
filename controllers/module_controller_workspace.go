// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"errors"

	tfc "github.com/hashicorp/go-tfe"
)

func (r *ModuleReconciler) getWorkspaceByName(ctx context.Context, m *moduleInstance) (*tfc.Workspace, error) {
	return m.tfClient.Client.Workspaces.Read(ctx, m.instance.Spec.Organization, m.instance.Spec.Workspace.Name)
}

func (r *ModuleReconciler) getWorkspaceByID(ctx context.Context, m *moduleInstance) (*tfc.Workspace, error) {
	return m.tfClient.Client.Workspaces.Read(ctx, m.instance.Spec.Organization, m.instance.Spec.Workspace.ID)
}

func (r *ModuleReconciler) getWorkspace(ctx context.Context, m *moduleInstance) (*tfc.Workspace, error) {
	specWorkspace := m.instance.Spec.Workspace

	if specWorkspace == nil {
		return &tfc.Workspace{}, errors.New("instance.Spec.Workspace is nil")
	}

	if specWorkspace.Name != "" {
		m.log.Info("Reconcile Module Workspace", "msg", "getting workspace by name")
		return r.getWorkspaceByName(ctx, m)
	}

	m.log.Info("Reconcile Module Workspace", "msg", "getting workspace by ID")
	return r.getWorkspaceByID(ctx, m)
}
