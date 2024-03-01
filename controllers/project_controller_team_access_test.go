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

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Project controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Project
		namespacedName = newNamespacedName()
		team           *tfc.Team
		project        = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create new teams
		team = createTeam(fmt.Sprintf("%s-team", project))
	})

	AfterAll(func() {
		err := tfClient.Teams.Delete(ctx, team.ID)
		Expect(err).Should(Succeed())
	})

	BeforeEach(func() {
		// Create a new project object for each test
		instance = &appv1alpha2.Project{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "app.terraform.io/v1alpha2",
				Kind:       "Project",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              namespacedName.Name,
				Namespace:         namespacedName.Namespace,
				DeletionTimestamp: nil,
				Finalizers:        []string{},
			},
			Spec: appv1alpha2.ProjectSpec{
				Organization: organization,
				Token: appv1alpha2.Token{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretNamespacedName.Name,
						},
						Key: secretKey,
					},
				},
				Name: project,
			},
			Status: appv1alpha2.ProjectStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes Project object and wait until the controller finishes the reconciliation after deletion of the object
		Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, namespacedName, instance)
			// The Kubernetes client will return error 'NotFound' on the Get operation once the object is deleted
			return errors.IsNotFound(err)
		}).Should(BeTrue())

		// Make sure that the Terraform Cloud project is deleted
		Eventually(func() bool {
			err := tfClient.Projects.Delete(ctx, instance.Status.ID)
			// The Terraform Cloud client will return the error 'ResourceNotFound' once the project does not exist
			return err == tfc.ErrResourceNotFound || err == nil
		}).Should(BeTrue())
	})

	Context("Project Team Access", func() {
		It("can handle pre-set team access", func() {
			instance.Spec.TeamAccess = []*appv1alpha2.ProjectTeamAccess{
				{
					Team: appv1alpha2.Team{
						Name: team.Name,
					},
					Access: tfc.TeamProjectAccessAdmin,
				},
			}
			// Create a new Kubernetes project object and wait until the controller finishes the reconciliation
			createProject(instance)
			isProjectTeamAccessReconciled(instance)

			prjTeamAccess := buildProjectTeamAccessByName(instance.Status.ID, nil)
			Expect(prjTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))
		})

		It("can handle custom team access", func() {
			instance.Spec.TeamAccess = []*appv1alpha2.ProjectTeamAccess{
				{
					Team: appv1alpha2.Team{
						Name: team.Name,
					},
					Access: tfc.TeamProjectAccessCustom,
					Custom: &appv1alpha2.CustomProjectPermissions{
						ProjectAccess: tfc.ProjectSettingsPermissionRead,
					},
				},
			}
			// Create a new Kubernetes project object and wait until the controller finishes the reconciliation
			createProject(instance)
			isProjectTeamAccessReconciled(instance)

			prjTeamAccess := buildProjectTeamAccessByName(instance.Status.ID, &appv1alpha2.CustomProjectPermissions{
				ProjectAccess:   tfc.ProjectSettingsPermissionRead,
				TeamManagement:  "none",
				CreateWorkspace: false,
				DeleteWorkspace: false,
				MoveWorkspace:   false,
				LockWorkspace:   false,
				Runs:            "read",
				RunTasks:        false,
				SentinelMocks:   "none",
				StateVersions:   "none",
				Variables:       "none",
			})
			Expect(prjTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))
		})

		It("can handle update from pre-set to custom team access", func() {
			instance.Spec.TeamAccess = []*appv1alpha2.ProjectTeamAccess{
				{
					Team: appv1alpha2.Team{
						Name: team.Name,
					},
					Access: tfc.TeamProjectAccessAdmin,
				},
			}
			// Create a new Kubernetes project object and wait until the controller finishes the reconciliation
			createProject(instance)
			isProjectTeamAccessReconciled(instance)

			prjTeamAccess := buildProjectTeamAccessByName(instance.Status.ID, nil)
			Expect(prjTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))

			// UPDATE TEAM ACCESS
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.TeamAccess = []*appv1alpha2.ProjectTeamAccess{
				{
					Team: appv1alpha2.Team{
						ID: team.ID,
					},
					Access: tfc.TeamProjectAccessCustom,
					Custom: &appv1alpha2.CustomProjectPermissions{
						ProjectAccess: tfc.ProjectSettingsPermissionRead,
					},
				},
			}

			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())

			prjTeamAccess = buildProjectTeamAccessByID(instance.Status.ID, &appv1alpha2.CustomProjectPermissions{
				ProjectAccess:   tfc.ProjectSettingsPermissionRead,
				TeamManagement:  "none",
				CreateWorkspace: false,
				DeleteWorkspace: false,
				MoveWorkspace:   false,
				LockWorkspace:   false,
				Runs:            "read",
				RunTasks:        false,
				SentinelMocks:   "none",
				StateVersions:   "none",
				Variables:       "none",
			})
			Expect(prjTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))
		})

		It("can handle update from custom to pre-set team access", func() {
			instance.Spec.TeamAccess = append(instance.Spec.TeamAccess, &appv1alpha2.ProjectTeamAccess{
				Team: appv1alpha2.Team{
					Name: team.Name,
				},
				Access: tfc.TeamProjectAccessCustom,
				Custom: &appv1alpha2.CustomProjectPermissions{
					ProjectAccess: tfc.ProjectSettingsPermissionRead,
				},
			})
			// Create a new Kubernetes project object and wait until the controller finishes the reconciliation
			createProject(instance)
			isProjectTeamAccessReconciled(instance)

			prjTeamAccess := buildProjectTeamAccessByName(instance.Status.ID, &appv1alpha2.CustomProjectPermissions{
				ProjectAccess:   tfc.ProjectSettingsPermissionRead,
				TeamManagement:  "none",
				CreateWorkspace: false,
				DeleteWorkspace: false,
				MoveWorkspace:   false,
				LockWorkspace:   false,
				Runs:            "read",
				RunTasks:        false,
				SentinelMocks:   "none",
				StateVersions:   "none",
				Variables:       "none",
			})
			Expect(prjTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))

			// UPDATE TEAM ACCESS
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.TeamAccess = []*appv1alpha2.ProjectTeamAccess{
				{
					Team: appv1alpha2.Team{
						ID: team.ID,
					},
					Access: tfc.TeamProjectAccessAdmin,
				},
			}

			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())

			prjTeamAccess = buildProjectTeamAccessByID(instance.Status.ID, nil)
			Expect(prjTeamAccess).Should(ConsistOf(instance.Spec.TeamAccess))
		})
	})
})

