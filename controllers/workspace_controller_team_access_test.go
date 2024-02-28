// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		team           *tfc.Team
		namespacedName = newNamespacedName()
		workspace      = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create new teams
		team = createTeam(fmt.Sprintf("%s-team", workspace))
	})

	AfterAll(func() {
		err := tfClient.Teams.Delete(ctx, team.ID)
		Expect(err).Should(Succeed())
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
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance)
	})

	Context("Workspace controller", func() {
		It("can handle pre-set team access", func() {
			instance.Spec.TeamAccess = []*appv1alpha2.TeamAccess{
				{
					Team: appv1alpha2.Team{
						Name: team.Name,
					},
					Access: "admin",
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isTeamAccessReconciled(instance)

			wsTeamAccess := buildWorkspaceTeamAccessByName(instance.Status.WorkspaceID, appv1alpha2.CustomPermissions{
				Runs:             "read",
				RunTasks:         false,
				Sentinel:         "none",
				StateVersions:    "none",
				Variables:        "none",
				WorkspaceLocking: false,
			})
			Expect(wsTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))
		})

		It("can handle custom team access", func() {
			instance.Spec.TeamAccess = []*appv1alpha2.TeamAccess{
				{
					Team: appv1alpha2.Team{
						Name: team.Name,
					},
					Access: "custom",
					Custom: appv1alpha2.CustomPermissions{
						Runs:             "plan",
						RunTasks:         true,
						Sentinel:         "read",
						StateVersions:    "read",
						Variables:        "read",
						WorkspaceLocking: true,
					},
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isTeamAccessReconciled(instance)

			wsTeamAccess := buildWorkspaceTeamAccessByName(instance.Status.WorkspaceID, appv1alpha2.CustomPermissions{
				Runs:             "plan",
				RunTasks:         true,
				Sentinel:         "read",
				StateVersions:    "read",
				Variables:        "read",
				WorkspaceLocking: true,
			})
			Expect(wsTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))
		})

		It("can handle update from pre-set to custom team access", func() {
			instance.Spec.TeamAccess = []*appv1alpha2.TeamAccess{
				{
					Team: appv1alpha2.Team{
						Name: team.Name,
					},
					Access: "admin",
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isTeamAccessReconciled(instance)

			wsTeamAccess := buildWorkspaceTeamAccessByName(instance.Status.WorkspaceID, appv1alpha2.CustomPermissions{
				Runs:             "read",
				RunTasks:         false,
				Sentinel:         "none",
				StateVersions:    "none",
				Variables:        "none",
				WorkspaceLocking: false,
			})
			Expect(wsTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))

			// UPDATE TEAM ACCESS
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.TeamAccess = []*appv1alpha2.TeamAccess{
				{
					Team: appv1alpha2.Team{
						ID: team.ID,
					},
					Access: "custom",
					Custom: appv1alpha2.CustomPermissions{
						Runs:             "apply",
						RunTasks:         true,
						Sentinel:         "read",
						StateVersions:    "write",
						Variables:        "write",
						WorkspaceLocking: true,
					},
				},
			}

			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())

				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())

			wsTeamAccess = buildWorkspaceTeamAccessByID(instance.Status.WorkspaceID, appv1alpha2.CustomPermissions{
				Runs:             "apply",
				RunTasks:         true,
				Sentinel:         "read",
				StateVersions:    "write",
				Variables:        "write",
				WorkspaceLocking: true,
			})
			Expect(wsTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))
		})

		It("can handle update from custom to pre-set team access", func() {
			instance.Spec.TeamAccess = append(instance.Spec.TeamAccess, &appv1alpha2.TeamAccess{
				Team: appv1alpha2.Team{
					Name: team.Name,
				},
				Access: "custom",
				Custom: appv1alpha2.CustomPermissions{
					Runs:             "apply",
					RunTasks:         true,
					Sentinel:         "read",
					StateVersions:    "write",
					Variables:        "write",
					WorkspaceLocking: true,
				},
			})
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isTeamAccessReconciled(instance)

			wsTeamAccess := buildWorkspaceTeamAccessByName(instance.Status.WorkspaceID, appv1alpha2.CustomPermissions{
				Runs:             "apply",
				RunTasks:         true,
				Sentinel:         "read",
				StateVersions:    "write",
				Variables:        "write",
				WorkspaceLocking: true,
			})
			Expect(wsTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))

			// UPDATE TEAM ACCESS
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.TeamAccess = []*appv1alpha2.TeamAccess{
				{
					Team: appv1alpha2.Team{
						ID: team.ID,
					},
					Access: "admin",
				},
			}

			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())

			wsTeamAccess = buildWorkspaceTeamAccessByID(instance.Status.WorkspaceID, appv1alpha2.CustomPermissions{
				Runs:             "read",
				RunTasks:         false,
				Sentinel:         "none",
				StateVersions:    "none",
				Variables:        "none",
				WorkspaceLocking: false,
			})
			Expect(wsTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))
		})
	})
})

func createTeam(teamName string) *tfc.Team {
	t, err := tfClient.Teams.Create(ctx, organization, tfc.TeamCreateOptions{
		Name:       tfc.String(teamName),
		Visibility: tfc.String("organization"),
	})
	Expect(err).Should(Succeed())
	Expect(t).ShouldNot(BeNil())
	return t
}

func isTeamAccessReconciled(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		Expect(instance.Spec.TeamAccess).ShouldNot(BeNil())

		teamAccesses, err := tfClient.TeamAccess.List(ctx, &tfc.TeamAccessListOptions{WorkspaceID: instance.Status.WorkspaceID})
		Expect(err).Should(Succeed())
		Expect(teamAccesses).ShouldNot(BeNil())

		return len(teamAccesses.Items) == len(instance.Spec.TeamAccess)
	}).Should(BeTrue())
}

func buildWorkspaceTeamAccessByName(wsID string, custom appv1alpha2.CustomPermissions) []*appv1alpha2.TeamAccess {
	return buildWorkspaceTeamAccess(wsID, true, custom)
}

func buildWorkspaceTeamAccessByID(wsID string, custom appv1alpha2.CustomPermissions) []*appv1alpha2.TeamAccess {
	return buildWorkspaceTeamAccess(wsID, false, custom)
}

func buildWorkspaceTeamAccess(wsID string, withTeamName bool, custom appv1alpha2.CustomPermissions) []*appv1alpha2.TeamAccess {
	teamAccesses, err := tfClient.TeamAccess.List(ctx, &tfc.TeamAccessListOptions{WorkspaceID: wsID})
	Expect(err).Should(Succeed())
	Expect(teamAccesses).ShouldNot(BeNil())

	wsTeamAccess := make([]*appv1alpha2.TeamAccess, len(teamAccesses.Items))

	for i, teamAccess := range teamAccesses.Items {
		t, err := tfClient.Teams.Read(ctx, teamAccess.Team.ID)
		Expect(err).Should(Succeed())
		Expect(t).ShouldNot(BeNil())
		team := appv1alpha2.Team{}
		if withTeamName {
			team.Name = t.Name
		} else {
			team.ID = t.ID
		}
		wsTeamAccess[i] = &appv1alpha2.TeamAccess{
			Team:   team,
			Access: string(teamAccess.Access),
			Custom: custom,
		}
	}

	return wsTeamAccess
}
