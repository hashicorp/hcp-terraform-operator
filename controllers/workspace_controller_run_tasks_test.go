// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Label("runTask"), Ordered, func() {
	var (
		instance  *appv1alpha2.Workspace
		workspace = fmt.Sprintf("kubernetes-operator-%v", GinkgoRandomSeed())
		// runTaskName = "kubernetes-operator-run-task" // fmt.Sprintf("kubernetes-operator-run-task-%v", GinkgoRandomSeed())
		runTaskID = "task-cyALezxxQBU4sfUQ" // ""
	)

	// KNOWN ISSUE
	//
	// Run Task should be created dynamically before run tests and then removed once tests are done.
	// However, due to a bug on the Terraform Cloud end, a Run Task cannot be removed immediately once the workspace is removed.
	// The Run Task remains associated with the deleted workspace due to the "cool down" period of ~15 minutes.
	//
	// Need to report this issue.

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create a Run Task
		// rt, err := tfClient.RunTasks.Create(ctx, organization, tfc.RunTaskCreateOptions{
		// 	Name:     runTaskName,
		// 	URL:      "https://example.com",
		// 	Category: "task", // MUST BE "task"
		// 	Enabled:  tfc.Bool(true),
		// })
		// Expect(err).Should(Succeed())
		// Expect(rt).ShouldNot(BeNil())
		// runTaskID = rt.ID
	})

	// AfterAll(func() {
	// 	err := tfClient.RunTasks.Delete(ctx, runTaskID)
	// 	Expect(err).Should(Succeed())
	// })

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
				RunTasks: []appv1alpha2.WorkspaceRunTask{
					{
						// At least one of the fields `ID` or `Name` is mandatory.
						// Set it up per test.
						Type:             "workspace-tasks", // MUST BE "workspace-tasks".
						EnforcementLevel: "advisory",
						Stage:            "post_plan",
					},
				},
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance, namespacedName)
	})

	Context("Workspace controller", func() {
		It("can create run task by ID", func() {
			instance.Spec.RunTasks[0].ID = runTaskID
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance, namespacedName)
		})

		// It("can create run task by Name", func() {
		// 	instance.Spec.RunTasks[0].ID = runTaskName
		// 	// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
		// 	createWorkspace(instance, namespacedName)
		// })

		// It("can delete run task", func() {
		// 	// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
		// 	createWorkspace(instance, namespacedName)
		// 	// NEED VALIDATION HERE
		// })

		// It("can restore deleted run task", func() {
		// 	// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
		// 	createWorkspace(instance, namespacedName)
		// 	// NEED VALIDATION HERE
		// })

		// It("can restore changed run task", func() {
		// 	// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
		// 	createWorkspace(instance, namespacedName)
		// 	// NEED VALIDATION HERE
		// })
	})
})
