// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
	"github.com/hashicorp/terraform-cloud-operator/internal/pointer"
)

var _ = Describe("Agent Pool controller", Ordered, func() {
	var (
		instance       *appv1alpha2.AgentPool
		namespacedName = newNamespacedName()
		agentPool      = fmt.Sprintf("kubernetes-operator-agent-pool-%v", randomNumber())
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
					{Name: "first"},
					{Name: "second"},
				},
			},
			Status: appv1alpha2.AgentPoolStatus{},
		}
	})

	AfterEach(func() {
		// DELETE AGENT POOL DEPLOYMENT
		did := types.NamespacedName{
			Name:      agentPoolDeploymentName(instance),
			Namespace: instance.GetNamespace(),
		}
		d := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      did.Name,
				Namespace: did.Namespace,
			},
		}
		Eventually(func() bool {
			err := k8sClient.Delete(ctx, d)
			return errors.IsNotFound(err) || err == nil
		}).Should(BeTrue())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, did, d)
			return errors.IsNotFound(err)
		}).Should(BeTrue())

		// DELETE AGENT POOL
		Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

		Eventually(func() bool {
			err := k8sClient.Get(ctx, namespacedName, instance)
			return errors.IsNotFound(err)
		}).Should(BeTrue())

		Eventually(func() bool {
			err := tfClient.AgentPools.Delete(ctx, instance.Status.AgentPoolID)
			return err == tfc.ErrResourceNotFound || err == nil
		}).Should(BeTrue())
	})

	Context("Agent Pool controller", func() {
		It("can create a new agent pool", func() {
			// CREATE A NEW AGENT POOL
			createTestAgentPool(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)
		})

		It("can update agent pool with a new token", func() {
			// CREATE A NEW AGENT POOL
			createTestAgentPool(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)

			// ADD ONE MORE AGENT TOKEN
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.AgentTokens = append(instance.Spec.AgentTokens, &appv1alpha2.AgentToken{Name: "third"})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())

			agentPoolTestGenerationsMatch(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)
		})

		It("can remove manually added tokens", func() {
			// CREATE A NEW AGENT POOL
			createTestAgentPool(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)

			// ADD A NEW TOKEN MANUALLY
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			tfClient.AgentTokens.Create(ctx, instance.Status.AgentPoolID, tfc.AgentTokenCreateOptions{
				Description: tfc.String("DeleteMe"),
			})

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				l, err := tfClient.AgentTokens.List(ctx, instance.Status.AgentPoolID)
				Expect(err).Should(Succeed())
				Expect(l).ShouldNot(BeNil())
				at := make(map[string]struct{})
				for _, t := range l.Items {
					at[t.ID] = struct{}{}
				}
				for _, st := range instance.Status.AgentTokens {
					if _, ok := at[st.ID]; !ok {
						return false
					}
				}
				return true
			}).Should(BeTrue())
		})

		It("can recreate agent token", func() {
			// CREATE A NEW AGENT POOL
			createTestAgentPool(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)

			// DELETE ONE AGENT TOKEN
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.AgentTokens = instance.Spec.AgentTokens[:len(instance.Spec.AgentTokens)-1]
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())

			agentPoolTestGenerationsMatch(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)
		})

		It("can recreate agent pool", func() {
			// CREATE A NEW AGENT POOL
			createTestAgentPool(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)

			// DELETE AGENT POOL FROM THE TFC AND WAIT FOR RECREATION
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(tfClient.AgentPools.Delete(ctx, instance.Status.AgentPoolID)).Should(Succeed())
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				l, err := tfClient.AgentPools.List(ctx, instance.Spec.Organization, &tfc.AgentPoolListOptions{})
				Expect(err).Should(Succeed())
				Expect(l).ShouldNot(BeNil())
				for _, a := range l.Items {
					if a.Name == instance.Spec.Name && a.ID == instance.Status.AgentPoolID {
						return true
					}
				}
				return false
			}).Should(BeTrue())

			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)
		})

		It("can delete agent tokens", func() {
			// CREATE A NEW AGENT POOL
			createTestAgentPool(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)

			// DELETE AGENT TOKENS AND WAIT FOR RECREATION
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			for _, t := range instance.Status.AgentTokens {
				err := tfClient.AgentTokens.Delete(ctx, t.ID)
				if err != nil {
					if err == tfc.ErrResourceNotFound {
						continue
					}
				}
			}
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				t, err := tfClient.AgentTokens.List(ctx, instance.Status.AgentPoolID)
				Expect(err).Should(Succeed())
				Expect(t).ShouldNot(BeNil())
				return len(t.Items) == len(instance.Spec.AgentTokens)
			}).Should(BeTrue())

			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)
		})

		It("can create agent deployments with all defaults", func() {
			// CREATE A NEW AGENT POOL
			createTestAgentPool(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).To(BeNil())

			// ADD EMPTY AgentDeployment BLOCK
			instance.Spec.AgentDeployment = &appv1alpha2.AgentDeployment{}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).ToNot(BeNil())
			Expect(instance.Spec.AgentDeployment.Replicas).To(BeNil())
			Expect(instance.Spec.AgentDeployment.Spec).To(BeNil())

			// VALIDATE AGENT DEPLOYMENT ATTRIBUTES
			validateAgentPoolDeployment(ctx, instance)
		})

		It("can create agent deployments with agent pool replica count", func() {
			// CREATE A NEW AGENT POOL
			createTestAgentPool(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).To(BeNil())

			// ADD EMPTY AgentDeployment BLOCK
			instance.Spec.AgentDeployment = &appv1alpha2.AgentDeployment{
				Replicas: pointer.PointerOf(int32(3)),
			}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).ToNot(BeNil())
			Expect(instance.Spec.AgentDeployment.Spec).To(BeNil())
			Expect(*instance.Spec.AgentDeployment.Replicas).To(BeNumerically("==", 3))

			// VALIDATE AGENT DEPLOYMENT ATTRIBUTES
			validateAgentPoolDeployment(ctx, instance)
		})

		It("can create agent deployments with agent pool custom container", func() {
			// CREATE A NEW AGENT POOL
			createTestAgentPool(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).To(BeNil())

			// ADD EMPTY AgentDeployment BLOCK
			instance.Spec.AgentDeployment = &appv1alpha2.AgentDeployment{
				Spec: &corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-agent",
							Image: "my-org/my-custom-agent",
						},
					},
				},
			}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).ToNot(BeNil())
			Expect(instance.Spec.AgentDeployment.Spec).ToNot(BeNil())
			Expect(instance.Spec.AgentDeployment.Replicas).To(BeNil())

			// VALIDATE AGENT DEPLOYMENT ATTRIBUTES
			validateAgentPoolDeployment(ctx, instance)
		})

		It("can delete agent deployments", func() {
			// CREATE A NEW AGENT POOL
			createTestAgentPool(instance)
			// VALIDATE SPEC AGAINST STATUS
			validateAgentPoolTestStatus(ctx, instance)
			// VALIDATE AGENT TOKENS
			validateAgentPoolTestTokens(ctx, instance)

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).To(BeNil())

			// ADD EMPTY AgentDeployment BLOCK
			instance.Spec.AgentDeployment = &appv1alpha2.AgentDeployment{}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).ToNot(BeNil())
			Expect(instance.Spec.AgentDeployment.Replicas).To(BeNil())
			Expect(instance.Spec.AgentDeployment.Spec).To(BeNil())

			// VALIDATE AGENT DEPLOYMENT ATTRIBUTES
			validateAgentPoolDeployment(ctx, instance)

			// SET AgentDeployment TO NIL
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.AgentDeployment = nil
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			validateAgentPoolDeploymentDeleted(ctx, instance)
		})

		It("can autoscale agent deployments", func() {
			createTestAgentPool(instance)

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).To(BeNil())

			instance.Spec.AgentDeployment = &appv1alpha2.AgentDeployment{}
			instance.Spec.AgentDeploymentAutoscaling = &appv1alpha2.AgentDeploymentAutoscaling{
				MinReplicas:           pointer.PointerOf(int32(3)),
				MaxReplicas:           pointer.PointerOf(int32(5)),
				CooldownPeriodSeconds: pointer.PointerOf(int32(60)),
			}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).ToNot(BeNil())
			Expect(instance.Spec.AgentDeployment.Replicas).To(BeNil())
			Expect(instance.Spec.AgentDeployment.Spec).To(BeNil())
			Expect(instance.Spec.AgentDeploymentAutoscaling).ToNot(BeNil())
			Expect(instance.Spec.AgentDeploymentAutoscaling.TargetWorkspaces).To(BeNil())
			Expect(instance.Spec.AgentDeploymentAutoscaling.MinReplicas).To(Equal(pointer.PointerOf(int32(3))))
			Expect(instance.Spec.AgentDeploymentAutoscaling.MaxReplicas).To(Equal(pointer.PointerOf(int32(5))))
			Expect(instance.Spec.AgentDeploymentAutoscaling.CooldownPeriodSeconds).To(Equal(pointer.PointerOf(int32(60))))
		})

		It("can autoscale agent deployments by targeting specific workspaces", func() {
			createTestAgentPool(instance)

			workspaceInstance := testWorkspace("test-workspace", "default", instance.Spec.Name)
			createWorkspace(workspaceInstance)

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).To(BeNil())

			instance.Spec.AgentDeployment = &appv1alpha2.AgentDeployment{}
			instance.Spec.AgentDeploymentAutoscaling = &appv1alpha2.AgentDeploymentAutoscaling{
				TargetWorkspaces: &[]appv1alpha2.TargetWorkspace{
					{Name: "test-workspace"},
					{WildcardName: "test-*"},
					{ID: workspaceInstance.Status.WorkspaceID},
				},
				MinReplicas:           pointer.PointerOf(int32(3)),
				MaxReplicas:           pointer.PointerOf(int32(5)),
				CooldownPeriodSeconds: pointer.PointerOf(int32(60)),
			}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			Expect(instance.Spec.AgentDeployment).ToNot(BeNil())
			Expect(instance.Spec.AgentDeployment.Replicas).To(BeNil())
			Expect(instance.Spec.AgentDeployment.Spec).To(BeNil())
			Expect(instance.Spec.AgentDeploymentAutoscaling).ToNot(BeNil())
			Expect(instance.Spec.AgentDeploymentAutoscaling.TargetWorkspaces).To(Equal(&[]appv1alpha2.TargetWorkspace{
				{Name: "test-workspace"},
				{WildcardName: "test-*"},
				{ID: workspaceInstance.Status.WorkspaceID},
			}))
			Expect(instance.Spec.AgentDeploymentAutoscaling.MinReplicas).To(Equal(pointer.PointerOf(int32(3))))
			Expect(instance.Spec.AgentDeploymentAutoscaling.MaxReplicas).To(Equal(pointer.PointerOf(int32(5))))
			Expect(instance.Spec.AgentDeploymentAutoscaling.CooldownPeriodSeconds).To(Equal(pointer.PointerOf(int32(60))))

			deleteWorkspace(workspaceInstance)
		})
	})
})

