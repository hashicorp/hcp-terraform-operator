// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName = newNamespacedName()
		workspace      = fmt.Sprintf("kubernetes-operator-%v", randomNumber())

		projectName  = fmt.Sprintf("project-%v", randomNumber())
		projectName2 = fmt.Sprintf("%v-2", projectName)
		projectID    = ""
		projectID2   = ""
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create Projects
		projectID = createTestProject(projectName)
		projectID2 = createTestProject(projectName2)
	})

	AfterAll(func() {
		// Clean up the Project
		err := tfClient.Projects.Delete(ctx, projectID)
		Expect(err).Should(Succeed())

		err = tfClient.Projects.Delete(ctx, projectID2)
		Expect(err).Should(Succeed())
	})

	BeforeEach(func() {
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
				Name: workspace,
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance)
	})

	Context("Project", func() {
		It("can handle project by name", func() {
			instance.Spec.Project = &appv1alpha2.WorkspaceProject{Name: projectName}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledProjectByName(instance)

			// Update the Project by Name
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.Project = &appv1alpha2.WorkspaceProject{Name: projectName2}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledProjectByName(instance)

			// Update the Project by ID
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.Project = &appv1alpha2.WorkspaceProject{ID: projectID}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledProjectByID(instance)

			// Move workspace to default project
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.Project = nil
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledDefaultProject(instance)
		})

		It("can handle project by id", func() {
			instance.Spec.Project = &appv1alpha2.WorkspaceProject{ID: projectID}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledProjectByID(instance)

			// Update the Project by ID
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.Project = &appv1alpha2.WorkspaceProject{ID: projectID2}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledProjectByID(instance)

			// Update the Project by Name
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.Project = &appv1alpha2.WorkspaceProject{Name: projectName}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledProjectByName(instance)

			// Move workspace to default project
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.Project = nil
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledDefaultProject(instance)
		})
	})
})

func createTestProject(projectName string) string {
	prj, err := tfClient.Projects.Create(ctx, organization, tfc.ProjectCreateOptions{
		Name: projectName,
	})
	Expect(err).Should(Succeed())
	Expect(prj).ShouldNot(BeNil())
	return prj.ID
}

func isReconciledProjectByID(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(ws).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		return ws.Project.ID == instance.Spec.Project.ID
	}).Should(BeTrue())
}

func isReconciledProjectByName(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(ws).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		prj, err := tfClient.Projects.Read(ctx, ws.Project.ID)
		Expect(prj).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		return prj.Name == instance.Spec.Project.Name
	}).Should(BeTrue())
}

func isReconciledDefaultProject(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(ws).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		org, err := tfClient.Organizations.Read(ctx, instance.Spec.Organization)
		Expect(org).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		return ws.Project.ID == org.DefaultProject.ID
	}).Should(BeTrue())
}
