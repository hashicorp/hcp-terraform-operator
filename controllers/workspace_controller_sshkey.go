package controllers

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

func (r *WorkspaceReconciler) getSSHKeyIDByName(ctx context.Context, instance *appv1alpha2.Workspace) (string, error) {
	SSHKeyName := instance.Spec.SSHKey.Name

	SSHKeys, err := r.tfClient.Client.SSHKeys.List(ctx, instance.Spec.Organization, &tfc.SSHKeyListOptions{})
	if err != nil {
		return "", err
	}

	for _, s := range SSHKeys.Items {
		if s.Name == SSHKeyName {
			return s.ID, nil
		}
	}

	return "", fmt.Errorf("ssh key ID was not found for ssh key name %q", SSHKeyName)
}

func (r *WorkspaceReconciler) getSSHKeyID(ctx context.Context, instance *appv1alpha2.Workspace) (string, error) {
	specSSHKey := instance.Spec.SSHKey

	if specSSHKey.Name != "" {
		r.log.Info("Reconcile SSH Key", "msg", "getting ssh key ID by name")
		return r.getSSHKeyIDByName(ctx, instance)
	}

	r.log.Info("Reconcile SSH Key", "msg", "getting ssh key ID from the spec.sshKey.ID")
	return specSSHKey.ID, nil
}

func (r *WorkspaceReconciler) reconcileSSHKey(ctx context.Context, instance *appv1alpha2.Workspace, workspace *tfc.Workspace) (*tfc.Workspace, error) {
	spec := instance.Spec

	if spec.SSHKey == nil && workspace.SSHKey != nil {
		r.log.Info("Reconcile SSH Key", "msg", "unassigning the ssh key")
		return r.tfClient.Client.Workspaces.UnassignSSHKey(ctx, workspace.ID)
	}

	if spec.SSHKey != nil {
		SSHKeyID, err := r.getSSHKeyID(ctx, instance)
		if err != nil {
			return workspace, err
		} else {
			r.log.Info("Reconcile SSH Key", "msg", "assigning the ssh key")
			return r.tfClient.Client.Workspaces.AssignSSHKey(ctx, workspace.ID, tfc.WorkspaceAssignSSHKeyOptions{
				SSHKeyID: tfc.String(SSHKeyID),
			})
		}
	}

	return workspace, nil
}
