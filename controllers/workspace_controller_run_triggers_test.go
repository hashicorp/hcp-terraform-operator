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

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName = newNamespacedName()
		workspace      = fmt.Sprintf("kubernetes-operator-%v", randomNumber())

		wsName  = fmt.Sprintf("kubernetes-operator-source-%v", randomNumber())
		wsName2 = fmt.Sprintf("kubernetes-operator-source2-%v", randomNumber())
		wsID    = ""
		wsID2   = ""
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create two new workspaces to act as a source for Run Triggers
		wsID = createWorkspaceForTests(wsName)
		wsID2 = createWorkspaceForTests(wsName2)
	})

	AfterAll(func() {
		err := tfClient.Workspaces.DeleteByID(ctx, wsID)
		Expect(err).Should(Succeed())

		err = tfClient.Workspaces.DeleteByID(ctx, wsID2)
		Expect(err).Should(Succeed())
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
							Name: secretNamespacedName.Name,
						},
						Key: secretKey,
					},
				},
				Name: workspace,
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance)
	})

	Context("Workspace controller", func() {
		It("can handle run triggers by name", func() {
			instance.Spec.RunTriggers = []appv1alpha2.RunTrigger{
				{Name: wsName},
				{Name: wsName2},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			Eventually(func() bool {
				rt, err := tfClient.RunTriggers.List(ctx, instance.Status.WorkspaceID, &tfc.RunTriggerListOptions{
					RunTriggerType: tfc.RunTriggerInbound,
				})
				Expect(rt).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return hasWorkspaceSource(wsID, rt) && hasWorkspaceSource(wsID2, rt)
			}).Should(BeTrue())
		})

		It("can handle run triggers by ID", func() {
			instance.Spec.RunTriggers = []appv1alpha2.RunTrigger{
				{ID: wsID},
				{ID: wsID2},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			Eventually(func() bool {
				rt, err := tfClient.RunTriggers.List(ctx, instance.Status.WorkspaceID, &tfc.RunTriggerListOptions{
					RunTriggerType: tfc.RunTriggerInbound,
				})
				Expect(rt).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return hasWorkspaceSource(wsID, rt) && hasWorkspaceSource(wsID2, rt)
			}).Should(BeTrue())
		})

		It("can handle run triggers by mix of Name and ID", func() {
			instance.Spec.RunTriggers = []appv1alpha2.RunTrigger{
				{ID: wsID},
				{Name: wsName2},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			Eventually(func() bool {
				rt, err := tfClient.RunTriggers.List(ctx, instance.Status.WorkspaceID, &tfc.RunTriggerListOptions{
					RunTriggerType: tfc.RunTriggerInbound,
				})
				Expect(rt).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return hasWorkspaceSource(wsID, rt) && hasWorkspaceSource(wsID2, rt)
			}).Should(BeTrue())
		})
	})
})

func hasWorkspaceSource(workspaceID string, runTriggerList *tfc.RunTriggerList) bool {
	for _, v := range runTriggerList.Items {
		if workspaceID == v.Sourceable.ID {
			return true
		}
	}

	return false
}