func isProjectTeamAccessReconciled(instance *appv1alpha2.Project) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		Expect(instance.Spec.TeamAccess).ShouldNot(BeNil())

		teamAccesses, err := tfClient.TeamProjectAccess.List(ctx, tfc.TeamProjectAccessListOptions{ProjectID: instance.Status.ID})
		Expect(err).Should(Succeed())
		Expect(teamAccesses).ShouldNot(BeNil())

		return len(teamAccesses.Items) == len(instance.Spec.TeamAccess)
	}).Should(BeTrue())
}

func buildProjectTeamAccessByName(ID string, custom *appv1alpha2.CustomProjectPermissions) []*appv1alpha2.ProjectTeamAccess {
	return buildProjectTeamAccess(ID, true, custom)
}

func buildProjectTeamAccessByID(ID string, custom *appv1alpha2.CustomProjectPermissions) []*appv1alpha2.ProjectTeamAccess {
	return buildProjectTeamAccess(ID, false, custom)
}

func buildProjectTeamAccess(ID string, withTeamName bool, custom *appv1alpha2.CustomProjectPermissions) []*appv1alpha2.ProjectTeamAccess {
	teamAccesses, err := tfClient.TeamProjectAccess.List(ctx, tfc.TeamProjectAccessListOptions{ProjectID: ID})
	Expect(err).Should(Succeed())
	Expect(teamAccesses).ShouldNot(BeNil())

	prjTeamAccess := make([]*appv1alpha2.ProjectTeamAccess, len(teamAccesses.Items))

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
		prjTeamAccess[i] = &appv1alpha2.ProjectTeamAccess{
			Team:   team,
			Access: teamAccess.Access,
			Custom: custom,
		}
	}

	return prjTeamAccess
}
