// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName = newNamespacedName()
		workspace      = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
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
				Name:             workspace,
				ApplyMethod:      "auto",
				AllowDestroyPlan: false,
				Description:      "Description",
				ExecutionMode:    "remote",
				WorkingDirectory: "aws/us-west-1/vpc",
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance)
	})

	Context("Workspace controller", func() {
		It("can create and delete a workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
		})

		It("can re-create a workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			initWorkspaceID := instance.Status.WorkspaceID

			// Delete the Terraform Cloud workspace
			Expect(tfClient.Workspaces.DeleteByID(ctx, instance.Status.WorkspaceID)).Should(Succeed())

			// Wait until the controller re-creates the workspace and updates Status.WorkspaceID with a new valid workspace ID
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.WorkspaceID != initWorkspaceID
			}).Should(BeTrue())

			// The Kubernetes workspace object should have Status.WorkspaceID with the valid workspace ID
			Expect(instance.Status.WorkspaceID).Should(HavePrefix("ws-"))
		})

		It("can clean up a workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			// Delete the Terraform Cloud workspace
			Expect(tfClient.Workspaces.DeleteByID(ctx, instance.Status.WorkspaceID)).Should(Succeed())
		})

		It("can change workspace name", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			// Update the Kubernetes workspace object Name
			instance.Spec.Name = fmt.Sprintf("%v-new", instance.Spec.Name)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			// Wait until the controller updates Terraform Cloud workspace
			Eventually(func() bool {
				ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
				Expect(ws).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return ws.Name == instance.Spec.Name
			}).Should(BeTrue())
		})

		It("can change basic workspace attributes", func() {
			// Make a copy of the original instance to then compare Workspace against it
			// It helps to make sure that the 'Spec' part is not mutated
			instanceCopy := instance.DeepCopy()

			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			// Validate that all attributes are set correctly
			// Do not validate the Terraform version since it will be set to the latest available by default
			Eventually(func() bool {
				ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
				Expect(ws).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return ws.AutoApply == applyMethodToBool(instanceCopy.Spec.ApplyMethod) &&
					ws.AllowDestroyPlan == instanceCopy.Spec.AllowDestroyPlan &&
					ws.Description == instanceCopy.Spec.Description &&
					ws.ExecutionMode == instanceCopy.Spec.ExecutionMode &&
					ws.WorkingDirectory == instanceCopy.Spec.WorkingDirectory
			}).Should(BeTrue())

			// Update the Kubernetes workspace object fields(basic workspace attributes)
			instance.Spec.ApplyMethod = "manual"
			instance.Spec.AllowDestroyPlan = true
			instance.Spec.Description = fmt.Sprintf("%v-new", instance.Spec.Description)
			instance.Spec.ExecutionMode = "local"
			instance.Spec.TerraformVersion = "1.4.4"
			instance.Spec.WorkingDirectory = fmt.Sprintf("%v/new", instance.Spec.WorkingDirectory)

			// Make a copy of the original instance to then compare Workspace against it
			// It helps to make sure that the 'Spec' part is not mutated
			instanceCopy = instance.DeepCopy()

			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			// Wait until the controller updates Terraform Cloud workspace
			Eventually(func() bool {
				ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
				Expect(ws).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return ws.AutoApply == applyMethodToBool(instanceCopy.Spec.ApplyMethod) &&
					ws.AllowDestroyPlan == instanceCopy.Spec.AllowDestroyPlan &&
					ws.Description == instanceCopy.Spec.Description &&
					ws.ExecutionMode == instanceCopy.Spec.ExecutionMode &&
					ws.TerraformVersion == instanceCopy.Spec.TerraformVersion &&
					ws.WorkingDirectory == instanceCopy.Spec.WorkingDirectory
			}).Should(BeTrue())
		})

		It("can keep Terraform version", func() {
			instance.Spec.TerraformVersion = "1.4.1"
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			// Remove TerraformVersion from the 'spec'
			instance.Spec.TerraformVersion = ""
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			// Wait until the controller updates Terraform Cloud workspace
			Eventually(func() bool {
				ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
				Expect(ws).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return instance.Spec.TerraformVersion == "" &&
					ws.TerraformVersion == instance.Status.TerraformVersion
			}).Should(BeTrue())
		})

		It("can handle workspace tags", func() {
			expectTags := []tfc.Tag{
				{Name: "kubernetes-operator"},
				{Name: "env:dev"},
			}

			instance.Spec.Tags = []appv1alpha2.Tag{"kubernetes-operator"}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			// Make sure that the TFC Workspace has all desired tags
			Eventually(func() bool {
				wsTags := listWorkspaceTags(instance.Status.WorkspaceID)
				expectTags := []tfc.Tag{
					{Name: "kubernetes-operator"},
				}
				return compareTags(wsTags, expectTags)
			}).Should(BeTrue())

			// Update the Kubernetes workspace tags
			instance.Spec.Tags = []appv1alpha2.Tag{"kubernetes-operator", "env:dev"}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates Terraform Cloud workspace correcly
			Eventually(func() bool {
				wsTags := listWorkspaceTags(instance.Status.WorkspaceID)
				return compareTags(wsTags, expectTags)
			}).Should(BeTrue())

			// Delete some tags manually
			err := tfClient.Workspaces.RemoveTags(ctx, instance.Status.WorkspaceID, tfc.WorkspaceRemoveTagsOptions{Tags: []*tfc.Tag{
				{Name: "kubernetes-operator"},
			}})
			Expect(err).Should(Succeed())
			// Make sure the controller restores all tags
			Eventually(func() bool {
				wsTags := listWorkspaceTags(instance.Status.WorkspaceID)
				return compareTags(wsTags, expectTags)
			}).Should(BeTrue())

			// Add some tags manually
			err = tfClient.Workspaces.AddTags(ctx, instance.Status.WorkspaceID, tfc.WorkspaceAddTagsOptions{Tags: []*tfc.Tag{
				{Name: "new-tag"},
			}})
			Expect(err).Should(Succeed())
			// Make sure the controller restores all tags
			Eventually(func() bool {
				wsTags := listWorkspaceTags(instance.Status.WorkspaceID)
				return compareTags(wsTags, expectTags)
			}).Should(BeTrue())
		})
	})
})

