// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	tfc "github.com/hashicorp/go-tfe"

	"github.com/hashicorp/hcp-terraform-operator/internal/slice"
)

func hasNotificationItem(notificaions []tfc.NotificationConfiguration, notificaion tfc.NotificationConfiguration) (int, bool) {
	for i, n := range notificaions {
		if notificaion.Name == n.Name && notificaion.DestinationType == n.DestinationType {
			return i, true
		}
	}
	return -1, false
}

func (r *WorkspaceReconciler) getOrgMembers(ctx context.Context, w *workspaceInstance) (map[string]*tfc.User, error) {
	emailUsers := make(map[string]*tfc.User)
	emails := make([]string, 0)
	for _, n := range w.instance.Spec.Notifications {
		emails = append(emails, n.EmailUsers...)
	}
	listOpts := &tfc.OrganizationMembershipListOptions{
		Emails: emails,
		ListOptions: tfc.ListOptions{
			PageSize: maxPageSize,
		},
	}
	for {
		members, err := w.tfClient.Client.OrganizationMemberships.List(ctx, w.instance.Spec.Organization, listOpts)
		if err != nil {
			return nil, err
		}
		for _, m := range members.Items {
			emailUsers[m.Email] = &tfc.User{ID: m.User.ID}
		}
		if members.NextPage == 0 {
			break
		}
		listOpts.PageNumber = members.NextPage
	}
	return emailUsers, nil
}

func (r *WorkspaceReconciler) getInstanceNotifications(ctx context.Context, w *workspaceInstance) ([]tfc.NotificationConfiguration, error) {
	if len(w.instance.Spec.Notifications) == 0 {
		return []tfc.NotificationConfiguration{}, nil
	}

	orgEmailUsers, err := r.getOrgMembers(ctx, w)
	if err != nil {
		return []tfc.NotificationConfiguration{}, err
	}

	notifications := make([]tfc.NotificationConfiguration, len(w.instance.Spec.Notifications))

	for i, notification := range w.instance.Spec.Notifications {
		var emailUsers []*tfc.User
		for _, emailUser := range notification.EmailUsers {
			if v, ok := orgEmailUsers[emailUser]; ok && v != nil {
				emailUsers = append(emailUsers, v)
			} else {
				w.log.Error(err, "Reconcile Notifications", "msg", fmt.Sprintf("user %s was not found", emailUser))
			}
		}
		triggers := make([]string, len(notification.Triggers))
		for i, t := range notification.Triggers {
			triggers[i] = string(t)
		}
		notifications[i] = tfc.NotificationConfiguration{
			Name:            notification.Name,
			DestinationType: notification.Type,
			URL:             notification.URL,
			Enabled:         notification.Enabled,
			Token:           notification.Token,
			Triggers:        triggers,
			EmailAddresses:  notification.EmailAddresses,
			EmailUsers:      emailUsers,
		}
	}

	return notifications, nil
}

func (r *WorkspaceReconciler) getWorkspaceNotifications(ctx context.Context, w *workspaceInstance) ([]tfc.NotificationConfiguration, error) {
	var notifications []tfc.NotificationConfiguration

	listOpts := &tfc.NotificationConfigurationListOptions{
		ListOptions: tfc.ListOptions{
			PageSize: maxPageSize,
		},
	}
	for {
		wn, err := w.tfClient.Client.NotificationConfigurations.List(ctx, w.instance.Status.WorkspaceID, listOpts)
		if err != nil {
			return nil, err
		}
		for _, notification := range wn.Items {
			notifications = append(notifications, tfc.NotificationConfiguration{
				ID:              notification.ID,
				Name:            notification.Name,
				DestinationType: notification.DestinationType,
				URL:             notification.URL,
				Enabled:         notification.Enabled,
				Token:           notification.Token,
				Triggers:        notification.Triggers,
				EmailAddresses:  notification.EmailAddresses,
				EmailUsers:      notification.EmailUsers,
			})
		}
		if wn.NextPage == 0 {
			break
		}
		listOpts.PageNumber = wn.NextPage
	}

	return notifications, nil
}

