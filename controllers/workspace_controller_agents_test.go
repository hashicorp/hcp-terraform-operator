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

		agentPoolName  = fmt.Sprintf("kubernetes-operator-agent-%v", randomNumber())
		agentPoolName2 = fmt.Sprintf("%v-2", agentPoolName)
		agentPoolID    = ""
		agentPoolID2   = ""
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create an Agent Pools
		agentPoolID = createAgentPool(agentPoolName)
		agentPoolID2 = createAgentPool(agentPoolName2)
	})

	AfterAll(func() {
		// Clean up the Agent Pools
		err := tfClient.AgentPools.Delete(ctx, agentPoolID)
		Expect(err).Should(Succeed())

		err = tfClient.AgentPools.Delete(ctx, agentPoolID2)
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
				Name:          workspace,
				ExecutionMode: "agent",
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance)
	})

	Context("Workspace controller", func() {
		It("can handle agent pool by name", func() {
			instance.Spec.AgentPool = &appv1alpha2.WorkspaceAgentPool{Name: agentPoolName}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledAgentPoolByName(instance)

			// Update the Agent Pool by Name
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.AgentPool = &appv1alpha2.WorkspaceAgentPool{Name: agentPoolName2}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledAgentPoolByName(instance)
		})

		It("can handle agent pool by id", func() {
			instance.Spec.AgentPool = &appv1alpha2.WorkspaceAgentPool{ID: agentPoolID}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledAgentPoolByID(instance)

			// Update the Agent Pool by ID
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.AgentPool = &appv1alpha2.WorkspaceAgentPool{ID: agentPoolID2}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledAgentPoolByID(instance)
		})
	})
})

func createAgentPool(agentPoolName string) string {
	ap, err := tfClient.AgentPools.Create(ctx, organization, tfc.AgentPoolCreateOptions{
		Name: tfc.String(agentPoolName),
	})
	Expect(err).Should(Succeed())
	Expect(ap).ShouldNot(BeNil())
	return ap.ID
}

func isReconciledAgentPoolByID(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(ws).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		return ws.AgentPool.ID == instance.Spec.AgentPool.ID
	}).Should(BeTrue())
}

func isReconciledAgentPoolByName(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(ws).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		ap, err := tfClient.AgentPools.Read(ctx, ws.AgentPool.ID)
		Expect(ap).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		return ap.Name == instance.Spec.AgentPool.Name
	}).Should(BeTrue())
}
