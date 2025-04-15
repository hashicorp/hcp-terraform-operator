// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Module Controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Module
		namespacedName = newNamespacedName()
		workspaceName  = ""
		workspace      *tfc.Workspace
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
		var err error
		workspaceName = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		// Create a new TFC Workspace
		workspace, err = tfClient.Workspaces.Create(ctx, organization, tfc.WorkspaceCreateOptions{
			Name:      &workspaceName,
			AutoApply: tfc.Bool(true),
		})
		Expect(err).Should(Succeed())
		Expect(workspace).ShouldNot(BeNil())

		// Create TFC Workspace variables
		_, err = tfClient.Variables.Create(ctx, workspace.ID, tfc.VariableCreateOptions{
			Key:      tfc.String("name"),
			Value:    tfc.String("Pluto"),
			HCL:      tfc.Bool(false),
			Category: tfc.Category(tfc.CategoryTerraform),
		})
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
			err := tfClient.Workspaces.DeleteByID(ctx, workspace.ID)
			// The HCP Terraform client will return the error 'ResourceNotFound' once the workspace does not exist
			return err == tfc.ErrResourceNotFound || err == nil
		}).Should(BeTrue())
	})

	Context("Module Deletion Policy", func() {
		It("can handle no destroy on deletion and retain deleteion policy", func() {
			// Create a new Module
			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: workspace.ID}
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
					instance.Status.Run.Status == string(tfc.RunApplied)
			}).Should(BeTrue())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())

			workspace, err := tfClient.Workspaces.ReadByID(ctx, workspace.ID)
			Expect(err).Should(Succeed())
			Expect(workspace).ShouldNot(BeNil())

			r, err := tfClient.Runs.Read(ctx, workspace.CurrentRun.ID)
			Expect(err).Should(Succeed())
			Expect(r).ShouldNot(BeNil())
			Expect(r.IsDestroy).To(BeFalse())
		})

		It("can handle no destroy on deletion and destroy deleteion policy", func() {
			// Create a new Module
			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: workspace.ID}
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
					instance.Status.Run.Status == string(tfc.RunApplied)
			}).Should(BeTrue())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())

			workspace, err := tfClient.Workspaces.ReadByID(ctx, workspace.ID)
			Expect(err).Should(Succeed())
			Expect(workspace).ShouldNot(BeNil())

			r, err := tfClient.Runs.Read(ctx, workspace.CurrentRun.ID)
			Expect(err).Should(Succeed())
			Expect(r).ShouldNot(BeNil())
			Expect(r.IsDestroy).To(BeTrue())
		})

		It("can handle destroy on deletion and retain deleteion policy", func() {
			// Create a new Module
			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: workspace.ID}
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
					instance.Status.Run.Status == string(tfc.RunApplied)
			}).Should(BeTrue())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())

			workspace, err := tfClient.Workspaces.ReadByID(ctx, workspace.ID)
			Expect(err).Should(Succeed())
			Expect(workspace).ShouldNot(BeNil())

			r, err := tfClient.Runs.Read(ctx, workspace.CurrentRun.ID)
			Expect(err).Should(Succeed())
			Expect(r).ShouldNot(BeNil())
			Expect(r.IsDestroy).To(BeTrue())
		})

		It("can handle destroy on deletion and destroy deleteion policy", func() {
			// Create a new Module
			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: workspace.ID}
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
					instance.Status.Run.Status == string(tfc.RunApplied)
			}).Should(BeTrue())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())

			workspace, err := tfClient.Workspaces.ReadByID(ctx, workspace.ID)
			Expect(err).Should(Succeed())
			Expect(workspace).ShouldNot(BeNil())

			r, err := tfClient.Runs.Read(ctx, workspace.CurrentRun.ID)
			Expect(err).Should(Succeed())
			Expect(r).ShouldNot(BeNil())
			Expect(r.IsDestroy).To(BeTrue())
		})
	})

})
