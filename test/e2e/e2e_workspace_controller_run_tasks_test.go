// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	abstractions "github.com/microsoft/kiota-abstractions-go"
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
		instance       *appv1alpha2.Workspace
		namespacedName types.NamespacedName
		workspace      string

		runTaskName  string
		runTaskName2 string
		runTaskID    string
		runTaskID2   string
	)

	// KNOWN ISSUE
	//
	// Run Task should be created dynamically before run tests and then removed once tests are done.
	// However, due to a bug on the HCP Terraform end, a Run Task cannot be removed immediately once the workspace is removed.
	// The Run Task remains associated with the deleted workspace due to the "cool down" period of ~15 minutes.
	//
	// IPL-3276.

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		runTaskName = fmt.Sprintf("kubernetes-operator-run-task-%v", randomNumber())
		runTaskName2 = fmt.Sprintf("kubernetes-operator-run-task-2-%v", randomNumber())
		runTaskID = createRunTaskForTest(runTaskName)
		runTaskID2 = createRunTaskForTest(runTaskName2)
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
	})

	AfterEach(func() {
		deleteWorkspace(instance)
		Expect(tfClient.API.Tasks().ById(runTaskID).Delete(ctx, nil)).Should(Succeed())
		Expect(tfClient.API.Tasks().ById(runTaskID2).Delete(ctx, nil)).Should(Succeed())
	})

	Context("Run Tasks", func() {
		It("can create run task by ID", func() {
			instance.Spec.RunTasks[0].ID = runTaskID
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isRunTasksReconciled(instance)
		})

		It("can create run task by Name", func() {
			instance.Spec.RunTasks[0].Name = runTaskName
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
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
			createWorkspaceResource(instance)
			isRunTasksReconciled(instance)
		})

		It("can delete run task", func() {
			instance.Spec.RunTasks[0].ID = runTaskID
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
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
			createWorkspaceResource(instance)
			isRunTasksReconciled(instance)

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.RunTasks[0].EnforcementLevel = "mandatory" // was "advisory"

			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isRunTasksReconciled(instance)
		})

		It("can restore deleted run task", func() {
			instance.Spec.RunTasks[0].ID = runTaskID
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isRunTasksReconciled(instance)

			deleteWorkspaceRunTasks(instance)
			isRunTasksReconciled(instance)
		})

		It("can revert manual changes in run task", func() {
			instance.Spec.RunTasks[0].ID = runTaskID
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isRunTasksReconciled(instance)

			// Make a manual change
			runTasksList, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Tasks().Get(ctx, nil)
			Expect(err).Should(Succeed())
			Expect(runTasksList).ShouldNot(BeNil())
			Expect(runTasksList.GetData()).Should(HaveLen(1))
			runTasks := runTasksList.GetData()
			level, err := models.ParseWorkspaceTasks_attributes_enforcementLevel("mandatory") // was "advisory"
			Expect(err).Should(Succeed())
			attrs := models.NewWorkspaceTasks_attributes()
			attrs.SetEnforcementLevel(level.(*models.WorkspaceTasks_attributes_enforcementLevel))
			attrs.SetStage(new("pre_plan")) // was "post_plan"
			data := models.NewWorkspaceTasks()
			data.SetTypeEscaped(new(models.WORKSPACETASKS_WORKSPACETASKS_TYPE))
			data.SetAttributes(attrs)
			body := models.NewWorkspaceTasksEnvelope()
			body.SetData(data)
			_, err = tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Tasks().ById(pointer.ValueOf(runTasks[0].GetId())).Patch(ctx, body, nil)
			Expect(err).Should(Succeed())
			isRunTasksReconciled(instance)
		})
	})
})

func deleteWorkspaceRunTasks(instance *appv1alpha2.Workspace) {
	rtList, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Tasks().Get(ctx, nil)
	Expect(err).Should(Succeed())
	Expect(rtList).ShouldNot(BeNil())

	for _, rt := range rtList.GetData() {
		err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Tasks().ById(pointer.ValueOf(rt.GetId())).Delete(ctx, nil)
		Expect(err).Should(Succeed())
	}

	Eventually(func() bool {
		rtList, err = tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Tasks().Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(rtList).ShouldNot(BeNil())
		return len(rtList.GetData()) == 0
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
			runTasksList, err := tfClient.API.Organizations().ByOrganization_name(instance.Spec.Organization).Tasks().Get(ctx,
				&abstractions.RequestConfiguration[organizations.ItemTasksRequestBuilderGetQueryParameters]{
					QueryParameters: &organizations.ItemTasksRequestBuilderGetQueryParameters{
						Pagesize: new(int32(100)),
					},
				})
			Expect(err).Should(Succeed())
			Expect(runTasksList).ShouldNot(BeNil())
			for _, r := range instance.Spec.RunTasks {
				id := r.ID
				if r.Name != "" {
					for _, rt := range runTasksList.GetData() {
						if rt.GetAttributes() != nil && pointer.ValueOf(rt.GetAttributes().GetName()) == r.Name {
							id = pointer.ValueOf(rt.GetId())
						}
					}
				}
				s[id] = []string{r.EnforcementLevel, r.Stage}
			}
		}

		// Get a slice of all Run Tasks that are assigned to the Workspace
		runTasks, err := tfClient.API.Workspaces().ByWorkspace_id(instance.Status.WorkspaceID).Tasks().Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(runTasks).ShouldNot(BeNil())
		w := make(map[string][]string)
		for _, r := range runTasks.GetData() {
			if r.GetAttributes() == nil || r.GetRelationships() == nil || r.GetRelationships().GetTask() == nil || r.GetRelationships().GetTask().GetData() == nil {
				continue
			}
			level := ""
			if r.GetAttributes().GetEnforcementLevel() != nil {
				level = r.GetAttributes().GetEnforcementLevel().String()
			}
			w[pointer.ValueOf(r.GetRelationships().GetTask().GetData().GetId())] = []string{level, pointer.ValueOf(r.GetAttributes().GetStage())}
		}

		return cmp.Equal(s, w)
	}).Should(BeTrue())
}

func createRunTaskForTest(name string) string {
	attrs := models.NewTasks_attributes()
	attrs.SetName(&name)
	attrs.SetUrl(new(webhookURL))
	attrs.SetCategory(new("task")) // MUST BE "task"
	attrs.SetEnabled(new(true))
	data := models.NewTasks()
	data.SetTypeEscaped(new(models.TASKS_TASKS_TYPE))
	data.SetAttributes(attrs)
	body := models.NewTasksEnvelope()
	body.SetData(data)
	rt, err := tfClient.API.Organizations().ByOrganization_name(organization).Tasks().Post(ctx, body, nil)
	Expect(err).Should(Succeed())
	Expect(rt).ShouldNot(BeNil())
	Expect(rt.GetData()).ShouldNot(BeNil())
	return pointer.ValueOf(rt.GetData().GetId())
}
