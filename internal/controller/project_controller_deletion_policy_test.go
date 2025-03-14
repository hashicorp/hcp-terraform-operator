// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

var _ = Describe("Project controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Project
		namespacedName = newNamespacedName()
		project        = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
	)

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

		// Make sure that the HCP Terraform project is deleted
		Eventually(func() bool {
			err := tfClient.Projects.Delete(ctx, instance.Status.ID)
			// The HCP Terraform client will return the error 'ResourceNotFound' once the workspace does not exist
			return err == tfc.ErrResourceNotFound || err == nil
		}).Should(BeTrue())
	})

	Context("Deletion Policy", func() {
		It("can preserve a project", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.ProjectDeletionPolicyRetain
			createProject(instance)
			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())
			prj, err := tfClient.Projects.Read(ctx, instance.Status.ID)
			//workspace, err := tfClient.Workspaces.ReadByID(ctx, workspaceID)
			Expect(err).Should(Succeed())
			Expect(prj).NotTo(BeNil())
		})

		It("can soft delete a project", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.ProjectDeletionPolicySoft
			createProject(instance)

			projectID := instance.Status.ID

			Eventually(func() bool {
				_, err := tfClient.Projects.Read(ctx, projectID)
				return err == tfc.ErrResourceNotFound
			}).Should(BeTrue())
		})
	})

})
