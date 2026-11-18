// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"errors"

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	abstractions "github.com/microsoft/kiota-abstractions-go"

	corev1 "k8s.io/api/core/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func (r *WorkspaceReconciler) getWorkspaces(ctx context.Context, w *workspaceInstance) (map[string]string, error) {
	o := make(map[string]string)
	query := &organizations.ItemWorkspacesRequestBuilderGetQueryParameters{
		Pagesize:   new(int32(MaxPageSize)),
		Pagenumber: new(int32(InitPageNumber)),
	}

	for {
		ws, err := w.tfClient.Client.API.Organizations().ByOrganization_name(w.instance.Spec.Organization).Workspaces().Get(ctx,
			&abstractions.RequestConfiguration[organizations.ItemWorkspacesRequestBuilderGetQueryParameters]{QueryParameters: query})
		if err != nil {
			return map[string]string{}, err
		}

		for _, item := range ws.GetData() {
			id := pointer.ValueOf(item.GetId())
			o[id] = id
			o[pointer.ValueOf(item.GetAttributes().GetName())] = id
		}

		next := nextPage(ws.GetMeta())
		if next == 0 {
			break
		}
		query.Pagenumber = &next
	}

	return o, nil
}

// nameOrID returns Name or ID from the passed structure where only one of them is set
func nameOrID(instanceWorkspace *appv1alpha2.ConsumerWorkspace) string {
	if instanceWorkspace.Name != "" {
		return instanceWorkspace.Name
	}

	return instanceWorkspace.ID
}

func getWorkspaceID(workspaces map[string]string, instanceWorkspace *appv1alpha2.ConsumerWorkspace) (string, error) {
	k := nameOrID(instanceWorkspace)
	if v, ok := workspaces[k]; ok {
		return v, nil
	}
	return "", errors.New("workspace ID not found")
}

// getInstanceRemoteStateSharing returns IDs of the workspaces the remote state is shared with.
func (r *WorkspaceReconciler) getInstanceRemoteStateSharing(ctx context.Context, w *workspaceInstance) ([]string, error) {
	iw := []string{}

	if len(w.instance.Spec.RemoteStateSharing.Workspaces) == 0 {
		return iw, nil
	}

	workspaces, err := r.getWorkspaces(ctx, w)
	if err != nil {
		return iw, err
	}

	for _, workspace := range w.instance.Spec.RemoteStateSharing.Workspaces {
		wID, err := getWorkspaceID(workspaces, workspace)
		if err != nil {
			w.log.Error(err, "Reconcile Remote State Sharing", "msg", "failed to get workspace ID")
			r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileRemoteStateSharing", "Failed to get workspace ID")
			return iw, err
		}
		iw = append(iw, wID)
	}

	return iw, nil
}

func (r *WorkspaceReconciler) reconcileRemoteStateSharing(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Remote State Sharing", "msg", "new reconciliation event")

	if w.instance.Spec.RemoteStateSharing == nil {
		return nil
	}

	instanceRemoteStateSharing, err := r.getInstanceRemoteStateSharing(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Remote State Sharing", "msg", "failed to get instance remote state sharing workspace sources")
		return err
	}

	if len(instanceRemoteStateSharing) > 0 {
		consumers := make([]models.WorkspacesIdentifierArrayDocument_dataable, 0, len(instanceRemoteStateSharing))
		for _, id := range instanceRemoteStateSharing {
			c := models.NewWorkspacesIdentifierArrayDocument_data()
			c.SetId(new(id))
			c.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIERARRAYDOCUMENT_DATA_TYPE))
			consumers = append(consumers, c)
		}
		body := models.NewWorkspacesIdentifierArrayDocument()
		body.SetData(consumers)
		err = w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).Relationships().RemoteStateConsumers().Patch(ctx, body, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
