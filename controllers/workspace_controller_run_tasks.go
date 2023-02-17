// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	tfc "github.com/hashicorp/go-tfe"
)

func runTasksDifference(a, b map[string]*tfc.WorkspaceRunTask) map[string]*tfc.WorkspaceRunTask {
	d := make(map[string]*tfc.WorkspaceRunTask)

	for k, v := range a {
		if _, ok := b[k]; !ok {
			d[k] = v
		}
	}

	return d
}

func getRunTasksToCreate(ctx context.Context, specRunTasks, workspaceRunTasks map[string]*tfc.WorkspaceRunTask) map[string]*tfc.WorkspaceRunTask {
	return runTasksDifference(specRunTasks, workspaceRunTasks)
}

func getRunTasksToUpdate(ctx context.Context, specRunTasks, workspaceRunTasks map[string]*tfc.WorkspaceRunTask) map[string]*tfc.WorkspaceRunTask {
	o := make(map[string]*tfc.WorkspaceRunTask)

	if len(specRunTasks) == 0 || len(workspaceRunTasks) == 0 {
		return o
	}

	for ik, iv := range specRunTasks {
		if wv, ok := workspaceRunTasks[ik]; ok {
			iv.ID = wv.ID
			if !cmp.Equal(iv, wv, cmpopts.IgnoreFields(tfc.WorkspaceRunTask{}, "Workspace")) {
				o[ik] = iv
			}
		}
	}

	return o
}

func getRunTasksToDelete(ctx context.Context, specRunTasks, workspaceRunTasks map[string]*tfc.WorkspaceRunTask) map[string]*tfc.WorkspaceRunTask {
	return runTasksDifference(workspaceRunTasks, specRunTasks)
}

func (r *WorkspaceReconciler) createRunTasks(ctx context.Context, w *workspaceInstance, create map[string]*tfc.WorkspaceRunTask) error {
	for _, rt := range create {
		_, err := w.tfClient.Client.WorkspaceRunTasks.Create(ctx, w.instance.Status.WorkspaceID, tfc.WorkspaceRunTaskCreateOptions{
			Type:             "workspace-task",
			EnforcementLevel: rt.EnforcementLevel,
			RunTask:          rt.RunTask,
			Stage:            &rt.Stage,
		})
		if err != nil {
			w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to create a new run task")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) updateRunTasks(ctx context.Context, w *workspaceInstance, update map[string]*tfc.WorkspaceRunTask) error {
	for _, v := range update {
		w.log.Info("Reconcile Run Tasks", "msg", "updating run task")
		_, err := w.tfClient.Client.WorkspaceRunTasks.Update(ctx, w.instance.Status.WorkspaceID, v.ID, tfc.WorkspaceRunTaskUpdateOptions{
			Type:             "workspace-task",
			EnforcementLevel: v.EnforcementLevel,
			Stage:            &v.Stage,
		})
		if err != nil {
			w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to update run task")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) deleteRunTasks(ctx context.Context, w *workspaceInstance, delete map[string]*tfc.WorkspaceRunTask) error {
	for _, d := range delete {
		err := w.tfClient.Client.WorkspaceRunTasks.Delete(ctx, w.instance.Status.WorkspaceID, d.ID)
		if err != nil {
			// if err == tfc.ErrResourceNotFound {
			// 	continue
			// }
			w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to delete run task")
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) getInstanceRunTasks(ctx context.Context, w *workspaceInstance) (map[string]*tfc.WorkspaceRunTask, error) {
	o := map[string]*tfc.WorkspaceRunTask{}

	for _, rt := range w.instance.Spec.RunTasks {
		o[rt.ID] = &tfc.WorkspaceRunTask{
			EnforcementLevel: tfc.TaskEnforcementLevel(rt.EnforcementLevel),
			Stage:            tfc.Stage(rt.Stage),
			RunTask:          &tfc.RunTask{ID: rt.ID},
		}
	}

	return o, nil
}

func (r *WorkspaceReconciler) getWorkspaceRunTasks(ctx context.Context, w *workspaceInstance) (map[string]*tfc.WorkspaceRunTask, error) {
	o := map[string]*tfc.WorkspaceRunTask{}

	wrt, err := w.tfClient.Client.WorkspaceRunTasks.List(ctx, w.instance.Status.WorkspaceID, &tfc.WorkspaceRunTaskListOptions{})
	if err != nil {
		return o, err
	}

	for _, rt := range wrt.Items {
		o[rt.RunTask.ID] = &tfc.WorkspaceRunTask{
			ID:               rt.ID,
			EnforcementLevel: rt.EnforcementLevel,
			Stage:            rt.Stage,
			RunTask:          &tfc.RunTask{ID: rt.RunTask.ID},
		}
	}

	return o, nil
}

func (r *WorkspaceReconciler) reconcileRunTasks(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Run Tasks", "msg", "new reconciliation event")

	specRunTasks, err := r.getInstanceRunTasks(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to get instance run tasks")
		return err
	}

	workspaceRunTasks, err := r.getWorkspaceRunTasks(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to get workspace run tasks")
		return err
	}

	create := getRunTasksToCreate(ctx, specRunTasks, workspaceRunTasks)
	if len(create) > 0 {
		w.log.Info("Reconcile Run Tasks", "msg", fmt.Sprintf("creating %d run tasks", len(create)))
		err := r.createRunTasks(ctx, w, create)
		if err != nil {
			w.log.Error(err, "Reconcile Run Tasks", "msg", "failed to create a run task")
			return err
		}
	}

	update := getRunTasksToUpdate(ctx, specRunTasks, workspaceRunTasks)
	if len(update) > 0 {
		w.log.Info("Reconcile Run Tasks", "msg", fmt.Sprintf("updating %d run tasks", len(update)))
		err := r.updateRunTasks(ctx, w, update)
		if err != nil {
			return err
		}
	}

	delete := getRunTasksToDelete(ctx, specRunTasks, workspaceRunTasks)
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
