// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
		if cloudEndpoint != tfcDefaultAddress {
			Skip("Does not run against TFC, skip this test")
		}
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
				Name:            workspace,
				ApplyMethod:     "auto",
				ApplyRunTrigger: "auto",
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		deleteWorkspace(instance)
	})

	Context("Runs", func() {
		It("can handle runs", func() {
			namespacedName := getNamespacedName(instance)
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			createAndUploadConfigurationVersion(instance.Status.WorkspaceID, "hoi", true)
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.Run.RunCompleted()
			}).Should(BeTrue())
		})

		It("can trigger a new run", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			createAndUploadConfigurationVersion(instance.Status.WorkspaceID, "hoi", true)
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.Run.RunCompleted()
			}).Should(BeTrue())

			// Trigger a new apply run with annotations
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.SetAnnotations(map[string]string{
				workspaceAnnotationRunNew:  annotationTrue,
				workspaceAnnotationRunType: runTypeApply,
			})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Annotations[workspaceAnnotationRunNew] == annotationTrue {
					return false
				}
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.Run.RunCompleted()
			}).Should(BeTrue())

			tf := "1.7.3"
			// Trigger a new plan run with annotations
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.SetAnnotations(map[string]string{
				workspaceAnnotationRunNew:              annotationTrue,
				workspaceAnnotationRunType:             runTypePlan,
				workspaceAnnotationRunTerraformVersion: tf,
			})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Annotations[workspaceAnnotationRunNew] == annotationTrue {
					return false
				}
				if instance.Status.Plan == nil {
					return false
				}
				return instance.Status.Plan.RunCompleted() && instance.Status.Plan.TerraformVersion == tf
			}).Should(BeTrue())
		})
	})
})
