// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	abstractions "github.com/microsoft/kiota-abstractions-go"

	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func (w *workspaceInstance) getSSHKeyID(ctx context.Context) (string, error) {
	if w.instance.Spec.SSHKey == nil {
		return "", errors.New("instance spec.SSHKey is nil")
	}

	if w.instance.Spec.SSHKey.Name != "" {
		w.log.Info("Reconcile SSH Key", "msg", "getting SSH key ID by name")

		query := &organizations.ItemSshKeysRequestBuilderGetQueryParameters{
			Pagenumber: new(int32(InitPageNumber)),
			Pagesize:   new(int32(MaxPageSize)),
		}

		for {
			sshKeyList, err := w.tfClient.Client.API.Organizations().ByOrganization_name(w.instance.Spec.Organization).SshKeys().Get(ctx,
				&abstractions.RequestConfiguration[organizations.ItemSshKeysRequestBuilderGetQueryParameters]{QueryParameters: query})
			if err != nil {
				return "", err
			}

			for _, s := range sshKeyList.GetData() {
				if s.GetAttributes() != nil && pointer.ValueOf(s.GetAttributes().GetName()) == w.instance.Spec.SSHKey.Name {
					return pointer.ValueOf(s.GetId()), nil
				}
			}

			next := nextPage(sshKeyList.GetMeta())
			if next == 0 {
				break
			}
			query.Pagenumber = &next
		}

		return "", fmt.Errorf("ssh key ID was not found for SSH key name %q", w.instance.Spec.SSHKey.Name)
	}

	w.log.Info("Reconcile SSH Key", "msg", "getting SSH key ID from the spec.sshKey.ID")
	return w.instance.Spec.SSHKey.ID, nil
}

// patchWorkspaceSSHKey assigns an SSH key to the workspace or, when sshKeyID is nil, unassigns the current one.
// The payload is the same as the one documented for the HCP Terraform API, where the SSH key ID is an attribute of the workspace resource.
func (w *workspaceInstance) patchWorkspaceSSHKey(ctx context.Context, workspaceID string, sshKeyID *string) (models.Workspacesable, error) {
	builder := w.tfClient.Client.API.Workspaces().ByWorkspace_id(workspaceID).Relationships().SshKey()
	ri, err := builder.ToPatchRequestInformation(ctx, models.NewSshKeysNullableIdentifierDocument(), nil)
	if err != nil {
		return nil, err
	}

	// A nil SSH key ID is serialized as JSON null which unassigns the SSH key.
	resp, err := sendRawJSON(ctx, builder.RequestAdapter, ri, map[string]any{
		"data": map[string]any{
			"type": "workspaces",
			"attributes": map[string]any{
				"id": sshKeyID,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	return resp.GetData(), nil
}

func (w *workspaceInstance) reconcileSSHKey(ctx context.Context, workspace models.Workspacesable) error {
	w.log.Info("Reconcile SSH Key", "msg", "new reconciliation event")

	var workspaceSSHKeyID string
	if k := workspace.GetRelationships().GetSshKey(); k != nil && k.GetData() != nil {
		workspaceSSHKeyID = pointer.ValueOf(k.GetData().GetId())
	}

	if w.instance.Spec.SSHKey == nil && workspaceSSHKeyID != "" {
		w.log.Info("Reconcile SSH Key", "msg", "removing the SSH key")
		if _, err := w.patchWorkspaceSSHKey(ctx, pointer.ValueOf(workspace.GetId()), nil); err != nil {
			return err
		}
		w.instance.Status.SSHKeyID = ""
	}

	if w.instance.Spec.SSHKey != nil {
		if workspaceSSHKeyID == "" || workspaceSSHKeyID != w.instance.Status.SSHKeyID {
			sshKeyID, err := w.getSSHKeyID(ctx)
			if err != nil {
				return err
			}

			w.log.Info("Reconcile SSH Key", "msg", "assigning the SSH key")
			updated, err := w.patchWorkspaceSSHKey(ctx, pointer.ValueOf(workspace.GetId()), &sshKeyID)
			if err != nil {
				return err
			}
			w.instance.Status.SSHKeyID = sshKeyID
			if updated != nil {
				if k := updated.GetRelationships().GetSshKey(); k != nil && k.GetData() != nil && pointer.ValueOf(k.GetData().GetId()) != "" {
					w.instance.Status.SSHKeyID = pointer.ValueOf(k.GetData().GetId())
				}
			}
		}
	}

	return nil
}
