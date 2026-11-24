// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

var _ = Describe("Agent Pool controller (existing pool reference)", Ordered, func() {
	var (
		instance         *appv1alpha2.AgentPool
		namespacedName   types.NamespacedName
		existingPool     *tfc.AgentPool
		existingPoolName string
	)

	BeforeAll(func() {
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		// Pre-create a pool directly in HCP Terraform, OUTSIDE the operator, to represent a pool
		// that already exists -- for example one the operator managed before its CR's status was
		// lost in a cluster rebuild.
		existingPoolName = fmt.Sprintf("kubernetes-operator-existing-%v", randomNumber())
		ap, err := tfClient.AgentPools.Create(ctx, organization, tfc.AgentPoolCreateOptions{
			Name: tfc.String(existingPoolName),
		})
		Expect(err).Should(Succeed())
		Expect(ap).ShouldNot(BeNil())
		existingPool = ap

		namespacedName = newNamespacedName()
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
				Name:                existingPoolName, // matches the pre-existing pool's name
				Organization:        organization,
				ExistingAgentPoolID: existingPool.ID, // the field under test
				Token: appv1alpha2.Token{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretNamespacedName.Name,
						},
						Key: secretKey,
					},
				},
			},
			Status: appv1alpha2.AgentPoolStatus{},
		}
	})

	AfterEach(func() {
		// cleanUpAgentPool deletes the CR and the pool named by status.agentPoolID. After a
		// successful adoption that is exactly existingPool.ID, so the pre-created pool is removed.
		cleanUpAgentPool(instance, namespacedName)
		// Best-effort: if adoption never populated status (e.g. the test failed early), the
		// pre-created pool would otherwise leak, so remove it directly.
		if existingPool != nil {
			_ = tfClient.AgentPools.Delete(ctx, existingPool.ID)
		}
	})

	Context("Existing pool reference", func() {
		It("adopts the referenced pool instead of creating a new one", func() {
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// The CR must adopt the pre-existing pool: status.agentPoolID equals the existing ID,
			// never a freshly created pool's ID.
			Eventually(func() string {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.AgentPoolID
			}).Should(Equal(existingPool.ID))

			// And no second pool with the same name was created -- exactly one pool bears this
			// name in the organization. This is the regression guard: without the fix, an empty
			// status would drive a blind create and leave two pools with this name.
			Eventually(func() int {
				l, err := tfClient.AgentPools.List(ctx, organization, &tfc.AgentPoolListOptions{})
				Expect(err).Should(Succeed())
				Expect(l).ShouldNot(BeNil())
				count := 0
				for _, a := range l.Items {
					if a.Name == existingPoolName {
						count++
					}
				}
				return count
			}).Should(Equal(1))
		})
	})
})
