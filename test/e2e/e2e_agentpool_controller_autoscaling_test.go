// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"errors"
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/hashicorp/go-tfe/v2/api/models"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/controller"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

var _ = Describe("Agent Pool controller", Ordered, func() {
	var (
		instance       *appv1alpha2.AgentPool
		namespacedName types.NamespacedName
		agentPool      string
		workspace      = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		agentPool = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		// Create a new module object for each test
		instance = &appv1alpha2.AgentPool{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "app.terraform.io/v1alpha2",
				Kind:       "AgentPool",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              namespacedName.Name,
				Namespace:         namespacedName.Namespace,
				DeletionTimestamp: nil,
				Finalizers:        []string{},
			},
			Spec: appv1alpha2.AgentPoolSpec{
				Name:         agentPool,
				Organization: organization,
				Token: appv1alpha2.Token{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretNamespacedName.Name,
						},
						Key: secretKey,
					},
				},
				AgentTokens: []*appv1alpha2.AgentAPIToken{
					{Name: "token"},
				},
				AgentDeployment: &appv1alpha2.AgentDeployment{
					Replicas: new(int32(0)),
				},
				AgentDeploymentAutoscaling: &appv1alpha2.AgentDeploymentAutoscaling{
					MinReplicas:           new(int32(0)),
					MaxReplicas:           new(int32(1)),
					CooldownPeriodSeconds: new(int32(5)),
				},
			},
			Status: appv1alpha2.AgentPoolStatus{},
		}
	})

	AfterEach(func() {
		Expect(tfClient.API.Organizations().ByOrganization_name(organization).Workspaces().ByWorkspace_name(workspace).Delete(ctx, nil)).To(Succeed())
		cleanUpAgentPoolDeployment(instance)
		cleanUpAgentPool(instance, namespacedName)
	})

	Context("Autoscaling", func() {
		It("fix: can update the status property on the first run", func() {
			// Create a new HCP Terraform Workspace
			wsAttrs := models.NewWorkspaces_attributes()
			wsAttrs.SetName(&workspace)
			wsAttrs.SetAutoApply(new(true))
			wsAttrs.SetExecutionMode(new(models.REMOTE_WORKSPACES_ATTRIBUTES_EXECUTIONMODE))
			wsID := pointer.ValueOf(createWorkspace(wsAttrs).GetId())
			// Create a new Run and execute it
			_ = createAndUploadConfigurationVersion(wsID, "hoi")
			Eventually(func() bool {
				ws, err := tfClient.API.Workspaces().ByWorkspace_id(wsID).Get(ctx, nil)
				Expect(err).Should(Succeed())
				rels := ws.GetData().GetRelationships()
				if rels == nil || rels.GetCurrentRun() == nil || rels.GetCurrentRun().GetData() == nil {
					return false
				}
				run, err := tfClient.API.Runs().ById(pointer.ValueOf(rels.GetCurrentRun().GetData().GetId())).Get(ctx, nil)
				Expect(err).Should(Succeed())
				return run.GetData().GetAttributes().GetStatus().String() == string(models.APPLIED_RUNS_ATTRIBUTES_STATUS.String())
			}).Should(BeTrue())
			// Create a new Agent Pool
			instance.Spec.DeletionPolicy = appv1alpha2.AgentPoolDeletionPolicyDestroy
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.AgentPoolID != ""
			}).Should(BeTrue())
			// Attrach the Workspace to the Agent pool
			apData := models.NewAgentPoolsHasOne_data()
			apData.SetId(&instance.Status.AgentPoolID)
			apData.SetTypeEscaped(new(models.AGENTPOOLS_AGENTPOOLSIDENTIFIER_TYPE))
			apRel := models.NewAgentPoolsHasOne()
			apRel.SetData(apData)
			wsRels := models.NewWorkspaces_relationships()
			wsRels.SetAgentPool(apRel)
			wsUpdAttrs := models.NewWorkspaces_attributes()
			wsUpdAttrs.SetExecutionMode(new(models.AGENT_WORKSPACES_ATTRIBUTES_EXECUTIONMODE))
			wsUpd := models.NewWorkspaces()
			wsUpd.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
			wsUpd.SetId(&wsID)
			wsUpd.SetAttributes(wsUpdAttrs)
			wsUpd.SetRelationships(wsRels)
			wsUpdBody := models.NewWorkspacesEnvelope()
			wsUpdBody.SetData(wsUpd)
			_, err := tfClient.API.Workspaces().ByWorkspace_id(wsID).Patch(ctx, wsUpdBody, nil)
			Expect(err).Should(Succeed())
			// Trigger a new run
			runAttrs := models.NewRuns_attributes()
			runAttrs.SetPlanOnly(new(false))
			runWsData := models.NewWorkspacesHasOne_data()
			runWsData.SetId(&wsID)
			runWsData.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIER_TYPE))
			runWs := models.NewWorkspacesHasOne()
			runWs.SetData(runWsData)
			runRels := models.NewRuns_relationships()
			runRels.SetWorkspace(runWs)
			runData := models.NewRuns()
			runData.SetTypeEscaped(new(models.RUNS_RUNS_TYPE))
			runData.SetAttributes(runAttrs)
			runData.SetRelationships(runRels)
			runBody := models.NewRunsEnvelope()
			runBody.SetData(runData)
			run, err := tfClient.API.Runs().Post(ctx, runBody, nil)
			Expect(err).Should(Succeed())
			Expect(run.GetData().GetId()).ShouldNot(BeNil())
			// Ensure it scales up
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.AgentDeploymentAutoscalingStatus == nil {
					return false
				}
				return *instance.Status.AgentDeploymentAutoscalingStatus.DesiredReplicas == 1
			}).Should(BeTrue())
		})
	})
	It("can scale up for a speculative plan run", func() {
		// Create a new HCP Terraform Workspace
		wsAttrs := models.NewWorkspaces_attributes()
		wsAttrs.SetName(&workspace)
		wsAttrs.SetAutoApply(new(true))
		wsAttrs.SetExecutionMode(new(models.REMOTE_WORKSPACES_ATTRIBUTES_EXECUTIONMODE))
		wsID := pointer.ValueOf(createWorkspace(wsAttrs).GetId())
		// New Run
		_ = createAndUploadConfigurationVersion(wsID, "hoi")
		Eventually(func() bool {
			ws, err := tfClient.API.Workspaces().ByWorkspace_id(wsID).Get(ctx, nil)
			Expect(err).Should(Succeed())
			rels := ws.GetData().GetRelationships()
			if rels == nil || rels.GetCurrentRun() == nil || rels.GetCurrentRun().GetData() == nil {
				return false
			}
			run, err := tfClient.API.Runs().ById(pointer.ValueOf(rels.GetCurrentRun().GetData().GetId())).Get(ctx, nil)
			Expect(err).Should(Succeed())
			return run.GetData().GetAttributes().GetStatus().String() == string(models.APPLIED_RUNS_ATTRIBUTES_STATUS.String())
		}).Should(BeTrue())
		// New Agent Pool
		instance.Spec.DeletionPolicy = appv1alpha2.AgentPoolDeletionPolicyDestroy
		Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
		Eventually(func() bool {
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			return instance.Status.AgentPoolID != ""
		}).Should(BeTrue())
		// Update Workspace
		apData := models.NewAgentPoolsHasOne_data()
		apData.SetId(&instance.Status.AgentPoolID)
		apData.SetTypeEscaped(new(models.AGENTPOOLS_AGENTPOOLSIDENTIFIER_TYPE))
		apRel := models.NewAgentPoolsHasOne()
		apRel.SetData(apData)
		wsRels := models.NewWorkspaces_relationships()
		wsRels.SetAgentPool(apRel)
		wsUpdAttrs := models.NewWorkspaces_attributes()
		wsUpdAttrs.SetExecutionMode(new(models.AGENT_WORKSPACES_ATTRIBUTES_EXECUTIONMODE))
		wsUpd := models.NewWorkspaces()
		wsUpd.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
		wsUpd.SetId(&wsID)
		wsUpd.SetAttributes(wsUpdAttrs)
		wsUpd.SetRelationships(wsRels)
		wsUpdBody := models.NewWorkspacesEnvelope()
		wsUpdBody.SetData(wsUpd)
		_, err := tfClient.API.Workspaces().ByWorkspace_id(wsID).Patch(ctx, wsUpdBody, nil)
		Expect(err).Should(Succeed())
		// New Speculative Plan
		runAttrs := models.NewRuns_attributes()
		runAttrs.SetPlanOnly(new(true))
		runWsData := models.NewWorkspacesHasOne_data()
		runWsData.SetId(&wsID)
		runWsData.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIER_TYPE))
		runWs := models.NewWorkspacesHasOne()
		runWs.SetData(runWsData)
		runRels := models.NewRuns_relationships()
		runRels.SetWorkspace(runWs)
		runData := models.NewRuns()
		runData.SetTypeEscaped(new(models.RUNS_RUNS_TYPE))
		runData.SetAttributes(runAttrs)
		runData.SetRelationships(runRels)
		runBody := models.NewRunsEnvelope()
		runBody.SetData(runData)
		run, err := tfClient.API.Runs().Post(ctx, runBody, nil)
		Expect(err).Should(Succeed())
		Expect(run.GetData().GetId()).ShouldNot(BeNil())
		// Ensure it scales up
		Eventually(func() bool {
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			if instance.Status.AgentDeploymentAutoscalingStatus == nil {
				return false
			}
			return *instance.Status.AgentDeploymentAutoscalingStatus.DesiredReplicas == 1
		}).Should(BeTrue())
	})
})

func cleanUpAgentPool(instance *appv1alpha2.AgentPool, nn types.NamespacedName) {
	Eventually(func() bool {
		err := k8sClient.Delete(ctx, instance)
		return kerrors.IsNotFound(err) || err == nil
	}).Should(BeTrue())

	Eventually(func() bool {
		err := k8sClient.Get(ctx, nn, instance)
		return kerrors.IsNotFound(err)
	}).Should(BeTrue())

	Eventually(func() bool {
		if instance.Status.AgentPoolID == "" {
			return true
		}
		err := tfClient.API.AgentPools().ByAgent_pool_id(instance.Status.AgentPoolID).Delete(ctx, nil)
		return err == nil || errors.Is(err, tfc.ErrNotFound)
	}).Should(BeTrue())
}

func cleanUpAgentPoolDeployment(instance *appv1alpha2.AgentPool) {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controller.AgentPoolDeploymentName(instance),
			Namespace: instance.GetNamespace(),
		},
	}
	Eventually(func() bool {
		err := k8sClient.Delete(ctx, d)
		return kerrors.IsNotFound(err)
	}).Should(BeTrue())
}
