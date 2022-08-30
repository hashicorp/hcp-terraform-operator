package controllers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		// ctx = context.TODO()

		instance  *appv1alpha2.Workspace
		workspace = fmt.Sprintf("kubernetes-operator-%v", GinkgoRandomSeed())

		sourceWorkspaceName = fmt.Sprintf("kubernetes-operator-source-%v", GinkgoRandomSeed())
		sourceWorkspaceID   = ""

		sourceWorkspaceName2 = fmt.Sprintf("kubernetes-operator-source2-%v", GinkgoRandomSeed())
		sourceWorkspaceID2   = ""
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(90 * time.Second)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create two new workspaces to act as a source for Run Triggers
		// Workspace[1]
		ws, err := tfClient.Workspaces.Create(ctx, organization, tfc.WorkspaceCreateOptions{
			Name: tfc.String(sourceWorkspaceName),
		})
		Expect(ws).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		sourceWorkspaceID = ws.ID

		// Workspace[2]
		ws, err = tfClient.Workspaces.Create(ctx, organization, tfc.WorkspaceCreateOptions{
			Name: tfc.String(sourceWorkspaceName2),
		})
		Expect(ws).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		sourceWorkspaceID2 = ws.ID
	})

	AfterAll(func() {
		err := tfClient.Workspaces.DeleteByID(ctx, sourceWorkspaceID)
		Expect(err).Should(Succeed())

		err = tfClient.Workspaces.DeleteByID(ctx, sourceWorkspaceID2)
		Expect(err).Should(Succeed())
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
				Name: workspace,
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {})

	Context("Workspace controller", func() {
		It("can handle run triggers by name", func() {
			instance.Spec.RunTriggers = []appv1alpha2.RunTrigger{
				{Name: sourceWorkspaceName},
				{Name: sourceWorkspaceName2},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)

			Eventually(func() bool {
				rt, err := tfClient.RunTriggers.List(ctx, instance.Status.WorkspaceID, &tfc.RunTriggerListOptions{
					RunTriggerType: tfc.RunTriggerInbound,
				})
				Expect(rt).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return hasWorkspaceSource(sourceWorkspaceID, rt) && hasWorkspaceSource(sourceWorkspaceID2, rt)
			}).Should(BeTrue())

			// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
			deleteWorkspace(instance, namespacedName)
		})

		It("can handle run triggers by ID", func() {
			instance.Spec.RunTriggers = []appv1alpha2.RunTrigger{
				{ID: sourceWorkspaceID},
				{ID: sourceWorkspaceID2},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)

			Eventually(func() bool {
				rt, err := tfClient.RunTriggers.List(ctx, instance.Status.WorkspaceID, &tfc.RunTriggerListOptions{
					RunTriggerType: tfc.RunTriggerInbound,
				})
				Expect(rt).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return hasWorkspaceSource(sourceWorkspaceID, rt) && hasWorkspaceSource(sourceWorkspaceID2, rt)
			}).Should(BeTrue())

			// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
			deleteWorkspace(instance, namespacedName)
		})

		It("can handle run triggers by mix of Name and ID", func() {
			instance.Spec.RunTriggers = []appv1alpha2.RunTrigger{
				{ID: sourceWorkspaceID},
				{Name: sourceWorkspaceName2},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)

			Eventually(func() bool {
				rt, err := tfClient.RunTriggers.List(ctx, instance.Status.WorkspaceID, &tfc.RunTriggerListOptions{
					RunTriggerType: tfc.RunTriggerInbound,
				})
				Expect(rt).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return hasWorkspaceSource(sourceWorkspaceID, rt) && hasWorkspaceSource(sourceWorkspaceID2, rt)
			}).Should(BeTrue())

			// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
			deleteWorkspace(instance, namespacedName)
		})
	})
})

func hasWorkspaceSource(workspaceID string, runTriggerList *tfc.RunTriggerList) bool {
	for _, v := range runTriggerList.Items {
		if workspaceID == v.Sourceable.ID {
			return true
		}
	}

	return false
}
