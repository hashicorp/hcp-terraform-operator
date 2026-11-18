// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"errors"
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe/v2"

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/workspaces"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
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
				Name:             workspace,
				ApplyMethod:      "auto",
				AllowDestroyPlan: false,
				Description:      "Deletion Policy",
				ExecutionMode:    "remote",
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		deleteWorkspace(instance)
	})

	Context("Deletion Policy", func() {
		It("can delete a resource that does not manage a workspace", func() {
			instance.Spec.Token.SecretKeyRef.Key = dummySecretKey
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
		})
		It("can retain a workspace", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicyRetain
			createWorkspaceResource(instance)
			workspaceID := instance.Status.WorkspaceID
			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return kerrors.IsNotFound(err)
			}).Should(BeTrue())
			workspace, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Get(ctx, nil)
			Expect(err).Should(Succeed())
			Expect(workspace).NotTo(BeNil())
		})
		It("can soft delete a workspace", func() {
			if cloudEndpoint != tfcDefaultAddress {
				Skip("Does not run against TFC, skip this test")
			}
			instance.Spec.AllowDestroyPlan = true
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicySoft
			createWorkspaceResource(instance)
			workspaceID := instance.Status.WorkspaceID

			cv := createAndUploadConfigurationVersion(instance.Status.WorkspaceID, "hoi")
			Eventually(func() bool {
				pageSize := int32(controller.MaxPageSize)
				pageNumber := int32(controller.InitPageNumber)
				for pageNumber != 0 {
					runs, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Runs().Get(ctx, &abstractions.RequestConfiguration[workspaces.ItemRunsRequestBuilderGetQueryParameters]{
						QueryParameters: &workspaces.ItemRunsRequestBuilderGetQueryParameters{
							Pagesize:   &pageSize,
							Pagenumber: &pageNumber,
						},
					})
					Expect(err).To(Succeed())
					for _, r := range runs.GetData() {
						if pointer.ValueOf(r.GetRelationships().GetConfigurationVersion().GetData().GetId()) == cv {
							return pointer.ValueOf(r.GetAttributes().GetStatus()).String() == "applied"
						}
					}
					pageNumber = nextPage(runs.GetMeta())
				}
				return false
			}).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

			runAttrs := models.NewRuns_attributes()
			runAttrs.SetIsDestroy(new(true))
			runAttrs.SetAutoApply(new(true))
			runRels := models.NewRuns_relationships()
			runWorkspace := models.NewWorkspacesIdentifier()
			runWorkspace.SetId(&workspaceID)
			runWorkspace.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIER_TYPE))
			runWorkspaceRel := models.NewWorkspacesHasOne()
			runWorkspaceRel.SetData(runWorkspace)
			runRels.SetWorkspace(runWorkspaceRel)
			runData := models.NewRuns()
			runData.SetTypeEscaped(new(models.RUNS_RUNS_TYPE))
			runData.SetAttributes(runAttrs)
			runData.SetRelationships(runRels)
			runBody := models.NewRunsEnvelope()
			runBody.SetData(runData)
			run, err := tfClient.API.Runs().Post(ctx, runBody, nil)
			Expect(err).To(Succeed())
			Expect(run).ToNot(BeNil())

			Eventually(func() bool {
				_, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Get(ctx, nil)
				return errors.Is(err, tfc.ErrNotFound)
			}).Should(BeTrue())
		})
		It("can destroy delete a workspace", func() {
			if cloudEndpoint != tfcDefaultAddress {
				Skip("Does not run against TFC, skip this test")
			}
			instance.Spec.AllowDestroyPlan = true
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicyDestroy
			createWorkspaceResource(instance)
			workspaceID := instance.Status.WorkspaceID

			cv := createAndUploadConfigurationVersion(instance.Status.WorkspaceID, "hoi")
			Eventually(func() bool {
				pageSize := int32(controller.MaxPageSize)
				pageNumber := int32(controller.InitPageNumber)
				for pageNumber != 0 {
					runs, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Runs().Get(ctx, &abstractions.RequestConfiguration[workspaces.ItemRunsRequestBuilderGetQueryParameters]{
						QueryParameters: &workspaces.ItemRunsRequestBuilderGetQueryParameters{
							Pagesize:   &pageSize,
							Pagenumber: &pageNumber,
						},
					})
					Expect(err).To(Succeed())
					for _, r := range runs.GetData() {
						if pointer.ValueOf(r.GetRelationships().GetConfigurationVersion().GetData().GetId()) == cv {
							return pointer.ValueOf(r.GetAttributes().GetStatus()).String() == "applied"
						}
					}
					pageNumber = nextPage(runs.GetMeta())
				}
				return false
			}).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

			var destroyRunID string
			Eventually(func() bool {
				ws, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Get(ctx, nil)
				Expect(err).To(Succeed())
				Expect(ws).ToNot(BeNil())
				currentRunID := pointer.ValueOf(ws.GetData().GetRelationships().GetCurrentRun().GetData().GetId())
				Expect(currentRunID).ToNot(BeEmpty())
				resp, err := tfClient.API.Runs().ById(currentRunID).Get(ctx, nil)
				Expect(err).To(Succeed())
				Expect(resp).ToNot(BeNil())
				run := resp.GetData()
				Expect(run).ToNot(BeNil())
				destroyRunID = pointer.ValueOf(run.GetId())

				return run.GetAttributes() != nil && pointer.ValueOf(run.GetAttributes().GetIsDestroy())
			}).Should(BeTrue())

			Eventually(func() bool {
				resp, err := tfClient.API.Runs().ById(destroyRunID).Get(ctx, nil)
				if errors.Is(err, tfc.ErrNotFound) {
					return true
				}
				if err == nil && resp != nil && pointer.ValueOf(resp.GetData().GetAttributes().GetStatus()).String() == "applied" {
					return true
				}

				return false
			}).Should(BeTrue())

			Eventually(func() bool {
				_, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Get(ctx, nil)
				return errors.Is(err, tfc.ErrNotFound)
			}).Should(BeTrue())
		})
		It("can force delete a workspace", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicyForce
			createWorkspaceResource(instance)
			workspaceID := instance.Status.WorkspaceID
			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return kerrors.IsNotFound(err)
			}).Should(BeTrue())
			workspace, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Get(ctx, nil)
			Expect(errors.Is(err, tfc.ErrNotFound)).To(BeTrue())
			Expect(workspace).To(BeNil())
		})
	})
})
