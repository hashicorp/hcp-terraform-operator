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
				Name:             workspace,
				ApplyMethod:      "auto",
				AllowDestroyPlan: false,
				Description:      "Deletion Policy",
				ExecutionMode:    "remote",
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
			if cloudEndpoint != tfcDefaultAddress {
				Skip("Does not run against TFC, skip this test")
			}
			instance.Spec.AllowDestroyPlan = true
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicySoft
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
			if cloudEndpoint != tfcDefaultAddress {
				Skip("Does not run against TFC, skip this test")
			}
			instance.Spec.AllowDestroyPlan = true
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicyDestroy
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

			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

			var destroyRunID string
			Eventually(func() bool {
				ws, err := tfClient.Workspaces.ReadByID(ctx, workspaceID)
				Expect(err).To(Succeed())
				Expect(ws).ToNot(BeNil())
				Expect(ws.CurrentRun).ToNot(BeNil())
				run, err := tfClient.Runs.Read(ctx, ws.CurrentRun.ID)
				Expect(err).To(Succeed())
				Expect(run).ToNot(BeNil())
				destroyRunID = run.ID

				return run.IsDestroy
			}).Should(BeTrue())

			Eventually(func() bool {
				run, err := tfClient.Runs.Read(ctx, destroyRunID)
				if err == tfc.ErrResourceNotFound || run.Status == tfc.RunApplied {
					return true
				}

				return false
			}).Should(BeTrue())

			Eventually(func() bool {
				_, err := tfClient.Workspaces.ReadByID(ctx, workspaceID)
				return err == tfc.ErrResourceNotFound
			}).Should(BeTrue())
		})
		It("can destroy delete a workspace when the destroy was retried manually after failing", func() {
			if cloudEndpoint != tfcDefaultAddress {
				Skip("Does not run against TFC, skip this test")
			}
			instance.Spec.AllowDestroyPlan = true
			instance.Spec.DeletionPolicy = appv1alpha2.DeletionPolicyDestroy
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

			var destroyRunID string
			Eventually(func() bool {
				ws, err := tfClient.Workspaces.ReadByID(ctx, workspaceID)
				Expect(err).To(Succeed())
				Expect(ws).ToNot(BeNil())
				Expect(ws.CurrentRun).ToNot(BeNil())
				run, err := tfClient.Runs.Read(ctx, ws.CurrentRun.ID)
				Expect(err).To(Succeed())
				Expect(run).ToNot(BeNil())
				destroyRunID = run.ID

				return run.IsDestroy
			}).Should(BeTrue())

			Eventually(func() bool {
				run, _ := tfClient.Runs.Read(ctx, destroyRunID)
				if run.Status == tfc.RunErrored {
					return true
				}

				return false
			}).Should(BeTrue())

			// put back a working configuration
			cv = createAndUploadConfigurationVersion(instance.Status.WorkspaceID, "hoi", true)

			// start a new destroy run manually
			run, err := tfClient.Runs.Create(ctx, tfc.RunCreateOptions{
				IsDestroy: tfc.Bool(true),
				Message:   tfc.String(runMessage),
				Workspace: &tfc.Workspace{
					ID: workspaceID,
				},
			})
			Expect(err).To(Succeed())
			Expect(run).ToNot(BeNil())

			var newDestroyRunID string
			Eventually(func() bool {
				ws, err := tfClient.Workspaces.ReadByID(ctx, workspaceID)
				Expect(err).To(Succeed())
				Expect(ws).ToNot(BeNil())
				Expect(ws.CurrentRun).ToNot(BeNil())
				run, err := tfClient.Runs.Read(ctx, ws.CurrentRun.ID)
				Expect(err).To(Succeed())
				Expect(run).ToNot(BeNil())
				newDestroyRunID = run.ID

				return run.IsDestroy && newDestroyRunID != destroyRunID
			}).Should(BeTrue())

			Eventually(func() bool {
				_, err := tfClient.Workspaces.ReadByID(ctx, workspaceID)
				return err == tfc.ErrResourceNotFound
			}).Should(BeTrue())
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
