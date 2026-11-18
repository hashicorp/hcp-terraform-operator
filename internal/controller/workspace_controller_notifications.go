// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	"github.com/hashicorp/go-tfe/v2/api/workspaces"
	abstractions "github.com/microsoft/kiota-abstractions-go"

	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
	"github.com/hashicorp/hcp-terraform-operator/internal/slice"
)

type notificationConfiguration struct {
	ID              string
	Name            string
	DestinationType string
	URL             string
	Enabled         bool
	Token           string
	Triggers        []string
	EmailAddresses  []string
	EmailUsers      []string // user IDs
}

func hasNotificationItem(notifications []notificationConfiguration, notification notificationConfiguration) (int, bool) {
	for i, n := range notifications {
		if notification.Name == n.Name && notification.DestinationType == n.DestinationType {
			return i, true
		}
	}
	return -1, false
}

// getOrgMembers returns a map of user email -> user ID.
func (r *WorkspaceReconciler) getOrgMembers(ctx context.Context, w *workspaceInstance) (map[string]string, error) {
	emailUsers := make(map[string]string)
	emails := make([]string, 0)
	for _, n := range w.instance.Spec.Notifications {
		emails = append(emails, n.EmailUsers...)
	}
	if len(emails) == 0 {
		return emailUsers, nil
	}
	query := &organizations.ItemOrganizationMembershipsRequestBuilderGetQueryParameters{
		Filteremail: new(strings.Join(emails, ",")),
		Pagesize:    new(int32(MaxPageSize)),
		Pagenumber:  new(int32(InitPageNumber)),
	}
	for {
		members, err := w.tfClient.Client.API.Organizations().ByOrganization_name(w.instance.Spec.Organization).OrganizationMemberships().Get(ctx,
			&abstractions.RequestConfiguration[organizations.ItemOrganizationMembershipsRequestBuilderGetQueryParameters]{QueryParameters: query})
		if err != nil {
			return nil, err
		}
		for _, m := range members.GetData() {
			if m.GetAttributes() == nil || m.GetRelationships() == nil || m.GetRelationships().GetUser() == nil || m.GetRelationships().GetUser().GetData() == nil {
				continue
			}
			emailUsers[pointer.ValueOf(m.GetAttributes().GetEmail())] = pointer.ValueOf(m.GetRelationships().GetUser().GetData().GetId())
		}
		next := nextPage(members.GetMeta())
		if next == 0 {
			break
		}
		query.Pagenumber = &next
	}
	return emailUsers, nil
}

