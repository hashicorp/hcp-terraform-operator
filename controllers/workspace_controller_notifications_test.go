// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"os"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/go-tfe"
	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Label("Notifications"), Ordered, func() {
	var (
		instance  *appv1alpha2.Workspace
		workspace = fmt.Sprintf("kubernetes-operator-%v", GinkgoRandomSeed())

		memberEmail  = fmt.Sprintf("kubernetes-operator-member-%v@hashicorp.com", GinkgoRandomSeed())
		memberEmail2 = fmt.Sprintf("kubernetes-operator-member-2-%v@hashicorp.com", GinkgoRandomSeed())
		memberID     = ""
		memberID2    = ""
	)

	BeforeAll(func() {
		memberID = createOrgMember(memberEmail)
		memberID2 = createOrgMember(memberEmail2)
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
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
							Name: namespacedName.Name,
						},
						Key: secretKey,
					},
				},
				Name:          workspace,
				Notifications: []appv1alpha2.Notification{},
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterAll(func() {
		Expect(tfClient.OrganizationMemberships.Delete(ctx, memberID)).Should(Succeed())
		Expect(tfClient.OrganizationMemberships.Delete(ctx, memberID2)).Should(Succeed())
	})

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance, namespacedName)
	})

	Context("Notifications", func() {
		It("can create single notification", func() {
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name: "slack",
				Type: tfe.NotificationDestinationTypeSlack,
				URL:  "https://example.com",
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		It("can create multiple notifications", func() {
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name: "slack",
				Type: tfe.NotificationDestinationTypeSlack,
				URL:  "https://example.com",
			})
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name:       "email",
				Type:       tfe.NotificationDestinationTypeEmail,
				EmailUsers: []string{memberEmail},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		It("can re-create notifications", func() {
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name: "slack",
				Type: tfe.NotificationDestinationTypeSlack,
				URL:  "https://example.com",
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)
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
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name:       "email",
				Type:       tfe.NotificationDestinationTypeEmail,
				EmailUsers: []string{memberEmail},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)
			// Validate reconciliation
			isNotificationsReconciled(instance)

			// Add a new email user
			instance.Spec.Notifications[0].EmailUsers = append(instance.Spec.Notifications[0].EmailUsers, memberEmail2)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		It("can delete notifications", func() {
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name:       "email",
				Type:       tfe.NotificationDestinationTypeEmail,
				EmailUsers: []string{memberEmail},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)
			// Validate reconciliation
			isNotificationsReconciled(instance)

			// Delete all notifications
			instance.Spec.Notifications = []appv1alpha2.Notification{}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		It("can create Terraform Enterprise email address notifications", func() {
			if _, ok := os.LookupEnv("TFE_ADDRESS"); !ok {
				Skip("Environment variable TFE_ADDRESS is either not set or empty")
			}
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name:           "email",
				Type:           tfe.NotificationDestinationTypeEmail,
				EmailAddresses: []string{"user@example.com"},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)
			// Validate reconciliation
			isNotificationsReconciled(instance)
		})
	})
})

func isNotificationsReconciled(instance *appv1alpha2.Workspace) {
	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		s := instance.Spec.Notifications

		notifications, err := tfClient.NotificationConfigurations.List(ctx, instance.Status.WorkspaceID, &tfc.NotificationConfigurationListOptions{})
		Expect(err).Should(Succeed())
		Expect(notifications).ShouldNot(BeNil())

		members, err := tfClient.OrganizationMemberships.List(ctx, instance.Spec.Organization, &tfc.OrganizationMembershipListOptions{})
		Expect(err).Should(Succeed())
		Expect(members).ShouldNot(BeNil())

		m := make(map[string]string)
		for _, ms := range members.Items {
			m[ms.User.ID] = ms.Email
		}

		if len(instance.Spec.Notifications) != len(notifications.Items) {
			return false
		}

		var w []appv1alpha2.Notification
		for _, n := range notifications.Items {
			var nt []appv1alpha2.NotificationTrigger
			for _, t := range n.Triggers {
				nt = append(nt, appv1alpha2.NotificationTrigger(t))
			}
			var nu []string
			for _, u := range n.EmailUsers {
				nu = append(nu, m[u.ID])
			}
			w = append(w, appv1alpha2.Notification{
				Name:           n.Name,
				Type:           n.DestinationType,
				Enabled:        n.Enabled,
				Token:          n.Token,
				Triggers:       nt,
				URL:            n.URL,
				EmailAddresses: n.EmailAddresses,
				EmailUsers:     nu,
			})
		}

		return cmp.Equal(s, w)
	}).Should(BeTrue())
}

func createOrgMember(email string) string {
	m, err := tfClient.OrganizationMemberships.Create(ctx, organization, tfc.OrganizationMembershipCreateOptions{Email: &email})
	Expect(err).Should(Succeed())
	Expect(m).ShouldNot(BeNil())
	return m.ID
}
