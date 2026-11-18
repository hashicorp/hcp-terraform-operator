// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
		instance        *appv1alpha2.Workspace
		namespacedName  types.NamespacedName
		workspace       string
		secretVariables *corev1.Secret
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		// Create a secret object that will be used by the controller to get variables
		secretVariables = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-variables", namespacedName.Name),
				Namespace: namespacedName.Namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"test": []byte("test"),
			},
		}
		Expect(k8sClient.Create(ctx, secretVariables)).Should(Succeed())
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
		Expect(k8sClient.Delete(ctx, secretVariables)).Should(Succeed())
	})

	Context("Terraform Variables", func() {
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
			createWorkspaceResource(instance)
			expectVariables := workspaceVariableToTFC(instance, "terraform")
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
			expectVariables = workspaceVariableToTFC(instance, "terraform")
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates HCP Terraform workspace correcly
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Delete all workspace variables manually and wait until the controller re-create it
			variables := listWorkspaceVars(instance.Status.WorkspaceID)
			for _, v := range variables {
				tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Vars().ById(v.ID).Delete(ctx, nil)
			}
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Update workspace variable manually and wait until the controller reconcile it
			variables = listWorkspaceVars(instance.Status.WorkspaceID)
			for _, v := range variables {
				attrs := models.NewVars_attributes()
				attrs.SetValue(new("newnwenew"))
				data := models.NewVars()
				data.SetTypeEscaped(new(models.VARS_VARS_TYPE))
				data.SetId(&v.ID)
				data.SetAttributes(attrs)
				body := models.NewVarsEnvelope()
				body.SetData(data)
				tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Vars().ById(v.ID).Patch(ctx, body, nil)
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
			createWorkspaceResource(instance)
			expectVariables := workspaceVariableToTFC(instance, "terraform")
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
			expectVariables = workspaceVariableToTFC(instance, "terraform")
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates HCP Terraform workspace correcly
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
			createWorkspaceResource(instance)
			expectVariables := workspaceVariableToTFC(instance, "terraform")
			// Make sure that the TFC Workspace has all desired Terraform variables
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Can make a variable no sensitive
			instance.Spec.TerraformVariables = []appv1alpha2.Variable{
				{
					Name:        "test",
					Description: "test2",
					Value:       "test2",
					HCL:         false,
					Sensitive:   false,
				},
			}
			expectVariables = workspaceVariableToTFC(instance, "terraform")
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates HCP Terraform workspace correcly
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())
		})
	})

	Context("Environment Variables", func() {
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
			createWorkspaceResource(instance)
			expectVariables := workspaceVariableToTFC(instance, "env")
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
			expectVariables = workspaceVariableToTFC(instance, "env")
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates HCP Terraform workspace correcly
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Delete all workspace variables manually and wait until the controller re-create it
			variables := listWorkspaceVars(instance.Status.WorkspaceID)
			for _, v := range variables {
				tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Vars().ById(v.ID).Delete(ctx, nil)
			}
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Update workspace variable manually and wait until the controller reconcile it
			variables = listWorkspaceVars(instance.Status.WorkspaceID)
			for _, v := range variables {
				attrs := models.NewVars_attributes()
				attrs.SetValue(new("newnwenew"))
				data := models.NewVars()
				data.SetTypeEscaped(new(models.VARS_VARS_TYPE))
				data.SetId(&v.ID)
				data.SetAttributes(attrs)
				body := models.NewVarsEnvelope()
				body.SetData(data)
				tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Vars().ById(v.ID).Patch(ctx, body, nil)
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
			createWorkspaceResource(instance)
			expectVariables := workspaceVariableToTFC(instance, "env")
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
			expectVariables = workspaceVariableToTFC(instance, "env")
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates HCP Terraform workspace correcly
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
			createWorkspaceResource(instance)
			expectVariables := workspaceVariableToTFC(instance, "env")
			// Make sure that the TFC Workspace has all desired environment variables
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())

			// Can make a variable no sensitive
			instance.Spec.EnvironmentVariables = []appv1alpha2.Variable{
				{
					Name:        "test",
					Description: "test2",
					Value:       "test2",
					HCL:         false,
					Sensitive:   false,
				},
			}
			expectVariables = workspaceVariableToTFC(instance, "env")
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates HCP Terraform workspace correcly
			Eventually(func() bool {
				variables := listWorkspaceVars(instance.Status.WorkspaceID)
				return compareVars(variables, expectVariables)
			}).Should(BeTrue())
		})
	})
})

// variable is a workspace variable, a flat replacement for the former tfc.Variable.
type variable struct {
	ID          string
	Key         string
	Value       string
	Description string
	Category    string
	HCL         bool
	Sensitive   bool
}

// compareVars compares two slices of variables and returns 'true' if they are equal and 'false' otherwise. It ignores field 'ID'.
func compareVars(aVars, bVars []variable) bool {
	if len(aVars) != len(bVars) {
		return false
	}

	return cmp.Equal(aVars, bVars, cmpopts.IgnoreFields(variable{}, "ID"))
}

// listWorkspaceVars returns a list of all variables assigned to the workspace
func listWorkspaceVars(workspaceID string) []variable {
	wsVars, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Vars().Get(ctx, nil)
	Expect(wsVars).ShouldNot(BeNil())
	Expect(err).Should(Succeed())
	var variables []variable
	for _, v := range wsVars.GetData() {
		bv := variable{ID: pointer.ValueOf(v.GetId())}
		if a := v.GetAttributes(); a != nil {
			bv.Key = pointer.ValueOf(a.GetKey())
			bv.Value = pointer.ValueOf(a.GetValue())
			bv.Description = pointer.ValueOf(a.GetDescription())
			bv.HCL = pointer.ValueOf(a.GetHcl())
			bv.Sensitive = pointer.ValueOf(a.GetSensitive())
			if c := a.GetCategory(); c != nil {
				bv.Category = c.String()
			}
		}
		variables = append(variables, bv)
	}

	return variables
}

func workspaceVariableToTFC(instance *appv1alpha2.Workspace, category string) []variable {
	instanceVariables := []appv1alpha2.Variable{}

	switch category {
	case "env":
		instanceVariables = instance.Spec.EnvironmentVariables
	case "terraform":
		instanceVariables = instance.Spec.TerraformVariables
	}

	variables := make([]variable, len(instanceVariables))

	for i, v := range instanceVariables {
		value := v.Value
		if v.Sensitive {
			value = ""
		}
		variables[i] = variable{
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
