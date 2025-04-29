// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
		if cloudEndpoint != tfcDefaultAddress {
			Skip("Does not run against TFC, skip this test")
		}
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
				Name:            workspace,
				ApplyMethod:     "auto",
				ApplyRunTrigger: "auto",
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		deleteWorkspace(instance)
	})

	Context("Retry", func() {
		It("can retry failed runs", func() {
			namespacedName := getNamespacedName(instance)
			instance.Spec.RetryPolicy = &appv1alpha2.RetryPolicy{
				BackoffLimit: -1,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			workspaceID := instance.Status.WorkspaceID

			// start a run that will fail
			cv := createAndUploadErroredConfigurationVersion(instance.Status.WorkspaceID, true)

			var runID string

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
							runID = r.ID
							return r.Status == tfc.RunErrored
						}
					}
					listOpts.PageNumber = runs.NextPage
				}
				return false
			}).Should(BeTrue())

			// Fix the code but no not start a run manually
			createAndUploadConfigurationVersion(instance.Status.WorkspaceID, "hoi", false)

			// a new run should be started automatically
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}
				return runID != instance.Status.Run.ID
			}).Should(BeTrue())

			// the number of failed attemps should be reset to 0
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Retry == nil {
					return false
				}
				return instance.Status.Retry.Failed == 0
			}).Should(BeTrue())

			// Since the code is fixed at some point a run will succeed
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Run == nil {
					return false
				}

				return runID != instance.Status.Run.ID && instance.Status.Run.RunCompleted()
			}).Should(BeTrue())
		})
		It("can retry until the limit of retries is reached", func() {
			namespacedName := getNamespacedName(instance)

			instance.Spec.RetryPolicy = &appv1alpha2.RetryPolicy{
				BackoffLimit: 2,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation

			createWorkspace(instance)
			workspaceID := instance.Status.WorkspaceID

			// start a run that will fail
			createAndUploadErroredConfigurationVersion(instance.Status.WorkspaceID, true)

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				if instance.Status.Retry == nil {
					return false
				}

				return instance.Status.Retry.Failed == 3
			}).Should(BeTrue())

			Eventually(func() bool {
				listOpts := tfc.ListOptions{
					PageNumber: 1,
					PageSize:   maxPageSize,
				}
				runCount := 0
				for listOpts.PageNumber != 0 {
					runs, err := tfClient.Runs.List(ctx, workspaceID, &tfc.RunListOptions{
						ListOptions: listOpts,
					})
					Expect(err).To(Succeed())
					runCount += len(runs.Items)
					listOpts.PageNumber = runs.NextPage
				}
				return runCount == 3
			}).Should(BeTrue())
		})
		It("can retry failed destroy runs when deleting the workspace", func() {
			instance.Spec.RetryPolicy = &appv1alpha2.RetryPolicy{
				BackoffLimit: -1,
			}
			instance.Spec.AllowDestroyPlan = true
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicyDestroy
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			workspaceID := instance.Status.WorkspaceID

			cv := createAndUploadConfigurationVersion(instance.Status.WorkspaceID, "hoi", true)
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

			// create an errored ConfigurationVersion for the delete to fail
			cv = createAndUploadErroredConfigurationVersion(instance.Status.WorkspaceID, false)

			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

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
							return r.Status == tfc.RunErrored
						}
					}
					listOpts.PageNumber = runs.NextPage
				}
				return false
			}).Should(BeTrue())

			// Fix the code but no not start a run manually
			createAndUploadConfigurationVersion(instance.Status.WorkspaceID, "hoi", false)

			// The retry should eventually delete the workspace
			Eventually(func() bool {
				_, err := tfClient.Workspaces.ReadByID(ctx, workspaceID)
				return err == tfc.ErrResourceNotFound
			}).Should(BeTrue())
		})
	})
})
