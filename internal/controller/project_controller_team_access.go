// // Copyright IBM Corp. 2022, 2025
// // SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	"github.com/hashicorp/go-tfe/v2/api/teamprojects"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	corev1 "k8s.io/api/core/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

// projectTeamAccess is a team access to a project.
// It is the plain comparable representation used to diff the spec against HCP Terraform.
type projectTeamAccess struct {
	ID     string
	Access string
	Custom appv1alpha2.CustomProjectPermissions
}

// setEnum converts a string into a generated Kiota enum value and sets it using a given setter.
// If the value is not a known enum member, the raw string is sent as additional data instead,
// so the API rejects it the same way it did with go-tfe v1, which did not validate values client-side.
func setEnum[T any](parse func(string) (any, error), v string, set func(*T), extra map[string]any, key string) {
	if r, err := parse(v); err == nil {
		if t, ok := r.(*T); ok && t != nil {
			set(t)
			return
		}
	}
	extra[key] = v
}

func (r *ProjectReconciler) getInstanceTeamAccess(ctx context.Context, p *projectInstance) (map[string]*projectTeamAccess, error) {
	o := map[string]*projectTeamAccess{}

	if p.instance.Spec.TeamAccess == nil {
		return o, nil
	}

	teams, err := r.getTeams(ctx, p)
	if err != nil {
		return o, err
	}

	for _, ta := range p.instance.Spec.TeamAccess {
		tID, err := getTeamID(teams, ta.Team)
		if err != nil {
			p.log.Error(err, "Reconcile Team Access", "msg", "failed to get team ID")
			r.Recorder.Event(&p.instance, corev1.EventTypeWarning, "ReconcileTeamAccess", "Failed to get team ID")
			return o, err
		}

		o[tID] = &projectTeamAccess{
			Access: ta.Access,
		}
		if ta.Access == models.CUSTOM_TEAMPROJECTS_ATTRIBUTES_ACCESS.String() && ta.Custom != nil {
			o[tID].Custom = *ta.Custom
		}
	}

	return o, nil
}

func (r *ProjectReconciler) getWorkspaceTeamAccess(ctx context.Context, p *projectInstance) (map[string]*projectTeamAccess, error) {
	o := map[string]*projectTeamAccess{}

	pageNumber := int32(InitPageNumber)
	for {
		resp, err := p.tfClient.Client.API.TeamProjects().Get(ctx, &abstractions.RequestConfiguration[teamprojects.TeamProjectsRequestBuilderGetQueryParameters]{
			QueryParameters: &teamprojects.TeamProjectsRequestBuilderGetQueryParameters{
				Filterprojectid: new(p.instance.Status.ID),
				Pagesize:        new(int32(MaxPageSize)),
				Pagenumber:      &pageNumber,
			},
		})
		if err != nil {
			return o, err
		}
		for _, t := range resp.GetData() {
			ta := &projectTeamAccess{ID: pointer.ValueOf(t.GetId())}
			if a := t.GetAttributes(); a != nil {
				if v := a.GetAccess(); v != nil {
					ta.Access = v.String()
				}
				if pa := a.GetProjectAccess(); pa != nil {
					if v := pa.GetSettings(); v != nil {
						ta.Custom.ProjectAccess = v.String()
					}
					if v := pa.GetTeams(); v != nil {
						ta.Custom.TeamManagement = v.String()
					}
				}
				if wa := a.GetWorkspaceAccess(); wa != nil {
					if v := wa.GetRuns(); v != nil {
						ta.Custom.Runs = v.String()
					}
					if v := wa.GetSentinelMocks(); v != nil {
						ta.Custom.SentinelMocks = v.String()
					}
					if v := wa.GetStateVersions(); v != nil {
						ta.Custom.StateVersions = v.String()
					}
					if v := wa.GetVariables(); v != nil {
						ta.Custom.Variables = v.String()
					}
					ta.Custom.CreateWorkspace = pointer.ValueOf(wa.GetCreate())
					ta.Custom.LockWorkspace = pointer.ValueOf(wa.GetLocking())
					ta.Custom.MoveWorkspace = pointer.ValueOf(wa.GetMove())
					ta.Custom.DeleteWorkspace = pointer.ValueOf(wa.GetDelete())
					ta.Custom.RunTasks = pointer.ValueOf(wa.GetRunTasks())
				}
			}
			if team := t.GetRelationships().GetTeam(); team != nil && team.GetData() != nil {
				o[pointer.ValueOf(team.GetData().GetId())] = ta
			}
		}
		next := nextPage(resp.GetMeta())
		if next == 0 {
			break
		}
		pageNumber = next
	}
	return o, nil
}

