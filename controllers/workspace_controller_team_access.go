package controllers

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
)

func getTeamID(teams map[string]*tfc.Team, instanceTeam appv1alpha2.Team) (string, error) {
	if instanceTeam.Name != "" {
		if t, ok := teams[instanceTeam.Name]; ok {
			return t.ID, nil
		}
		return "", fmt.Errorf("team ID was not found by name %q", instanceTeam.Name)
	}

	if instanceTeam.ID != "" {
		for _, v := range teams {
			if v.ID == instanceTeam.ID {
				return v.ID, nil
			}
		}
	}

	return "", fmt.Errorf("team ID was not found by ID %q", instanceTeam.ID)
}

func (r *WorkspaceReconciler) getInstanceTeamAccess(ctx context.Context, instance *appv1alpha2.Workspace) (map[string]*tfc.TeamAccess, error) {
	o := map[string]*tfc.TeamAccess{}

	if instance.Spec.TeamAccess == nil {
		return o, nil
	}

	teams, err := r.getTeams(ctx, instance)
	if err != nil {
		return o, err
	}

	for _, ta := range instance.Spec.TeamAccess {
		tID, err := getTeamID(teams, ta.Team)
		if err != nil {
			r.log.Error(err, "Reconcile Team Access", "msg", "failed to get team ID")
			r.Recorder.Event(instance, corev1.EventTypeWarning, "ReconcileTeamAccess", "Failed to get team ID")
			return o, err
		}

		o[tID] = &tfc.TeamAccess{
			Team: &tfc.Team{
				ID: tID,
			},
			Workspace: &tfc.Workspace{
				ID: instance.Status.WorkspaceID,
			},
			Access:           tfc.AccessType(ta.Access),
			Runs:             tfc.RunsPermissionType(ta.Custom.Runs),
			RunTasks:         ta.Custom.RunTasks,
			SentinelMocks:    tfc.SentinelMocksPermissionType(ta.Custom.Sentinel),
			StateVersions:    tfc.StateVersionsPermissionType(ta.Custom.StateVersions),
			Variables:        tfc.VariablesPermissionType(ta.Custom.Variables),
			WorkspaceLocking: ta.Custom.WorkspaceLocking,
		}
	}

	return o, nil
}

func (r *WorkspaceReconciler) getWorkspaceTeamAccess(ctx context.Context, instance *appv1alpha2.Workspace) (map[string]*tfc.TeamAccess, error) {
	o := map[string]*tfc.TeamAccess{}

	t, err := r.tfClient.Client.TeamAccess.List(ctx, &tfc.TeamAccessListOptions{WorkspaceID: instance.Status.WorkspaceID})
	if err != nil {
		return o, err
	}

	for _, ta := range t.Items {
		o[ta.Team.ID] = ta
	}
	return o, nil
}

func (r *WorkspaceReconciler) getTeams(ctx context.Context, instance *appv1alpha2.Workspace) (map[string]*tfc.Team, error) {
	teams := make(map[string]*tfc.Team)

	fTeams := []string{}
	for _, t := range instance.Spec.TeamAccess {
		if t.Team.Name != "" {
			fTeams = append(fTeams, t.Team.Name)
		}
	}

	tl, err := r.tfClient.Client.Teams.List(ctx, instance.Spec.Organization, &tfc.TeamListOptions{
		Names: fTeams,
	})
	if err != nil {
		return teams, err
	}

	for _, t := range tl.Items {
		teams[t.Name] = t
	}

	return teams, nil
}

func teamAccessDifference(a, b map[string]*tfc.TeamAccess) map[string]*tfc.TeamAccess {
	d := make(map[string]*tfc.TeamAccess)

	for k, v := range a {
		if _, ok := b[k]; !ok {
			d[k] = v
		}
	}

	return d
}

func getTeamAccessToCreate(ctx context.Context, specTeamAccess, workspaceTeamAccess map[string]*tfc.TeamAccess) map[string]*tfc.TeamAccess {
	return teamAccessDifference(specTeamAccess, workspaceTeamAccess)
}

func getTeamAccessToDelete(ctx context.Context, specTeamAccess, workspaceTeamAccess map[string]*tfc.TeamAccess) map[string]*tfc.TeamAccess {
	return teamAccessDifference(workspaceTeamAccess, specTeamAccess)
}

