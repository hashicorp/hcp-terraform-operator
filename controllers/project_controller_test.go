// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Project controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Project
		namespacedName = newNamespacedName()
		project        = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
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
		// Create a new project object for each test
		instance = &appv1alpha2.Project{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "app.terraform.io/v1alpha2",
				Kind:       "Project",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              namespacedName.Name,
				Namespace:         namespacedName.Namespace,
				DeletionTimestamp: nil,
				Finalizers:        []string{},
			},
			Spec: appv1alpha2.ProjectSpec{
				Organization: organization,
				Token: appv1alpha2.Token{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretNamespacedName.Name,
						},
						Key: secretKey,
					},
				},
				Name: project,
			},
			Status: appv1alpha2.ProjectStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes Project object and wait until the controller finishes the reconciliation after deletion of the object
		Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, namespacedName, instance)
			// The Kubernetes client will return error 'NotFound' on the Get operation once the object is deleted
			return errors.IsNotFound(err)
		}).Should(BeTrue())

		// Make sure that the Terraform Cloud project is deleted
		Eventually(func() bool {
			err := tfClient.Projects.Delete(ctx, instance.Status.ID)
			// The Terraform Cloud client will return the error 'ResourceNotFound' once the workspace does not exist
			return err == tfc.ErrResourceNotFound || err == nil
		}).Should(BeTrue())
	})

	Context("Project controller", func() {
		It("can create and delete a project", func() {
			// Create a new Kubernetes project object and wait until the controller finishes the reconciliation
			createProject(instance)
		})
		It("can restore a project", func() {
			// Create a new Kubernetes project object and wait until the controller finishes the reconciliation
			createProject(instance)

			initProjectID := instance.Status.ID

			// Delete the Terraform Cloud project
			Expect(tfClient.Projects.Delete(ctx, instance.Status.ID)).Should(Succeed())

			// Wait until the controller re-creates the project and updates Status.ID with a new valid project ID
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ID != initProjectID
			}).Should(BeTrue())

			// The Kubernetes project object should have Status.ID with the valid project ID
			Expect(instance.Status.ID).Should(HavePrefix("prj-"))
		})
		It("can change basic project attributes", func() {
			// Create a new Kubernetes project object and wait until the controller finishes the reconciliation
			createProject(instance)

			// Update the Kubernetes project object Name
			instance.Spec.Name = fmt.Sprintf("%v-new", instance.Spec.Name)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			// Wait until the controller updates Terraform Cloud workspace
			Eventually(func() bool {
				prj, err := tfClient.Projects.Read(ctx, instance.Status.ID)
				Expect(prj).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return prj.Name == instance.Status.Name
			}).Should(BeTrue())
		})
		It("can revert external changes", func() {
			// Create a new Kubernetes project object and wait until the controller finishes the reconciliation
			createProject(instance)

			// Change the Terraform Cloud project name
			prj, err := tfClient.Projects.Update(ctx, instance.Status.ID, tfc.ProjectUpdateOptions{
				Name: tfc.String(fmt.Sprintf("%v-new", instance.Spec.Name)),
			})
			Expect(prj).ShouldNot(BeNil())
			Expect(err).Should(Succeed())

			// Wait until the controller updates Terraform Cloud project
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				Expect(err).Should(Succeed())
				prj, err := tfClient.Projects.Read(ctx, instance.Status.ID)
				Expect(prj).ShouldNot(BeNil())
				Expect(err).Should(Succeed())

				return prj.Name == instance.Status.Name
			}).Should(BeTrue())
		})
	})
})

func createProject(instance *appv1alpha2.Project) {
	namespacedName := getNamespacedName(instance)

	// Create a new Kubernetes project object
	Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
	// Wait until the controller finishes the reconciliation
	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		return instance.Status.ObservedGeneration == instance.Generation
	}).Should(BeTrue())

	// The Kubernetes project object should have Status.ID with the valid project ID
	Expect(instance.Status.ID).Should(HavePrefix("prj-"))
}