func agentPoolTestGenerationsMatch(instance *appv1alpha2.AgentPool) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		return instance.Status.ObservedGeneration == instance.Generation
	}).Should(BeTrue())
}

func createTestAgentPool(instance *appv1alpha2.AgentPool) {
	namespacedName := getNamespacedName(instance)

	Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		return instance.Status.ObservedGeneration == instance.Generation
	}).Should(BeTrue())

	Expect(instance.Status.AgentPoolID).Should(HavePrefix("apool-"))
}

func validateAgentPoolTestStatus(ctx context.Context, instance *appv1alpha2.AgentPool) {
	// VALIDATE SPEC AGAINST STATUS
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		spt := make([]string, len(instance.Spec.AgentTokens))
		for i, t := range instance.Spec.AgentTokens {
			spt[i] = t.Name
		}

		st := make([]string, len(instance.Status.AgentTokens))
		for i, t := range instance.Status.AgentTokens {
			st[i] = t.Name
		}

		return compareAgentTokens(spt, st)
	}).Should(BeTrue())
}

func validateAgentPoolTestTokens(ctx context.Context, instance *appv1alpha2.AgentPool) {
	// VALIDATE TOKENS
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		at, err := tfClient.AgentTokens.List(ctx, instance.Status.AgentPoolID)
		Expect(err).Should(Succeed())
		Expect(at).ShouldNot(BeNil())
		ct := make([]string, len(at.Items))
		for i, t := range at.Items {
			ct[i] = t.ID
		}

		kt := make([]string, len(instance.Status.AgentTokens))
		for i, t := range instance.Status.AgentTokens {
			kt[i] = t.ID
		}

		return compareAgentTokens(ct, kt)
	}).Should(BeTrue())

}

