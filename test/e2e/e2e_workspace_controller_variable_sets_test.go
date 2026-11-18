// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/controller"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName types.NamespacedName
		workspace      string

		variableSetName        string
		variableSetName2       string
		variableSetGlobalName  string
		variableSetProjectName string
		variableSetID          string
		variableSetID2         string
		variableSetGlobalID    string
		variableSetProjectID   string

		projectName string
		projectID   string
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		variableSetName = fmt.Sprintf("variable-set-%v", randomNumber())
		variableSetName2 = fmt.Sprintf("%v-2", variableSetName)
		variableSetGlobalName = fmt.Sprintf("variable-set-global-%v", randomNumber())
		variableSetProjectName = fmt.Sprintf("variable-set-project-%v", randomNumber())
		variableSetID = createVariableSet(variableSetName, false)
		variableSetID2 = createVariableSet(variableSetName2, false)
		variableSetGlobalID = createVariableSet(variableSetGlobalName, true)
		variableSetProjectID = createVariableSet(variableSetProjectName, false)

		// Add variable set to the project
		projectName = fmt.Sprintf("project-variable-set-%v", randomNumber())
		projectID = createTestProject(projectName)
		prjData := models.NewProjectsIdentifierArrayDocument_data()
		prjData.SetId(&projectID)
		prjData.SetTypeEscaped(new(models.PROJECTS_PROJECTSIDENTIFIERARRAYDOCUMENT_DATA_TYPE))
		prjBody := models.NewProjectsIdentifierArrayDocument()
		prjBody.SetData([]models.ProjectsIdentifierArrayDocument_dataable{prjData})
		tfClient.API.Varsets().ByVarset_id(variableSetProjectID).Relationships().Projects().Post(ctx, prjBody, nil)

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
		Expect(tfClient.API.Varsets().ByVarset_id(variableSetID).Delete(ctx, nil)).Should(Succeed())
		Expect(tfClient.API.Varsets().ByVarset_id(variableSetID2).Delete(ctx, nil)).Should(Succeed())
		Expect(tfClient.API.Varsets().ByVarset_id(variableSetGlobalID).Delete(ctx, nil)).Should(Succeed())
		Expect(tfClient.API.Projects().ByProject_id(projectID).Delete(ctx, nil)).Should(Succeed())
		Expect(tfClient.API.Varsets().ByVarset_id(variableSetProjectID).Delete(ctx, nil)).Should(Succeed())
	})

	Context("VariableSet", func() {
		It("can be handled by ID", func() {
			instance.Spec.VariableSets = []appv1alpha2.WorkspaceVariableSet{
				{ID: variableSetID},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledVariableSet(instance)

			// Ada a new empty variable set
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.VariableSets = append(instance.Spec.VariableSets, appv1alpha2.WorkspaceVariableSet{ID: variableSetID2})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledVariableSet(instance)

			// Manually remove variable set from the workspace
			removeWs := models.NewWorkspacesIdentifierArrayDocument_data()
			removeWs.SetId(&instance.Status.WorkspaceID)
			removeWs.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIERARRAYDOCUMENT_DATA_TYPE))
			removeBody := models.NewWorkspacesIdentifierArrayDocument()
			removeBody.SetData([]models.WorkspacesIdentifierArrayDocument_dataable{removeWs})
			Expect(tfClient.API.Varsets().ByVarset_id(variableSetID2).Relationships().Workspaces().Delete(ctx, removeBody, nil)).Should(Succeed())
			isReconciledVariableSet(instance)

			// Add a new global variable set
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.VariableSets = append(instance.Spec.VariableSets, appv1alpha2.WorkspaceVariableSet{ID: variableSetGlobalID})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledVariableSet(instance)

			// Add a variable set from the different project
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.VariableSets = append(instance.Spec.VariableSets, appv1alpha2.WorkspaceVariableSet{ID: variableSetProjectID})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledVariableSet(instance)

			// Manually remove different project variable set from the workspace
			removeWs2 := models.NewWorkspacesIdentifierArrayDocument_data()
			removeWs2.SetId(&instance.Status.WorkspaceID)
			removeWs2.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIERARRAYDOCUMENT_DATA_TYPE))
			removeBody2 := models.NewWorkspacesIdentifierArrayDocument()
			removeBody2.SetData([]models.WorkspacesIdentifierArrayDocument_dataable{removeWs2})
			Expect(tfClient.API.Varsets().ByVarset_id(variableSetProjectID).Relationships().Workspaces().Delete(ctx, removeBody2, nil)).Should(Succeed())
			isReconciledVariableSet(instance)
		})
	})

	Context("VariableSet", func() {
		It("can be handled by Name", func() {
			instance.Spec.VariableSets = []appv1alpha2.WorkspaceVariableSet{
				{Name: variableSetName},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledVariableSet(instance)

			// Ada a new empty variable set
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.VariableSets = append(instance.Spec.VariableSets, appv1alpha2.WorkspaceVariableSet{Name: variableSetName2})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledVariableSet(instance)

			// Manually remove variable set from the workspace
			removeWs := models.NewWorkspacesIdentifierArrayDocument_data()
			removeWs.SetId(&instance.Status.WorkspaceID)
			removeWs.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIERARRAYDOCUMENT_DATA_TYPE))
			removeBody := models.NewWorkspacesIdentifierArrayDocument()
			removeBody.SetData([]models.WorkspacesIdentifierArrayDocument_dataable{removeWs})
			Expect(tfClient.API.Varsets().ByVarset_id(variableSetID2).Relationships().Workspaces().Delete(ctx, removeBody, nil)).Should(Succeed())
			isReconciledVariableSet(instance)

			// Add a new global variable set
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.VariableSets = append(instance.Spec.VariableSets, appv1alpha2.WorkspaceVariableSet{Name: variableSetGlobalName})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledVariableSet(instance)

			// Add a variable set from the different project
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.VariableSets = append(instance.Spec.VariableSets, appv1alpha2.WorkspaceVariableSet{Name: variableSetProjectName})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledVariableSet(instance)

			// Manually remove different project variable set from the workspace
			removeWs2 := models.NewWorkspacesIdentifierArrayDocument_data()
			removeWs2.SetId(&instance.Status.WorkspaceID)
			removeWs2.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIERARRAYDOCUMENT_DATA_TYPE))
			removeBody2 := models.NewWorkspacesIdentifierArrayDocument()
			removeBody2.SetData([]models.WorkspacesIdentifierArrayDocument_dataable{removeWs2})
			Expect(tfClient.API.Varsets().ByVarset_id(variableSetProjectID).Relationships().Workspaces().Delete(ctx, removeBody2, nil)).Should(Succeed())
			isReconciledVariableSet(instance)
		})
	})
})

