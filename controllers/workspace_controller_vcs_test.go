// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance     *appv1alpha2.Workspace
		workspace    = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		oAuthTokenID = os.Getenv("TFC_OAUTH_TOKEN")
		repository   = os.Getenv("TFC_VCS_REPO")
	)

	namespacedName := types.NamespacedName{
		Name:      "this",
		Namespace: "default",
	}

	BeforeAll(func() {
		if oAuthTokenID == "" {
			Skip("Environment variable TFC_OAUTH_TOKEN is either not set or empty")
		}
		if repository == "" {
			Skip("Environment variable TFC_VCS_REPO is either not set or empty")
		}
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
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
				Name: workspace,
				VersionControl: &appv1alpha2.VersionControl{
					OAuthTokenID: oAuthTokenID,
					Repository:   repository,
					Branch:       "operator",
				},
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance)
	})

	Context("Workspace controller", func() {
		It("can attach VCS to the workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
				Expect(ws).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				if ws.VCSRepo == nil {
					return false
				}
				return ws.VCSRepo.OAuthTokenID == instance.Spec.VersionControl.OAuthTokenID &&
					ws.VCSRepo.Identifier == instance.Spec.VersionControl.Repository &&
					ws.VCSRepo.Branch == instance.Spec.VersionControl.Branch
			}).Should(BeTrue())
		})

		It("can update VCS", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			instance.Spec.VersionControl.Branch = "main"
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
				Expect(ws).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				if ws.VCSRepo == nil {
					return false
				}
				return ws.VCSRepo.OAuthTokenID == instance.Spec.VersionControl.OAuthTokenID &&
					ws.VCSRepo.Identifier == instance.Spec.VersionControl.Repository &&
					ws.VCSRepo.Branch == instance.Spec.VersionControl.Branch
			}).Should(BeTrue())
		})

		It("can revert manual changes VCS", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			ws, err := tfClient.Workspaces.UpdateByID(ctx, instance.Status.WorkspaceID, tfc.WorkspaceUpdateOptions{
				VCSRepo: &tfc.VCSRepoOptions{
					Branch: tfc.String("main"),
				},
			})
			Expect(ws).ShouldNot(BeNil())
			Expect(err).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
				Expect(ws).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				if ws.VCSRepo == nil {
					return false
				}
				return ws.VCSRepo.OAuthTokenID == instance.Spec.VersionControl.OAuthTokenID &&
					ws.VCSRepo.Identifier == instance.Spec.VersionControl.Repository &&
					ws.VCSRepo.Branch == instance.Spec.VersionControl.Branch
			}).Should(BeTrue())
		})

		It("can revert manual detach of VCS from the workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			ws, err := tfClient.Workspaces.RemoveVCSConnectionByID(ctx, instance.Status.WorkspaceID)
			Expect(ws).ShouldNot(BeNil())
			Expect(err).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
				Expect(ws).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				if ws.VCSRepo == nil {
					return false
				}
				return ws.VCSRepo.OAuthTokenID == instance.Spec.VersionControl.OAuthTokenID &&
					ws.VCSRepo.Identifier == instance.Spec.VersionControl.Repository &&
					ws.VCSRepo.Branch == instance.Spec.VersionControl.Branch
			}).Should(BeTrue())
		})

		It("can detach VCS from the workspace", func() {
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)

			instance.Spec.VersionControl = nil
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
				Expect(ws).ShouldNot(BeNil())
				Expect(err).Should(Succeed())
				return ws.VCSRepo == nil
			}).Should(BeTrue())
		})
	})
})
