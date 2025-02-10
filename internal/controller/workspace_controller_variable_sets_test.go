// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName types.NamespacedName
		workspace      string

		variableSetName  string
		variableSetName2 string
		variableSetID    string
		variableSetID2   string
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
		variableSetID = createTestVariableSet(variableSetName, false)
		variableSetID2 = createTestVariableSet(variableSetName2, false)

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
	})

	Context("VariableSet", func() {
		It("can be handled by ID", func() {
			instance.Spec.VariableSets = []appv1alpha2.WorkspaceVariableSet{
				{ID: variableSetID},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledVariableSetByID(instance)

			// Update the VariableSet by ID
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.VariableSets = []appv1alpha2.WorkspaceVariableSet{
				{ID: variableSetID2},
			}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledVariableSetByID(instance)
		})
	})
})

func createTestVariableSet(variableSetName string, global bool) string {
	vs, err := tfClient.VariableSets.Create(ctx, organization, &tfc.VariableSetCreateOptions{
		Name:   &variableSetName,
		Global: tfc.Bool(global),
	})
	Expect(err).Should(Succeed())
	Expect(vs).ShouldNot(BeNil())
	return vs.ID
}

func isReconciledVariableSetByID(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)
	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		variableSets, err := tfClient.VariableSets.List(ctx, instance.Spec.Organization, &tfc.VariableSetListOptions{
			ListOptions: tfc.ListOptions{
				PageSize: maxPageSize,
			},
		})
		Expect(err).Should(Succeed())
		for _, vs := range variableSets.Items {
			if vs.ID == instance.Spec.VariableSets[0].ID {
				return true
			}
		}
		return false
	}).Should(BeTrue())
}
