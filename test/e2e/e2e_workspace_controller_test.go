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
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
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
				ApplyRunTrigger:  "auto",
				AllowDestroyPlan: false,
				Description:      "Description",
				ExecutionMode:    "remote",
				WorkingDirectory: "aws/us-west-1/vpc",
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		deleteWorkspace(instance)
	})

	Context("Workspace controller", func() {
		It("can create and delete a workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
		})

		It("can re-create a workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			initWorkspaceID := instance.Status.WorkspaceID

			// Delete the HCP Terraform workspace
			Expect(tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Delete(ctx, nil)).Should(Succeed())

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
			createWorkspaceResource(instance)

			// Delete the HCP Terraform workspace
			Expect(tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Delete(ctx, nil)).Should(Succeed())
		})

		It("can change workspace name", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			// Update the Kubernetes workspace object Name
			instance.Spec.Name = fmt.Sprintf("%v-new", instance.Spec.Name)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			// Wait until the controller updates HCP Terraform workspace
			Eventually(func() bool {
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				return pointer.ValueOf(ws.GetName()) == instance.Spec.Name
			}).Should(BeTrue())
		})

		It("can change basic workspace attributes", func() {
			// Make a copy of the original instance to then compare Workspace against it
			// It helps to make sure that the 'Spec' part is not mutated
			instanceCopy := instance.DeepCopy()

			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			// Validate that all attributes are set correctly
			// Do not validate the Terraform version since it will be set to the latest available by default
			Eventually(func() bool {
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				return pointer.ValueOf(ws.GetAutoApply()) == controller.ApplyMethodToBool(instanceCopy.Spec.ApplyMethod) &&
					pointer.ValueOf(ws.GetAutoApplyRunTrigger()) == controller.ApplyRunTriggerToBool(instanceCopy.Spec.ApplyRunTrigger) &&
					pointer.ValueOf(ws.GetAllowDestroyPlan()) == instanceCopy.Spec.AllowDestroyPlan &&
					pointer.ValueOf(ws.GetDescription()) == instanceCopy.Spec.Description &&
					pointer.ValueOf(ws.GetExecutionMode()).String() == instanceCopy.Spec.ExecutionMode &&
					pointer.ValueOf(ws.GetWorkingDirectory()) == instanceCopy.Spec.WorkingDirectory
			}).Should(BeTrue())

			// Update the Kubernetes workspace object fields(basic workspace attributes)
			instance.Spec.ApplyMethod = "manual"
			instance.Spec.ApplyRunTrigger = "manual"
			instance.Spec.AllowDestroyPlan = true
			instance.Spec.Description = fmt.Sprintf("%v-new", instance.Spec.Description)
			instance.Spec.ExecutionMode = "local"
			instance.Spec.TerraformVersion = "1.4.4"
			instance.Spec.WorkingDirectory = fmt.Sprintf("%v/new", instance.Spec.WorkingDirectory)

			// Make a copy of the original instance to then compare Workspace against it
			// It helps to make sure that the 'Spec' part is not mutated
			instanceCopy = instance.DeepCopy()

			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			// Wait until the controller updates HCP Terraform workspace
			Eventually(func() bool {
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				return pointer.ValueOf(ws.GetAutoApply()) == controller.ApplyMethodToBool(instanceCopy.Spec.ApplyMethod) &&
					pointer.ValueOf(ws.GetAutoApplyRunTrigger()) == controller.ApplyRunTriggerToBool(instanceCopy.Spec.ApplyRunTrigger) &&
					pointer.ValueOf(ws.GetAllowDestroyPlan()) == instanceCopy.Spec.AllowDestroyPlan &&
					pointer.ValueOf(ws.GetDescription()) == instanceCopy.Spec.Description &&
					pointer.ValueOf(ws.GetExecutionMode()).String() == instanceCopy.Spec.ExecutionMode &&
					pointer.ValueOf(ws.GetTerraformVersion()) == instanceCopy.Spec.TerraformVersion &&
					pointer.ValueOf(ws.GetWorkingDirectory()) == instanceCopy.Spec.WorkingDirectory
			}).Should(BeTrue())
		})

		It("can keep Terraform version", func() {
			instance.Spec.TerraformVersion = "1.4.1"
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			// Remove TerraformVersion from the 'spec'
			instance.Spec.TerraformVersion = ""
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			// Wait until the controller updates HCP Terraform workspace
			Eventually(func() bool {
				resp, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				ws := resp.GetData().GetAttributes()
				return instance.Spec.TerraformVersion == "" &&
					pointer.ValueOf(ws.GetTerraformVersion()) == instance.Status.TerraformVersion
			}).Should(BeTrue())
		})

		It("can handle workspace tags", func() {
			expectTags := []string{"kubernetes-operator", "env:dev"}

			instance.Spec.Tags = []appv1alpha2.Tag{"kubernetes-operator"}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			// Make sure that the TFC Workspace has all desired tags
			Eventually(func() bool {
				wsTags := listWorkspaceTags(instance.Status.WorkspaceID)
				expectTags := []string{"kubernetes-operator"}
				return compareTags(wsTags, expectTags)
			}).Should(BeTrue())

			// Update the Kubernetes workspace tags
			instance.Spec.Tags = []appv1alpha2.Tag{"kubernetes-operator", "env:dev"}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			// Wait until the controller updates HCP Terraform workspace correcly
			Eventually(func() bool {
				wsTags := listWorkspaceTags(instance.Status.WorkspaceID)
				return compareTags(wsTags, expectTags)
			}).Should(BeTrue())

			// Delete some tags manually
			tagsBuilder := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Relationships().Tags()
			tags, err := tagsBuilder.Get(ctx, nil)
			Expect(err).Should(Succeed())
			for _, t := range tags.GetData() {
				if pointer.ValueOf(t.GetAttributes().GetName()) == "kubernetes-operator" {
					tag := models.NewTagsRemoveArrayDocument_data()
					tag.SetId(t.GetId())
					tag.SetTypeEscaped(new(models.TAGS_TAGSREMOVEARRAYDOCUMENT_DATA_TYPE))
					body := models.NewTagsRemoveArrayDocument()
					body.SetData([]models.TagsRemoveArrayDocument_dataable{tag})
					Expect(tagsBuilder.Delete(ctx, body, nil)).Should(Succeed())
				}
			}
			// Make sure the controller restores all tags
			Eventually(func() bool {
				wsTags := listWorkspaceTags(instance.Status.WorkspaceID)
				return compareTags(wsTags, expectTags)
			}).Should(BeTrue())

			// Add some tags manually
			tagAttrs := models.NewTagsCreateArrayDocument_data_attributes()
			tagAttrs.SetName(new("new-tag"))
			tag := models.NewTagsCreateArrayDocument_data()
			tag.SetTypeEscaped(new(models.TAGS_TAGSCREATEARRAYDOCUMENT_DATA_TYPE))
			tag.SetAttributes(tagAttrs)
			body := models.NewTagsCreateArrayDocument()
			body.SetData([]models.TagsCreateArrayDocument_dataable{tag})
			Expect(tagsBuilder.Post(ctx, body, nil)).Should(Succeed())
			// Make sure the controller restores all tags
			Eventually(func() bool {
				wsTags := listWorkspaceTags(instance.Status.WorkspaceID)
				return compareTags(wsTags, expectTags)
			}).Should(BeTrue())
		})
	})
})

