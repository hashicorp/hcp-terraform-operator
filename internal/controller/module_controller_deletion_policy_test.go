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

		// Make sure that the HCP Terraform workspace is deleted
		Eventually(func() bool {
			err := tfClient.Workspaces.Delete(ctx, organization, workspace)
			// The HCP Terraform client will return the error 'ResourceNotFound' once the workspace does not exist
			return err == tfc.ErrResourceNotFound || err == nil
		}).Should(BeTrue())
	})

	Context("Deletion Policy", func() {
		It("can preserve a module", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.ModuleDeletionPolicyRetain

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return err == nil
			}).Should(BeTrue())

			if instance.Spec.DestroyOnDeletion {
				Eventually(func() bool {
					err := tfClient.Workspaces.Delete(ctx, organization, workspace)
					return err == tfc.ErrResourceNotFound || err == nil
				}).Should(BeTrue())
			}

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())

		})
		It("can destroy a module", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.ModuleDeletionPolicyDestroy

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())

			if instance.Spec.DestroyOnDeletion {
				Eventually(func() bool {
					err := tfClient.Workspaces.Delete(ctx, organization, workspace)
					return err == tfc.ErrResourceNotFound || err == nil
				}).Should(BeTrue())
			}

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())

		})
	})

})