func createVariableSet(name string, global bool) string {
	attrs := models.NewVarsets_attributes()
	attrs.SetName(&name)
	attrs.SetGlobal(&global)
	data := models.NewVarsets()
	data.SetTypeEscaped(new(models.VARSETS_VARSETS_TYPE))
	data.SetAttributes(attrs)
	body := models.NewVarsetsEnvelope()
	body.SetData(data)
	vs, err := tfClient.API.Organizations().ByOrganization_name(organization).Varsets().Post(ctx, body, nil)
	Expect(err).Should(Succeed())
	Expect(vs).ShouldNot(BeNil())
	Expect(vs.GetData()).ShouldNot(BeNil())
	return pointer.ValueOf(vs.GetData().GetId())
}

func isReconciledVariableSet(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)
	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		return instance.Generation == instance.Status.ObservedGeneration
	}).Should(BeTrue())
	variableSets, err := tfClient.API.Organizations().ByOrganization_name(instance.Spec.Organization).Varsets().Get(ctx,
		&abstractions.RequestConfiguration[organizations.ItemVarsetsRequestBuilderGetQueryParameters]{
			QueryParameters: &organizations.ItemVarsetsRequestBuilderGetQueryParameters{
				Pagesize: new(int32(controller.MaxPageSize)),
			},
		})
	Expect(err).Should(Succeed())
	Expect(variableSets).ShouldNot(BeNil())
	vs := make(map[string]struct{})
	for _, v := range variableSets.GetData() {
		vs[pointer.ValueOf(v.GetId())] = struct{}{}
	}
	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		for _, v := range instance.Status.VariableSets {
			if _, ok := vs[v.ID]; !ok {
				return false
			}
		}
		return true
	}).Should(BeTrue())
}
