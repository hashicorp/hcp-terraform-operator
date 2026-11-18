// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"

	tfc "github.com/hashicorp/go-tfe/v2"

	"github.com/hashicorp/go-tfe/v2/api/teamprojects"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

var _ = Describe("Project controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Project
		namespacedName = newNamespacedName()
		teamName       string
		teamID         string
		project        = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create new teams
		teamName = fmt.Sprintf("%s-team", project)
		teamID = pointer.ValueOf(createTeam(teamName).GetId())
	})

	AfterAll(func() {
		err := tfClient.API.Teams().ById(teamID).Delete(ctx, nil)
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
			return kerrors.IsNotFound(err)
		}).Should(BeTrue())

		// Make sure that the HCP Terraform project is deleted
		Eventually(func() bool {
			err := tfClient.API.Projects().ByProject_id(instance.Status.ID).Delete(ctx, nil)
			// The HCP Terraform client will return the error 'ResourceNotFound' once the project does not exist
			return errors.Is(err, tfc.ErrNotFound) || err == nil
		}).Should(BeTrue())
	})

	Context("Project Team Access", func() {
		It("can handle pre-set team access", func() {
			instance.Spec.TeamAccess = []*appv1alpha2.ProjectTeamAccess{
				{
					Team: appv1alpha2.Team{
						Name: teamName,
					},
					Access: models.ADMIN_TEAMPROJECTS_ATTRIBUTES_ACCESS.String(),
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
						Name: teamName,
					},
					Access: models.CUSTOM_TEAMPROJECTS_ATTRIBUTES_ACCESS.String(),
					Custom: &appv1alpha2.CustomProjectPermissions{
						ProjectAccess: models.READ_TEAMPROJECTS_ATTRIBUTES_PROJECTACCESS_SETTINGS.String(),
					},
				},
			}
			// Create a new Kubernetes project object and wait until the controller finishes the reconciliation
			createProject(instance)
			isProjectTeamAccessReconciled(instance)

			prjTeamAccess := buildProjectTeamAccessByName(instance.Status.ID, &appv1alpha2.CustomProjectPermissions{
				ProjectAccess:   models.READ_TEAMPROJECTS_ATTRIBUTES_PROJECTACCESS_SETTINGS.String(),
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
						Name: teamName,
					},
					Access: models.ADMIN_TEAMPROJECTS_ATTRIBUTES_ACCESS.String(),
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
						ID: teamID,
					},
					Access: models.CUSTOM_TEAMPROJECTS_ATTRIBUTES_ACCESS.String(),
					Custom: &appv1alpha2.CustomProjectPermissions{
						ProjectAccess: models.READ_TEAMPROJECTS_ATTRIBUTES_PROJECTACCESS_SETTINGS.String(),
					},
				},
			}

			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
				return instance.Status.ObservedGeneration == instance.Generation
			}).Should(BeTrue())

			prjTeamAccess = buildProjectTeamAccessByID(instance.Status.ID, &appv1alpha2.CustomProjectPermissions{
				ProjectAccess:   models.READ_TEAMPROJECTS_ATTRIBUTES_PROJECTACCESS_SETTINGS.String(),
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
					Name: teamName,
				},
				Access: models.CUSTOM_TEAMPROJECTS_ATTRIBUTES_ACCESS.String(),
				Custom: &appv1alpha2.CustomProjectPermissions{
					ProjectAccess: models.READ_TEAMPROJECTS_ATTRIBUTES_PROJECTACCESS_SETTINGS.String(),
				},
			})
			// Create a new Kubernetes project object and wait until the controller finishes the reconciliation
			createProject(instance)
			isProjectTeamAccessReconciled(instance)

			prjTeamAccess := buildProjectTeamAccessByName(instance.Status.ID, &appv1alpha2.CustomProjectPermissions{
				ProjectAccess:   models.READ_TEAMPROJECTS_ATTRIBUTES_PROJECTACCESS_SETTINGS.String(),
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
						ID: teamID,
					},
					Access: models.ADMIN_TEAMPROJECTS_ATTRIBUTES_ACCESS.String(),
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

		teamAccesses, err := tfClient.API.TeamProjects().Get(ctx, &abstractions.RequestConfiguration[teamprojects.TeamProjectsRequestBuilderGetQueryParameters]{
			QueryParameters: &teamprojects.TeamProjectsRequestBuilderGetQueryParameters{
				Filterprojectid: &instance.Status.ID,
			},
		})
		Expect(err).Should(Succeed())

		return len(teamAccesses.GetData()) == len(instance.Spec.TeamAccess)
	}).Should(BeTrue())
}

func buildProjectTeamAccessByName(ID string, custom *appv1alpha2.CustomProjectPermissions) []*appv1alpha2.ProjectTeamAccess {
	return buildProjectTeamAccess(ID, true, custom)
}

func buildProjectTeamAccessByID(ID string, custom *appv1alpha2.CustomProjectPermissions) []*appv1alpha2.ProjectTeamAccess {
	return buildProjectTeamAccess(ID, false, custom)
}

func buildProjectTeamAccess(ID string, withTeamName bool, custom *appv1alpha2.CustomProjectPermissions) []*appv1alpha2.ProjectTeamAccess {
	teamAccesses, err := tfClient.API.TeamProjects().Get(ctx, &abstractions.RequestConfiguration[teamprojects.TeamProjectsRequestBuilderGetQueryParameters]{
		QueryParameters: &teamprojects.TeamProjectsRequestBuilderGetQueryParameters{
			Filterprojectid: &ID,
		},
	})
	Expect(err).Should(Succeed())

	prjTeamAccess := make([]*appv1alpha2.ProjectTeamAccess, len(teamAccesses.GetData()))

	for i, teamAccess := range teamAccesses.GetData() {
		teamID := pointer.ValueOf(teamAccess.GetRelationships().GetTeam().GetData().GetId())
		t, err := tfClient.API.Teams().ById(teamID).Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(t).ShouldNot(BeNil())
		team := appv1alpha2.Team{}
		if withTeamName {
			team.Name = pointer.ValueOf(t.GetData().GetAttributes().GetName())
		} else {
			team.ID = pointer.ValueOf(t.GetData().GetId())
		}
		prjTeamAccess[i] = &appv1alpha2.ProjectTeamAccess{
			Team:   team,
			Access: string(teamAccess.GetAttributes().GetAccess().String()),
			Custom: custom,
		}
	}

	return prjTeamAccess
}