func validateAgentPoolDeployment(ctx context.Context, instance *appv1alpha2.AgentPool) {
	Expect(instance).ToNot(BeNil())

	agentDeployment := &appsv1.Deployment{}
	did := types.NamespacedName{
		Name:      agentPoolDeploymentName(instance),
		Namespace: instance.GetNamespace(),
	}
	Eventually(func() error {
		return k8sClient.Get(ctx, did, agentDeployment)
	}).Should(Succeed())

	Expect(agentDeployment.Spec.Replicas).ToNot(BeNil())
	Expect(agentDeployment.Spec.Strategy.Type).To(Equal(appsv1.RollingUpdateDeploymentStrategyType))
	Expect(agentDeployment.Spec.Strategy.RollingUpdate).ToNot(BeNil())
	Expect(agentDeployment.Spec.Strategy.RollingUpdate.MaxSurge.IntVal).To(BeNumerically("==", 0))
	if instance.Spec.AgentDeployment == nil {
		Expect(*agentDeployment.Spec.Replicas).To(BeNumerically("==", 1))
		Expect(agentDeployment.Spec.Template.Spec.Containers).To(HaveLen(1))
		Expect(agentDeployment.Spec.Template.Spec.Containers[0].Name).To(Equal(defaultAgentContainerName))
		Expect(agentDeployment.Spec.Template.Spec.Containers[0].Image).To(Equal(defaultAgentImage))
		return
	}
	if instance.Spec.AgentDeployment.Replicas == nil {
		Expect(*agentDeployment.Spec.Replicas).To(BeNumerically("==", 1))
	} else {
		Expect(*agentDeployment.Spec.Replicas).To(BeNumerically("==", *instance.Spec.AgentDeployment.Replicas))
	}
	if instance.Spec.AgentDeployment.Spec == nil {
		Expect(agentDeployment.Spec.Template.Spec.Containers).To(HaveLen(1))
		Expect(agentDeployment.Spec.Template.Spec.Containers[0].Name).To(Equal(defaultAgentContainerName))
		Expect(agentDeployment.Spec.Template.Spec.Containers[0].Image).To(Equal(defaultAgentImage))
		return
	}
	Expect(agentDeployment.Spec.Template.Spec.Containers).To(HaveLen(len(instance.Spec.AgentDeployment.Spec.Containers)))
	for ci, c := range instance.Spec.AgentDeployment.Spec.Containers {
		Expect(agentDeployment.Spec.Template.Spec.Containers[ci].Name).To(Equal(c.Name))
		Expect(agentDeployment.Spec.Template.Spec.Containers[ci].Image).To(Equal(c.Image))
	}
}

