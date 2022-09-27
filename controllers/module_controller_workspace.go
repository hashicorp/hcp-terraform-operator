package controllers

import (
	"context"
	"errors"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

func (r *ModuleReconciler) getWorkspaceByName(ctx context.Context, instance *appv1alpha2.Module) (*tfc.Workspace, error) {
	specWorkspaceName := instance.Spec.Workspace.Name

	return r.tfClient.Client.Workspaces.Read(ctx, instance.Spec.Organization, specWorkspaceName)
}

func (r *ModuleReconciler) getWorkspaceByID(ctx context.Context, instance *appv1alpha2.Module) (*tfc.Workspace, error) {
	specWorkspaceID := instance.Spec.Workspace.ID

	return r.tfClient.Client.Workspaces.Read(ctx, instance.Spec.Organization, specWorkspaceID)
}

func (r *ModuleReconciler) getWorkspace(ctx context.Context, instance *appv1alpha2.Module) (*tfc.Workspace, error) {
	specWorkspace := instance.Spec.Workspace

	if specWorkspace == nil {
		return &tfc.Workspace{}, errors.New("instance.Spec.Workspace is nil")
	}

	if specWorkspace.Name != "" {
		r.log.Info("Reconcile Module Workspace", "msg", "getting workspace by name")
		return r.getWorkspaceByName(ctx, instance)
	}

	r.log.Info("Reconcile Module Workspace", "msg", "getting workspace by ID")
	return r.getWorkspaceByID(ctx, instance)
}
