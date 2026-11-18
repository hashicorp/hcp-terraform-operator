// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	"github.com/hashicorp/go-tfe/v2/api/workspaces"
	abstractions "github.com/microsoft/kiota-abstractions-go"

	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

type runTask struct {
	ID               string
	RunTaskID        string
	EnforcementLevel string
	Stage            string
}

func runTasksDifference(a, b map[string]*runTask) map[string]*runTask {
	d := make(map[string]*runTask)

	for k, v := range a {
		if _, ok := b[k]; !ok {
			d[k] = v
		}
	}

	return d
}

func getRunTasksToCreate(spec, ws map[string]*runTask) map[string]*runTask {
	return runTasksDifference(spec, ws)
}

func getRunTasksToUpdate(spec, ws map[string]*runTask) map[string]*runTask {
	o := make(map[string]*runTask)

	if len(spec) == 0 || len(ws) == 0 {
		return o
	}

	for ik, iv := range spec {
		if wv, ok := ws[ik]; ok {
			iv.ID = wv.ID
			if !cmp.Equal(iv, wv, cmpopts.IgnoreFields(runTask{}, "ID")) {
				o[ik] = iv
			}
		}
	}

	return o
}

func getRunTasksToDelete(spec, ws map[string]*runTask) map[string]*runTask {
	return runTasksDifference(ws, spec)
}

func (r *WorkspaceReconciler) createRunTasks(ctx context.Context, w *workspaceInstance, create map[string]*runTask) error {
	for _, rt := range create {
		taskData := models.NewTasksIdentifier()
		taskData.SetId(new(rt.RunTaskID))
		taskData.SetTypeEscaped(new(models.TASKS_TASKSIDENTIFIER_TYPE))
		task := models.NewTasksHasOne()
		task.SetData(taskData)

		rels := models.NewWorkspaceTasks_relationships()
		rels.SetTask(task)

		attrs := models.NewWorkspaceTasks_attributes()
		// go-tfe v1 sent the enforcement level as is (even when empty) and left the validation to the API.
		// An unknown (or empty) value is sent as a raw attribute since the generated enum cannot hold it.
		if lv, err := models.ParseWorkspaceTasks_attributes_enforcementLevel(rt.EnforcementLevel); err == nil && lv != nil {
			attrs.SetEnforcementLevel(lv.(*models.WorkspaceTasks_attributes_enforcementLevel))
		} else {
			attrs.SetAdditionalData(map[string]any{"enforcement-level": rt.EnforcementLevel})
		}
		attrs.SetStage(new(rt.Stage))

		data := models.NewWorkspaceTasks()
		data.SetTypeEscaped(new(models.WORKSPACETASKS_WORKSPACETASKS_TYPE))
		data.SetAttributes(attrs)
		data.SetRelationships(rels)
		body := models.NewWorkspaceTasksEnvelope()
		body.SetData(data)

		_, err := w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).Tasks().Post(ctx, body, nil)
		if err != nil {
			w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to create a new run task")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) updateRunTasks(ctx context.Context, w *workspaceInstance, update map[string]*runTask) error {
	for _, v := range update {
		w.log.Info("Reconcile Run Tasks", "msg", "updating run task")
		attrs := models.NewWorkspaceTasks_attributes()
		// go-tfe v1 omitted the enforcement level when empty on update.
		if lv, err := models.ParseWorkspaceTasks_attributes_enforcementLevel(v.EnforcementLevel); err == nil && lv != nil {
			attrs.SetEnforcementLevel(lv.(*models.WorkspaceTasks_attributes_enforcementLevel))
		} else if v.EnforcementLevel != "" {
			attrs.SetAdditionalData(map[string]any{"enforcement-level": v.EnforcementLevel})
		}
		attrs.SetStage(new(v.Stage))

		data := models.NewWorkspaceTasks()
		data.SetTypeEscaped(new(models.WORKSPACETASKS_WORKSPACETASKS_TYPE))
		data.SetAttributes(attrs)
		body := models.NewWorkspaceTasksEnvelope()
		body.SetData(data)

		_, err := w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).Tasks().ById(v.ID).Patch(ctx, body, nil)
		if err != nil {
			w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to update run task")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) deleteRunTasks(ctx context.Context, w *workspaceInstance, delete map[string]*runTask) error {
	for _, d := range delete {
		err := w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).Tasks().ById(d.ID).Delete(ctx, nil)
		if err != nil {
			w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to delete run task")
			return err
		}
	}

	return nil
}

func hasRunTaskName(w *workspaceInstance) bool {
	for _, rt := range w.instance.Spec.RunTasks {
		if rt.Name != "" {
			return true
		}
	}

	return false
}

func (r *WorkspaceReconciler) getInstanceRunTasks(ctx context.Context, w *workspaceInstance) (map[string]*runTask, error) {
	o := map[string]*runTask{}

	rl := make(map[string]string)
	if hasRunTaskName(w) {
		query := &organizations.ItemTasksRequestBuilderGetQueryParameters{
			Pagesize:   new(int32(MaxPageSize)),
			Pagenumber: new(int32(InitPageNumber)),
		}
		for {
			rt, err := w.tfClient.Client.API.Organizations().ByOrganization_name(w.instance.Spec.Organization).Tasks().Get(ctx,
				&abstractions.RequestConfiguration[organizations.ItemTasksRequestBuilderGetQueryParameters]{QueryParameters: query})
			if err != nil {
				return o, err
			}
			for _, t := range rt.GetData() {
				if t.GetAttributes() == nil {
					continue
				}
				rl[pointer.ValueOf(t.GetAttributes().GetName())] = pointer.ValueOf(t.GetId())
			}
			next := nextPage(rt.GetMeta())
			if next == 0 {
				break
			}
			query.Pagenumber = &next
		}
	}

	for _, rt := range w.instance.Spec.RunTasks {
		id := rt.ID
		if rt.Name != "" {
			id = rl[rt.Name]
		}
		o[id] = &runTask{
			EnforcementLevel: rt.EnforcementLevel,
			Stage:            rt.Stage,
			RunTaskID:        id,
		}
	}

	return o, nil
}

func (r *WorkspaceReconciler) getWorkspaceRunTasks(ctx context.Context, w *workspaceInstance) (map[string]*runTask, error) {
	o := map[string]*runTask{}

	query := &workspaces.ItemTasksRequestBuilderGetQueryParameters{
		Pagesize:   new(int32(MaxPageSize)),
		Pagenumber: new(int32(InitPageNumber)),
	}
	for {
		wrt, err := w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).Tasks().Get(ctx,
			&abstractions.RequestConfiguration[workspaces.ItemTasksRequestBuilderGetQueryParameters]{QueryParameters: query})
		if err != nil {
			return o, err
		}
		for _, rt := range wrt.GetData() {
			t := &runTask{ID: pointer.ValueOf(rt.GetId())}
			if rel := rt.GetRelationships(); rel != nil && rel.GetTask() != nil && rel.GetTask().GetData() != nil {
				t.RunTaskID = pointer.ValueOf(rel.GetTask().GetData().GetId())
			}
			if attrs := rt.GetAttributes(); attrs != nil {
				if v := attrs.GetEnforcementLevel(); v != nil {
					t.EnforcementLevel = v.String()
				}
				t.Stage = pointer.ValueOf(attrs.GetStage())
			}
			o[t.RunTaskID] = t
		}
		next := nextPage(wrt.GetMeta())
		if next == 0 {
			break
		}
		query.Pagenumber = &next
	}

	return o, nil
}

