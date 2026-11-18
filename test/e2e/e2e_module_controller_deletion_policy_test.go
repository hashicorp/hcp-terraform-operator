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

	"github.com/hashicorp/go-tfe/v2/api/models"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

var _ = Describe("Module сontroller", Ordered, func() {
	var (
		instance       *appv1alpha2.Module
		namespacedName = newNamespacedName()
		workspaceName  = ""
		workspaceID    string
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
		workspaceName = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		// Create a new HCP Terraform Workspace
		wsAttrs := models.NewWorkspaces_attributes()
		wsAttrs.SetName(&workspaceName)
		wsAttrs.SetAutoApply(new(true))
		wsAttrs.SetExecutionMode(new(models.REMOTE_WORKSPACES_ATTRIBUTES_EXECUTIONMODE))
		workspaceID = pointer.ValueOf(createWorkspace(wsAttrs).GetId())

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
		_, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Vars().Post(ctx, varBody, nil)
		Expect(err).Should(Succeed())
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
		// Make sure that the HCP Terraform workspace is deleted
		Eventually(func() bool {
			err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Delete(ctx, nil)
			// The HCP Terraform client will return the error 'ResourceNotFound' once the workspace does not exist
			return errors.Is(err, tfc.ErrNotFound) || err == nil
		}).Should(BeTrue())
	})

	Context("Module Deletion Policy", func() {
		It("can handle no destroy on deletion and retain deleteion policy", func() {
			// Create a new Module
			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: workspaceID}
			instance.Spec.DestroyOnDeletion = false
			instance.Spec.DeletionPolicy = appv1alpha2.ModuleDeletionPolicyRetain
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
			// Make sure a new module is created and executed
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.ObservedGeneration == instance.Generation &&
					instance.Status.Run.Status == string(models.APPLIED_RUNS_ATTRIBUTES_STATUS.String())
			}).Should(BeTrue())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return kerrors.IsNotFound(err)
			}).Should(BeTrue())

			ws, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Get(ctx, nil)
			Expect(err).Should(Succeed())
			currentRun := ws.GetData().GetRelationships().GetCurrentRun()
			Expect(currentRun).ShouldNot(BeNil())
			Expect(currentRun.GetData()).ShouldNot(BeNil())
			currentRunID := pointer.ValueOf(currentRun.GetData().GetId())
			Expect(currentRunID).ShouldNot(BeEmpty())

			r, err := tfClient.API.Runs().ById(currentRunID).Get(ctx, nil)
			Expect(err).Should(Succeed())
			Expect(pointer.ValueOf(r.GetData().GetAttributes().GetIsDestroy())).To(BeFalse())
		})

		It("can handle no destroy on deletion and destroy deleteion policy", func() {
			// Create a new Module
			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: workspaceID}
			instance.Spec.DestroyOnDeletion = false
			instance.Spec.DeletionPolicy = appv1alpha2.ModuleDeletionPolicyDestroy
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
			// Make sure a new module is created and executed
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.ObservedGeneration == instance.Generation &&
					instance.Status.Run.Status == string(models.APPLIED_RUNS_ATTRIBUTES_STATUS.String())
			}).Should(BeTrue())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return kerrors.IsNotFound(err)
			}).Should(BeTrue())

			ws, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Get(ctx, nil)
			Expect(err).Should(Succeed())
			currentRun := ws.GetData().GetRelationships().GetCurrentRun()
			Expect(currentRun).ShouldNot(BeNil())
			Expect(currentRun.GetData()).ShouldNot(BeNil())
			currentRunID := pointer.ValueOf(currentRun.GetData().GetId())
			Expect(currentRunID).ShouldNot(BeEmpty())

			r, err := tfClient.API.Runs().ById(currentRunID).Get(ctx, nil)
			Expect(err).Should(Succeed())
			Expect(pointer.ValueOf(r.GetData().GetAttributes().GetIsDestroy())).To(BeTrue())
		})

		It("can handle destroy on deletion and retain deleteion policy", func() {
			// Create a new Module
			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: workspaceID}
			instance.Spec.DestroyOnDeletion = true
			instance.Spec.DeletionPolicy = appv1alpha2.ModuleDeletionPolicyRetain
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
			// Make sure a new module is created and executed
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.ObservedGeneration == instance.Generation &&
					instance.Status.Run.Status == string(models.APPLIED_RUNS_ATTRIBUTES_STATUS.String())
			}).Should(BeTrue())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return kerrors.IsNotFound(err)
			}).Should(BeTrue())

			ws, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Get(ctx, nil)
			Expect(err).Should(Succeed())
			currentRun := ws.GetData().GetRelationships().GetCurrentRun()
			Expect(currentRun).ShouldNot(BeNil())
			Expect(currentRun.GetData()).ShouldNot(BeNil())
			currentRunID := pointer.ValueOf(currentRun.GetData().GetId())
			Expect(currentRunID).ShouldNot(BeEmpty())

			r, err := tfClient.API.Runs().ById(currentRunID).Get(ctx, nil)
			Expect(err).Should(Succeed())
			Expect(pointer.ValueOf(r.GetData().GetAttributes().GetIsDestroy())).To(BeTrue())
		})

		It("can handle destroy on deletion and destroy deleteion policy", func() {
			// Create a new Module
			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: workspaceID}
			instance.Spec.DestroyOnDeletion = true
			instance.Spec.DeletionPolicy = appv1alpha2.ModuleDeletionPolicyDestroy
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
			// Make sure a new module is created and executed
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.ObservedGeneration == instance.Generation &&
					instance.Status.Run.Status == string(models.APPLIED_RUNS_ATTRIBUTES_STATUS.String())
			}).Should(BeTrue())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return kerrors.IsNotFound(err)
			}).Should(BeTrue())

			ws, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Get(ctx, nil)
			Expect(err).Should(Succeed())
			currentRun := ws.GetData().GetRelationships().GetCurrentRun()
			Expect(currentRun).ShouldNot(BeNil())
			Expect(currentRun.GetData()).ShouldNot(BeNil())
			currentRunID := pointer.ValueOf(currentRun.GetData().GetId())
			Expect(currentRunID).ShouldNot(BeEmpty())

			r, err := tfClient.API.Runs().ById(currentRunID).Get(ctx, nil)
			Expect(err).Should(Succeed())
			Expect(pointer.ValueOf(r.GetData().GetAttributes().GetIsDestroy())).To(BeTrue())
		})
	})

})