// getTeams returns teams of the instance team access as a map of the team name to the team ID.
func (r *ProjectReconciler) getTeams(ctx context.Context, p *projectInstance) (map[string]string, error) {
	teams := make(map[string]string)

	fTeams := []string{}
	for _, t := range p.instance.Spec.TeamAccess {
		if t.Team.Name != "" {
			fTeams = append(fTeams, t.Team.Name)
		}
	}

	qp := &organizations.ItemTeamsRequestBuilderGetQueryParameters{
		Pagesize: new(int32(MaxPageSize)),
	}
	if len(fTeams) > 0 {
		qp.Filternames = new(strings.Join(fTeams, ","))
	}
	pageNumber := int32(InitPageNumber)
	for {
		qp.Pagenumber = &pageNumber
		resp, err := p.tfClient.Client.API.Organizations().ByOrganization_name(p.instance.Spec.Organization).Teams().Get(ctx, &abstractions.RequestConfiguration[organizations.ItemTeamsRequestBuilderGetQueryParameters]{
			QueryParameters: qp,
		})
		if err != nil {
			return teams, err
		}
		for _, t := range resp.GetData() {
			if t.GetAttributes() == nil {
				continue
			}
			teams[pointer.ValueOf(t.GetAttributes().GetName())] = pointer.ValueOf(t.GetId())
		}
		next := nextPage(resp.GetMeta())
		if next == 0 {
			break
		}
		pageNumber = next
	}

	return teams, nil
}

func teamProjectAccessDifference(a, b map[string]*projectTeamAccess) map[string]*projectTeamAccess {
	d := make(map[string]*projectTeamAccess)

	for k, v := range a {
		if _, ok := b[k]; !ok {
			d[k] = v
		}
	}

	return d
}

func getTeamProjectAccessToCreate(specTeamAccess, projectTeamAccess map[string]*projectTeamAccess) map[string]*projectTeamAccess {
	return teamProjectAccessDifference(specTeamAccess, projectTeamAccess)
}

func getTeamProjectAccessToDelete(specTeamAccess, projectTeamAccess map[string]*projectTeamAccess) map[string]*projectTeamAccess {
	return teamProjectAccessDifference(projectTeamAccess, specTeamAccess)
}

func getTeamProjectAccessToUpdate(specTeamAccess, workspaceTeamAccess map[string]*projectTeamAccess) map[string]*projectTeamAccess {
	ta := make(map[string]*projectTeamAccess)

	if len(specTeamAccess) == 0 || len(workspaceTeamAccess) == 0 {
		return ta
	}

	for ik, iv := range specTeamAccess {
		if wv, ok := workspaceTeamAccess[ik]; ok {
			iv.ID = wv.ID
			if iv.Access == models.CUSTOM_TEAMPROJECTS_ATTRIBUTES_ACCESS.String() {
				if iv.Access != wv.Access || iv.Custom != wv.Custom {
					ta[ik] = iv
				}
			} else {
				if iv.Access != wv.Access {
					ta[ik] = iv
				}
			}
		}
	}

	return ta
}

