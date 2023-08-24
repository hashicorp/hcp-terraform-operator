// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

// HELPERS

// getTags return a map that maps consist of all tags defined in a object specification(spec.tags)
// and values 'true' to simulate the Set structure
func getTags(instance *appv1alpha2.Workspace) map[string]bool {
	tags := make(map[string]bool)

	if len(instance.Spec.Tags) == 0 {
		return tags
	}

	for _, t := range instance.Spec.Tags {
		tags[string(t)] = true
	}

	return tags
}

// getWorkspaceTags return a map that maps consist of all tags assigned to workspace
// and values 'true' to simulate the Set structure
func getWorkspaceTags(workspace *tfc.Workspace) map[string]bool {
	tags := make(map[string]bool)

	if len(workspace.TagNames) == 0 {
		return tags
	}

	for _, t := range workspace.TagNames {
		tags[t] = true
	}

	return tags
}

// getTagsToAdd returns a list of tags that need to be added to the workspace.
func getTagsToAdd(instanceTags, workspaceTags map[string]bool) []*tfc.Tag {
	return tagDifference(instanceTags, workspaceTags)
}

// getTagsToRemove returns a list of tags that need to be removed from the workspace.
func getTagsToRemove(instanceTags, workspaceTags map[string]bool) []*tfc.Tag {
	return tagDifference(workspaceTags, instanceTags)
}

// tagDifference returns the list of tags(type tfc.Tag) that consists of the elements of leftTags
// which are not elements of rightTags
func tagDifference(leftTags, rightTags map[string]bool) []*tfc.Tag {
	var d []*tfc.Tag

	for t := range leftTags {
		if !rightTags[t] {
			d = append(d, &tfc.Tag{Name: t})
		}
	}

	return d
}

// addWorkspaceTags adds tags to workspace
func (r *WorkspaceReconciler) addWorkspaceTags(ctx context.Context, w *workspaceInstance, tags []*tfc.Tag) error {
	if len(tags) == 0 {
		return nil
	}

	return w.tfClient.Client.Workspaces.AddTags(ctx, w.instance.Status.WorkspaceID, tfc.WorkspaceAddTagsOptions{Tags: tags})
}

// removeWorkspaceTags removes tags from workspace
func (r *WorkspaceReconciler) removeWorkspaceTags(ctx context.Context, w *workspaceInstance, tags []*tfc.Tag) error {
	if len(tags) == 0 {
		return nil
	}

	return w.tfClient.Client.Workspaces.RemoveTags(ctx, w.instance.Status.WorkspaceID, tfc.WorkspaceRemoveTagsOptions{Tags: tags})
}

func (r *WorkspaceReconciler) reconcileTags(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	w.log.Info("Reconcile Tags", "msg", "new reconciliation event")

	instanceTags := getTags(&w.instance)
	workspaceTags := getWorkspaceTags(workspace)

	removeTags := getTagsToRemove(instanceTags, workspaceTags)
	if len(removeTags) > 0 {
		w.log.Info("Reconcile Tags", "msg", "removing tags from the workspace")
		err := r.removeWorkspaceTags(ctx, w, removeTags)
		if err != nil {
			return err
		}
	}

	addTags := getTagsToAdd(instanceTags, workspaceTags)
	if len(addTags) > 0 {
		w.log.Info("Reconcile Tags", "msg", "adding tags from the workspace")
		err := r.addWorkspaceTags(ctx, w, addTags)
		if err != nil {
			return err
		}
	}

	return nil
}
