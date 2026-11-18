// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/workspaces"
	"github.com/hashicorp/go-tfe/v2/api/workspaces/item/runtriggers"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/controller"
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
		wsName = fmt.Sprintf("kubernetes-operator-source-%v", randomNumber())
		wsName2 = fmt.Sprintf("kubernetes-operator-source-2-%v", randomNumber())
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

	Context("Run Triggers", func() {
		It("can handle run triggers by name", func() {
			instance.Spec.RunTriggers = []appv1alpha2.RunTrigger{
				{Name: wsName},
				{Name: wsName2},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			Eventually(func() bool {
				sources := listInboundRunTriggerSources(instance.Status.WorkspaceID)
				return sources[wsID] && sources[wsID2]
			}).Should(BeTrue())
		})

		It("can handle run triggers by ID", func() {
			instance.Spec.RunTriggers = []appv1alpha2.RunTrigger{
				{ID: wsID},
				{ID: wsID2},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			Eventually(func() bool {
				sources := listInboundRunTriggerSources(instance.Status.WorkspaceID)
				return sources[wsID] && sources[wsID2]
			}).Should(BeTrue())
		})

		It("can handle run triggers by mix of Name and ID", func() {
			instance.Spec.RunTriggers = []appv1alpha2.RunTrigger{
				{ID: wsID},
				{Name: wsName2},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			Eventually(func() bool {
				sources := listInboundRunTriggerSources(instance.Status.WorkspaceID)
				return sources[wsID] && sources[wsID2]
			}).Should(BeTrue())
		})
	})
})

// listInboundRunTriggerSources returns a set of IDs of the source workspaces of the inbound run triggers of a given workspace.
func listInboundRunTriggerSources(workspaceID string) map[string]bool {
	GinkgoHelper()
	sources := make(map[string]bool)
	query := &workspaces.ItemRunTriggersRequestBuilderGetQueryParameters{
		FilterrunTriggertype: new(runtriggers.INBOUND_GETFILTERRUNTRIGGERTYPEQUERYPARAMETERTYPE),
		Pagesize:             new(int32(controller.MaxPageSize)),
	}
	for {
		rt, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).RunTriggers().Get(ctx,
			&abstractions.RequestConfiguration[workspaces.ItemRunTriggersRequestBuilderGetQueryParameters]{QueryParameters: query})
		Expect(err).Should(Succeed())
		Expect(rt).ShouldNot(BeNil())
		for _, v := range rt.GetData() {
			if v.GetRelationships() == nil || v.GetRelationships().GetSourceable() == nil || v.GetRelationships().GetSourceable().GetData() == nil {
				continue
			}
			sources[pointer.ValueOf(v.GetRelationships().GetSourceable().GetData().GetId())] = true
		}
		next := nextPage(rt.GetMeta())
		if next == 0 {
			break
		}
		query.Pagenumber = &next
	}
	return sources
}
