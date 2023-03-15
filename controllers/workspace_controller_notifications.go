// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	tfc "github.com/hashicorp/go-tfe"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

func hasNotification(s []tfc.NotificationConfiguration, f tfc.NotificationConfiguration) (int, bool) {
	for i, v := range s {
		if cmp.Equal(v, f, cmpopts.IgnoreFields(tfc.NotificationConfiguration{}, "ID", "CreatedAt", "DeliveryResponses", "Triggers", "UpdatedAt", "EmailAddresses", "Subscribable", "EmailUsers")) {
			return i, true
		}

	}
	return -1, false
}

func notificationsDifference(a, b []tfc.NotificationConfiguration) []tfc.NotificationConfiguration {
	var d []tfc.NotificationConfiguration
	bc := make([]tfc.NotificationConfiguration, len(b))

	copy(bc, b)

	for _, av := range a {
		i, t := hasNotification(bc, av)
		if t {
			bc = appv1alpha2.RemoveFromSlice(bc, i)
		} else {
			d = append(d, av)
		}
	}

	return d
}

func (r *WorkspaceReconciler) getInstanceNotifications(ctx context.Context, w *workspaceInstance) ([]tfc.NotificationConfiguration, error) {
	if len(w.instance.Spec.Notifications) == 0 {
		return nil, nil
	}

	o := make([]tfc.NotificationConfiguration, len(w.instance.Spec.Notifications))

	for i, n := range w.instance.Spec.Notifications {
		o[i] = tfc.NotificationConfiguration{
			Name:            n.Name,
			DestinationType: n.Type,
			URL:             n.URL,
			Enabled:         n.Enabled,
			Token:           n.Token,
			// Triggers:        n.Triggers,
			// EmailAddresses:  n.EmailAddresses,
			// EmailUsers:      n.EmailUsers,
		}
	}

	return o, nil
}

func (r *WorkspaceReconciler) getWorkspaceNotifications(ctx context.Context, w *workspaceInstance) ([]tfc.NotificationConfiguration, error) {
	wn, err := w.tfClient.Client.NotificationConfigurations.List(ctx, w.instance.Status.WorkspaceID, &tfc.NotificationConfigurationListOptions{})
	if err != nil {
		return nil, err
	}

	o := make([]tfc.NotificationConfiguration, len(wn.Items))

	for i, n := range wn.Items {
		o[i] = tfc.NotificationConfiguration{
			ID:              n.ID,
			Name:            n.Name,
			DestinationType: n.DestinationType,
			URL:             n.URL,
			Enabled:         n.Enabled,
			Token:           n.Token,
			// Triggers:        n.Triggers,
			// EmailAddresses:  n.EmailAddresses,
			// EmailUsers:      n.EmailUsers,
		}
	}

	return o, nil
}

func getNotificationsToCreate(ctx context.Context, spec, ws []tfc.NotificationConfiguration) []tfc.NotificationConfiguration {
	return notificationsDifference(spec, ws)
}

func getNotificationsToUpdate(ctx context.Context, spec, ws []tfc.NotificationConfiguration) []tfc.NotificationConfiguration {
	o := []tfc.NotificationConfiguration{}

	if len(spec) == 0 || len(ws) == 0 {
		return o
	}

	// for ik, iv := range spec {
	// 	if wv, ok := ws[ik]; ok {
	// 		iv.ID = wv.ID
	// 		if !cmp.Equal(iv, wv, cmpopts.IgnoreFields(tfc.NotificationConfiguration{}, "Workspace")) {
	// 			o[ik] = iv
	// 		}
	// 	}
	// }

	return o
}

func getNotificationsToDelete(ctx context.Context, spec, ws []tfc.NotificationConfiguration) []tfc.NotificationConfiguration {
	return notificationsDifference(ws, spec)
}

func (r *WorkspaceReconciler) createNotifications(ctx context.Context, w *workspaceInstance, create []tfc.NotificationConfiguration) error {
	for _, n := range create {
		_, err := w.tfClient.Client.NotificationConfigurations.Create(ctx, w.instance.Status.WorkspaceID, tfc.NotificationConfigurationCreateOptions{
			Name:            &n.Name,
			DestinationType: &n.DestinationType,
			URL:             &n.URL,
			Enabled:         &n.Enabled,
		})
		if err != nil {
			w.log.Error(err, "Reconcile Notifications", "msg", "failed to create a new notification")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) updateNotifications(ctx context.Context, w *workspaceInstance, update []tfc.NotificationConfiguration) error {
	// for _, v := range update {
	// 	w.log.Info("Reconcile Notifications", "msg", "updating notificaion")
	// 	_, err := w.tfClient.Client.NotificationConfigurations.Update(ctx, w.instance.Status.WorkspaceID, v.ID, tfc.NotificationConfigurations{
	// 		Type:             "workspace-task",
	// 		EnforcementLevel: v.EnforcementLevel,
	// 		Stage:            &v.Stage,
	// 	})
	// 	if err != nil {
	// 		w.log.Error(err, "Reconcile Notifications", "msg", "failed to update notificaion")
	// 		return err
	// 	}
	// }

	return nil
}

func (r *WorkspaceReconciler) deleteNotifications(ctx context.Context, w *workspaceInstance, delete []tfc.NotificationConfiguration) error {
	for _, d := range delete {
		err := w.tfClient.Client.NotificationConfigurations.Delete(ctx, d.ID)
		if err != nil {
			w.log.Error(err, "Reconcile Notifications", "msg", "failed to delete notificaions")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) reconcileNotifications(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Notifications", "msg", "new reconciliation event")

	spec, err := r.getInstanceNotifications(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Notifications", "msg", "failed to get instance notificaions")
		return err
	}

	ws, err := r.getWorkspaceNotifications(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Notifications", "msg", "failed to get workspace notifications")
		return err
	}

	create := getNotificationsToCreate(ctx, spec, ws)
	if len(create) > 0 {
		w.log.Info("Reconcile Notifications", "msg", fmt.Sprintf("creating %d notifications", len(create)))
		err := r.createNotifications(ctx, w, create)
		if err != nil {
			w.log.Error(err, "Reconcile Notifications", "msg", "failed to create a notificaion")
			return err
		}
	}

	update := getNotificationsToUpdate(ctx, spec, ws)
	if len(update) > 0 {
		w.log.Info("Reconcile Notifications", "msg", fmt.Sprintf("updating %d notifications", len(update)))
		err := r.updateNotifications(ctx, w, update)
		if err != nil {
			w.log.Error(err, "Reconcile Notifications", "msg", "failed to update a notificaion")
			return err
		}
	}

	delete := getNotificationsToDelete(ctx, spec, ws)
	if len(delete) > 0 {
		w.log.Info("Reconcile Notifications", "msg", fmt.Sprintf("deleting %d notifications", len(delete)))
		err := r.deleteNotifications(ctx, w, delete)
		if err != nil {
			w.log.Error(err, "Reconcile Notifications", "msg", "failed to delete a notificaion")
			return err
		}
	}

	return nil
}
