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

		wsName  = fmt.Sprintf("%s-share", workspace)
		wsName2 = fmt.Sprintf("%s-share2", workspace)
		wsID    = ""
		wsID2   = ""
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create new workspaces for tests
		wsID = createWorkspaceForTests(wsName)
		wsID2 = createWorkspaceForTests(wsName2)
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

	AfterAll(func() {
		// Clean up additional workspaces
		err := tfClient.Workspaces.DeleteByID(ctx, wsID)
		Expect(err).Should(Succeed())

		err = tfClient.Workspaces.DeleteByID(ctx, wsID2)
		Expect(err).Should(Succeed())

	})

	Context("Workspace controller", func() {
		It("can enable remote state sharing for all workspaces", func() {
			instance.Spec.RemoteStateSharing = &appv1alpha2.RemoteStateSharing{
				AllWorkspaces: true,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledGlobalRemoteStateSharing(instance)

			// Manually change Global Remote State Sharing to false
			_, err := tfClient.Workspaces.UpdateByID(ctx, instance.Status.WorkspaceID, tfc.WorkspaceUpdateOptions{
				GlobalRemoteState: tfc.Bool(false),
			})
			Expect(err).Should(Succeed())
			// Wait for restoration
			isReconciledGlobalRemoteStateSharing(instance)
		})

		It("can enable remote state sharing for specific workspaces by name", func() {
			instance.Spec.RemoteStateSharing = &appv1alpha2.RemoteStateSharing{
				Workspaces: []*appv1alpha2.ConsumerWorkspace{
					{Name: wsName},
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledRemoteStateSharingForWorkspaces(instance, wsID)

			// Manually delete the workspace from Remote State Sharing
			err := tfClient.Workspaces.RemoveRemoteStateConsumers(ctx, instance.Status.WorkspaceID, tfc.WorkspaceRemoveRemoteStateConsumersOptions{
				Workspaces: []*tfc.Workspace{
					{ID: wsID},
				},
			})
			Expect(err).Should(Succeed())
			// Wait for restoration
			isReconciledRemoteStateSharingForWorkspaces(instance, wsID)

			// Manually add the workspace from Remote State Sharing
			err = tfClient.Workspaces.AddRemoteStateConsumers(ctx, instance.Status.WorkspaceID, tfc.WorkspaceAddRemoteStateConsumersOptions{
				Workspaces: []*tfc.Workspace{
					{ID: wsID2},
				},
			})
			Expect(err).Should(Succeed())
			// Wait for restoration
			isReconciledRemoteStateSharingForWorkspaces(instance, wsID)
		})

		It("can enable remote state sharing for specific workspaces by ID", func() {
			instance.Spec.RemoteStateSharing = &appv1alpha2.RemoteStateSharing{
				Workspaces: []*appv1alpha2.ConsumerWorkspace{
					{Name: wsName},
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledRemoteStateSharingForWorkspaces(instance, wsID)

			// Manually delete the workspace from Remote State Sharing
			err := tfClient.Workspaces.RemoveRemoteStateConsumers(ctx, instance.Status.WorkspaceID, tfc.WorkspaceRemoveRemoteStateConsumersOptions{
				Workspaces: []*tfc.Workspace{
					{ID: wsID},
				},
			})
			Expect(err).Should(Succeed())
			// Wait for restoration
			isReconciledRemoteStateSharingForWorkspaces(instance, wsID)

			// Manually add the workspace from Remote State Sharing
			err = tfClient.Workspaces.AddRemoteStateConsumers(ctx, instance.Status.WorkspaceID, tfc.WorkspaceAddRemoteStateConsumersOptions{
				Workspaces: []*tfc.Workspace{
					{ID: wsID2},
				},
			})
			Expect(err).Should(Succeed())
			// Wait for restoration
			isReconciledRemoteStateSharingForWorkspaces(instance, wsID)
		})
	})
})

func isReconciledGlobalRemoteStateSharing(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		return ws.GlobalRemoteState
	}).Should(BeTrue())
}

func isReconciledRemoteStateSharingForWorkspaces(instance *appv1alpha2.Workspace, workspaceID string) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		rsc, err := tfClient.Workspaces.ListRemoteStateConsumers(ctx, instance.Status.WorkspaceID, &tfc.RemoteStateConsumersListOptions{})
		Expect(err).Should(Succeed())
		Expect(rsc).ShouldNot(BeNil())
		if len(rsc.Items) == 0 {
			return false
		}
		for _, r := range rsc.Items {
			if r.ID == workspaceID && len(rsc.Items) == len(instance.Spec.RemoteStateSharing.Workspaces) {
				return true
			}
		}
		return false
	}).Should(BeTrue())
}

func createWorkspaceForTests(wsName string) string {
	ws, err := tfClient.Workspaces.Create(ctx, organization, tfc.WorkspaceCreateOptions{
		Name: &wsName,
	})
	Expect(err).Should(Succeed())
	Expect(ws).ShouldNot(BeNil())
	return ws.ID
}
