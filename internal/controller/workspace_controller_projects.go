// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-tfe/v2/api/organizations"
	abstractions "github.com/microsoft/kiota-abstractions-go"

	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func (w *workspaceInstance) getProjectIDByName(ctx context.Context) (string, error) {
	projectName := w.instance.Spec.Project.Name
	query := &organizations.ItemProjectsRequestBuilderGetQueryParameters{
		Filternames: &projectName,
		Pagesize:    new(int32(MaxPageSize)),
	}

	for {
		projects, err := w.tfClient.Client.API.Organizations().ByOrganization_name(w.instance.Spec.Organization).Projects().Get(ctx,
			&abstractions.RequestConfiguration[organizations.ItemProjectsRequestBuilderGetQueryParameters]{QueryParameters: query})
		if err != nil {
			return "", err
		}

		for _, p := range projects.GetData() {
			if p.GetAttributes() != nil && pointer.ValueOf(p.GetAttributes().GetName()) == projectName {
				return pointer.ValueOf(p.GetId()), nil
			}
		}

		next := nextPage(projects.GetMeta())
		if next == 0 {
			break
		}
		query.Pagenumber = &next
	}

	return "", fmt.Errorf("project ID not found for project name %q", projectName)
}

func (w *workspaceInstance) getProjectID(ctx context.Context) (string, error) {
	specProject := w.instance.Spec.Project
	if specProject == nil {
		return "", fmt.Errorf("'spec.Project' is not set")
	}

	if specProject.Name != "" {
		w.log.Info("Reconcile Project", "msg", "getting project ID by name")
		return w.getProjectIDByName(ctx)
	}

	w.log.Info("Reconcile Project", "msg", "getting project ID from the spec.Project.ID")
	return specProject.ID, nil
}
