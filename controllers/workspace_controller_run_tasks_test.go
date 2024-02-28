// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tfc "github.com/hashicorp/go-tfe"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName = newNamespacedName()
		workspace      = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		runTaskName    = fmt.Sprintf("kubernetes-operator-run-task-%v", randomNumber())
		runTaskName2   = fmt.Sprintf("kubernetes-operator-run-task-2-%v", randomNumber())
		runTaskID      = ""
		runTaskID2     = ""
	)

	// KNOWN ISSUE
	//
	// Run Task should be created dynamically before run tests and then removed once tests are done.
	// However, due to a bug on the Terraform Cloud end, a Run Task cannot be removed immediately once the workspace is removed.
	// The Run Task remains associated with the deleted workspace due to the "cool down" period of ~15 minutes.
	//
	// IPL-3276.

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
				Name: workspace,
				RunTasks: []appv1alpha2.WorkspaceRunTask{
					{
						// At least one of the fields `ID` or `Name` is mandatory.
						// Set it up per test.
						EnforcementLevel: "advisory",
						Stage:            "post_plan",
					},
				},
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}

		runTaskID = createRunTaskForTest(runTaskName)
		runTaskID2 = createRunTaskForTest(runTaskName2)
	})

	AfterEach(func() {
		// Delete all Run Tasks from the Workspace before deleting the Workspace, otherwise, it won't be possible to delete the Run Tasks instantly.
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		instance.Spec.RunTasks = []appv1alpha2.WorkspaceRunTask{}
		Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
		isRunTasksReconciled(instance)
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance)
		// Delete Run Task 1
		err := tfClient.RunTasks.Delete(ctx, runTaskID)
		Expect(err).Should(Succeed())
		// Delete Run Task 2
		err = tfClient.RunTasks.Delete(ctx, runTaskID2)
		Expect(err).Should(Succeed())
	})

	Context("Workspace controller", func() {
		It("can create run task by ID", func() {
			instance.Spec.RunTasks[0].ID = runTaskID
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isRunTasksReconciled(instance)
		})

		It("can create run task by Name", func() {
			instance.Spec.RunTasks[0].Name = runTaskName
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isRunTasksReconciled(instance)
		})

		It("can create 2 run tasks by Name and ID", func() {
			instance.Spec.RunTasks[0].Name = runTaskName
			instance.Spec.RunTasks = append(instance.Spec.RunTasks, appv1alpha2.WorkspaceRunTask{
				ID:               runTaskID2,
				EnforcementLevel: "advisory",
				Stage:            "post_plan",
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isRunTasksReconciled(instance)
		})

		It("can delete run task", func() {
			instance.Spec.RunTasks[0].ID = runTaskID
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isRunTasksReconciled(instance)

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			// Delete Run Tasks from the spec
			instance.Spec.RunTasks = []appv1alpha2.WorkspaceRunTask{}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isRunTasksReconciled(instance)
		})

		It("can update run task", func() {
			instance.Spec.RunTasks[0].ID = runTaskID
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isRunTasksReconciled(instance)

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.RunTasks[0].EnforcementLevel = "mandatory" // was "advisory"

			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isRunTasksReconciled(instance)
		})

		It("can restore deleted run task", func() {
			instance.Spec.RunTasks[0].ID = runTaskID
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isRunTasksReconciled(instance)

			deleteWorkspaceRunTasks(instance)
			isRunTasksReconciled(instance)
		})

		It("can revert manual changes in run task", func() {
			instance.Spec.RunTasks[0].ID = runTaskID
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isRunTasksReconciled(instance)

			// Make a manual change
			runTasksList, err := tfClient.WorkspaceRunTasks.List(ctx, instance.Status.WorkspaceID, &tfc.WorkspaceRunTaskListOptions{})
			Expect(err).Should(Succeed())
			Expect(runTasksList).ShouldNot(BeNil())
			Expect(runTasksList.Items).Should(HaveLen(1))
			runTasks := runTasksList.Items
			_, err = tfClient.WorkspaceRunTasks.Update(ctx, instance.Status.WorkspaceID, runTasks[0].ID, tfc.WorkspaceRunTaskUpdateOptions{
				EnforcementLevel: "mandatory",                          // was "advisory"
				Stage:            (*tfc.Stage)(tfc.String("pre_plan")), // was "post_plan"
			})
			Expect(err).Should(Succeed())
			isRunTasksReconciled(instance)
		})
	})
})

func deleteWorkspaceRunTasks(instance *appv1alpha2.Workspace) {
	rtList, err := tfClient.WorkspaceRunTasks.List(ctx, instance.Status.WorkspaceID, &tfc.WorkspaceRunTaskListOptions{})
	Expect(err).Should(Succeed())
	Expect(rtList).ShouldNot(BeNil())

	for _, rt := range rtList.Items {
		err := tfClient.WorkspaceRunTasks.Delete(ctx, instance.Status.WorkspaceID, rt.ID)
		Expect(err).Should(Succeed())
	}

	Eventually(func() bool {
		rtList, err = tfClient.WorkspaceRunTasks.List(ctx, instance.Status.WorkspaceID, &tfc.WorkspaceRunTaskListOptions{})
		Expect(err).Should(Succeed())
		Expect(rtList).ShouldNot(BeNil())
		return len(rtList.Items) == 0
	}).Should(BeTrue())
}

func isRunTasksReconciled(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		// Get a slice of all Run Tasks that are defined in spec
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		s := make(map[string][]string)
		// This hendle the case when Spec.RunTasks is empty, so we can do less API calls.
		if len(instance.Spec.RunTasks) > 0 {
			runTasksList, err := tfClient.RunTasks.List(ctx, instance.Spec.Organization, &tfc.RunTaskListOptions{})
			Expect(err).Should(Succeed())
			Expect(runTasksList).ShouldNot(BeNil())
			for _, r := range instance.Spec.RunTasks {
				id := r.ID
				if r.Name != "" {
					for _, rt := range runTasksList.Items {
						if rt.Name == r.Name {
							id = rt.ID
						}
					}
				}
				s[id] = []string{r.EnforcementLevel, r.Stage}
			}
		}

		// Get a slice of all Run Tasks that are assigned to the Workspace
		runTasks, err := tfClient.WorkspaceRunTasks.List(ctx, instance.Status.WorkspaceID, &tfc.WorkspaceRunTaskListOptions{})
		Expect(err).Should(Succeed())
		Expect(runTasks).ShouldNot(BeNil())
		w := make(map[string][]string)
		for _, r := range runTasks.Items {
			w[r.RunTask.ID] = []string{string(r.EnforcementLevel), string(r.Stage)}
		}

		return cmp.Equal(s, w)
	}).Should(BeTrue())
}

func createRunTaskForTest(name string) string {
	rt, err := tfClient.RunTasks.Create(ctx, organization, tfc.RunTaskCreateOptions{
		Name:     name,
		URL:      "https://example.com",
		Category: "task", // MUST BE "task"
		Enabled:  tfc.Bool(true),
	})
	Expect(err).Should(Succeed())
	Expect(rt).ShouldNot(BeNil())
	return rt.ID
}
