// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"

	"github.com/hashicorp/go-tfe/v2/api/models"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
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
func getWorkspaceTags(workspace models.Workspacesable) map[string]bool {
	tags := make(map[string]bool)

	tagNames := workspace.GetAttributes().GetTagNames()
	if len(tagNames) == 0 {
		return tags
	}

	for _, t := range tagNames {
		tags[t] = true
	}

	return tags
}

// getTagsToAdd returns a list of tags that need to be added to the workspace.
func getTagsToAdd(instanceTags, workspaceTags map[string]bool) []string {
	return tagDifference(instanceTags, workspaceTags)
}

// getTagsToRemove returns a list of tags that need to be removed from the workspace.
func getTagsToRemove(instanceTags, workspaceTags map[string]bool) []string {
	return tagDifference(workspaceTags, instanceTags)
}

// tagDifference returns the list of tag names that consists of the elements of leftTags
// which are not elements of rightTags
func tagDifference(leftTags, rightTags map[string]bool) []string {
	var d []string
	for t := range leftTags {
		if !rightTags[t] {
			d = append(d, t)
		}
	}
	return d
}

// addWorkspaceTags adds tags to workspace
func (r *WorkspaceReconciler) addWorkspaceTags(ctx context.Context, w *workspaceInstance, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	items := make([]models.TagsCreateArrayDocument_dataable, 0, len(tags))
	for _, t := range tags {
		attrs := models.NewTagsCreateArrayDocument_data_attributes()
		attrs.SetName(new(t))
		item := models.NewTagsCreateArrayDocument_data()
		item.SetTypeEscaped(new(models.TAGS_TAGSCREATEARRAYDOCUMENT_DATA_TYPE))
		item.SetAttributes(attrs)
		items = append(items, item)
	}
	body := models.NewTagsCreateArrayDocument()
	body.SetData(items)

	return w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).Relationships().Tags().Post(ctx, body, nil)
}

// removeWorkspaceTags removes tags from workspace
func (r *WorkspaceReconciler) removeWorkspaceTags(ctx context.Context, w *workspaceInstance, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	// go-tfe v1 removed tags by name: the request items carried the resource type and the tag name attribute only, without an ID.
	// The same payload is sent here (no extra GET request, thus no pagination concerns).
	items := make([]models.TagsRemoveArrayDocument_dataable, 0, len(tags))
	for _, t := range tags {
		item := models.NewTagsRemoveArrayDocument_data()
		item.SetTypeEscaped(new(models.TAGS_TAGSREMOVEARRAYDOCUMENT_DATA_TYPE))
		item.SetAdditionalData(map[string]any{"attributes": map[string]any{"name": t}})
		items = append(items, item)
	}
	body := models.NewTagsRemoveArrayDocument()
	body.SetData(items)

	return w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).Relationships().Tags().Delete(ctx, body, nil)
}

func (r *WorkspaceReconciler) reconcileTags(ctx context.Context, w *workspaceInstance, workspace models.Workspacesable) error {
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
