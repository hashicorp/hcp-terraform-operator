// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

var _ = Describe("Agent Token controller", Ordered, func() {
	var (
		instance       *appv1alpha2.AgentToken
		namespacedName types.NamespacedName
		agentPoolName  string
		agentPoolID    string
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		// Create an agent pool to associate with the agent token
		agentPoolName = fmt.Sprintf("kubernetes-operator-agent-pool-%v", randomNumber())
		ap, err := tfClient.AgentPools.Create(ctx, organization, tfc.AgentPoolCreateOptions{
			Name: &agentPoolName,
		})
		Expect(err).Should(Succeed())
		Expect(ap).NotTo(BeNil())
		agentPoolID = ap.ID
		for i := range 3 {
			tfClient.AgentTokens.Create(ctx, agentPoolID, tfc.AgentTokenCreateOptions{
				Description: tfc.String(fmt.Sprintf("token-%d", i)),
			})
		}
		// Create a new agent token custom resource for each test
		namespacedName = newNamespacedName()
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
				AgentPool: appv1alpha2.AgentPoolRef{
					Name: agentPoolName,
				},
				AgentTokens: []appv1alpha2.AgentAPIToken{
					{Name: "first"},
					{Name: "second"},
					{Name: "third"},
				},
				SecretName: fmt.Sprintf("%s-tokens", agentPoolName),
			},
			Status: appv1alpha2.AgentTokenStatus{},
		}
	})

	AfterEach(func() {
		Eventually(func() bool {
			err := tfClient.AgentPools.Delete(ctx, agentPoolID)
			return err == nil || err == tfc.ErrResourceNotFound
		}).Should(BeTrue())
	})

	Context("Deletion Policy", func() {
		It("can remove only managed tokens", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.AgentTokenDeletionPolicyDestroy
			instance.Spec.ManagementPolicy = appv1alpha2.AgentTokenManagementPolicyMerge
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())

			tid := make(map[string]struct{})
			for _, t := range instance.Spec.AgentTokens {
				tid[t.ID] = struct{}{}
			}

			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return kerrors.IsNotFound(err)
			}).Should(BeTrue())

			at, err := tfClient.AgentTokens.List(ctx, agentPoolID)
			Expect(err).Should(Succeed())
			Expect(at).NotTo(BeNil())

			for _, t := range at.Items {
				_, exists := tid[t.ID]
				Expect(exists).To(BeFalse(), fmt.Sprintf("token ID %s should have been deleted", t.ID))
			}
		})
	})
})
