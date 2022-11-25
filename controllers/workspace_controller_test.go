package controllers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance  *appv1alpha2.Workspace
		workspace = fmt.Sprintf("kubernetes-operator-%v", GinkgoRandomSeed())
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
							Name: namespacedName.Name,
						},
						Key: secretKey,
					},
				},
				Name:             workspace,
				ApplyMethod:      "auto",
				AllowDestroyPlan: true,
				Description:      "Description",
				ExecutionMode:    "remote",
				TerraformVersion: "1.2.3",
				WorkingDirectory: "aws/us-west-1/vpc",
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance, namespacedName)
	})

	Context("Workspace controller", func() {
		It("can create and delete a workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)
		})

		It("can re-create a workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)

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
			createWorkspace(instance, namespacedName)

			// Delete the Terraform Cloud workspace
			Expect(tfClient.Workspaces.DeleteByID(ctx, instance.Status.WorkspaceID)).Should(Succeed())
		})

		It("can change workspace name", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)

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
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)

			// Update the Kubernetes workspace object fields(basic workspace attributes)
			instance.Spec.ApplyMethod = "manual"
			instance.Spec.AllowDestroyPlan = false
			instance.Spec.Description = fmt.Sprintf("%v-new", instance.Spec.Description)
			instance.Spec.ExecutionMode = "local"
			instance.Spec.TerraformVersion = "1.2.1"
			instance.Spec.WorkingDirectory = fmt.Sprintf("%v/new", instance.Spec.WorkingDirectory)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			// Wait until the controller updates Terraform Cloud workspace
			Eventually(func() bool {
				ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
				Expect(ws).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return ws.AutoApply == applyMethodToBool(instance.Spec.ApplyMethod) &&
					ws.AllowDestroyPlan == instance.Spec.AllowDestroyPlan &&
					ws.Description == instance.Spec.Description &&
					ws.ExecutionMode == instance.Spec.ExecutionMode &&
					ws.TerraformVersion == instance.Spec.TerraformVersion &&
					ws.WorkingDirectory == instance.Spec.WorkingDirectory
			}).Should(BeTrue())
		})

		It("can handle workspace tags", func() {
			expectTags := []tfc.Tag{
				{Name: "kubernetes-operator"},
				{Name: "env:dev"},
			}

			instance.Spec.Tags = []string{"kubernetes-operator"}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)
			// Make sure that the TFC Workspace has all desired tags
			Eventually(func() bool {
				wsTags := listWorkspaceTags(instance.Status.WorkspaceID)
				expectTags := []tfc.Tag{
					{Name: "kubernetes-operator"},
				}
				return compareTags(wsTags, expectTags)
			}).Should(BeTrue())

			// Update the Kubernetes workspace tags
			instance.Spec.Tags = []string{"kubernetes-operator", "env:dev"}
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

func createWorkspace(instance *appv1alpha2.Workspace, namespacedName types.NamespacedName) {
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

func deleteWorkspace(instance *appv1alpha2.Workspace, namespacedName types.NamespacedName) {
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
