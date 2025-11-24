// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
	"maps"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/slice"
)

var _ = Describe("AgentToken Controller", Ordered, func() {
	var (
		instance       *appv1alpha2.AgentToken
		namespacedName = newNamespacedName()
		poolName       = fmt.Sprintf("kubernetes-operator-agent-pool-%v", randomNumber())
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		// Create a new agent pool object for each test
		instance = &appv1alpha2.AgentToken{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "app.terraform.io/v1alpha2",
				Kind:       "AgentToken",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              namespacedName.Name,
				Namespace:         namespacedName.Namespace,
				DeletionTimestamp: nil,
				Finalizers:        []string{},
			},
			Spec: appv1alpha2.AgentTokenSpec{
				Organization: organization,
				Token: appv1alpha2.Token{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretNamespacedName.Name,
						},
						Key: secretKey,
					},
				},
				DeletionPolicy: appv1alpha2.AgentTokenDeletionPolicyRetain,
				AgentTokens: []appv1alpha2.AgentAPIToken{
					{Name: "first"},
					{Name: "second"},
				},
				ManagementPolicy: appv1alpha2.AgentTokenManagementPolicyMerge,
				SecretName:       namespacedName.Name,
			},
			Status: appv1alpha2.AgentTokenStatus{},
		}
	})

	AfterEach(func() {
		// DELETE AGENT TOKEN RESOURCE
		Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

		Eventually(func() bool {
			err := k8sClient.Get(ctx, namespacedName, instance)
			return kerrors.IsNotFound(err)
		}).Should(BeTrue())

		Eventually(func() bool {
			if instance.Status.AgentPool == nil {
				return true
			}
			err := tfClient.AgentPools.Delete(ctx, instance.Status.AgentPool.ID)
			return err == tfc.ErrResourceNotFound
		}).Should(BeTrue())
	})

	Context("When reconciling a resource", func() {
		It("should successfully manage tokens with merge policy", func() {
			// CREATE AGENT POOL WITH ONE TOKEN
			pool := createAgentPoolWithToken(poolName)
			// CREATE KUBERNETES ITEM
			instance.Spec.AgentPool = appv1alpha2.AgentPoolRef{
				ID: pool.ID,
			}
			instance.Spec.SecretName = string(appv1alpha2.AgentTokenManagementPolicyMerge)
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())
			validateTokensPolicyMerge(ctx, instance)
			validateTokensSecretSync(ctx, instance)
			// ADD ONE MORE TOKEN
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.AgentTokens = append(instance.Spec.AgentTokens, appv1alpha2.AgentAPIToken{
				Name: "third",
			})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			validateTokensPolicyMerge(ctx, instance)
			validateTokensSecretSync(ctx, instance)
			// REMOVE ONE TOKEN
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.AgentTokens = slice.RemoveFromSlice(instance.Spec.AgentTokens, len(instance.Spec.AgentTokens)-1)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			validateTokensPolicyMerge(ctx, instance)
			validateTokensSecretSync(ctx, instance)
			// CAN RESTORE TOKEN
			Expect(tfClient.AgentTokens.Delete(ctx, instance.Status.AgentTokens[0].ID)).Should(Succeed())
			validateTokensPolicyMerge(ctx, instance)
			validateTokensSecretSync(ctx, instance)
		})
		It("should successfully manage tokens with owner policy", func() {
			// CREATE AGENT POOL WITH ONE TOKEN
			pool := createAgentPoolWithToken(poolName)
			// CREATE KUBERNETES ITEM
			instance.Spec.AgentPool = appv1alpha2.AgentPoolRef{
				ID: pool.ID,
			}
			instance.Spec.ManagementPolicy = appv1alpha2.AgentTokenManagementPolicyOwner
			instance.Spec.SecretName = string(appv1alpha2.AgentTokenManagementPolicyOwner)
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())
			validateTokensPolicyOwner(ctx, instance)
			validateTokensSecretSync(ctx, instance)
			// ADD ONE MORE TOKEN
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.AgentTokens = append(instance.Spec.AgentTokens, appv1alpha2.AgentAPIToken{
				Name: "third",
			})
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			validateTokensPolicyOwner(ctx, instance)
			validateTokensSecretSync(ctx, instance)
			// REMOVE ONE TOKEN
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.AgentTokens = slice.RemoveFromSlice(instance.Spec.AgentTokens, len(instance.Spec.AgentTokens)-1)
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			validateTokensPolicyOwner(ctx, instance)
			validateTokensSecretSync(ctx, instance)
			// CAN RESTORE TOKEN
			Expect(tfClient.AgentTokens.Delete(ctx, instance.Status.AgentTokens[0].ID)).Should(Succeed())
			validateTokensPolicyOwner(ctx, instance)
			validateTokensSecretSync(ctx, instance)
		})
	})
})

