// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
	"github.com/hashicorp/terraform-cloud-operator/internal/pointer"
)

var _ = Describe("Module controller", Ordered, func() {
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
			return errors.IsNotFound(err)
		}).Should(BeTrue())

		// Make sure that the Terraform Cloud workspace is deleted
		Eventually(func() bool {
			err := tfClient.Workspaces.Delete(ctx, organization, workspace)
			// The Terraform Cloud client will return the error 'ResourceNotFound' once the workspace does not exist
			return err == tfc.ErrResourceNotFound || err == nil
		}).Should(BeTrue())
	})

	Context("Module controller", func() {
		It("can create, run and destroy module, ref to Workspace by Name", func() {
			// Create a new TFC Workspace
			ws, err := tfClient.Workspaces.Create(ctx, organization, tfc.WorkspaceCreateOptions{
				Name:      &workspace,
				AutoApply: tfc.Bool(true),
			})
			Expect(err).Should(Succeed())
			Expect(ws).ShouldNot(BeNil())

			// Create TFC Workspace variables
			_, err = tfClient.Variables.Create(ctx, ws.ID, tfc.VariableCreateOptions{
				Key:      tfc.String("name"),
				Value:    tfc.String("Pluto"),
				HCL:      tfc.Bool(false),
				Category: tfc.Category(tfc.CategoryTerraform),
			})
			Expect(err).Should(Succeed())

			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{Name: ws.Name}
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
				return instance.Status.ConfigurationVersion.Status == string(tfc.ConfigurationUploaded)
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.ConfigurationVersion.Status).NotTo(BeEquivalentTo(string(tfc.ConfigurationErrored)))

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.Run.Status == string(tfc.RunApplied)
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.Run.Status).NotTo(BeEquivalentTo(string(tfc.RunErrored)))
		})
		It("can create, run and destroy module, ref to Workspace by ID", func() {
			// Create a new TFC Workspace
			ws, err := tfClient.Workspaces.Create(ctx, organization, tfc.WorkspaceCreateOptions{
				Name:      &workspace,
				AutoApply: tfc.Bool(true),
			})
			Expect(err).Should(Succeed())
			Expect(ws).ShouldNot(BeNil())

			// Create TFC Workspace variables
			_, err = tfClient.Variables.Create(ctx, ws.ID, tfc.VariableCreateOptions{
				Key:      tfc.String("name"),
				Value:    tfc.String("Pluto"),
				HCL:      tfc.Bool(false),
				Category: tfc.Category(tfc.CategoryTerraform),
			})
			Expect(err).Should(Succeed())

			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: ws.ID}
			// Create a new Module
			instance.Spec.Name = "operator"
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
				return instance.Status.ConfigurationVersion.Status == string(tfc.ConfigurationUploaded)
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.ConfigurationVersion.Status).NotTo(BeEquivalentTo(string(tfc.ConfigurationErrored)))

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.Run.Status == string(tfc.RunApplied)
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.Run.Status).NotTo(BeEquivalentTo(string(tfc.RunErrored)))
		})
		It("can handle external deletion", func() {
			// Create a new TFC Workspace
			ws, err := tfClient.Workspaces.Create(ctx, organization, tfc.WorkspaceCreateOptions{
				Name:      &workspace,
				AutoApply: tfc.Bool(true),
			})
			Expect(err).Should(Succeed())
			Expect(ws).ShouldNot(BeNil())

			// Create TFC Workspace variables
			_, err = tfClient.Variables.Create(ctx, ws.ID, tfc.VariableCreateOptions{
				Key:      tfc.String("name"),
				Value:    tfc.String("Pluto"),
				HCL:      tfc.Bool(false),
				Category: tfc.Category(tfc.CategoryTerraform),
			})
			Expect(err).Should(Succeed())

			instance.Spec.Workspace = &appv1alpha2.ModuleWorkspace{ID: ws.ID}
			// Create a new Module
			instance.Spec.Name = "operator"
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
				return instance.Status.Run.Status == string(tfc.RunApplied)
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.Run.Status).NotTo(BeEquivalentTo(string(tfc.RunErrored)))

			// Manually run destrory
			dr, err := tfClient.Runs.Create(ctx, tfc.RunCreateOptions{
				IsDestroy: tfc.Bool(true),
				Workspace: &tfc.Workspace{
					ID: instance.Status.WorkspaceID,
				},
			})
			Expect(err).Should(Succeed())
			Expect(dr).ShouldNot(BeNil())

			Expect(k8sClient.Delete(ctx, instance, &client.DeleteOptions{
				PropagationPolicy: pointer.PointerOf(metav1.DeletePropagationBackground),
			})).Should(Succeed())

			// Make sure the destroy run ID is the same as the manual one
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.DestroyRunID == dr.ID
			}).Should(BeTrue())
		})
	})
})
