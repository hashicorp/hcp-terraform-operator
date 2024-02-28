// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName = types.NamespacedName{
			Name:      "this",
			Namespace: "default",
		}
		secretVariables *corev1.Secret
		workspace       = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create a secret object that will be used by the controller to get variables
		secretVariables = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "variables",
				Namespace: namespacedName.Namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"test": []byte("test"),
			},
		}
		Expect(k8sClient.Create(ctx, secretVariables)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, secretVariables)).Should(Succeed())
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

	Context("Reconcile terraform variables", func() {
		It("can handle workspace terraform variables", func() {
			instance.Spec.TerraformVariables = []appv1alpha2.Variable{
				{
					Name:        "test",
					Description: "test",
					Value:       "test",
					HCL:         false,
					Sensitive:   false,
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			expectVariables := workspaceVariableToTFC(instance, tfc.CategoryTerraform)
			// Make sure that the TFC Workspace has all desired Terraform variables
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Update the Kubernetes workspace terraform vars
			instance.Spec.TerraformVariables = []appv1alpha2.Variable{
				{
					Name:        "test2",
					Description: "test2",
					Value:       "test2",
					HCL:         false,
					Sensitive:   false,
				},
			}
			expectVariables = workspaceVariableToTFC(instance, tfc.CategoryTerraform)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates Terraform Cloud workspace correcly
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Delete all workspace variables manually and wait until the controller re-create it
			variables := listWorkspaceVars(instance.Status.WorkspaceID)
			for _, v := range variables {
				tfClient.Variables.Delete(ctx, instance.Status.WorkspaceID, v.ID)
			}
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Update workspace variable manually and wait until the controller reconcile it
			variables = listWorkspaceVars(instance.Status.WorkspaceID)
			for _, v := range variables {
				tfClient.Variables.Update(ctx, instance.Status.WorkspaceID, v.ID, tfc.VariableUpdateOptions{Value: tfc.String("newnwenew")})
			}
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())
		})

		It("can make workspace terraform variables sensitive", func() {
			instance.Spec.TerraformVariables = []appv1alpha2.Variable{
				{
					Name:        "test",
					Description: "test",
					Value:       "test",
					HCL:         false,
					Sensitive:   false,
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			expectVariables := workspaceVariableToTFC(instance, tfc.CategoryTerraform)
			// Make sure that the TFC Workspace has all desired Terraform variables
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Can make a variable sensitive
			instance.Spec.TerraformVariables = []appv1alpha2.Variable{
				{
					Name:        "test3",
					Description: "test3",
					Value:       "test3",
					HCL:         false,
					Sensitive:   true,
				},
			}
			expectVariables = workspaceVariableToTFC(instance, tfc.CategoryTerraform)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates Terraform Cloud workspace correcly
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())
		})

		It("can make workspace terraform variables no sensitive", func() {
			instance.Spec.TerraformVariables = []appv1alpha2.Variable{
				{
					Name:        "test",
					Description: "test",
					Value:       "test",
					HCL:         false,
					Sensitive:   true,
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			expectVariables := workspaceVariableToTFC(instance, tfc.CategoryTerraform)
			// Make sure that the TFC Workspace has all desired Terraform variables
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Can make a variable no sensitive
			instance.Spec.TerraformVariables = []appv1alpha2.Variable{
				{
					Name:        "test2",
					Description: "test2",
					Value:       "test2",
					HCL:         false,
					Sensitive:   false,
				},
			}
			expectVariables = workspaceVariableToTFC(instance, tfc.CategoryTerraform)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates Terraform Cloud workspace correcly
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())
		})
	})

	Context("Reconcile environment variables", func() {
		It("can handle workspace environment variables", func() {
			instance.Spec.EnvironmentVariables = []appv1alpha2.Variable{
				{
					Name:        "test",
					Description: "test",
					Value:       "test",
					HCL:         false,
					Sensitive:   false,
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			expectVariables := workspaceVariableToTFC(instance, tfc.CategoryEnv)
			// Make sure that the TFC Workspace has all desired Environment variables
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Update the Kubernetes workspace environment vars
			instance.Spec.EnvironmentVariables = []appv1alpha2.Variable{
				{
					Name:        "test2",
					Description: "test2",
					Value:       "test2",
					HCL:         false,
					Sensitive:   false,
				},
			}
			expectVariables = workspaceVariableToTFC(instance, tfc.CategoryEnv)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates Terraform Cloud workspace correcly
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Delete all workspace variables manually and wait until the controller re-create it
			variables := listWorkspaceVars(instance.Status.WorkspaceID)
			for _, v := range variables {
				tfClient.Variables.Delete(ctx, instance.Status.WorkspaceID, v.ID)
			}
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Update workspace variable manually and wait until the controller reconcile it
			variables = listWorkspaceVars(instance.Status.WorkspaceID)
			for _, v := range variables {
				tfClient.Variables.Update(ctx, instance.Status.WorkspaceID, v.ID, tfc.VariableUpdateOptions{Value: tfc.String("newnwenew")})
			}
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())
		})

		It("can make workspace environment variables sensitive", func() {
			instance.Spec.EnvironmentVariables = []appv1alpha2.Variable{
				{
					Name:        "test",
					Description: "test",
					Value:       "test",
					HCL:         false,
					Sensitive:   false,
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			expectVariables := workspaceVariableToTFC(instance, tfc.CategoryEnv)
			// Make sure that the TFC Workspace has all desired environment variables
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Can make a variable sensitive
			instance.Spec.EnvironmentVariables = []appv1alpha2.Variable{
				{
					Name:        "test3",
					Description: "test3",
					Value:       "test3",
					HCL:         false,
					Sensitive:   true,
				},
			}
			expectVariables = workspaceVariableToTFC(instance, tfc.CategoryEnv)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates Terraform Cloud workspace correcly
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())
		})

		It("can make workspace environment variables no sensitive", func() {
			instance.Spec.EnvironmentVariables = []appv1alpha2.Variable{
				{
					Name:        "test",
					Description: "test",
					Value:       "test",
					HCL:         false,
					Sensitive:   true,
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			expectVariables := workspaceVariableToTFC(instance, tfc.CategoryEnv)
			// Make sure that the TFC Workspace has all desired environment variables
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Can make a variable no sensitive
			instance.Spec.EnvironmentVariables = []appv1alpha2.Variable{
				{
					Name:        "test2",
					Description: "test2",
					Value:       "test2",
					HCL:         false,
					Sensitive:   false,
				},
			}
			expectVariables = workspaceVariableToTFC(instance, tfc.CategoryEnv)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates Terraform Cloud workspace correcly
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())
		})
	})
})

var _ = Describe("Makes Sensitive Workspace Variables No Sensitive", func() {
	Context("Update Variables", func() {
		It("Instance No Sensitive, Workspace No Sensitive", func() {
			instanceVariable := false
			workspaceVariable := false
			r := isNoLongerSensitive(instanceVariable, workspaceVariable)
			Expect(r).Should(BeFalse())
		})
		It("Instance No Sensitive, Workspace Sensitive", func() {
			instanceVariable := false
			workspaceVariable := true
			r := isNoLongerSensitive(instanceVariable, workspaceVariable)
			Expect(r).Should(BeTrue())
		})
		It("Instance Sensitive, Workspace No Sensitive", func() {
			instanceVariable := true
			workspaceVariable := false
			r := isNoLongerSensitive(instanceVariable, workspaceVariable)
			Expect(r).Should(BeFalse())
		})
		It("Instance Sensitive, Workspace Sensitive", func() {
			instanceVariable := true
			workspaceVariable := true
			r := isNoLongerSensitive(instanceVariable, workspaceVariable)
			Expect(r).Should(BeFalse())
		})
	})
})

// compareVars compares two slices of variables and returns 'true' if they are equal and 'false' otherwise. It ignores field 'ID'.
func compareVars(aVars, bVars []tfc.Variable) bool {
	if len(aVars) != len(bVars) {
		return false
	}

	return cmp.Equal(aVars, bVars, cmpopts.IgnoreFields(tfc.Variable{}, "ID", "VersionID", "Workspace"))
}

// listWorkspaceVars returns a list of all variables assigned to the workspace
func listWorkspaceVars(workspaceID string) []tfc.Variable {
	wsVars, err := tfClient.Variables.List(ctx, workspaceID, &tfc.VariableListOptions{})
	Expect(wsVars).ShouldNot(BeNil())
	Expect(err).Should(Succeed())
	var variables []tfc.Variable
	for _, v := range wsVars.Items {
		variables = append(variables, *v)
	}

	return variables
}

func workspaceVariableToTFC(instance *appv1alpha2.Workspace, category tfc.CategoryType) []tfc.Variable {
	instanceVariables := []appv1alpha2.Variable{}

	switch category {
	case tfc.CategoryEnv:
		instanceVariables = instance.Spec.EnvironmentVariables
	case tfc.CategoryTerraform:
		instanceVariables = instance.Spec.TerraformVariables
	}

	variables := make([]tfc.Variable, len(instanceVariables))

	for i, v := range instanceVariables {
		value := v.Value
		if v.Sensitive {
			value = ""
		}
		variables[i] = tfc.Variable{
			Key:         v.Name,
			Description: v.Description,
			Value:       value,
			HCL:         v.HCL,
			Sensitive:   v.Sensitive,
			Category:    category,
		}
	}

	return variables
}
