// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/controller"
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
		tfClient.VariableSets.ApplyToProjects(ctx, variableSetProjectID, tfc.VariableSetApplyToProjectsOptions{
			Projects: []*tfc.Project{
				{ID: projectID},
			},
		})

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
		Expect(tfClient.VariableSets.Delete(ctx, variableSetID)).Should(Succeed())
		Expect(tfClient.VariableSets.Delete(ctx, variableSetID2)).Should(Succeed())
		Expect(tfClient.VariableSets.Delete(ctx, variableSetGlobalID)).Should(Succeed())
		Expect(tfClient.Projects.Delete(ctx, projectID)).Should(Succeed())
		Expect(tfClient.VariableSets.Delete(ctx, variableSetProjectID)).Should(Succeed())
	})

	Context("VariableSet", func() {
		It("can be handled by ID", func() {
			instance.Spec.VariableSets = []appv1alpha2.WorkspaceVariableSet{
				{ID: variableSetID},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledVariableSet(instance)

			// Ada a new empty variable set
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.VariableSets = append(instance.Spec.VariableSets, appv1alpha2.WorkspaceVariableSet{ID: variableSetID2})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledVariableSet(instance)

			// Manually remove variable set from the workspace
			Expect(tfClient.VariableSets.RemoveFromWorkspaces(ctx, variableSetID2, &tfc.VariableSetRemoveFromWorkspacesOptions{
				Workspaces: []*tfc.Workspace{
					{ID: instance.Status.WorkspaceID},
				},
			})).Should(Succeed())
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
			Expect(tfClient.VariableSets.RemoveFromWorkspaces(ctx, variableSetProjectID, &tfc.VariableSetRemoveFromWorkspacesOptions{
				Workspaces: []*tfc.Workspace{
					{ID: instance.Status.WorkspaceID},
				},
			})).Should(Succeed())
			isReconciledVariableSet(instance)
		})
	})

	Context("VariableSet", func() {
		It("can be handled by Name", func() {
			instance.Spec.VariableSets = []appv1alpha2.WorkspaceVariableSet{
				{Name: variableSetName},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledVariableSet(instance)

			// Ada a new empty variable set
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.VariableSets = append(instance.Spec.VariableSets, appv1alpha2.WorkspaceVariableSet{Name: variableSetName2})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledVariableSet(instance)

			// Manually remove variable set from the workspace
			Expect(tfClient.VariableSets.RemoveFromWorkspaces(ctx, variableSetID2, &tfc.VariableSetRemoveFromWorkspacesOptions{
				Workspaces: []*tfc.Workspace{
					{ID: instance.Status.WorkspaceID},
				},
			})).Should(Succeed())
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
			Expect(tfClient.VariableSets.RemoveFromWorkspaces(ctx, variableSetProjectID, &tfc.VariableSetRemoveFromWorkspacesOptions{
				Workspaces: []*tfc.Workspace{
					{ID: instance.Status.WorkspaceID},
				},
			})).Should(Succeed())
			isReconciledVariableSet(instance)
		})
	})
})

func createVariableSet(name string, global bool) string {
	vs, err := tfClient.VariableSets.Create(ctx, organization, &tfc.VariableSetCreateOptions{
		Name:   &name,
		Global: tfc.Bool(global),
	})
	Expect(err).Should(Succeed())
	Expect(vs).ShouldNot(BeNil())
	return vs.ID
}

func isReconciledVariableSet(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)
	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		return instance.Generation == instance.Status.ObservedGeneration
	}).Should(BeTrue())
	variableSets, err := tfClient.VariableSets.List(ctx, instance.Spec.Organization, &tfc.VariableSetListOptions{
		ListOptions: tfc.ListOptions{
			PageSize: controller.MaxPageSize,
		},
	})
	vs := make(map[string]struct{})
	for _, v := range variableSets.Items {
		vs[v.ID] = struct{}{}
	}
	Expect(err).Should(Succeed())
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
