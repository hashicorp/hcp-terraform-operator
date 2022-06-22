package controllers

import (
	"context"
	"fmt"
	"os"
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
		ctx = context.TODO()

		instance *appv1alpha2.Workspace
		secret   *corev1.Secret

		organization   = os.Getenv("TFC_ORG")
		terraformToken = os.Getenv("TFC_TOKEN")

		secretKey = "token"
		workspace = fmt.Sprintf("kubernetes-operator-%v", GinkgoRandomSeed())
	)

	namespacedName := types.NamespacedName{
		Name:      "this",
		Namespace: "default",
	}

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(90 * time.Second)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create a secret object that will be used by the controller
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey: []byte(terraformToken),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
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
				Token: appv1alpha2.SecretKeyRef{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secret.Name,
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
		// Take a pause before the next spec
		time.Sleep(2 * time.Second)
	})

	Context("Workspace controller", func() {
		It("can create and delete a workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)

			// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
			deleteWorkspace(instance, namespacedName)
		})

		It("can re-create a workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)

			initWorkspaceID := instance.Status.WorkspaceID

			// Delete the Terraform Cloud workspace
			Expect(tfClient.Workspaces.DeleteByID(ctx, instance.Status.WorkspaceID)).Should(Succeed())

			// Wait until the controller re-creates the workspace and updates Status.WorkspaceID with a new valid workspace ID
			Eventually(func() bool {
				k8sClient.Get(ctx, namespacedName, instance)
				return instance.Status.WorkspaceID != initWorkspaceID
			}).Should(BeTrue())

			// The Kubernetes workspace object should have Status.WorkspaceID with the valid workspace ID
			Expect(instance.Status.WorkspaceID).Should(HavePrefix("ws-"))

			// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
			deleteWorkspace(instance, namespacedName)
		})

		It("can clean up a workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)

			// Delete the Terraform Cloud workspace
			Expect(tfClient.Workspaces.DeleteByID(ctx, instance.Status.WorkspaceID)).Should(Succeed())

			// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
			deleteWorkspace(instance, namespacedName)
		})

		It("can update a workspace", func() {
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

			// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
			deleteWorkspace(instance, namespacedName)
		})
	})
})

func createWorkspace(instance *appv1alpha2.Workspace, namespacedName types.NamespacedName) {
	// Create a new Kubernetes workspace object
	Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
	// Wait until the controller finishes the reconciliation
	Eventually(func() bool {
		k8sClient.Get(ctx, namespacedName, instance)
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

	// Wait until the Terraform Cloud workspace is deleted
	Eventually(func() bool {
		k8sClient.Get(ctx, namespacedName, instance)
		_, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		// The Terraform Cloud client will return the error 'ResourceNotFound' on the "ReadByID" operation once the workspace is deleted
		return err == tfc.ErrResourceNotFound
	}).Should(BeTrue())
}
