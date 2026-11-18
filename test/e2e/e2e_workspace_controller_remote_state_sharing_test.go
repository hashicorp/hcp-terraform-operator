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
		wsName = fmt.Sprintf("%v-share", workspace)
		wsName2 = fmt.Sprintf("%v-2", wsName)
		wsAttrs := models.NewWorkspaces_attributes()
		wsAttrs.SetName(&wsName)
		wsID = pointer.ValueOf(createWorkspace(wsAttrs).GetId())
		wsAttrs2 := models.NewWorkspaces_attributes()
		wsAttrs2.SetName(&wsName2)
		wsID2 = pointer.ValueOf(createWorkspace(wsAttrs2).GetId())

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
		Expect(tfClient.API.Workspaces().ByWorkspace_id(wsID).Delete(ctx, nil)).Should(Succeed())
		Expect(tfClient.API.Workspaces().ByWorkspace_id(wsID2).Delete(ctx, nil)).Should(Succeed())
	})

	Context("Remote State Sharing", func() {
		It("can enable remote state sharing for all workspaces", func() {
			instance.Spec.RemoteStateSharing = &appv1alpha2.RemoteStateSharing{
				AllWorkspaces: true,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledGlobalRemoteStateSharing(instance)

			// Manually change Global Remote State Sharing to false
			attrs := models.NewWorkspaces_attributes()
			attrs.SetGlobalRemoteState(new(false))
			data := models.NewWorkspaces()
			data.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
			data.SetAttributes(attrs)
			body := models.NewWorkspacesEnvelope()
			body.SetData(data)
			_, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Patch(ctx, body, nil)
			Expect(err).Should(Succeed())
			// Wait for restoration
			isReconciledGlobalRemoteStateSharing(instance)
		})

		It("can disable remote state sharing for all workspaces", func() {
			instance.Spec.RemoteStateSharing = &appv1alpha2.RemoteStateSharing{
				AllWorkspaces: false,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledGlobalRemoteStateSharing(instance)

			// Manually change Global Remote State Sharing to true
			attrs := models.NewWorkspaces_attributes()
			attrs.SetGlobalRemoteState(new(true))
			data := models.NewWorkspaces()
			data.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
			data.SetAttributes(attrs)
			body := models.NewWorkspacesEnvelope()
			body.SetData(data)
			_, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Patch(ctx, body, nil)
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
			createWorkspaceResource(instance)
			isReconciledRemoteStateSharingForWorkspaces(instance, wsID)

			// Manually delete the workspace from Remote State Sharing
			err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Relationships().RemoteStateConsumers().Delete(ctx, remoteStateConsumers(wsID), nil)
			Expect(err).Should(Succeed())
			// Wait for restoration
			isReconciledRemoteStateSharingForWorkspaces(instance, wsID)

			// Manually add the workspace from Remote State Sharing
			err = tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Relationships().RemoteStateConsumers().Post(ctx, remoteStateConsumers(wsID2), nil)
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
			createWorkspaceResource(instance)
			isReconciledRemoteStateSharingForWorkspaces(instance, wsID)

			// Manually delete the workspace from Remote State Sharing
			err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Relationships().RemoteStateConsumers().Delete(ctx, remoteStateConsumers(wsID), nil)
			Expect(err).Should(Succeed())
			// Wait for restoration
			isReconciledRemoteStateSharingForWorkspaces(instance, wsID)

			// Manually add the workspace from Remote State Sharing
			err = tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Relationships().RemoteStateConsumers().Post(ctx, remoteStateConsumers(wsID2), nil)
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
		ws, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		Expect(ws.GetData()).ShouldNot(BeNil())
		Expect(ws.GetData().GetAttributes()).ShouldNot(BeNil())
		return pointer.ValueOf(ws.GetData().GetAttributes().GetGlobalRemoteState()) == instance.Spec.RemoteStateSharing.AllWorkspaces
	}).Should(BeTrue())
}

func isReconciledRemoteStateSharingForWorkspaces(instance *appv1alpha2.Workspace, workspaceID string) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		rsc, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Relationships().RemoteStateConsumers().Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(rsc).ShouldNot(BeNil())
		items := rsc.GetData()
		if len(items) == 0 {
			return false
		}
		for _, r := range items {
			if pointer.ValueOf(r.GetId()) == workspaceID && len(items) == len(instance.Spec.RemoteStateSharing.Workspaces) {
				return true
			}
		}
		return false
	}).Should(BeTrue())
}

func createWorkspace(o models.Workspaces_attributesable) models.Workspacesable {
	GinkgoHelper()
	data := models.NewWorkspaces()
	data.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
	data.SetAttributes(o)
	body := models.NewWorkspacesEnvelope()
	body.SetData(data)
	resp, err := tfClient.API.Organizations().ByOrganization_name(organization).Workspaces().Post(ctx, body, nil)
	Expect(err).Should(Succeed())
	Expect(resp).ShouldNot(BeNil())
	Expect(resp.GetData()).ShouldNot(BeNil())
	return resp.GetData()
}

func remoteStateConsumers(ids ...string) models.WorkspacesIdentifierArrayDocumentable {
	consumers := make([]models.WorkspacesIdentifierArrayDocument_dataable, 0, len(ids))
	for _, id := range ids {
		c := models.NewWorkspacesIdentifierArrayDocument_data()
		c.SetId(new(id))
		c.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIERARRAYDOCUMENT_DATA_TYPE))
		consumers = append(consumers, c)
	}
	body := models.NewWorkspacesIdentifierArrayDocument()
	body.SetData(consumers)
	return body
}