func (r *ProjectReconciler) createTeamProjectAccess(ctx context.Context, p *projectInstance, createTeamAccess map[string]*projectTeamAccess) error {
	for tID, v := range createTeamAccess {
		attrs := models.NewTeamProjects_attributes()
		setEnum(models.ParseTeamProjects_attributes_access, string(v.Access), attrs.SetAccess, attrs.GetAdditionalData(), "access")
		if v.Access == models.CUSTOM_TEAMPROJECTS_ATTRIBUTES_ACCESS.String() {
			pa := models.NewTeamProjects_attributes_projectAccess()
			setEnum(models.ParseTeamProjects_attributes_projectAccess_settings, string(v.Custom.ProjectAccess), pa.SetSettings, pa.GetAdditionalData(), "settings")
			setEnum(models.ParseTeamProjects_attributes_projectAccess_teams, string(v.Custom.TeamManagement), pa.SetTeams, pa.GetAdditionalData(), "teams")
			attrs.SetProjectAccess(pa)

			wa := models.NewTeamProjects_attributes_workspaceAccess()
			setEnum(models.ParseTeamProjects_attributes_workspaceAccess_runs, string(v.Custom.Runs), wa.SetRuns, wa.GetAdditionalData(), "runs")
			setEnum(models.ParseTeamProjects_attributes_workspaceAccess_sentinelMocks, string(v.Custom.SentinelMocks), wa.SetSentinelMocks, wa.GetAdditionalData(), "sentinel-mocks")
			setEnum(models.ParseTeamProjects_attributes_workspaceAccess_stateVersions, string(v.Custom.StateVersions), wa.SetStateVersions, wa.GetAdditionalData(), "state-versions")
			setEnum(models.ParseTeamProjects_attributes_workspaceAccess_variables, string(v.Custom.Variables), wa.SetVariables, wa.GetAdditionalData(), "variables")
			wa.SetCreate(new(v.Custom.CreateWorkspace))
			wa.SetLocking(new(v.Custom.LockWorkspace))
			wa.SetMove(new(v.Custom.MoveWorkspace))
			wa.SetDelete(new(v.Custom.DeleteWorkspace))
			wa.SetRunTasks(new(v.Custom.RunTasks))
			attrs.SetWorkspaceAccess(wa)
		}

		projectData := models.NewProjectsHasOne_data()
		projectData.SetId(new(p.instance.Status.ID))
		projectData.SetTypeEscaped(new(models.PROJECTS_PROJECTSIDENTIFIER_TYPE))
		projectRel := models.NewProjectsHasOne()
		projectRel.SetData(projectData)

		teamData := models.NewTeamsIdentifier()
		teamData.SetId(new(tID))
		teamData.SetTypeEscaped(new(models.TEAMS_TEAMSIDENTIFIER_TYPE))
		teamRel := models.NewTeamsHasOne()
		teamRel.SetData(teamData)

		rels := models.NewTeamProjects_relationships()
		rels.SetProject(projectRel)
		rels.SetTeam(teamRel)

		data := models.NewTeamProjects()
		data.SetTypeEscaped(new(models.TEAMPROJECTS_TEAMPROJECTS_TYPE))
		data.SetAttributes(attrs)
		data.SetRelationships(rels)

		body := models.NewTeamProjectsEnvelope()
		body.SetData(data)

		_, err := p.tfClient.Client.API.TeamProjects().Post(ctx, body, nil)
		if err != nil {
			p.log.Error(err, "Reconcile Team Access", "msg", "failed to create a new team access")
			return err
		}
	}

	return nil
}

func (r *ProjectReconciler) deleteTeamAccess(ctx context.Context, p *projectInstance, deleteTeamAccess map[string]*projectTeamAccess) error {
	for _, v := range deleteTeamAccess {
		err := p.tfClient.Client.API.TeamProjects().ByTeam_project_id(v.ID).Delete(ctx, nil)
		if err != nil {
			p.log.Error(err, "Reconcile Team Access", "msg", "failed to delete team access")
			return err
		}
	}

	return nil
}

