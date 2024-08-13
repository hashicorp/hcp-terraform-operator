// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

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

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName types.NamespacedName
		workspace      string
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
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
				Name:        workspace,
				ApplyMethod: "auto",
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		deleteWorkspace(instance)
	})

	Context("Deletion Policy", func() {
		It("can delete a resource that does not manage a workspace", func() {
			instance.Spec.Token.SecretKeyRef.Key = dummySecretKey
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
		})
		It("can retain a workspace", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicyRetain
			createWorkspace(instance)
			workspaceID := instance.Status.WorkspaceID
			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())
			workspace, err := tfClient.Workspaces.ReadByID(ctx, workspaceID)
			Expect(err).Should(Succeed())
			Expect(workspace).NotTo(BeNil())
		})
		It("can soft delete a workspace", func() {
			instance.Spec.AllowDestroyPlan = true
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicySoft
			createWorkspace(instance)
			workspaceID := instance.Status.WorkspaceID

			cv := createAndUploadConfigurationVersion(instance, "hoi")
			Eventually(func() bool {
				listOpts := tfc.ListOptions{
					PageNumber: 1,
					PageSize:   maxPageSize,
				}
				for listOpts.PageNumber != 0 {
					runs, err := tfClient.Runs.List(ctx, workspaceID, &tfc.RunListOptions{
						ListOptions: listOpts,
					})
					Expect(err).To(Succeed())
					for _, r := range runs.Items {
						if r.ConfigurationVersion.ID == cv.ID {
							return r.Status == tfc.RunApplied
						}
					}
					listOpts.PageNumber = runs.NextPage
				}
				return false
			}).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

			run, err := tfClient.Runs.Create(ctx, tfc.RunCreateOptions{
				IsDestroy: tfc.Bool(true),
				Workspace: &tfc.Workspace{
					ID: workspaceID,
				},
				AutoApply: tfc.Bool(true),
			})
			Expect(err).To(Succeed())
			Expect(run).ToNot(BeNil())

			Eventually(func() bool {
				_, err := tfClient.Workspaces.ReadByID(ctx, workspaceID)
				return err == tfc.ErrResourceNotFound
			}).Should(BeTrue())
		})
		It("can destroy delete a workspace", func() {
			// Not yet implemented
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicyDestroy
		})
		It("can force delete a workspace", func() {
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicyForce
			createWorkspace(instance)
			workspaceID := instance.Status.WorkspaceID
			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, instance)
				return errors.IsNotFound(err)
			}).Should(BeTrue())
			workspace, err := tfClient.Workspaces.ReadByID(ctx, workspaceID)
			Expect(err).To(MatchError(tfc.ErrResourceNotFound))
			Expect(workspace).To(BeNil())
		})
	})
})
