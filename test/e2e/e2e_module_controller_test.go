// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"errors"
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/hashicorp/go-tfe/v2/api/models"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

var _ = Describe("Module сontroller", Ordered, func() {
	var (
		instance       *appv1alpha2.Module
		namespacedName = newNamespacedName()
		workspace      = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
	)

	BeforeAll(func() {
		if cloudEndpoint != tfcDefaultAddress {
			Skip("Does not run against TFC, skip this test")
		}
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		// Create a new module object for each test
		instance = &appv1alpha2.Module{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "app.terraform.io/v1alpha2",
				Kind:       "Module",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              namespacedName.Name,
				Namespace:         namespacedName.Namespace,
				DeletionTimestamp: nil,
				Finalizers:        []string{},
			},
			Spec: appv1alpha2.ModuleSpec{
				Name:         "this",
				Organization: organization,
				Token: appv1alpha2.Token{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretNamespacedName.Name,
						},
						Key: secretKey,
					},
				},
				Module: &appv1alpha2.ModuleSource{
					Source:  "hashicorp/animal/demo",
					Version: "1.0.0",
				},
				DestroyOnDeletion: true,
				Variables: []appv1alpha2.ModuleVariable{
					{
						Name: "name",
					},
				},
				Outputs: []appv1alpha2.ModuleOutput{
					{
						Name:      "animal",
						Sensitive: false,
					},
				},
			},
			Status: appv1alpha2.ModuleStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes Module object and wait until the controller finishes the reconciliation after deletion of the object
		Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, namespacedName, instance)
			// The Kubernetes client will return error 'NotFound' on the "Get" operation once the object is deleted
			return kerrors.IsNotFound(err)
		}).Should(BeTrue())

		// Make sure that the HCP Terraform workspace is deleted
		Eventually(func() bool {
			err := tfClient.API.Organizations().ByOrganization_name(organization).Workspaces().ByWorkspace_name(workspace).Delete(ctx, nil)
			return errors.Is(err, tfc.ErrNotFound)
		}).Should(BeTrue())
	})

	Context("Can Handle", func() {
		It("Reference to Workspace by Name", func() {
			// Create a new HCP Terraform Workspace
			wsAttrs := models.NewWorkspaces_attributes()
			wsAttrs.SetName(&workspace)
			wsAttrs.SetAutoApply(new(true))
			wsAttrs.SetExecutionMode(new(models.REMOTE_WORKSPACES_ATTRIBUTES_EXECUTIONMODE))
			wsID := pointer.ValueOf(createWorkspace(wsAttrs).GetId())

			// Create TFC Workspace variables
			varAttrs := models.NewVars_attributes()
			varAttrs.SetKey(new("name"))
			varAttrs.SetValue(new("Pluto"))
			varAttrs.SetHcl(new(false))
			varAttrs.SetCategory(new(models.TERRAFORM_VARS_ATTRIBUTES_CATEGORY))
			varData := models.NewVars()
			varData.SetTypeEscaped(new(models.VARS_VARS_TYPE))
			varData.SetAttributes(varAttrs)
			varBody := models.NewVarsEnvelope()
			varBody.SetData(varData)
			_, err := tfClient.API.Workspaces().ByWorkspace_id(wsID).Vars().Post(ctx, varBody, nil)
			Expect(err).Should(Succeed())

			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{Name: workspace}
			// Create a new Module
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Make sure a new module is created and executed
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.ConfigurationVersion == nil {
					return false
				}
				return instance.Status.ConfigurationVersion.Status == "uploaded"
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.ConfigurationVersion.Status).NotTo(BeEquivalentTo("errored"))

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.Run.Status == string(models.APPLIED_RUNS_ATTRIBUTES_STATUS.String())
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.Run.Status).NotTo(BeEquivalentTo(string(models.ERRORED_RUNS_ATTRIBUTES_STATUS.String())))
		})
		It("Reference to Workspace by ID", func() {
			// Create a new HCP Terraform Workspace
			wsAttrs := models.NewWorkspaces_attributes()
			wsAttrs.SetName(&workspace)
			wsAttrs.SetAutoApply(new(true))
			wsAttrs.SetExecutionMode(new(models.REMOTE_WORKSPACES_ATTRIBUTES_EXECUTIONMODE))
			wsID := pointer.ValueOf(createWorkspace(wsAttrs).GetId())

			// Create TFC Workspace variables
			varAttrs := models.NewVars_attributes()
			varAttrs.SetKey(new("name"))
			varAttrs.SetValue(new("Pluto"))
			varAttrs.SetHcl(new(false))
			varAttrs.SetCategory(new(models.TERRAFORM_VARS_ATTRIBUTES_CATEGORY))
			varData := models.NewVars()
			varData.SetTypeEscaped(new(models.VARS_VARS_TYPE))
			varData.SetAttributes(varAttrs)
			varBody := models.NewVarsEnvelope()
			varBody.SetData(varData)
			_, err := tfClient.API.Workspaces().ByWorkspace_id(wsID).Vars().Post(ctx, varBody, nil)
			Expect(err).Should(Succeed())

			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: wsID}
			// Create a new Module
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Make sure a new module is created and executed
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.ConfigurationVersion == nil {
					return false
				}
				return instance.Status.ConfigurationVersion.Status == "uploaded"
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.ConfigurationVersion.Status).NotTo(BeEquivalentTo("errored"))

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.Run.Status == string(models.APPLIED_RUNS_ATTRIBUTES_STATUS.String())
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.Run.Status).NotTo(BeEquivalentTo(string(models.ERRORED_RUNS_ATTRIBUTES_STATUS.String())))
		})
		It("External Deletion", func() {
			// Create a new HCP Terraform Workspace
			wsAttrs := models.NewWorkspaces_attributes()
			wsAttrs.SetName(&workspace)
			wsAttrs.SetAutoApply(new(true))
			wsAttrs.SetExecutionMode(new(models.REMOTE_WORKSPACES_ATTRIBUTES_EXECUTIONMODE))
			wsID := pointer.ValueOf(createWorkspace(wsAttrs).GetId())

			// Create TFC Workspace variables
			varAttrs := models.NewVars_attributes()
			varAttrs.SetKey(new("name"))
			varAttrs.SetValue(new("Pluto"))
			varAttrs.SetHcl(new(false))
			varAttrs.SetCategory(new(models.TERRAFORM_VARS_ATTRIBUTES_CATEGORY))
			varData := models.NewVars()
			varData.SetTypeEscaped(new(models.VARS_VARS_TYPE))
			varData.SetAttributes(varAttrs)
			varBody := models.NewVarsEnvelope()
			varBody.SetData(varData)
			_, err := tfClient.API.Workspaces().ByWorkspace_id(wsID).Vars().Post(ctx, varBody, nil)
			Expect(err).Should(Succeed())

			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: wsID}
			// Create a new Module
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Make sure a new module is created and executed
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.Run.Status == string(models.APPLIED_RUNS_ATTRIBUTES_STATUS.String())
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.Run.Status).NotTo(BeEquivalentTo(string(models.ERRORED_RUNS_ATTRIBUTES_STATUS.String())))

			// Manually run destrory
			runAttrs := models.NewRuns_attributes()
			runAttrs.SetIsDestroy(new(true))
			runWsData := models.NewWorkspacesHasOne_data()
			runWsData.SetId(&instance.Status.WorkspaceID)
			runWsData.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIER_TYPE))
			runWs := models.NewWorkspacesHasOne()
			runWs.SetData(runWsData)
			runRels := models.NewRuns_relationships()
			runRels.SetWorkspace(runWs)
			runData := models.NewRuns()
			runData.SetTypeEscaped(new(models.RUNS_RUNS_TYPE))
			runData.SetAttributes(runAttrs)
			runData.SetRelationships(runRels)
			runBody := models.NewRunsEnvelope()
			runBody.SetData(runData)
			run, err := tfClient.API.Runs().Post(ctx, runBody, nil)
			Expect(err).Should(Succeed())
			drID := pointer.ValueOf(run.GetData().GetId())
			Expect(drID).ShouldNot(BeEmpty())

			Expect(k8sClient.Delete(ctx, instance, &client.DeleteOptions{
				PropagationPolicy: pointer.PointerOf(metav1.DeletePropagationBackground),
			})).Should(Succeed())

			// Make sure the destroy run ID is the same as the manual one
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.DestroyRunID == drID
			}).Should(BeTrue())
		})
	})
})
