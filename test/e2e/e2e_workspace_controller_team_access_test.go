// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/teamworkspaces"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/controller"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName types.NamespacedName
		workspace      string
		team           models.Teamsable
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		team = createTeam(fmt.Sprintf("%s-team", workspace))
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
		deleteWorkspace(instance)
		Expect(tfClient.API.Teams().ById(pointer.ValueOf(team.GetId())).Delete(ctx, nil)).Should(Succeed())
	})

	Context("Team Access", func() {
		It("can handle pre-set team access", func() {
			instance.Spec.TeamAccess = []*appv1alpha2.TeamAccess{
				{
					Team: appv1alpha2.Team{
						Name: pointer.ValueOf(team.GetAttributes().GetName()),
					},
					Access: "admin",
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
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
						Name: pointer.ValueOf(team.GetAttributes().GetName()),
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
			createWorkspaceResource(instance)
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
						Name: pointer.ValueOf(team.GetAttributes().GetName()),
					},
					Access: "admin",
				},
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
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
						ID: pointer.ValueOf(team.GetId()),
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
					Name: pointer.ValueOf(team.GetAttributes().GetName()),
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
			createWorkspaceResource(instance)
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
						ID: pointer.ValueOf(team.GetId()),
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

func createTeam(teamName string) models.Teamsable {
	visibility, err := models.ParseTeams_attributes_visibility("organization")
	Expect(err).Should(Succeed())
	attrs := models.NewTeams_attributes()
	attrs.SetName(&teamName)
	attrs.SetVisibility(visibility.(*models.Teams_attributes_visibility))
	data := models.NewTeams()
	data.SetTypeEscaped(new(models.TEAMS_TEAMS_TYPE))
	data.SetAttributes(attrs)
	body := models.NewTeamsEnvelope()
	body.SetData(data)
	resp, err := tfClient.API.Organizations().ByOrganization_name(organization).Teams().Post(ctx, body, nil)
	Expect(err).Should(Succeed())
	Expect(resp).ShouldNot(BeNil())
	Expect(resp.GetData()).ShouldNot(BeNil())
	return resp.GetData()
}

func isTeamAccessReconciled(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		Expect(instance.Spec.TeamAccess).ShouldNot(BeNil())

		teamAccesses, err := tfClient.API.TeamWorkspaces().Get(ctx, &abstractions.RequestConfiguration[teamworkspaces.TeamWorkspacesRequestBuilderGetQueryParameters]{
			QueryParameters: &teamworkspaces.TeamWorkspacesRequestBuilderGetQueryParameters{
				Filterworkspaceid: &instance.Status.WorkspaceID,
				Pagesize:          new(int32(controller.MaxPageSize)),
			},
		})
		Expect(err).Should(Succeed())
		Expect(teamAccesses).ShouldNot(BeNil())

		return len(teamAccesses.GetData()) == len(instance.Spec.TeamAccess)
	}).Should(BeTrue())
}

func buildWorkspaceTeamAccessByName(wsID string, custom appv1alpha2.CustomPermissions) []*appv1alpha2.TeamAccess {
	return buildWorkspaceTeamAccess(wsID, true, custom)
}

func buildWorkspaceTeamAccessByID(wsID string, custom appv1alpha2.CustomPermissions) []*appv1alpha2.TeamAccess {
	return buildWorkspaceTeamAccess(wsID, false, custom)
}

func buildWorkspaceTeamAccess(wsID string, withTeamName bool, custom appv1alpha2.CustomPermissions) []*appv1alpha2.TeamAccess {
	teamAccesses, err := tfClient.API.TeamWorkspaces().Get(ctx, &abstractions.RequestConfiguration[teamworkspaces.TeamWorkspacesRequestBuilderGetQueryParameters]{
		QueryParameters: &teamworkspaces.TeamWorkspacesRequestBuilderGetQueryParameters{
			Filterworkspaceid: &wsID,
			Pagesize:          new(int32(controller.MaxPageSize)),
		},
	})
	Expect(err).Should(Succeed())
	Expect(teamAccesses).ShouldNot(BeNil())

	wsTeamAccess := make([]*appv1alpha2.TeamAccess, len(teamAccesses.GetData()))

	for i, teamAccess := range teamAccesses.GetData() {
		Expect(teamAccess.GetRelationships()).ShouldNot(BeNil())
		Expect(teamAccess.GetRelationships().GetTeam()).ShouldNot(BeNil())
		Expect(teamAccess.GetRelationships().GetTeam().GetData()).ShouldNot(BeNil())
		teamID := pointer.ValueOf(teamAccess.GetRelationships().GetTeam().GetData().GetId())
		t, err := tfClient.API.Teams().ById(teamID).Get(ctx, nil)
		Expect(err).Should(Succeed())
		Expect(t).ShouldNot(BeNil())
		Expect(t.GetData()).ShouldNot(BeNil())
		team := appv1alpha2.Team{}
		if withTeamName {
			Expect(t.GetData().GetAttributes()).ShouldNot(BeNil())
			team.Name = pointer.ValueOf(t.GetData().GetAttributes().GetName())
		} else {
			team.ID = pointer.ValueOf(t.GetData().GetId())
		}
		Expect(teamAccess.GetAttributes()).ShouldNot(BeNil())
		access := ""
		if a := teamAccess.GetAttributes().GetAccess(); a != nil {
			access = a.String()
		}
		wsTeamAccess[i] = &appv1alpha2.TeamAccess{
			Team:   team,
			Access: access,
			Custom: custom,
		}
	}

	return wsTeamAccess
}