func createWorkspace(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	// Create a new Kubernetes workspace object
	Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
	// Wait until the controller finishes the reconciliation
	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		return instance.Status.ObservedGeneration == instance.Generation
	}).Should(BeTrue())

	// The Kubernetes workspace object should have Status.WorkspaceID with the valid workspace ID
	Expect(instance.Status.WorkspaceID).Should(HavePrefix("ws-"))
}

func deleteWorkspace(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	// Delete the Kubernetes workspace object
	Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

	// Wait until the controller finishes the reconciliation after the deletion of the object
	Eventually(func() bool {
		err := k8sClient.Get(ctx, namespacedName, instance)
		// The Kubernetes client will return error 'NotFound' on the "Get" operation once the object is deleted
		return errors.IsNotFound(err)
	}).Should(BeTrue())

	// Make sure that the Terraform Cloud workspace is deleted
	Eventually(func() bool {
		err := tfClient.Workspaces.Delete(ctx, instance.Spec.Organization, instance.Spec.Name)
		// The Terraform Cloud client will return the error 'ResourceNotFound' once the workspace does not exist
		return err == tfc.ErrResourceNotFound || err == nil
	}).Should(BeTrue())
}

// compareTags compares two slices of tags and returns 'true' if they are equal and 'false' otherwise
func compareTags(aTags, bTags []tfc.Tag) bool {
	if len(aTags) != len(bTags) {
		return false
	}

	// TODO find a better way to do this, i.e DeepEqual
	for _, at := range aTags {
		found := false
		for _, bt := range bTags {
			if at.Name == bt.Name {
				found = true
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// listWorkspaceTags returns a list of all tags assigned to the workspace
func listWorkspaceTags(workspaceID string) []tfc.Tag {
	ws, err := tfClient.Workspaces.ReadByID(ctx, workspaceID)
	Expect(ws).ShouldNot(BeNil())
	Expect(err).Should(Succeed())
	tags := make([]tfc.Tag, len(ws.TagNames))
	for i, t := range ws.TagNames {
		tags[i] = tfc.Tag{Name: t}
	}

	return tags
}
