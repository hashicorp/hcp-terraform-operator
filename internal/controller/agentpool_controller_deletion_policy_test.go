// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

var _ = Describe("Agent Pool controller", Ordered, func() {
	var (
		instance       *appv1alpha2.AgentPool
		namespacedName types.NamespacedName
		agentPool      string
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		agentPool = fmt.Sprintf("kubernetes-operator-agent-pool-%v", randomNumber())
		// Create a new agent pool custom resource for each test
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
					{Name: "first"},
					{Name: "second"},
				},
			},
			Status: appv1alpha2.AgentPoolStatus{},
		}
	})

	AfterEach(func() {
		cleanUpAgentPoolDeployment(instance)
		cleanUpAgentPool(instance, namespacedName)
	})

	Context("Deletion Policy", func() {
		It("can delete a resource that does not manage an agent pool", func() {
			instance.Spec.Token.SecretKeyRef.Key = dummySecretKey
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
		})
		It("can retain an agent pool", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.AgentPoolDeletionPolicyRetain
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())
			apID := instance.Status.AgentPoolID

			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())

			ap, err := tfClient.AgentPools.Read(ctx, apID)
			Expect(err).Should(Succeed())
			Expect(ap).NotTo(BeNil())
		})
		It("can destroy delete an agent pool", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.AgentPoolDeletionPolicyDestroy
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())
			apID := instance.Status.AgentPoolID

			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())

			ap, err := tfClient.AgentPools.Read(ctx, apID)
			Expect(err).Should(Equal(tfc.ErrResourceNotFound))
			Expect(ap).To(BeNil())
		})
	})
})
