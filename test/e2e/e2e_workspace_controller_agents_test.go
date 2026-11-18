// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName types.NamespacedName
		workspace      string

		agentPoolName  string
		agentPoolName2 string
		agentPoolID    string
		agentPoolID2   string
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		agentPoolName = fmt.Sprintf("kubernetes-operator-agent-%v", randomNumber())
		agentPoolName2 = fmt.Sprintf("%v-2", agentPoolName)
		agentPoolID = createAgentPool(agentPoolName)
		agentPoolID2 = createAgentPool(agentPoolName2)
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
		deleteWorkspace(instance)
		Expect(tfClient.API.AgentPools().ByAgent_pool_id(agentPoolID).Delete(ctx, nil)).Should(Succeed())
		Expect(tfClient.API.AgentPools().ByAgent_pool_id(agentPoolID2).Delete(ctx, nil)).Should(Succeed())
	})

	Context("Agent Execution Mode", func() {
		It("can handle agent pool by name", func() {
			instance.Spec.AgentPool = &appv1alpha2.AgentPoolRef{Name: agentPoolName}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledAgentPoolByName(instance)

			// Update the Agent Pool by Name
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.AgentPool = &appv1alpha2.AgentPoolRef{Name: agentPoolName2}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledAgentPoolByName(instance)
		})

		It("can handle agent pool by id", func() {
			instance.Spec.AgentPool = &appv1alpha2.AgentPoolRef{ID: agentPoolID}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledAgentPoolByID(instance)

			// Update the Agent Pool by ID
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.AgentPool = &appv1alpha2.AgentPoolRef{ID: agentPoolID2}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledAgentPoolByID(instance)
		})
	})
})

func createAgentPool(agentPoolName string) string {
	attrs := models.NewAgentPools_attributes()
	attrs.SetName(&agentPoolName)
	data := models.NewAgentPools()
	data.SetAttributes(attrs)
	body := models.NewAgentPoolsEnvelope()
	body.SetData(data)
	ap, err := tfClient.API.Organizations().ByOrganization_name(organization).AgentPools().Post(ctx, body, nil)
	Expect(err).Should(Succeed())
	Expect(ap).ShouldNot(BeNil())
	Expect(ap.GetData()).ShouldNot(BeNil())
	return pointer.ValueOf(ap.GetData().GetId())
}

func isReconciledAgentPoolByID(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		Expect(ws.GetData()).ShouldNot(BeNil())
		return pointer.ValueOf(ws.GetData().GetRelationships().GetAgentPool().GetData().GetId()) == instance.Spec.AgentPool.ID
	}).Should(BeTrue())
}

func isReconciledAgentPoolByName(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		Expect(ws.GetData()).ShouldNot(BeNil())
		ap, err := tfClient.API.AgentPools().ByAgent_pool_id(pointer.ValueOf(ws.GetData().GetRelationships().GetAgentPool().GetData().GetId())).Get(ctx, nil)
		Expect(ap).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		Expect(ap.GetData()).ShouldNot(BeNil())
		Expect(ap.GetData().GetAttributes()).ShouldNot(BeNil())
		return pointer.ValueOf(ap.GetData().GetAttributes().GetName()) == instance.Spec.AgentPool.Name
	}).Should(BeTrue())
}
