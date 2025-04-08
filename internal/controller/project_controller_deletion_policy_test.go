// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

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
		workspace      = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
	)

	BeforeAll(func() {
		if tfClient.IsEnterprise() {
			Skip("Does not run against Terraform Enterprise, skip this test")
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
		Expect(k8sClient.Delete(ctx, instance)).To(
			Or(
				Succeed(),
				WithTransform(errors.IsNotFound, BeTrue()),
			),
		)
		Eventually(func() bool {
			return errors.IsNotFound(k8sClient.Get(ctx, namespacedName, instance))
		}).Should(BeTrue())

		Eventually(func() bool {
			err := tfClient.Projects.Delete(ctx, instance.Status.ID)
			return err == tfc.ErrResourceNotFound || err == nil
		}).Should(BeTrue())

		Eventually(func() bool {
			err := tfClient.Workspaces.Delete(ctx, organization, workspace)
			return err == tfc.ErrResourceNotFound || err == nil
		}).Should(BeTrue())
	})

	Context("Deletion Policy", func() {
		It("can preserve a project", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.ProjectDeletionPolicyRetain
			createProject(instance)
			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				return errors.IsNotFound(k8sClient.Get(ctx, namespacedName, instance))
			}).Should(BeTrue())
			prj, err := tfClient.Projects.Read(ctx, instance.Status.ID)
			Expect(err).Should(Succeed())
			Expect(prj).NotTo(BeNil())
		})

		It("can soft delete a project", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.ProjectDeletionPolicySoft
			createProject(instance)
			// Create a workspace and assign it to a newley created project
			ws, err := tfClient.Workspaces.Create(ctx, organization, tfc.WorkspaceCreateOptions{
				Name: &workspace,
				Project: &tfc.Project{
					ID: instance.Status.ID,
				},
			})
			Expect(err).Should(Succeed())
			Expect(ws).NotTo(BeNil())
			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

			// Attach the workspace to the default project
			org, err := tfClient.Organizations.Read(ctx, instance.Spec.Organization)
			Expect(err).Should(Succeed())
			Expect(org).NotTo(BeNil())
			ws, err = tfClient.Workspaces.UpdateByID(ctx, ws.ID, tfc.WorkspaceUpdateOptions{
				Project: &tfc.Project{
					ID: org.DefaultProject.ID,
				},
			})
			Expect(err).Should(Succeed())
			Expect(ws).NotTo(BeNil())

			// Wait until the CR is gone
			Eventually(func() bool {
				return errors.IsNotFound(k8sClient.Get(ctx, namespacedName, instance))
			}).Should(BeTrue())

			// Delete the workspace
			Expect(tfClient.Workspaces.DeleteByID(ctx, ws.ID)).Should(Succeed())
		})

	})

})
