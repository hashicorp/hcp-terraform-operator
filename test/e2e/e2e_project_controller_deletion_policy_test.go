// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"errors"
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

var _ = Describe("Project controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Project
		namespacedName = newNamespacedName()
		project        = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		workspace      = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
	)

	BeforeAll(func() {
		if isEnterprise() {
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
				WithTransform(kerrors.IsNotFound, BeTrue()),
			),
		)
		Eventually(func() bool {
			return kerrors.IsNotFound(k8sClient.Get(ctx, namespacedName, instance))
		}).Should(BeTrue())

		Eventually(func() bool {
			err := tfClient.API.Projects().ByProject_id(instance.Status.ID).Delete(ctx, nil)
			return errors.Is(err, tfc.ErrNotFound) || err == nil
		}).Should(BeTrue())

		Eventually(func() bool {
			err := tfClient.API.Organizations().ByOrganization_name(organization).Workspaces().ByWorkspace_name(workspace).Delete(ctx, nil)
			return errors.Is(err, tfc.ErrNotFound) || err == nil
		}).Should(BeTrue())
	})

	Context("Deletion Policy", func() {
		It("can preserve a project", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.ProjectDeletionPolicyRetain
			createProject(instance)
			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				return kerrors.IsNotFound(k8sClient.Get(ctx, namespacedName, instance))
			}).Should(BeTrue())
			prj, err := tfClient.API.Projects().ByProject_id(instance.Status.ID).Get(ctx, nil)
			Expect(err).Should(Succeed())
			Expect(pointer.ValueOf(prj.GetData().GetAttributes().GetName())).NotTo(BeEmpty())
		})

		It("can soft delete a project", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.ProjectDeletionPolicySoft
			createProject(instance)
			// Create a workspace and assign it to a newley created project
			wsAttrs := models.NewWorkspaces_attributes()
			wsAttrs.SetName(&workspace)
			wsAttrs.SetAutoApply(new(false))
			wsPrj := models.NewProjectsHasOne_data()
			wsPrj.SetId(&instance.Status.ID)
			wsPrj.SetTypeEscaped(new(models.PROJECTS_PROJECTSIDENTIFIER_TYPE))
			wsPrjRel := models.NewProjectsHasOne()
			wsPrjRel.SetData(wsPrj)
			wsRels := models.NewWorkspaces_relationships()
			wsRels.SetProject(wsPrjRel)
			wsData := models.NewWorkspaces()
			wsData.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
			wsData.SetAttributes(wsAttrs)
			wsData.SetRelationships(wsRels)
			wsBody := models.NewWorkspacesEnvelope()
			wsBody.SetData(wsData)
			ws, err := tfClient.API.Organizations().ByOrganization_name(organization).Workspaces().Post(ctx, wsBody, nil)
			Expect(err).Should(Succeed())
			wsID := pointer.ValueOf(ws.GetData().GetId())
			Expect(wsID).NotTo(BeEmpty())
			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

			// Attach the workspace to the default project
			org, err := tfClient.API.Organizations().ByOrganization_name(instance.Spec.Organization).Get(ctx, nil)
			Expect(err).Should(Succeed())
			defPrj := models.NewProjectsHasOne_data()
			defPrj.SetId(org.GetData().GetRelationships().GetDefaultProject().GetData().GetId())
			defPrj.SetTypeEscaped(new(models.PROJECTS_PROJECTSIDENTIFIER_TYPE))
			defPrjRel := models.NewProjectsHasOne()
			defPrjRel.SetData(defPrj)
			updRels := models.NewWorkspaces_relationships()
			updRels.SetProject(defPrjRel)
			updData := models.NewWorkspaces()
			updData.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
			updData.SetId(&wsID)
			updData.SetRelationships(updRels)
			updBody := models.NewWorkspacesEnvelope()
			updBody.SetData(updData)
			_, err = tfClient.API.Workspaces().ByWorkspace_id(wsID).Patch(ctx, updBody, nil)
			Expect(err).Should(Succeed())

			// Wait until the CR is gone
			Eventually(func() bool {
				return kerrors.IsNotFound(k8sClient.Get(ctx, namespacedName, instance))
			}).Should(BeTrue())

			// Delete the workspace
			Expect(tfClient.API.Workspaces().ByWorkspace_id(wsID).Delete(ctx, nil)).Should(Succeed())
		})

	})

})
