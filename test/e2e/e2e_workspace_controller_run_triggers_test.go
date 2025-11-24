// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName types.NamespacedName
		workspace      string

		wsName  string
		wsName2 string
		wsID    string
		wsID2   string
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		wsName = fmt.Sprintf("kubernetes-operator-source-%v", randomNumber())
		wsName2 = fmt.Sprintf("kubernetes-operator-source-2-%v", randomNumber())
		wsID = createWorkspaceForTests(wsName)
		wsID2 = createWorkspaceForTests(wsName2)
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
		deleteWorkspace(instance)
		Expect(tfClient.Workspaces.DeleteByID(ctx, wsID)).Should(Succeed())
		Expect(tfClient.Workspaces.DeleteByID(ctx, wsID2)).Should(Succeed())
	})

	Context("Run Triggers", func() {
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
