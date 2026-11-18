// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"
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
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		if cloudEndpoint != tfcDefaultAddress {
			Skip("Does not run against TFC, skip this test")
		}
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
				Name:        workspace,
				ApplyMethod: "auto",
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		deleteWorkspace(instance)
	})

	Context("Outputs", func() {
		It("can handle outputs", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)

			outputValue := "hoi"
			cvID := createAndUploadConfigurationVersion(instance.Status.WorkspaceID, outputValue)

			By("Validating configuration version and workspace run")
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}

				runs, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Runs().Get(ctx, nil)
				Expect(err).Should(Succeed())
				Expect(runs).ShouldNot(BeNil())

				for _, r := range runs.GetData() {
					if instance.Status.Run == nil {
						return false
					}
					if pointer.ValueOf(r.GetRelationships().GetConfigurationVersion().GetData().GetId()) == cvID && pointer.ValueOf(r.GetId()) == instance.Status.Run.OutputRunID {
						return true
					}
				}
				return false
			}).Should(BeTrue())

			outputsNamespacedName := types.NamespacedName{
				Name:      controller.OutputObjectName(namespacedName.Name),
				Namespace: namespacedName.Namespace,
			}

			s := &corev1.Secret{}
			By("Validating sensitive outputs")
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, outputsNamespacedName, s)).Should(Succeed())
				if v, ok := s.Data["sensitive"]; ok {
					if string(v) == outputValue {
						return true
					}
				}
				return false
			}).Should(BeTrue())

			cm := &corev1.ConfigMap{}
			By("Validating non-sensitive outputs")
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, outputsNamespacedName, cm)).Should(Succeed())
				if v, ok := cm.Data["non_sensitive"]; ok {
					if v == outputValue {
						return true
					}
				}
				return false
			}).Should(BeTrue())
		})
	})
})

// createAndUploadConfigurationVersion creates a configuration version, uploads Terraform code to it, and returns its ID.
func createAndUploadConfigurationVersion(workspaceID string, outputValue string) string {
	GinkgoHelper()
	// Create a temporary dir in the current one
	cd, err := os.Getwd()
	Expect(err).Should(Succeed())
	td, err := os.MkdirTemp(cd, "tf-*")
	Expect(err).Should(Succeed())
	defer os.RemoveAll(td)
	// Create a temporary file in the temporary dir
	f, err := os.CreateTemp(td, "*.tf")
	Expect(err).Should(Succeed())
	defer os.Remove(f.Name())
	// Terraform code to upload
	tf := fmt.Sprintf(`
				resource "random_uuid" "this" {}
				output "sensitive" {
					value = %[1]q
					sensitive = true
				}
				output "non_sensitive" {
					value = %[1]q
				}`, outputValue)
	// Save the Terraform code to the temporary file
	_, err = f.WriteString(tf)
	Expect(err).Should(Succeed())

	cvAttrs := models.NewConfigurationVersions_attributes()
	cvAttrs.SetAutoQueueRuns(new(true))
	cvAttrs.SetSpeculative(new(false))
	cvData := models.NewConfigurationVersions()
	cvData.SetTypeEscaped(new(models.CONFIGURATIONVERSIONS_CONFIGURATIONVERSIONS_TYPE))
	cvData.SetAttributes(cvAttrs)
	cvBody := models.NewConfigurationVersionsEnvelope()
	cvBody.SetData(cvData)
	cvResp, err := tfClient.API.Workspaces().ByWorkspace_id(workspaceID).ConfigurationVersions().Post(ctx, cvBody, nil)
	Expect(err).Should(Succeed())
	Expect(cvResp).ShouldNot(BeNil())
	cv := cvResp.GetData()
	Expect(cv).ShouldNot(BeNil())
	Expect(cv.GetAttributes()).ShouldNot(BeNil())
	cvID := pointer.ValueOf(cv.GetId())

	Expect(uploadConfigurationVersion(pointer.ValueOf(cv.GetAttributes().GetUploadUrl()), td)).Should(Succeed())

	Eventually(func() bool {
		c, err := tfClient.API.ConfigurationVersions().ByConfiguration_version_id(cvID).Get(ctx, nil)
		if err != nil || c == nil || c.GetData() == nil || c.GetData().GetAttributes() == nil || c.GetData().GetAttributes().GetStatus() == nil {
			return false
		}
		return c.GetData().GetAttributes().GetStatus().String() == "uploaded"
	}).Should(BeTrue())

	return cvID
}