func createWorkspaceResource(instance *appv1alpha2.Workspace) {
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
	Expect(k8sClient.Delete(ctx, instance)).To(
		Or(
			Succeed(),
			WithTransform(kerrors.IsNotFound, BeTrue()),
		),
	)

	// Wait until the controller finishes the reconciliation after the deletion of the object
	Eventually(func() bool {
		err := k8sClient.Get(ctx, namespacedName, instance)
		// The Kubernetes client will return error 'NotFound' on the "Get" operation once the object is deleted
		return kerrors.IsNotFound(err)
	}).Should(BeTrue())

	// Make sure that the HCP Terraform workspace is deleted
	Eventually(func() bool {
		err := tfClient.API.Organizations().ByOrganization_name(instance.Spec.Organization).Workspaces().ByWorkspace_name(instance.Spec.Name).Delete(ctx, nil)
		// The HCP Terraform client will return the error 'NotFound' once the workspace does not exist
		return errors.Is(err, tfc.ErrNotFound) || err == nil
	}).Should(BeTrue())
}

// compareTags compares two slices of tags and returns 'true' if they are equal and 'false' otherwise
func compareTags(aTags, bTags []string) bool {
	if len(aTags) != len(bTags) {
		return false
	}

	// TODO find a better way to do this, i.e DeepEqual
	for _, at := range aTags {
		found := false
		for _, bt := range bTags {
			if at == bt {
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
func listWorkspaceTags(workspaceID string) []string {
	resp, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).Get(ctx, nil)
	Expect(err).Should(Succeed())

	return resp.GetData().GetAttributes().GetTagNames()
}
