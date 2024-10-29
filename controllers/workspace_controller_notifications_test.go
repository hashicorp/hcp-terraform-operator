// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
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
		Expect(tfClient.OrganizationMemberships.Delete(ctx, memberID)).Should(Succeed())
		Expect(tfClient.OrganizationMemberships.Delete(ctx, memberID2)).Should(Succeed())
	})

	Context("Notifications", func() {
		It("can create single notification", func() {
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name: "slack",
				Type: tfc.NotificationDestinationTypeSlack,
				URL:  webhookURL,
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		It("can create multiple notifications", func() {
			if cloudEndpoint != tfcDefaultAddress {
				Skip("Does not run against TFE, skip this test")
			}
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name: "slack",
				Type: tfc.NotificationDestinationTypeSlack,
				URL:  webhookURL,
			})
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name:       "email",
				Type:       tfc.NotificationDestinationTypeEmail,
				EmailUsers: []string{memberEmail},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		It("can re-create notifications", func() {
			if cloudEndpoint != tfcDefaultAddress {
				Skip("Does not run against TFE, skip this test")
			}
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name: "slack",
				Type: tfc.NotificationDestinationTypeSlack,
				URL:  webhookURL,
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			// Validate reconciliation
			isNotificationsReconciled(instance)

			// Delete notifications manually
			notifications, err := tfClient.NotificationConfigurations.List(ctx, instance.Status.WorkspaceID, &tfc.NotificationConfigurationListOptions{})
			Expect(err).Should(Succeed())
			Expect(notifications).ShouldNot(BeNil())
			for _, n := range notifications.Items {
				Expect(tfClient.NotificationConfigurations.Delete(ctx, n.ID)).Should(Succeed())
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
				Type:       tfc.NotificationDestinationTypeEmail,
				EmailUsers: []string{memberEmail},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
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
				Type:       tfc.NotificationDestinationTypeEmail,
				EmailUsers: []string{memberEmail},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
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
				Type:           tfc.NotificationDestinationTypeEmail,
				EmailAddresses: []string{"user@example.com"},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})
	})
})

func isNotificationsReconciled(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)
	Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())

	memberships := make(map[string]string)
	listOpts := &tfc.OrganizationMembershipListOptions{
		ListOptions: tfc.ListOptions{
			PageSize: maxPageSize,
		},
		Status: tfc.OrganizationMembershipInvited,
	}
	for {
		list, err := tfClient.OrganizationMemberships.List(ctx, instance.Spec.Organization, listOpts)
		Expect(err).Should(Succeed())
		Expect(list).ShouldNot(BeNil())

		for _, ms := range list.Items {
			memberships[ms.User.ID] = ms.Email
		}
		if list.NextPage == 0 {
			break
		}
		listOpts.PageNumber = list.NextPage
	}

	Eventually(func() []appv1alpha2.Notification {
		notifications, err := tfClient.NotificationConfigurations.List(ctx, instance.Status.WorkspaceID, &tfc.NotificationConfigurationListOptions{})
		Expect(err).Should(Succeed())
		Expect(notifications).ShouldNot(BeNil())

		// Do not use make()
		// workspace must be nil if there are no triggers
		var workspace []appv1alpha2.Notification
		for _, n := range notifications.Items {
			// Do not use make()
			// t must be nil if there are no triggers
			var t []appv1alpha2.NotificationTrigger
			for _, v := range n.Triggers {
				t = append(t, appv1alpha2.NotificationTrigger(v))
			}
			// Do not use make()
			// eu must be nil if there are no email users
			var eu []string
			for _, v := range n.EmailUsers {
				eu = append(eu, memberships[v.ID])
			}
			workspace = append(workspace, appv1alpha2.Notification{
				Name:           n.Name,
				Type:           n.DestinationType,
				Enabled:        n.Enabled,
				Token:          n.Token,
				Triggers:       t,
				URL:            n.URL,
				EmailAddresses: n.EmailAddresses,
				EmailUsers:     eu,
			})
		}

		return workspace
	}).Should(ContainElements(instance.Spec.Notifications))
}

func createOrgMember(email string) string {
	m, err := tfClient.OrganizationMemberships.Create(ctx, organization, tfc.OrganizationMembershipCreateOptions{Email: &email})
	Expect(err).Should(Succeed())
	Expect(m).ShouldNot(BeNil())

	return m.ID
}