func (r *WorkspaceReconciler) reconcileRunTasks(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Run Tasks", "msg", "new reconciliation event")

	spec, err := r.getInstanceRunTasks(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to get instance run tasks")
		return err
	}

	ws, err := r.getWorkspaceRunTasks(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to get workspace run tasks")
		return err
	}

	create := getRunTasksToCreate(spec, ws)
	if len(create) > 0 {
		w.log.Info("Reconcile Run Tasks", "msg", fmt.Sprintf("creating %d run tasks", len(create)))
		err := r.createRunTasks(ctx, w, create)
		if err != nil {
			w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to create a run task")
			return err
		}
	}

	update := getRunTasksToUpdate(spec, ws)
	if len(update) > 0 {
		w.log.Info("Reconcile Run Tasks", "msg", fmt.Sprintf("updating %d run tasks", len(update)))
		err := r.updateRunTasks(ctx, w, update)
		if err != nil {
			w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to update a run task")
			return err
		}
	}

	delete := getRunTasksToDelete(spec, ws)
	if len(delete) > 0 {
		w.log.Info("Reconcile Run Tasks", "msg", fmt.Sprintf("deleting %d run tasks", len(delete)))
		err := r.deleteRunTasks(ctx, w, delete)
		if err != nil {
			w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to delete a run task")
			return err
		}
	}

	return nil
}
