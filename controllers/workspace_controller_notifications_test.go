// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Label("Notifications"), Ordered, func() {
	var (
		instance  *appv1alpha2.Workspace
		workspace = fmt.Sprintf("kubernetes-operator-%v", GinkgoRandomSeed())

		memberEmail = fmt.Sprintf("kubernetes-operator-member-%v@hashicorp.com", GinkgoRandomSeed())
	)

	BeforeAll(func() {
		member, err := tfClient.OrganizationMemberships.Create(ctx, organization, tfc.OrganizationMembershipCreateOptions{Email: &memberEmail})
		Expect(err).Should(Succeed())
		Expect(member).ShouldNot(BeNil())
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

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance, namespacedName)
	})

	Context("Notifications", func() {
		It("can create single notification", func() {
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name: "slack",
				Type: "slack",
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
				Type: "slack",
				URL:  "https://example.com",
			})
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name:       "email",
				Type:       "email",
				EmailUsers: []string{memberEmail},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)

			// Validate reconciliation
			isNotificationsReconciled(instance)
		})

		// TODO
		// It("can create Terraform Enterprise email address notifications", func() {
		// 	// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
		// 	createWorkspace(instance, namespacedName)
		// })
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

		w := make([]appv1alpha2.Notification, len(notifications.Items))
		for i, n := range notifications.Items {
			var nt []appv1alpha2.NotificationTrigger
			for _, t := range n.Triggers {
				nt = append(nt, appv1alpha2.NotificationTrigger(t))
			}
			var nu []string
			for _, u := range n.EmailUsers {
				nu = append(nu, m[u.ID])
			}
			w[i] = appv1alpha2.Notification{
				Name:           n.Name,
				Type:           n.DestinationType,
				Enabled:        n.Enabled,
				Token:          n.Token,
				Triggers:       nt,
				URL:            n.URL,
				EmailAddresses: n.EmailAddresses,
				EmailUsers:     nu,
			}
		}

		return cmp.Equal(s, w)
	}).Should(BeTrue())
}
