// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	"github.com/hashicorp/go-tfe/v2/api/organizations/item/organizationmemberships"
	"github.com/hashicorp/go-tfe/v2/api/workspaces"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/controller"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName types.NamespacedName
		workspace      string

		memberEmail  string
		memberEmail2 string
		memberID     string
		memberID2    string
	)

	BeforeAll(func() {

		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		memberEmail = fmt.Sprintf("kubernetes-operator-member-%v@hashicorp.com", randomNumber())
		memberEmail2 = fmt.Sprintf("kubernetes-operator-member-2-%v@hashicorp.com", randomNumber())
		memberID = createOrgMember(memberEmail)
		memberID2 = createOrgMember(memberEmail2)
		// Create a new workspace object for each test
		instance = &appv1alpha2.Workspace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "app.terraform.io/v1alpha2",
				Kind:       "Workspace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              namespacedName.Name,
				Namespace:         namespacedName.Namespace,
				DeletionTimestamp: nil,
				Finalizers:        []string{},
			},
			Spec: appv1alpha2.WorkspaceSpec{
				Organization: organization,
				Token: appv1alpha2.Token{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretNamespacedName.Name,
						},
						Key: secretKey,
					},
				},
				Name:          workspace,
				Description:   "Notifications",
				Notifications: []appv1alpha2.Notification{},
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance)
		Expect(tfClient.API.OrganizationMemberships().ByOrganization_membership_id(memberID).Delete(ctx, nil)).Should(Succeed())
		Expect(tfClient.API.OrganizationMemberships().ByOrganization_membership_id(memberID2).Delete(ctx, nil)).Should(Succeed())
	})

	Context("Notifications", func() {
		It("can create single notification", func() {
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name: "slack",
				Type: models.SLACK_NOTIFICATIONCONFIGURATIONS_ATTRIBUTES_DESTINATIONTYPE.String(),
				URL:  webhookURL,
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		It("can create multiple notifications", func() {
			if cloudEndpoint != tfcDefaultAddress {
				Skip("Does not run against TFE, skip this test")
			}
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name: "slack",
				Type: models.SLACK_NOTIFICATIONCONFIGURATIONS_ATTRIBUTES_DESTINATIONTYPE.String(),
				URL:  webhookURL,
			})
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name:       "email",
				Type:       models.EMAIL_NOTIFICATIONCONFIGURATIONS_ATTRIBUTES_DESTINATIONTYPE.String(),
				EmailUsers: []string{memberEmail},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		It("can re-create notifications", func() {
			if cloudEndpoint != tfcDefaultAddress {
				Skip("Does not run against TFE, skip this test")
			}
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name: "slack",
				Type: models.SLACK_NOTIFICATIONCONFIGURATIONS_ATTRIBUTES_DESTINATIONTYPE.String(),
				URL:  webhookURL,
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			// Validate reconciliation
			isNotificationsReconciled(instance)

			// Delete notifications manually
			notifications, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).NotificationConfigurations().Get(ctx, nil)
			Expect(err).Should(Succeed())
			Expect(notifications).ShouldNot(BeNil())
			for _, n := range notifications.GetData() {
				Expect(tfClient.API.NotificationConfigurations().ByNotification_configuration_id(pointer.ValueOf(n.GetId())).Delete(ctx, nil)).Should(Succeed())
			}

			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		It("can update notifications", func() {
			if cloudEndpoint != tfcDefaultAddress {
				Skip("Does not run against TFE, skip this test")
			}
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name:       "email",
				Type:       models.EMAIL_NOTIFICATIONCONFIGURATIONS_ATTRIBUTES_DESTINATIONTYPE.String(),
				EmailUsers: []string{memberEmail},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			// Validate reconciliation
			isNotificationsReconciled(instance)

			// Add a new email user
			instance.Spec.Notifications[0].EmailUsers = append(instance.Spec.Notifications[0].EmailUsers, memberEmail2)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		It("can delete notifications", func() {
			if cloudEndpoint != tfcDefaultAddress {
				Skip("Does not run against TFE, skip this test")
			}
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name:       "email",
				Type:       models.EMAIL_NOTIFICATIONCONFIGURATIONS_ATTRIBUTES_DESTINATIONTYPE.String(),
				EmailUsers: []string{memberEmail},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			// Validate reconciliation
			isNotificationsReconciled(instance)

			// Delete all notifications
			instance.Spec.Notifications = []appv1alpha2.Notification{}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		It("can create Terraform Enterprise email address notifications", func() {
			if cloudEndpoint == tfcDefaultAddress {
				Skip("Does not run against HCP Terraform, skip this test")
			}
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name:           "email",
				Type:           models.EMAIL_NOTIFICATIONCONFIGURATIONS_ATTRIBUTES_DESTINATIONTYPE.String(),
				EmailAddresses: []string{"user@example.com"},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})
	})
})

func isNotificationsReconciled(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)
	Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())

	memberships := make(map[string]string)
	listQuery := &organizations.ItemOrganizationMembershipsRequestBuilderGetQueryParameters{
		Pagesize:     new(int32(controller.MaxPageSize)),
		Filterstatus: new(organizationmemberships.INVITED_GETFILTERSTATUSQUERYPARAMETERTYPE),
	}
	for {
		list, err := tfClient.API.Organizations().ByOrganization_name(instance.Spec.Organization).OrganizationMemberships().Get(ctx,
			&abstractions.RequestConfiguration[organizations.ItemOrganizationMembershipsRequestBuilderGetQueryParameters]{QueryParameters: listQuery})
		Expect(err).Should(Succeed())
		Expect(list).ShouldNot(BeNil())

		for _, ms := range list.GetData() {
			if ms.GetRelationships() == nil || ms.GetRelationships().GetUser() == nil || ms.GetRelationships().GetUser().GetData() == nil || ms.GetAttributes() == nil {
				continue
			}
			memberships[pointer.ValueOf(ms.GetRelationships().GetUser().GetData().GetId())] = pointer.ValueOf(ms.GetAttributes().GetEmail())
		}
		next := nextPage(list.GetMeta())
		if next == 0 {
			break
		}
		listQuery.Pagenumber = &next
	}

	Eventually(func() []appv1alpha2.Notification {
		notifications, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).NotificationConfigurations().Get(ctx,
			&abstractions.RequestConfiguration[workspaces.ItemNotificationConfigurationsRequestBuilderGetQueryParameters]{
				QueryParameters: &workspaces.ItemNotificationConfigurationsRequestBuilderGetQueryParameters{
					Pagesize: new(int32(controller.MaxPageSize)),
				},
			})
		Expect(err).Should(Succeed())
		Expect(notifications).ShouldNot(BeNil())

		// Do not use make()
		// workspace must be nil if there are no triggers
		var workspace []appv1alpha2.Notification
		for _, n := range notifications.GetData() {
			attrs := n.GetAttributes()
			if attrs == nil {
				continue
			}
			// Do not use make()
			// t must be nil if there are no triggers
			var t []appv1alpha2.NotificationTrigger
			for _, v := range attrs.GetTriggers() {
				t = append(t, appv1alpha2.NotificationTrigger(v))
			}

			// Do not use make()
			// eu must be nil if there are no email users
			var eu []string
			if n.GetRelationships() != nil && n.GetRelationships().GetUsers() != nil {
				for _, v := range n.GetRelationships().GetUsers().GetData() {
					eu = append(eu, memberships[pointer.ValueOf(v.GetId())])
				}
			}

			destinationType := ""
			if attrs.GetDestinationType() != nil {
				destinationType = attrs.GetDestinationType().String()
			}

			workspace = append(workspace, appv1alpha2.Notification{
				Name:           pointer.ValueOf(attrs.GetName()),
				Type:           destinationType,
				Enabled:        pointer.ValueOf(attrs.GetEnabled()),
				Token:          pointer.ValueOf(attrs.GetToken()),
				Triggers:       t,
				URL:            pointer.ValueOf(attrs.GetUrl()),
				EmailAddresses: attrs.GetEmailAddresses(),
				EmailUsers:     eu,
			})
		}

		return workspace
	}).Should(ContainElements(instance.Spec.Notifications))
}

func createOrgMember(email string) string {
	attrs := models.NewOrganizationMemberships_attributes()
	attrs.SetEmail(&email)
	data := models.NewOrganizationMemberships()
	data.SetTypeEscaped(new(models.ORGANIZATIONMEMBERSHIPS_ORGANIZATIONMEMBERSHIPS_TYPE))
	data.SetAttributes(attrs)
	body := models.NewOrganizationMembershipsEnvelope()
	body.SetData(data)
	m, err := tfClient.API.Organizations().ByOrganization_name(organization).OrganizationMemberships().Post(ctx, body, nil)
	Expect(err).Should(Succeed())
	Expect(m).ShouldNot(BeNil())
	Expect(m.GetData()).ShouldNot(BeNil())
	return pointer.ValueOf(m.GetData().GetId())
}