func validateAgentPoolDeploymentDeleted(ctx context.Context, instance *appv1alpha2.AgentPool) {
	namespacedName := getNamespacedName(instance)

	Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
	Expect(instance.Spec.AgentDeployment).To(BeNil())

	did := types.NamespacedName{
		Name:      agentPoolDeploymentName(instance),
		Namespace: instance.GetNamespace(),
	}
	Eventually(func() bool {
		err := k8sClient.Get(ctx, did, &appsv1.Deployment{})
		return errors.IsNotFound(err)
	}).Should(BeTrue())
}

func compareAgentTokens(aTokens, bTokens []string) bool {
	if len(aTokens) != len(bTokens) {
		return false
	}

	for _, at := range aTokens {
		found := false
		for _, bt := range bTokens {
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

func testWorkspace(name, namespace, agentPoolName string) *appv1alpha2.Workspace {
	workspaceName := types.NamespacedName{
		Name:      "test-workspace-autoscaling",
		Namespace: "default",
	}
	instance := &appv1alpha2.Workspace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "app.terraform.io/v1alpha2",
			Kind:       "Workspace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              workspaceName.Name,
			Namespace:         workspaceName.Namespace,
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
			Name:          fmt.Sprintf("test-workspace-%v", randomNumber()),
			ExecutionMode: "agent",
			AgentPool: &appv1alpha2.WorkspaceAgentPool{
				Name: agentPoolName,
			},
		},
		Status: appv1alpha2.WorkspaceStatus{},
	}
	return instance
}
