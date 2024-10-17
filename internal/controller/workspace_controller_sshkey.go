// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"errors"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
)

func (w *workspaceInstance) getSSHKeyID(ctx context.Context) (string, error) {
	if w.instance.Spec.SSHKey == nil {
		return "", errors.New("instance spec.SSHKey is nil")
	}

	if w.instance.Spec.SSHKey.Name != "" {
		w.log.Info("Reconcile SSH Key", "msg", "getting SSH key ID by name")
		listOpts := &tfc.SSHKeyListOptions{
			ListOptions: tfc.ListOptions{
				PageNumber: 1,
				PageSize:   maxPageSize,
			},
		}
		for {
			sshKeyList, err := w.tfClient.Client.SSHKeys.List(ctx, w.instance.Spec.Organization, listOpts)
			if err != nil {
				return "", err
			}

			for _, s := range sshKeyList.Items {
				if s.Name == w.instance.Spec.SSHKey.Name {
					return s.ID, nil
				}
			}

			if sshKeyList.NextPage == 0 {
				break
			}
			listOpts.PageNumber = sshKeyList.NextPage
		}
		return "", fmt.Errorf("ssh key ID was not found for SSH key name %q", w.instance.Spec.SSHKey.Name)
	}

	w.log.Info("Reconcile SSH Key", "msg", "getting SSH key ID from the spec.sshKey.ID")
	return w.instance.Spec.SSHKey.ID, nil
}

func (w *workspaceInstance) reconcileSSHKey(ctx context.Context, workspace *tfc.Workspace) error {
	w.log.Info("Reconcile SSH Key", "msg", "new reconciliation event")

	if w.instance.Spec.SSHKey == nil && workspace.SSHKey != nil {
		w.log.Info("Reconcile SSH Key", "msg", "removing the SSH key")
		if _, err := w.tfClient.Client.Workspaces.UnassignSSHKey(ctx, workspace.ID); err != nil {
			return err
		}
		w.instance.Status.SSHKeyID = ""
	}

	if w.instance.Spec.SSHKey != nil {
		if workspace.SSHKey == nil || workspace.SSHKey.ID != w.instance.Status.SSHKeyID {
			sshKeyID, err := w.getSSHKeyID(ctx)
			if err != nil {
				return err
			}
			w.log.Info("Reconcile SSH Key", "msg", "assigning the SSH key")
			opt := tfc.WorkspaceAssignSSHKeyOptions{
				SSHKeyID: &sshKeyID,
			}
			workspace, err = w.tfClient.Client.Workspaces.AssignSSHKey(ctx, workspace.ID, opt)
			if err != nil {
				return err
			}
			w.instance.Status.SSHKeyID = workspace.SSHKey.ID
		}
	}

	return nil
}