func (r *WorkspaceReconciler) getInstanceNotifications(ctx context.Context, w *workspaceInstance) ([]notificationConfiguration, error) {
	if len(w.instance.Spec.Notifications) == 0 {
		return []notificationConfiguration{}, nil
	}

	orgEmailUsers, err := r.getOrgMembers(ctx, w)
	if err != nil {
		return []notificationConfiguration{}, err
	}

	notifications := make([]notificationConfiguration, len(w.instance.Spec.Notifications))

	for i, notification := range w.instance.Spec.Notifications {
		var emailUsers []string
		for _, emailUser := range notification.EmailUsers {
			if v, ok := orgEmailUsers[emailUser]; ok {
				emailUsers = append(emailUsers, v)
			} else {
				w.log.Error(err, "Reconcile Notifications", "msg", fmt.Sprintf("user %s was not found", emailUser))
				return []notificationConfiguration{}, fmt.Errorf("user %s was not found", emailUser)
			}
		}
		triggers := make([]string, len(notification.Triggers))
		for i, t := range notification.Triggers {
			triggers[i] = string(t)
		}
		notifications[i] = notificationConfiguration{
			Name:            notification.Name,
			DestinationType: string(notification.Type),
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

func (r *WorkspaceReconciler) getWorkspaceNotifications(ctx context.Context, w *workspaceInstance) ([]notificationConfiguration, error) {
	var notifications []notificationConfiguration

	query := &workspaces.ItemNotificationConfigurationsRequestBuilderGetQueryParameters{
		Pagesize:   new(int32(MaxPageSize)),
		Pagenumber: new(int32(InitPageNumber)),
	}
	for {
		wn, err := w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).NotificationConfigurations().Get(ctx,
			&abstractions.RequestConfiguration[workspaces.ItemNotificationConfigurationsRequestBuilderGetQueryParameters]{QueryParameters: query})
		if err != nil {
			return nil, err
		}
		for _, d := range wn.GetData() {
			n := notificationConfiguration{ID: pointer.ValueOf(d.GetId())}
			if a := d.GetAttributes(); a != nil {
				n.Name = pointer.ValueOf(a.GetName())
				if v := a.GetDestinationType(); v != nil {
					n.DestinationType = v.String()
				}
				n.URL = pointer.ValueOf(a.GetUrl())
				n.Enabled = pointer.ValueOf(a.GetEnabled())
				n.Token = pointer.ValueOf(a.GetToken())
				n.Triggers = a.GetTriggers()
				n.EmailAddresses = a.GetEmailAddresses()
			}
			if rel := d.GetRelationships(); rel != nil && rel.GetUsers() != nil {
				for _, u := range rel.GetUsers().GetData() {
					n.EmailUsers = append(n.EmailUsers, pointer.ValueOf(u.GetId()))
				}
			}
			notifications = append(notifications, n)
		}
		next := nextPage(wn.GetMeta())
		if next == 0 {
			break
		}
		query.Pagenumber = &next
	}

	return notifications, nil
}

func (w *workspaceInstance) createNotification(ctx context.Context, notification notificationConfiguration) error {
	w.log.Info("Reconcile Notifications", "msg", "creating notification")

	attrs := models.NewNotificationConfigurations_attributes()
	attrs.SetName(new(notification.Name))
	// go-tfe v1 sent the destination type as is and left the validation to the API.
	if dt, perr := models.ParseNotificationConfigurations_attributes_destinationType(notification.DestinationType); perr == nil && dt != nil {
		attrs.SetDestinationType(dt.(*models.NotificationConfigurations_attributes_destinationType))
	} else {
		attrs.SetAdditionalData(map[string]any{"destination-type": notification.DestinationType})
	}
	attrs.SetUrl(new(notification.URL))
	attrs.SetEnabled(new(notification.Enabled))
	attrs.SetToken(new(notification.Token))
	attrs.SetTriggers(notification.Triggers)
	attrs.SetEmailAddresses(notification.EmailAddresses)

	rels := models.NewNotificationConfigurations_relationships()
	// go-tfe v1 omitted the 'users' relationship when there are no users to send.
	if len(notification.EmailUsers) > 0 {
		users := make([]models.UsersIdentifierable, 0, len(notification.EmailUsers))
		for _, id := range notification.EmailUsers {
			user := models.NewUsersIdentifier()
			user.SetId(new(id))
			user.SetTypeEscaped(new(models.USERS_USERSIDENTIFIER_TYPE))
			users = append(users, user)
		}
		usersRel := models.NewUsersHasMany()
		usersRel.SetData(users)
		rels.SetUsers(usersRel)
	}

	data := models.NewNotificationConfigurations()
	data.SetTypeEscaped(new(models.NOTIFICATIONCONFIGURATIONS_NOTIFICATIONCONFIGURATIONS_TYPE))
	data.SetAttributes(attrs)
	data.SetRelationships(rels)
	body := models.NewNotificationConfigurationsEnvelope()
	body.SetData(data)

	_, err := w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).NotificationConfigurations().Post(ctx, body, nil)
	if err != nil {
		w.log.Error(err, "Reconcile Notifications", "msg", fmt.Sprintf("failed to create a new notification %q", notification.ID))
		return err
	}

	return nil
}

func (w *workspaceInstance) updateNotification(ctx context.Context, sn, wn notificationConfiguration) error {
	if !cmp.Equal(sn, wn, cmpopts.IgnoreFields(notificationConfiguration{}, "ID")) {
		w.log.Info("Reconcile Notifications", "msg", fmt.Sprintf("updating notification %q", wn.ID))

		attrs := models.NewNotificationConfigurations_attributes()
		attrs.SetName(new(sn.Name))
		attrs.SetUrl(new(sn.URL))
		attrs.SetEnabled(new(sn.Enabled))
		attrs.SetToken(new(sn.Token))
		attrs.SetTriggers(sn.Triggers)
		attrs.SetEmailAddresses(sn.EmailAddresses)

		rels := models.NewNotificationConfigurations_relationships()
		// go-tfe v1 omitted the 'users' relationship when there are no users to send.
		if len(sn.EmailUsers) > 0 {
			users := make([]models.UsersIdentifierable, 0, len(sn.EmailUsers))
			for _, id := range sn.EmailUsers {
				user := models.NewUsersIdentifier()
				user.SetId(new(id))
				user.SetTypeEscaped(new(models.USERS_USERSIDENTIFIER_TYPE))
				users = append(users, user)
			}
			usersRel := models.NewUsersHasMany()
			usersRel.SetData(users)
			rels.SetUsers(usersRel)
		}

		data := models.NewNotificationConfigurations()
		data.SetTypeEscaped(new(models.NOTIFICATIONCONFIGURATIONS_NOTIFICATIONCONFIGURATIONS_TYPE))
		data.SetAttributes(attrs)
		data.SetRelationships(rels)
		body := models.NewNotificationConfigurationsEnvelope()
		body.SetData(data)

		_, err := w.tfClient.Client.API.NotificationConfigurations().ByNotification_configuration_id(wn.ID).Patch(ctx, body, nil)
		if err != nil {
			w.log.Error(err, "Reconcile Notifications", "msg", fmt.Sprintf("failed to update notification %q", wn.ID))
			return err
		}
	}

	return nil
}

func (w *workspaceInstance) deleteNotification(ctx context.Context, notification notificationConfiguration) error {
	w.log.Info("Reconcile Notifications", "msg", fmt.Sprintf("deleting notification %q", notification.ID))
	if err := w.tfClient.Client.API.NotificationConfigurations().ByNotification_configuration_id(notification.ID).Delete(ctx, nil); err != nil {
		w.log.Error(err, "Reconcile Notifications", "msg", fmt.Sprintf("failed to delete notification %q", notification.ID))
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
		w.log.Error(err, "Reconcile Notifications", "msg", "failed to get instance notifications")
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