func validateTokensSecretSync(ctx context.Context, instance *appv1alpha2.AgentToken) {
	snn := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      instance.Spec.SecretName,
	}
	nn := getNamespacedName(instance)
	Eventually(func() bool {
		s := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, snn, s)).Should(Succeed())
		st := make(map[string]struct{})
		for name := range s.Data {
			st[name] = struct{}{}
		}

		Expect(k8sClient.Get(ctx, nn, instance)).Should(Succeed())
		kt := make(map[string]struct{})
		for _, t := range instance.Status.AgentTokens {
			kt[t.Name] = struct{}{}
		}

		return maps.Equal(st, kt)
	}).Should(BeTrue())
}

func validateTokensPolicyOwner(ctx context.Context, instance *appv1alpha2.AgentToken) {
	nn := getNamespacedName(instance)
	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, nn, instance)).Should(Succeed())
		if instance.Generation != instance.Status.ObservedGeneration {
			return false
		}
		at, err := tfClient.AgentTokens.List(ctx, instance.Status.AgentPool.ID)
		Expect(err).Should(Succeed())
		Expect(at).ShouldNot(BeNil())

		ct := make(map[string]struct{}, len(at.Items))
		for _, t := range at.Items {
			ct[t.ID] = struct{}{}
		}

		kt := make(map[string]struct{}, len(instance.Status.AgentTokens))
		for _, t := range instance.Status.AgentTokens {
			kt[t.ID] = struct{}{}
		}

		return maps.Equal(ct, kt)
	}).Should(BeTrue())
}

func validateTokensPolicyMerge(ctx context.Context, instance *appv1alpha2.AgentToken) {
	nn := getNamespacedName(instance)
	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, nn, instance)).Should(Succeed())
		if instance.Generation != instance.Status.ObservedGeneration {
			return false
		}
		at, err := tfClient.AgentTokens.List(ctx, instance.Status.AgentPool.ID)
		Expect(err).Should(Succeed())
		Expect(at).ShouldNot(BeNil())
		if len(at.Items) != len(instance.Spec.AgentTokens)+1 {
			return false
		}

		tokens := make(map[string]struct{}, len(at.Items))
		for _, t := range at.Items {
			tokens[t.ID] = struct{}{}
		}

		for _, t := range instance.Status.AgentTokens {
			if _, ok := tokens[t.ID]; !ok {
				return false
			}
		}
		return true
	}).Should(BeTrue())
}

func createAgentPoolWithToken(name string) *tfc.AgentPool {
	// CREATE AGENT POOL
	pool, err := tfClient.AgentPools.Create(ctx, organization, tfc.AgentPoolCreateOptions{
		Name: tfc.String(name),
	})
	Expect(err).Should(Succeed())
	// CREATE AGENT TOKEN
	_, err = tfClient.AgentTokens.Create(ctx, pool.ID, tfc.AgentTokenCreateOptions{
		Description: tfc.String("token"),
	})
	Expect(err).Should(Succeed())

	return pool
}
