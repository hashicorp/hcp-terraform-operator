// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName types.NamespacedName
		workspace      string

		projectName  string
		projectName2 string
		projectID    string
		projectID2   string
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		projectName = fmt.Sprintf("project-%v", randomNumber())
		projectName2 = fmt.Sprintf("%v-2", projectName)
		projectID = createTestProject(projectName)
		projectID2 = createTestProject(projectName2)
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
		deleteWorkspace(instance)
		Expect(tfClient.API.Projects().ByProject_id(projectID).Delete(ctx, nil)).Should(Succeed())
		Expect(tfClient.API.Projects().ByProject_id(projectID2).Delete(ctx, nil)).Should(Succeed())
	})

	Context("Project", func() {
		It("can be handled by name", func() {
			instance.Spec.Project = &appv1alpha2.WorkspaceProject{Name: projectName}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
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

		It("can be handled by ID", func() {
			instance.Spec.Project = &appv1alpha2.WorkspaceProject{ID: projectID}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
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
	attrs := models.NewProjects_attributes()
	attrs.SetName(&projectName)
	data := models.NewProjects()
	data.SetTypeEscaped(new(models.PROJECTS_PROJECTS_TYPE))
	data.SetAttributes(attrs)
	body := models.NewProjectsEnvelope()
	body.SetData(data)
	prj, err := tfClient.API.Organizations().ByOrganization_name(organization).Projects().Post(ctx, body, nil)
	Expect(err).Should(Succeed())
	Expect(prj).ShouldNot(BeNil())
	Expect(prj.GetData()).ShouldNot(BeNil())
	return pointer.ValueOf(prj.GetData().GetId())
}

func isReconciledProjectByID(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		Expect(ws.GetData()).ShouldNot(BeNil())
		return pointer.ValueOf(ws.GetData().GetRelationships().GetProject().GetData().GetId()) == instance.Spec.Project.ID
	}).Should(BeTrue())
}

func isReconciledProjectByName(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		Expect(ws.GetData()).ShouldNot(BeNil())
		prj, err := tfClient.API.Projects().ByProject_id(pointer.ValueOf(ws.GetData().GetRelationships().GetProject().GetData().GetId())).Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(prj).ShouldNot(BeNil())
		Expect(prj.GetData()).ShouldNot(BeNil())
		Expect(prj.GetData().GetAttributes()).ShouldNot(BeNil())
		return pointer.ValueOf(prj.GetData().GetAttributes().GetName()) == instance.Spec.Project.Name
	}).Should(BeTrue())
}

func isReconciledDefaultProject(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		Expect(ws.GetData()).ShouldNot(BeNil())
		org, err := tfClient.API.Organizations().ByOrganization_name(instance.Spec.Organization).Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(org).ShouldNot(BeNil())
		Expect(org.GetData()).ShouldNot(BeNil())
		Expect(org.GetData().GetRelationships()).ShouldNot(BeNil())
		Expect(org.GetData().GetRelationships().GetDefaultProject()).ShouldNot(BeNil())
		Expect(org.GetData().GetRelationships().GetDefaultProject().GetData()).ShouldNot(BeNil())
		defaultProjectID := pointer.ValueOf(org.GetData().GetRelationships().GetDefaultProject().GetData().GetId())
		return defaultProjectID == pointer.ValueOf(ws.GetData().GetRelationships().GetProject().GetData().GetId()) && defaultProjectID == instance.Status.DefaultProjectID
	}).Should(BeTrue())
}
