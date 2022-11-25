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
		instance  *appv1alpha2.Module
		workspace = fmt.Sprintf("kubernetes-operator-%v", GinkgoRandomSeed())
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(120 * time.Second)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		// Create a new module object for each test
		instance = &appv1alpha2.Module{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "app.terraform.io/v1alpha2",
				Kind:       "Module",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              namespacedName.Name,
				Namespace:         namespacedName.Namespace,
				DeletionTimestamp: nil,
				Finalizers:        []string{},
			},
			Spec: appv1alpha2.ModuleSpec{
				Organization: organization,
				Token: appv1alpha2.Token{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: namespacedName.Name,
						},
						Key: secretKey,
					},
				},
				Module: &appv1alpha2.ModuleSource{
					Source:  "app.terraform.io/kubernetes-operator/module-random/provider",
					Version: "0.0.4",
				},
				DestroyOnDeletion: true,
				Variables: []appv1alpha2.ModuleVariable{
					{
						Name: "string_length",
					},
				},
				Outputs: []appv1alpha2.ModuleOutput{
					{
						Name:      "bool",
						Sensitive: false,
					},
					{
						Name:      "secret",
						Sensitive: true,
					},
				},
				Workspace: &appv1alpha2.ModuleWorkspace{
					Name: workspace,
				},
			},
			Status: appv1alpha2.ModuleStatus{},
		}
	})

	AfterEach(func() {
		err := tfClient.Workspaces.Delete(ctx, organization, workspace)
		Expect(err).Should(Succeed())
	})

	Context("Module controller", func() {
		It("can create, run and destroy module", func() {
			// Create a new TFC Workspace
			ws, err := tfClient.Workspaces.Create(ctx, organization, tfc.WorkspaceCreateOptions{
				Name:      &workspace,
				AutoApply: tfc.Bool(true),
			})
			Expect(err).Should(Succeed())
			Expect(ws).ShouldNot(BeNil())

			// Create TFC Workspace variables
			_, err = tfClient.Variables.Create(ctx, ws.ID, tfc.VariableCreateOptions{
				Key:      tfc.String("string_length"),
				Value:    tfc.String("512"),
				HCL:      tfc.Bool(true),
				Category: tfc.Category(tfc.CategoryTerraform),
			})
			Expect(err).Should(Succeed())

			// Create a new Module
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Make sure a new module is created and executed
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.ConfigurationVersion == nil {
					return false
				}
				return instance.Status.ConfigurationVersion.Status == string(tfc.ConfigurationUploaded) ||
					// If the Configuration Version upload is errored then exit from the Eventually and validate it after
					instance.Status.ConfigurationVersion.Status == string(tfc.ConfigurationErrored)
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.ConfigurationVersion.Status).NotTo(BeEquivalentTo(string(tfc.ConfigurationErrored)))

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return instance.Status.Run.Status == string(tfc.RunApplied) ||
					// If the Run execution is errored then exit from the Eventually and validate it after
					instance.Status.Run.Status == string(tfc.RunErrored)
			}).Should(BeTrue())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Status.Run.Status).NotTo(BeEquivalentTo(string(tfc.RunErrored)))

			// Delete the Kubernetes Module object and wait until the controller finishes the reconciliation after deletion of the object
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				// The Kubernetes client will return error 'NotFound' on the "Get" operation once the object is deleted
				return errors.IsNotFound(err)
			}).Should(BeTrue())
		})
	})
})
