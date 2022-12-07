package controllers

import (
	"context"
	"fmt"
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

var _ = Describe("Agent Pool controller", Ordered, func() {
	var (
		instance  *appv1alpha2.AgentPool
		agentPool = fmt.Sprintf("kubernetes-operator-agent-pool-%v", GinkgoRandomSeed())
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
							Name: namespacedName.Name,
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

		It("can recreate agent token", func() {
			// CREATE A NEW AGENT POOL
			createTestAgentPool(instance)
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
	})
})

func agentPoolTestGenerationsMatch(instance *appv1alpha2.AgentPool) {
	namespacedName = types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		return instance.Status.ObservedGeneration == instance.Generation
	}).Should(BeTrue())
}

func createTestAgentPool(instance *appv1alpha2.AgentPool) {
	namespacedName = types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}

	Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		return instance.Status.ObservedGeneration == instance.Generation
	}).Should(BeTrue())

	Expect(instance.Status.AgentPoolID).Should(HavePrefix("apool-"))
}

func validateAgentPoolTestStatus(ctx context.Context, instance *appv1alpha2.AgentPool) {
	// VALIDATE SPEC AGAINST STATUS
	namespacedName := types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}
	Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
	spt := make([]string, len(instance.Spec.AgentTokens))
	for i, t := range instance.Spec.AgentTokens {
		spt[i] = t.Name
	}
	st := make([]string, len(instance.Status.AgentTokens))
	for i, t := range instance.Status.AgentTokens {
		st[i] = t.Name
	}
	Expect(spt).Should(ConsistOf(st))

}

func validateAgentPoolTestTokens(ctx context.Context, instance *appv1alpha2.AgentPool) {
	// VALIDATE TOKENS
	namespacedName := types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}
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
	Expect(ct).Should(ConsistOf(kt))
}