func getTeamAccessToUpdate(ctx context.Context, specTeamAccess, workspaceTeamAccess map[string]*tfc.TeamAccess) map[string]*tfc.TeamAccess {
	ta := make(map[string]*tfc.TeamAccess)

	if len(specTeamAccess) == 0 || len(workspaceTeamAccess) == 0 {
		return ta
	}

	for ik, iv := range specTeamAccess {
		if wv, ok := workspaceTeamAccess[ik]; ok {
			iv.ID = wv.ID
			if iv.Access == tfc.AccessCustom {
				if !cmp.Equal(iv, wv, cmpopts.IgnoreFields(tfc.TeamAccess{}, "ID", "Team", "Workspace")) {
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

func (r *WorkspaceReconciler) createTeamAccess(ctx context.Context, workspaceID string, createTeamAccess map[string]*tfc.TeamAccess) error {
	for tID, v := range createTeamAccess {
		option := tfc.TeamAccessAddOptions{
			Workspace: &tfc.Workspace{
				ID: workspaceID,
			},
			Team: &tfc.Team{
				ID: tID,
			},
			Access: &v.Access,
		}

		if v.Access == tfc.AccessCustom {
			option.Runs = &v.Runs
			option.RunTasks = &v.RunTasks
			option.SentinelMocks = &v.SentinelMocks
			option.StateVersions = &v.StateVersions
			option.Variables = &v.Variables
			option.WorkspaceLocking = &v.WorkspaceLocking
		}

		_, err := r.tfClient.Client.TeamAccess.Add(ctx, option)
		if err != nil {
			r.log.Error(err, "Reconcile Team Access", "msg", "failed to create a new team access")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) deleteTeamAccess(ctx context.Context, deleteTeamAccess map[string]*tfc.TeamAccess) error {
	for _, v := range deleteTeamAccess {
		err := r.tfClient.Client.TeamAccess.Remove(ctx, v.ID)
		if err != nil {
			r.log.Error(err, "Reconcile Team Access", "msg", "failed to delete team access")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) updateTeamAccess(ctx context.Context, updateTeamAccess map[string]*tfc.TeamAccess) error {
	for _, v := range updateTeamAccess {
		r.log.Info("Reconcile Team Access", "msg", "updating team access")
		option := tfc.TeamAccessUpdateOptions{
			Access: &v.Access,
		}

		if v.Access == tfc.AccessCustom {
			option.Runs = &v.Runs
			option.RunTasks = &v.RunTasks
			option.SentinelMocks = &v.SentinelMocks
			option.StateVersions = &v.StateVersions
			option.Variables = &v.Variables
			option.WorkspaceLocking = &v.WorkspaceLocking
		}

		_, err := r.tfClient.Client.TeamAccess.Update(ctx, v.ID, option)
		if err != nil {
			r.log.Error(err, "Reconcile Team Access", "msg", "failed to update team access")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) reconcileTeamAccess(ctx context.Context, instance *appv1alpha2.Workspace, workspace *tfc.Workspace) error {
	r.log.Info("Reconcile Team Access", "msg", "new reconciliation event")

	specTeamAccess, err := r.getInstanceTeamAccess(ctx, instance)
	if err != nil {
		r.log.Error(err, "Reconcile Team Access", "msg", "failed to get instance team access")
		return err
	}

	workspaceTeamAccess, err := r.getWorkspaceTeamAccess(ctx, instance)
	if err != nil {
		r.log.Error(err, "Reconcile Team Access", "msg", "failed to get workspace team access")
		return err
	}

	createTeamAccess := getTeamAccessToCreate(ctx, specTeamAccess, workspaceTeamAccess)
	if len(createTeamAccess) > 0 {
		r.log.Info("Reconcile Team Access", "msg", fmt.Sprintf("creating %d team accesses", len(createTeamAccess)))
		err := r.createTeamAccess(ctx, instance.Status.WorkspaceID, createTeamAccess)
		if err != nil {
			return err
		}
	}

	updateTeamAccess := getTeamAccessToUpdate(ctx, specTeamAccess, workspaceTeamAccess)
	if len(updateTeamAccess) > 0 {
		r.log.Info("Reconcile Team Access", "msg", fmt.Sprintf("updating %d team accesses", len(updateTeamAccess)))
		err := r.updateTeamAccess(ctx, updateTeamAccess)
		if err != nil {
			return err
		}
	}

	deleteTeamAccess := getTeamAccessToDelete(ctx, specTeamAccess, workspaceTeamAccess)
	if len(deleteTeamAccess) > 0 {
		r.log.Info("Reconcile Team Access", "msg", fmt.Sprintf("deleting %d team accesses", len(deleteTeamAccess)))
		err := r.deleteTeamAccess(ctx, deleteTeamAccess)
		if err != nil {
			return err
		}
	}

	return nil
}
