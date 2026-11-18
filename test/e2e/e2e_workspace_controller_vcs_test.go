// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"
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
		oAuthTokenID   = os.Getenv("TFC_OAUTH_TOKEN")
		repository     = os.Getenv("TFC_VCS_REPO")
	)

	BeforeAll(func() {
		if oAuthTokenID == "" {
			Skip("Environment variable TFC_OAUTH_TOKEN is either not set or empty")
		}
		if repository == "" {
			Skip("Environment variable TFC_VCS_REPO is either not set or empty")
		}
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
				Name: workspace,
				VersionControl: &appv1alpha2.VersionControl{
					OAuthTokenID:       oAuthTokenID,
					Repository:         repository,
					Branch:             "operator",
					SpeculativePlans:   true,
					EnableFileTriggers: false,
				},
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		deleteWorkspace(instance)
	})

	Context("VCS", func() {
		It("can attach VCS to the workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				if ws.GetVcsRepo() == nil {
					return false
				}
				return pointer.ValueOf(ws.GetVcsRepo().GetOauthTokenId()) == instance.Spec.VersionControl.OAuthTokenID &&
					pointer.ValueOf(ws.GetVcsRepo().GetIdentifier()) == instance.Spec.VersionControl.Repository &&
					pointer.ValueOf(ws.GetVcsRepo().GetBranch()) == instance.Spec.VersionControl.Branch
			}).Should(BeTrue())
		})

		It("can update VCS", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			instance.Spec.VersionControl.Branch = "main"
			instance.Spec.VersionControl.EnableFileTriggers = true
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				if ws.GetVcsRepo() == nil {
					return false
				}
				return pointer.ValueOf(ws.GetVcsRepo().GetOauthTokenId()) == instance.Spec.VersionControl.OAuthTokenID &&
					pointer.ValueOf(ws.GetVcsRepo().GetIdentifier()) == instance.Spec.VersionControl.Repository &&
					pointer.ValueOf(ws.GetVcsRepo().GetBranch()) == instance.Spec.VersionControl.Branch &&
					pointer.ValueOf(ws.GetFileTriggersEnabled()) == instance.Spec.VersionControl.EnableFileTriggers
			}).Should(BeTrue())
		})

		It("can revert manual changes VCS", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			vcs := models.NewWorkspaces_attributes_vcsRepo()
			vcs.SetBranch(new("main"))
			attrs := models.NewWorkspaces_attributes()
			attrs.SetVcsRepo(vcs)
			attrs.SetFileTriggersEnabled(new(true))
			patchData := models.NewWorkspaces()
			patchData.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
			patchData.SetAttributes(attrs)
			patchBody := models.NewWorkspacesEnvelope()
			patchBody.SetData(patchData)
			_, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Patch(ctx, patchBody, nil)
			Expect(err).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				if ws.GetVcsRepo() == nil {
					return false
				}
				return pointer.ValueOf(ws.GetVcsRepo().GetOauthTokenId()) == instance.Spec.VersionControl.OAuthTokenID &&
					pointer.ValueOf(ws.GetVcsRepo().GetIdentifier()) == instance.Spec.VersionControl.Repository &&
					pointer.ValueOf(ws.GetVcsRepo().GetBranch()) == instance.Spec.VersionControl.Branch &&
					pointer.ValueOf(ws.GetFileTriggersEnabled()) == instance.Spec.VersionControl.EnableFileTriggers
			}).Should(BeTrue())
		})

		It("can revert manual detach of VCS from the workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			patchWorkspaceRaw(instance.Status.WorkspaceID, map[string]any{"vcs-repo": nil})

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				if ws.GetVcsRepo() == nil {
					return false
				}
				return pointer.ValueOf(ws.GetVcsRepo().GetOauthTokenId()) == instance.Spec.VersionControl.OAuthTokenID &&
					pointer.ValueOf(ws.GetVcsRepo().GetIdentifier()) == instance.Spec.VersionControl.Repository &&
					pointer.ValueOf(ws.GetVcsRepo().GetBranch()) == instance.Spec.VersionControl.Branch
			}).Should(BeTrue())
		})

		It("can detach VCS from the workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			instance.Spec.VersionControl = nil
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				return ws.GetVcsRepo() == nil
			}).Should(BeTrue())
		})

		It("can disable speculative plan", func() {
			// Make a copy of the original instance to then compare Workspace against it
			// It helps to make sure that the 'Spec' part is not mutated
			instanceCopy := instance.DeepCopy()

			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			// Validate that all attributes are set correctly
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				if ws.GetVcsRepo() == nil {
					return false
				}
				return pointer.ValueOf(ws.GetVcsRepo().GetOauthTokenId()) == instanceCopy.Spec.VersionControl.OAuthTokenID &&
					pointer.ValueOf(ws.GetVcsRepo().GetIdentifier()) == instanceCopy.Spec.VersionControl.Repository &&
					pointer.ValueOf(ws.GetVcsRepo().GetBranch()) == instanceCopy.Spec.VersionControl.Branch &&
					pointer.ValueOf(ws.GetSpeculativeEnabled()) == instanceCopy.Spec.VersionControl.SpeculativePlans
			}).Should(BeTrue())

			// Update the Kubernetes workspace object fields
			instance.Spec.VersionControl.SpeculativePlans = false
			// Make a copy of the original instance to then compare Workspace against it
			// It helps to make sure that the 'Spec' part is not mutated
			instanceCopy = instance.DeepCopy()

			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Validate that all attributes are set correctly
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				if ws.GetVcsRepo() == nil {
					return false
				}
				return pointer.ValueOf(ws.GetVcsRepo().GetOauthTokenId()) == instanceCopy.Spec.VersionControl.OAuthTokenID &&
					pointer.ValueOf(ws.GetVcsRepo().GetIdentifier()) == instanceCopy.Spec.VersionControl.Repository &&
					pointer.ValueOf(ws.GetVcsRepo().GetBranch()) == instanceCopy.Spec.VersionControl.Branch &&
					pointer.ValueOf(ws.GetSpeculativeEnabled()) == instanceCopy.Spec.VersionControl.SpeculativePlans
			}).Should(BeTrue())
		})

		It("cat trigger a new apply run on creation", func() {
			instance.SetAnnotations(map[string]string{
				controller.WorkspaceAnnotationRunNew:  controller.MetaTrue,
				controller.WorkspaceAnnotationRunType: controller.RunTypeApply,
			})
			instance.Spec.ApplyMethod = "auto"
			instance.Spec.ApplyRunTrigger = "auto"
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Annotations[controller.WorkspaceAnnotationRunNew] == controller.MetaTrue {
					return false
				}
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.Run.RunCompleted()
			}).Should(BeTrue())
		})

		It("cat trigger a new plan run on creation", func() {
			instance.SetAnnotations(map[string]string{
				controller.WorkspaceAnnotationRunNew:  controller.MetaTrue,
				controller.WorkspaceAnnotationRunType: controller.RunTypePlan,
			})
			instance.Spec.ApplyMethod = "auto"
			instance.Spec.ApplyRunTrigger = "auto"
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Plan == nil {
					return false
				}
				return instance.Status.Plan.RunCompleted()
			}).Should(BeTrue())
		})

		It("cat handle file trigger patterns", func() {
			instance.Spec.VersionControl.EnableFileTriggers = true
			instance.Spec.VersionControl.TriggerPatterns = []string{"/modules/", "/variables/"}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				if ws.GetVcsRepo() == nil {
					return false
				}
				return pointer.ValueOf(ws.GetFileTriggersEnabled()) == instance.Spec.VersionControl.EnableFileTriggers &&
					slices.Compare(ws.GetTriggerPatterns(), instance.Spec.VersionControl.TriggerPatterns) == 0
			}).Should(BeTrue())

			// Revert manual changes
			attrs := models.NewWorkspaces_attributes()
			attrs.SetTriggerPatterns([]string{"/var/"})
			patchData := models.NewWorkspaces()
			patchData.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
			patchData.SetAttributes(attrs)
			patchBody := models.NewWorkspacesEnvelope()
			patchBody.SetData(patchData)
			_, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Patch(ctx, patchBody, nil)
			Expect(err).Should(Succeed())
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				if ws.GetVcsRepo() == nil {
					return false
				}
				return pointer.ValueOf(ws.GetFileTriggersEnabled()) == instance.Spec.VersionControl.EnableFileTriggers &&
					slices.Compare(ws.GetTriggerPatterns(), instance.Spec.VersionControl.TriggerPatterns) == 0
			}).Should(BeTrue())
		})

		It("cat handle file trigger prefixes", func() {
			instance.Spec.WorkingDirectory = "/mofules/"
			instance.Spec.VersionControl.EnableFileTriggers = true
			instance.Spec.VersionControl.TriggerPrefixes = []string{"/modules/", "/variables/"}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				if ws.GetVcsRepo() == nil {
					return false
				}
				return pointer.ValueOf(ws.GetFileTriggersEnabled()) == instance.Spec.VersionControl.EnableFileTriggers &&
					slices.Compare(ws.GetTriggerPrefixes(), instance.Spec.VersionControl.TriggerPrefixes) == 0
			}).Should(BeTrue())

			// Revert manual changes
			attrs := models.NewWorkspaces_attributes()
			attrs.SetTriggerPrefixes([]string{"/var/"})
			patchData := models.NewWorkspaces()
			patchData.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
			patchData.SetAttributes(attrs)
			patchBody := models.NewWorkspacesEnvelope()
			patchBody.SetData(patchData)
			_, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Patch(ctx, patchBody, nil)
			Expect(err).Should(Succeed())
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				if ws.GetVcsRepo() == nil {
					return false
				}
				return pointer.ValueOf(ws.GetFileTriggersEnabled()) == instance.Spec.VersionControl.EnableFileTriggers &&
					slices.Compare(ws.GetTriggerPrefixes(), instance.Spec.VersionControl.TriggerPrefixes) == 0
			}).Should(BeTrue())
		})
	})
})
