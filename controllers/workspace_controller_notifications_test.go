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

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Label("Notifications"), Ordered, func() {
	var (
		instance  *appv1alpha2.Workspace
		workspace = fmt.Sprintf("kubernetes-operator-%v", GinkgoRandomSeed())
	)

	BeforeAll(func() {
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
		It("can create slack notifications", func() {
			instance.Spec.Notifications = append(instance.Spec.Notifications, appv1alpha2.Notification{
				Name: "this",
				Type: "slack",
				URL:  "https://example.com",
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)
		})
	})
})