func (w *workspaceInstance) createNotification(ctx context.Context, notificaion tfc.NotificationConfiguration) error {
	w.log.Info("Reconcile Notifications", "msg", "creating notificaion")
	nt := make([]tfc.NotificationTriggerType, len(notificaion.Triggers))
	for i, t := range notificaion.Triggers {
		nt[i] = tfc.NotificationTriggerType(t)
	}
	_, err := w.tfClient.Client.NotificationConfigurations.Create(ctx, w.instance.Status.WorkspaceID, tfc.NotificationConfigurationCreateOptions{
		Name:            &notificaion.Name,
		DestinationType: &notificaion.DestinationType,
		URL:             &notificaion.URL,
		Enabled:         &notificaion.Enabled,
		Token:           &notificaion.Token,
		Triggers:        nt,
		EmailUsers:      notificaion.EmailUsers,
		EmailAddresses:  notificaion.EmailAddresses,
	})
	if err != nil {
		w.log.Error(err, "Reconcile Notifications", "msg", fmt.Sprintf("failed to create a new notification %q", notificaion.ID))
		return err
	}

	return nil
}

func (w *workspaceInstance) updateNotification(ctx context.Context, sn, wn tfc.NotificationConfiguration) error {
	w.log.Info("Reconcile Notifications", "msg", fmt.Sprintf("updating notificaion %q", wn.ID))
	if !cmp.Equal(sn, wn, cmpopts.IgnoreFields(tfc.NotificationConfiguration{}, "ID", "CreatedAt", "DeliveryResponses", "UpdatedAt", "Subscribable")) {
		triggers := make([]tfc.NotificationTriggerType, len(sn.Triggers))
		for i, t := range sn.Triggers {
			triggers[i] = tfc.NotificationTriggerType(t)
		}
		_, err := w.tfClient.Client.NotificationConfigurations.Update(ctx, wn.ID, tfc.NotificationConfigurationUpdateOptions{
			Name:           &sn.Name,
			Enabled:        &sn.Enabled,
			Token:          &sn.Token,
			Triggers:       triggers,
			URL:            &sn.URL,
			EmailAddresses: sn.EmailAddresses,
			EmailUsers:     sn.EmailUsers,
		})
		if err != nil {
			w.log.Error(err, "Reconcile Notifications", "msg", fmt.Sprintf("failed to update notificaion %q", wn.ID))
			return err
		}
	}

	return nil
}

func (w *workspaceInstance) deleteNotification(ctx context.Context, notification tfc.NotificationConfiguration) error {
	w.log.Info("Reconcile Notifications", "msg", fmt.Sprintf("deleting notificaion %q", notification.ID))
	if err := w.tfClient.Client.NotificationConfigurations.Delete(ctx, notification.ID); err != nil {
		w.log.Error(err, "Reconcile Notifications", "msg", fmt.Sprintf("failed to delete notificaion %q", notification.ID))
		return err
	}

	return nil
}

func (r *WorkspaceReconciler) reconcileNotifications(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Notifications", "msg", "new reconciliation event")

	workspaceNotifications, err := r.getWorkspaceNotifications(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Notifications", "msg", "failed to get workspace notifications")
		return err
	}

	specNotifications, err := r.getInstanceNotifications(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Notifications", "msg", "failed to get instance notificaions")
		return err
	}

	if len(specNotifications) == 0 && len(workspaceNotifications) == 0 {
		w.log.Info("Reconcile Notifications", "msg", "there are no notifications both in spec and workspace")
		return nil
	}

	w.log.Info("Reconcile Notifications", "msg", fmt.Sprintf("there are %d notifications in spec", len(specNotifications)))
	w.log.Info("Reconcile Notifications", "msg", fmt.Sprintf("there are %d notifications in workspace", len(workspaceNotifications)))

	for _, sn := range specNotifications {
		if i, ok := hasNotificationItem(workspaceNotifications, sn); ok {
			if err := w.updateNotification(ctx, sn, workspaceNotifications[i]); err != nil {
				return err
			}
			workspaceNotifications = slice.RemoveFromSlice(workspaceNotifications, i)
		} else {
			if err := w.createNotification(ctx, sn); err != nil {
				return err
			}
		}
	}

	for _, wn := range workspaceNotifications {
		if err := w.deleteNotification(ctx, wn); err != nil {
			return err
		}
	}

	return nil
}