func (r *ProjectReconciler) updateTeamAccess(ctx context.Context, p *projectInstance, updateTeamAccess map[string]*projectTeamAccess) error {
	for _, v := range updateTeamAccess {
		p.log.Info("Reconcile Team Access", "msg", "updating team access")

		attrs := models.NewTeamProjects_attributes()
		setEnum(models.ParseTeamProjects_attributes_access, string(v.Access), attrs.SetAccess, attrs.GetAdditionalData(), "access")
		if v.Access == models.CUSTOM_TEAMPROJECTS_ATTRIBUTES_ACCESS.String() {
			pa := models.NewTeamProjects_attributes_projectAccess()
			setEnum(models.ParseTeamProjects_attributes_projectAccess_settings, string(v.Custom.ProjectAccess), pa.SetSettings, pa.GetAdditionalData(), "settings")
			setEnum(models.ParseTeamProjects_attributes_projectAccess_teams, string(v.Custom.TeamManagement), pa.SetTeams, pa.GetAdditionalData(), "teams")
			attrs.SetProjectAccess(pa)

			wa := models.NewTeamProjects_attributes_workspaceAccess()
			setEnum(models.ParseTeamProjects_attributes_workspaceAccess_runs, string(v.Custom.Runs), wa.SetRuns, wa.GetAdditionalData(), "runs")
			setEnum(models.ParseTeamProjects_attributes_workspaceAccess_sentinelMocks, string(v.Custom.SentinelMocks), wa.SetSentinelMocks, wa.GetAdditionalData(), "sentinel-mocks")
			setEnum(models.ParseTeamProjects_attributes_workspaceAccess_stateVersions, string(v.Custom.StateVersions), wa.SetStateVersions, wa.GetAdditionalData(), "state-versions")
			setEnum(models.ParseTeamProjects_attributes_workspaceAccess_variables, string(v.Custom.Variables), wa.SetVariables, wa.GetAdditionalData(), "variables")
			wa.SetCreate(new(v.Custom.CreateWorkspace))
			wa.SetLocking(new(v.Custom.LockWorkspace))
			wa.SetMove(new(v.Custom.MoveWorkspace))
			wa.SetDelete(new(v.Custom.DeleteWorkspace))
			wa.SetRunTasks(new(v.Custom.RunTasks))
			attrs.SetWorkspaceAccess(wa)
		}

		data := models.NewTeamProjects()
		data.SetTypeEscaped(new(models.TEAMPROJECTS_TEAMPROJECTS_TYPE))
		data.SetId(new(v.ID))
		data.SetAttributes(attrs)

		body := models.NewTeamProjectsEnvelope()
		body.SetData(data)

		_, err := p.tfClient.Client.API.TeamProjects().ByTeam_project_id(v.ID).Patch(ctx, body, nil)
		if err != nil {
			p.log.Error(err, "Reconcile Team Access", "msg", "failed to update team access")
			return err
		}
	}

	return nil
}

func (r *ProjectReconciler) reconcileTeamAccess(ctx context.Context, p *projectInstance) error {
	p.log.Info("Reconcile Team Access", "msg", "new reconciliation event")

	specTeamAccess, err := r.getInstanceTeamAccess(ctx, p)
	if err != nil {
		p.log.Error(err, "Reconcile Team Access", "msg", "failed to get instance team access")
		return err
	}

	projectTeamAccess, err := r.getWorkspaceTeamAccess(ctx, p)
	if err != nil {
		p.log.Error(err, "Reconcile Team Access", "msg", "failed to get project team access")
		return err
	}

	createTeamAccess := getTeamProjectAccessToCreate(specTeamAccess, projectTeamAccess)
	if len(createTeamAccess) > 0 {
		p.log.Info("Reconcile Team Access", "msg", fmt.Sprintf("creating %d team accesses", len(createTeamAccess)))
		err := r.createTeamProjectAccess(ctx, p, createTeamAccess)
		if err != nil {
			return err
		}
	}

	updateTeamAccess := getTeamProjectAccessToUpdate(specTeamAccess, projectTeamAccess)
	if len(updateTeamAccess) > 0 {
		p.log.Info("Reconcile Team Access", "msg", fmt.Sprintf("updating %d team accesses", len(updateTeamAccess)))
		err := r.updateTeamAccess(ctx, p, updateTeamAccess)
		if err != nil {
			return err
		}
	}

	deleteTeamAccess := getTeamProjectAccessToDelete(specTeamAccess, projectTeamAccess)
	if len(deleteTeamAccess) > 0 {
		p.log.Info("Reconcile Team Access", "msg", fmt.Sprintf("deleting %d team accesses", len(deleteTeamAccess)))
		err := r.deleteTeamAccess(ctx, p, deleteTeamAccess)
		if err != nil {
			return err
		}
	}

	return nil
}
