// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Agent Pool controller", Ordered, func() {
	var (
		instance       *appv1alpha2.AgentPool
		namespacedName = newNamespacedName()
		agentPool      = fmt.Sprintf("kubernetes-operator-agent-pool-%v", randomNumber())
		workspace      = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
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
				AgentTokens: []*appv1alpha2.AgentToken{
					{Name: "token"},
				},
				AgentDeployment: &appv1alpha2.AgentDeployment{
					Replicas: pointer.PointerOf(int32(0)),
				},
				AgentDeploymentAutoscaling: &appv1alpha2.AgentDeploymentAutoscaling{
					MinReplicas:           pointer.PointerOf(int32(0)),
					MaxReplicas:           pointer.PointerOf(int32(1)),
					CooldownPeriodSeconds: pointer.PointerOf(int32(5)),
				},
			},
			Status: appv1alpha2.AgentPoolStatus{},
		}
	})

	AfterEach(func() {
		Expect(tfClient.Workspaces.Delete(ctx, organization, workspace)).To(Succeed())
		// Delete Agent Pool CR
		Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, namespacedName, instance)
			return errors.IsNotFound(err)
		}).Should(BeTrue())
	})

	Context("Autoscaling", func() {
		Context("Autoscaling", func() {
			It("fix: can update the status property on the first run", func() {
				// Create a new Workspace
				ws, err := tfClient.Workspaces.Create(ctx, organization, tfc.WorkspaceCreateOptions{
					Name:      &workspace,
					AutoApply: tfc.Bool(true),
				})
				Expect(err).Should(Succeed())
				Expect(ws).ShouldNot(BeNil())
				// Create a new Run and execute it
				_ = createAndUploadConfigurationVersion(ws.ID, "hoi", true)
				Eventually(func() bool {
					ws, err = tfClient.Workspaces.ReadByID(ctx, ws.ID)
					Expect(err).Should(Succeed())
					Expect(ws).ShouldNot(BeNil())
					if ws.CurrentRun == nil {
						return false
					}
					run, err := tfClient.Runs.Read(ctx, ws.CurrentRun.ID)
					Expect(err).Should(Succeed())
					Expect(run).ShouldNot(BeNil())
					return run.Status == tfc.RunApplied
				}).Should(BeTrue())
				// Create a new Agent Pool
				instance.Spec.DeletionPolicy = appv1alpha2.AgentPoolDeletionPolicyDestroy
				Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
				Eventually(func() bool {
					Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
					return instance.Status.AgentPoolID != ""
				}).Should(BeTrue())
				// Attrach the Workspace to the Agent pool
				ws, err = tfClient.Workspaces.UpdateByID(ctx, ws.ID, tfc.WorkspaceUpdateOptions{
					ExecutionMode: pointer.PointerOf("agent"),
					AgentPoolID:   &instance.Status.AgentPoolID,
				})
				Expect(err).Should(Succeed())
				Expect(ws).ShouldNot(BeNil())
				// Trigger a new run
				run, err := tfClient.Runs.Create(ctx, tfc.RunCreateOptions{
					PlanOnly: pointer.PointerOf(false),
					Workspace: &tfc.Workspace{
						ID: ws.ID,
					},
				})
				Expect(err).Should(Succeed())
				Expect(run).ShouldNot(BeNil())
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
			// New Workspace
			ws, err := tfClient.Workspaces.Create(ctx, organization, tfc.WorkspaceCreateOptions{
				Name:      &workspace,
				AutoApply: tfc.Bool(true),
			})
			Expect(err).Should(Succeed())
			Expect(ws).ShouldNot(BeNil())
			// New Run
			_ = createAndUploadConfigurationVersion(ws.ID, "hoi", true)
			Eventually(func() bool {
				ws, err = tfClient.Workspaces.ReadByID(ctx, ws.ID)
				Expect(err).Should(Succeed())
				Expect(ws).ShouldNot(BeNil())
				if ws.CurrentRun == nil {
					return false
				}
				run, err := tfClient.Runs.Read(ctx, ws.CurrentRun.ID)
				Expect(err).Should(Succeed())
				Expect(run).ShouldNot(BeNil())
				return run.Status == tfc.RunApplied
			}).Should(BeTrue())
			// New Agent Pool
			instance.Spec.DeletionPolicy = appv1alpha2.AgentPoolDeletionPolicyDestroy
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.AgentPoolID != ""
			}).Should(BeTrue())
			// Update Workspace
			ws, err = tfClient.Workspaces.UpdateByID(ctx, ws.ID, tfc.WorkspaceUpdateOptions{
				ExecutionMode: pointer.PointerOf("agent"),
				AgentPoolID:   &instance.Status.AgentPoolID,
			})
			Expect(err).Should(Succeed())
			Expect(ws).ShouldNot(BeNil())
			// New Speculative Plan
			run, err := tfClient.Runs.Create(ctx, tfc.RunCreateOptions{
				PlanOnly: pointer.PointerOf(true),
				Workspace: &tfc.Workspace{
					ID: ws.ID,
				},
			})
			Expect(err).Should(Succeed())
			Expect(run).ShouldNot(BeNil())
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
})
