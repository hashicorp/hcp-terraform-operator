// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	"github.com/hashicorp/go-tfe/v2/api/teamworkspaces"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	corev1 "k8s.io/api/core/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

const teamAccessCustom = "custom"

type teamAccess struct {
	ID               string
	TeamID           string
	Access           string
	Runs             string
	RunTasks         bool
	SentinelMocks    string
	StateVersions    string
	Variables        string
	WorkspaceLocking bool
}

// getTeamID resolves a team ID from a map of team name -> team ID.
func getTeamID(teams map[string]string, instanceTeam appv1alpha2.Team) (string, error) {
	if instanceTeam.Name != "" {
		if id, ok := teams[instanceTeam.Name]; ok {
			return id, nil
		}
		return "", fmt.Errorf("team ID was not found by name %q", instanceTeam.Name)
	}

	if instanceTeam.ID != "" {
		for _, id := range teams {
			if id == instanceTeam.ID {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("team ID was not found by ID %q", instanceTeam.ID)
}

func (r *WorkspaceReconciler) getInstanceTeamAccess(ctx context.Context, w *workspaceInstance) (map[string]*teamAccess, error) {
	o := map[string]*teamAccess{}

	if w.instance.Spec.TeamAccess == nil {
		return o, nil
	}

	teams, err := r.getTeams(ctx, w)
	if err != nil {
		return o, err
	}

	for _, ta := range w.instance.Spec.TeamAccess {
		tID, err := getTeamID(teams, ta.Team)
		if err != nil {
			w.log.Error(err, "Reconcile Team Access", "msg", "failed to get team ID")
			r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileTeamAccess", "Failed to get team ID")
			return o, err
		}

		o[tID] = &teamAccess{
			TeamID:           tID,
			Access:           ta.Access,
			Runs:             ta.Custom.Runs,
			RunTasks:         ta.Custom.RunTasks,
			SentinelMocks:    ta.Custom.Sentinel,
			StateVersions:    ta.Custom.StateVersions,
			Variables:        ta.Custom.Variables,
			WorkspaceLocking: ta.Custom.WorkspaceLocking,
		}
	}

	return o, nil
}

func (r *WorkspaceReconciler) getWorkspaceTeamAccess(ctx context.Context, w *workspaceInstance) (map[string]*teamAccess, error) {
	o := map[string]*teamAccess{}

	query := &teamworkspaces.TeamWorkspacesRequestBuilderGetQueryParameters{
		Filterworkspaceid: &w.instance.Status.WorkspaceID,
		Pagesize:          new(int32(MaxPageSize)),
		Pagenumber:        new(int32(InitPageNumber)),
	}
	for {
		t, err := w.tfClient.Client.API.TeamWorkspaces().Get(ctx,
			&abstractions.RequestConfiguration[teamworkspaces.TeamWorkspacesRequestBuilderGetQueryParameters]{QueryParameters: query})
		if err != nil {
			return o, err
		}
		for _, d := range t.GetData() {
			ta := &teamAccess{ID: pointer.ValueOf(d.GetId())}
			if rel := d.GetRelationships(); rel != nil && rel.GetTeam() != nil && rel.GetTeam().GetData() != nil {
				ta.TeamID = pointer.ValueOf(rel.GetTeam().GetData().GetId())
			}
			if attrs := d.GetAttributes(); attrs != nil {
				if v := attrs.GetAccess(); v != nil {
					ta.Access = v.String()
				}
				if v := attrs.GetRuns(); v != nil {
					ta.Runs = v.String()
				}
				ta.RunTasks = pointer.ValueOf(attrs.GetRunTasks())
				if v := attrs.GetSentinelMocks(); v != nil {
					ta.SentinelMocks = v.String()
				}
				if v := attrs.GetStateVersions(); v != nil {
					ta.StateVersions = v.String()
				}
				if v := attrs.GetVariables(); v != nil {
					ta.Variables = v.String()
				}
				ta.WorkspaceLocking = pointer.ValueOf(attrs.GetWorkspaceLocking())
			}
			o[ta.TeamID] = ta
		}
		next := nextPage(t.GetMeta())
		if next == 0 {
			break
		}
		query.Pagenumber = &next
	}
	return o, nil
}

// getTeams returns a map of team name -> team ID for the teams referenced by name in the spec.
func (r *WorkspaceReconciler) getTeams(ctx context.Context, w *workspaceInstance) (map[string]string, error) {
	teams := make(map[string]string)

	fTeams := []string{}
	for _, t := range w.instance.Spec.TeamAccess {
		if t.Team.Name != "" {
			fTeams = append(fTeams, t.Team.Name)
		}
	}

	query := &organizations.ItemTeamsRequestBuilderGetQueryParameters{
		Pagesize:   new(int32(MaxPageSize)),
		Pagenumber: new(int32(InitPageNumber)),
	}
	if len(fTeams) > 0 {
		query.Filternames = new(strings.Join(fTeams, ","))
	}
	for {
		tl, err := w.tfClient.Client.API.Organizations().ByOrganization_name(w.instance.Spec.Organization).Teams().Get(ctx,
			&abstractions.RequestConfiguration[organizations.ItemTeamsRequestBuilderGetQueryParameters]{QueryParameters: query})
		if err != nil {
			return teams, err
		}
		for _, t := range tl.GetData() {
			if t.GetAttributes() == nil {
				continue
			}
			teams[pointer.ValueOf(t.GetAttributes().GetName())] = pointer.ValueOf(t.GetId())
		}
		next := nextPage(tl.GetMeta())
		if next == 0 {
			break
		}
		query.Pagenumber = &next
	}

	return teams, nil
}

func teamAccessDifference(a, b map[string]*teamAccess) map[string]*teamAccess {
	d := make(map[string]*teamAccess)

	for k, v := range a {
		if _, ok := b[k]; !ok {
			d[k] = v
		}
	}

	return d
}

func getTeamAccessToCreate(specTeamAccess, workspaceTeamAccess map[string]*teamAccess) map[string]*teamAccess {
	return teamAccessDifference(specTeamAccess, workspaceTeamAccess)
}

func getTeamAccessToDelete(specTeamAccess, workspaceTeamAccess map[string]*teamAccess) map[string]*teamAccess {
	return teamAccessDifference(workspaceTeamAccess, specTeamAccess)
}

func getTeamAccessToUpdate(specTeamAccess, workspaceTeamAccess map[string]*teamAccess) map[string]*teamAccess {
	ta := make(map[string]*teamAccess)

	if len(specTeamAccess) == 0 || len(workspaceTeamAccess) == 0 {
		return ta
	}

	for ik, iv := range specTeamAccess {
		if wv, ok := workspaceTeamAccess[ik]; ok {
			iv.ID = wv.ID
			if iv.Access == teamAccessCustom {
				if !cmp.Equal(iv, wv, cmpopts.IgnoreFields(teamAccess{}, "ID", "TeamID")) {
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

// teamAccessAttributes builds the team-workspace attributes for a given team access.
//
// go-tfe v1 sent every given string as is, including empty ones, and left the validation to the API.
// The generated enum types cannot hold an empty or unknown value, thus such values are sent
// as additional (raw) attributes to keep the request payload the same.
func teamAccessAttributes(v *teamAccess) models.TeamWorkspaces_attributesable {
	attrs := models.NewTeamWorkspaces_attributes()
	raw := map[string]any{}

	access, ok := parseEnum[models.TeamWorkspaces_attributes_access](v.Access, models.ParseTeamWorkspaces_attributes_access)
	if ok {
		attrs.SetAccess(access)
	} else {
		raw["access"] = v.Access
	}

	if v.Access == teamAccessCustom {
		runs, ok := parseEnum[models.TeamWorkspaces_attributes_runs](v.Runs, models.ParseTeamWorkspaces_attributes_runs)
		if ok {
			attrs.SetRuns(runs)
		} else {
			raw["runs"] = v.Runs
		}
		sentinel, ok := parseEnum[models.TeamWorkspaces_attributes_sentinelMocks](v.SentinelMocks, models.ParseTeamWorkspaces_attributes_sentinelMocks)
		if ok {
			attrs.SetSentinelMocks(sentinel)
		} else {
			raw["sentinel-mocks"] = v.SentinelMocks
		}
		stateVersions, ok := parseEnum[models.TeamWorkspaces_attributes_stateVersions](v.StateVersions, models.ParseTeamWorkspaces_attributes_stateVersions)
		if ok {
			attrs.SetStateVersions(stateVersions)
		} else {
			raw["state-versions"] = v.StateVersions
		}
		variables, ok := parseEnum[models.TeamWorkspaces_attributes_variables](v.Variables, models.ParseTeamWorkspaces_attributes_variables)
		if ok {
			attrs.SetVariables(variables)
		} else {
			raw["variables"] = v.Variables
		}
		attrs.SetRunTasks(new(v.RunTasks))
		attrs.SetWorkspaceLocking(new(v.WorkspaceLocking))
	}

	if len(raw) > 0 {
		attrs.SetAdditionalData(raw)
	}

	return attrs
}

// parseEnum parses a string into a Kiota enum value. It reports false when the value is not a known enum member (including an empty string).
func parseEnum[T any](v string, parse func(string) (any, error)) (*T, bool) {
	r, err := parse(v)
	if err != nil || r == nil {
		return nil, false
	}
	t, ok := r.(*T)
	return t, ok
}

func (r *WorkspaceReconciler) createTeamAccess(ctx context.Context, w *workspaceInstance, createTeamAccess map[string]*teamAccess) error {
	workspaceID := w.instance.Status.WorkspaceID
	for tID, v := range createTeamAccess {
		attrs := teamAccessAttributes(v)

		teamData := models.NewTeamsIdentifier()
		teamData.SetId(new(tID))
		teamData.SetTypeEscaped(new(models.TEAMS_TEAMSIDENTIFIER_TYPE))
		team := models.NewTeamsHasOne()
		team.SetData(teamData)

		wsData := models.NewWorkspacesHasOne_data()
		wsData.SetId(new(workspaceID))
		wsData.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIER_TYPE))
		ws := models.NewWorkspacesHasOne()
		ws.SetData(wsData)

		rels := models.NewTeamWorkspaces_relationships()
		rels.SetTeam(team)
		rels.SetWorkspace(ws)

		data := models.NewTeamWorkspaces()
		data.SetTypeEscaped(new(models.TEAMWORKSPACES_TEAMWORKSPACES_TYPE))
		data.SetAttributes(attrs)
		data.SetRelationships(rels)
		body := models.NewTeamWorkspacesEnvelope()
		body.SetData(data)

		_, err := w.tfClient.Client.API.TeamWorkspaces().Post(ctx, body, nil)
		if err != nil {
			w.log.Error(err, "Reconcile Team Access", "msg", "failed to create a new team access")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) deleteTeamAccess(ctx context.Context, w *workspaceInstance, deleteTeamAccess map[string]*teamAccess) error {
	for _, v := range deleteTeamAccess {
		err := w.tfClient.Client.API.TeamWorkspaces().ByTeam_workspace_id(v.ID).Delete(ctx, nil)
		if err != nil {
			w.log.Error(err, "Reconcile Team Access", "msg", "failed to delete team access")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) updateTeamAccess(ctx context.Context, w *workspaceInstance, updateTeamAccess map[string]*teamAccess) error {
	for _, v := range updateTeamAccess {
		w.log.Info("Reconcile Team Access", "msg", "updating team access")
		attrs := teamAccessAttributes(v)

		data := models.NewTeamWorkspaces()
		data.SetTypeEscaped(new(models.TEAMWORKSPACES_TEAMWORKSPACES_TYPE))
		data.SetAttributes(attrs)
		body := models.NewTeamWorkspacesEnvelope()
		body.SetData(data)

		_, err := w.tfClient.Client.API.TeamWorkspaces().ByTeam_workspace_id(v.ID).Patch(ctx, body, nil)
		if err != nil {
			w.log.Error(err, "Reconcile Team Access", "msg", "failed to update team access")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) reconcileTeamAccess(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Team Access", "msg", "new reconciliation event")

	specTeamAccess, err := r.getInstanceTeamAccess(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Team Access", "msg", "failed to get instance team access")
		return err
	}

	workspaceTeamAccess, err := r.getWorkspaceTeamAccess(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Team Access", "msg", "failed to get workspace team access")
		return err
	}

	createTeamAccess := getTeamAccessToCreate(specTeamAccess, workspaceTeamAccess)
	if len(createTeamAccess) > 0 {
		w.log.Info("Reconcile Team Access", "msg", fmt.Sprintf("creating %d team accesses", len(createTeamAccess)))
		err := r.createTeamAccess(ctx, w, createTeamAccess)
		if err != nil {
			return err
		}
	}

	updateTeamAccess := getTeamAccessToUpdate(specTeamAccess, workspaceTeamAccess)
	if len(updateTeamAccess) > 0 {
		w.log.Info("Reconcile Team Access", "msg", fmt.Sprintf("updating %d team accesses", len(updateTeamAccess)))
		err := r.updateTeamAccess(ctx, w, updateTeamAccess)
		if err != nil {
			return err
		}
	}

	deleteTeamAccess := getTeamAccessToDelete(specTeamAccess, workspaceTeamAccess)
	if len(deleteTeamAccess) > 0 {
		w.log.Info("Reconcile Team Access", "msg", fmt.Sprintf("deleting %d team accesses", len(deleteTeamAccess)))
		err := r.deleteTeamAccess(ctx, w, deleteTeamAccess)
		if err != nil {
			return err
		}
	}

	return nil
}
